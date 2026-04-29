# Data CLI — MVP Tech Spec

Three commands — `connect`, `query`, `schema` — across three data source types: Postgres, JSON files, and BigQuery.

---

## Validation Goal

**Hypothesis:** a coding agent (Claude Code, OpenCode, Gemini CLI, etc.) can use Data CLI as its data tool with zero extra integration — no gateway, no MCP server, no protocol adapter. Just shell calls.

Agents already shell out to CLI tools. If Data CLI is a well-behaved CLI, agents can discover connections, inspect schemas, and run queries today, without any special support.

**Success criteria — an agent, without human help, can:**

1. List available connections and understand what data sources exist
2. Inspect the schema of any table or file to understand structure before querying
3. Run a query and get output it can reason over (JSON preferred)
4. Receive clear enough errors to self-correct and retry

**What this means for the MVP:**

- `--format json` is required on all three commands, not just `data query`
- `data connections list --format json` must be machine-readable
- `data schema --format json` must return structured column metadata
- Error messages go to stderr as plain text; exit code 1 on any failure
- Credentials stay in config — the agent only ever sees connection names, never DSNs or keys

---

## Data Source Config

Config lives at `~/.data/config.toml`. Each source type has its own shape.

### Postgres

```toml
[connections.prod]
driver    = "postgres"
dsn       = "postgres://user:pass@host:5432/dbname"
readonly  = true
row_limit = 1000
timeout   = 30

# DSN can be an env var reference or a secret manager path
[connections.staging]
driver = "postgres"
dsn    = "env:STAGING_DATABASE_URL"

[connections.prod-secure]
driver = "postgres"
dsn    = "gcp-secret:projects/my-project/secrets/db-url/versions/latest"
```

**Postgres config fields:**

| Field       | Type    | Default | Description                          |
| ----------- | ------- | ------- | ------------------------------------ |
| `driver`    | string  | —       | Must be `"postgres"`                 |
| `dsn`       | string  | —       | Connection string, env ref, or secret path |
| `readonly`  | bool    | `false` | Reject non-SELECT queries            |
| `row_limit` | int     | `500`   | Cap result rows (0 = unlimited)      |
| `timeout`   | int     | `30`    | Query timeout in seconds             |

### JSON File

```toml
[connections.events]
driver = "json"
path   = "./data/events.json"

[connections.remote-events]
driver = "json"
path   = "/var/log/app/events.json"
```

**JSON config fields:**

| Field    | Type   | Default | Description                          |
| -------- | ------ | ------- | ------------------------------------ |
| `driver` | string | —       | Must be `"json"`                     |
| `path`   | string | —       | Absolute or relative path to `.json` file |

> JSON files are queried via embedded DuckDB using `read_json_auto()`. The file path becomes the implicit table. Schema is inferred automatically from the JSON structure.

### BigQuery

```toml
[connections.warehouse]
driver     = "bigquery"
project    = "my-gcp-project"
dataset    = "analytics"
row_limit  = 2000
timeout    = 60

[connections.warehouse-sa]
driver          = "bigquery"
project         = "my-gcp-project"
dataset         = "analytics"
credentials     = "/path/to/service-account.json"
```

**BigQuery config fields:**

| Field         | Type   | Default | Description                                      |
| ------------- | ------ | ------- | ------------------------------------------------ |
| `driver`      | string | —       | Must be `"bigquery"`                             |
| `project`     | string | —       | GCP project ID                                   |
| `dataset`     | string | —       | Default dataset (used in schema listing)         |
| `credentials` | string | —       | Path to service account JSON (omit for ADC)      |
| `row_limit`   | int    | `500`   | Cap result rows (0 = unlimited)                  |
| `timeout`     | int    | `30`    | Query timeout in seconds                         |

> If `credentials` is omitted, ADC (Application Default Credentials) is used — `gcloud auth application-default login` or `GOOGLE_APPLICATION_CREDENTIALS` env var.

### Defaults block

```toml
[defaults]
row_limit = 500
timeout   = 30
```

Per-connection values override defaults. Defaults apply when a field is unset.

---

## DSN Resolution (Postgres only)

The `dsn` field supports three forms, resolved at connection time:

| Form                              | Example                                                  |
| --------------------------------- | -------------------------------------------------------- |
| Literal string                    | `"postgres://user:pass@host:5432/db"`                   |
| Env var reference                 | `"env:DATABASE_URL"`                                    |
| GCP Secret Manager path           | `"gcp-secret:projects/p/secrets/s/versions/latest"`    |

Resolution is handled in `internal/config/secrets.go` before the DSN is passed to the driver.

---

## Commands

### `data connect`

Save a named connection to `~/.data/config.toml`.

```
data connect <name> <dsn-or-path>
data connect <name> --driver bigquery --project <id> --dataset <ds>
```

**Examples:**

```bash
# Postgres — DSN directly
data connect prod postgres://user:pass@localhost:5432/mydb

# Postgres — env var reference
data connect staging env:STAGING_DATABASE_URL

# JSON file
data connect events ./data/events.json

# BigQuery — interactive prompt for project/dataset
data connect warehouse --driver bigquery --project my-project --dataset analytics
```

**Behavior:**
- Writes or updates the named entry in `~/.data/config.toml`
- Driver is inferred from the DSN prefix (`postgres://` → postgres, `.json` extension → json) or set explicitly via `--driver`
- Prints confirmation: `Connection "prod" saved.`
- `--test` flag: attempts a connection and prints success/error before saving

**Supporting commands:**

```bash
data connections list                      # Table of name + driver (default)
data connections list --format json        # Machine-readable for agents
data connections remove <name>             # Remove entry from config
```

`--format json` output:

```json
[
  { "name": "prod",      "driver": "postgres" },
  { "name": "events",    "driver": "json"     },
  { "name": "warehouse", "driver": "bigquery" }
]
```

DSNs and credentials are never included in list output.

---

### `data query`

Run SQL against a named connection.

```
data query <name> <sql>
data query <name> <sql> [--format table|csv|json|md] [--limit N]
```

**Examples:**

```bash
# Postgres
data query prod "SELECT id, email FROM users LIMIT 10"

# JSON file — path is the implicit table name
data query events "SELECT type, count(*) FROM events GROUP BY type"

# BigQuery — dataset is the default schema
data query warehouse "SELECT date, sum(revenue) FROM orders GROUP BY date ORDER BY date"

# Output formats
data query prod "SELECT * FROM orders" --format csv
data query prod "SELECT * FROM orders" --format json
data query prod "SELECT * FROM orders" --format md

# Row limit override
data query prod "SELECT * FROM logs" --limit 50
```

**Behavior:**
- Respects `row_limit` from config; `--limit` flag overrides per-invocation
- Respects `timeout` from config
- If `readonly = true` and SQL is not a SELECT, exits with error before sending to DB
- Prints column headers + rows in the chosen format to stdout
- Errors go to stderr; exit code 1 on failure
- Default format: `table`

**Output — table format (default):**

```
id   │ email                │ created_at
─────┼──────────────────────┼─────────────────────
1    │ alice@example.com    │ 2024-01-15 09:22:01
2    │ bob@example.com      │ 2024-01-16 14:10:55

2 rows  (12ms)
```

---

### `data schema`

Inspect the structure of a data source.

```
data schema <name>                         # List all tables
data schema <name> <table>                 # Describe columns of a specific table
data schema <name> [--format table|json]   # json for agent use
```

**Examples:**

```bash
# Postgres — list tables
data schema prod

# Postgres — describe a table
data schema prod orders

# JSON file — describe inferred schema
data schema events

# BigQuery — list tables in the configured dataset
data schema warehouse

# BigQuery — describe a table
data schema warehouse orders
```

**`data schema <name>` output:**

```
Tables in prod
  orders
  users
  products
  sessions
```

**`data schema <name> <table>` output (table format):**

```
Table: orders
┌─────────────┬──────────────┬──────────┬──────────────┐
│ Column      │ Type         │ Nullable │ Default      │
├─────────────┼──────────────┼──────────┼──────────────┤
│ id          │ uuid         │ NO       │ gen_random…  │
│ user_id     │ uuid         │ NO       │              │
│ status      │ varchar(20)  │ NO       │ pending      │
│ total       │ numeric(10,2)│ NO       │              │
│ created_at  │ timestamptz  │ NO       │ now()        │
└─────────────┴──────────────┴──────────┴──────────────┘
```

**`data schema <name> <table> --format json` output (agent use):**

```json
{
  "table": "orders",
  "columns": [
    { "name": "id",         "type": "uuid",          "nullable": false, "default": "gen_random_uuid()" },
    { "name": "user_id",    "type": "uuid",          "nullable": false, "default": null },
    { "name": "status",     "type": "varchar(20)",   "nullable": false, "default": "pending" },
    { "name": "total",      "type": "numeric(10,2)", "nullable": false, "default": null },
    { "name": "created_at", "type": "timestamptz",   "nullable": false, "default": "now()" }
  ]
}
```

**Per-driver schema behavior:**

| Driver   | `schema <name>`                        | `schema <name> <table>`                     |
| -------- | -------------------------------------- | ------------------------------------------- |
| postgres | `SELECT table_name FROM information_schema.tables` | `SELECT column_name, data_type, is_nullable, column_default FROM information_schema.columns` |
| json     | Prints `"events"` (the filename stem)  | Infers columns via `DESCRIBE SELECT * FROM read_json_auto(path) LIMIT 0` in DuckDB |
| bigquery | Lists tables in the configured dataset | Uses BigQuery `TableMetadata` API           |

---

## Repository Structure

```
data-cli/
├── cmd/
│   ├── main.go
│   ├── root.go             # Root cobra command, --config flag
│   ├── connect.go          # data connect / data connections
│   ├── query.go            # data query
│   └── schema.go           # data schema
│
├── internal/
│   ├── config/
│   │   ├── config.go       # TOML load/save; Config, ConnectionConfig structs
│   │   └── secrets.go      # DSN resolution: literal / env: / gcp-secret:
│   │
│   ├── connector/
│   │   ├── connector.go    # Connector interface + shared types
│   │   ├── registry.go     # map[driver] → factory func
│   │   ├── postgres.go     # pgx/v5 implementation
│   │   ├── json.go         # DuckDB-backed JSON implementation
│   │   └── bigquery.go     # cloud.google.com/go/bigquery implementation
│   │
│   └── output/
│       ├── table.go        # Terminal table renderer
│       ├── csv.go
│       ├── json.go
│       └── markdown.go
│
├── go.mod
├── go.sum
└── Makefile
```

---

## Connector Interface

```go
// internal/connector/connector.go

type QueryOptions struct {
    RowLimit int
    Timeout  time.Duration
    ReadOnly bool
}

type Column struct {
    Name     string
    Type     string
    Nullable bool
    Default  *string
}

type Result struct {
    Columns []Column
    Rows    [][]any
    Elapsed time.Duration
}

type Connector interface {
    Connect(ctx context.Context) error
    Query(ctx context.Context, sql string, opts QueryOptions) (*Result, error)
    ListTables(ctx context.Context) ([]string, error)
    DescribeTable(ctx context.Context, table string) ([]Column, error)
    Close() error
}
```

---

## Config Structs

```go
// internal/config/config.go

type Config struct {
    Connections map[string]ConnectionConfig `toml:"connections"`
    Defaults    DefaultsConfig              `toml:"defaults"`
}

type ConnectionConfig struct {
    Driver      string `toml:"driver"`               // "postgres" | "json" | "bigquery"
    DSN         string `toml:"dsn,omitempty"`         // postgres only
    Path        string `toml:"path,omitempty"`        // json only
    Project     string `toml:"project,omitempty"`     // bigquery only
    Dataset     string `toml:"dataset,omitempty"`     // bigquery only
    Credentials string `toml:"credentials,omitempty"` // bigquery only; empty = ADC
    ReadOnly    bool   `toml:"readonly,omitempty"`
    RowLimit    int    `toml:"row_limit,omitempty"`
    Timeout     int    `toml:"timeout,omitempty"`
}

type DefaultsConfig struct {
    RowLimit int `toml:"row_limit"`
    Timeout  int `toml:"timeout"`
}
```

---

## Driver Implementations

### Postgres (`internal/connector/postgres.go`)

- Library: `github.com/jackc/pgx/v5`
- `Connect`: open pool via `pgxpool.New`; ping to verify
- `Query`: `pool.Query(ctx, sql)`; scan rows into `[][]any`
- `ListTables`: query `information_schema.tables WHERE table_schema = 'public'`
- `DescribeTable`: query `information_schema.columns WHERE table_name = $1`
- `Close`: `pool.Close()`

### JSON (`internal/connector/json.go`)

- Library: `github.com/marcboeker/go-duckdb` (CGo)
- Single in-memory DuckDB instance per connection
- `Connect`: open DuckDB; validate file exists and is readable
- `Query`: rewrites bare table name to `read_json_auto('<path>')` if needed; executes via `database/sql`
- `ListTables`: returns `[]string{stem(path)}` — JSON has one implicit table
- `DescribeTable`: runs `DESCRIBE SELECT * FROM read_json_auto('<path>') LIMIT 0`
- `Close`: close DuckDB connection

### BigQuery (`internal/connector/bigquery.go`)

- Library: `cloud.google.com/go/bigquery`
- `Connect`: create `bigquery.NewClient`; if `Credentials` set, use `option.WithCredentialsFile`, else ADC
- `Query`: `client.Query(sql).Read(ctx)`; iterate `RowIterator` into `[][]any`
- `ListTables`: `client.Dataset(dataset).Tables(ctx)` iterator
- `DescribeTable`: `client.Dataset(dataset).Table(table).Metadata(ctx)` → `Schema`
- `Close`: `client.Close()`

---

## Dependencies

| Package                          | Purpose                    |
| -------------------------------- | -------------------------- |
| `github.com/spf13/cobra`         | CLI commands and flags     |
| `github.com/BurntSushi/toml`     | Config file parsing        |
| `github.com/jackc/pgx/v5`        | PostgreSQL driver          |
| `github.com/marcboeker/go-duckdb`| DuckDB — JSON file queries |
| `cloud.google.com/go/bigquery`   | BigQuery client            |
| `google.golang.org/api`          | GCP auth / ADC             |

---

## Testing Strategy

- **No mocks for connectors** — all connector tests hit real backends
- Postgres: testcontainers (`testcontainers-go`) spins a Postgres container per test run
- JSON: temp `.json` files written by test setup, cleaned up after
- BigQuery: integration tests require `BIGQUERY_TEST_PROJECT` env var; skipped in CI unless set
- Config: table-driven unit tests for TOML parse/write and DSN resolution
- Output: golden file tests for table/csv/json/md renderers

```bash
make test                          # runs all tests (Docker required for Postgres)
go test ./internal/connector/...   # connector tests only
go test ./internal/config/...      # config + secrets tests
BIGQUERY_TEST_PROJECT=my-proj go test ./internal/connector/ -run TestBigQuery -v
```

---

## CLI UX Rules

- All errors → stderr, exit code 1
- All data output → stdout (pipeline-safe)
- `--format` default: `table`
- Elapsed time printed after result in `table` format; omitted in csv/json/md
- `--quiet` flag: suppress elapsed time and row count footer
- `--config <path>` global flag: override default config path (`~/.data/config.toml`)

---

## Build

```makefile
build:
	go build -o bin/data ./cmd

test:
	go test ./... -race -count=1

lint:
	golangci-lint run ./...
```

CGo is required for DuckDB (JSON connector). Set `CGO_ENABLED=1` in all build targets.

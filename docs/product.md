# Data CLI — Product Specification

---

## Principles

- **Single binary** — one file, no runtime, no dependencies. `brew install`, done.
- **Depth over breadth** — fewer connectors that work perfectly beats many that are flaky
- **Pipe-friendly** — every output mode works in shell pipelines and CI
- **Local first** — works entirely offline, no account required, no telemetry by default
- **Open source always** — MIT licensed, no open core limits on CLI features

---

## Data Sources

| Source        | Priority | Notes               |
| ------------- | -------- | ------------------- |
| PostgreSQL    | P0       | via `pgx`           |
| SQLite        | P0       | via `go-sqlite3`    |
| CSV files     | P0       | via DuckDB          |
| JSON files    | P0       | via DuckDB          |
| MySQL         | P1       | via `go-sql-driver` |
| BigQuery      | P1       | via GCP client      |
| Parquet files | P1       | via DuckDB          |
| MariaDB       | P2       | MySQL-compatible    |

---

## Features

### Connection Management

Store named connections in `~/.data/config.toml`. Reference by name across all commands.

```
data connect <name> <dsn>     Save a named connection
data connect prod postgres://user:pass@host:5432/db
data connect local ./dev.db
data connect warehouse bigquery://project/dataset

data connections list          List all saved connections
data connections remove <name> Remove a connection
```

DSN can be a plain string, an environment variable reference (`env:DATABASE_URL`), or a secret manager path (`gcp-secret:projects/x/secrets/y/versions/latest`).

Per-connection options:

- `readonly = true` — refuse any non-SELECT query
- `row_limit = 1000` — cap results to avoid runaway queries
- `timeout = 30` — query timeout in seconds

---

### Interactive TUI

Launch with `data query <conn>` (no SQL argument).

**SQL editor pane**

- Syntax highlighting
- Multi-line editing
- Execute with `Ctrl+Enter`

**Results pane**

- Scrollable table with fixed headers
- Column resizing
- Copy cell / copy row

**Schema browser pane**

- Tree view of tables and columns
- Toggle with `Tab`

**Query history**

- Persisted across sessions in `~/.data/history`
- Searchable with `Ctrl+R`

---

### Non-interactive Query Mode

```bash
# Run query, print table to stdout
data query prod "SELECT * FROM orders LIMIT 10"

# Explicit format flags
data query prod "SELECT * FROM orders" --format table   # default
data query prod "SELECT * FROM orders" --format csv
data query prod "SELECT * FROM orders" --format json
data query prod "SELECT * FROM orders" --format md

# Write to file
data query prod "SELECT * FROM orders" --format csv --out orders.csv

# Pipe-friendly
data query prod "SELECT id, email FROM users" --format csv | \
  data query - "SELECT count(*) FROM stdin WHERE email LIKE '%@gmail.com'"
```

---

### File Source Querying

Query CSV, JSON, and Parquet files directly using SQL. Powered by embedded DuckDB — no separate installation required.

```bash
# Query a single file
data query orders.csv "SELECT status, count(*) FROM orders GROUP BY status"

# Join a file with a database table
data query prod \
  "SELECT u.name, o.total FROM users u JOIN 'uploads/orders.csv' o ON u.id = o.user_id"

# Auto-detect format from file extension
data query metrics.parquet "SELECT date, avg(value) FROM metrics GROUP BY date"
```

---

### Export

```bash
data export <conn> <sql> [flags]

Flags:
  --format   csv | json | md | table (default: csv)
  --out      output file path (default: stdout)
  --header   include header row (default: true)
  --limit    max rows (default: unlimited)

Examples:
  data export prod "SELECT * FROM orders" --format csv --out orders.csv
  data export prod "SELECT * FROM users" --format json
  data export prod "SELECT month, revenue FROM summary" --format md
```

---

### Schema Browser

```bash
data schema <conn>              # Interactive TUI browser
data schema <conn> --list       # Print table names
data schema <conn> <table>      # Print column definitions for a table

Example output (data schema prod orders):
  Table: orders
  ┌─────────────┬──────────────┬──────────┬─────────┐
  │ Column      │ Type         │ Nullable │ Default │
  ├─────────────┼──────────────┼──────────┼─────────┤
  │ id          │ uuid         │ NO       │ gen()   │
  │ user_id     │ uuid         │ NO       │         │
  │ status      │ varchar(20)  │ NO       │ pending │
  │ total       │ numeric(10,2)│ NO       │         │
  │ created_at  │ timestamptz  │ NO       │ now()   │
  └─────────────┴──────────────┴──────────┴─────────┘
```

---

### Terminal Visualization

```bash
data viz <conn> <sql> --chart <type>

Chart types:
  bar     Horizontal or vertical bar chart
  line    Line chart (requires date/numeric x-axis)

Examples:
  data viz prod \
    "SELECT DATE(created_at) as day, count(*) as orders FROM orders GROUP BY day ORDER BY day" \
    --chart line

  data viz prod \
    "SELECT status, count(*) as n FROM orders GROUP BY status" \
    --chart bar
```

---

### Data Gateway

Expose named connections to AI agents securely. Full specification in the Data Gateway docs.

```bash
data gateway                    Start the Data Gateway
  --config ~/.data/config.toml
  --port 7070
```

---

## Configuration File

```toml
# ~/.data/config.toml

[connections.prod]
driver      = "postgres"
dsn         = "env:DATABASE_URL"
readonly    = true
row_limit   = 1000
timeout     = 30

[connections.local]
driver  = "sqlite"
dsn     = "./dev.db"

[connections.warehouse]
driver  = "bigquery"
project = "my-gcp-project"
dataset = "analytics"

[defaults]
format    = "table"
row_limit = 500
timeout   = 30
```

---

## Out of Scope

- Write operations (INSERT / UPDATE / DELETE) — read-only tool
- MongoDB / NoSQL — SQL-first, v2 consideration
- Browser or web UI — terminal only
- Authentication / user management — single-user local tool
- Telemetry or analytics — no phone-home, ever

# Data CLI — Technical Specification

## Language

**Go** — single binary distribution, fast startup (~50ms), excellent cross-compilation, strong stdlib for CLI and networking.

---

## Repository Structure

```
data-cli/
├── cmd/
│   ├── main.go
│   ├── root.go             # Root command, global flags
│   ├── connect.go          # data connect
│   ├── connections.go      # data connections list/remove
│   ├── query.go            # data query (TUI + non-interactive)
│   ├── export.go           # data export
│   ├── schema.go           # data schema
│   ├── viz.go              # data viz
│   └── gateway.go          # data gateway
│
├── internal/
│   ├── config/
│   │   ├── config.go       # TOML load/save, validation
│   │   └── secrets.go      # env var + secret manager DSN resolution
│   │
│   ├── connector/
│   │   ├── connector.go    # Connector interface
│   │   ├── registry.go     # Driver registration by name
│   │   ├── postgres.go
│   │   ├── mysql.go
│   │   ├── sqlite.go
│   │   ├── bigquery.go
│   │   └── duckdb.go       # File sources — CSV, Parquet, JSON
│   │
│   ├── tui/
│   │   ├── app.go          # Root bubbletea model
│   │   ├── editor.go       # SQL editor pane
│   │   ├── results.go      # Results table pane
│   │   ├── schema.go       # Schema browser pane
│   │   └── history.go      # Query history
│   │
│   ├── output/
│   │   ├── table.go        # Terminal table renderer
│   │   ├── csv.go
│   │   ├── json.go
│   │   └── markdown.go
│   │
│   ├── viz/
│   │   └── chart.go        # Terminal bar/line charts
│   │
│   └── gateway/
│       ├── server.go       # HTTP + WebSocket server
│       ├── auth.go         # Token validation
│       ├── router.go       # Request routing to connectors
│       ├── roles.go        # Role-based access enforcement
│       ├── guard.go        # SELECT-only SQL enforcement
│       └── adapter/
│           ├── mcp/        # MCP protocol adapter
│           │   ├── stdio.go
│           │   └── http.go
│           └── openai/     # OpenAI tool calling adapter (v2)
│               └── tools.go
│
├── go.mod
├── go.sum
├── Makefile
├── .goreleaser.yaml
└── README.md
```

---

## Connector Interface

Every data source implements a single interface. The rest of the codebase only talks to this interface — never to a specific driver.

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

Adding a new connector = implement this interface + register in `registry.go`. Nothing else changes.

---

## DuckDB for File Sources

DuckDB is embedded as a CGo dependency. It handles CSV, Parquet, and JSON querying transparently — the file path becomes the table reference.

```go
// internal/connector/duckdb.go

// File paths are passed as table references in SQL:
// SELECT * FROM 'orders.csv' LIMIT 10
// SELECT * FROM 'metrics.parquet' WHERE date > '2024-01-01'
// SELECT * FROM read_json_auto('events.json')

func (d *DuckDBConnector) Query(ctx context.Context, sql string, opts QueryOptions) (*Result, error) {
    // DuckDB handles file:// paths natively in SQL
    // No pre-processing needed for standard file queries
}
```

For cross-source queries (file + database), DuckDB's `postgres_scan` and `mysql_scan` extensions are used to federate queries.

---

## TUI Architecture

Built with Bubbletea — a functional, message-driven TUI framework. Three panes compose the main view:

```
┌─────────────────────────────────────────────┐
│  Schema Browser  │  SQL Editor               │
│  (toggle: Tab)   │                           │
│                  │  SELECT *                 │
│  ▼ orders        │  FROM orders              │
│    id            │  WHERE status = 'pending' │
│    user_id       │  LIMIT 100                │
│    status        │                           │
│    total         ├───────────────────────────┤
│    created_at    │  Results                  │
│  ▶ users         │  id  │ user_id │ status   │
│  ▶ products      │  ... │ ...     │ pending  │
└─────────────────────────────────────────────┘
```

```go
// internal/tui/app.go

type Model struct {
    editor   editor.Model
    results  results.Model
    schema   schema.Model
    focus    pane          // editor | results | schema
    conn     connector.Connector
    history  history.Model
}

type Msg interface{}
type QueryRunMsg struct{ SQL string }
type QueryDoneMsg struct{ Result *connector.Result; Err error }
type SchemaLoadedMsg struct{ Tables []string }
```

Panes communicate via messages, not direct function calls. Each pane is independently testable.

---

## Data Gateway Architecture

The Gateway runs as a subprocess of `data gateway`. It exposes the connector layer over HTTP/WebSocket with auth and role enforcement layered on top.

```
Incoming request (any AI tool)
         │
         ▼
   gateway/server.go
         │
   gateway/auth.go          ← validate workspace token + request token
         │
   gateway/roles.go         ← check role has access to this connection + table
         │
   gateway/guard.go         ← reject non-SELECT SQL
         │
   connector/registry.go    ← route to correct connector
         │
         ▼
   Database / File
```

### SQL Guard

```go
// internal/gateway/guard.go

func EnforceReadOnly(sql string) error {
    stmt, err := sqlparser.Parse(strings.TrimSpace(sql))
    if err != nil {
        return fmt.Errorf("invalid SQL: %w", err)
    }
    switch stmt.(type) {
    case *sqlparser.Select, *sqlparser.With, *sqlparser.Explain, *sqlparser.Show:
        return nil
    default:
        return errors.New("only SELECT queries are permitted through the gateway")
    }
}
```

This check runs before role enforcement and before the connector is called. Writes cannot pass through regardless of configuration.

### Protocol Adapters

Adapters are thin translation layers — they convert protocol-specific requests into the gateway's internal `QueryRequest` struct.

```go
// internal/gateway/adapter/mcp/stdio.go

// Receives MCP tool call → extracts sql + connection → calls gateway.Query()
// Returns MCP tool result format

// internal/gateway/adapter/openai/tools.go  (v2)

// Receives OpenAI function call → extracts sql + connection → calls gateway.Query()
// Returns OpenAI tool result format
```

Adding a new AI protocol = one new file in `adapter/`. The connector layer, auth, role enforcement, and SQL guard are untouched.

---

## Dependencies

### Core CLI

| Package       | Purpose                     |
| ------------- | --------------------------- |
| `spf13/cobra` | Command structure and flags |
| `spf13/viper` | Config file management      |

### TUI

| Package                   | Purpose                          |
| ------------------------- | -------------------------------- |
| `charmbracelet/bubbletea` | TUI framework                    |
| `charmbracelet/bubbles`   | Table, textarea, list components |
| `charmbracelet/lipgloss`  | Terminal styling                 |

### Connectors

| Package                        | Purpose                     |
| ------------------------------ | --------------------------- |
| `jackc/pgx/v5`                 | PostgreSQL                  |
| `go-sql-driver/mysql`          | MySQL / MariaDB             |
| `mattn/go-sqlite3`             | SQLite (CGo)                |
| `cloud.google.com/go/bigquery` | BigQuery                    |
| `marcboeker/go-duckdb`         | DuckDB — file sources (CGo) |

### Gateway

| Package             | Purpose                 |
| ------------------- | ----------------------- |
| `gorilla/websocket` | WebSocket tunnel        |
| `golang-jwt/jwt/v5` | Token validation        |
| `xwb1989/sqlparser` | SQL guard / AST parsing |

### Visualization

| Package                 | Purpose                  |
| ----------------------- | ------------------------ |
| `guptarohit/asciigraph` | Terminal line/bar charts |

### Config

| Package           | Purpose      |
| ----------------- | ------------ |
| `BurntSushi/toml` | TOML parsing |

### MCP Adapter

| Package                       | Purpose                     |
| ----------------------------- | --------------------------- |
| `modelcontextprotocol/go-sdk` | MCP protocol (stdio + HTTP) |

---

## Build & Release

```makefile
# Makefile

build:
	go build -o bin/data ./cmd

test:
	go test ./... -race -cover

lint:
	golangci-lint run

# Cross-compile
release-local:
	GOOS=darwin  GOARCH=arm64  go build -o dist/data-darwin-arm64  ./cmd
	GOOS=darwin  GOARCH=amd64  go build -o dist/data-darwin-amd64  ./cmd
	GOOS=linux   GOARCH=amd64  go build -o dist/data-linux-amd64   ./cmd
	GOOS=linux   GOARCH=arm64  go build -o dist/data-linux-arm64   ./cmd
	GOOS=windows GOARCH=amd64  go build -o dist/data-windows-amd64.exe ./cmd
```

GoReleaser handles GitHub releases, Homebrew tap publishing, and Docker image builds in CI.

---

## Development Priorities

| Priority | Component                   | Reason                                               |
| -------- | --------------------------- | ---------------------------------------------------- |
| 1        | `connector/`                | Core value — get Postgres + SQLite + CSV right first |
| 2        | `cmd/query` non-interactive | Daily use, scripting, CI                             |
| 3        | `cmd/export`                | Natural companion to query                           |
| 4        | `gateway/`                  | AI agent integration — core differentiator           |
| 5        | `gateway/adapter/mcp/`      | Claude Code validation                               |
| 6        | `tui/`                      | Polish comes after utility                           |
| 7        | `viz/`                      | Nice to have, not blocking                           |
| 8        | `gateway/adapter/openai/`   | v2, after community feedback                         |

---

## Testing Strategy

- Unit tests per connector against real DBs in Docker (testcontainers)
- Integration tests for gateway auth, role enforcement, and SQL guard
- TUI tested with bubbletea's test helpers
- No mocks for connectors — real DB connections in test suite

```bash
# Run tests with Docker-based DB fixtures
make test

# Run gateway security tests only
go test ./internal/gateway/... -run TestSQLGuard -v
go test ./internal/gateway/... -run TestRoleEnforcement -v
```

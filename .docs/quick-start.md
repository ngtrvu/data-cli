# Quick Start

## Install

```bash
curl -sSL https://raw.githubusercontent.com/ngtrvu/data-cli/main/install.sh | sh
```

## Build from source

**Prerequisites:** Go 1.22+, GCC (for DuckDB CGo), Docker (for tests)

```bash
git clone https://github.com/ngtrvu/data-cli
cd data-cli
go mod download
make build
# → bin/data
```

---

## Config

Copy the example config and edit it:

```bash
cp config.example.toml bin/config/config.toml
```

Or let the CLI create it automatically on first `connect`.

Config is loaded from the first location that exists:

1. `<bin-dir>/config/config.toml` — portable, next to the binary
2. `~/.data/config/config.toml` — home install

---

## Try it

```bash
# Add a connection
bin/data connect prod postgres://user:pass@localhost:5432/mydb

# List connections
bin/data list

# Inspect schema
bin/data schema prod
bin/data schema prod orders

# Run a query
bin/data query prod "SELECT * FROM orders LIMIT 10"
bin/data query prod "SELECT * FROM orders" --format json

# Remove a connection
bin/data remove prod
```

---

## Project Structure

```
cmd/                  CLI entry point and commands
internal/config/      Config file load/save, DSN resolution
internal/connector/   Connector interface + drivers
  postgres/           PostgreSQL via pgx
  json/               JSON files via DuckDB
  bigquery/           BigQuery via GCP client
internal/output/      Output formatters (table, csv, json, md)
```

---

## Adding a New Connector

1. Create `internal/connector/<name>/<name>.go`
2. Implement the `connector.Connector` interface
3. Call `connector.Register("<name>", ...)` in `init()`
4. Blank-import the package in `cmd/connect.go`, `cmd/query.go`, `cmd/schema.go`

---

Connector tests hit real backends — Docker is required for Postgres.

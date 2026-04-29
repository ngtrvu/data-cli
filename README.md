# Data CLI

**Data CLI for your AI agents.**

Data CLI lets you connect to any data source — a database, a file, a data warehouse — and query it from the terminal. It is designed to work seamlessly with AI coding agents like Claude Code, OpenCode, and Gemini CLI, so your agent can explore, understand, and query your data without ever touching your credentials.

---

## Install

```bash
curl -sSL https://raw.githubusercontent.com/ngtrvu/data-cli/main/install.sh | sh
```

Or build from source — see [quick start](./docs/quick-start.md).

---

## About Data CLI

When you work with an AI coding agent, you often need to give it access to your data — a production database, a JSON log file, a BigQuery dataset. Today, that means copy-pasting schema, writing one-off scripts, or exposing credentials you'd rather keep private.

Data CLI solves this cleanly:

- You define your data sources once in a config file
- Your agent calls `data query`, `data schema`, or `data list` like any other CLI tool
- Credentials stay on your machine — the agent only ever sees connection names and results

It works for humans too. Run queries, inspect schemas, and explore data directly from your terminal.

---

## Quick Start

See the [Quick Start](./docs/quick-start.md) for setup, config, and first commands.

---

## Features

**Initialize a local config**
Set up a project-level config file in the current directory. Local config takes priority over your global `~/.data/config/config.toml`.

```bash
data init   # creates .data/config.toml in the current directory
```

**Connect to anything**
Add a named connection to a Postgres database, a JSON file, or a BigQuery dataset in one command. Reference it by name everywhere else.

```bash
data connect prod postgres://user:pass@localhost:5432/mydb
data connect events ./logs/events.json
data connect warehouse --driver bigquery --project my-project --dataset analytics
```

**Query with SQL**
Run SQL against any connection. Get results as a table, CSV, JSON, or Markdown.

```bash
data query prod "SELECT id, email FROM users LIMIT 10"
data query events "SELECT type, count(*) FROM events GROUP BY type"
data query warehouse "SELECT date, sum(revenue) FROM orders GROUP BY date"

# Agent-friendly JSON output
data query prod "SELECT * FROM orders" --format json

# Run a query from a .sql file
data query prod --file path/to/query.sql

# Variable substitution in SQL files or inline queries
data query prod --file report.sql --var cutoff=2026-01-01
data query prod "SELECT * FROM users WHERE created_at > '{{cutoff}}'" --var cutoff=2026-04-01
```

**Inspect schemas**
See what tables and columns exist before writing a query. Agents use this to understand your data structure automatically.

```bash
data schema prod                         # list all tables
data schema prod orders                  # describe columns
data schema prod orders --format json    # machine-readable for agents
```

**Manage connections**

```bash
data list             # show all configured sources
data remove staging   # remove a connection
```

**Upgrade**

```bash
data upgrade          # upgrade to the latest release
data version          # print the current version
```

---

## Architecture

Data CLI is a single Go binary. No runtime, no daemon, no account required.

```
.data/config.toml            ← local project config (highest priority)
~/.data/config/config.toml   ← global config (fallback)
        │
        ▼
   data init / connect / query / schema / list
        │
        ▼
┌───────────────────────────────┐
│         Connector Layer        │
│  postgres  │  json  │ bigquery │
└───────────────────────────────┘
        │
        ▼
   stdout (table / csv / json / md)
```

**Three data source types:**


| Type           | Driver     | How it works                            |
| -------------- | ---------- | --------------------------------------- |
| Database       | `postgres` | Connects via `pgx`, standard SQL        |
| File           | `json`     | Embedded DuckDB, query JSON with SQL    |
| Data warehouse | `bigquery` | GCP client, uses ADC or service account |


**Config file** (`.data/config.toml` locally, or `~/.data/config/config.toml` globally):

```toml
[connections.prod]
driver    = "postgres"
dsn       = "env:DATABASE_URL"   # literal, env:VAR, or gcp-secret:...
readonly  = true
row_limit = 1000

[connections.events]
driver = "json"
path   = "./logs/events.json"

[connections.warehouse]
driver  = "bigquery"
project = "my-gcp-project"
dataset = "analytics"

[defaults]
row_limit = 500
timeout   = 30
```

See `[config.example.toml](./config.example.toml)` for all options.

---

## License

MIT — see [LICENSE](./LICENSE).
# Data CLI

> A universal data tool for developers. Connect, query, explore, export, and visualize data from any source — from the terminal.

---

## What It Is

Data CLI is a single binary that replaces the fragmented set of tools developers use to work with data daily:

- `psql` / `mysql` / `sqlite3` for querying
- Custom scripts for export
- Manual copy-paste to feed data to AI agents
- Separate tools for files vs databases

One tool. Every source. Zero runtime dependencies.

---

## Core Capabilities

**Query anything**
Connect to Postgres, MySQL, SQLite, BigQuery, or query CSV, Parquet, and JSON files directly — all with the same interface.

**Interactive TUI**
A fast terminal UI with a SQL editor, schema browser, scrollable results, and query history. Open it like a text editor, not a legacy REPL.

**Scripting and export**
Pipe-friendly non-interactive mode. Run a query, get CSV. Automate in shell scripts. Use in CI pipelines.

**Terminal visualization**
Bar and line charts rendered directly in the terminal. No Python, no Jupyter, no browser required.

**Data Gateway**  
Expose your data sources securely to any AI agent — Claude Code, Codex, OpenCode, or a custom agent. Credentials never leave your machine. You define exactly what is visible.

---

## Why Open Source

Data tooling should be owned by the developer, not locked to a vendor or a cloud. Data CLI is open source so you can read the code, trust it with your production credentials, and run it anywhere — local machine, on-prem server, or cloud VM.

---

## Quick Start

```bash
# Install
brew install data-cli

# Add a connection
data connect prod postgres://user:pass@localhost:5432/mydb

# Query interactively
data query prod

# Run a one-off query
data query prod "SELECT count(*) FROM orders WHERE created_at > now() - interval '7 days'"

# Export to CSV
data export prod "SELECT * FROM orders LIMIT 1000" --format csv --out orders.csv

# Query a CSV file directly
data query orders.csv "SELECT status, count(*) FROM orders GROUP BY status"

# Start the Data Gateway for AI agent access
data gateway --config ~/.data-cli/config.toml
```

---

## Who It's For

**Backend and data engineers** who live in the terminal and want a single fast tool across all their data sources.

**AI-native developers** using Claude Code, Codex, or OpenCode — who today manually copy-paste schema and data into chat. The Data Gateway solves this permanently.

**Anyone who works with files and databases** and wants SQL as a universal query language across both.

---

## Project Status

Currently in active development. Built in Go. Contributions welcome.

See [product-spec.md](./.docs/product.md) for the full feature roadmap.
See [tech-spec.md](./.docs/tech.md) for architecture and technical decisions.
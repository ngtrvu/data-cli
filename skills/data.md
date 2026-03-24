You have access to the `data` CLI for querying data sources.

## Workflow

Always follow this order:
1. `data list` — see what connections are available
2. `data schema <name>` — list tables in a connection
3. `data schema <name> <table>` — inspect columns before writing SQL
4. `data query <name> "<sql>"` — run the query

## Commands

```bash
# List all configured data sources
data list
data list --format json

# Inspect schema
data schema <name>                        # list tables
data schema <name> <table>                # describe columns
data schema <name> <table> --format json  # structured output

# Query
data query <name> "<sql>"
data query <name> "<sql>" --format json
data query <name> "<sql>" --format csv
data query <name> "<sql>" --limit 50

# Manage connections
data connect <name> <dsn>
data remove <name>
```

## Rules

- Always run `data schema` before querying an unfamiliar table
- Credentials are never visible — only connection names are exposed
- Use `--format json` when you need to process results
- Use `--limit` to avoid large result sets when exploring
- Errors go to stderr with exit code 1

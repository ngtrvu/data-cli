# Using Data CLI with Claude Code

Two things to set up: install the binary, then tell Claude Code how to use it.

---

## 1. Install

```bash
# Build from source
git clone https://github.com/ngtrvu/data-cli
cd data-cli
make build
sudo cp bin/data /usr/local/bin/data
```

---

## 2. Add a connection

```bash
data connect prod postgres://user:pass@localhost:5432/mydb
data connect events ./logs/events.json
data connect warehouse --driver bigquery --project my-project --dataset analytics

# Verify
data list
```

---

## 3. Install the skill (global — works in every project)

Copy the skill file to your Claude Code skills directory:

```bash
cp skills/data.md ~/.claude/skills/data.md
```

Claude Code will now know how to use `data` commands in any project.

---

## 4. Or add a CLAUDE.md to your project

If you prefer per-project setup, copy the template into your project:

```bash
cp skills/data.md your-project/CLAUDE.md
```

---

## Try it

Open Claude Code in any project and ask:

```
What data sources do I have?
Show me the schema for the orders table in prod
How many orders were placed this week?
```

Claude Code will run `data list` → `data schema` → `data query` automatically.

---

## Config location

```
~/.data/config/config.toml       # home install
<bin-dir>/config/config.toml     # portable / next to binary
```

See `config.example.toml` for all available options.

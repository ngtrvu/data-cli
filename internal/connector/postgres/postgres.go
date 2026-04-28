package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ngtrvu/data-cli/internal/config"
	"github.com/ngtrvu/data-cli/internal/connector"
)

func init() {
	connector.Register("postgres", func(cfg config.ConnectionConfig) (connector.Connector, error) {
		return &postgresConnector{cfg: cfg}, nil
	})
}

type postgresConnector struct {
	cfg  config.ConnectionConfig
	pool *pgxpool.Pool
}

func (p *postgresConnector) Connect(ctx context.Context) error {
	dsn, err := config.Resolve(p.cfg.DSN)
	if err != nil {
		return err
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return fmt.Errorf("postgres connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return fmt.Errorf("postgres ping: %w", err)
	}
	p.pool = pool
	return nil
}

func (p *postgresConnector) Query(ctx context.Context, sql string, opts connector.QueryOptions) (*connector.Result, error) {
	if opts.ReadOnly && !isSelect(sql) {
		return nil, fmt.Errorf("connection is read-only: only SELECT queries are allowed")
	}
	query := sql
	if opts.RowLimit > 0 {
		query = fmt.Sprintf("SELECT * FROM (%s) _q LIMIT %d", sql, opts.RowLimit)
	}

	start := time.Now()
	rows, err := p.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	fields := rows.FieldDescriptions()
	cols := make([]connector.Column, len(fields))
	for i, f := range fields {
		cols[i] = connector.Column{Name: f.Name, Type: oidName(f.DataTypeOID)}
	}

	var result [][]any
	for rows.Next() {
		vals, err := rows.Values()
		if err != nil {
			return nil, err
		}
		result = append(result, vals)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &connector.Result{Columns: cols, Rows: result, Elapsed: time.Since(start)}, nil
}

func (p *postgresConnector) ListTables(ctx context.Context) ([]string, error) {
	rows, err := p.pool.Query(ctx,
		`SELECT table_name FROM information_schema.tables
		 WHERE table_schema = 'public' AND table_type = 'BASE TABLE'
		 ORDER BY table_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tables = append(tables, name)
	}
	return tables, rows.Err()
}

func (p *postgresConnector) DescribeTable(ctx context.Context, table string) ([]connector.Column, error) {
	schema := "public"
	tableName := table
	if parts := strings.SplitN(table, ".", 2); len(parts) == 2 {
		schema, tableName = parts[0], parts[1]
	}
	rows, err := p.pool.Query(ctx,
		`SELECT column_name, data_type, is_nullable, column_default
		 FROM information_schema.columns
		 WHERE table_schema = $1 AND table_name = $2
		 ORDER BY ordinal_position`, schema, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var cols []connector.Column
	for rows.Next() {
		var name, typ, nullable string
		var def *string
		if err := rows.Scan(&name, &typ, &nullable, &def); err != nil {
			return nil, err
		}
		cols = append(cols, connector.Column{
			Name:     name,
			Type:     typ,
			Nullable: nullable == "YES",
			Default:  def,
		})
	}
	return cols, rows.Err()
}

func (p *postgresConnector) Close() error {
	if p.pool != nil {
		p.pool.Close()
	}
	return nil
}

func isSelect(sql string) bool {
	s := strings.TrimSpace(strings.ToUpper(sql))
	return strings.HasPrefix(s, "SELECT") ||
		strings.HasPrefix(s, "WITH") ||
		strings.HasPrefix(s, "EXPLAIN") ||
		strings.HasPrefix(s, "SHOW")
}

// oidName maps common Postgres OIDs to human-readable type names.
var oidNames = map[uint32]string{
	16: "bool", 20: "int8", 21: "int2", 23: "int4", 25: "text",
	700: "float4", 701: "float8", 1042: "char", 1043: "varchar",
	1114: "timestamp", 1184: "timestamptz", 2950: "uuid", 3802: "jsonb",
}

func oidName(oid uint32) string {
	if name, ok := oidNames[oid]; ok {
		return name
	}
	return fmt.Sprintf("oid:%d", oid)
}

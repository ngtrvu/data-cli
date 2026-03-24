package json

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/marcboeker/go-duckdb"
	"github.com/ngtrvu/data-cli/internal/config"
	"github.com/ngtrvu/data-cli/internal/connector"
)

func init() {
	connector.Register("json", func(cfg config.ConnectionConfig) (connector.Connector, error) {
		return &jsonConnector{cfg: cfg}, nil
	})
}

type jsonConnector struct {
	cfg config.ConnectionConfig
	db  *sql.DB
}

func (j *jsonConnector) Connect(ctx context.Context) error {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		return fmt.Errorf("duckdb open: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return fmt.Errorf("duckdb ping: %w", err)
	}
	j.db = db
	return nil
}

func (j *jsonConnector) Query(ctx context.Context, query string, opts connector.QueryOptions) (*connector.Result, error) {
	// Replace the table stem (filename without extension) with the DuckDB read_json_auto call.
	stem := tableStem(j.cfg.Path)
	query = strings.ReplaceAll(query, stem, fmt.Sprintf("read_json_auto('%s')", j.cfg.Path))

	if opts.RowLimit > 0 {
		query = fmt.Sprintf("SELECT * FROM (%s) _q LIMIT %d", query, opts.RowLimit)
	}

	start := time.Now()
	rows, err := j.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("duckdb query: %w", err)
	}
	defer rows.Close()

	colNames, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	cols := make([]connector.Column, len(colNames))
	for i, n := range colNames {
		cols[i] = connector.Column{Name: n, Nullable: true}
	}

	var result [][]any
	for rows.Next() {
		vals := make([]any, len(colNames))
		ptrs := make([]any, len(colNames))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		row := make([]any, len(vals))
		copy(row, vals)
		result = append(result, row)
	}

	return &connector.Result{Columns: cols, Rows: result, Elapsed: time.Since(start)}, rows.Err()
}

func (j *jsonConnector) ListTables(ctx context.Context) ([]string, error) {
	return []string{tableStem(j.cfg.Path)}, nil
}

func (j *jsonConnector) DescribeTable(ctx context.Context, table string) ([]connector.Column, error) {
	// DuckDB DESCRIBE returns: column_name, column_type, null, key, default, extra
	rows, err := j.db.QueryContext(ctx,
		fmt.Sprintf("DESCRIBE SELECT * FROM read_json_auto('%s') LIMIT 0", j.cfg.Path))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cols []connector.Column
	for rows.Next() {
		var name, typ string
		var null, key, def, extra any
		if err := rows.Scan(&name, &typ, &null, &key, &def, &extra); err != nil {
			return nil, err
		}
		cols = append(cols, connector.Column{Name: name, Type: typ, Nullable: true})
	}
	return cols, rows.Err()
}

func (j *jsonConnector) Close() error {
	if j.db != nil {
		return j.db.Close()
	}
	return nil
}

func tableStem(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

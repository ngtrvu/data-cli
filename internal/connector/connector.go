package connector

import (
	"context"
	"time"
)

// QueryOptions controls query execution behaviour.
type QueryOptions struct {
	RowLimit int
	Timeout  time.Duration
	ReadOnly bool
}

// Column describes a single column returned by a query or schema inspection.
type Column struct {
	Name     string
	Type     string
	Nullable bool
	Default  *string
}

// Result holds the output of a query.
type Result struct {
	Columns []Column
	Rows    [][]any
	Elapsed time.Duration
}

// Connector is the single interface every data source must implement.
type Connector interface {
	Connect(ctx context.Context) error
	Query(ctx context.Context, sql string, opts QueryOptions) (*Result, error)
	ListTables(ctx context.Context) ([]string, error)
	DescribeTable(ctx context.Context, table string) ([]Column, error)
	Close() error
}

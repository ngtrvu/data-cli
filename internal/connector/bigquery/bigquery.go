package bigquery

import (
	"context"
	"fmt"
	"time"

	bq "cloud.google.com/go/bigquery"
	"github.com/ngtrvu/data-cli/internal/config"
	"github.com/ngtrvu/data-cli/internal/connector"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

func init() {
	connector.Register("bigquery", func(cfg config.ConnectionConfig) (connector.Connector, error) {
		return &bqConnector{cfg: cfg}, nil
	})
}

type bqConnector struct {
	cfg    config.ConnectionConfig
	client *bq.Client
}

func (b *bqConnector) Connect(ctx context.Context) error {
	var opts []option.ClientOption
	if b.cfg.Credentials != "" {
		opts = append(opts, option.WithCredentialsFile(b.cfg.Credentials))
	}
	client, err := bq.NewClient(ctx, b.cfg.Project, opts...)
	if err != nil {
		return fmt.Errorf("bigquery connect: %w", err)
	}
	b.client = client
	return nil
}

func (b *bqConnector) Query(ctx context.Context, sql string, opts connector.QueryOptions) (*connector.Result, error) {
	start := time.Now()
	it, err := b.client.Query(sql).Read(ctx)
	if err != nil {
		return nil, fmt.Errorf("bigquery query: %w", err)
	}

	schema := it.Schema
	cols := make([]connector.Column, len(schema))
	for i, f := range schema {
		cols[i] = connector.Column{
			Name:     f.Name,
			Type:     string(f.Type),
			Nullable: !f.Required,
		}
	}

	var rows [][]any
	for {
		if opts.RowLimit > 0 && len(rows) >= opts.RowLimit {
			break
		}
		var row []bq.Value
		if err := it.Next(&row); err == iterator.Done {
			break
		} else if err != nil {
			return nil, err
		}
		anyRow := make([]any, len(row))
		for i, v := range row {
			anyRow[i] = v
		}
		rows = append(rows, anyRow)
	}

	return &connector.Result{Columns: cols, Rows: rows, Elapsed: time.Since(start)}, nil
}

func (b *bqConnector) ListTables(ctx context.Context) ([]string, error) {
	it := b.client.Dataset(b.cfg.Dataset).Tables(ctx)
	var tables []string
	for {
		t, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		tables = append(tables, t.TableID)
	}
	return tables, nil
}

func (b *bqConnector) DescribeTable(ctx context.Context, table string) ([]connector.Column, error) {
	meta, err := b.client.Dataset(b.cfg.Dataset).Table(table).Metadata(ctx)
	if err != nil {
		return nil, fmt.Errorf("bigquery describe: %w", err)
	}
	cols := make([]connector.Column, len(meta.Schema))
	for i, f := range meta.Schema {
		cols[i] = connector.Column{
			Name:     f.Name,
			Type:     string(f.Type),
			Nullable: !f.Required,
		}
	}
	return cols, nil
}

func (b *bqConnector) Close() error {
	if b.client != nil {
		return b.client.Close()
	}
	return nil
}

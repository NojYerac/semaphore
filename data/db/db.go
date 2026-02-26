package db

import (
	"context"

	sq "github.com/Masterminds/squirrel"
	_ "github.com/lib/pq"
	"github.com/nojyerac/go-lib/db"
	"github.com/nojyerac/semaphore/data"
)

var _ data.Source = (*DataSource)(nil)

type DataSource struct {
	db db.Database
}

func NewDataSource(ctx context.Context, database db.Database) *DataSource {
	dataSource := &DataSource{
		db: database,
	}
	dataSource.Migrate(ctx)
	return dataSource
}

func (d *DataSource) Migrate(ctx context.Context) error {
	_, err := d.db.Exec(ctx, `
	CREATE TABLE IF NOT EXISTS flags (
		id SERIAL PRIMARY KEY,
		name TEXT NOT NULL,
		enabled BOOLEAN NOT NULL
	);
	`)
	return err
}

func (d *DataSource) GetFlags(ctx context.Context) ([]data.Flag, error) {
	query := sq.Select("id", "name", "enabled").From("flags")
	sql, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}
	flags := make([]data.Flag, 0)
	err = d.db.Select(ctx, &flags, sql, args...)
	return flags, err
}

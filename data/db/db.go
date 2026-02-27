package db

import (
	"context"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/nojyerac/go-lib/db"
	"github.com/nojyerac/semaphore/data"
)

var _ data.Source = (*DataSource)(nil)

type DataSource struct {
	db db.Database
}

func NewDataSource(ctx context.Context, database db.Database) (*DataSource, error) {
	dataSource := &DataSource{
		db: database,
	}
	err := dataSource.Migrate(ctx)
	return dataSource, err
}

func (d *DataSource) Migrate(ctx context.Context) error {
	_, err := d.db.Exec(ctx, `
	CREATE TABLE IF NOT EXISTS feature_flags (
		id UUID PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		description TEXT,
		enabled BOOLEAN NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP NOT NULL DEFAULT NOW()
	);

	CREATE TYPE strategy_type AS ENUM ('percentage', 'user_id', 'custom');

	CREATE TABLE IF NOT EXISTS strategies (
		id SERIAL PRIMARY KEY,
		flag_id UUID REFERENCES feature_flags(id) ON DELETE CASCADE,
		type strategy_type NOT NULL,
		payload JSONB NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_strategies_flag_id ON strategies(flag_id);

	CREATE TABLE IF NOT EXISTS audit_logs (
		id UUID PRIMARY KEY,
		flag_id UUID REFERENCES feature_flags(id),
		action TEXT NOT NULL,
		timestamp TIMESTAMP NOT NULL DEFAULT NOW(),
		user TEXT NOT NULL,
		details TEXT
	);

	CREATE FUNCTION update_timestamp() RETURNS trigger AS $$
	BEGIN
		NEW.updated_at = NOW();
		RETURN NEW;
	END;
	$$ LANGUAGE plpgsql;

	CREATE TRIGGER update_feature_flags_updated_at
	BEFORE UPDATE ON feature_flags
	FOR EACH ROW
	EXECUTE FUNCTION update_timestamp();
`)
	return err
}

func getFlagBaseQuery() sq.SelectBuilder {
	return sq.Select(
		"f.id",
		"f.name",
		"f.description",
		"f.enabled",
		"f.created_at",
		"f.updated_at",
		"JSON_AGG(JSON_BUILD_OBJECT('type', s.type, 'payload', s.payload)) AS strategies",
	).
		From("feature_flags f").
		LeftJoin("strategies s ON f.id = s.flag_id").
		GroupBy("f.id", "f.name", "f.description", "f.enabled", "f.created_at", "f.updated_at")
}

func (d *DataSource) GetFlags(ctx context.Context) ([]*data.FeatureFlag, error) {
	query := getFlagBaseQuery()
	sql, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}
	flags := make([]*data.FeatureFlag, 0)
	err = d.db.Select(ctx, &flags, sql, args...)
	return flags, err
}

func (d *DataSource) GetFlagByID(ctx context.Context, id string) (*data.FeatureFlag, error) {
	query := getFlagBaseQuery().Where(sq.Eq{"f.id": id})
	sql, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}
	flag := &data.FeatureFlag{}
	err = d.db.Get(ctx, flag, sql, args...)
	if err != nil {
		return nil, err
	}
	return flag, nil
}

func (d *DataSource) CreateFlag(ctx context.Context, flag *data.FeatureFlag) (string, error) {
	id := uuid.New().String()
	query := sq.Insert("feature_flags").
		Columns("id", "name", "description", "enabled").
		Values(id, flag.Name, flag.Description, flag.Enabled)
	sql, args, err := query.ToSql()
	if err != nil {
		return "", err
	}
	tx, err := d.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() {
		if err != nil || recover() != nil {
			_ = tx.Rollback(ctx)
		}
	}()
	_, err = d.db.Exec(ctx, sql, args...)
	if err != nil {
		return "", err
	}
	stratQuery := sq.Insert("strategies").
		Columns("flag_id", "type", "payload")
	for _, strategy := range flag.Strategies {
		stratQuery = stratQuery.Values(flag.ID, strategy.Type, strategy.Payload)
	}
	sql, args, err = stratQuery.ToSql()
	if err != nil {
		return id, err
	}
	_, err = d.db.Exec(ctx, sql, args...)
	if err != nil {
		return "", err
	}
	err = tx.Commit(ctx)
	return id, nil
}

func (d *DataSource) UpdateFlag(ctx context.Context, flag *data.FeatureFlag) error {
	query := sq.Update("feature_flags").
		Set("name", flag.Name).
		Set("description", flag.Description).
		Set("enabled", flag.Enabled).
		Where(sq.Eq{"id": flag.ID})
	sql, args, err := query.ToSql()
	if err != nil {
		return err
	}
	_, err = d.db.Exec(ctx, sql, args...)
	return err
}

func (d *DataSource) DeleteFlag(ctx context.Context, id string) error {
	query := sq.Delete("feature_flags").Where(sq.Eq{"id": id})
	sql, args, err := query.ToSql()
	if err != nil {
		return err
	}
	_, err = d.db.Exec(ctx, sql, args...)
	return err
}

func (d *DataSource) EvaluateFlag(ctx context.Context, flagID, userID string) (bool, error) {
	// For simplicity, this is a stub. A real implementation would evaluate the strategies.
	flag, err := d.GetFlagByID(ctx, flagID)
	if err != nil {
		return false, err
	}
	return flag.Enabled, nil
}

func (d *DataSource) GetAuditLogs(ctx context.Context, flagID string) ([]data.AuditLog, error) {
	return nil, nil // Stub implementation
}

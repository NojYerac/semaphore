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

	DO $$
	BEGIN
		CREATE TYPE strategy_type AS ENUM ('percentage_rollout', 'user_targeting', 'group_targeting');
	EXCEPTION
		WHEN duplicate_object THEN
			-- type already exists, ignore
			NULL;
	END;
	$$;

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
		user_id UUID NOT NULL,
		details TEXT
	);

	CREATE OR REPLACE FUNCTION update_timestamp() RETURNS trigger AS $$
	BEGIN
		NEW.updated_at = NOW();
		RETURN NEW;
	END;
	$$ LANGUAGE plpgsql;

	DO $$
	BEGIN
		IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_feature_flags_updated_at') THEN
			CREATE TRIGGER update_feature_flags_updated_at
			BEFORE UPDATE ON feature_flags
			FOR EACH ROW
			EXECUTE FUNCTION update_timestamp();
		END IF;
	END;
	$$;
`)
	return err
}

func getFlagBaseQuery() sq.SelectBuilder {
	strategies := `COALESCE(
		JSON_AGG(JSON_BUILD_OBJECT('type', s.type, 'payload', s.payload))
		FILTER (WHERE s.id IS NOT NULL), '[]') AS strategies`
	return sq.Select(
		"f.id",
		"f.name",
		"f.description",
		"f.enabled",
		"f.created_at",
		"f.updated_at",
		strategies,
	).
		PlaceholderFormat(sq.Dollar).
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
		Values(id, flag.Name, flag.Description, flag.Enabled).
		PlaceholderFormat(sq.Dollar)
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
	_, err = tx.Exec(ctx, sql, args...)
	if err != nil {
		return "", err
	}
	if len(flag.Strategies) == 0 {
		err = tx.Commit(ctx)
		return id, err
	}
	stratQuery := sq.Insert("strategies").
		Columns("flag_id", "type", "payload").
		PlaceholderFormat(sq.Dollar)
	for _, strategy := range flag.Strategies {
		stratQuery = stratQuery.Values(id, strategy.Type, strategy.Payload)
	}
	sql, args, err = stratQuery.ToSql()
	if err != nil {
		return id, err
	}
	_, err = tx.Exec(ctx, sql, args...)
	if err != nil {
		return "", err
	}
	err = tx.Commit(ctx)
	return id, err
}

func (d *DataSource) UpdateFlag(ctx context.Context, flag *data.FeatureFlag) error {
	query := sq.Update("feature_flags").
		Set("name", flag.Name).
		Set("description", flag.Description).
		Set("enabled", flag.Enabled).
		Where(sq.Eq{"id": flag.ID}).
		PlaceholderFormat(sq.Dollar)
	sql, args, err := query.ToSql()
	if err != nil {
		return err
	}
	_, err = d.db.Exec(ctx, sql, args...)
	return err
}

func (d *DataSource) DeleteFlag(ctx context.Context, id string) error {
	query := sq.Delete("feature_flags").Where(sq.Eq{"id": id}).PlaceholderFormat(sq.Dollar)
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

package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/nojyerac/go-lib/audit"
	"github.com/nojyerac/go-lib/auth"
	"github.com/nojyerac/go-lib/db"
	"github.com/nojyerac/semaphore/data"
)

var _ data.Source = (*DataSource)(nil)

type DataSource struct {
	db db.Database
	al audit.AuditLogger
}

type Option func(*DataSource)

const (
	auditActionFlagCreated = "flag.created"
	auditActionFlagUpdated = "flag.updated"
	auditActionFlagDeleted = "flag.deleted"
	defaultAuditActorID    = "11111111-1111-4111-8111-111111111111"
)

func WithAuditLogger(logger audit.AuditLogger) Option {
	return func(source *DataSource) {
		if source == nil || logger == nil {
			return
		}
		source.al = logger
	}
}

func NewDataSource(ctx context.Context, database db.Database, opts ...Option) (*DataSource, error) {
	auditLogger, err := audit.NewAuditLogger(audit.NewConfiguration())
	if err != nil {
		return nil, err
	}

	dataSource := &DataSource{
		db: database,
		al: auditLogger,
	}
	for _, opt := range opts {
		opt(dataSource)
	}
	err = dataSource.Migrate(ctx)
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
	querySQL, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}
	flags := make([]*data.FeatureFlag, 0)
	err = d.db.Select(ctx, &flags, querySQL, args...)
	return flags, err
}

func (d *DataSource) GetFlagByID(ctx context.Context, id string) (*data.FeatureFlag, error) {
	return getFlagByID(ctx, d.db, id)
}

func getFlagByID(ctx context.Context, executor db.DataInterface, id string) (*data.FeatureFlag, error) {
	query := getFlagBaseQuery().Where(sq.Eq{"f.id": id})
	querySQL, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}
	flag := &data.FeatureFlag{}
	err = executor.Get(ctx, flag, querySQL, args...)
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
	querySQL, args, err := query.ToSql()
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
	_, err = tx.Exec(ctx, querySQL, args...)
	if err != nil {
		return "", err
	}
	if len(flag.Strategies) > 0 {
		stratQuery := sq.Insert("strategies").
			Columns("flag_id", "type", "payload").
			PlaceholderFormat(sq.Dollar)
		for _, strategy := range flag.Strategies {
			stratQuery = stratQuery.Values(id, strategy.Type, strategy.Payload)
		}
		querySQL, args, err = stratQuery.ToSql()
		if err != nil {
			return id, err
		}
		_, err = tx.Exec(ctx, querySQL, args...)
		if err != nil {
			return "", err
		}
	}
	flagForAudit := *flag
	flagForAudit.ID = id
	err = d.logCreate(ctx, &flagForAudit)
	if err != nil {
		return "", err
	}
	err = tx.Commit(ctx)
	return id, err
}

func (d *DataSource) UpdateFlag(ctx context.Context, flag *data.FeatureFlag) error {
	tx, err := d.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil || recover() != nil {
			_ = tx.Rollback(ctx)
		}
	}()
	before, err := getFlagByID(ctx, tx, flag.ID)
	if err != nil {
		return err
	}
	query := sq.Update("feature_flags").
		Set("name", flag.Name).
		Set("description", flag.Description).
		Set("enabled", flag.Enabled).
		Where(sq.Eq{"id": flag.ID}).
		PlaceholderFormat(sq.Dollar)
	querySQL, args, err := query.ToSql()
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, querySQL, args...)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `DELETE FROM strategies WHERE flag_id = $1`, flag.ID)
	if err != nil {
		return err
	}
	if len(flag.Strategies) > 0 {
		stratQuery := sq.Insert("strategies").
			Columns("flag_id", "type", "payload").
			PlaceholderFormat(sq.Dollar)
		for _, strategy := range flag.Strategies {
			stratQuery = stratQuery.Values(flag.ID, strategy.Type, strategy.Payload)
		}
		querySQL, args, err = stratQuery.ToSql()
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, querySQL, args...)
		if err != nil {
			return err
		}
	}
	err = d.logUpdate(ctx, before, flag)
	if err != nil {
		return err
	}
	err = tx.Commit(ctx)
	return err
}

func (d *DataSource) DeleteFlag(ctx context.Context, id string) error {
	tx, err := d.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil || recover() != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	before, err := getFlagByID(ctx, tx, id)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	query := sq.Delete("feature_flags").Where(sq.Eq{"id": id}).PlaceholderFormat(sq.Dollar)
	querySQL, args, err := query.ToSql()
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, querySQL, args...)
	if err != nil {
		return err
	}

	err = d.logDelete(ctx, before)
	if err != nil {
		return err
	}

	err = tx.Commit(ctx)
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

func (d *DataSource) logCreate(ctx context.Context, flag *data.FeatureFlag) error {
	actorID, err := auditActor(ctx)
	if err != nil {
		return err
	}

	return d.al.LogChange(ctx, actorID, auditActionFlagCreated, nil, flagToAuditMap(flag))
}

func (d *DataSource) logUpdate(ctx context.Context, before, after *data.FeatureFlag) error {
	actorID, err := auditActor(ctx)
	if err != nil {
		return err
	}
	beforeMap := flagToAuditMap(before)
	afterMap := flagToAuditMap(after)

	return d.al.LogChange(ctx, actorID, auditActionFlagUpdated, beforeMap, afterMap)
}

func (d *DataSource) logDelete(ctx context.Context, before *data.FeatureFlag) error {
	actorID, err := auditActor(ctx)
	if err != nil {
		return err
	}

	return d.al.LogChange(ctx, actorID, auditActionFlagDeleted, flagToAuditMap(before), nil)
}

func auditActor(ctx context.Context) (actorID string, err error) {
	claims, ok := auth.FromContext(ctx)
	if !ok || claims == nil || claims.Subject == "" {
		return "", errors.New("no actor claims in context")
	}
	parsed, err := uuid.Parse(claims.Subject)
	if err != nil {
		return "", err
	}
	if parsed.Version() == 4 {
		return parsed.String(), nil
	}
	return "", errors.New("invalid actor UUID")
}

func flagToAuditMap(flag *data.FeatureFlag) map[string]any {
	if flag == nil {
		return nil
	}
	var err error
	var strategies []byte
	if (len(flag.Strategies) == 0) || flag.Strategies == nil {
		strategies = []byte("[]")
	} else {
		strategies, err = json.Marshal(flag.Strategies)
	}
	if err != nil {
		strategies = []byte("[]")
	}

	return map[string]any{
		"id":          flag.ID,
		"name":        flag.Name,
		"description": flag.Description,
		"enabled":     flag.Enabled,
		"strategies":  string(strategies),
	}
}

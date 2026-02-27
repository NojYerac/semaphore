# Feature‑Flag Data Model

## SQL Schema

```sql
CREATE TABLE feature_flags (
  id          UUID PRIMARY KEY,
  name        VARCHAR(255) NOT NULL UNIQUE,
  description TEXT,
  enabled     BOOLEAN NOT NULL DEFAULT FALSE,
  created_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
  updated_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now()
);

CREATE TABLE flag_strategies (
  id          UUID PRIMARY KEY,
  flag_id     UUID REFERENCES feature_flags(id) ON DELETE CASCADE,
  type        VARCHAR(50) NOT NULL,
  payload     JSONB NOT NULL,
  created_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
  updated_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now()
);

CREATE TABLE audit_events (
  id          UUID PRIMARY KEY,
  flag_id     UUID REFERENCES feature_flags(id) ON DELETE SET NULL,
  actor_id    UUID NOT NULL,
  action      VARCHAR(50) NOT NULL,
  details     JSONB,
  occurred_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now()
);
```

### Strategy Payloads

| Type       | Payload JSON Schema
|------------|----------------------
| `percentage` | `{ "percentage": 0.25 }`
| `user` | `{ "user_ids": ["uuid1", "uuid2"] }`
| `group` | `{ "group_ids": ["group1", "group2"] }`

## Go Structs

```go
// engine/flag.go
package engine

import "time"

type FeatureFlag struct {
    ID          string            `json:"id"`
    Name        string            `json:"name"`
    Description *string           `json:"description,omitempty"`
    Enabled     bool              `json:"enabled"`
    Strategies  []Strategy        `json:"strategies"`
    CreatedAt   time.Time         `json:"created_at"`
    UpdatedAt   time.Time         `json:"updated_at"`
}

type Strategy struct {
    Type    string          `json:"type"`
    Payload json.RawMessage `json:"payload"`
}
```

## Migration

Use Goose or golang-migrate to generate migrations.

## Next Steps

- Write Go structs for audit events.
- Create repository interfaces for CRUD.
- Add unit tests for mapping.

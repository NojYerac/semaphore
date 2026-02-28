# Semaphore

Lightweight feature-flag service written in Go.

Semaphore provides CRUD and evaluation APIs over HTTP and gRPC, supports percentage/user/group targeting strategies, and stores flag state in PostgreSQL.

## Current Status

Implemented:

- HTTP API for create/read/update/delete/evaluate flag operations
- gRPC API for create/read/update/delete/evaluate flag operations
- Evaluation engine with:
  - `percentage_rollout`
  - `user_targeting`
  - `group_targeting`
- PostgreSQL-backed persistence for flags and strategies
- Service telemetry hooks (logging, tracing, metrics, health)

Planned next:

- Authentication and role-based authorization
- End-to-end audit logging exposure
- GitHub Actions CI/CD
- First-class Go SDK package
- Operational hardening (limits, migrations, rate limiting)

See implementation backlog in [plans/README.md](./plans/README.md).

## Repository Layout

- `api/`: protobuf + OpenAPI definitions
- `config/`: service configuration bootstrap
- `data/`: domain models, source interfaces, evaluation engine integration
- `data/db/`: PostgreSQL datasource and schema bootstrap
- `data/engine/`: flag evaluation logic
- `transport/http/`: HTTP handlers
- `transport/rpc/`: gRPC service handlers
- `semaphore/main.go`: service entrypoint
- `testclient/`: sample client calls for local testing
- `plans/`: active roadmap and task breakdowns

## API Summary

### HTTP

Routes are mounted under `/api` by the shared HTTP server.

- `GET /api/flags`
- `POST /api/flags`
- `GET /api/flags/{id}`
- `PUT /api/flags/{id}`
- `DELETE /api/flags/{id}`
- `POST /api/flags/{id}/evaluate`

OpenAPI spec: `api/openapi.yml`

### gRPC

Service: `flag.FlagService`

- `GetFlag`
- `ListFlags` (server streaming)
- `CreateFlag`
- `UpdateFlag`
- `DeleteFlag`
- `Evaluate`

Proto contract: `api/flag.proto`

## Data Model (High Level)

Core entities:

- `feature_flags`: id, name, description, enabled, timestamps
- `strategies`: linked strategy rows with typed JSON payloads
- `audit_logs`: table exists and is reserved for mutation audit history integration

Strategy payload examples:

- `percentage_rollout`: `{ "percentage": 50 }`
- `user_targeting`: `{ "user_ids": ["<uuid>"] }`
- `group_targeting`: `{ "group_ids": ["<uuid>"] }`

## Local Development

Prerequisites:

- Go (matching `go.mod`)
- PostgreSQL reachable by configured connection string
- `protoc`, `mockery`, and `golangci-lint` for generation/lint flows

### Build

```bash
./scripts/build.sh
```

### Generate + lint

```bash
./scripts/generate.sh
```

### Run tests

```bash
go test ./...
```

### Start service

```bash
go run ./semaphore/main.go
```

### Exercise service

```bash
go run ./testclient/main.go
```

## Configuration

Configuration is loaded and validated at startup through shared `go-lib` configuration packages.

Major configuration areas:

- Logging
- Database
- HTTP server
- Transport wiring
- Tracing
- Health checks

See [config/config.go](./config/config.go) for registration order and loaded config groups.

## Testing

The repository includes unit tests for:

- Data model conversions and validation
- Engine evaluation logic
- Database source behavior (using sqlmock)
- HTTP handlers
- gRPC handlers

Primary test command:

```bash
go test ./...
```

## Roadmap

Current roadmap items are tracked in [plans/](./plans):

1. [Authentication and Authorization](./plans/01-AUTHN-AUTHZ.md)
2. [End-to-End Audit Logging](./plans/02-AUDIT-LOGGING.md)
3. [CI/CD with GitHub Actions](./plans/03-CI-CD-GITHUB-ACTIONS.md)
4. [First-Class Go Client SDK](./plans/04-GO-CLIENT-SDK.md)
5. [Operational Hardening](./plans/05-OPERATIONS-HARDENING.md)

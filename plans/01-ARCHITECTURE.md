# Feature‑Flag Engine Architecture

## Overview

The feature‑flag engine is a lightweight, embeddable component that enables runtime configuration of application behaviour. It supports

- **flag evaluation** – decide whether a flag is enabled for a specific user, group or environment.
- **rollout strategies** – percentage roll‑outs, gradual roll‑outs, canary, and targeted rollout.
- **audit and visibility** – every change is logged and the current state is queryable.
- **security** – only authorised users can mutate flags; clients can only read.

The architecture is split into three layers:

1. **Engine** – pure Go library that performs evaluation logic.
2. **API Server** – gRPC/HTTP service exposing CRUD for flags and an evaluation endpoint. It enforces authentication and logs audit events.
3. **Client SDK** – thin wrapper around the API that can be embedded in other services or CLI tools.

The following diagram shows the interaction flow.

![Architecture Diagram](./architecture.svg)

## Data Model

```mermaid
type FeatureFlag = {
  id: string
  name: string
  description?: string
  enabled: boolean
  strategies: Strategy[]
  created_at: string
  updated_at: string
}

union Strategy = {
  type: "percentage"
  percentage: number
} | {
  type: "user"
  user_ids: string[]
} | {
  type: "group"
  group_ids: string[]
}
```

Flags are persisted in a simple SQL table. A `flags` table holds the flag record; a `strategies` table holds the strategies linked via a foreign key.

## Security Model

- **API**: Uses mutual TLS for server‑side authentication and JSON Web Tokens (JWT) for client authentication.
- **Authorization**: RBAC – roles such as `flag_admin`, `flag_reader` control CRUD access.
- **Audit**: Every mutation writes to an `audit_events` table.

## Deployment

- The engine is built as a Go library (mod `semaphore/engine`).
- The API server (`semaphore/api`) is a standalone binary.
- The client SDK is a Go package (`semaphore/sdk`).
- Configuration is loaded from environment variables and a YAML config.

## Next Steps

1. Create the database schema.
2. Implement the engine package.
3. Build the API server with middleware.
4. Develop the client SDK.
5. Write tests and integration pipelines.
6. Update documentation and README.

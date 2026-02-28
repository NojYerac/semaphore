# Task 01: Authentication and Authorization

## Problem

The service exposes full flag mutation and evaluation endpoints over HTTP and gRPC with no identity or role checks.

## Goal

Require authenticated callers and enforce role-based access control for all API operations.

## Scope

- Add authentication middleware/interceptors for HTTP and gRPC.
- Support bearer JWT validation (issuer, audience, expiry).
- Add RBAC policy with at least:
  - `flag_reader`: list/get/evaluate
  - `flag_admin`: create/update/delete
- Return consistent authorization errors (`401`, `403` / gRPC equivalents).
- Add configuration for auth provider and policy mappings.

## Out of Scope

- Multi-tenant policy engines.
- External identity provider provisioning.

## Deliverables

- HTTP middleware for auth + role checks.
- gRPC unary and stream interceptors for auth + role checks.
- Shared `auth` package with token parsing and claims abstraction.
- Integration tests covering allowed/denied cases per endpoint.
- Documentation updates for required headers and role matrix.

## Suggested Implementation Steps

1. Add auth configuration (issuer, audience, JWKS URL or static key, required roles per endpoint).
2. Implement token validator and claims extraction package.
3. Build HTTP middleware and apply at route registration.
4. Build gRPC interceptors and apply in server bootstrap.
5. Add endpoint → role mapping table used by both transports.
6. Add test fixtures for valid/expired/invalid role tokens.

## Acceptance Criteria

- All API requests without valid identity are rejected.
- Mutating endpoints deny `flag_reader` and allow `flag_admin`.
- Read/evaluate endpoints allow both `flag_reader` and `flag_admin`.
- Auth failures are observable in logs/metrics without leaking token contents.

## Risks / Notes

- Keep transport-level logic thin; centralize policy decisions in a shared package.
- Ensure trace/span context keeps caller identity metadata safely (no PII beyond subject/role IDs).

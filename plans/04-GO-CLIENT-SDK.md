# Task 04: First-Class Go Client SDK

## Problem

Consumers currently depend on generated protobuf stubs or ad-hoc HTTP calls; there is no stable SDK package with ergonomic APIs.

## Goal

Provide a supported Go SDK for service-to-service integration and CLI consumers.

## Scope

- Create package `sdk/` with typed client and options.
- Support both HTTP and gRPC transports (or define one as initial default).
- Add auth injection and retry/timeouts.
- Expose CRUD + evaluate APIs.

## Out of Scope

- SDKs for non-Go languages.
- Local/offline evaluation cache in first iteration.

## Deliverables

- `sdk/client.go` and request/response types.
- Configurable authentication (`BearerTokenProvider` style).
- Retries/backoff for transient errors.
- Integration tests using `httptest` and/or in-process gRPC server.
- Usage examples in README.

## Suggested Implementation Steps

1. Define public client interface and constructor options.
2. Implement transport abstraction and error normalization.
3. Add auth and timeout middleware/hooks.
4. Implement flag operations and evaluate operation.
5. Add tests and examples.
6. Version package API and add changelog discipline.

## Acceptance Criteria

- SDK can create, retrieve, update, delete, list, and evaluate flags.
- Auth headers/metadata are automatically attached.
- Errors include operation context and status codes.
- Example app compiles and runs against local service.

## Risks / Notes

- Keep SDK surface small and stable to avoid frequent breaking changes.
- Document backward compatibility policy before external adoption.

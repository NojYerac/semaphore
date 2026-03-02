# Task 02B: Externalized Audit Logging Pipeline

## Problem

Current planning centers on writing audit records in the same database as flags. That pattern is simple but does not meet stronger expectations for tamper resistance, centralized visibility, and compliance-grade retention.

## Goal

Shift audit logging to an externalized pipeline while preserving reliable, transaction-safe capture of all flag mutations.

## Scope

- Capture create/update/delete audit events in mutation workflows.
- Stage events in an internal outbox within the same mutation transaction.
- Dispatch staged events asynchronously to an external audit sink.
- Provide delivery status observability and operational controls.
- Keep a bounded local query surface for recent troubleshooting (optional cache/read model).

## Out of Scope

- Building a full SIEM product in this repository.
- Long-term storage engine implementation in Semaphore.
- Real-time streaming analytics/dashboarding.

## Deliverables

- Data-layer additions:
  - `audit_outbox` table and repository methods (`Append`, `Claim`, `Ack`, `Fail`, `Requeue`).
  - migration scripts for outbox schema.
- Mutation integration:
  - create/update/delete append one outbox audit event in the same transaction as flag mutation.
- Worker/dispatcher:
  - background publisher loop with retry/backoff + DLQ policy.
  - graceful shutdown and lease timeout recovery.
- Transport/API updates:
  - keep `GET /flags/{id}/audit` contract, backed by external provider where possible.
  - add status/health visibility for audit pipeline lag and failures.
- Tests:
  - mutation atomicity (no orphan events, no missing events on success).
  - dispatcher retries/idempotency behavior.
  - failure-path tests (sink unavailable, poison payload, lease expiry).

## Suggested Implementation Steps

1. Add `audit_outbox` schema + repository contract.
2. Update mutation transactions to append outbox event with actor, action, resource, diff.
3. Implement dispatcher worker with claim/ack/fail semantics.
4. Add external sink interface and one concrete adapter target (config-driven).
5. Add DLQ strategy (`max_attempts`, reason code, last error summary).
6. Implement read-path strategy:
   - primary: external audit query API/client.
   - fallback: local bounded read model (if enabled).
7. Add metrics, tracing, and admin diagnostics endpoints.
8. Add integration coverage for end-to-end capture + delivery guarantees.

## Acceptance Criteria

- Every successful flag create/update/delete produces exactly one outbox event.
- Failed mutations never produce committed outbox events.
- Dispatcher publishes events with at-least-once semantics.
- Duplicate publish attempts are safe due to idempotent `event_id` usage.
- Audit query path returns deterministic ordering (timestamp desc, tie-breaker by event ID).
- Pipeline exposes metrics for queued, published, retried, failed, and DLQ counts.

## Security and Compliance Requirements

- Event envelope includes: actor ID, action, flag ID/resource key, timestamp, request/trace ID, schema version.
- Sensitive fields in `details` are redacted/tokenized before dispatch.
- TLS required for sink transport.
- Audit events are immutable after publish; corrections occur via compensating events.
- Retention and legal hold are delegated to external sink policy.

## Migration Strategy

1. Introduce outbox writes while retaining existing in-DB audit writes behind dual-write flag.
2. Validate delivery parity and counts between internal and external stores.
3. Switch read path to external source (with fallback during cutover).
4. Disable legacy in-DB writes after stability window.
5. Keep temporary backfill job for missed historical windows if needed.

## Risks / Notes

- External sink outages can increase backlog; enforce queue age alerts and replay runbooks.
- Strict ordering across all resources is expensive; use per-resource ordering guarantees.
- Cost can grow with verbose payloads; prefer compact diffs and bounded metadata.

# Task 02: End-to-End Audit Logging

## Problem

Audit schema/types exist, but writes and reads are not completed in the active request paths.

## Goal

Produce reliable audit trails for all flag mutations and expose query access for operations and compliance needs.

## Scope

- Write audit rows on create, update, and delete operations.
- Record actor identity from auth context.
- Persist structured details (before/after values or patch-like diff).
- Implement retrieval methods in data source.
- Expose audit API endpoint(s) in HTTP and gRPC.

## Out of Scope

- External SIEM ingestion.
- Long-term archival pipelines.

## Deliverables

- Data-layer methods for `CreateAuditLog` and `GetAuditLogs`.
- Transaction-safe audit writes in mutation workflows.
- HTTP endpoint: `GET /flags/{id}/audit` (or equivalent) with pagination.
- gRPC RPC for querying audit logs.
- Tests validating audit writes and reads, including error conditions.

## Suggested Implementation Steps

1. Define audit repository interface and models for request/response.
2. Implement SQL write/read queries and wire to source layer.
3. Update create/update/delete flows to include audit write in same transaction.
4. Add transport handlers and protobuf messages for audit retrieval.
5. Add pagination (`limit`, `cursor` or `offset`) and ordering by timestamp desc.
6. Add unit/integration coverage for mutation + audit consistency.

## Acceptance Criteria

- Every successful create/update/delete request results in one audit record.
- Audit record includes actor ID, action, timestamp, flag ID, and details payload.
- Audit retrieval returns deterministic ordering and supports pagination.
- Failed mutations do not create orphaned audit rows.

## Risks / Notes

- Keep details payload bounded in size to avoid oversized row growth.
- For updates, prefer compact diff payloads rather than full object snapshots.

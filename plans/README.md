# Semaphore Plans

This directory now tracks **implementation tasks that are not yet represented in code**.

## Completed plan areas (removed)

The previous design docs were removed because these areas already exist in the repository in working form:

- Core architecture and service wiring
- Data model and persistence for flags/strategies
- HTTP and gRPC CRUD/evaluate APIs
- Evaluation engine (percentage, user, group targeting)

## Active backlog

1. [02-AUDIT-LOGGING.md](./02-AUDIT-LOGGING.md)  
  Implement end-to-end audit event capture, querying, and exposure.
2. [03-CI-CD-GITHUB-ACTIONS.md](./03-CI-CD-GITHUB-ACTIONS.md)  
  Add repeatable CI/CD pipelines using GitHub Actions.
3. [04-GO-CLIENT-SDK.md](./04-GO-CLIENT-SDK.md)  
  Build a supported SDK package instead of relying on generated stubs only.
4. [05-OPERATIONS-HARDENING.md](./05-OPERATIONS-HARDENING.md)  
  Add production hardening: limits, lifecycle, migrations, and reliability checks.

## Completed

- [01-AUTHN-AUTHZ.md](./01-AUTHN-AUTHZ.md)

## Prioritization

Recommended order for highest impact:

1. Audit logging
2. CI pipeline
3. SDK
4. Operational hardening

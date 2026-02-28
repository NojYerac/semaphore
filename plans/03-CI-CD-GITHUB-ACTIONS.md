# Task 03: CI/CD with GitHub Actions

## Problem

The repository has build/generate scripts but no automated GitHub Actions workflows.

## Goal

Create reproducible CI and release automation for validation, packaging, and deployment readiness.

## Scope

- Add CI workflow for pull requests and main branch pushes.
- Run lint, unit tests, and build artifacts.
- Cache Go modules for faster pipeline runs.
- Add release workflow for tagged versions.

## Out of Scope

- Production deployment to a specific cloud provider.
- Terraform or cluster provisioning.

## Deliverables

- `.github/workflows/ci.yml`
- `.github/workflows/release.yml`
- Optional coverage upload step.
- README badge(s) and contributor instructions.

## Suggested Implementation Steps

1. Create CI workflow matrix for supported Go version(s).
2. Add steps: checkout, setup-go, cache, `go mod download`, lint, tests, build.
3. Add artifact upload for built binary.
4. Create release workflow triggered by tags (`v*`).
5. Attach binary artifacts and checksums to release.
6. Add branch protection guidance in docs.

## Acceptance Criteria

- Every PR runs lint + tests + build automatically.
- Main branch stays green under required checks.
- Tagging a release creates downloadable artifacts.
- Workflow runtimes are stable and observable.

## Risks / Notes

- Keep workflows minimal at first; expand only after baseline stability.
- Prefer fail-fast strategy to reduce wasted CI minutes.

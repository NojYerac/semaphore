# Roadmap - Remediation (Mars Audit)

## High Priority (Security & Ops)
- [ ] **Hardcoded Credentials:** Remove plaintext passwords from `docker-compose.yml`. Use `.env` file (and `.gitignore` it).
- [ ] **Bloated Build Context:** Create `.dockerignore` to exclude `.git`, `testclient`, `mocks`.
- [ ] **Service Health:** Add a `healthcheck` to the `semaphore` service using the `/health` endpoint.

## Documentation
- [ ] **Fix Architecture Diagram:** Reverse arrows to show proper Hexagonal dependency direction (Infrastructure -> Core).
- [ ] **Sharpen the Hook:** Rewrite the headline to be punchier ("Zero-latency, strict-consistency feature flag engine").

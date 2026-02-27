# Development Environment Setup for Lightweight Feature-Flag / Rollout Engine

## 1. Programming Language
- Use **Go** for backend development.

## 2. Frameworks and Libraries
- **REST + gRPC API**: Use Go's built-in `net/http` for REST and `google.golang.org/grpc` for gRPC.
- **Database**: Choose between **Postgres** or **Redis** for data storage.
- **Kubernetes**: Prepare for deployment in a Kubernetes environment.

## 3. Development Tools
- **Go Modules**: Initialize Go modules for dependency management.
- **Docker**: Set up Docker for containerization.
- **Kubernetes CLI**: Install `kubectl` for managing Kubernetes clusters.

## 4. Logging and Metrics
- Implement structured logging using libraries like `logrus` or `zap`.
- Set up metrics collection with Prometheus or similar tools.

## 5. Testing Framework
- Use Go's built-in testing framework for unit tests.
- Consider using `testify` for assertions.

## 6. Documentation
- Document the setup process in this markdown file for future reference.
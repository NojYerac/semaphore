# Logging and Metrics Configuration for Lightweight Feature-Flag / Rollout Engine

## 1. Structured Logging
- Use `logrus` for structured logging.
- Install the package:
  ```bash
  go get github.com/sirupsen/logrus
  ```
- Example usage:
  ```go
  package main

  import (
      "github.com/sirupsen/logrus"
  )

  func main() {
      log := logrus.New()
      log.Info("Starting the application...")
  }
  ```

## 2. Metrics Collection
- Use Prometheus for metrics collection.
- Install the package:
  ```bash
  go get github.com/prometheus/client_golang/prometheus
  ```
- Example usage:
  ```go
  package main

  import (
      "github.com/prometheus/client_golang/prometheus"
      "github.com/prometheus/client_golang/prometheus/promhttp"
      "net/http"
  )

  func main() {
      http.Handle("/metrics", promhttp.Handler())
      http.ListenAndServe(":8080", nil)
  }
  ```

## 3. Documentation
- Document the logging and metrics setup in this markdown file for future reference.
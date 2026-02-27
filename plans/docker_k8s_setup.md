# Docker and Kubernetes CLI Setup for Lightweight Feature-Flag / Rollout Engine

## 1. Docker
- Install Docker Desktop from https://www.docker.com/products/docker-desktop or use Homebrew:
  ```bash
  brew install --cask docker
  ```
- After installation, start Docker Desktop and ensure it is running.
- Verify installation:
  ```bash
  docker --version
  ```

## 2. kubectl (Kubernetes CLI)
- Install via Homebrew:
  ```bash
  brew install kubectl
  ```
- Verify installation:
  ```bash
  kubectl version --client
  ```

## 3. Helm (Optional but recommended for managing Kubernetes charts)
- Install via Homebrew:
  ```bash
  brew install helm
  ```
- Verify installation:
  ```bash
  helm version
  ```

## 4. Docker Compose
- Docker Desktop includes Docker Compose v2. Verify:
  ```bash
  docker compose version
  ```

## 5. Additional Configuration
- For local Kubernetes cluster, consider using Docker Desktop’s built-in Kubernetes or minikube:
  ```bash
  brew install minikube
  minikube start
  ```

## 6. Documentation
- Add these instructions to the project documentation so future developers can quickly set up the environment.

---

### Quick Checklist
- [ ] Docker installed and running
- [ ] kubectl installed and configured
- [ ] (Optional) Helm installed
- [ ] (Optional) Minikube or local cluster set up

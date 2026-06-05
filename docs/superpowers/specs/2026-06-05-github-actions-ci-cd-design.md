# Payroll GitHub Actions CI/CD Pipeline

**Date**: 2026-06-05
**Status**: Approved

## Overview

Set up a 4-job GitHub Actions CI/CD pipeline for the payroll monorepo (Go backend + React frontend), mirroring the proven nepocorp pipeline pattern.

## Triggers & Concurrency

- **Push to `main`**: full pipeline (build → test → docker push → deploy)
- **PR to `main`**: build + test only (no push/deploy)
- **Concurrency**: cancel in-progress runs for the same branch (`deploy-${{ github.ref }}`)

## Job 1: Build & Lint

Runs on every push and PR.

| Step | Command |
|------|---------|
| Checkout | `actions/checkout@v4` |
| Go setup | `actions/setup-go@v5`, go-version: '1.26', cache via go.sum |
| Go vet | `go vet ./...` |
| Go fmt check | `test -z $(gofmt -l .)` — fail if any unformatted files |
| Node setup | `actions/setup-node@v4`, node-version: '20' |
| Yarn install | `cd frontend && yarn install --frozen-lockfile` |
| Frontend typecheck | `cd frontend && npx tsc -b` |
| Frontend build | `cd frontend && npx vite build` |

## Job 2: Tests

Runs on every push and PR, in parallel with Job 1.

| Step | Command |
|------|---------|
| Backend unit tests | `cd backend && go test -race ./...` |
| Frontend unit tests | `cd frontend && yarn install --frozen-lockfile && yarn test:run` |

No integration tests in CI (those require MySQL + Redis and are run locally via `make api-test`).

## Job 3: Docker Push (main branch only)

Runs only on push to main, after build + test pass. Uses matrix strategy to build backend and frontend images in parallel.

| Step | Details |
|------|---------|
| Matrix | backend: `franknguyenvd/payroll-backend`, frontend: `franknguyenvd/payroll-frontend` |
| Buildx | `docker/setup-buildx-action@v3` |
| Login | Docker Hub via `DOCKERHUB_USERNAME` / `DOCKERHUB_PASSWORD` secrets |
| Build & push | `docker/build-push-action@v6`, platform `linux/amd64`, tags `latest` + short SHA |
| Cache | `cache-from: type=registry,ref=${{ matrix.image }}:latest` |

### Frontend build args

```yaml
build-args: |
  VITE_API_BASE_URL=https://tingting.vip
  VITE_API_VERSION=/api/v1
  VITE_API_TIMEOUT=30000
  VITE_GOOGLE_CLIENT_ID=${{ secrets.VITE_GOOGLE_CLIENT_ID }}
```

### Matrix definition

```yaml
strategy:
  matrix:
    include:
      - name: backend
        image: franknguyenvd/payroll-backend
        context: backend
      - name: frontend
        image: franknguyenvd/payroll-frontend
        context: frontend
```

## Job 4: Deploy to Production (main branch only)

Runs only on push to main, after Docker push completes.

| Step | Details |
|------|---------|
| SSH deploy | `appleboy/ssh-action@v1` to `pay.1stop.app` |
| Pull images | `cd /opt/payroll && docker compose pull frontend backend` |
| Restart backend | `docker compose up -d --force-recreate --no-deps backend` |
| Health check | Poll `http://localhost:8080/api/v1/health` for up to 20s |
| Restart frontend | `docker compose up -d --force-recreate --no-deps frontend` |
| Prune | `docker image prune -f` |

## Required GitHub Secrets

| Secret | Purpose | Already in nepocorp? |
|--------|---------|---------------------|
| `DOCKERHUB_USERNAME` | Docker Hub login | Yes — reuse value |
| `DOCKERHUB_PASSWORD` | Docker Hub token | Yes — reuse value |
| `PROD_SSH_KEY` | SSH key for `pay.1stop.app` | New — generate for payroll server |
| `VITE_GOOGLE_CLIENT_ID` | Frontend build arg | New — from payroll `.env` |

## File to Create

`.github/workflows/ci-cd.yml` — single workflow file, ~180 lines.

## Out of Scope

- Integration tests in CI (requires MySQL + Redis service containers)
- Database migrations in CI (handled separately)
- Staging environment
- Slack/email notifications

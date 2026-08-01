# Load environment variables from .env file.
# Export ONLY real KEY=value assignments — the naive `sed 's/=.*//'` form also
# turned comment text (e.g. "# GHCR_TOKEN is a secret...") into empty exported
# variable names, clobbering GHCR_TOKEN's ?= fallback below.
include .env
export $(shell grep -E '^[A-Za-z_][A-Za-z0-9_]*=' .env | sed 's/=.*//')

# Variables
GHCR_OWNER ?= 4rankng
IMAGE_NAME = payroll-frontend
TAG ?= latest
FULL_IMAGE_NAME = ghcr.io/$(GHCR_OWNER)/$(IMAGE_NAME):$(TAG)

# GHCR auth token. Prefer an explicit shell env var; otherwise read it from
# ~/.zshrc (per .env comment, the token lives there — not in .env) so deploys
# work from non-interactive shells (editors, CI, this agent) that never source
# ~/.zshrc. awk (not sed) because Make's $(shell) mis-parses sed + tr pipes.
GHCR_TOKEN ?= $(shell awk -F= '/export GHCR_TOKEN=/{print $$2}' $(HOME)/.zshrc | head -1)

# Buildx builder (shared with backend) — persistent docker-container builder.
# The builder's own volume caches layers; BUILD_CACHE_DIR adds an on-host cache
# export/import so builds skip the slow `pnpm install` step WITHOUT a ~100s
# registry manifest round-trip (type=registry,ref=...) on every build.
BUILDER ?= bb
BUILD_CACHE_DIR ?= $(HOME)/.cache/payroll-frontend-buildx

.PHONY: push
push:
	@docker buildx inspect $(BUILDER) >/dev/null 2>&1 || docker buildx create --name $(BUILDER) --driver docker-container --use
	@docker buildx use $(BUILDER)
	@docker buildx inspect --bootstrap >/dev/null
	@echo "Logging in to GHCR as $(GHCR_OWNER)"
	@echo "$(GHCR_TOKEN)" | docker login ghcr.io --username $(GHCR_OWNER) --password-stdin
	@echo "Building & pushing amd64 image with production environment variables: $(FULL_IMAGE_NAME)"
	docker buildx build \
		--platform linux/amd64 \
		--cache-from type=local,src=$(BUILD_CACHE_DIR) \
		--cache-to type=local,dest=$(BUILD_CACHE_DIR),mode=max \
		--build-arg BUILDKIT_INLINE_CACHE=1 \
		--build-arg VITE_API_BASE_URL=https://tingting.vip \
		--build-arg VITE_API_VERSION=/api/v1 \
		--build-arg VITE_API_TIMEOUT=30000 \
		--build-arg VITE_GOOGLE_CLIENT_ID=$(VITE_GOOGLE_CLIENT_ID) \
		--tag $(FULL_IMAGE_NAME) \
		--push \
		.
	@echo "Successfully pushed $(FULL_IMAGE_NAME)"

# Deploy to production droplet
deploy:
	@echo "🚀 Deploying frontend and backend on production droplet..."
	@echo "📡 Pulling latest frontend/backend images..."
	ssh root@146.190.90.27 "cd /opt/payroll && docker compose pull frontend backend"
	@echo "🔄 Restarting frontend and backend services..."
	ssh root@146.190.90.27 "cd /opt/payroll && docker compose up -d --force-recreate --no-deps frontend backend"
	@echo "🧹 Cleaning up unused images..."
	ssh root@146.190.90.27 "docker image prune -f"
	@echo "✅ Frontend/backend deployment complete!"
	@echo "🌐 Frontend: https://tingting.vip"
	@echo "🔧 Backend API: https://tingting.vip/api"

# Run unit tests
.PHONY: test
test:
	@echo "Running unit tests..."
	yarn test

# Run unit tests once (CI mode)
.PHONY: test-ci
test-ci:
	@echo "Running unit tests in CI mode..."
	yarn test:run

# Run unit tests with UI
.PHONY: test-ui
test-ui:
	@echo "Running unit tests with UI..."
	yarn test:ui









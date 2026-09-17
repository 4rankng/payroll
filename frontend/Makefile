# Load environment variables from .env file (optional — local dev only;
# push/deploy must run without it: GHCR_OWNER has a ?= fallback and the GHCR
# token is resolved at runtime via ../scripts/ghcr-token.sh).
# Export ONLY real KEY=value assignments — the naive `sed 's/=.*//'` form also
# turned comment text (e.g. "# GHCR_TOKEN is a secret...") into empty exported
# variable names, clobbering GHCR_TOKEN's ?= fallback below.
-include .env
export $(shell test -f .env && grep -E '^[A-Za-z_][A-Za-z0-9_]*=' .env | sed 's/=.*//' || true)

# Variables
GHCR_OWNER ?= 4rankng
IMAGE_NAME = payroll-frontend
TAG ?= latest
FULL_IMAGE_NAME = ghcr.io/$(GHCR_OWNER)/$(IMAGE_NAME):$(TAG)

# GHCR auth token is resolved inside the push recipe at runtime via
# ../scripts/ghcr-token.sh (env GHCR_TOKEN → Docker keychain → legacy ~/.zshrc).
# It is deliberately NOT a make variable: a $(shell) failure would silently
# expand to an empty token and surface as a cryptic "denied: denied" at login,
# and make -n would print the expanded token.
# NOTE for future editors: ../scripts resolves from frontend/ (and backend/) to
# the repo-root scripts dir; this only holds while these Makefiles sit one
# level below the repo root.

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
	@GHCR_TOKEN=$$(../scripts/ghcr-token.sh payroll-frontend) || { \
		echo "❌ GHCR token resolution failed — see message above."; exit 1; }; \
	echo "Logging in to GHCR as $(GHCR_OWNER)"; \
	printf '%s\n' "$$GHCR_TOKEN" | docker login ghcr.io --username $(GHCR_OWNER) --password-stdin
	@echo "Building & pushing amd64 image with production environment variables: $(FULL_IMAGE_NAME)"
	docker buildx build \
		--platform linux/amd64 \
		--cache-from type=local,src=$(BUILD_CACHE_DIR) \
		--cache-to type=local,dest=$(BUILD_CACHE_DIR),mode=max \
		--build-arg BUILDKIT_INLINE_CACHE=1 \
		--build-arg VITE_API_BASE_URL=https://tingting.vip \
		--build-arg VITE_API_VERSION=/api/v1 \
		--build-arg VITE_API_TIMEOUT=30000 \
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









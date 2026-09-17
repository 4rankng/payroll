.PHONY: deploy dev db api-test backup restore sandbox mirror-bases

# ─────────────────────────────────────────────────────────────────────────────
# GHCR base-image mirroring
#
# We vendor upstream base images into our own GHCR namespace so builds never
# depend on Docker Hub at build time (Docker Hub TLS-handshake timeouts are the
# #1 cause of deploy failures from Vietnam). `crane copy` is registry-to-
# registry — it does NOT route through the local Docker daemon, so it works
# even when `docker pull` from Docker Hub is failing locally. It preserves the
# exact upstream digest, so trust == Docker Official Image (no third-party
# mirror risk, appropriate for a money-moving system).
#
# After bumping a base image, re-run `make mirror-bases` and copy the printed
# digest into the matching FROM line in backend/Dockerfile or frontend/Dockerfile.
# ─────────────────────────────────────────────────────────────────────────────
GHCR_OWNER ?= 4rankng
BASE_IMAGES ?= node:22-alpine golang:1.26-alpine

# Mirror upstream Docker Hub base images into GHCR (registry-to-registry) and
# print the digest each image should be pinned to in the Dockerfiles.
mirror-bases:
	@command -v crane >/dev/null 2>&1 || { echo "❌ crane not installed. Run: brew install crane"; exit 1; }
	@for img in $(BASE_IMAGES); do \
		src="docker.io/library/$$img"; \
		dst="ghcr.io/$(GHCR_OWNER)/$$img"; \
		echo "🖼️  Mirroring $$src -> $$dst"; \
		crane copy "$$src" "$$dst" || { echo "❌ Failed to copy $$src"; exit 1; }; \
		echo "   pin digest: $$(crane digest $$dst)"; \
	done
	@echo "✅ Done. Pin the digests above into the Dockerfiles' FROM lines."

# Build & push all images, then deploy to production
deploy:
	@echo "=== Building & pushing frontend ==="
	cd frontend && make push
	@echo "=== Building & pushing backend ==="
	cd backend && make push
	@echo "=== Deploying to production ==="
	cd backend && make deploy

# Start both frontend and backend dev servers
dev:
	@echo "🚀 Starting development environment..."
	$(MAKE) -C backend dev

db:
	$(MAKE) -C backend db

api-test:
	$(MAKE) -C backend api-test

# Start the local mock OnePay/9Pay sandbox (port 9001) for advance-payout testing
sandbox:
	$(MAKE) -C backend sandbox

# Backup production database to OneDrive
backup:
	$(MAKE) -C backend backup

# Load latest database backup from OneDrive
restore:
	$(MAKE) -C backend restore

# Open Adminer over an SSH tunnel -> http://localhost:18081 (no internet exposure).
# Starts the Adminer container on prod, forwards localhost:18081 -> prod loopback:8081, opens the page.
# Ctrl-C closes the tunnel. Safer than adminer-on: never opens UFW 8081 or the public /adminer path.
adminer:
	@echo "=== Opening Adminer over SSH tunnel ==="
	cd backend && make adminer

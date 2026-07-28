.PHONY: deploy dev backup restore sandbox

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

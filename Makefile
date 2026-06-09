.PHONY: deploy dev backup restore

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

# Backup production database to OneDrive
backup:
	$(MAKE) -C backend backup

# Load latest database backup from OneDrive
restore:
	$(MAKE) -C backend restore

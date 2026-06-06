.PHONY: deploy dev

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

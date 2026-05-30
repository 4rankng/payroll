# Load environment variables from .env file
include .env
export $(shell sed 's/=.*//' .env)

# Variables
IMAGE_NAME = payroll-frontend
TAG ?= latest
FULL_IMAGE_NAME = $(DOCKERHUB_USERNAME)/$(IMAGE_NAME):$(TAG)

.PHONY: push
push:
	@echo "Logging in to DockerHub as $(DOCKERHUB_USERNAME)"
	@echo "$(DOCKERHUB_PASSWORD)" | docker login --username $(DOCKERHUB_USERNAME) --password-stdin
	@echo "Building Docker image for amd64 with production environment variables: $(FULL_IMAGE_NAME)"
	docker build --no-cache --platform linux/amd64 \
		--build-arg VITE_API_BASE_URL=https://tingting.vip \
		--build-arg VITE_API_VERSION=/api/v1 \
		--build-arg VITE_API_TIMEOUT=30000 \
		--build-arg VITE_GOOGLE_CLIENT_ID=$(VITE_GOOGLE_CLIENT_ID) \
		-t $(FULL_IMAGE_NAME) .
	@echo "Pushing Docker image: $(FULL_IMAGE_NAME)"
	docker push $(FULL_IMAGE_NAME)
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









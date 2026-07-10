.PHONY: backend db dev push deploy backup restore adminer-on adminer-off test api-test sync clean frontend sandbox sandbox-logs sandbox-rebuild onepay-topup push-demo deploy-demo demo-db

# Load environment variables from .env file
include .env
export

COMPOSE_FILE := docker-compose.dev.yml
PROD_SERVER := tingting.vip
DEMO_SERVER := demo.tingting.vip
DEMO_TAG    := demo

define check-docker
	@echo "🐳 Checking Docker status..."
	@if ! docker info >/dev/null 2>&1; then \
		echo "⚠️  Docker is not running. Starting Docker Desktop..."; \
		open -a Docker; \
		echo "⏳ Waiting for Docker to start (up to 60 seconds)..."; \
		timeout 60 sh -c 'while ! docker info >/dev/null 2>&1; do sleep 2; done' || (echo "❌ Docker failed to start within 60 seconds. Please start Docker manually." && exit 1); \
		echo "✅ Docker is now running!"; \
	fi
endef

frontend:
	@cd ../frontend && yarn dev

backend:
	@echo "🔄 Starting backend with hot-reload..."
	@pwd && \
	if ! command -v air >/dev/null 2>&1; then \
		echo "📦 Installing Air (hot-reload)..."; \
		go install github.com/air-verse/air@latest; \
	fi; \
	air -c .air.toml

db:
	$(check-docker)
	@echo "🧹 Removing stale containers..."
	-docker rm -f payroll-9pay-mock 2>/dev/null
	-docker rm -f payroll-onepay-mock 2>/dev/null
	@echo "📦 Rebuilding sandbox services (no cache)..."
	docker compose -f $(COMPOSE_FILE) build --no-cache sandbox
	@echo "🐳 Starting all services..."
	docker compose -f $(COMPOSE_FILE) up -d --build mysql redis adminer sandbox
	@echo "⏳ Waiting for services to be ready..."
	@timeout 60 sh -c 'until docker compose -f $(COMPOSE_FILE) exec mysql mysqladmin ping -h localhost --silent; do sleep 2; done' || echo "⚠️  MySQL took longer than expected to start"
	@timeout 30 sh -c 'until docker compose -f $(COMPOSE_FILE) exec redis redis-cli ping | grep PONG; do sleep 2; done' || echo "⚠️  Redis took longer than expected to start"
	@echo "✅ All services are ready!"
	@echo ""
	@echo "📡 Services:"
	@echo "   MySQL:       localhost:3306"
	@echo "   Redis:       localhost:6379"
	@echo "   Adminer:     http://localhost:8081"
	@echo ""
	@echo "🧪 Sandbox (APP_ENV=development):"
	@echo "   Mock (9Pay + OnePay): localhost:9001"
	@echo "   Email:       in-process (captured, not sent)"

dev:
	@echo "🚀 Starting development environment..."
	$(check-docker)
	@echo "🔪 Killing any process using port 8080..."
	@lsof -ti:8080 | xargs kill -9 2>/dev/null || true
	@echo "🔪 Killing any process using port 3000..."
	@lsof -ti:3000 | xargs kill -9 2>/dev/null || true
	@echo "🖥️  Starting frontend in detached mode..."
	@cd ../frontend && yarn dev &
	@echo "🔄 Starting backend with hot-reload..."
	@pwd && \
	if ! command -v air >/dev/null 2>&1; then \
		echo "📦 Installing Air (hot-reload)..."; \
		go install github.com/air-verse/air@latest; \
	fi; \
	air -c .air.toml

# Optimized Docker push target
# - Uses buildx with docker-container driver (supports registry cache)
# - Builds + pushes in one command
# - Tags immutable SHA + latest
# - Uses registry cache for fast rebuilds

IMAGE_NAME := $(DOCKERHUB_USERNAME)/payroll-backend
GIT_SHA    := $(shell git rev-parse --short HEAD)
BUILDER    := bb

push:
	@docker buildx inspect $(BUILDER) >/dev/null 2>&1 || docker buildx create --name $(BUILDER) --driver docker-container --use
	@docker buildx use $(BUILDER)
	@docker buildx inspect --bootstrap >/dev/null
	@echo "🔐 Logging into DockerHub..."
	@docker logout >/dev/null 2>&1 || true
	@echo "$(DOCKERHUB_PASSWORD)" | docker login -u "$(DOCKERHUB_USERNAME)" --password-stdin

	@echo "🏗️  Building & pushing AMD64 image (with inline cache)..."
	docker buildx build \
		--platform linux/amd64 \
		--tag $(IMAGE_NAME):latest \
		--tag $(IMAGE_NAME):$(GIT_SHA) \
		--cache-from type=registry,ref=$(IMAGE_NAME):latest \
		--build-arg BUILDKIT_INLINE_CACHE=1 \
		--push \
		.

	@echo "✅ Push complete: $(IMAGE_NAME):latest ($(GIT_SHA))"

deploy:
	@echo "🚀 Deploying frontend and backend on production droplet..."
	@echo "📡 Pulling latest frontend/backend images..."
	ssh root@$(PROD_SERVER) "cd /opt/payroll && docker compose pull frontend backend"
	@echo "🔄 Restarting frontend and backend services..."
	ssh root@$(PROD_SERVER) "cd /opt/payroll && docker compose up -d --force-recreate --no-deps frontend backend"
	@echo "🧹 Cleaning up unused images..."
	ssh root@$(PROD_SERVER) "docker image prune -f"
	@echo "✅ Frontend/backend deployment complete!"
	@echo "🌐 Frontend: https://$(PROD_SERVER)"
	@echo "🔧 Backend API: https://$(PROD_SERVER)/api"

# DEMO targets — build/push :demo images, deploy to demo.tingting.vip, reload demo DB.
# Demo stack coexists with vantai on the same host (separate compose, ports, nginx vhost).
push-demo:
	@docker buildx inspect $(BUILDER) >/dev/null 2>&1 || docker buildx create --name $(BUILDER) --driver docker-container --use
	@docker buildx use $(BUILDER)
	@docker buildx inspect --bootstrap >/dev/null
	@echo "🔐 Logging into DockerHub..."
	@docker logout >/dev/null 2>&1 || true
	@echo "$(DOCKERHUB_PASSWORD)" | docker login -u "$(DOCKERHUB_USERNAME)" --password-stdin
	@echo "🏗️  Building & pushing AMD64 backend image (:$(DEMO_TAG))..."
	docker buildx build \
		--platform linux/amd64 \
		--tag $(IMAGE_NAME):$(DEMO_TAG) \
		--push \
		.
	@echo "✅ Push complete: $(IMAGE_NAME):$(DEMO_TAG)"

deploy-demo:
	@echo "🚀 Deploying to $(DEMO_SERVER)..."
	@echo "📡 Pulling :demo frontend/backend images..."
	ssh root@$(DEMO_SERVER) "cd /opt/payroll && docker compose pull backend frontend"
	@echo "🔄 Restarting demo frontend/backend services..."
	ssh root@$(DEMO_SERVER) "cd /opt/payroll && docker compose up -d --force-recreate --no-deps backend frontend"
	@echo "🧹 Cleaning up unused images..."
	ssh root@$(DEMO_SERVER) "docker image prune -f"
	@echo "✅ Demo deployment complete!"
	@echo "🌐 Frontend: https://$(DEMO_SERVER)"
	@echo "🔧 Backend API: https://$(DEMO_SERVER)/api"

# Reload demo MySQL from LOCAL payroll_db (local dev DB). Overwrites demo data.
# Does NOT touch vantai (Postgres) or demo images.
demo-db:
	@echo "💾 Dumping local payroll_db → restoring to demo $(DEMO_SERVER)..."
	docker exec payroll-mysql mysqldump -u root -p'$(ROOT_PASSWORD)' --default-character-set=utf8mb4 --single-transaction --add-drop-table --routines --triggers payroll_db \
		| ssh root@$(DEMO_SERVER) "docker exec -i payroll-mysql mysql --default-character-set=utf8mb4 -u root -p'$(DEMO_DB_ROOT_PASSWORD)' payroll_db"
	@echo "🔐 Resetting all demo users to 'Admin123' (using app peppers)..."
	@HASH=$$(go run ./cmd/hashpw Admin123) && \
	printf "UPDATE users SET password='%s';" "$$HASH" \
		| ssh root@$(DEMO_SERVER) "docker exec -i payroll-mysql mysql -u root -p'$(DEMO_DB_ROOT_PASSWORD)' payroll_db"
	@echo "✅ Demo DB reloaded from local dev."

# Backup production database
backup:
	@echo "💾 Starting database backup from production server..."
	@TIMESTAMP=$$(date +%Y-%m-%d_%H%M%S) && \
	BACKUP_DIR="/Users/dev/Library/CloudStorage/OneDrive-Personal/backup/payroll_mysql_backup" && \
	BACKUP_FILE="payroll_mysql_backup_$$TIMESTAMP.sql" && \
	BACKUP_FILE_GZ="payroll_mysql_backup_$$TIMESTAMP.sql.gz" && \
	echo "📁 Creating backup directory if it doesn't exist..." && \
	mkdir -p "$$BACKUP_DIR" && \
	echo "🔗 Connecting to production server ($(PROD_SERVER))..." && \
	echo "📊 Creating MySQL dump of payroll_db database..." && \
	ssh root@$(PROD_SERVER) "docker exec \$$(docker ps --filter 'name=mysql' --format '{{.Names}}' | head -1) mysqldump -u root -p'$(MYSQL_ROOT_PASSWORD_PROD)' --single-transaction --routines --triggers --complete-insert payroll_db > /tmp/$$BACKUP_FILE" && \
	echo "🗜️  Compressing backup file..." && \
	ssh root@$(PROD_SERVER) "gzip /tmp/$$BACKUP_FILE" && \
	echo "✅ Verifying backup file exists and is not empty..." && \
	ssh root@$(PROD_SERVER) "if [ ! -s /tmp/$$BACKUP_FILE_GZ ]; then echo '❌ Backup file is empty or does not exist!'; exit 1; fi" && \
	echo "📥 Downloading backup file to local machine..." && \
	scp root@$(PROD_SERVER):/tmp/$$BACKUP_FILE_GZ "$$BACKUP_DIR/$$BACKUP_FILE_GZ" && \
	echo "🧹 Cleaning up temporary file on server..." && \
	ssh root@$(PROD_SERVER) "rm -f /tmp/$$BACKUP_FILE_GZ" && \
	echo "✅ Backup completed successfully!" && \
	echo "📂 Backup saved to: $$BACKUP_DIR/$$BACKUP_FILE_GZ" && \
	echo "📊 Backup size: $$(du -h "$$BACKUP_DIR/$$BACKUP_FILE_GZ" | cut -f1)" && \
	echo "🔍 To verify backup content: gunzip -c $$BACKUP_DIR/$$BACKUP_FILE_GZ | head -20"

# Load latest database backup from OneDrive
restore:
	@echo "📥 Loading latest database backup from OneDrive..."
	@BACKUP_DIR="/Users/dev/Library/CloudStorage/OneDrive-Personal/backup/payroll_mysql_backup" && \
	LATEST_BACKUP=$$(ls -t "$$BACKUP_DIR"/payroll_mysql_backup_*.sql.gz 2>/dev/null | head -1) && \
	if [ -z "$$LATEST_BACKUP" ]; then \
		echo "❌ No backup files found in $$BACKUP_DIR"; \
		exit 1; \
	fi && \
	echo "📂 Found latest backup: $$(basename $$LATEST_BACKUP)" && \
	echo "📊 Backup size: $$(du -h "$$LATEST_BACKUP" | cut -f1)" && \
	echo "🗑️  Dropping existing database..." && \
	docker exec payroll-mysql mysql -u root -p'$(ROOT_PASSWORD)' -e "DROP DATABASE IF EXISTS payroll_db;" && \
	echo "🆕 Creating fresh database..." && \
	docker exec payroll-mysql mysql -u root -p'$(ROOT_PASSWORD)' -e "CREATE DATABASE payroll_db;" && \
	echo "⏳ Decompressing and restoring to payroll_db..." && \
	gunzip -c "$$LATEST_BACKUP" | docker exec -i payroll-mysql mysql -u root -p'$(ROOT_PASSWORD)' payroll_db && \
	echo "✅ Database restored successfully from backup!" && \
	echo "🔐 Generating Argon2id hash for 'Admin123' (using app peppers)..." && \
	HASH=$$(go run ./cmd/hashpw Admin123) && \
	echo "🔐 Updating ALL user passwords to 'Admin123'..." && \
	docker exec payroll-mysql mysql -u root -p'$(ROOT_PASSWORD)' payroll_db \
		-e "UPDATE users SET password='$$HASH';" && \
	echo "✅ All users now have password: Admin123"

# Reset every user's password to "Admin123" without dropping the DB.
# Useful when you want a fresh login state but want to keep current dev data.
reset-passwords:
	@echo "🔐 Generating Argon2id hash for 'Admin123' (using app peppers)..."
	@HASH=$$(go run ./cmd/hashpw Admin123) && \
	docker exec payroll-mysql mysql -u root -p'$(ROOT_PASSWORD)' payroll_db \
		-e "UPDATE users SET password='$$HASH';" && \
	echo "✅ All users now have password: Admin123"

# Lint command for code quality
lint:
	@echo "🔍 Running code formatting and linting..."
	gofmt -s -w .
	@if command -v golangci-lint >/dev/null 2>&1; then \
		if ! golangci-lint run ./...; then \
			echo "⚠️  golangci-lint failed, falling back to go vet ./..."; \
			go vet ./...; \
		fi; \
	else \
		echo "⚠️  golangci-lint not found. Install it for better linting."; \
		go vet ./...; \
	fi

# Run unit tests across the repo
test:
	@echo "🧪 Running unit tests with coverage..."
	@rm -f coverage.out coverage.html
	GOCACHE=$(CURDIR)/.gocache go test -covermode=atomic -coverprofile=coverage.out ./...
	@echo "\n📈 Coverage summary (total):"
	@go tool cover -func=coverage.out | grep total || true
	@go tool cover -html=coverage.out -o coverage.html
	@echo "✅ HTML report generated at: coverage.html"

# Run integration API tests against local backend
api-test:
	@echo "🧪 Running API integration tests..."
	go run ./tests/integration/

# Enable adminer access from internet
adminer-on:
	@echo "🔓 Enabling adminer access from internet..."
	@ssh root@$(PROD_SERVER) "cd /opt/payroll && docker compose start adminer && ufw allow from any to any port 8081 proto tcp"
	@echo "✅ Adminer is now accessible from internet"
	@echo "🌐 Access adminer at: https://$(PROD_SERVER)/adminer"

adminer-off:
	@echo "🔒 Disabling adminer access from internet..."
	@ssh root@$(PROD_SERVER) "ufw delete allow from any to any port 8081 proto tcp 2>/dev/null || true && cd /opt/payroll && docker compose stop adminer"
	@echo "✅ Adminer access from internet is now disabled"
	@echo "🔐 Port 8081 blocked and adminer container stopped"

sync:
	@echo "📤 Syncing templates to production..."
	@scp templates/*.xlsx root@$(PROD_SERVER):/opt/payroll/templates/
	@scp templates/email/*.html root@$(PROD_SERVER):/opt/payroll/templates/email/

# Unified mock server (OnePay PayOut + 9Pay disbursement) on :9001.
# `make db` also starts it; this target starts only the sandbox service.
sandbox:
	$(check-docker)
	docker compose -f $(COMPOSE_FILE) up -d --build sandbox

sandbox-logs:
	docker compose -f $(COMPOSE_FILE) logs -f sandbox

sandbox-rebuild:
	docker compose -f $(COMPOSE_FILE) build --no-cache sandbox
	docker compose -f $(COMPOSE_FILE) up -d sandbox

# Top up the mock OnePay balance (default +100,000,000 VND on localhost:9001).
onepay-topup:
	cd sandbox/onepay && ./topup.sh $${AMOUNT:-100000000} $${HOST:-http://localhost:9001}

# Clean up unused Docker resources
clean:
	@echo "🧹 Cleaning up Docker resources..."
	docker system prune -a --volumes
	@echo "✅ Docker cleanup completed!"

# Open Adminer over an SSH tunnel -> http://localhost:18081 (no internet exposure).
# Starts the Adminer container on prod, forwards localhost:18081 -> prod loopback:8081, opens the page.
# Ctrl-C closes the tunnel. Safer than adminer-on: never opens UFW 8081 or the public /adminer path.
adminer:
	@echo "🔓 Opening Adminer over SSH tunnel (private, no public exposure)..."
	@ssh root@$(PROD_SERVER) "cd /opt/payroll && docker compose start adminer" 2>/dev/null || true
	@echo "🌐 Adminer: http://localhost:18081   (Server: payroll-mysql ; creds from DB_DSN in /opt/payroll/.env)"
	@echo "⏎  Press Ctrl-C to close the tunnel."
	@(sleep 1 && (open http://localhost:18081 || xdg-open http://localhost:18081 || true)) &
	@ssh -N -L 18081:127.0.0.1:8081 root@$(PROD_SERVER)

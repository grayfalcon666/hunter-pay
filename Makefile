.PHONY: infra-up infra-down infra-reset \
	createdb dropdb \
	migrate-simple_bank migrate-escrow_db migrate-payment_db \
	migratedown-simple_bank migratedown-escrow_db migratedown-payment_db \
	dev dev-stop \
	server-simplebank server-escrow_bounty server-user_profile server-payment server-gateway \
	server-all

# =============================================================================
# Infrastructure
# =============================================================================

infra-up:
	docker compose up -d

infra-down:
	docker compose down

infra-reset:
	docker compose down -v
	docker compose up -d

# =============================================================================
# Database management (runs inside the postgres container)
# =============================================================================

createdb:
	docker exec escrow-postgres createdb --username=root --owner=root simple_bank || true
	docker exec escrow-postgres createdb --username=root --owner=root escrow_db     || true
	docker exec escrow-postgres createdb --username=root --owner=root payment_db     || true

dropdb:
	docker exec escrow-postgres dropdb simple_bank || true
	docker exec escrow-postgres dropdb escrow_db     || true
	docker exec escrow-postgres dropdb payment_db     || true

# =============================================================================
# Migrations
# =============================================================================

SIMPLE_BANK_URL=postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable
ESCROW_DB_URL=postgresql://root:secret@localhost:5433/escrow_db?sslmode=disable
PAYMENT_DB_URL=postgresql://root:secret@localhost:5434/payment_db?sslmode=disable

migrate-simple_bank:
	migrate -path ./migrations/simple_bank -database "$(SIMPLE_BANK_URL)" -verbose up

migrate-escrow_db:
	migrate -path ./migrations/escrow_db -database "$(ESCROW_DB_URL)" -verbose up

migrate-payment_db:
	migrate -path ./migrations/payment_db -database "$(PAYMENT_DB_URL)" -verbose up

migratedown-simple_bank:
	migrate -path ./migrations/simple_bank -database "$(SIMPLE_BANK_URL)" -verbose down 1

migratedown-escrow_db:
	migrate -path ./migrations/escrow_db -database "$(ESCROW_DB_URL)" -verbose down 1

migratedown-payment_db:
	migrate -path ./migrations/payment_db -database "$(PAYMENT_DB_URL)" -verbose down 1

# =============================================================================
# Services
# =============================================================================

server-simplebank:
	cd simplebank && go run main.go

server-escrow_bounty:
	cd escrow-bounty && go run main.go

server-user_profile:
	cd user-profile-service && go run main.go

server-payment:
	cd payment-service && go run main.go

server-gateway:
	cd gateway && go run main.go

server-all:
	cd simplebank           && go run main.go &
	cd escrow-bounty       && go run main.go &
	cd user-profile-service && go run main.go &
	cd payment-service     && go run main.go &
	cd gateway             && go run main.go

# =============================================================================
# tmux dev session
# =============================================================================

dev:
	./dev.sh

dev-stop:
	./dev.sh --kill

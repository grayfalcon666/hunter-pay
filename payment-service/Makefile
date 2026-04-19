DB_URL=postgresql://root:secret@localhost:5434/payment_db?sslmode=disable

server:
	go run main.go

postgres:
	@echo "Using shared escrow-postgres container on port 5434"

createdb:
	docker exec escrow-postgres createdb --username=root --owner=root payment_db

dropdb:
	docker exec escrow-postgres dropdb payment_db

migrateup:
	migrate -path ../migrations/payment_db -database "$(DB_URL)" -verbose up

migratedown:
	migrate -path ../migrations/payment_db -database "$(DB_URL)" -verbose down

proto:
	rm -f pb/*.go
	mkdir -p doc/swagger
	rm -f doc/swagger/*.swagger.json
	protoc --proto_path=proto \
		--go_out=pb --go_opt=paths=source_relative \
		--go-grpc_out=pb --go-grpc_opt=paths=source_relative \
		--grpc-gateway_out=pb --grpc-gateway_opt=paths=source_relative \
		--openapiv2_out=doc/swagger --openapiv2_opt=allow_merge=true,merge_file_name=payment_service \
		proto/*.proto

.PHONY: server postgres createdb dropdb migrateup migratedown proto
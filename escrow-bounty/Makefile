DB_URL=postgresql://root:secret@localhost:5433/escrow_db?sslmode=disable

server:
	go run main.go

postgres:
	@echo "Using shared escrow-postgres container on port 5433"

createdb:
	docker exec escrow-postgres createdb --username=root --owner=root escrow_db

dropdb:
	docker exec -it escrow-postgres dropdb escrow_db

migrateup:
	migrate -path ../migrations/escrow_db -database "$(DB_URL)" -verbose up

migratedown:
	migrate -path ../migrations/escrow_db -database "$(DB_URL)" -verbose down

proto:
	rm -f pb/*.go
	mkdir -p doc/swagger
	rm -f doc/swagger/*.swagger.json
	protoc --proto_path=proto \
		--go_out=pb --go_opt=paths=source_relative \
		--go-grpc_out=pb --go-grpc_opt=paths=source_relative \
		--grpc-gateway_out=pb --grpc-gateway_opt=paths=source_relative \
		--openapiv2_out=doc/swagger --openapiv2_opt=allow_merge=true,merge_file_name=escrow_bounty \
		proto/*.proto

mock:
	mockgen -package mockdb -destination db/mock/store.go github.com/grayfalcon666/escrow-bounty/db Store,BankClient

.PHONY: server postgres createdb dropdb migrateup migratedown proto
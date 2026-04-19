ifneq (,$(wildcard ./app.env))
    include ./app.env
    export
endif

DB_URL=$(DB_SOURCE)

postgres:
	docker exec escrow-postgres createdb --username=root --owner=root simple_bank

createdb:
	docker exec escrow-postgres createdb --username=root --owner=root simple_bank

redis:
	docker run --name redis -p 6379:6379 -d redis:7-alpine

dropdb:
	docker exec escrow-postgres dropdb simple_bank

migrateup:
	$(eval STEP := $(filter-out $@,$(MAKECMDGOALS)))
	migrate -path ../migrations/simple_bank -database "$(DB_URL)" -verbose up $(STEP)

migratedown:
	@# 提取参数，无参数时默认1
	$(eval STEP := $(filter-out $@,$(MAKECMDGOALS)))
	migrate -path ../migrations/simple_bank -database "$(DB_URL)" -verbose down $(if $(STEP),$(STEP),1)

migrateversion:
	migrate -path ../migrations/simple_bank -database "$(DB_URL)" version

sqlc:
	sqlc generate

test:
	go test -v -cover ./...

server:
	go run main.go

mockgen:
	mockgen -package mockdb -destination ./db/mock/store.go simplebank/db/sqlc Store

proto:
	rm -f pb/*.go
	rm -f doc/swagger/*.swagger.json
	protoc --proto_path=proto --go_out=pb --go_opt=paths=source_relative \
	--go-grpc_out=pb --go-grpc_opt=paths=source_relative \
	--grpc-gateway_out=pb --grpc-gateway_opt=paths=source_relative \
	--openapiv2_out=doc/swagger --openapiv2_opt=allow_merge=true,merge_file_name=simple_bank \
	proto/*.proto

# 伪目标声明 (防止和同名文件冲突)
.PHONY: postgres createdb dropdb migrateup migratedown sqlc test server mockgen proto

# 处理Makefile传参的兼容逻辑（比如make migrateup 2时，忽略多余参数）
%:
	@:
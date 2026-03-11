postgres:
	docker run --name bank-pg --network bank-network -p 5432:5432 \
		-e POSTGRES_USER=root -e POSTGRES_PASSWORD=password -d postgres

createdb:
	docker exec -it bank-pg createdb --username=root --owner=root bank

dropdb:
	docker exec -it bank-pg dropdb bank

migrateup:
	migrate -path db/migration -database "postgresql://root:password@localhost:5432/bank?sslmode=disable" -verbose up

migrateup1:
	migrate -path db/migration -database "postgresql://root:password@localhost:5432/bank?sslmode=disable" -verbose up 1

migratedown:
	migrate -path db/migration -database "postgresql://root:password@localhost:5432/bank?sslmode=disable" -verbose down

migratedown1:
	migrate -path db/migration -database "postgresql://root:password@localhost:5432/bank?sslmode=disable" -verbose down 1

db_docs:
	dbdocs build doc/db.dbml

db_schema:
	dbml2sql doc/db.dbml -o doc/schema.sql --postgres

sqlc:
	sqlc generate

test:
	go test -v -cover ./...

server:
	go run main.go

# ─────────────────────────────────────────────────────────────
# proto: regenerates all Go pb files + a merged OpenAPI spec.
#
# What happens on `make proto`:
#   1. Cleans all previously generated pb/*.go files
#   2. Cleans the old swagger spec
#   3. Ensures doc/swagger/ output directory exists
#   4. Runs protoc with four plugins in one pass:
#        --go_out            → Go message types  (pb/*.go)
#        --go-grpc_out       → gRPC server/client stubs
#        --grpc-gateway_out  → HTTP↔gRPC gateway
#        --openapiv2_out     → Merged OpenAPI 2.0 JSON spec
#
# The openapiv2 flags:
#   allow_merge=true        → collapses all .proto files into one spec
#   merge_file_name=go_bank → output: doc/swagger/go_bank.swagger.json
#
# After running this command the spec is embedded into the binary via
# //go:embed in gapi/swagger.go — no separate file copy needed in prod.
# protoc --proto_path=proto --go_out=pb --go_opt=paths=source_relative --go-grpc_out=pb --go-grpc_opt=paths=source_relative --grpc-gateway_out=pb --grpc-gateway_opt=paths=source_relative --openapiv2_out=doc/swagger --openapiv2_opt=allow_merge=true --openapiv2_opt=merge_file_name=go_bank --openapiv2_opt=output_format=json proto/*.proto
# cp doc/swagger/go_bank.swagger.json gapi/go_bank.swagger.json
# ─────────────────────────────────────────────────────────────
proto:
	rm -f pb/*.go
	rm -f doc/swagger/*.json
	mkdir -p doc/swagger
	protoc \
		--proto_path=proto \
		--go_out=pb --go_opt=paths=source_relative \
		--go-grpc_out=pb --go-grpc_opt=paths=source_relative \
		--grpc-gateway_out=pb --grpc-gateway_opt=paths=source_relative \
		--openapiv2_out=doc/swagger \
		--openapiv2_opt=allow_merge=true \
		--openapiv2_opt=merge_file_name=go_bank \
		--openapiv2_opt=output_format=json \
		proto/*.proto
	cp doc/swagger/go_bank.swagger.json gapi/go_bank.swagger.json

evans:
	evans --host localhost --port 9090 -r repl

.PHONY: createdb dropdb postgres migrateup migratedown migrateup1 migratedown1 \
        db_docs db_schema sqlc test server proto evans
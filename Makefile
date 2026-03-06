postgres:
	docker run --name bank-pg  --network bank-network -p 5432:5432 -e POSTGRES_USER=root -e POSTGRES_PASSWORD=password -d postgres

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

.PHONY: createdb dropdb postgres migrateup migratedown migrateup1 migratedown1 db_docs db_schema sqlc test server
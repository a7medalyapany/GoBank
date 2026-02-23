#!/bin/sh

set -e

echo "Running database migrations..."
/go-bank/migrate -path /go-bank/migration -database "$DB_URL" -verbose up

echo "Starting the server..."
exec "$@"

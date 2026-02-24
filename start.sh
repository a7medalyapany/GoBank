#!/bin/sh
set -e

echo "Loading environment variables..."
export $(cat app.env | xargs)

echo "Running database migrations..."
/go-bank/migrate -path /go-bank/migration -database "$DB_URL" -verbose up

echo "Starting the server..."
exec "$@"
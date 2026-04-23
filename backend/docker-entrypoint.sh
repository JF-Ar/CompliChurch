#!/bin/sh
set -e

echo "Running database migrations..."
migrate -path /app/db/migrations -database "$DATABASE_URL" up

echo "Starting server..."
exec /app/api

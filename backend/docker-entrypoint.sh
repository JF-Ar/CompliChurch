#!/bin/sh
set -e

echo "Running database migrations..."
echo "Migrations directory contents:"
ls /app/db/migrations
migrate -source "file:///app/db/migrations" -database "$DATABASE_URL" up

echo "Starting server..."
exec /app/api

#!/bin/sh
set -e

echo "waiting for postgres to be ready..."
until pg_isready -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" >/dev/null 2>&1; do
  echo "postgres not ready yet, sleeping..."
  sleep 1
done

echo "postgres is ready — running migrations"
goose -dir ./migrations postgres "host=$DB_HOST port=$DB_PORT user=$DB_USER password=$DB_PASS dbname=$DB_NAME sslmode=disable" up

echo "starting wallet service"
./wallet-service

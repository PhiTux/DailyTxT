#!/bin/sh
set -e

# Start the Go backend in background
# Environment variables consumed by backend (see utils.InitSettings):
#   DATA_PATH, DEVELOPMENT, SECRET_TOKEN, LOGOUT_AFTER_DAYS, ALLOWED_HOSTS, INDENT, ALLOW_REGISTRATION, ADMIN_PASSWORD

# Ensure data directory exists
mkdir -p "${DATA_PATH:-/data}"

# Run backend
/usr/local/bin/dailytxt &
BACKEND_PID=$!

echo "Started backend (PID $BACKEND_PID)"

# Start nginx in foreground
exec nginx -g 'daemon off;'

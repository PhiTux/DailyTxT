#!/bin/sh
set -e

# Ensure data directory exists
mkdir -p "${DATA_PATH:-/data}"

# Run backend in background
/usr/local/bin/dailytxt &
BACKEND_PID=$!

echo "Started backend (PID $BACKEND_PID)"

# Edit some files to make dailytxt work on a subpath provided by BASE_PATH 
if [ -n "${BASE_PATH:-}" ]; then
    echo "Configuring frontend for BASE_PATH: $BASE_PATH"

    # remove leading and trailing slash if exists
    BASE_PATH="${BASE_PATH#/}"
    BASE_PATH="${BASE_PATH%/}"
    # print the BASE_PATH being used
    echo "Using BASE_PATH: '$BASE_PATH'"

    # Update base href in app.html
    sed -i "s|href=\"/|href=\"|g" /usr/share/nginx/html/index.html
    
    # Update base path in app.html
    sed -i "s|base: \"|base: \"/$BASE_PATH|g" /usr/share/nginx/html/index.html

    # Update import-paths in app.html
    sed -i "s|import(\"|import(\"/$BASE_PATH|g" /usr/share/nginx/html/index.html

    # Update manifest.webmanifest base
    sed -i "s|start_url\":\"/|start_url\":\"/$BASE_PATH|g" /usr/share/nginx/html/manifest.webmanifest
    sed -i "s|src\":\"|src\":\"/$BASE_PATH|g" /usr/share/nginx/html/manifest.webmanifest
fi

# Start nginx in foreground
exec nginx -g 'daemon off;'

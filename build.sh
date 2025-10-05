#!/usr/bin/env bash
set -euo pipefail

# Build script: pulls translations from Tolgee, then builds the Docker image.
# Requirements:
# - Environment variable TOLGEE_API_KEY must be set
# - tolgee CLI available
# - Docker installed
#
# Usage:
#   TOLGEE_API_KEY=... ./build.sh [IMAGE_TAG]
#
# Example:
#   export TOLGEE_API_KEY=xxxxx
#   ./build.sh phitux/dailytxt:2.x.x-testing.1

if [[ $# -lt 1 ]]; then
	echo "Usage: ./build.sh <IMAGE_TAG>"
	exit 1
fi
IMAGE_TAG="$1"

if [[ -z "${TOLGEE_API_KEY:-}" ]]; then
	echo "Error: TOLGEE_API_KEY is not set."
	echo "Please export your Tolgee API key, e.g.:"
	echo "  export TOLGEE_API_KEY=xxxxxxxxxxxxxxxx"
	exit 1
fi

echo "[1/3] Writing IMAGE_TAG (=version) into backend/version ..."
echo "$IMAGE_TAG" > ./backend/version

echo "[2/3] Pulling translations from Tolgee into frontend/src/i18n ..."
tolgee pull --path ./frontend/src/i18n --api-key $TOLGEE_API_KEY

echo "[3/3] Building Docker image: $IMAGE_TAG"
docker build -t phitux/dailytxt:$IMAGE_TAG .

echo "Done. Image built: $IMAGE_TAG"

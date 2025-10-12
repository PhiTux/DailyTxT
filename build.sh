#!/usr/bin/env bash
set -euo pipefail

# Build script: pulls translations from Tolgee, then builds the Docker image.
# Requirements:
# - Environment variable TOLGEE_API_KEY must be set
# - tolgee CLI available
# - Docker installed
#
# Usage:
#   TOLGEE_API_KEY=... ./build.sh IMAGE_TAG [--push]
#
# Example:
#   export TOLGEE_API_KEY=xxxxx
#   ./build.sh phitux/dailytxt:2.x.x-testing.1 [--push]

if [[ $# -lt 1 ]]; then
	echo "Usage: ./build.sh <IMAGE_TAG> [--push]"
	exit 1
fi
IMAGE_TAG="$1"
PUSH_FLAG="${2:-}"

if [[ -z "${TOLGEE_API_KEY:-}" ]]; then
	echo "Error: TOLGEE_API_KEY is not set."
	echo "Please export your Tolgee API key, e.g.:"
	echo "  export TOLGEE_API_KEY=xxxxxxxxxxxxxxxx"
	exit 1
fi

echo "[1/3] Writing IMAGE_TAG (=version) into backend/version ..."
echo "$IMAGE_TAG" > ./backend/version

echo -e "\n[2/3] Pulling translations from Tolgee into frontend/src/i18n ..."
tolgee pull --path ./frontend/src/i18n --api-key $TOLGEE_API_KEY

echo -e "\n[3/3] Building Docker image: $IMAGE_TAG"
if [[ -z "$PUSH_FLAG" ]]; then
	# Default: local docker build (no push)
	docker build -t phitux/dailytxt:$IMAGE_TAG .
else
	case "$PUSH_FLAG" in
		--push)
			echo "Pushing image to Docker registry..."
			# Build for multiple platforms and push to registry
			docker buildx build \
				--platform linux/amd64,linux/arm64 \
				-t phitux/dailytxt:$IMAGE_TAG \
				--push .
			;;
		*)
			echo "Unknown second argument: '$PUSH_FLAG'"
			echo "Usage: ./build.sh <IMAGE_TAG> [--push]"
			exit 1
			;;
	esac
fi

echo -e "\nDone. Image built: $IMAGE_TAG"

#!/usr/bin/env bash
set -euo pipefail

# Build script: pulls translations from Tolgee, then builds the Docker image.
# Requirements:
# - Environment variable TOLGEE_API_KEY must be set
# - tolgee CLI available
# - Docker installed
#
# Usage:
#   TOLGEE_API_KEY=... ./build.sh PRIMARY_TAG [SECONDARY_TAG] [--push]
#
# Example:
#   export TOLGEE_API_KEY=xxxxx
#   ./build.sh phitux/dailytxt:2.x.x latest --push
#   ./build.sh phitux/dailytxt:2.x.x --push
#   ./build.sh phitux/dailytxt:2.x.x-testing.2

if [[ $# -lt 1 ]]; then
	echo "Usage: ./build.sh <PRIMARY_TAG> [SECONDARY_TAG] [--push]"
	exit 1
fi
PRIMARY_TAG="$1"
SECONDARY_TAG=""
PUSH_FLAG=""

# Parse optional SECONDARY_TAG and --push flag (in any order for 2nd/3rd args)
if [[ $# -ge 2 ]]; then
	if [[ "$2" == "--push" ]]; then
		PUSH_FLAG="--push"
	else
		SECONDARY_TAG="$2"
	fi
fi
if [[ $# -ge 3 ]]; then
	if [[ "$3" == "--push" ]]; then
		PUSH_FLAG="--push"
	else
		echo "Unknown third argument: '$3'"
		echo "Usage: ./build.sh <PRIMARY_TAG> [SECONDARY_TAG] [--push]"
		exit 1
	fi
fi

# Disallow 'latest' as PRIMARY_TAG; only allowed as SECONDARY_TAG
if [[ "$PRIMARY_TAG" == "latest" ]]; then
	echo "Error: PRIMARY_TAG must not be 'latest'. Use a specific version (e.g., '2.3.1') and optionally set SECONDARY_TAG to 'latest'."
	exit 1
fi

if [[ -z "${TOLGEE_API_KEY:-}" ]]; then
	echo "Error: TOLGEE_API_KEY is not set."
	echo "Please export your Tolgee API key, e.g.:"
	echo "  export TOLGEE_API_KEY=xxxxxxxxxxxxxxxx"
	exit 1
fi



echo "[1/3] Writing IMAGE_TAG (=version) into backend/version ..."
echo "$PRIMARY_TAG" > ./backend/version

echo -e "\n[2/3] Pulling translations from Tolgee into frontend/src/i18n ..."
tolgee pull --path ./frontend/src/i18n --api-key $TOLGEE_API_KEY

echo -e "\n[3/3] Building Docker image: $PRIMARY_TAG${SECONDARY_TAG:+, $SECONDARY_TAG}"
if [[ -z "$PUSH_FLAG" ]]; then
	# Default: local docker build (no push)
	if [[ -n "$SECONDARY_TAG" ]]; then
		docker build \
			-t phitux/dailytxt:$PRIMARY_TAG \
			-t phitux/dailytxt:$SECONDARY_TAG \
			.
	else
		docker build -t phitux/dailytxt:$PRIMARY_TAG .
	fi
else
	case "$PUSH_FLAG" in
		--push)
			echo "Pushing image to Docker registry..."
			# Build for multiple platforms and push to registry
			if [[ -n "$SECONDARY_TAG" ]]; then
				docker buildx build \
					--platform linux/amd64,linux/arm64 \
					-t phitux/dailytxt:$PRIMARY_TAG \
					-t phitux/dailytxt:$SECONDARY_TAG \
					--push .
			else
				docker buildx build \
					--platform linux/amd64,linux/arm64 \
					-t phitux/dailytxt:$PRIMARY_TAG \
					--push .
			fi
			;;
		*)
			echo "Unknown flag: '$PUSH_FLAG'"
			echo "Usage: ./build.sh <PRIMARY_TAG> [SECONDARY_TAG] [--push]"
			exit 1
			;;
	esac
fi

if [[ -z "$PUSH_FLAG" ]]; then
	PUSH_STATUS="(not pushed)"
else
	PUSH_STATUS="(PUSHED!)"
fi
echo -e "\nDone. Image built: $PRIMARY_TAG${SECONDARY_TAG:+, $SECONDARY_TAG} $PUSH_STATUS"

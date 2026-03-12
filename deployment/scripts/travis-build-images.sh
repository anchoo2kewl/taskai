#!/bin/bash
# Travis CI image build script for TaskAI
# Builds multi-arch Docker images and pushes to Docker Hub
#
# Required Travis CI env vars:
#   DOCKERHUB_USERNAME, DOCKERHUB_TOKEN

set -euo pipefail

GIT_SHA=$(git rev-parse --short HEAD)
VERSION=$(cat VERSION 2>/dev/null || echo "dev")
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

echo "=== Building TaskAI Images ==="
echo "Version: $VERSION"
echo "Commit:  $GIT_SHA"
echo ""

# Login to Docker Hub
echo "$DOCKERHUB_TOKEN" | docker login -u "$DOCKERHUB_USERNAME" --password-stdin

# Set up buildx for multi-arch
docker buildx create --name multiarch --use 2>/dev/null || docker buildx use multiarch

# 1. API image
echo "--- Building taskai-api image ---"
docker buildx build \
  --platform linux/arm64,linux/amd64 \
  --file api/Dockerfile \
  --build-arg "VERSION=$VERSION" \
  --build-arg "GIT_COMMIT=$GIT_SHA" \
  --build-arg "BUILD_TIME=$BUILD_TIME" \
  --tag "anchoo2kewl/taskai-api:latest" \
  --tag "anchoo2kewl/taskai-api:$GIT_SHA" \
  --push \
  ./api

# 2. Web image (context is root — Dockerfile copies from web/ and docs/)
echo "--- Building taskai-web image ---"
docker buildx build \
  --platform linux/arm64,linux/amd64 \
  --file web/Dockerfile \
  --build-arg "VERSION=$VERSION" \
  --build-arg "GIT_COMMIT=$GIT_SHA" \
  --build-arg "BUILD_TIME=$BUILD_TIME" \
  --tag "anchoo2kewl/taskai-web:latest" \
  --tag "anchoo2kewl/taskai-web:$GIT_SHA" \
  --push \
  .

# 3. MCP image
echo "--- Building taskai-mcp image ---"
docker buildx build \
  --platform linux/arm64,linux/amd64 \
  --file mcp/Dockerfile \
  --tag "anchoo2kewl/taskai-mcp:latest" \
  --tag "anchoo2kewl/taskai-mcp:$GIT_SHA" \
  --push \
  ./mcp

# 4. Yjs processor image
echo "--- Building taskai-yjs image ---"
docker buildx build \
  --platform linux/arm64,linux/amd64 \
  --file api/internal/yjs-processor/Dockerfile \
  --tag "anchoo2kewl/taskai-yjs:latest" \
  --tag "anchoo2kewl/taskai-yjs:$GIT_SHA" \
  --push \
  ./api/internal/yjs-processor

echo ""
echo "=== TaskAI images pushed ==="
echo "Tags: latest, $GIT_SHA"

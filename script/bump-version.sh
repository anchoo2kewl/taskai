#!/bin/bash

# Bump version script for TaskAI
# Usage: ./script/bump-version.sh [major|minor|patch]

set -e

VERSION_FILE="VERSION"

if [ ! -f "$VERSION_FILE" ]; then
    echo "ERROR: VERSION file not found"
    exit 1
fi

# Read current version
CURRENT_VERSION=$(cat $VERSION_FILE)
echo "Current version: $CURRENT_VERSION"

# Parse version
IFS='.' read -r MAJOR MINOR PATCH <<< "$CURRENT_VERSION"

# Determine bump type (default to patch)
BUMP_TYPE="${1:-patch}"

case "$BUMP_TYPE" in
    major)
        MAJOR=$((MAJOR + 1))
        MINOR=0
        PATCH=0
        ;;
    minor)
        MINOR=$((MINOR + 1))
        PATCH=0
        ;;
    patch)
        PATCH=$((PATCH + 1))
        ;;
    *)
        echo "ERROR: Invalid bump type. Use: major, minor, or patch"
        exit 1
        ;;
esac

# Create new version
NEW_VERSION="$MAJOR.$MINOR.$PATCH"

# Write new version
echo "$NEW_VERSION" > $VERSION_FILE
echo "Bumped version: $CURRENT_VERSION -> $NEW_VERSION"

# Git operations
git add $VERSION_FILE
git commit -m "chore: bump version to $NEW_VERSION"
git tag -a "v$NEW_VERSION" -m "Version $NEW_VERSION"

echo ""
echo "Version bumped successfully!"
echo "To push: git push origin main --tags"
echo "To deploy staging: gh workflow run deploy-staging.yml"
echo "To deploy production: gh workflow run deploy-production.yml -f ref=\"main\""

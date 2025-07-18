#!/bin/bash
set -euo pipefail

# Check if VERSION is set and not empty
if [ -z "${VERSION:-}" ]; then
  echo "ERROR: VERSION environment variable is not set."
  exit 1
fi

PUBLIC_REPO="bitrise-io/bitrise-plugins-agent"
DIST_DIR="dist"

# Gather binaries into an array (works on macOS Bash 3.2)
BINARIES=()
while IFS= read -r file; do
  BINARIES+=("$file")
done < <(find "$DIST_DIR" -type f -name 'bitrise-plugins-agent*' ! -name "*.txt")

# Build artifacts in snapshot mode (doesn't publish, just builds)
GORELEASER_CURRENT_TAG=${VERSION} goreleaser release --snapshot --clean

# Create release if not exists (ignore error if already exists)
gh release create "$VERSION" --repo "$PUBLIC_REPO" --title "$VERSION" --notes "Automated release" || true

# Upload all binaries (overwrite if already present)
gh release upload "$VERSION" "${BINARIES[@]}" --repo "$PUBLIC_REPO" --clobber

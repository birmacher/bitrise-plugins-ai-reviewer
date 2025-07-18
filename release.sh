#!/bin/bash
set -euo pipefail

if [ -z "${VERSION:-}" ]; then
  echo "ERROR: VERSION environment variable is not set."
  exit 1
fi

PUBLIC_REPO="bitrise-io/bitrise-plugins-agent"
DIST_DIR="dist"

# Build binaries
GORELEASER_CURRENT_TAG=${VERSION} goreleaser release --snapshot --clean

# Flatten the dist folder
find "$DIST_DIR" -mindepth 2 -type f -name 'bitrise-plugins-agent' | while IFS= read -r filepath; do
  dirpath=$(dirname "$filepath")
  parentdir=$(basename "$dirpath")
  flatname=$(echo "$parentdir" | sed -E 's/_v[0-9.]+$//')
  mv "$filepath" "$DIST_DIR/$flatname"
  rmdir "$dirpath" 2>/dev/null || true
done

# Gather binaries into an array
BINARIES=()
while IFS= read -r file; do
  BINARIES+=("$file")
done < <(find "$DIST_DIR" -maxdepth 1 -type f -name 'bitrise-plugins-agent*' ! -name "*.txt")

# Create release if not exists
gh release create "$VERSION" --repo "$PUBLIC_REPO" --title "$VERSION" --notes "Automated release" || true

# Upload all binaries
gh release upload "$VERSION" "${BINARIES[@]}" --repo "$PUBLIC_REPO" --clobber

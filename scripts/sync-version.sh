#!/bin/sh
# Sync the embedded version file with the repository VERSION file.
# Run this before building if VERSION has changed.
set -eu

root=$(cd "$(dirname "$0")/.." && pwd)
src="$root/VERSION"
dst="$root/internal/version/version.txt"

if [ ! -f "$src" ]; then
    echo "VERSION file not found at $src" >&2
    exit 1
fi

if [ -f "$dst" ] && cmp -s "$src" "$dst"; then
    exit 0
fi

cp "$src" "$dst"
echo "Updated internal/version/version.txt from VERSION"

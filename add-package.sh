#!/usr/bin/env bash

if [ $# -eq 0 ]; then
    echo "Usage: $0 <package-name> [package-name...]"
    exit 1
fi

PACKAGES_FILE="$(dirname "$0")/package-sets/global.nix"

for PACKAGE in "$@"; do
    # Insert the package before the closing bracket
    sed -i '' "/^]$/i\\
  $PACKAGE
" "$PACKAGES_FILE"
    echo "Added '$PACKAGE' to package-sets/global.nix"
done

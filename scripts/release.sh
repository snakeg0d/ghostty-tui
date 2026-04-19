#!/usr/bin/env bash
# SPDX-License-Identifier: MIT
# Part of ghostty-tui — https://github.com/snakeg0d/ghostty-tui
#
# Build a release ZIP for the current host triple.
#
# Output:
#   dist/ghostty-tui-v<version>-<target-triple>.zip
#
# Contents:
#   ghostty-tui                      (release binary)
#   scripts/ghostty-manager.sh       (the bash companion)
#   scripts/release.sh               (this script — for reproducibility)
#   assets/                          (themes, shaders, configs)
#   docs/                            (ASSETS.md, ARCHITECTURE.md, ADOPTING.md)
#   README.md / LICENSE / CHANGELOG.md / CONTRIBUTING.md
#
# Usage:
#   ./scripts/release.sh
#
# Cross-compile hint:
#   CARGO_BUILD_TARGET=x86_64-unknown-linux-gnu ./scripts/release.sh
#   (the host triple in the filename follows rustc's `-vV | grep host`
#    unless CARGO_BUILD_TARGET is set, in which case it uses that.)

set -euo pipefail

cd "$(dirname "$0")/.."

VERSION=$(grep '^version' Cargo.toml | head -1 | cut -d'"' -f2)

if [ -n "${CARGO_BUILD_TARGET:-}" ]; then
    TARGET="$CARGO_BUILD_TARGET"
    cargo build --release --target "$TARGET"
    BIN="target/$TARGET/release/ghostty-tui"
else
    TARGET=$(rustc -vV | awk '/^host:/ {print $2}')
    cargo build --release
    BIN="target/release/ghostty-tui"
fi

if [ ! -x "$BIN" ]; then
    echo "error: binary not found at $BIN" >&2
    exit 1
fi

DIST_NAME="ghostty-tui-v${VERSION}-${TARGET}"
DIST_DIR="dist/${DIST_NAME}"

rm -rf "$DIST_DIR"
mkdir -p "$DIST_DIR/scripts"

cp "$BIN"                       "$DIST_DIR/"
cp scripts/ghostty-manager.sh   "$DIST_DIR/scripts/"
cp scripts/release.sh           "$DIST_DIR/scripts/"
cp -R assets                    "$DIST_DIR/"
cp -R docs                      "$DIST_DIR/"
cp README.md LICENSE CHANGELOG.md CONTRIBUTING.md "$DIST_DIR/"

(
    cd dist
    rm -f "${DIST_NAME}.zip"
    zip -qr "${DIST_NAME}.zip" "${DIST_NAME}"
)

echo ""
echo "Built: dist/${DIST_NAME}.zip"
ls -lh "dist/${DIST_NAME}.zip"
echo ""
echo "Contents preview:"
unzip -l "dist/${DIST_NAME}.zip" | tail -n +4 | head -n 25
echo "  ..."
unzip -l "dist/${DIST_NAME}.zip" | tail -n 5

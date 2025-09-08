#!/bin/bash
# SPDX-License-Identifier: MIT
# Copyright 2025 Oppie Thunder Contributors
#
# Build script for Helios native platform

set -euo pipefail

# Configuration
VERSION="${1:-0.0.1}"
BUILD_DIR="./build"
BINARY_NAME="helios"
MAIN_PATH="./cmd/helios-cli"

echo "🚀 Building Helios v${VERSION} for native platform..."

# Clean and create build directory
rm -rf "${BUILD_DIR}"
mkdir -p "${BUILD_DIR}"

# Detect current platform
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

# Normalize architecture names
case "$ARCH" in
    x86_64)
        ARCH="amd64"
        ;;
    aarch64)
        ARCH="arm64"
        ;;
esac

output_name="${BINARY_NAME}-${OS}-${ARCH}"
archive_name="${BINARY_NAME}-${VERSION}-${OS}-${ARCH}.tar.gz"

echo "📦 Building ${output_name}..."

# Build binary
go build \
    -ldflags="-w -s -X main.version=${VERSION}" \
    -o "${BUILD_DIR}/${output_name}" \
    "${MAIN_PATH}"

echo "✅ Built binary: ${BUILD_DIR}/${output_name}"

# Test the binary
echo "🧪 Testing binary..."
"${BUILD_DIR}/${output_name}" --help

# Create tarball
cd "${BUILD_DIR}"
tar -czf "${archive_name}" "${output_name}"
echo "✅ Created archive: ${archive_name}"

# Create checksums
sha256sum "${archive_name}" > "checksums.txt"
echo "🔐 Generated checksums"

cd ..

echo "🎉 Build complete!"
echo "📋 Built for: ${OS}-${ARCH}"
echo "📦 Archive: ${BUILD_DIR}/${archive_name}"
echo "📂 Build directory: ${BUILD_DIR}/"

ls -la "${BUILD_DIR}/"
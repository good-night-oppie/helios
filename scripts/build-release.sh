#!/bin/bash
# SPDX-License-Identifier: MIT
# Copyright 2025 Oppie Thunder Contributors
#
# Build script for Helios cross-platform releases

set -euo pipefail

# Configuration
VERSION="${1:-0.0.1}"
BUILD_DIR="./build"
BINARY_NAME="helios"
MAIN_PATH="./cmd/helios-cli"

# Platform configurations
PLATFORMS=(
    "linux/amd64"
    "linux/arm64"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
    "windows/arm64"
)

echo "🚀 Building Helios v${VERSION} for multiple platforms..."

# Clean and create build directory
rm -rf "${BUILD_DIR}"
mkdir -p "${BUILD_DIR}"

# Build for each platform
for platform in "${PLATFORMS[@]}"; do
    platform_split=(${platform//\// })
    GOOS=${platform_split[0]}
    GOARCH=${platform_split[1]}
    
    output_name="${BINARY_NAME}-${GOOS}-${GOARCH}"
    if [ $GOOS = "windows" ]; then
        output_name="${output_name}.exe"
    fi
    
    echo "📦 Building ${output_name}..."
    
    env GOOS=$GOOS GOARCH=$GOARCH go build \
        -ldflags="-w -s -X main.version=${VERSION}" \
        -o "${BUILD_DIR}/${output_name}" \
        "${MAIN_PATH}"
    
    # Create tarball for non-Windows platforms
    if [ $GOOS != "windows" ]; then
        cd "${BUILD_DIR}"
        tar -czf "${BINARY_NAME}-${VERSION}-${GOOS}-${GOARCH}.tar.gz" "${output_name}"
        rm "${output_name}"
        cd ..
        echo "✅ Created ${BINARY_NAME}-${VERSION}-${GOOS}-${GOARCH}.tar.gz"
    else
        # Create zip for Windows
        cd "${BUILD_DIR}"
        zip -q "${BINARY_NAME}-${VERSION}-${GOOS}-${GOARCH}.zip" "${output_name}"
        rm "${output_name}"
        cd ..
        echo "✅ Created ${BINARY_NAME}-${VERSION}-${GOOS}-${GOARCH}.zip"
    fi
done

# Create checksums
echo "🔐 Generating checksums..."
cd "${BUILD_DIR}"
sha256sum * > "checksums.txt"
cd ..

echo "🎉 Build complete! Artifacts in ${BUILD_DIR}/"
ls -la "${BUILD_DIR}/"

echo ""
echo "📋 Release Summary:"
echo "Version: ${VERSION}"
echo "Platforms: ${#PLATFORMS[@]}"
echo "Build directory: ${BUILD_DIR}/"
echo ""
echo "Next steps:"
echo "1. Test binaries"
echo "2. Create GitHub release"
echo "3. Upload artifacts"
#!/bin/bash

# build-release.sh - Build gitf binaries for multiple platforms
# Usage: ./build-release.sh [version]

VERSION="${1:-0.1.0}"
BUILD_DIR="dist"
LDFLAGS="-s -w -X main.Version=${VERSION}"

echo "🚀 Building GitF v${VERSION} for multiple platforms..."
mkdir -p "${BUILD_DIR}"

# Array of platforms: (os/arch)
PLATFORMS=(
    "linux/amd64"
    "linux/arm64"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
)

build_platform() {
    local platform=$1
    local os="${platform%%/*}"
    local arch="${platform##*/}"
    local output="${BUILD_DIR}/gitf-${os}-${arch}"
    
    if [ "$os" = "windows" ]; then
        output="${output}.exe"
    fi
    
    echo "📦 Building for ${os}/${arch}..."
    GOOS="${os}" GOARCH="${arch}" go build \
        -ldflags "${LDFLAGS}" \
        -o "${output}" \
        ./cmd/gitf
    
    if [ $? -eq 0 ]; then
        local size=$(du -h "${output}" | cut -f1)
        echo "   ✅ ${output} (${size})"
    else
        echo "   ❌ Failed to build ${os}/${arch}"
        return 1
    fi
}

# Build for all platforms
for platform in "${PLATFORMS[@]}"; do
    build_platform "$platform"
done

# Create checksums
echo ""
echo "📝 Generating checksums..."
cd "${BUILD_DIR}"
sha256sum gitf-* > SHA256SUMS
echo "✅ Checksums written to SHA256SUMS"
cd ..

echo ""
echo "✨ Release build complete!"
echo "📁 Binaries location: ${BUILD_DIR}/"
echo ""
echo "📊 Summary:"
du -h "${BUILD_DIR}"/*
echo ""
echo "📋 Checksums:"
cat "${BUILD_DIR}/SHA256SUMS"

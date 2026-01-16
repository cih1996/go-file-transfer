#!/bin/bash

# Clean up previous builds
rm -rf dist
mkdir -p dist

# Helper function to build
build() {
    OS=$1
    ARCH=$2
    NAME="jp-file"
    SERVER_NAME="jp-server"
    
    echo "Building for $OS/$ARCH..."
    
    SUFFIX=""
    if [ "$OS" == "windows" ]; then
        SUFFIX=".exe"
    fi

    # Build jp-file
    env GOOS=$OS GOARCH=$ARCH go build -o "dist/${NAME}-${OS}-${ARCH}${SUFFIX}" ./cmd/jp-file
    # Build jp-server
    env GOOS=$OS GOARCH=$ARCH go build -o "dist/${SERVER_NAME}-${OS}-${ARCH}${SUFFIX}" ./cmd/jp-server
}

# Mac (Intel & Apple Silicon)
build darwin amd64
build darwin arm64

# Linux (x86_64 & ARM64)
build linux amd64
build linux arm64

# Windows (x64)
build windows amd64

echo "Build complete! Check the dist/ directory."
ls -lh dist/

#!/bin/bash

# Configuration
REPO="cih1996/go-file-transfer" # You need to replace this
BINARY_NAME="${TARGET:-jp-file}"
INSTALL_DIR="/usr/local/bin"

# Detect OS and Arch
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

if [ "$ARCH" == "x86_64" ]; then
    ARCH="amd64"
elif [ "$ARCH" == "aarch64" ] || [ "$ARCH" == "arm64" ]; then
    ARCH="arm64"
else
    echo "Unsupported architecture: $ARCH"
    exit 1
fi

echo "Detected system: $OS/$ARCH"

# Determine download URL (Using GitHub Releases - Latest)
GITHUB_URL="https://github.com/$REPO/releases/latest/download/${BINARY_NAME}-${OS}-${ARCH}"
# Using a common GitHub Proxy
MIRROR_URL="https://github-1308564197.cos.ap-guangzhou.myqcloud.com/go-file-transfer/latest/${BINARY_NAME}-${OS}-${ARCH}"

# Temporary file
TMP_FILE="/tmp/${BINARY_NAME}"

download_file() {
    url=$1
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL -o "$TMP_FILE" "$url"
    elif command -v wget >/dev/null 2>&1; then
        wget -qO "$TMP_FILE" "$url"
    else
        echo "Error: curl or wget is required."
        exit 1
    fi
}

echo "Trying to download from domestic mirror ($MIRROR_URL)..."
if download_file "$MIRROR_URL"; then
    echo "Download from mirror successful."
else
    echo "Mirror failed. Trying official GitHub ($GITHUB_URL)..."
    if ! download_file "$GITHUB_URL"; then
        echo "Error: Download failed from both sources."
        exit 1
    fi
fi

if [ ! -f "$TMP_FILE" ]; then
    echo "Download failed."
    exit 1
fi

chmod +x "$TMP_FILE"

echo "Installing to $INSTALL_DIR..."
# Try to move with sudo if needed
if [ -w "$INSTALL_DIR" ]; then
    mv "$TMP_FILE" "$INSTALL_DIR/$BINARY_NAME"
else
    sudo mv "$TMP_FILE" "$INSTALL_DIR/$BINARY_NAME"
fi

if [ $? -eq 0 ]; then
    echo "Successfully installed $BINARY_NAME to $INSTALL_DIR"
    echo "Run '$BINARY_NAME' to get started."
else
    echo "Installation failed."
    exit 1
fi

#!/bin/sh
set -e

# Termidar installer
# Usage: curl -sSfL https://raw.githubusercontent.com/N-Erickson/termidar/main/install.sh | sh

REPO="N-Erickson/termidar"
INSTALL_DIR="/usr/local/bin"
BINARY="termidar"

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
  darwin) OS="darwin" ;;
  linux)  OS="linux" ;;
  *)
    echo "Error: Unsupported OS: $OS"
    echo "Download manually from https://github.com/$REPO/releases"
    exit 1
    ;;
esac

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *)
    echo "Error: Unsupported architecture: $ARCH"
    echo "Download manually from https://github.com/$REPO/releases"
    exit 1
    ;;
esac

# Get latest version
VERSION=$(curl -sSf "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | sed -E 's/.*"v([^"]+)".*/\1/')
if [ -z "$VERSION" ]; then
  echo "Error: Could not determine latest version"
  exit 1
fi

echo "Installing termidar v${VERSION} (${OS}/${ARCH})..."

# Download
FILENAME="${BINARY}_${VERSION}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/$REPO/releases/download/v${VERSION}/${FILENAME}"

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

curl -sSfL "$URL" -o "$TMPDIR/$FILENAME"

# Extract
tar xzf "$TMPDIR/$FILENAME" -C "$TMPDIR"

# Install
if [ -w "$INSTALL_DIR" ]; then
  mv "$TMPDIR/$BINARY" "$INSTALL_DIR/$BINARY"
else
  echo "Need sudo to install to $INSTALL_DIR"
  sudo mv "$TMPDIR/$BINARY" "$INSTALL_DIR/$BINARY"
fi

chmod +x "$INSTALL_DIR/$BINARY"

echo ""
echo "termidar v${VERSION} installed to $INSTALL_DIR/$BINARY"
echo "Run 'termidar' to start!"

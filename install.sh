#!/bin/sh
# Install data-cli — https://github.com/ngtrvu/data-cli
set -e

REPO="ngtrvu/data-cli"
BINARY="data"

# Use /usr/local/bin if writable, otherwise fall back to ~/.local/bin
if [ -w "/usr/local/bin" ]; then
  INSTALL_DIR="/usr/local/bin"
else
  INSTALL_DIR="$HOME/.local/bin"
  mkdir -p "$INSTALL_DIR"
fi

# ── Detect OS and arch ────────────────────────────────────────────────────────

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64) ARCH="arm64" ;;
  arm64)   ARCH="arm64" ;;
  *)
    echo "Unsupported architecture: $ARCH"
    exit 1
    ;;
esac

case "$OS" in
  darwin|linux) ;;
  *)
    echo "Unsupported OS: $OS. On Windows use: scoop install data-cli"
    exit 1
    ;;
esac

# ── Resolve latest version ────────────────────────────────────────────────────

VERSION=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" \
  | grep '"tag_name"' | sed 's/.*"tag_name": *"\(.*\)".*/\1/')

if [ -z "$VERSION" ]; then
  echo "Could not determine latest version"
  exit 1
fi

# ── Download and install ──────────────────────────────────────────────────────

ARCHIVE="data-cli_${VERSION#v}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/$REPO/releases/download/$VERSION/$ARCHIVE"
TMP=$(mktemp -d)

echo "Installing data-cli $VERSION ($OS/$ARCH)..."

curl -fsSL "$URL" -o "$TMP/$ARCHIVE"
tar -xzf "$TMP/$ARCHIVE" -C "$TMP"
install -m755 "$TMP/data-cli_${VERSION#v}_${OS}_${ARCH}/$BINARY" "$INSTALL_DIR/$BINARY"
rm -rf "$TMP"

echo "Installed to $INSTALL_DIR/$BINARY"

if [ "$INSTALL_DIR" = "$HOME/.local/bin" ]; then
  echo ""
  echo "Add to PATH if not already:"
  echo "  echo 'export PATH=\"\$HOME/.local/bin:\$PATH\"' >> ~/.zshrc && source ~/.zshrc"
fi

echo "Run: data --help"

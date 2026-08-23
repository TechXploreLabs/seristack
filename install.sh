#!/usr/bin/env bash
set -euo pipefail

REPO="TechXploreLabs/seristack"
BINARY_NAME="seristack"
INSTALL_DIR="/usr/local/bin"

# ------------------------------------------------------------
# Detect OS
# ------------------------------------------------------------
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"

case "$OS" in
  linux*)
    OS="linux"
    ;;
  darwin*)
    OS="darwin"
    ;;
  *)
    echo "Error: Unsupported OS: $OS"
    exit 1
    ;;
esac

# ------------------------------------------------------------
# Detect Architecture
# ------------------------------------------------------------
ARCH="$(uname -m)"

case "$ARCH" in
  x86_64|amd64)
    ARCH="amd64"
    ;;
  aarch64|arm64)
    ARCH="arm64"
    ;;
  *)
    echo "Error: Unsupported architecture: $ARCH"
    exit 1
    ;;
esac

# ------------------------------------------------------------
# Get latest release
# ------------------------------------------------------------
echo "Checking latest seristack release..."

LATEST_TAG="$(
  curl -fsSL \
    -H "Accept: application/vnd.github+json" \
    "https://api.github.com/repos/$REPO/releases/latest" |
  grep '"tag_name":' |
  head -1 |
  sed -E 's/.*"tag_name":[[:space:]]*"([^"]+)".*/\1/'
)"

if [[ -z "$LATEST_TAG" ]]; then
  echo "Error: Could not determine latest release."
  exit 1
fi

VERSION="${LATEST_TAG#v}"

# ------------------------------------------------------------
# Build archive name
# ------------------------------------------------------------
ARCHIVE_NAME="${BINARY_NAME}_${VERSION}_${OS}_${ARCH}.tar.gz"

DOWNLOAD_URL="https://github.com/$REPO/releases/download/${LATEST_TAG}/${ARCHIVE_NAME}"

echo ""
echo "seristack release : $LATEST_TAG"
echo "OS                : $OS"
echo "Architecture      : $ARCH"
echo "Archive           : $ARCHIVE_NAME"
echo ""

# ------------------------------------------------------------
# Temporary directory
# ------------------------------------------------------------
TMP_DIR="$(mktemp -d)"

cleanup() {
  rm -rf "$TMP_DIR"
}

trap cleanup EXIT

ARCHIVE_PATH="$TMP_DIR/$ARCHIVE_NAME"

# ------------------------------------------------------------
# Download
# ------------------------------------------------------------
echo "Downloading..."

curl -fL \
  --retry 3 \
  --retry-delay 2 \
  "$DOWNLOAD_URL" \
  -o "$ARCHIVE_PATH"

# ------------------------------------------------------------
# Extract
# ------------------------------------------------------------
echo "Extracting..."

tar -xzf "$ARCHIVE_PATH" -C "$TMP_DIR"

BINARY_PATH="$TMP_DIR/$BINARY_NAME"

if [[ ! -f "$BINARY_PATH" ]]; then
  echo "Error: $BINARY_NAME was not found in the release archive."
  echo ""
  echo "Archive contents:"
  tar -tzf "$ARCHIVE_PATH"
  exit 1
fi

chmod +x "$BINARY_PATH"

# ------------------------------------------------------------
# Install
# ------------------------------------------------------------
echo "Installing $BINARY_NAME to $INSTALL_DIR..."

sudo install -m 0755 "$BINARY_PATH" "$INSTALL_DIR/$BINARY_NAME"

# ------------------------------------------------------------
# Verify
# ------------------------------------------------------------
echo ""
echo "Installation complete!"
echo ""

if command -v "$BINARY_NAME" >/dev/null 2>&1; then
  "$BINARY_NAME" --help
else
  echo "Installed successfully:"
  echo "  $INSTALL_DIR/$BINARY_NAME"
  echo ""
  echo "$INSTALL_DIR is not currently in your PATH."
fi

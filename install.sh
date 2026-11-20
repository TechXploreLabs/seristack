#!/usr/bin/env bash
# seristack installer for Linux and macOS.
#
# Environment overrides:
#   SERISTACK_VERSION  exact release tag to install, e.g. v0.4.2 (default: latest)
#   INSTALL_DIR        target directory (default: /usr/local/bin)
set -euo pipefail

REPO="TechXploreLabs/seristack"
BINARY_NAME="seristack"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

die()  { echo "Error: $*" >&2; exit 1; }
warn() { echo "Warning: $*" >&2; }
need() { command -v "$1" >/dev/null 2>&1 || die "'$1' is required but not installed."; }

need curl
need tar
need uname
need mktemp

sha256_of() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$1" | awk '{print $1}'
  else
    return 1
  fi
}

# ------------------------------------------------------------
# Detect OS and architecture
# ------------------------------------------------------------
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$OS" in
  linux*)  OS="linux" ;;
  darwin*) OS="darwin" ;;
  mingw*|msys*|cygwin*)
    die "Windows detected. Use the PowerShell installer instead." ;;
  *) die "Unsupported OS: $OS" ;;
esac

ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64)  ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) die "Unsupported architecture: $ARCH" ;;
esac

# ------------------------------------------------------------
# Resolve the release tag (no GitHub API, so no API rate limit)
# ------------------------------------------------------------
archive_name_for() { echo "${BINARY_NAME}_${1#v}_${OS}_${ARCH}.tar.gz"; }

if [[ -n "${SERISTACK_VERSION:-}" ]]; then
  TAG="${SERISTACK_VERSION}"
else
  echo "Checking latest $BINARY_NAME release (tags starting with 'v')..."

  # The releases feed lists published releases, newest first. It needs no API
  # token and has no API rate limit. Keep tags like v1.2.3 (a digit after the
  # 'v'); skip other releases in the repo and pre-releases such as v1.2.3-rc1.
  FEED="$(curl -fsSL "https://github.com/$REPO/releases.atom")" \
    || die "Could not reach GitHub to list releases."
  CANDIDATES="$(printf '%s\n' "$FEED" \
    | grep -oE '/releases/tag/v[0-9][^"<]*' \
    | sed 's#.*/releases/tag/##' \
    | grep -v -- '-' || true)"

  # Use the newest one that really has a build for this OS and architecture.
  TAG=""
  for t in $CANDIDATES; do
    if curl -fsSL -r 0-0 -o /dev/null \
         "https://github.com/$REPO/releases/download/$t/$(archive_name_for "$t")" 2>/dev/null; then
      TAG="$t"
      break
    fi
  done

  [[ -n "$TAG" ]] || die "No recent release with a tag starting with 'v' has a ${OS}/${ARCH} build. Set SERISTACK_VERSION=<tag> to install a specific one."
fi

[[ "$TAG" =~ ^v[0-9]+\.[0-9]+ ]] || die "Release tag must start with 'v' and a version number (got '$TAG')."
VERSION="${TAG#v}"

ARCHIVE_NAME="$(archive_name_for "$TAG")"
BASE_URL="https://github.com/$REPO/releases/download/${TAG}"

echo ""
echo "seristack release : $TAG"
echo "OS                : $OS"
echo "Architecture      : $ARCH"
echo "Archive           : $ARCHIVE_NAME"
echo ""

# ------------------------------------------------------------
# Temporary directory
# ------------------------------------------------------------
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

ARCHIVE_PATH="$TMP_DIR/$ARCHIVE_NAME"

# ------------------------------------------------------------
# Download
# ------------------------------------------------------------
echo "Downloading..."
curl -fL --retry 3 --retry-delay 2 "$BASE_URL/$ARCHIVE_NAME" -o "$ARCHIVE_PATH" \
  || die "Download failed: $BASE_URL/$ARCHIVE_NAME"

# ------------------------------------------------------------
# Verify checksum (when the release publishes a checksums file)
# ------------------------------------------------------------
SUMS_PATH="$TMP_DIR/checksums.txt"
SUMS_FOUND=""
for name in "${BINARY_NAME}_${VERSION}_checksums.txt" "checksums.txt"; do
  if curl -fsSL "$BASE_URL/$name" -o "$SUMS_PATH" 2>/dev/null; then
    SUMS_FOUND="$name"
    break
  fi
done

if [[ -n "$SUMS_FOUND" ]]; then
  EXPECTED="$(awk -v f="$ARCHIVE_NAME" '$2 == f || $2 == "*" f {print $1; exit}' "$SUMS_PATH")"
  [[ -n "$EXPECTED" ]] || die "No checksum entry for $ARCHIVE_NAME in $SUMS_FOUND."
  ACTUAL="$(sha256_of "$ARCHIVE_PATH")" \
    || die "Neither sha256sum nor shasum is available to verify the download."
  [[ "$EXPECTED" == "$ACTUAL" ]] \
    || die "Checksum mismatch for $ARCHIVE_NAME (expected $EXPECTED, got $ACTUAL)."
  echo "Checksum verified."
else
  warn "this release has no checksums file; the download could not be verified."
fi

# ------------------------------------------------------------
# Extract
# ------------------------------------------------------------
echo "Extracting..."
tar -xzf "$ARCHIVE_PATH" -C "$TMP_DIR" || die "Failed to extract the archive."

BINARY_PATH="$(find "$TMP_DIR" -type f -name "$BINARY_NAME" | head -n 1)"
if [[ -z "$BINARY_PATH" ]]; then
  echo "Archive contents:" >&2
  tar -tzf "$ARCHIVE_PATH" >&2
  die "$BINARY_NAME was not found in the release archive."
fi
chmod +x "$BINARY_PATH"

# ------------------------------------------------------------
# Install: no sudo unless it is needed
# ------------------------------------------------------------
echo "Installing $BINARY_NAME to $INSTALL_DIR..."

if [[ -w "$INSTALL_DIR" ]] || { [[ ! -e "$INSTALL_DIR" ]] && mkdir -p "$INSTALL_DIR" 2>/dev/null; }; then
  install -m 0755 "$BINARY_PATH" "$INSTALL_DIR/$BINARY_NAME"
elif command -v sudo >/dev/null 2>&1; then
  echo "Using sudo to write to $INSTALL_DIR."
  sudo install -m 0755 "$BINARY_PATH" "$INSTALL_DIR/$BINARY_NAME"
else
  INSTALL_DIR="$HOME/.local/bin"
  warn "cannot write to the target directory and sudo is not available; using $INSTALL_DIR."
  mkdir -p "$INSTALL_DIR"
  install -m 0755 "$BINARY_PATH" "$INSTALL_DIR/$BINARY_NAME"
fi

# ------------------------------------------------------------
# Verify the file that was just installed (not whatever is first in PATH)
# ------------------------------------------------------------
echo ""
if "$INSTALL_DIR/$BINARY_NAME" --help >/dev/null 2>&1; then
  echo "Installation complete: $INSTALL_DIR/$BINARY_NAME ($TAG)"
else
  warn "installed to $INSTALL_DIR/$BINARY_NAME, but running it with --help failed."
fi

case ":$PATH:" in
  *":$INSTALL_DIR:"*) ;;
  *) echo ""; echo "$INSTALL_DIR is not in your PATH. Add it, for example:"
     echo "  export PATH=\"$INSTALL_DIR:\$PATH\"" ;;
esac

FOUND="$(command -v "$BINARY_NAME" 2>/dev/null || true)"
if [[ -n "$FOUND" && "$FOUND" != "$INSTALL_DIR/$BINARY_NAME" ]]; then
  warn "another $BINARY_NAME comes first in PATH: $FOUND"
fi

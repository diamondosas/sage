#!/usr/bin/env sh
# Ensure Sage CLI binary is installed and accessible
set -eu

if command -v sage >/dev/null 2>&1; then
  exit 0
fi

SKILL_DIR="$(cd "$(dirname "$0")/.." && pwd)"
BIN_DIR="${HOME}/.local/bin"
mkdir -p "$BIN_DIR"

if [ -x "${BIN_DIR}/sage" ]; then
  export PATH="${BIN_DIR}:${PATH}"
  exit 0
fi

# Determine OS and Architecture
OS="$(uname -s)"
case "$OS" in
  Linux*)  PLATFORM="linux" ;;
  Darwin*) PLATFORM="darwin" ;;
  CYGWIN*|MINGW*|MSYS*) PLATFORM="windows" ;;
  *) echo "Unsupported OS: $OS" >&2; exit 1 ;;
esac

ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported CPU arch: $ARCH" >&2; exit 1 ;;
esac

EXT=""
[ "$PLATFORM" = "windows" ] && EXT=".exe"

REPO="${SAGE_REPO:-diamondosas/sage}"
BINARY_NAME="sage-${PLATFORM}-${ARCH}${EXT}"
DOWNLOAD_URL="https://github.com/${REPO}/releases/latest/download/${BINARY_NAME}"
TARGET="${BIN_DIR}/sage${EXT}"

echo "Downloading Sage CLI binary from ${DOWNLOAD_URL}..." >&2

if command -v curl >/dev/null 2>&1; then
  curl -fsSL "$DOWNLOAD_URL" -o "$TARGET" || true
elif command -v wget >/dev/null 2>&1; then
  wget -qO "$TARGET" "$DOWNLOAD_URL" || true
fi

if [ -f "$TARGET" ] && [ -s "$TARGET" ]; then
  chmod +x "$TARGET"
  echo "Sage binary installed to ${TARGET}" >&2
  exit 0
else
  echo "Could not download binary. Please install Sage manually: https://github.com/${REPO}" >&2
  exit 1
fi

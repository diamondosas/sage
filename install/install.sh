#!/usr/bin/env bash
set -e

# --- Variables (Edit these if needed) ---
# Repository to download releases from (format: owner/repo)
# CHANGE HERE: Replace with your GitHub repository owner/name
REPO="${SAGE_REPO:-your-org/sage}"
# Target installation directory (e.g., /usr/local/bin, ~/.local/bin, or ./bin)
# CHANGE HERE: Set custom install directory via INSTALL_DIR env var
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
# Desired release tag ('latest' or specific version like 'v1.0.0')
TAG="${SAGE_TAG:-latest}"

# Detect OS and Architecture matching GitHub Actions workflow matrix
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

# Construct binary name as defined in .github/workflows
BIN="sage-${OS}-${ARCH}"
SRC=""

# 1. Use prebuilt binary if present in ./dist or ./bin
if [ -f "./dist/${BIN}" ]; then
  SRC="./dist/${BIN}"
elif [ -f "./bin/sage" ]; then
  SRC="./bin/sage"
# 2. Or compile locally if Go is installed
elif command -v go >/dev/null 2>&1 && [ -d "./cmd/sage" ]; then
  echo "Compiling Sage with local Go compiler..."
  mkdir -p ./bin
  go build -o ./bin/sage ./cmd/sage
  SRC="./bin/sage"
# 3. Otherwise download from GitHub Releases (no Go required!)
else
  URL="https://github.com/${REPO}/releases/download/${TAG}/${BIN}"
  SRC="/tmp/sage"
  echo "Downloading prebuilt sage (${OS}/${ARCH}) - no Go required..."
  if ! curl -fsSL "$URL" -o "$SRC"; then
    echo "⚠️  Release binary not found at ${URL}."
    echo "    Falling back to local build if Go is available."
    if command -v go >/dev/null 2>&1; then
      go build -o ./bin/sage ./cmd/sage
      SRC="./bin/sage"
    else
      echo "❌ Error: Neither prebuilt binary nor Go compiler found."
      exit 1
    fi
  fi
fi

# Ensure target installation directory exists
mkdir -p "$INSTALL_DIR" 2>/dev/null || true

# Install binary
echo "Installing to ${INSTALL_DIR}..."
chmod +x "$SRC"

SRC_REAL="$(cd "$(dirname "$SRC")" 2>/dev/null && pwd)/$(basename "$SRC")"
DEST_REAL="$(cd "$INSTALL_DIR" 2>/dev/null && pwd)/sage"

if [ "$SRC_REAL" != "$DEST_REAL" ]; then
  if [ -w "$INSTALL_DIR" ]; then
    cp "$SRC" "${INSTALL_DIR}/sage"
  elif command -v sudo >/dev/null 2>&1 && [ -t 0 ]; then
    sudo cp "$SRC" "${INSTALL_DIR}/sage"
  else
    # Fallback to user home ~/.local/bin if /usr/local/bin is not writable
    INSTALL_DIR="${HOME}/.local/bin"
    mkdir -p "$INSTALL_DIR"
    cp "$SRC" "${INSTALL_DIR}/sage"
    echo "Installed to ${INSTALL_DIR} (no sudo required)"
  fi
fi

# Ensure directory is in PATH
case ":$PATH:" in
  *":${INSTALL_DIR}:"*) ;;
  *)
    SHELL_RC="${HOME}/.$(basename "$SHELL")rc"
    echo "export PATH=\"\$PATH:${INSTALL_DIR}\"" >> "$SHELL_RC"
    echo "Added ${INSTALL_DIR} to ${SHELL_RC}. Reload your shell or run: source ${SHELL_RC}"
    ;;
esac

echo "Sage installed successfully! Run 'sage --help' to get started."

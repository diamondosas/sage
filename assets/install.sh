#!/usr/bin/env sh
# Sage Installation Script
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/diamondosas/sage/main/assets/install.sh | sh
# Options (via environment variables):
#   SAGE_VERSION    Specific version tag to install (default: latest)
#   BINDIR          Installation directory (default: /usr/local/bin or ~/.local/bin)
#   SAGE_REPO       GitHub repository (default: diamondosas/sage)

set -eu

# Formatting and colors
if [ -t 1 ]; then
  RED='\033[0;31m'
  GREEN='\033[0;32m'
  BLUE='\033[0;34m'
  YELLOW='\033[1;33m'
  BOLD='\033[1m'
  NC='\033[0m'
else
  RED=''
  GREEN=''
  BLUE=''
  YELLOW=''
  BOLD=''
  NC=''
fi

info() {
  printf "${BLUE}${BOLD}==>${NC} ${BOLD}%s${NC}\n" "$1"
}

success() {
  printf "${GREEN}${BOLD}==>${NC} ${GREEN}${BOLD}%s${NC}\n" "$1"
}

warn() {
  printf "${YELLOW}${BOLD}warning:${NC} %s\n" "$1" >&2
}

error() {
  printf "${RED}${BOLD}error:${NC} %s\n" "$1" >&2
  exit 1
}

# Check for required download tools
downloader=""
if command -v curl >/dev/null 2>&1; then
  downloader="curl"
elif command -v wget >/dev/null 2>&1; then
  downloader="wget"
else
  error "Neither curl nor wget was found. Please install either tool to proceed."
fi

download_file() {
  url="$1"
  dest="$2"
  if [ "$downloader" = "curl" ]; then
    curl -fsSL "$url" -o "$dest"
  else
    wget -qO "$dest" "$url"
  fi
}

fetch_text() {
  url="$1"
  if [ "$downloader" = "curl" ]; then
    curl -fsSL "$url" 2>/dev/null || true
  else
    wget -qO- "$url" 2>/dev/null || true
  fi
}

# Detect OS
detect_os() {
  kernel="$(uname -s)"
  case "$kernel" in
    Linux*)   echo "linux" ;;
    Darwin*)  echo "darwin" ;;
    CYGWIN*|MINGW*|MSYS*) echo "windows" ;;
    *) error "Unsupported operating system kernel: $kernel" ;;
  esac
}

# Detect Architecture
detect_arch() {
  machine="$(uname -m)"
  case "$machine" in
    x86_64|amd64)   echo "amd64" ;;
    aarch64|arm64)  echo "arm64" ;;
    armv8*)         echo "arm64" ;;
    *) error "Unsupported CPU architecture: $machine (Sage provides binaries for amd64 and arm64)" ;;
  esac
}

OS="$(detect_os)"
ARCH="$(detect_arch)"
EXT=""
if [ "$OS" = "windows" ]; then
  EXT=".exe"
fi

REPO="${SAGE_REPO:-diamondosas/sage}"
REQUESTED_VERSION="${SAGE_VERSION:-latest}"
BINARY_NAME="sage-${OS}-${ARCH}${EXT}"
TARGET_EXEC="sage${EXT}"

info "Detected platform: ${OS}/${ARCH}"

# Determine release version and download URL
if [ "$REQUESTED_VERSION" = "latest" ]; then
  info "Checking for latest release of ${REPO}..."
  API_URL="https://api.github.com/repos/${REPO}/releases/latest"
  API_RESPONSE="$(fetch_text "$API_URL")"
  
  # Try parsing tag_name from JSON response
  TAG_NAME="$(printf '%s\n' "$API_RESPONSE" | grep '"tag_name":' | head -n 1 | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/' || true)"
  
  if [ -n "$TAG_NAME" ]; then
    VERSION="$TAG_NAME"
    DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${BINARY_NAME}"
  else
    # Fallback to direct latest download URL without API dependency (handles rate limits)
    VERSION="latest"
    DOWNLOAD_URL="https://github.com/${REPO}/releases/latest/download/${BINARY_NAME}"
  fi
else
  VERSION="$REQUESTED_VERSION"
  DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${BINARY_NAME}"
fi

info "Resolved version: ${VERSION}"
info "Downloading ${BINARY_NAME}..."

# Setup temporary scratch workspace
TMP_DIR="$(mktemp -d -t sage-install-XXXXXX)"
cleanup() {
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT INT TERM

TMP_FILE="${TMP_DIR}/${TARGET_EXEC}"

# Download binary with fallback retry if direct URL fails
if ! download_file "$DOWNLOAD_URL" "$TMP_FILE"; then
  # Fallback: check if assets are packaged as tar.gz or zip
  TAR_URL="https://github.com/${REPO}/releases/download/${VERSION}/sage-${OS}-${ARCH}.tar.gz"
  ZIP_URL="https://github.com/${REPO}/releases/download/${VERSION}/sage-${OS}-${ARCH}.zip"
  
  if download_file "$TAR_URL" "${TMP_DIR}/sage.tar.gz" 2>/dev/null; then
    tar -xzf "${TMP_DIR}/sage.tar.gz" -C "$TMP_DIR"
    find "$TMP_DIR" -type f -name "$TARGET_EXEC" -exec mv {} "$TMP_FILE" \;
  elif download_file "$ZIP_URL" "${TMP_DIR}/sage.zip" 2>/dev/null && command -v unzip >/dev/null 2>&1; then
    unzip -q -o "${TMP_DIR}/sage.zip" -d "$TMP_DIR"
    find "$TMP_DIR" -type f -name "$TARGET_EXEC" -exec mv {} "$TMP_FILE" \;
  else
    error "Failed to download Sage binary from ${DOWNLOAD_URL}. Please ensure release assets exist for your platform."
  fi
fi

if [ ! -s "$TMP_FILE" ]; then
  error "Downloaded binary file is empty."
fi

chmod +x "$TMP_FILE"

# Determine target installation directory
if [ -n "${BINDIR:-}" ]; then
  INSTALL_DIR="$BINDIR"
elif [ -n "${INSTALL_DIR:-}" ]; then
  INSTALL_DIR="$INSTALL_DIR"
elif [ "$(id -u)" -eq 0 ]; then
  INSTALL_DIR="/usr/local/bin"
elif [ -w "/usr/local/bin" ]; then
  INSTALL_DIR="/usr/local/bin"
else
  INSTALL_DIR="${HOME}/.local/bin"
fi

mkdir -p "$INSTALL_DIR"
DEST_PATH="${INSTALL_DIR}/${TARGET_EXEC}"

# Move executable into place (handling permissions)
info "Installing to ${DEST_PATH}..."
if [ -w "$INSTALL_DIR" ]; then
  mv "$TMP_FILE" "$DEST_PATH"
else
  if command -v sudo >/dev/null 2>&1; then
    info "Elevated permissions required for ${INSTALL_DIR}. Using sudo..."
    sudo mv "$TMP_FILE" "$DEST_PATH"
  else
    error "Cannot write to ${INSTALL_DIR} and sudo is unavailable. Set BINDIR to a writable directory (e.g. BINDIR=~/.local/bin)."
  fi
fi

chmod +x "$DEST_PATH"

success "Sage binary successfully installed to ${DEST_PATH}!"

# Install Skills for AI Agents & Editors (.agents, .claude, .cursor)
info "Installing Agent Skill..."

install_skill_files() {
  target_dir="$1"
  mkdir -p "${target_dir}/references" "${target_dir}/scripts"

  if [ -n "$REPO_ROOT" ] && [ -d "${REPO_ROOT}/skills/sage" ]; then
    cp -R "${REPO_ROOT}/skills/sage/"* "${target_dir}/"
  else
    SKILL_BASE="https://raw.githubusercontent.com/${REPO}/main/skills/sage"
    download_file "${SKILL_BASE}/SKILL.md" "${target_dir}/SKILL.md" || true
    download_file "${SKILL_BASE}/references/language-syntax.md" "${target_dir}/references/language-syntax.md" || true
    download_file "${SKILL_BASE}/scripts/ensure-sage.sh" "${target_dir}/scripts/ensure-sage.sh" || true
  fi

  if [ -f "${target_dir}/scripts/ensure-sage.sh" ]; then
    chmod +x "${target_dir}/scripts/ensure-sage.sh"
  fi
}

REPO_ROOT="$(cd "$(dirname "$0")/.." 2>/dev/null && pwd || echo "")"

# 1. Universal Agent Skills standard (~/.agents/skills/sage)
install_skill_files "${HOME}/.agents/skills/sage"
# 2. Claude Code global (~/.claude/skills/sage)
install_skill_files "${HOME}/.claude/skills/sage"
# 3. Cursor AI global (~/.cursor/skills/sage)
install_skill_files "${HOME}/.cursor/skills/sage"

# 4. Project-level directory if run in a project
if [ -d ".git" ]; then
  install_skill_files "./.agents/skills/sage"
fi

success "Sage Skill successfully installed for all supported editors!"
info "Installed to: ~/.agents/skills/sage, ~/.claude/skills/sage, ~/.cursor/skills/sage"

# Verify installation & PATH check
case ":${PATH}:" in
  *:"${INSTALL_DIR}":*)
    ;;
  *)
    echo ""
    warn "${INSTALL_DIR} is not in your current PATH environment variable."
    printf "To run 'sage' from anywhere, add the following line to your shell configuration file:\n\n"
    if [ -n "${ZSH_VERSION:-}" ] || [ -f "${HOME}/.zshrc" ]; then
      printf "  ${BOLD}export PATH=\"%s:\$PATH\"${NC}  # add to ~/.zshrc\n\n" "$INSTALL_DIR"
    else
      printf "  ${BOLD}export PATH=\"%s:\$PATH\"${NC}  # add to ~/.bashrc\n\n" "$INSTALL_DIR"
    fi
    ;;
esac

echo ""
printf "${BOLD}Quick Start:${NC}\n"
printf "  sage init         # Create starter SAGE.md architecture diagram\n"
printf "  sage ui           # Launch the visualizer web interface\n"
printf "  sage --help       # View CLI options\n\n"


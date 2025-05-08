#!/bin/bash
#
# This script downloads and installs the latest version of ithena-cli.
# It attempts to install to /usr/local/bin, and will use sudo if necessary.
#
# Usage:
#   curl -sfL https://raw.githubusercontent.com/ithena-one/ithena-cli/main/install.sh | bash
#   or
#   curl -sfL https://github.com/ithena-one/ithena-cli/releases/latest/download/install.sh | bash
#   (if you attach this script as a release asset)

set -e # Exit immediately if a command exits with a non-zero status.
set -u # Treat unset variables as an error.
set -o pipefail # The return value of a pipeline is the status of the last command to exit with a non-zero status.

# --- Configuration ---
GITHUB_OWNER="ithena-one"
GITHUB_REPO="ithena-cli"
BINARY_NAME="ithena-cli"
INSTALL_DIR="/usr/local/bin"
TMP_DIR="" # Will be set by mktemp

# --- Helper Functions ---
echo_info() {
  echo "[INFO] $1"
}

echo_error() {
  echo "[ERROR] $1" >&2
  exit 1
}

cleanup() {
  if [ -n "$TMP_DIR" ] && [ -d "$TMP_DIR" ]; then
    echo_info "Cleaning up temporary directory: $TMP_DIR"
    rm -rf "$TMP_DIR"
  fi
}

# Register cleanup function to be called on script exit
trap cleanup EXIT

# --- Main Script ---

# 1. Determine OS and Architecture
OS_KERNEL=$(uname -s)
OS_ARCH=$(uname -m)

TARGET_OS=""
TARGET_ARCH=""

case "$OS_KERNEL" in
  Linux)
    TARGET_OS="linux"
    ;;
  Darwin)
    TARGET_OS="darwin"
    ;;
  *)
    echo_error "Unsupported operating system: $OS_KERNEL. Only Linux and macOS (Darwin) are supported."
    ;;
esac

case "$OS_ARCH" in
  x86_64)
    TARGET_ARCH="amd64"
    ;;
  amd64) # Some systems might report amd64 directly
    TARGET_ARCH="amd64"
    ;;
  arm64)
    TARGET_ARCH="arm64"
    ;;
  aarch64) # Common for Linux ARM64
    TARGET_ARCH="arm64"
    ;;
  *)
    echo_error "Unsupported architecture: $OS_ARCH. Only amd64 (x86_64) and arm64 (aarch64) are supported."
    ;;
esac

echo_info "Detected OS: $TARGET_OS, Architecture: $TARGET_ARCH"

# 2. Get the latest release tag from GitHub API
echo_info "Fetching the latest release tag from GitHub..."
LATEST_TAG_JSON=$(curl -sSfL "https://api.github.com/repos/${GITHUB_OWNER}/${GITHUB_REPO}/releases/latest")
if [ -z "$LATEST_TAG_JSON" ]; then
  echo_error "Could not fetch release information from GitHub."
fi

LATEST_TAG=$(echo "$LATEST_TAG_JSON" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
if [ -z "$LATEST_TAG" ]; then
  echo_error "Could not parse the latest release tag from GitHub API response."
fi
echo_info "Latest version is: $LATEST_TAG"

# 3. Construct the download URL
# GoReleaser archive name template: {{ .ProjectName }}_{{ .Os }}_{{ .Arch }}.tar.gz
# (The {{if .Arm}}v{{.Arm}}{{end}} part is not relevant for amd64/arm64)
ARCHIVE_NAME="${GITHUB_REPO}_${TARGET_OS}_${TARGET_ARCH}.tar.gz"
DOWNLOAD_URL="https://github.com/${GITHUB_OWNER}/${GITHUB_REPO}/releases/download/${LATEST_TAG}/${ARCHIVE_NAME}"

# 4. Download the tar.gz
echo_info "Downloading $BINARY_NAME from $DOWNLOAD_URL"
TMP_DIR=$(mktemp -d) # Create a temporary directory
curl -sfL "$DOWNLOAD_URL" -o "$TMP_DIR/$ARCHIVE_NAME"
echo_info "Download complete."

# 5. Extract the binary
echo_info "Extracting $BINARY_NAME from $ARCHIVE_NAME..."
# Extract only the binary into the temp directory. GoReleaser archives usually contain the binary at the root.
tar -xzf "$TMP_DIR/$ARCHIVE_NAME" -C "$TMP_DIR" "$BINARY_NAME"
if [ ! -f "$TMP_DIR/$BINARY_NAME" ]; then
    echo_error "Failed to extract $BINARY_NAME from the archive."
fi
echo_info "Extraction complete."

# 6. Make it executable
chmod +x "$TMP_DIR/$BINARY_NAME"
echo_info "$BINARY_NAME made executable."

# 7. Move it to INSTALL_DIR
echo_info "Attempting to install $BINARY_NAME to $INSTALL_DIR/$BINARY_NAME..."
if [ -w "$INSTALL_DIR" ]; then
  mv "$TMP_DIR/$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
  echo_info "$BINARY_NAME installed successfully to $INSTALL_DIR/$BINARY_NAME."
else
  echo_info "$INSTALL_DIR is not writable. Attempting with sudo..."
  if sudo mv "$TMP_DIR/$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"; then
    echo_info "$BINARY_NAME installed successfully to $INSTALL_DIR/$BINARY_NAME with sudo."
  else
    echo_error "Failed to install $BINARY_NAME. Please try running the script with sudo, or install manually."
  fi
fi

# Cleanup is handled by the trap
echo_info "Installation complete! You may need to open a new terminal or run 'source ~/.bashrc' (or equivalent for your shell) for the command to be available."
echo_info "Verify by running: $BINARY_NAME --version" 
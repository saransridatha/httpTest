#!/bin/bash

# --- Configuration ---
TOOL_NAME="httptest"
REPO_PATH="github.com/saransridatha/httptest"

# --- Colors for better output ---
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# --- Functions ---
log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

check_go_installed() {
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed. Please install Go first: https://go.dev/doc/install"
        exit 1
    fi
    log_info "Go is installed."
}

install_tool() {
    log_info "Installing ${TOOL_NAME}..."
    if go install "${REPO_PATH}@latest"; then
        log_info "${TOOL_NAME} installed successfully!"
    else
        log_error "Failed to install ${TOOL_NAME}. Please check the error messages above."
        exit 1
    fi
}

configure_path() {
    local go_bin_path="$(go env GOPATH)/bin"
    local shell_rc_file=""

    # Determine shell RC file
    if [[ "$SHELL" == *"bash"* ]]; then
        shell_rc_file="$HOME/.bashrc"
    elif [[ "$SHELL" == *"zsh"* ]]; then
        shell_rc_file="$HOME/.zshrc"
    else
        log_warn "Unsupported shell ($SHELL). Please manually add $go_bin_path to your PATH."
        return
    fi

    local export_line="export PATH=\"$PATH:$go_bin_path\""

    if grep -qF "$export_line" "$shell_rc_file"; then
        log_info "PATH already configured in $shell_rc_file."
    else
        log_info "Adding Go binary path to $shell_rc_file..."
        echo "" >> "$shell_rc_file"
        echo "# Add Go binaries to PATH for ${TOOL_NAME}" >> "$shell_rc_file"
        echo "$export_line" >> "$shell_rc_file"
        log_info "PATH configured in $shell_rc_file."
    fi
}

# --- Main execution ---
log_info "Starting ${TOOL_NAME} installation..."

check_go_installed
install_tool
configure_path

log_info "Installation script finished."
log_info "To use ${TOOL_NAME}, please run: ${YELLOW}source ~/.bashrc${NC} (or ~/.zshrc) or open a new terminal."
log_info "Then you can run: ${YELLOW}${TOOL_NAME} -h${NC}"

# --- Cleanup ---
# This script is designed to be run from the cloned repository.
# We remove the repository after successful installation.
log_info "Cleaning up installation files..."
cd .. # Go up one directory from the cloned repo
rm -rf "$(basename "$PWD")" # Remove the cloned repo directory
log_info "Cleanup complete. Repository removed."
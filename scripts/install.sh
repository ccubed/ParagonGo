#!/usr/bin/env sh
# GoMud installer for Linux and macOS.
#
# Usage (one-liner):
#   curl -fsSL https://raw.githubusercontent.com/GoMudEngine/GoMud/master/scripts/install.sh | sh
#
# Environment variables:
#   GOMUD_DIR   Override the install directory (default: $HOME/GoMud)

set -eu

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------

GOMUD_REPO="https://github.com/GoMudEngine/GoMud.git"
GOMUD_DIR="${GOMUD_DIR:-$HOME/GoMud}"

# Minimum Go version required (must match go.mod)
MIN_GO_MAJOR=1
MIN_GO_MINOR=24

GO_INSTALL_DIR="/usr/local/go"
GO_DL_API="https://go.dev/dl/?mode=json"

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

info()  { printf '==> %s\n' "$*"; }
warn()  { printf 'warning: %s\n' "$*" >&2; }
fatal() { printf 'error: %s\n' "$*" >&2; exit 1; }

# Run a command, prepending sudo only when not already root.
maybe_sudo() {
    if [ "$(id -u)" -eq 0 ]; then
        "$@"
    else
        sudo "$@"
    fi
}

# Download a URL to a local file. Prefers curl, falls back to wget.
download() {
    url=$1
    dest=$2
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL "$url" -o "$dest"
    elif command -v wget >/dev/null 2>&1; then
        wget -q "$url" -O "$dest"
    else
        fatal "Neither curl nor wget found. Install one and re-run."
    fi
}

# Fetch URL to stdout.
fetch() {
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL "$1"
    elif command -v wget >/dev/null 2>&1; then
        wget -q "$1" -O -
    else
        fatal "Neither curl nor wget found. Install one and re-run."
    fi
}

# Verify a file against a known SHA256 hex digest.
verify_sha256() {
    file=$1
    expected=$2
    if command -v sha256sum >/dev/null 2>&1; then
        actual=$(sha256sum "$file" | awk '{print $1}')
    elif command -v shasum >/dev/null 2>&1; then
        actual=$(shasum -a 256 "$file" | awk '{print $1}')
    else
        warn "No sha256sum or shasum found; skipping checksum verification."
        return 0
    fi
    if [ "$actual" != "$expected" ]; then
        fatal "SHA256 mismatch for $file. Expected $expected, got $actual."
    fi
}

# ---------------------------------------------------------------------------
# OS / arch detection
# ---------------------------------------------------------------------------

detect_os_arch() {
    raw_os=$(uname -s)
    raw_arch=$(uname -m)

    case "$raw_os" in
    Linux)  GO_OS="linux" ;;
    Darwin) GO_OS="darwin" ;;
    *)
        fatal "Unsupported operating system: $raw_os.
For Windows, use the PowerShell installer:
  irm https://raw.githubusercontent.com/GoMudEngine/GoMud/master/scripts/install.ps1 | iex"
        ;;
    esac

    case "$raw_arch" in
    x86_64)          GO_ARCH="amd64" ;;
    aarch64 | arm64) GO_ARCH="arm64" ;;
    armv6l)          GO_ARCH="armv6l" ;;
    i386 | i686)     GO_ARCH="386" ;;
    *)               fatal "Unsupported architecture: $raw_arch." ;;
    esac
}

# ---------------------------------------------------------------------------
# Version comparison
# ---------------------------------------------------------------------------

# Returns 0 if the installed Go version meets the minimum requirement.
go_version_ok() {
    if ! command -v go >/dev/null 2>&1; then
        return 1
    fi
    ver=$(go version 2>/dev/null | awk '{print $3}' | sed 's/go//')
    major=$(printf '%s' "$ver" | cut -d. -f1)
    minor=$(printf '%s' "$ver" | cut -d. -f2)
    if [ "$major" -gt "$MIN_GO_MAJOR" ]; then
        return 0
    fi
    if [ "$major" -eq "$MIN_GO_MAJOR" ] && [ "$minor" -ge "$MIN_GO_MINOR" ]; then
        return 0
    fi
    return 1
}

# ---------------------------------------------------------------------------
# Git installation
# ---------------------------------------------------------------------------

install_git() {
    info "git not found. Attempting to install..."

    if command -v apt-get >/dev/null 2>&1; then
        maybe_sudo apt-get update -qq
        maybe_sudo apt-get install -y git
    elif command -v dnf >/dev/null 2>&1; then
        maybe_sudo dnf install -y git
    elif command -v yum >/dev/null 2>&1; then
        maybe_sudo yum install -y git
    elif command -v brew >/dev/null 2>&1; then
        brew install git
    else
        fatal "Cannot install git automatically on this system.
Please install git manually:
  https://git-scm.com/downloads
Then re-run this installer."
    fi
}

# ---------------------------------------------------------------------------
# Go installation
# ---------------------------------------------------------------------------

install_go() {
    info "Fetching latest stable Go release information..."

    json=$(fetch "$GO_DL_API")

    # Extract the first stable version string from the JSON array.
    go_version=$(printf '%s' "$json" | grep -o '"version":"go[^"]*"' | head -1 | sed 's/"version":"//;s/"//')
    if [ -z "$go_version" ]; then
        fatal "Could not determine the latest Go version from $GO_DL_API."
    fi

    # Build the expected filename for this OS/arch combination.
    archive_name="${go_version}.${GO_OS}-${GO_ARCH}.tar.gz"

    # Extract the SHA256 for that specific filename from the JSON.
    sha256=$(printf '%s' "$json" | tr '{}' '\n' | grep "\"filename\":\"${archive_name}\"" | grep -o '"sha256":"[^"]*"' | sed 's/"sha256":"//;s/"//')

    if [ -z "$sha256" ]; then
        fatal "Could not find SHA256 for $archive_name in the Go download API response."
    fi

    dl_url="https://dl.google.com/go/${archive_name}"

    tmp_dir=$(mktemp -d)
    trap 'rm -rf "$tmp_dir"' EXIT

    archive_path="${tmp_dir}/${archive_name}"

    info "Downloading $go_version ($GO_OS/$GO_ARCH)..."
    download "$dl_url" "$archive_path"

    info "Verifying checksum..."
    verify_sha256 "$archive_path" "$sha256"

    info "Installing Go to $GO_INSTALL_DIR (may require sudo)..."
    maybe_sudo rm -rf "$GO_INSTALL_DIR"
    maybe_sudo tar -C /usr/local -xzf "$archive_path"

    # Persist PATH update to ~/.profile if not already present.
    profile_line='export PATH=$PATH:/usr/local/go/bin'
    if [ -f "$HOME/.profile" ] && grep -qF '/usr/local/go/bin' "$HOME/.profile"; then
        : # already present
    else
        printf '\n%s\n' "$profile_line" >> "$HOME/.profile"
        info "Added /usr/local/go/bin to PATH in ~/.profile."
    fi

    # Make Go available in the current session.
    export PATH="$PATH:/usr/local/go/bin"

    info "Go $go_version installed."
}

# ---------------------------------------------------------------------------
# GoMud clone / update
# ---------------------------------------------------------------------------

setup_gomud_repo() {
    if [ -d "$GOMUD_DIR/.git" ]; then
        info "GoMud directory already exists at $GOMUD_DIR. Pulling latest changes..."
        git -C "$GOMUD_DIR" pull
    elif [ -d "$GOMUD_DIR" ]; then
        fatal "$GOMUD_DIR exists but is not a git repository.
Remove or rename it, or set GOMUD_DIR to a different path, and re-run."
    else
        info "Cloning GoMud into $GOMUD_DIR..."
        git clone "$GOMUD_REPO" "$GOMUD_DIR"
    fi
}

# ---------------------------------------------------------------------------
# Build
# ---------------------------------------------------------------------------

build_gomud() {
    info "Building GoMud..."
    cd "$GOMUD_DIR"
    go generate
    CGO_ENABLED=0 go build -trimpath -a -o go-mud-server
    info "Build complete: $GOMUD_DIR/go-mud-server"
}

# ---------------------------------------------------------------------------
# Next steps
# ---------------------------------------------------------------------------

print_next_steps() {
    printf '\n'
    printf '================================================================\n'
    printf ' GoMud is ready!\n'
    printf '================================================================\n'
    printf '\n'
    printf 'Before starting for the first time, set an admin password:\n'
    printf '\n'
    printf '  cd %s\n' "$GOMUD_DIR"
    printf '  go run ./cmd/reset-admin-pw\n'
    printf '\n'
    printf 'Start the server:\n'
    printf '\n'
    printf '  cd %s\n' "$GOMUD_DIR"
    printf '  ./go-mud-server\n'
    printf '\n'
    printf 'Connect:\n'
    printf '  Web client : http://localhost/webclient\n'
    printf '  Web admin  : http://localhost/admin/\n'
    printf '  Telnet     : localhost:33333\n'
    printf '\n'
    printf 'For the full developer workflow (make run, make test, etc.):\n'
    printf '  https://github.com/GoMudEngine/GoMud#build-commands\n'
    printf '\n'
    if ! printf '%s' "$PATH" | grep -q '/usr/local/go/bin'; then
        printf 'NOTE: /usr/local/go/bin was added to ~/.profile.\n'
        printf 'Run the following to make "go" available in your current shell:\n'
        printf '\n'
        printf '  source ~/.profile\n'
        printf '\n'
    fi
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

detect_os_arch

if ! command -v git >/dev/null 2>&1; then
    install_git
fi

if go_version_ok; then
    info "Go $(go version | awk '{print $3}') already installed and meets the minimum requirement."
else
    install_go
fi

setup_gomud_repo
build_gomud
print_next_steps

#!/usr/bin/env sh
set -e

REPO="BobbyNooby/agamoto"
BINARY="agamoto"

FORCE=false
for arg in "$@"; do
    case "$arg" in
        --force|-f) FORCE=true ;;
    esac
done

echo "[agamoto] Installing..."

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
    linux*) OS="linux" ;;
    darwin*) OS="darwin" ;;
    msys*|cygwin*|mingw*|nt|win*)
        echo "[agamoto] Windows is not supported by this installer."
        echo "[agamoto] Install Go and run: go install github.com/${REPO}/cmd/${BINARY}@latest"
        exit 1
        ;;
    *) echo "[agamoto] Unsupported OS: $OS"; exit 1 ;;
esac

# Detect arch
ARCH=$(uname -m)
case "$ARCH" in
    x86_64|amd64) ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *) echo "[agamoto] Unsupported arch: $ARCH"; exit 1 ;;
esac

echo "[agamoto] Detected $OS/$ARCH"

# Check if already installed
EXISTING=$(command -v agamoto || true)
if [ -n "$EXISTING" ] && [ "$FORCE" != "true" ]; then
    echo "[agamoto] Already installed at $EXISTING"
    echo "[agamoto] Version: $(agamoto --version)"
    echo "[agamoto] To update: go install github.com/${REPO}/cmd/${BINARY}@latest"
    echo "[agamoto] To reinstall: curl -fsSL https://github.com/${REPO}/raw/main/install.sh | sh -s -- --force"
    exit 0
fi

# Check nmap
if ! command -v nmap >/dev/null 2>&1; then
    echo "[agamoto] nmap not found. Installing..."
    case "$OS" in
        darwin)
            if command -v brew >/dev/null 2>&1; then
                brew install nmap
            else
                echo "[agamoto] Homebrew not found. Please install nmap manually: https://nmap.org/download.html"
                exit 1
            fi
            ;;
        linux)
            if command -v apt >/dev/null 2>&1; then
                sudo apt-get update
                sudo apt-get install -y nmap
            elif command -v dnf >/dev/null 2>&1; then
                sudo dnf install -y nmap
            elif command -v pacman >/dev/null 2>&1; then
                sudo pacman -S --noconfirm nmap
            else
                echo "[agamoto] No supported package manager found. Please install nmap manually."
                exit 1
            fi
            ;;
    esac
else
    echo "[agamoto] nmap found"
fi

# Check Go
if ! command -v go >/dev/null 2>&1; then
    echo "[agamoto] Go not found. Install from https://go.dev/dl/"
    exit 1
fi

# Install agamoto
echo "[agamoto] Installing agamoto via go install..."
go install "github.com/${REPO}/cmd/${BINARY}@latest"

# Verify + PATH help
INSTALL_DIR="$(go env GOPATH)/bin"
if command -v agamoto >/dev/null 2>&1; then
    echo "[agamoto] Installed: $(agamoto --version)"
else
    echo "[agamoto] Installed to $INSTALL_DIR but agamoto is not on PATH."
    echo "[agamoto] Add this to your shell config:"
    echo "  export PATH=\"\$PATH:$INSTALL_DIR\""
    echo "[agamoto] Then reload with: source ~/.zshrc  (or ~/.bashrc, etc.)"
fi

echo "[agamoto] To update later: go install github.com/${REPO}/cmd/${BINARY}@latest"
echo "[agamoto] Done. Run 'agamoto doctor' to verify."

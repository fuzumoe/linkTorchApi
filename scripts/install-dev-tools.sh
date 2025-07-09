#!/bin/bash

set -e  # Exit on any error

command_exists() {
    command -v "$1" >/dev/null 2>&1
}

get_latest_go_version() {
    curl -s https://golang.org/VERSION?m=text | head -1
}

# Install Go
install_go() {
    echo "Installing Go..."

    if command_exists go; then
        echo "Go is already installed: $(go version)"
        return 0
    fi

    # Get latest Go version
    GO_VERSION=$(get_latest_go_version)
    if [ -z "$GO_VERSION" ]; then
        GO_VERSION="go1.21.5"
        echo "Could not fetch latest Go version, using fallback: $GO_VERSION"
    fi

    # Determine architecture
    ARCH=$(uname -m)
    case $ARCH in
        x86_64) ARCH="amd64" ;;
        aarch64) ARCH="arm64" ;;
        *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
    esac

    # Download and install Go
    GO_TARBALL="${GO_VERSION}.linux-${ARCH}.tar.gz"
    GO_URL="https://golang.org/dl/${GO_TARBALL}"

    echo "Downloading Go ${GO_VERSION} for ${ARCH}..."
    cd /tmp
    wget -q --show-progress "$GO_URL"

    # Remove existing Go installation
    sudo rm -rf /usr/local/go

    # Extract and install
    echo "Installing Go to /usr/local/go..."
    sudo tar -C /usr/local -xzf "$GO_TARBALL"

    # Add Go to PATH if not already there
    if ! grep -q "/usr/local/go/bin" ~/.bashrc; then
        echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
        echo 'export GOPATH=$HOME/go' >> ~/.bashrc
        echo 'export PATH=$PATH:$GOPATH/bin' >> ~/.bashrc
    fi

    # Set PATH for current session
    export PATH=$PATH:/usr/local/go/bin
    export GOPATH=$HOME/go
    export PATH=$PATH:$GOPATH/bin

    echo "Go installed successfully: $(go version)"
}

# Install Make
install_make() {
    echo "Installing Make..."

    if command_exists make; then
        echo "Make is already installed"
        return 0
    fi

    sudo apt-get update -qq
    sudo apt-get install -y build-essential

    echo "Make installed successfully"
}

# Install pre-commit using curl/bash method
install_precommit() {
    echo "Installing pre-commit..."

    if command_exists pre-commit; then
        echo "pre-commit is already installed"
        return 0
    fi

    # Create local bin directory if it doesn't exist
    mkdir -p ~/.local/bin

    # Download and install pre-commit binary
    echo "Downloading pre-commit binary..."
    curl -Lo ~/.local/bin/pre-commit https://github.com/pre-commit/pre-commit/releases/latest/download/pre-commit-$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m)

    # Make it executable
    chmod +x ~/.local/bin/pre-commit

    # Add user bin to PATH if not already there
    if ! grep -q "$HOME/.local/bin" ~/.bashrc; then
        echo 'export PATH=$PATH:$HOME/.local/bin' >> ~/.bashrc
    fi

    export PATH=$PATH:$HOME/.local/bin

    echo "pre-commit installed successfully"
}

# Install Go linting tools
install_go_linters() {
    echo "Installing Go linting tools..."

    # Ensure GOPATH is set
    if [ -z "$GOPATH" ]; then
        export GOPATH=$HOME/go
        export PATH=$PATH:$GOPATH/bin
    fi

    # Install golangci-lint
    if ! command_exists golangci-lint; then
        echo "Installing golangci-lint..."
        curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin
    else
        echo "golangci-lint is already installed"
    fi

    # Install other useful Go tools
    echo "Installing additional Go tools..."

    go install golang.org/x/tools/cmd/goimports@latest
    go install golang.org/x/tools/cmd/godoc@latest
    go install honnef.co/go/tools/cmd/staticcheck@latest
    go install github.com/kisielk/errcheck@latest
    go install mvdan.cc/gofumpt@latest

    echo "Go linting tools installed successfully"
}

# Main installation function
main() {
    echo "Starting development tools installation..."

    # Check if running as root
    if [ "$EUID" -eq 0 ]; then
        echo "Please don't run this script as root"
        exit 1
    fi

    # Install tools
    install_make
    install_go
    install_precommit
    install_go_linters

    echo "Development tools installation completed!"
    echo "Please run 'source ~/.bashrc' or restart your terminal to use the new tools"

    # Show installed versions
    echo ""
    echo "Installed tools:"
    echo "- Go: $(go version 2>/dev/null || echo 'Not found')"
    echo "- Make: $(make --version 2>/dev/null | head -1 || echo 'Not found')"
    echo "- pre-commit: $(pre-commit --version 2>/dev/null || echo 'Not found')"
    echo "- golangci-lint: $(golangci-lint --version 2>/dev/null || echo 'Not found')"
}

# Run main function
main "$@"

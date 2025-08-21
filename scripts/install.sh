#!/bin/bash
set -e

# CoachGPT Installation Script

REPO="BrianGreenhill/coachgpt"
INSTALL_DIR="/usr/local/bin"
BINARY_NAME="coachgpt"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case $ARCH in
    x86_64|amd64)
        ARCH="amd64"
        ;;
    aarch64|arm64)
        ARCH="arm64"
        ;;
    *)
        echo -e "${RED}Unsupported architecture: $ARCH${NC}"
        exit 1
        ;;
esac

case $OS in
    darwin)
        PLATFORM="darwin-$ARCH"
        ;;
    linux)
        PLATFORM="linux-$ARCH"
        ;;
    *)
        echo -e "${RED}Unsupported OS: $OS${NC}"
        echo "Please download manually from: https://github.com/$REPO/releases"
        exit 1
        ;;
esac

echo -e "${GREEN}CoachGPT Installer${NC}"
echo "OS: $OS"
echo "Architecture: $ARCH"
echo "Platform: $PLATFORM"
echo ""

# Get latest release
echo -e "${YELLOW}Fetching latest release...${NC}"
LATEST_RELEASE=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST_RELEASE" ]; then
    echo -e "${RED}Failed to fetch latest release${NC}"
    exit 1
fi

echo "Latest release: $LATEST_RELEASE"

# Download URL
DOWNLOAD_URL="https://github.com/$REPO/releases/download/$LATEST_RELEASE/$BINARY_NAME-$PLATFORM"

echo -e "${YELLOW}Downloading $BINARY_NAME...${NC}"
if command -v curl >/dev/null 2>&1; then
    curl -L "$DOWNLOAD_URL" -o "$BINARY_NAME"
elif command -v wget >/dev/null 2>&1; then
    wget "$DOWNLOAD_URL" -O "$BINARY_NAME"
else
    echo -e "${RED}Neither curl nor wget found. Please install one of them.${NC}"
    exit 1
fi

# Make executable
chmod +x "$BINARY_NAME"

# Install to system PATH
echo -e "${YELLOW}Installing to $INSTALL_DIR...${NC}"
if [ -w "$INSTALL_DIR" ]; then
    mv "$BINARY_NAME" "$INSTALL_DIR/"
else
    echo "Requesting sudo access to install to $INSTALL_DIR"
    sudo mv "$BINARY_NAME" "$INSTALL_DIR/"
fi

echo -e "${GREEN}✅ CoachGPT installed successfully!${NC}"
echo ""
echo "🏃 Next steps:"
echo "1. Run the interactive setup wizard:"
echo "   coachgpt config"
echo ""
echo "2. Or set up environment variables manually:"
echo "   export STRAVA_CLIENT_ID=\"your_client_id\""
echo "   export STRAVA_CLIENT_SECRET=\"your_client_secret\""
echo "   export STRAVA_HRMAX=\"185\""
echo ""
echo "3. Optionally, for Hevy integration:"
echo "   export HEVY_API_KEY=\"your_api_key\""
echo ""
echo "4. Run CoachGPT:"
echo "   coachgpt                # Latest Strava activity"
echo "   coachgpt --strength     # Latest Hevy workout"
echo "   coachgpt --help         # Show help"
echo ""
echo "📖 For setup instructions: https://github.com/$REPO#setup"

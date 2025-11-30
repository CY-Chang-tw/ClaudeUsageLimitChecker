#!/bin/bash

# Claude Usage Limit Checker - Linux Installation Script
# This script automates the installation and setup process

set -e  # Exit on error

echo "=================================="
echo "Claude Usage Limit Checker Setup"
echo "=================================="
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if running as root
if [ "$EUID" -eq 0 ]; then
    echo -e "${RED}Please do not run this script as root${NC}"
    exit 1
fi

# Get current directory and user
INSTALL_DIR=$(pwd)
CURRENT_USER=$(whoami)

echo -e "${GREEN}[1/7] Checking prerequisites...${NC}"

# Check for Go installation
if ! command -v go &> /dev/null; then
    echo -e "${YELLOW}Go is not installed. Installing Go...${NC}"

    # Detect OS
    if [ -f /etc/debian_version ]; then
        sudo apt update
        sudo apt install -y golang-go
    elif [ -f /etc/redhat-release ]; then
        sudo yum install -y golang
    else
        echo -e "${RED}Unsupported OS. Please install Go manually: https://golang.org/dl/${NC}"
        exit 1
    fi
else
    echo -e "${GREEN}✓ Go is already installed ($(go version))${NC}"
fi

echo ""
echo -e "${GREEN}[2/7] Installing Go dependencies...${NC}"
go mod download

echo ""
echo -e "${GREEN}[3/7] Building application...${NC}"
go build -o ClaudeUsageLimitChecker
chmod +x ClaudeUsageLimitChecker
echo -e "${GREEN}✓ Build successful${NC}"

echo ""
echo -e "${GREEN}[4/7] Setting up environment configuration...${NC}"
if [ ! -f .env ]; then
    cp .env.example .env
    echo -e "${YELLOW}⚠ Please edit .env file with your credentials:${NC}"
    echo "  - CLAUDE_SESSION_KEY"
    echo "  - CLAUDE_ORG_ID"
    echo "  - DISCORD_WEBHOOK_URL"
    echo ""
    read -p "Press Enter to edit .env now, or Ctrl+C to edit later..."
    ${EDITOR:-nano} .env
else
    echo -e "${GREEN}✓ .env file already exists${NC}"
fi

echo ""
echo -e "${GREEN}[5/7] Creating systemd service...${NC}"

# Create systemd service file
sudo tee /etc/systemd/system/claude-monitor.service > /dev/null <<EOF
[Unit]
Description=Claude Usage Limit Checker
After=network.target

[Service]
Type=simple
User=${CURRENT_USER}
WorkingDirectory=${INSTALL_DIR}
ExecStart=${INSTALL_DIR}/ClaudeUsageLimitChecker
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

echo -e "${GREEN}✓ Systemd service created${NC}"

echo ""
echo -e "${GREEN}[6/7] Enabling and starting service...${NC}"
sudo systemctl daemon-reload
sudo systemctl enable claude-monitor
sudo systemctl start claude-monitor

echo ""
echo -e "${GREEN}[7/7] Verifying installation...${NC}"
sleep 2

if sudo systemctl is-active --quiet claude-monitor; then
    echo -e "${GREEN}✓ Service is running!${NC}"
else
    echo -e "${RED}✗ Service failed to start${NC}"
    echo "Check logs with: sudo journalctl -u claude-monitor -n 50"
    exit 1
fi

echo ""
echo "=================================="
echo -e "${GREEN}Installation Complete!${NC}"
echo "=================================="
echo ""
echo "Useful commands:"
echo "  • Check status:    sudo systemctl status claude-monitor"
echo "  • View logs:       sudo journalctl -u claude-monitor -f"
echo "  • Restart service: sudo systemctl restart claude-monitor"
echo "  • Stop service:    sudo systemctl stop claude-monitor"
echo ""
echo "The monitor is now running in the background!"
echo "You should receive a test notification in Discord shortly."
echo ""

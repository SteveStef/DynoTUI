#!/bin/bash

# DynoTUI One-Shot Installation Script

set -e

APP_NAME="dynotui"
REPO_URL="https://github.com/stevestef/dynotui.git"
INSTALL_DIR="$HOME/.local/bin"
MODEL_ID="amazon.nova-lite-v1:0"

echo "----------------------------------------"
echo "  DynoTUI Installer"
echo "----------------------------------------"

# 1. Check for Go
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed. Please install Go 1.21 or later."
    echo "Visit: https://go.dev/doc/install"
    exit 1
fi

# 2. Handle Remote vs Local Execution
if [[ ! -d ".git" ]] || [[ "$(basename $(git rev-parse --show-toplevel 2>/dev/null))" != "$APP_NAME" ]]; then
    TEMP_DIR=$(mktemp -d)
    echo "Cloning repository to temporary directory: $TEMP_DIR"
    git clone --depth 1 "$REPO_URL" "$TEMP_DIR"
    cd "$TEMP_DIR"
    TRAP_CLEANUP="rm -rf $TEMP_DIR"
    trap "$TRAP_CLEANUP" EXIT
fi

# 3. Build the binary
echo "Building $APP_NAME..."
go build -o "$APP_NAME" .

# 4. Prepare Installation Directory
if [ ! -d "$INSTALL_DIR" ]; then
    echo "Creating directory: $INSTALL_DIR"
    mkdir -p "$INSTALL_DIR"
fi

# 5. Move binary
echo "Installing to $INSTALL_DIR/$APP_NAME..."
mv "$APP_NAME" "$INSTALL_DIR/"
chmod +x "$INSTALL_DIR/$APP_NAME"

# 6. Prerequisites Verification
echo ""
echo "--- Post-Installation Check ---"

# Check AWS Credentials
AWS_CREDS_FOUND=false
if [[ -n "$AWS_ACCESS_KEY_ID" && -n "$AWS_SECRET_ACCESS_KEY" ]]; then
    AWS_CREDS_FOUND=true
    echo "✅ Found AWS credentials in environment variables."
elif [[ -f "$HOME/.aws/credentials" ]]; then
    AWS_CREDS_FOUND=true
    echo "✅ Found AWS credentials file in ~/.aws/credentials"
else
    echo "⚠️  WARNING: No AWS credentials detected."
    echo "   Please run 'aws configure' or set AWS_ACCESS_KEY_ID/AWS_SECRET_ACCESS_KEY."
fi

# Bedrock Model Access Reminder
echo "ℹ️  BEDROCK ACCESS: Ensure your account has access to '$MODEL_ID'"
echo "   in the us-east-1 region via the AWS Console (Bedrock > Model Access)."

# 7. Final Instructions
echo ""
echo "----------------------------------------"
echo "Success! $APP_NAME has been installed."
echo ""

# Check PATH
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo "NOTE: $INSTALL_DIR is not in your PATH."
    echo "Add this to your .bashrc or .zshrc:"
    echo "  export PATH=\"\$PATH:$INSTALL_DIR\""
    echo ""
fi

echo "Run the application with: $APP_NAME"
echo "----------------------------------------"

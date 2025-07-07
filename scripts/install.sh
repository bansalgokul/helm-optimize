#!/usr/bin/env bash

set -e

# Default target architecture is amd64
TARGET_ARCH=${HELM_PLUGIN_TARGET_ARCH:-amd64}

# Detect the operating system
if [ "$(uname)" == "Darwin" ]; then
  OS=darwin
elif [ "$(expr substr $(uname -s) 1 5)" == "Linux" ]; then
  OS=linux
elif [ "$(expr substr $(uname -s) 1 10)" == "MINGW32_NT" ] || [ "$(expr substr $(uname -s) 1 10)" == "MINGW64_NT" ]; then
  OS=windows
else
  echo "Unsupported operating system"
  exit 1
fi

# Set binary name based on OS
if [ "$OS" == "windows" ]; then
  BINARY_NAME=helm-optimize.exe
else
  BINARY_NAME=helm-optimize
fi

# Create bin directory if it doesn't exist
mkdir -p $HELM_PLUGIN_DIR/bin

# Build the plugin from source if this is a development setup
if [ -f "$HELM_PLUGIN_DIR/go.mod" ]; then
  echo "Building helm-optimize from source..."
  cd $HELM_PLUGIN_DIR
  go build -o bin/$BINARY_NAME ./cmd/optimize
  chmod +x bin/$BINARY_NAME
  exit 0
fi

# In the future, we could add code here to download pre-built binaries
# For now, we're assuming a local development setup
echo "This plugin currently requires manual building from source."
echo "Run: go build -o bin/$BINARY_NAME main.go"
exit 1

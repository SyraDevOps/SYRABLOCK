#!/bin/bash

# Build script for SYRABLOCK Terminal Unificado V10
# This script compiles the unified terminal for multiple platforms

echo "🚀 SYRABLOCK V10 - Build Script"
echo "================================"
echo ""

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Create build directory
mkdir -p build

echo -e "${YELLOW}Building for Linux...${NC}"
go build -o build/syrablock_terminal cli_terminal.go
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ Linux build successful!${NC}"
else
    echo "❌ Linux build failed!"
    exit 1
fi

echo ""
echo -e "${YELLOW}Building for Windows...${NC}"
GOOS=windows GOARCH=amd64 go build -o build/syrablock_terminal.exe cli_terminal.go
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ Windows build successful!${NC}"
else
    echo "❌ Windows build failed!"
fi

echo ""
echo -e "${YELLOW}Building for macOS...${NC}"
GOOS=darwin GOARCH=amd64 go build -o build/syrablock_terminal_macos cli_terminal.go
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ macOS build successful!${NC}"
else
    echo "❌ macOS build failed!"
fi

echo ""
echo "================================"
echo -e "${GREEN}✅ Build complete!${NC}"
echo ""
echo "Executables created in ./build/ directory:"
ls -lh build/
echo ""
echo "To run:"
echo "  Linux:   ./build/syrablock_terminal"
echo "  Windows: ./build/syrablock_terminal.exe"
echo "  macOS:   ./build/syrablock_terminal_macos"

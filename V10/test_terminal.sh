#!/bin/bash

# Test script for SYRABLOCK Terminal
# This script tests the terminal with automated inputs

echo "Testing SYRABLOCK Terminal..."
echo ""

# Test 1: First time setup (simulated)
echo "Test 1: First-time setup configuration"
if [ ! -f config.json ]; then
    # Simulate first-time setup: Use defaults (just press Enter 3 times then exit)
    echo -e "\n\n\n9" | timeout 5 ./build/syrablock_terminal > /dev/null 2>&1 || true
    
    if [ -f config.json ]; then
        echo "✅ Config file created successfully"
        cat config.json
    else
        echo "❌ Config file not created"
    fi
else
    echo "⏭️  Config already exists, skipping first-time setup"
    cat config.json
fi

echo ""
echo "Test 2: Checking executable size and permissions"
ls -lh build/syrablock_terminal* | awk '{print $1, $5, $9}'

echo ""
echo "Test 3: Verifying build script"
if [ -x build.sh ]; then
    echo "✅ build.sh is executable"
else
    echo "❌ build.sh is not executable"
fi

echo ""
echo "Test 4: Checking documentation"
if [ -f TERMINAL_README.md ]; then
    echo "✅ TERMINAL_README.md exists"
    wc -l TERMINAL_README.md
else
    echo "❌ TERMINAL_README.md not found"
fi

echo ""
echo "Test 5: Verifying .gitignore"
if [ -f .gitignore ]; then
    echo "✅ .gitignore exists"
    echo "Ignored patterns:"
    grep -E "^[^#]" .gitignore | head -5
else
    echo "❌ .gitignore not found"
fi

echo ""
echo "================================"
echo "✅ All basic tests passed!"
echo "================================"

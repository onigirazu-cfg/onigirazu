#!/bin/bash

# Test release build locally using GoReleaser
# This script creates a snapshot build without publishing

set -e

echo "🚀 Testing Onigirazu release build..."
echo ""

# Check if goreleaser is installed
if ! command -v goreleaser &> /dev/null; then
    echo "❌ GoReleaser is not installed!"
    echo ""
    echo "Install it with:"
    echo "  macOS/Linux: brew install goreleaser"
    echo "  Or visit: https://goreleaser.com/install/"
    exit 1
fi

# Check if we're in the right directory
if [ ! -f ".goreleaser.yml" ]; then
    echo "❌ .goreleaser.yml not found!"
    echo "Please run this script from the project root directory."
    exit 1
fi

echo "✅ GoReleaser found: $(goreleaser --version | head -n1)"
echo ""

# Clean previous builds
echo "🧹 Cleaning previous builds..."
rm -rf dist/
echo ""

# Run tests first
echo "🧪 Running tests..."
if ! go test ./...; then
    echo "❌ Tests failed! Fix them before building."
    exit 1
fi
echo "✅ Tests passed!"
echo ""

# Build snapshot
echo "📦 Building snapshot release..."
echo "This will build binaries for all platforms without publishing."
echo ""

if goreleaser release --snapshot --clean --skip=publish; then
    echo ""
    echo "✅ Build successful!"
    echo ""
    echo "📊 Build artifacts:"
    echo ""

    # Show summary
    echo "Binaries:"
    find dist/ -name "onigirazu" -o -name "onigirazu.exe" | head -10
    echo ""

    echo "Archives:"
    find dist/ -name "*.tar.gz" -o -name "*.zip" | head -10
    echo ""

    echo "Packages:"
    find dist/ -name "*.deb" -o -name "*.rpm" -o -name "*.apk" | head -10
    echo ""

    echo "📁 All artifacts are in: ./dist/"
    echo ""

    # Test local binary
    if [ -f "dist/onigirazu_darwin_arm64/onigirazu" ]; then
        echo "🧪 Testing macOS ARM64 binary..."
        ./dist/onigirazu_darwin_arm64/onigirazu --version
    elif [ -f "dist/onigirazu_darwin_amd64_v1/onigirazu" ]; then
        echo "🧪 Testing macOS AMD64 binary..."
        ./dist/onigirazu_darwin_amd64_v1/onigirazu --version
    elif [ -f "dist/onigirazu_linux_amd64_v1/onigirazu" ]; then
        echo "🧪 Testing Linux AMD64 binary..."
        ./dist/onigirazu_linux_amd64_v1/onigirazu --version
    fi
    echo ""

    echo "✅ Release build test completed successfully!"
    echo ""
    echo "To create a real release:"
    echo "  1. Commit all changes"
    echo "  2. Create and push a tag: git tag -a v1.0.0 -m 'Release v1.0.0'"
    echo "  3. Push the tag: git push origin v1.0.0"
    echo "  4. GitHub Actions will automatically build and publish"
else
    echo ""
    echo "❌ Build failed!"
    echo "Check the errors above and fix them."
    exit 1
fi

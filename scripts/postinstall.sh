#!/bin/bash

# Post-installation script for Onigirazu (Linux packages)
# This script runs after package installation with root/sudo privileges

set -e

# Configuration directory
CONF_DIR="/etc/onigirazu"
CONF_FILE="$CONF_DIR/onigirazu.yml"
DEFAULT_CONF="/usr/share/onigirazu/onigirazu.default.yml"

# Create config if it doesn't exist
if [ ! -f "$CONF_FILE" ] && [ -f "$DEFAULT_CONF" ]; then
    echo "Creating default configuration in $CONF_DIR..."
    cp "$DEFAULT_CONF" "$CONF_FILE"
    chmod 644 "$CONF_FILE"
    echo "✓ Default configuration created at $CONF_FILE"
else
    if [ -f "$CONF_FILE" ]; then
        echo "✓ Configuration already exists at $CONF_FILE"
    else
        echo "⚠ Warning: Default configuration template not found at $DEFAULT_CONF"
    fi
fi

# Ensure proper permissions on config directory
chmod 755 "$CONF_DIR" 2>/dev/null || true

echo ""
echo "✓ Onigirazu has been installed successfully!"
echo ""
echo "📋 Available configuration templates:"
echo "  - /usr/share/onigirazu/onigirazu.default.yml    (all options)"
echo "  - /usr/share/onigirazu/examples/onigirazu.minimal.yml     (minimal setup)"
echo "  - /usr/share/onigirazu/examples/onigirazu.production.yml  (hardened)"
echo "  - /usr/share/onigirazu/examples/onigirazu.docker.yml      (containers)"
echo ""
echo "🚀 To get started:"
echo "  1. Verify installation: onigirazu --version"
echo "  2. View configuration: cat $CONF_FILE"
echo "  3. Run a test: onigirazu --help"
echo ""
echo "📖 Documentation: https://github.com/onigirazu-cfg/onigirazu/docs/"
echo "💬 Support: https://github.com/onigirazu-cfg/onigirazu/issues"

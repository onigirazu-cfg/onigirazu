#!/bin/bash

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SSH_DIR="$SCRIPT_DIR/ssh"

echo "Setting up Docker test environment..."

mkdir -p "$SSH_DIR"

if [ ! -f "$SSH_DIR/id_rsa" ]; then
    echo "Generating SSH key pair..."
    ssh-keygen -t rsa -b 2048 -f "$SSH_DIR/id_rsa" -N "" -C "onigirazu-test"
fi

if [ ! -f "$SSH_DIR/authorized_keys" ]; then
    echo "Creating authorized_keys..."
    cp "$SSH_DIR/id_rsa.pub" "$SSH_DIR/authorized_keys"
    chmod 600 "$SSH_DIR/authorized_keys"
fi

chmod 600 "$SSH_DIR/id_rsa"
chmod 644 "$SSH_DIR/id_rsa.pub"

echo "SSH keys ready at $SSH_DIR"
echo ""
echo "To start containers:"
echo "  docker-compose -f docker-compose.test.yml up -d"
echo ""
echo "To test:"
echo "  make docker-test ubuntu"

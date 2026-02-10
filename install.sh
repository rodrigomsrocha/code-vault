#!/bin/bash

set -e

echo "🗄️  Code Vault Installer"
echo "======================="
echo ""

# Check prerequisites
echo "📋 Checking prerequisites..."

if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go 1.21+ first."
    exit 1
fi

if ! command -v docker &> /dev/null; then
    echo "❌ Docker is not installed. Please install Docker first."
    exit 1
fi

if ! command -v docker-compose &> /dev/null; then
    echo "❌ Docker Compose is not installed. Please install Docker Compose first."
    exit 1
fi

echo "✅ All prerequisites met"
echo ""

# Build CLI
echo "🔨 Building CLI..."
go build -o vault ./cmd/vault

echo "✅ CLI built successfully"
echo ""

# Install binary
read -p "📦 Install to /usr/local/bin? (requires sudo) [y/N]: " install_global
if [[ $install_global =~ ^[Yy]$ ]]; then
    sudo mv vault /usr/local/bin/
    echo "✅ Installed to /usr/local/bin/vault"
else
    echo "ℹ️  Binary available at ./vault"
    echo "   Add to PATH or run with ./vault"
fi

echo ""

# Start infrastructure
read -p "🐳 Start infrastructure (PostgreSQL + SeaweedFS)? [y/N]: " start_infra
if [[ $start_infra =~ ^[Yy]$ ]]; then
    docker-compose -f composes/docker-compose.yml up -d
    echo "✅ Infrastructure started"
    echo ""
    echo "⏳ Waiting for services to be ready..."
    sleep 5
fi

echo ""

# Initialize
read -p "🎉 Initialize Code Vault? [y/N]: " init_vault
if [[ $init_vault =~ ^[Yy]$ ]]; then
    if command -v vault &> /dev/null; then
        vault init
    else
        ./vault init
    fi
fi

echo ""
echo "🎉 Installation complete!"
echo ""
echo "Next steps:"
echo "  1. Edit config: ~/.config/code-vault/config.yaml"
echo "  2. Create your first snippet: vault create <file>"
echo "  3. List snippets: vault list"
echo "  4. Interactive mode: vault get"
echo ""
echo "Documentation: https://github.com/yourusername/code-vault"

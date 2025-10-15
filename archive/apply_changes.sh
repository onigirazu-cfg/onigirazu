#!/bin/bash

# Script to apply variable management changes to onigirazu repository

set -e

SOURCE_DIR="/Users/denys.rastiegaiev/work/onigirazu_project/onigirazu"
TARGET_DIR="/Users/denys.rastiegaiev/work/onigirazu_clean"
REPO_URL="git@github.com:onigirazu-cfg/onigirazu.git"

echo "🍙 Onigirazu Variable Management Changes Application Script"
echo "=========================================================="
echo ""

# Step 1: Clone fresh repository
echo "📥 Step 1: Cloning fresh repository..."
if [ -d "$TARGET_DIR" ]; then
    echo "   Removing existing directory..."
    rm -rf "$TARGET_DIR"
fi

git clone "$REPO_URL" "$TARGET_DIR"
cd "$TARGET_DIR"

# Step 2: Create feature branch
echo ""
echo "🌿 Step 2: Creating feature branch..."
git checkout -b feature/variable-management

# Step 3: Copy modified files
echo ""
echo "📝 Step 3: Copying modified files..."

# Copy execution_engine.go
echo "   - Copying execution_engine.go..."
cp "$SOURCE_DIR/internal/engine/execution_engine.go" "$TARGET_DIR/internal/engine/execution_engine.go"

# Copy apply.go
echo "   - Copying apply.go..."
cp "$SOURCE_DIR/internal/cli/apply.go" "$TARGET_DIR/internal/cli/apply.go"

# Copy interfaces.go
echo "   - Copying interfaces.go..."
cp "$SOURCE_DIR/internal/interfaces/interfaces.go" "$TARGET_DIR/internal/interfaces/interfaces.go"

# Copy execution_engine_test.go
echo "   - Copying execution_engine_test.go..."
cp "$SOURCE_DIR/internal/engine/execution_engine_test.go" "$TARGET_DIR/internal/engine/execution_engine_test.go"

# Step 4: Copy new example files
echo ""
echo "📁 Step 4: Creating examples/variables directory..."
mkdir -p "$TARGET_DIR/examples/variables"

echo "   - Copying playbook.yml..."
cp "/Users/denys.rastiegaiev/work/onigirazu_project/examples/variables/playbook.yml" "$TARGET_DIR/examples/variables/playbook.yml"

echo "   - Copying inventory.yml..."
cp "/Users/denys.rastiegaiev/work/onigirazu_project/examples/variables/inventory.yml" "$TARGET_DIR/examples/variables/inventory.yml"

echo "   - Copying README.md..."
cp "/Users/denys.rastiegaiev/work/onigirazu_project/examples/variables/README.md" "$TARGET_DIR/examples/variables/README.md"

# Step 5: Copy documentation
echo ""
echo "📚 Step 5: Copying documentation..."
cp "/Users/denys.rastiegaiev/work/onigirazu_project/IMPLEMENTATION_SUMMARY.md" "$TARGET_DIR/IMPLEMENTATION_SUMMARY.md"

# Step 6: Run tests
echo ""
echo "🧪 Step 6: Running tests..."
cd "$TARGET_DIR"
go test ./internal/engine/... -v

# Step 7: Git add and commit
echo ""
echo "💾 Step 7: Committing changes..."
git add .
git commit -m "feat: Add command-line and environment variable support for playbook variables

- Implemented --extra-vars/-e flag for command-line variable passing
- Added automatic loading of ONIGIRAZU_VAR_* environment variables
- Established proper variable priority: playbook < env vars < extra vars
- Added comprehensive test suite (7 new tests, all passing)
- Created examples and documentation for variable usage
- Thread-safe variable management with mutex protection

Variable Priority Chain:
1. Playbook variables (lowest priority)
2. Environment variables (ONIGIRAZU_VAR_* prefix)
3. Command-line extra variables (highest priority)

Files Modified:
- internal/engine/execution_engine.go
- internal/cli/apply.go
- internal/interfaces/interfaces.go
- internal/engine/execution_engine_test.go

Files Created:
- examples/variables/playbook.yml
- examples/variables/inventory.yml
- examples/variables/README.md
- IMPLEMENTATION_SUMMARY.md"

# Step 8: Create tag
echo ""
echo "🏷️  Step 8: Creating release tag..."
git tag -a v1.30.0 -m "Release v1.30.0: Variable Management System

Features:
- Command-line variable passing via --extra-vars/-e flag
- Environment variable support with ONIGIRAZU_VAR_* prefix
- Proper variable priority hierarchy
- Thread-safe variable management
- Comprehensive test coverage
- Examples and documentation

This release establishes flexible variable management in Onigirazu,
following Ansible-like patterns for familiarity."

# Step 9: Push changes
echo ""
echo "🚀 Step 9: Pushing changes to GitHub..."
echo "   - Pushing branch..."
git push -u origin feature/variable-management

echo "   - Pushing tag..."
git push origin v1.30.0

echo ""
echo "✅ Done! Changes have been applied and pushed."
echo ""
echo "Next steps:"
echo "1. Go to https://github.com/onigirazu-cfg/onigirazu/pulls"
echo "2. Create a Pull Request from 'feature/variable-management' to 'main'"
echo "3. After PR is merged, create a GitHub Release for v1.30.0"
echo ""

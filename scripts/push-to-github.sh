#!/bin/bash
# Push Vajra to GitHub

set -e

echo "=== Pushing Vajra to GitHub ==="

# Navigate to project
cd "$(dirname "$0")/.." || exit 1

# Check if git repo exists
if [ ! -d .git ]; then
    echo "Initializing Git repository..."
    git init
    git config user.name "Vajra Bot"
    git config user.email "bot@vajra.local"
fi

# Add files
echo "Adding files..."
git add .

# Commit
echo "Committing..."
git add -A
git commit -m "Initial Vajra release with full installer pipeline" || echo "Nothing to commit"

# Add remote
REMOTE_URL="https://github.com/Ritiksuman07/Vajra.git"
git remote remove origin 2>/dev/null || true
git remote add origin "$REMOTE_URL"

# Rename branch to main
git branch -M main 2>/dev/null || true

# Push
echo "Pushing to GitHub..."
git push -u origin main --force

echo "✅ Done! Check https://github.com/Ritiksuman07/Vajra"
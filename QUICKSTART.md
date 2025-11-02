# Quick Start Guide

## 🚀 Get Started in 5 Minutes

### 1. Update Your GitHub Username

```bash
# Replace YOURUSERNAME with your actual GitHub username
export GH_USER="your-github-username"

sed -i '' "s/yourusername/$GH_USER/g" go.mod
sed -i '' "s/yourusername/$GH_USER/g" main.go
sed -i '' "s/yourusername/$GH_USER/g" .goreleaser.yml
sed -i '' "s/yourusername/$GH_USER/g" readme.md

# Update dependencies
go mod tidy

# Rebuild to test
make build
```

### 2. Create GitHub Repositories

Create two repositories on GitHub:
1. `vervideos` (main project) - https://github.com/new
2. `homebrew-tap` (formula repository) - https://github.com/new

Both should be **public**.

### 3. Setup GitHub Token

```bash
# Get token from: https://github.com/settings/tokens/new
# Scopes needed: repo, workflow

export GITHUB_TOKEN="your_token_here"
echo 'export GITHUB_TOKEN="your_token_here"' >> ~/.zshrc
```

### 4. Push to GitHub

```bash
git init
git add .
git commit -m "Initial commit: vervideos CLI tool"
git remote add origin https://github.com/$GH_USER/vervideos.git
git branch -M main
git push -u origin main
```

### 5. Create First Release

```bash
# Install GoReleaser if you haven't
brew install goreleaser

# Tag and release
git tag v0.1.0
git push origin v0.1.0
goreleaser release --clean
```

### 6. Test Installation

```bash
brew tap $GH_USER/tap
brew install vervideos
vervideos --help
```

## 🎉 Done!

Your CLI tool is now available via Homebrew!

Share it with:
```bash
brew tap YOUR-USERNAME/tap
brew install vervideos
```

## 📖 Need More Details?

See [SETUP_GUIDE.md](SETUP_GUIDE.md) for the complete step-by-step guide.


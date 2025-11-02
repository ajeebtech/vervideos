# Setup Guide for Homebrew Distribution

This guide will walk you through publishing `vervideos` to Homebrew using Option A (Custom Tap).

## ✅ What's Already Done

- ✓ Go project structure created
- ✓ CLI commands implemented (`init`, `save`, `list`, `restore`, `diff`, `version`)
- ✓ Build system configured (Makefile)
- ✓ GoReleaser configuration ready
- ✓ Binary builds and runs successfully

## 📋 Prerequisites

Before you begin, make sure you have:

1. **Go** installed (you already have this ✓)
2. **Git** installed and configured
3. **GitHub account** created
4. **GoReleaser** installed:
   ```bash
   brew install goreleaser
   ```

## 🚀 Step-by-Step Instructions

### Step 1: Create GitHub Repository for the Main Project

1. Go to https://github.com/new
2. Create a new repository named `vervideos`
3. Make it **public** (required for Homebrew)
4. Don't initialize with README (we already have one)

### Step 2: Update GitHub Username in Files

Replace `yourusername` with your actual GitHub username in these files:

**File: `go.mod`** - Update line 1:
```go
module github.com/YOURUSERNAME/vervideos
```

**File: `main.go`** - Update line 4:
```go
"github.com/YOURUSERNAME/vervideos/cmd"
```

**File: `.goreleaser.yml`** - Update lines 39-40:
```yaml
tap:
  owner: YOURUSERNAME
  name: homebrew-tap
```

And line 42:
```yaml
homepage: "https://github.com/YOURUSERNAME/vervideos"
```

You can do this with:
```bash
# Replace YOURUSERNAME with your actual GitHub username
sed -i '' 's/yourusername/YOURUSERNAME/g' go.mod
sed -i '' 's/yourusername/YOURUSERNAME/g' main.go
sed -i '' 's/yourusername/YOURUSERNAME/g' .goreleaser.yml
sed -i '' 's/yourusername/YOURUSERNAME/g' readme.md
```

### Step 3: Initialize Git and Push to GitHub

```bash
# Initialize git (if not already done)
git init

# Add all files
git add .

# Commit
git commit -m "Initial commit: vervideos CLI tool"

# Add your GitHub remote (replace YOURUSERNAME)
git remote add origin https://github.com/YOURUSERNAME/vervideos.git

# Rename branch to main if needed
git branch -M main

# Push to GitHub
git push -u origin main
```

### Step 4: Create GitHub Personal Access Token

GoReleaser needs permission to create releases and push to your homebrew-tap repo.

1. Go to: https://github.com/settings/tokens/new
2. Give it a name: "GoReleaser"
3. Set expiration (e.g., "No expiration" or 1 year)
4. Select scopes:
   - ✓ `repo` (all)
   - ✓ `workflow`
5. Click "Generate token"
6. **Copy the token** (you won't see it again!)
7. Save it as an environment variable:
   ```bash
   export GITHUB_TOKEN="your_token_here"
   
   # Add to your ~/.zshrc to make it permanent
   echo 'export GITHUB_TOKEN="your_token_here"' >> ~/.zshrc
   ```

### Step 5: Create the Homebrew Tap Repository

1. Go to https://github.com/new
2. Create a new repository named `homebrew-tap`
3. Make it **public**
4. Initialize with a README
5. That's it! GoReleaser will automatically create the formula here

### Step 6: Create Your First Release

```bash
# Make sure all changes are committed
git add .
git commit -m "Ready for first release"
git push

# Create and push a version tag
git tag v0.1.0
git push origin v0.1.0

# Run GoReleaser (this will create the release and update homebrew-tap)
goreleaser release --clean
```

This will:
- Build binaries for macOS, Linux, and Windows (both amd64 and arm64)
- Create a GitHub release with all the binaries
- Generate checksums
- Create a Homebrew formula in your `homebrew-tap` repository
- Upload everything automatically

### Step 7: Test Your Homebrew Installation

Once the release is complete, test it:

```bash
# Install from your custom tap
brew tap YOURUSERNAME/tap
brew install vervideos

# Test it works
vervideos --help
vervideos version
```

## 🎉 Success!

Your tool is now available via Homebrew! Anyone can install it with:

```bash
brew tap YOURUSERNAME/tap
brew install vervideos
```

## 📦 Making Future Releases

Whenever you want to release a new version:

```bash
# Make your changes
git add .
git commit -m "Add new features"
git push

# Create a new version tag
git tag v0.2.0
git push origin v0.2.0

# Release
goreleaser release --clean
```

GoReleaser will automatically:
- Update the Homebrew formula
- Create a new GitHub release
- Build all the binaries

Users can update with:
```bash
brew upgrade vervideos
```

## 🔧 Useful Commands

### Test release locally (without publishing)
```bash
goreleaser release --snapshot --clean
```

### Check GoReleaser configuration
```bash
goreleaser check
```

### Build just for your platform
```bash
make build
```

### Install locally for testing
```bash
make install
```

## 📝 Important Notes

1. **Version tags must start with 'v'** (e.g., v0.1.0, v1.0.0)
2. **Follow semantic versioning**: MAJOR.MINOR.PATCH
3. **Keep your GITHUB_TOKEN secure** - never commit it to git
4. **The homebrew-tap repo should be public**
5. **Test thoroughly before each release**

## 🐛 Troubleshooting

### GoReleaser fails with authentication error
- Check your GITHUB_TOKEN is set: `echo $GITHUB_TOKEN`
- Make sure the token has `repo` and `workflow` scopes

### Homebrew tap not updating
- Check the homebrew-tap repository exists and is public
- Verify the username in `.goreleaser.yml` is correct

### Build fails
- Run `go mod tidy` to ensure dependencies are correct
- Run `goreleaser check` to validate configuration

## 🎯 Next Steps

1. **Add real functionality** - Currently the commands show placeholder output
2. **Add tests** - Create `*_test.go` files
3. **Add documentation** - Expand the README with more examples
4. **Add CI/CD** - Set up GitHub Actions for automated testing
5. **Consider submitting to Homebrew Core** - Once popular!

## 📚 Resources

- [GoReleaser Documentation](https://goreleaser.com/)
- [Homebrew Formula Cookbook](https://docs.brew.sh/Formula-Cookbook)
- [Semantic Versioning](https://semver.org/)
- [Cobra CLI Library](https://github.com/spf13/cobra)

---

Need help? Check the documentation links above or open an issue on GitHub!


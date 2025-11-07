# How to Create a GitHub Release

## Automated Release (Recommended)

1. **Commit all your changes:**
   ```bash
   git add .
   git commit -m "Your commit message"
   git push origin main
   ```

2. **Create and push a new tag:**
   ```bash
   # For a patch release (0.1.0 -> 0.1.1)
   git tag v0.1.1
   git push origin v0.1.1
   
   # For a minor release (0.1.0 -> 0.2.0)
   git tag v0.2.0
   git push origin v0.2.0
   
   # For a major release (0.1.0 -> 1.0.0)
   git tag v1.0.0
   git push origin v1.0.0
   ```

3. **GitHub Actions will automatically:**
   - Build binaries for all platforms (Linux, macOS, Windows)
   - Create a GitHub release with all artifacts
   - Generate checksums
   - Update Homebrew formula (if configured)

## Manual Release (Alternative)

If you prefer to create a release manually:

1. **Build locally with GoReleaser:**
   ```bash
   # Test the release (dry-run)
   goreleaser release --snapshot
   
   # Create actual release (requires GITHUB_TOKEN)
   export GITHUB_TOKEN=your_token_here
   goreleaser release
   ```

2. **Or create a release on GitHub:**
   - Go to https://github.com/ajeebtech/vervideos/releases
   - Click "Draft a new release"
   - Choose a tag or create a new one
   - Add release notes
   - Upload binaries manually

## Version Numbering

Follow [Semantic Versioning](https://semver.org/):
- **MAJOR** version (1.0.0): Breaking changes
- **MINOR** version (0.1.0): New features, backwards compatible
- **PATCH** version (0.0.1): Bug fixes, backwards compatible

## Current Version

Check current version:
```bash
git tag --sort=-v:refname | head -1
```


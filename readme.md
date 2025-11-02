# vervideos

A versioning command-line tool for video editors.

## 📹 About

`vervideos` helps video editors manage different versions of their projects, making it easy to save, restore, and compare various cuts and edits.

## 🚀 Installation

### Via Homebrew (recommended)

```bash
brew tap yourusername/tap
brew install vervideos
```

### From Source

```bash
git clone https://github.com/yourusername/vervideos.git
cd vervideos
make build
make install
```

## 📖 Usage

### Initialize a project
```bash
vervideos init my-video-project
```

### Save a version
```bash
vervideos save "rough-cut-1"
```

### List all versions
```bash
vervideos list
```

### Restore a version
```bash
vervideos restore v1.0.0
```

### Compare versions
```bash
vervideos diff v1.0.0 v1.1.0
```

### Check version
```bash
vervideos version
```

## 🛠 Development

### Build
```bash
make build
```

### Install locally
```bash
make install
```

### Clean build artifacts
```bash
make clean
```

## 📝 Commands

- `init [project-name]` - Initialize a new video project with version control
- `save [version-name]` - Save a new version of your project
- `list` - List all versions
- `restore [version]` - Restore a specific version
- `diff [version1] [version2]` - Compare two versions
- `version` - Display the version information

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details.

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

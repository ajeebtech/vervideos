# vervids

A local version control system for Adobe After Effects projects with Docker storage.

## 📹 About

`vervids` is a command-line tool that provides version control for `.aepx` (Adobe After Effects XML) files and their assets. It automatically tracks your project file and all referenced assets (videos, images, audio files), storing them either locally or in Docker for efficient version management.

## 🚀 Installation

### From Source

```bash
git clone https://github.com/ajeebtech/vervideos.git
cd vervideos
make build
make install
```

### Download Binary

Download the latest release for your platform from the [releases page](https://github.com/ajeebtech/vervideos/releases).

## 📖 Usage

### Initialize a project
Start tracking an After Effects project:
```bash
vervids init "project.aepx"
```

Or use Docker for storage (recommended for large projects):
```bash
vervids init --docker "project.aepx"
```

This creates a `.vervids` directory and stores the initial version along with all referenced assets.

### Commit a new version
After making changes to your `.aepx` file or assets:
```bash
vervids commit "Added intro animation"
```

Each commit stores a complete snapshot of your project file and all assets with the message.

### Check version
```bash
vervids version
```

## 📂 How It Works

### Asset Tracking
1. **Python Parser**: Automatically parses your `.aepx` file (XML format)
2. **Asset Discovery**: Finds all referenced files (MP4, PNG, SVG, etc.)
3. **Versioning**: Stores both the project file and all assets for each version

### Storage Options

#### Local Storage (Default)
```bash
vervids init "project.aepx"
```
- Creates `.vervids` directory in your project folder
- Each version stored in `.vervids/versions/vXXX/`
- Assets stored in `.vervids/versions/vXXX/assets/`

#### Docker Storage (Recommended)
```bash
vervids init --docker "project.aepx"
```
- Creates a Docker container for efficient storage
- Uses Docker volumes for persistent data
- Ideal for large projects with many assets
- Easily backup entire project vault

### Directory Structure

**Local Storage:**
```
your-project/
├── project.aepx                 # Your working file
├── footage/
│   ├── video1.mp4
│   └── image1.png
└── .vervids/
    ├── config.json              # Project metadata & version history
    └── versions/
        ├── v000/
        │   ├── project.aepx     # Initial version
        │   └── assets/
        │       ├── video1.mp4
        │       └── image1.png
        └── v001/
            ├── project.aepx     # "Added intro"
            └── assets/
                ├── video1.mp4
                └── image1.png
```

**Docker Storage:**
All files stored in Docker volume `vervids-data` under `/storage/projects/`

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

### Run tests
```bash
make test
```

## 📝 Available Commands

- `init [path/to/project.aepx]` - Initialize version control for an .aepx file
  - `--docker, -d` - Use Docker for storage
- `commit [message]` - Save a new version with all assets and a commit message
- `version` - Display the version information

## 🐳 Docker Integration

### Prerequisites
```bash
docker --version  # Ensure Docker is installed
```

### Setup Docker Storage
```bash
# Initialize with Docker
vervids init --docker "project.aepx"

# Docker container is automatically created and managed
docker ps | grep vervids-storage
```

### Docker Commands
```bash
# View storage volume
docker volume inspect vervids-data

# View files in Docker
docker exec vervids-storage ls -la /storage/projects/

# Backup Docker volume
docker run --rm -v vervids-data:/data -v $(pwd):/backup alpine tar czf /backup/vervids-backup.tar.gz /data
```

## ✅ Features

- ✅ Initialize version control for `.aepx` files
- ✅ Automatic asset tracking (videos, images, audio, etc.)
- ✅ Local storage option
- ✅ Docker storage integration
- ✅ Python-based `.aepx` parser
- ✅ JSON metadata tracking
- ✅ Version history with messages and timestamps

## 🎯 Roadmap

- [ ] `list` - List all versions with timestamps and messages
- [ ] `restore [version]` - Restore a specific version with assets
- [ ] `diff [v1] [v2]` - Compare two versions
- [ ] Compression for large files
- [ ] Branch support for alternative edits
- [ ] Web UI for browsing versions
- [ ] Asset deduplication across versions

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details.

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

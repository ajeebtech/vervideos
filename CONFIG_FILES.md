# Vervids Config Files

This document explains the config files created by `vervids init` and how they're managed.

## File Structure

When you run `vervids init "path/to/file.aepx"`, the following files are created:

### 1. Per-Project Config File
**Location**: `<directory-containing-aepx>/.vervids/config.json`

This is the **main project configuration file** that contains:
- Project name
- Project path (original .aepx file path)
- Version history (all commits/versions)
- Docker volume information
- Asset tracking information

**Important Notes**:
- This file is created in the **same directory as the .aepx file**
- The `vervids init` command automatically changes to the .aepx file's directory before creating this file
- Each project has its own `.vervids/config.json` file
- This file is **NOT global** - it's per-project

### 2. Global Context File
**Location**: `~/.vervids/current_project.json`

This file stores the **currently active/selected project**:
- Project name
- Path to the project's config.json file

**Important Notes**:
- This is a **global file** (one per user)
- It tracks which project is currently selected
- When you run `vervids init`, this file is updated to point to the new project
- If you switch projects (via `vervids list`), this file is updated

## Directory Structure Example

```
/path/to/project/
├── myproject.aepx          # Original file (deleted after init)
└── .vervids/
    └── config.json         # Project config (created by init)

~/.vervids/
└── current_project.json   # Global context (created/updated by init)
```

## How `vervids init` Works

1. **Takes the .aepx file path** (can be relative or absolute)
2. **Converts to absolute path** for consistency
3. **Changes working directory** to the directory containing the .aepx file
4. **Creates `.vervids/` directory** in that location (if it doesn't exist)
5. **Creates `.vervids/config.json`** with project metadata
6. **Copies .aepx file and assets to Docker** storage
7. **Deletes the original .aepx file** from local filesystem
8. **Updates `~/.vervids/current_project.json`** to set this as the active project

## For Plugin Developers

If you're calling `vervids init` from a plugin:

1. **The config file location depends on where the .aepx file is located**
   - If you pass `/path/to/project/file.aepx`, the config will be at `/path/to/project/.vervids/config.json`
   - The init command automatically handles directory changes

2. **The working directory changes during init**
   - The command changes to the .aepx file's directory
   - It restores the original directory when done
   - This ensures the config is always created next to the project file

3. **To find the config file after init:**
   - Check the init command output - it now shows the config file path
   - Or look in the same directory as the .aepx file: `<aepx-dir>/.vervids/config.json`
   - Or check the global context: `~/.vervids/current_project.json` contains the path

4. **Multiple projects are supported**
   - Each project has its own `.vervids/config.json`
   - The global context file tracks which one is active
   - Projects don't overwrite each other's configs

## Troubleshooting

### "Could not find config file for project"
This usually means:
- The `.vervids/config.json` file doesn't exist in the expected location
- The project was initialized in a different directory
- The config file was deleted or moved

**Solution**: Re-run `vervids init` in the correct directory, or navigate to the project directory where `.vervids/config.json` exists.

### Config file in wrong location
If the config file is created in the wrong place:
- Make sure you're passing the full path to the .aepx file
- The init command uses the .aepx file's directory, not the current working directory
- Check the "Working directory" message in the init output

## Summary

- **Per-project config**: `.vervids/config.json` (in the .aepx file's directory)
- **Global context**: `~/.vervids/current_project.json` (in home directory)
- **No overwriting**: Each project has its own config file
- **Automatic directory handling**: Init command handles directory changes automatically


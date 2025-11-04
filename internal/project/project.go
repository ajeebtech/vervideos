package project

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ajeebtech/vervideos/internal/assets"
	"github.com/ajeebtech/vervideos/internal/docker"
	"github.com/ajeebtech/vervideos/internal/storage"
	"github.com/ajeebtech/vervideos/internal/tracking"
)

// AssetInfo represents an asset file tracked in a version
type AssetInfo struct {
	OriginalPath string `json:"original_path"`
	RelativePath string `json:"relative_path"`
	Filename     string `json:"filename"`
	Extension    string `json:"extension"`
	Size         int64  `json:"size"`
	DockerPath   string `json:"docker_path"`
}

// Version represents a single version/commit of the project
type Version struct {
	Number       int         `json:"number"`
	Message      string      `json:"message"`
	Timestamp    time.Time   `json:"timestamp"`
	Size         int64       `json:"size"`
	FilePath     string      `json:"file_path"`
	DockerPath   string      `json:"docker_path"`
	Assets       []AssetInfo `json:"assets"`
	AssetCount   int         `json:"asset_count"`
	TotalSize    int64       `json:"total_size"`
}

// Project represents a vervids project
type Project struct {
	ProjectName  string    `json:"project_name"`
	ProjectPath  string    `json:"project_path"`
	CreatedAt    time.Time `json:"created_at"`
	Versions     []Version `json:"versions"`
    UseDocker    bool      `json:"use_docker"`
	DockerVolume string    `json:"docker_volume,omitempty"`
}

// Initialize creates a new project with the initial version (Docker-only storage)
func Initialize(aepxFilePath string) (*Project, error) {
    // Create .vervids directory structure (local metadata)
    if err := storage.Initialize(); err != nil {
        return nil, fmt.Errorf("failed to create .vervids directory: %w", err)
    }

    // Ensure Docker is ready
    if err := docker.EnsureDockerReady(); err != nil {
        return nil, err
    }

	// Get file info
	fileSize, err := storage.GetFileSize(aepxFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file size: %w", err)
	}

	// Create project
    proj := &Project{
		ProjectName:  filepath.Base(aepxFilePath),
		ProjectPath:  aepxFilePath,
		CreatedAt:    time.Now(),
		Versions:     []Version{},
        UseDocker:    true,
		DockerVolume: docker.VolumeName,
	}

	// Create initial version (version 0)
	version := Version{
		Number:     0,
		Message:    "Initial version",
		Timestamp:  time.Now(),
		Size:       fileSize,
		Assets:     []AssetInfo{},
		AssetCount: 0,
		TotalSize:  fileSize,
	}

	// Parse .aepx file for assets
	scriptPath := assets.GetParserScriptPath()
	parseResult, err := assets.ParseAEPX(aepxFilePath, scriptPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse .aepx file: %w", err)
	}

    // Store the project file and assets in Docker
    // Use project filename (without extension) as project ID
    versionDir := fmt.Sprintf("v%03d", version.Number)
    projectBaseName := strings.TrimSuffix(filepath.Base(aepxFilePath), filepath.Ext(aepxFilePath))
    projectID := sanitizeProjectName(projectBaseName)
    dockerVersionDir := filepath.Join(docker.StoragePath, projectID, versionDir)

    if err := docker.CreateDirectory(dockerVersionDir); err != nil {
        return nil, fmt.Errorf("failed to create version directory in Docker: %w", err)
    }

    // Copy .aepx file
    dockerProjectPath := filepath.Join(dockerVersionDir, filepath.Base(aepxFilePath))
    if err := docker.CopyToContainer(aepxFilePath, dockerProjectPath); err != nil {
        return nil, fmt.Errorf("failed to copy project file to Docker: %w", err)
    }
    version.DockerPath = dockerProjectPath

    // Create shared assets directory at project level (not per version)
    // Use the same projectID from above
    sharedAssetsDir := filepath.Join(docker.StoragePath, projectID, "assets")
    if err := docker.CreateDirectory(sharedAssetsDir); err != nil {
        return nil, fmt.Errorf("failed to create shared assets directory in Docker: %w", err)
    }

    // Copy assets (only if they don't already exist in shared pool)
    for _, asset := range parseResult.Assets {
        sharedAssetPath := filepath.Join(sharedAssetsDir, asset.Filename)
        
        // Check if asset already exists
        if !docker.PathExistsInContainer(sharedAssetPath) {
            // Copy new asset to shared pool
            if err := docker.CopyToContainer(asset.Path, sharedAssetPath); err != nil {
                fmt.Printf("Warning: failed to copy asset %s: %v\n", asset.Filename, err)
                continue
            }
            fmt.Printf("✓ Copied new asset: %s\n", asset.Filename)
        } else {
            fmt.Printf("✓ Reusing existing asset: %s\n", asset.Filename)
        }
        
        // Reference shared asset (not version-specific)
        version.Assets = append(version.Assets, AssetInfo{
            OriginalPath: asset.Path,
            RelativePath: asset.RelativePath,
            Filename:     asset.Filename,
            Extension:    asset.Extension,
            Size:         asset.Size,
            DockerPath:   sharedAssetPath, // Point to shared location
        })
    }

	version.AssetCount = len(version.Assets)
	version.TotalSize = parseResult.TotalSize

	// Convert AssetInfo to AssetInfoInput for tracking
	currentAssetsInput := make([]tracking.AssetInfoInput, len(version.Assets))
	for i, asset := range version.Assets {
		currentAssetsInput[i] = tracking.AssetInfoInput{
			Filename:   asset.Filename,
			Extension:  asset.Extension,
			Size:       asset.Size,
			DockerPath: asset.DockerPath,
		}
	}

	// Create asset tracking for initial version (no previous version to compare)
	track := tracking.CreateTracking(version.Number, version.Message, currentAssetsInput, []tracking.AssetInfoInput{})
	if err := tracking.SaveTracking(version.Number, dockerVersionDir, track); err != nil {
		fmt.Printf("Warning: failed to save asset tracking: %v\n", err)
	}

	proj.Versions = append(proj.Versions, version)

	// Save config
	if err := proj.Save(); err != nil {
		return nil, fmt.Errorf("failed to save config: %w", err)
	}

	return proj, nil
}

// Load loads the project from config.json in current directory
func Load() (*Project, error) {
	configPath := storage.GetConfigPath()
	return LoadFromPath(configPath)
}

// LoadFromPath loads a project from a specific config.json path
func LoadFromPath(configPath string) (*Project, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var proj Project
	if err := json.Unmarshal(data, &proj); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &proj, nil
}

// sanitizeProjectName creates a safe project ID from a filename
func sanitizeProjectName(name string) string {
	// Remove invalid characters for filesystem/docker paths
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	name = strings.ReplaceAll(name, "..", "_")
	// Remove other potentially problematic characters
	invalidChars := []string{":", "*", "?", "\"", "<", ">", "|"}
	for _, char := range invalidChars {
		name = strings.ReplaceAll(name, char, "_")
	}
	// Limit length
	if len(name) > 100 {
		name = name[:100]
	}
	return name
}

// Save saves the project to config.json
func (p *Project) Save() error {
	configPath := storage.GetConfigPath()

	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// Commit creates a new version of the project using the stored project path
func (p *Project) Commit(message string) (*Version, error) {
	return p.CommitWithPath(message, p.ProjectPath)
}

// CommitWithPath creates a new version of the project using the provided .aepx file path
func (p *Project) CommitWithPath(message string, aepxFilePath string) (*Version, error) {
	// Get next version number
	nextVersion := len(p.Versions)

	// Get current file size
	fileSize, err := storage.GetFileSize(aepxFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file size: %w", err)
	}

	// Create version
	version := Version{
		Number:     nextVersion,
		Message:    message,
		Timestamp:  time.Now(),
		Size:       fileSize,
		Assets:     []AssetInfo{},
		AssetCount: 0,
		TotalSize:  fileSize,
	}

    // Parse .aepx file for assets
	scriptPath := assets.GetParserScriptPath()
	parseResult, err := assets.ParseAEPX(aepxFilePath, scriptPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse .aepx file: %w", err)
	}

    // Ensure Docker is ready
    if err := docker.EnsureDockerReady(); err != nil {
        return nil, err
    }

    // Store the file and assets in Docker
    // Use project filename (without extension) as project ID
    versionDir := fmt.Sprintf("v%03d", version.Number)
    projectBaseName := strings.TrimSuffix(filepath.Base(aepxFilePath), filepath.Ext(aepxFilePath))
    projectID := sanitizeProjectName(projectBaseName)
    dockerVersionDir := filepath.Join(docker.StoragePath, projectID, versionDir)

    if err := docker.CreateDirectory(dockerVersionDir); err != nil {
        return nil, fmt.Errorf("failed to create version directory in Docker: %w", err)
    }

    // Copy .aepx file
    dockerProjectPath := filepath.Join(dockerVersionDir, filepath.Base(aepxFilePath))
    if err := docker.CopyToContainer(aepxFilePath, dockerProjectPath); err != nil {
        return nil, fmt.Errorf("failed to copy project file to Docker: %w", err)
    }
    version.DockerPath = dockerProjectPath

    // Use shared assets directory at project level
    // Use the same projectID from above
    sharedAssetsDir := filepath.Join(docker.StoragePath, projectID, "assets")
    if err := docker.CreateDirectory(sharedAssetsDir); err != nil {
        return nil, fmt.Errorf("failed to ensure shared assets directory exists: %w", err)
    }

    // Get all previously used assets from all previous versions
    previousAssetsMap := make(map[string]string) // filename -> docker path
    for _, prevVersion := range p.Versions {
        for _, prevAsset := range prevVersion.Assets {
            previousAssetsMap[prevAsset.Filename] = prevAsset.DockerPath
        }
    }

    // Copy only new assets, reuse existing ones
    for _, asset := range parseResult.Assets {
        sharedAssetPath := filepath.Join(sharedAssetsDir, asset.Filename)
        
        // Check if asset already exists in shared pool
        if !docker.PathExistsInContainer(sharedAssetPath) {
            // Copy new asset to shared pool
            if err := docker.CopyToContainer(asset.Path, sharedAssetPath); err != nil {
                fmt.Printf("Warning: failed to copy asset %s: %v\n", asset.Filename, err)
                continue
            }
            fmt.Printf("✓ Copied new asset: %s\n", asset.Filename)
        } else {
            // Asset already exists, use existing path
            if existingPath := previousAssetsMap[asset.Filename]; existingPath != "" {
                sharedAssetPath = existingPath
            }
            fmt.Printf("✓ Reusing existing asset: %s\n", asset.Filename)
        }
        
        // Reference shared asset
        version.Assets = append(version.Assets, AssetInfo{
            OriginalPath: asset.Path,
            RelativePath: asset.RelativePath,
            Filename:     asset.Filename,
            Extension:    asset.Extension,
            Size:         asset.Size,
            DockerPath:   sharedAssetPath, // Point to shared location
        })
    }

	version.AssetCount = len(version.Assets)
	version.TotalSize = parseResult.TotalSize

	// Convert current AssetInfo to AssetInfoInput for tracking
	currentAssetsInput := make([]tracking.AssetInfoInput, len(version.Assets))
	for i, asset := range version.Assets {
		currentAssetsInput[i] = tracking.AssetInfoInput{
			Filename:   asset.Filename,
			Extension:  asset.Extension,
			Size:       asset.Size,
			DockerPath: asset.DockerPath,
		}
	}

	// Get previous version's assets for comparison
	previousAssetsInput := make([]tracking.AssetInfoInput, 0)
	if len(p.Versions) > 0 {
		previousAssets := p.Versions[len(p.Versions)-1].Assets
		previousAssetsInput = make([]tracking.AssetInfoInput, len(previousAssets))
		for i, asset := range previousAssets {
			previousAssetsInput[i] = tracking.AssetInfoInput{
				Filename:   asset.Filename,
				Extension:  asset.Extension,
				Size:       asset.Size,
				DockerPath: asset.DockerPath,
			}
		}
	}

	// Create asset tracking comparing with previous version
	track := tracking.CreateTracking(version.Number, version.Message, currentAssetsInput, previousAssetsInput)
	if err := tracking.SaveTracking(version.Number, dockerVersionDir, track); err != nil {
		fmt.Printf("Warning: failed to save asset tracking: %v\n", err)
	}

	// Update project path to the latest committed file
	p.ProjectPath = aepxFilePath

	// Add version to project
	p.Versions = append(p.Versions, version)

	// Save config
	if err := p.Save(); err != nil {
		return nil, fmt.Errorf("failed to save config: %w", err)
	}

	return &version, nil
}

// GetVersion returns a specific version by number
func (p *Project) GetVersion(number int) (*Version, error) {
	if number < 0 || number >= len(p.Versions) {
		return nil, fmt.Errorf("version %d does not exist", number)
	}
	return &p.Versions[number], nil
}

// GetLatestVersion returns the most recent version
func (p *Project) GetLatestVersion() *Version {
	if len(p.Versions) == 0 {
		return nil
	}
	return &p.Versions[len(p.Versions)-1]
}

// ProjectInfo represents basic info about a project found in Docker
type ProjectInfo struct {
	Name       string
	DockerPath string
}

// GetAllProjects scans Docker storage and returns all projects
func GetAllProjects() ([]ProjectInfo, error) {
	if err := docker.EnsureDockerReady(); err != nil {
		return nil, err
	}

	// List all directories that contain version folders (v000, v001, etc.)
	// This finds actual projects, not just top-level folders
	output, err := docker.ExecInContainer("sh", "-c", fmt.Sprintf(
		"find %s -type d -name 'v[0-9][0-9][0-9]' -mindepth 2 -maxdepth 2 | sed 's|/v[0-9][0-9][0-9]$||' | sort -u",
		docker.StoragePath))
	if err != nil {
		return []ProjectInfo{}, nil // No projects found, return empty
	}

	var projects []ProjectInfo
	lines := strings.Split(strings.TrimSpace(output), "\n")
	seen := make(map[string]bool)
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Get the project directory (the parent of the version folder)
		projectPath := line
		// Extract project name: could be direct child of /vervids or nested
		relPath := strings.TrimPrefix(projectPath, docker.StoragePath+"/")
		parts := strings.Split(relPath, "/")
		var projectName string
		if len(parts) == 1 {
			projectName = parts[0]
		} else {
			// Use the last part as project name
			projectName = parts[len(parts)-1]
		}
		
		// Try to find config.json to get actual project name
		// Search in current directory and common project locations
		home := os.Getenv("HOME")
		searchDirs := []string{
			".",
			filepath.Join(home, "Documents"),
			filepath.Join(home, "Desktop"),
			filepath.Join(home, "Projects"),
		}
		
		foundName := projectName // default
		for _, baseDir := range searchDirs {
			// Check if there's a directory matching the project name
			if entries, err := os.ReadDir(baseDir); err == nil {
				for _, entry := range entries {
					if entry.IsDir() {
						configPath := filepath.Join(baseDir, entry.Name(), storage.VerVidsDir, storage.ConfigFile)
						if data, err := os.ReadFile(configPath); err == nil {
							var proj Project
							if json.Unmarshal(data, &proj) == nil {
								// Check if this config's docker path matches
								configProjectID := sanitizeProjectName(strings.TrimSuffix(proj.ProjectName, filepath.Ext(proj.ProjectName)))
								if configProjectID == projectName || strings.Contains(projectName, configProjectID) || strings.Contains(configProjectID, projectName) {
									foundName = strings.TrimSuffix(proj.ProjectName, filepath.Ext(proj.ProjectName))
									break
								}
							}
						}
					}
				}
				if foundName != projectName {
					break
				}
			}
		}
		projectName = foundName
		
		// Use full path as unique key to avoid duplicates
		if projectName != "" && !seen[projectPath] {
			seen[projectPath] = true
			projects = append(projects, ProjectInfo{
				Name:       projectName,
				DockerPath: projectPath,
			})
		}
	}

	return projects, nil
}

// FindProjectConfig searches for a config.json file that matches a project name
func FindProjectConfig(projectName string) (string, error) {
	// Search common locations for projects with this name
	// This is a simple approach - look in current directory and parent
	searchPaths := []string{
		".",
		filepath.Dir("."),
	}

	for _, searchPath := range searchPaths {
		configPath := filepath.Join(searchPath, storage.VerVidsDir, storage.ConfigFile)
		if _, err := os.Stat(configPath); err == nil {
			// Check if this config matches the project name
			data, err := os.ReadFile(configPath)
			if err != nil {
				continue
			}
			var proj Project
			if err := json.Unmarshal(data, &proj); err != nil {
				continue
			}
			// Match if project directory name matches
			configDir := filepath.Base(filepath.Dir(filepath.Dir(configPath)))
			if configDir == projectName || strings.Contains(proj.ProjectName, projectName) {
				return configPath, nil
			}
		}
	}

	return "", fmt.Errorf("config not found for project: %s", projectName)
}

// RemoveVersion removes a version by number from the project and compacts the slice.
func (p *Project) RemoveVersion(number int) error {
    if number < 0 || number >= len(p.Versions) {
        return fmt.Errorf("version %d does not exist", number)
    }
    // Remove without re-numbering historical versions (keep numbers stable)
    filtered := make([]Version, 0, len(p.Versions))
    for _, v := range p.Versions {
        if v.Number != number {
            filtered = append(filtered, v)
        }
    }
    p.Versions = filtered
    return p.Save()
}

// PruneMissingDockerVersions removes versions whose Docker-backed files are missing.
// Returns the number of versions removed.
func (p *Project) PruneMissingDockerVersions() (int, error) {
    // Ensure Docker ready (in case we need to exec)
    if err := docker.EnsureDockerReady(); err != nil {
        return 0, err
    }
    removed := 0
    kept := make([]Version, 0, len(p.Versions))
    for _, v := range p.Versions {
        if v.DockerPath == "" {
            // If no docker path recorded, keep (legacy/local); or drop? choose keep
            kept = append(kept, v)
            continue
        }
        if docker.PathExistsInContainer(v.DockerPath) {
            kept = append(kept, v)
            continue
        }
        removed++
    }
    if removed > 0 {
        p.Versions = kept
        if err := p.Save(); err != nil {
            return removed, err
        }
    }
    return removed, nil
}


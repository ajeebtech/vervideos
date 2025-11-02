package project

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ajeebtech/vervideos/internal/assets"
	"github.com/ajeebtech/vervideos/internal/docker"
	"github.com/ajeebtech/vervideos/internal/storage"
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

// Initialize creates a new project with the initial version
func Initialize(aepxFilePath string, useDocker bool) (*Project, error) {
	// Create .vervids directory structure
	if err := storage.Initialize(); err != nil {
		return nil, fmt.Errorf("failed to create .vervids directory: %w", err)
	}

	// If using Docker, ensure container is running
	if useDocker {
		if !docker.IsDockerInstalled() {
			return nil, fmt.Errorf("Docker is not installed")
		}

		if !docker.IsContainerRunning() {
			fmt.Println("Starting Docker storage container...")
			if err := docker.CreateContainer(); err != nil {
				return nil, fmt.Errorf("failed to create Docker container: %w", err)
			}
		}
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
		UseDocker:    useDocker,
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

	// Store the project file and assets
	versionDir := fmt.Sprintf("v%03d", version.Number)
	
	if useDocker {
		// Store in Docker
		projectID := filepath.Base(filepath.Dir(aepxFilePath))
		dockerVersionDir := filepath.Join(docker.StoragePath, projectID, versionDir)
		
		// Create directory in container
		if err := docker.CreateDirectory(dockerVersionDir); err != nil {
			return nil, fmt.Errorf("failed to create version directory in Docker: %w", err)
		}

		// Copy .aepx file
		dockerProjectPath := filepath.Join(dockerVersionDir, filepath.Base(aepxFilePath))
		if err := docker.CopyToContainer(aepxFilePath, dockerProjectPath); err != nil {
			return nil, fmt.Errorf("failed to copy project file to Docker: %w", err)
		}
		version.DockerPath = dockerProjectPath

		// Copy assets
		for _, asset := range parseResult.Assets {
			dockerAssetPath := filepath.Join(dockerVersionDir, "assets", asset.Filename)
			if err := docker.CopyToContainer(asset.Path, dockerAssetPath); err != nil {
				fmt.Printf("Warning: failed to copy asset %s: %v\n", asset.Filename, err)
				continue
			}
			
			version.Assets = append(version.Assets, AssetInfo{
				OriginalPath: asset.Path,
				RelativePath: asset.RelativePath,
				Filename:     asset.Filename,
				Extension:    asset.Extension,
				Size:         asset.Size,
				DockerPath:   dockerAssetPath,
			})
		}
	} else {
		// Store locally
		localVersionDir := filepath.Join(storage.GetVersionsDir(), versionDir)
		if err := os.MkdirAll(localVersionDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create version directory: %w", err)
		}

		// Copy .aepx file
		destPath := filepath.Join(localVersionDir, filepath.Base(aepxFilePath))
		if err := storage.CopyFile(aepxFilePath, destPath); err != nil {
			return nil, fmt.Errorf("failed to copy file: %w", err)
		}
		version.FilePath = destPath

		// Copy assets
		assetsDir := filepath.Join(localVersionDir, "assets")
		if err := os.MkdirAll(assetsDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create assets directory: %w", err)
		}

		for _, asset := range parseResult.Assets {
			assetDestPath := filepath.Join(assetsDir, asset.Filename)
			if err := storage.CopyFile(asset.Path, assetDestPath); err != nil {
				fmt.Printf("Warning: failed to copy asset %s: %v\n", asset.Filename, err)
				continue
			}
			
			version.Assets = append(version.Assets, AssetInfo{
				OriginalPath: asset.Path,
				RelativePath: asset.RelativePath,
				Filename:     asset.Filename,
				Extension:    asset.Extension,
				Size:         asset.Size,
				DockerPath:   assetDestPath,
			})
		}
	}

	version.AssetCount = len(version.Assets)
	version.TotalSize = parseResult.TotalSize
	proj.Versions = append(proj.Versions, version)

	// Save config
	if err := proj.Save(); err != nil {
		return nil, fmt.Errorf("failed to save config: %w", err)
	}

	return proj, nil
}

// Load loads the project from config.json
func Load() (*Project, error) {
	configPath := storage.GetConfigPath()

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

// Commit creates a new version of the project
func (p *Project) Commit(message string) (*Version, error) {
	// Get next version number
	nextVersion := len(p.Versions)

	// Get current file size
	fileSize, err := storage.GetFileSize(p.ProjectPath)
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
	parseResult, err := assets.ParseAEPX(p.ProjectPath, scriptPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse .aepx file: %w", err)
	}

	// Store the file and assets
	versionDir := fmt.Sprintf("v%03d", version.Number)

	if p.UseDocker {
		// Ensure Docker container is running
		if !docker.IsContainerRunning() {
			if err := docker.StartContainer(); err != nil {
				return nil, fmt.Errorf("failed to start Docker container: %w", err)
			}
		}

		// Store in Docker
		projectID := filepath.Base(filepath.Dir(p.ProjectPath))
		dockerVersionDir := filepath.Join(docker.StoragePath, projectID, versionDir)
		
		// Create directory in container
		if err := docker.CreateDirectory(dockerVersionDir); err != nil {
			return nil, fmt.Errorf("failed to create version directory in Docker: %w", err)
		}

		// Copy .aepx file
		dockerProjectPath := filepath.Join(dockerVersionDir, filepath.Base(p.ProjectPath))
		if err := docker.CopyToContainer(p.ProjectPath, dockerProjectPath); err != nil {
			return nil, fmt.Errorf("failed to copy project file to Docker: %w", err)
		}
		version.DockerPath = dockerProjectPath

		// Copy assets
		for _, asset := range parseResult.Assets {
			dockerAssetPath := filepath.Join(dockerVersionDir, "assets", asset.Filename)
			if err := docker.CopyToContainer(asset.Path, dockerAssetPath); err != nil {
				fmt.Printf("Warning: failed to copy asset %s: %v\n", asset.Filename, err)
				continue
			}
			
			version.Assets = append(version.Assets, AssetInfo{
				OriginalPath: asset.Path,
				RelativePath: asset.RelativePath,
				Filename:     asset.Filename,
				Extension:    asset.Extension,
				Size:         asset.Size,
				DockerPath:   dockerAssetPath,
			})
		}
	} else {
		// Store locally
		localVersionDir := filepath.Join(storage.GetVersionsDir(), versionDir)
		if err := os.MkdirAll(localVersionDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create version directory: %w", err)
		}

		// Copy .aepx file
		destPath := filepath.Join(localVersionDir, filepath.Base(p.ProjectPath))
		if err := storage.CopyFile(p.ProjectPath, destPath); err != nil {
			return nil, fmt.Errorf("failed to copy file: %w", err)
		}
		version.FilePath = destPath

		// Copy assets
		assetsDir := filepath.Join(localVersionDir, "assets")
		if err := os.MkdirAll(assetsDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create assets directory: %w", err)
		}

		for _, asset := range parseResult.Assets {
			assetDestPath := filepath.Join(assetsDir, asset.Filename)
			if err := storage.CopyFile(asset.Path, assetDestPath); err != nil {
				fmt.Printf("Warning: failed to copy asset %s: %v\n", asset.Filename, err)
				continue
			}
			
			version.Assets = append(version.Assets, AssetInfo{
				OriginalPath: asset.Path,
				RelativePath: asset.RelativePath,
				Filename:     asset.Filename,
				Extension:    asset.Extension,
				Size:         asset.Size,
				DockerPath:   assetDestPath,
			})
		}
	}

	version.AssetCount = len(version.Assets)
	version.TotalSize = parseResult.TotalSize

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


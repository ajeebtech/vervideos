package storage

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const (
	VerVidsDir     = ".vervids"
	ConfigFile     = "config.json"
	VersionsDir    = "versions"
	ContextFile    = "current_project.json"
)

// ProjectContext stores the currently selected project
type ProjectContext struct {
	ProjectName string `json:"project_name"`
	ConfigPath  string `json:"config_path"`
}

// IsInitialized checks if .vervids directory exists in current directory
func IsInitialized() bool {
	_, err := os.Stat(VerVidsDir)
	return err == nil
}

// Initialize creates the .vervids directory structure
func Initialize() error {
	// Get current working directory for error messages
	cwd, _ := os.Getwd()
	
	// Check if current directory is writable
	if err := checkDirectoryWritable("."); err != nil {
		return fmt.Errorf("cannot create .vervids directory in '%s': %w. Please ensure you have write permissions.", cwd, err)
	}
	
	// Create .vervids directory
	if err := os.Mkdir(VerVidsDir, 0755); err != nil {
		if os.IsPermission(err) {
			return fmt.Errorf("permission denied: cannot create .vervids directory in '%s'. Please check directory permissions.", cwd)
		}
		if os.IsExist(err) {
			// Directory already exists, that's fine
		} else {
			return fmt.Errorf("failed to create .vervids directory: %w", err)
		}
	}

	// Create versions directory
	versionsPath := filepath.Join(VerVidsDir, VersionsDir)
	if err := os.Mkdir(versionsPath, 0755); err != nil {
		if os.IsPermission(err) {
			return fmt.Errorf("permission denied: cannot create versions directory. Please check directory permissions.")
		}
		if os.IsExist(err) {
			// Directory already exists, that's fine
		} else {
			return fmt.Errorf("failed to create versions directory: %w", err)
		}
	}

	return nil
}

// checkDirectoryWritable checks if a directory is writable by attempting to create a test file
func checkDirectoryWritable(dir string) error {
	testFile := filepath.Join(dir, ".vervids_write_test")
	
	// Try to create a test file
	f, err := os.Create(testFile)
	if err != nil {
		return fmt.Errorf("directory is not writable: %w", err)
	}
	f.Close()
	
	// Clean up test file
	os.Remove(testFile)
	return nil
}

// GetConfigPath returns the path to config.json
func GetConfigPath() string {
	return filepath.Join(VerVidsDir, ConfigFile)
}

// GetVersionsDir returns the path to versions directory
func GetVersionsDir() string {
	return filepath.Join(VerVidsDir, VersionsDir)
}

// GetVersionPath returns the path for a specific version
func GetVersionPath(versionNum int) string {
	return filepath.Join(GetVersionsDir(), filepath.Base(GetVersionsDir()), string(rune(versionNum)))
}

// CopyFile copies a file from src to dst
func CopyFile(src, dst string) error {
	// Create destination directory if it doesn't exist
	dstDir := filepath.Dir(dst)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return err
	}

	// Open source file
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	// Create destination file
	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	// Copy contents
	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	// Sync to ensure data is written
	return destFile.Sync()
}

// GetFileSize returns the size of a file in bytes
func GetFileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// GetContextPath returns the path to the current project context file
func GetContextPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory
		return filepath.Join(".vervids", ContextFile)
	}
	contextDir := filepath.Join(home, ".vervids")
	os.MkdirAll(contextDir, 0755)
	return filepath.Join(contextDir, ContextFile)
}

// SaveContext saves the current project context
func SaveContext(context *ProjectContext) error {
	contextPath := GetContextPath()
	data, err := json.MarshalIndent(context, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(contextPath, data, 0644)
}

// LoadContext loads the current project context
func LoadContext() (*ProjectContext, error) {
	contextPath := GetContextPath()
	data, err := os.ReadFile(contextPath)
	if err != nil {
		return nil, err
	}
	var context ProjectContext
	if err := json.Unmarshal(data, &context); err != nil {
		return nil, err
	}
	return &context, nil
}

// HasContext checks if a project context exists
func HasContext() bool {
	contextPath := GetContextPath()
	_, err := os.Stat(contextPath)
	return err == nil
}

// ClearContext removes the current project context
func ClearContext() error {
	contextPath := GetContextPath()
	return os.Remove(contextPath)
}


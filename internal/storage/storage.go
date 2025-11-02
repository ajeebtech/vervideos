package storage

import (
	"io"
	"os"
	"path/filepath"
)

const (
	VerVidsDir     = ".vervids"
	ConfigFile     = "config.json"
	VersionsDir    = "versions"
)

// IsInitialized checks if .vervids directory exists in current directory
func IsInitialized() bool {
	_, err := os.Stat(VerVidsDir)
	return err == nil
}

// Initialize creates the .vervids directory structure
func Initialize() error {
	// Create .vervids directory
	if err := os.Mkdir(VerVidsDir, 0755); err != nil {
		return err
	}

	// Create versions directory
	versionsPath := filepath.Join(VerVidsDir, VersionsDir)
	if err := os.Mkdir(versionsPath, 0755); err != nil {
		return err
	}

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


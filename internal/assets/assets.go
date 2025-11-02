package assets

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Asset represents a file referenced in the .aepx project
type Asset struct {
	Path         string `json:"path"`
	RelativePath string `json:"relative_path"`
	Filename     string `json:"filename"`
	Extension    string `json:"extension"`
	Size         int64  `json:"size"`
}

// ParseResult represents the output from the Python parser
type ParseResult struct {
	ProjectFile   string   `json:"project_file"`
	Assets        []Asset  `json:"assets"`
	MissingAssets []string `json:"missing_assets"`
	TotalSize     int64    `json:"total_size"`
}

// ParseAEPX runs the Python script to parse an .aepx file
func ParseAEPX(aepxPath string, scriptPath string) (*ParseResult, error) {
	// Run the Python parser
	cmd := exec.Command("python3", scriptPath, aepxPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if exit code is 2 (missing assets warning)
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 2 {
				// Continue parsing - exit code 2 means missing assets but valid output
			} else {
				return nil, fmt.Errorf("failed to parse .aepx file (exit %d): %s", exitErr.ExitCode(), string(output))
			}
		} else {
			return nil, fmt.Errorf("failed to run parser: %w - output: %s", err, string(output))
		}
	}

	// Parse JSON output
	var result ParseResult
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON output: %w (output was: %s)", err, string(output))
	}

	return &result, nil
}

// GetParserScriptPath returns the path to the Python parser script
func GetParserScriptPath() string {
	// Try to find the script in common locations
	possiblePaths := []string{
		"scripts/parse_aepx.py",                                    // Relative to current dir
		"/Users/jatin/Documents/vervideos/scripts/parse_aepx.py",  // Absolute path (dev)
		"/usr/local/share/vervids/scripts/parse_aepx.py",          // Installed location
	}
	
	for _, path := range possiblePaths {
		absPath, _ := filepath.Abs(path)
		if _, err := os.Stat(absPath); err == nil {
			return absPath
		}
	}
	
	// If none found, return absolute dev path as default
	return "/Users/jatin/Documents/vervideos/scripts/parse_aepx.py"
}


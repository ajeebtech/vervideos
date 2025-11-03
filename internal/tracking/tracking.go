package tracking

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ajeebtech/vervideos/internal/docker"
)

// AssetInfoInput represents asset info for tracking (to avoid import cycle)
type AssetInfoInput struct {
	Filename     string
	Extension    string
	Size         int64
	DockerPath   string
}

// AssetStatus represents the status of an asset in a commit
type AssetStatus struct {
	Filename     string `json:"filename"`
	Path         string `json:"path"`
	Extension    string `json:"extension"`
	Size         int64  `json:"size"`
	Status       string `json:"status"` // "present", "missing", "removed", "new"
	Present      bool   `json:"present"`
	InPrevious   bool   `json:"in_previous"`
}

// AssetTracking represents the complete asset tracking for a commit
type AssetTracking struct {
	Version       int           `json:"version"`
	CommitMessage string        `json:"commit_message"`
	Timestamp     string        `json:"timestamp"`
	Assets        []AssetStatus `json:"assets"`
	TotalAssets   int           `json:"total_assets"`
	PresentAssets int           `json:"present_assets"`
	MissingAssets int           `json:"missing_assets"`
	NewAssets     int           `json:"new_assets"`
	RemovedAssets int           `json:"removed_assets"`
}

// SaveTracking saves asset tracking JSON to Docker
func SaveTracking(version int, versionDir string, tracking *AssetTracking) error {
	// Create tracking JSON file
	jsonData, err := json.MarshalIndent(tracking, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal tracking data: %w", err)
	}

	// Save to local temp file first
	tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("asset-tracking-v%03d.json", version))
	if err := os.WriteFile(tmpFile, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write tracking file: %w", err)
	}
	defer os.Remove(tmpFile) // Clean up temp file

	// Copy to Docker
	dockerPath := filepath.Join(versionDir, "asset-tracking.json")
	if err := docker.CopyToContainer(tmpFile, dockerPath); err != nil {
		return fmt.Errorf("failed to copy tracking file to Docker: %w", err)
	}

	return nil
}

// LoadTracking loads asset tracking JSON from Docker
func LoadTracking(versionDir string) (*AssetTracking, error) {
	dockerPath := filepath.Join(versionDir, "asset-tracking.json")
	
	// Copy from Docker to temp file
	tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("tracking-%d.json", os.Getpid()))
	defer os.Remove(tmpFile)
	
	if err := docker.CopyFromContainer(dockerPath, tmpFile); err != nil {
		return nil, fmt.Errorf("failed to copy tracking file from Docker: %w", err)
	}

	data, err := os.ReadFile(tmpFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read tracking file: %w", err)
	}

	var tracking AssetTracking
	if err := json.Unmarshal(data, &tracking); err != nil {
		return nil, fmt.Errorf("failed to parse tracking file: %w", err)
	}

	return &tracking, nil
}

// CreateTracking creates asset tracking by comparing current assets with previous version
func CreateTracking(version int, commitMessage string, currentAssets []AssetInfoInput, previousAssets []AssetInfoInput) *AssetTracking {
	tracking := &AssetTracking{
		Version:       version,
		CommitMessage: commitMessage,
		Timestamp:     time.Now().Format(time.RFC3339),
		Assets:        []AssetStatus{},
	}

	// Create map of previous assets for quick lookup
	previousMap := make(map[string]bool)
	for _, asset := range previousAssets {
		previousMap[asset.Filename] = true
	}

	// Process current assets
	currentMap := make(map[string]bool)
	for _, asset := range currentAssets {
		currentMap[asset.Filename] = true
		status := AssetStatus{
			Filename:   asset.Filename,
			Path:       asset.DockerPath,
			Extension:  asset.Extension,
			Size:       asset.Size,
			Present:    true,
			InPrevious: previousMap[asset.Filename],
		}

		if previousMap[asset.Filename] {
			status.Status = "present"
		} else {
			status.Status = "new"
			tracking.NewAssets++
		}
		tracking.Assets = append(tracking.Assets, status)
		tracking.PresentAssets++
	}

	// Find removed assets (in previous but not in current)
	for _, asset := range previousAssets {
		if !currentMap[asset.Filename] {
			tracking.Assets = append(tracking.Assets, AssetStatus{
				Filename:   asset.Filename,
				Path:       asset.DockerPath,
				Extension:  asset.Extension,
				Size:       asset.Size,
				Status:     "removed",
				Present:    false,
				InPrevious: true,
			})
			tracking.RemovedAssets++
		}
	}

	tracking.TotalAssets = len(tracking.Assets)
	tracking.MissingAssets = tracking.TotalAssets - tracking.PresentAssets

	return tracking
}


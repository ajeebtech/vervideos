package assets

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Asset represents a file referenced in the .aepx project
type Asset struct {
	Path         string `json:"path"`
	RelativePath string `json:"relative_path"`
	Filename     string `json:"filename"`
	Extension    string `json:"extension"`
	Size         int64  `json:"size"`
}

// ParseResult represents the output from the parser
type ParseResult struct {
	ProjectFile   string   `json:"project_file"`
	Assets        []Asset  `json:"assets"`
	MissingAssets []string `json:"missing_assets"`
	TotalSize     int64    `json:"total_size"`
}

// ParseAEPX parses an .aepx file and extracts all asset references (native Go implementation)
func ParseAEPX(aepxPath string, scriptPath string) (*ParseResult, error) {
	// scriptPath parameter is kept for backward compatibility but not used
	
	result := &ParseResult{
		ProjectFile:   "",
		Assets:        []Asset{},
		MissingAssets: []string{},
		TotalSize:     0,
	}

	// Get absolute path
	absPath, err := filepath.Abs(aepxPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}
	result.ProjectFile = absPath

	// Open and read the XML file
	file, err := os.Open(aepxPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Parse XML using decoder to handle large files efficiently
	decoder := xml.NewDecoder(file)
	assetPaths := make(map[string]bool) // Use map to avoid duplicates

	for {
		token, err := decoder.Token()
		if err != nil {
			break // End of file or error
		}

		switch se := token.(type) {
		case xml.StartElement:
			// Check local name (handles namespaced elements)
			localName := se.Name.Local
			
			// Method 1: Look for fileReference elements with fullpath attribute (most common in .aepx)
			// This handles both namespaced and non-namespaced elements
			if localName == "fileReference" {
				for _, attr := range se.Attr {
					// Check both namespaced and non-namespaced attributes
					if attr.Name.Local == "fullpath" && attr.Value != "" {
						path := strings.TrimSpace(attr.Value)
						if path != "" {
							assetPaths[path] = true
						}
					}
				}
			}

			// Method 2: Look for fullpath elements (text content)
			if localName == "fullpath" {
				var elem struct {
					Text string `xml:",chardata"`
				}
				if err := decoder.DecodeElement(&elem, &se); err == nil {
					if elem.Text != "" {
						path := strings.TrimSpace(elem.Text)
						if path != "" {
							assetPaths[path] = true
						}
					}
				}
			}

			// Method 3: Look for file/path/src/source elements
			if localName == "file" || localName == "path" || 
			   localName == "src" || localName == "source" {
				var elem struct {
					Text string `xml:",chardata"`
				}
				if err := decoder.DecodeElement(&elem, &se); err == nil {
					if elem.Text != "" {
						path := strings.TrimSpace(elem.Text)
						if path != "" {
							assetPaths[path] = true
						}
					}
				}
				// Also check attributes
				for _, attr := range se.Attr {
					if strings.Contains(strings.ToLower(attr.Name.Local), "path") ||
					   strings.Contains(strings.ToLower(attr.Name.Local), "file") {
						path := strings.TrimSpace(attr.Value)
						if path != "" {
							assetPaths[path] = true
						}
					}
				}
			}
		}
	}

	// Process each asset path
	projectDir := filepath.Dir(absPath)
	
	for assetPath := range assetPaths {
		if assetPath == "" {
			continue
		}

		// Skip URLs
		if strings.HasPrefix(assetPath, "http://") || 
		   strings.HasPrefix(assetPath, "https://") || 
		   strings.HasPrefix(assetPath, "file://") {
			continue
		}

		// Convert to absolute path if relative
		if !filepath.IsAbs(assetPath) {
			assetPath = filepath.Join(projectDir, assetPath)
		}

		// Normalize the path
		assetPath = filepath.Clean(assetPath)

		// Check if file exists
		info, err := os.Stat(assetPath)
		if err == nil && !info.IsDir() {
			// File exists
			relPath, _ := filepath.Rel(projectDir, assetPath)
			ext := filepath.Ext(assetPath)
			
			result.Assets = append(result.Assets, Asset{
				Path:         assetPath,
				RelativePath: relPath,
				Filename:     filepath.Base(assetPath),
				Extension:    ext,
				Size:         info.Size(),
			})
			result.TotalSize += info.Size()
		} else {
			// File missing
			result.MissingAssets = append(result.MissingAssets, assetPath)
		}
	}

	// Sort for consistency
	sort.Slice(result.Assets, func(i, j int) bool {
		return result.Assets[i].Path < result.Assets[j].Path
	})
	sort.Strings(result.MissingAssets)

	return result, nil
}

// GetParserScriptPath is kept for backward compatibility but no longer needed
// Returns empty string since we no longer use Python scripts
func GetParserScriptPath() string {
	return ""
}

// UpdateAssetPaths updates asset paths in an .aepx file
// It replaces paths that don't exist locally with new paths (typically from Docker storage)
// pathMap maps old paths to new paths
func UpdateAssetPaths(aepxPath string, pathMap map[string]string) error {
	// Read the entire file
	data, err := os.ReadFile(aepxPath)
	if err != nil {
		return fmt.Errorf("failed to read .aepx file: %w", err)
	}

	content := string(data)
	updated := false

	// Replace each path in the map
	for oldPath, newPath := range pathMap {
		// Replace all occurrences of the old path with the new path
		// Use strings.ReplaceAll to handle all occurrences
		if strings.Contains(content, oldPath) {
			content = strings.ReplaceAll(content, oldPath, newPath)
			updated = true
		}
	}

	// Only write if we made changes
	if updated {
		if err := os.WriteFile(aepxPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write updated .aepx file: %w", err)
		}
	}

	return nil
}

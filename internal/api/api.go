package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ajeebtech/vervideos/internal/docker"
	"github.com/ajeebtech/vervideos/internal/project"
	"github.com/ajeebtech/vervideos/internal/storage"
)

// APIResponse is a standard API response wrapper
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// ProjectListItem represents a project in the projects list
type ProjectListItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DockerPath  string `json:"docker_path"`
	CommitCount int    `json:"commit_count,omitempty"`
}

// CommitItem represents a single commit/version
type CommitItem struct {
	Number      int    `json:"number"`
	Message     string `json:"message"`
	Timestamp   string `json:"timestamp"`
	Size        int64  `json:"size"`
	AssetCount  int    `json:"asset_count"`
	TotalSize   int64  `json:"total_size"`
	Branch      string `json:"branch,omitempty"`
}

// BranchItem represents a single branch
type BranchItem struct {
	Name         string  `json:"name"`
	SourceBranch *string `json:"source_branch,omitempty"` // Use pointer to allow null in JSON
	SourceVersion int    `json:"source_version"`
	VersionCount int    `json:"version_count"`
	CreatedAt    string  `json:"created_at"`
}

// BranchesResponse contains branches for a project
type BranchesResponse struct {
	Current string       `json:"current"`
	Branches []BranchItem `json:"branches"`
}

// CurrentBranchResponse contains current branch info
type CurrentBranchResponse struct {
	Branch        string  `json:"branch"`
	SourceBranch  *string `json:"source_branch,omitempty"` // Use pointer to allow null in JSON
	SourceVersion int     `json:"source_version"`
}

// ProjectCommitsResponse contains commits for a project
type ProjectCommitsResponse struct {
	ProjectID   string       `json:"project_id"`
	ProjectName string       `json:"project_name"`
	Commits     []CommitItem `json:"commits"`
}

// StartServer starts the HTTP API server on the specified port
func StartServer(port int) error {
	mux := http.NewServeMux()
	// Register more specific routes first
	// The exact match /api/projects/match must come before the prefix /api/projects/
	mux.HandleFunc("/api/projects/match", handleMatchProject)
	mux.HandleFunc("/api/projects", handleListProjects)
	mux.HandleFunc("/api/projects/", handleProjectRoutes)
	mux.HandleFunc("/health", handleHealth)
	
	http.Handle("/", mux)

	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("🌐 Starting vervids API server on http://localhost%s\n", addr)
	fmt.Printf("📡 API endpoints:\n")
	fmt.Printf("   GET /api/projects - List all projects\n")
	fmt.Printf("   GET /api/projects/match?path={filePath} - Match file path to project\n")
	fmt.Printf("   GET /api/projects/{id}/commits - Get commits for a project\n")
	fmt.Printf("   GET /api/projects/{id}/branches - List all branches\n")
	fmt.Printf("   GET /api/projects/{id}/branch/current - Get current branch\n")
	fmt.Printf("   POST /api/projects/{id}/branches/switch - Switch branch\n")
	fmt.Printf("   POST /api/projects/{id}/pull - Pull a version to local filesystem\n")
	fmt.Printf("   GET /health - Health check\n")

	return http.ListenAndServe(addr, nil)
}

// handleHealth handles health check requests
func handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    map[string]string{"status": "ok"},
	})
}

// handleListProjects handles GET /api/projects
func handleListProjects(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	projects, err := project.GetAllProjects()
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get projects: %v", err))
		return
	}

	// Convert to API format with project IDs
	projectList := make([]ProjectListItem, 0, len(projects))
	for _, p := range projects {
		// Extract project ID from DockerPath
		// DockerPath is like /vervids/project_name or /vervids/nested/path/project_name
		relPath := strings.TrimPrefix(p.DockerPath, "/vervids/")
		parts := strings.Split(relPath, "/")
		projectID := parts[len(parts)-1] // Get the last part (actual project ID)

		// Try to get commit count by loading the project
		commitCount := 0
		configPath := findProjectConfig(p.Name)
		if configPath != "" {
			if proj, err := project.LoadFromPath(configPath); err == nil {
				commitCount = len(proj.Versions)
			}
		}

		projectList = append(projectList, ProjectListItem{
			ID:          projectID,
			Name:        p.Name,
			DockerPath:  p.DockerPath,
			CommitCount: commitCount,
		})
	}

	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    projectList,
	})
}

// handleProjectRoutes routes requests to appropriate handlers based on path
func handleProjectRoutes(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// Check for branch-related endpoints
	if strings.HasSuffix(path, "/branches") {
		handleListBranches(w, r)
		return
	}
	if strings.HasSuffix(path, "/branch/current") {
		handleGetCurrentBranch(w, r)
		return
	}
	if strings.HasSuffix(path, "/branches/switch") {
		handleSwitchBranch(w, r)
		return
	}

	// Check if this is a pull request
	if strings.HasSuffix(path, "/pull") {
		handlePullVersion(w, r)
		return
	}

	// Default to commits handler
	handleGetProjectCommits(w, r)
}

// handleGetProjectCommits handles GET /api/projects/{id}/commits
func handleGetProjectCommits(w http.ResponseWriter, r *http.Request) {
	// Check if this is the match endpoint BEFORE processing
	// Go's ServeMux will match /api/projects/match to /api/projects/ handler
	// So we need to check for it explicitly
	if r.URL.Path == "/api/projects/match" || strings.HasPrefix(r.URL.Path, "/api/projects/match?") {
		handleMatchProject(w, r)
		return
	}

	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Extract project ID from path
	// Path format: /api/projects/{id}/commits
	// Example: /api/projects/sloppy/commits
	path := strings.TrimPrefix(r.URL.Path, "/api/projects/")
	
	// Remove /commits suffix if present
	if strings.HasSuffix(path, "/commits") {
		path = strings.TrimSuffix(path, "/commits")
	}
	
	// Remove trailing slash
	path = strings.TrimSuffix(path, "/")
	projectID := path

	if projectID == "" {
		writeError(w, http.StatusBadRequest, "Project ID is required. Use: GET /api/projects/{id}/commits")
		return
	}

	// Find the project by ID
	projects, err := project.GetAllProjects()
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get projects: %v", err))
		return
	}

	var targetProject *project.ProjectInfo
	for i := range projects {
		p := &projects[i]
		// Extract project ID from DockerPath
		relPath := strings.TrimPrefix(p.DockerPath, "/vervids/")
		parts := strings.Split(relPath, "/")
		projectIDFromPath := parts[len(parts)-1] // Get the last part (actual project ID)

		if projectIDFromPath == projectID {
			targetProject = p
			break
		}
	}

	if targetProject == nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Project with ID '%s' not found", projectID))
		return
	}

	// Find and load the project config
	configPath := findProjectConfig(targetProject.Name)
	if configPath == "" {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Config file not found for project '%s'", targetProject.Name))
		return
	}

	proj, err := project.LoadFromPath(configPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to load project: %v", err))
		return
	}

	// Convert versions to commits
	commits := make([]CommitItem, 0, len(proj.Versions))
	for _, v := range proj.Versions {
		branchName := v.Branch
		if branchName == "" {
			branchName = "main"
		}
		commits = append(commits, CommitItem{
			Number:     v.Number,
			Message:    v.Message,
			Timestamp:  v.Timestamp.Format("2006-01-02 15:04:05"),
			Size:       v.Size,
			AssetCount: v.AssetCount,
			TotalSize:  v.TotalSize,
			Branch:     branchName,
		})
	}

	response := ProjectCommitsResponse{
		ProjectID:   projectID,
		ProjectName: proj.ProjectName,
		Commits:     commits,
	}

	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    response,
	})
}

// findProjectConfig searches for a project's config.json file
func findProjectConfig(projectName string) string {
	home := os.Getenv("HOME")
	searchDirs := []string{
		".",
		filepath.Join(home, "Documents"),
		filepath.Join(home, "Desktop"),
		filepath.Join(home, "Projects"),
	}

	for _, baseDir := range searchDirs {
		if entries, err := os.ReadDir(baseDir); err == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					configPath := filepath.Join(baseDir, entry.Name(), storage.VerVidsDir, storage.ConfigFile)
					if _, err := os.Stat(configPath); err == nil {
						if proj, err := project.LoadFromPath(configPath); err == nil {
							// Check if this project matches
							if strings.Contains(strings.ToLower(proj.ProjectName), strings.ToLower(projectName)) ||
								strings.Contains(strings.ToLower(projectName), strings.ToLower(proj.ProjectName)) {
								return configPath
							}
						}
					}
				}
			}
		}
		// Also check if .vervids exists directly in baseDir
		configPath := filepath.Join(baseDir, storage.VerVidsDir, storage.ConfigFile)
		if _, err := os.Stat(configPath); err == nil {
			if proj, err := project.LoadFromPath(configPath); err == nil {
				if strings.Contains(strings.ToLower(proj.ProjectName), strings.ToLower(projectName)) ||
					strings.Contains(strings.ToLower(projectName), strings.ToLower(proj.ProjectName)) {
					return configPath
				}
			}
		}
	}

	return ""
}

// handleMatchProject handles GET /api/projects/match?path={filePath}
func handleMatchProject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Get path parameter from query string
	filePath := r.URL.Query().Get("path")
	if filePath == "" {
		writeError(w, http.StatusBadRequest, "Path parameter is required. Use: GET /api/projects/match?path={filePath}")
		return
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid file path: %v", err))
		return
	}

	// Get the directory containing the file (even if file doesn't exist yet)
	fileDir := filepath.Dir(absPath)
	fileName := filepath.Base(absPath)
	
	// Check if directory exists (file might not exist yet, which is okay)
	if _, err := os.Stat(fileDir); os.IsNotExist(err) {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Directory not found: %s", fileDir))
		return
	}

	// Extract base name for matching (handle both .aep and .aepx)
	baseName := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	// Also try with .aepx extension in case the file is .aep but project is .aepx
	projectName := baseName + ".aepx"
	
	// Use comprehensive search similar to CLI's findProjectConfigFile
	configPath := findProjectConfigComprehensive(projectName, absPath, fileDir)

	// If config file found, load the project
	var proj *project.Project
	
	if configPath != "" {
		if _, err := os.Stat(configPath); err == nil {
			if loadedProj, err := project.LoadFromPath(configPath); err == nil {
				proj = loadedProj
			}
		}
	}
	
	// If still not found, try to recreate from Docker storage (like CLI does)
	if proj == nil {
		recreatedPath, err := recreateConfigFromDockerAPI(projectName)
		if err == nil && recreatedPath != "" {
			if loadedProj, err := project.LoadFromPath(recreatedPath); err == nil {
				proj = loadedProj
			}
		}
	}
	
	// If we have a project, return it
	if proj != nil {
		// Get project ID from Docker path
		projects, err := project.GetAllProjects()
		if err == nil {
			for _, p := range projects {
				// Match by project name
				if strings.EqualFold(strings.TrimSuffix(p.Name, ".aepx"), 
					strings.TrimSuffix(proj.ProjectName, ".aepx")) {
					// Extract project ID from DockerPath
					relPath := strings.TrimPrefix(p.DockerPath, "/vervids/")
					parts := strings.Split(relPath, "/")
					projectID := parts[len(parts)-1]

					writeJSON(w, http.StatusOK, APIResponse{
						Success: true,
						Data: ProjectListItem{
							ID:          projectID,
							Name:        proj.ProjectName,
							DockerPath:  p.DockerPath,
							CommitCount: len(proj.Versions),
						},
					})
					return
				}
			}
		}
	}

	// If not found, return 404
	writeError(w, http.StatusNotFound, "No project found matching the given path")
}

// findProjectConfigComprehensive uses the same comprehensive search logic as the CLI
func findProjectConfigComprehensive(projectName, filePath, fileDir string) string {
	home := os.Getenv("HOME")
	// Expanded search directories to match CLI
	searchDirs := []string{
		".",
		filepath.Join(home, "Documents"),
		filepath.Join(home, "Desktop"),
		filepath.Join(home, "Projects"),
		filepath.Join(home, "Downloads"),
		filepath.Join(home, "Movies"),
		filepath.Join(home, "Pictures"),
		filepath.Join(home, "Videos"),
		filepath.Join(home, "Library", "Application Support"),
		filepath.Join(home, ".local", "share"),
	}

	// First, check the directory containing the file
	configPath := filepath.Join(fileDir, storage.VerVidsDir, storage.ConfigFile)
	if _, err := os.Stat(configPath); err == nil {
		if proj, err := project.LoadFromPath(configPath); err == nil {
			if matchesProject(proj.ProjectName, projectName) {
				return configPath
			}
		}
	}

	// Search in parent directories (up to 3 levels)
	currentDir := fileDir
	for i := 0; i < 3; i++ {
		parentConfig := filepath.Join(currentDir, storage.VerVidsDir, storage.ConfigFile)
		if _, err := os.Stat(parentConfig); err == nil {
			if proj, err := project.LoadFromPath(parentConfig); err == nil {
				if matchesProject(proj.ProjectName, projectName) {
					return parentConfig
				}
			}
		}
		parent := filepath.Dir(currentDir)
		if parent == currentDir {
			break // Reached root
		}
		currentDir = parent
	}

	// Search in common directories (one level deep)
	for _, baseDir := range searchDirs {
		if entries, err := os.ReadDir(baseDir); err == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					potentialConfigPath := filepath.Join(baseDir, entry.Name(), storage.VerVidsDir, storage.ConfigFile)
					if _, err := os.Stat(potentialConfigPath); err == nil {
						if proj, err := project.LoadFromPath(potentialConfigPath); err == nil {
							if matchesProject(proj.ProjectName, projectName) {
								return potentialConfigPath
							}
						}
					}
				}
			}
		}
		// Also check if .vervids exists directly in baseDir
		directConfigPath := filepath.Join(baseDir, storage.VerVidsDir, storage.ConfigFile)
		if _, err := os.Stat(directConfigPath); err == nil {
			if proj, err := project.LoadFromPath(directConfigPath); err == nil {
				if matchesProject(proj.ProjectName, projectName) {
					return directConfigPath
				}
			}
		}
	}

	// If not found, try recursive search (max depth 5)
	deepSearchDirs := []string{
		filepath.Join(home, "Documents"),
		filepath.Join(home, "Projects"),
		filepath.Join(home, "Desktop"),
		filepath.Join(home, "Downloads"),
	}
	
	for _, baseDir := range deepSearchDirs {
		if found := findConfigRecursiveAPI(baseDir, projectName, 0, 5); found != "" {
			return found
		}
	}

	return ""
}

// matchesProject checks if a project name matches the search name
func matchesProject(projName, searchName string) bool {
	projNameLower := strings.ToLower(projName)
	searchNameLower := strings.ToLower(searchName)
	projBaseName := strings.TrimSuffix(projNameLower, ".aepx")
	searchBaseName := strings.TrimSuffix(searchNameLower, ".aepx")
	
	return strings.Contains(projBaseName, searchBaseName) ||
		strings.Contains(searchBaseName, projBaseName) ||
		strings.Contains(projNameLower, searchNameLower) ||
		strings.Contains(searchNameLower, projNameLower)
}

// findConfigRecursiveAPI recursively searches for config.json files (API version)
func findConfigRecursiveAPI(dir string, projectName string, depth int, maxDepth int) string {
	if depth > maxDepth {
		return ""
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		if entry.IsDir() {
			// Skip hidden directories and .vervids itself
			if strings.HasPrefix(entry.Name(), ".") && entry.Name() != "." && entry.Name() != ".." {
				continue
			}

			// Check for config.json in this directory's .vervids subdirectory
			configPath := filepath.Join(dir, entry.Name(), storage.VerVidsDir, storage.ConfigFile)
			if _, err := os.Stat(configPath); err == nil {
				if proj, err := project.LoadFromPath(configPath); err == nil {
					if matchesProject(proj.ProjectName, projectName) {
						return configPath
					}
				}
			}

			// Recurse into subdirectories
			subDir := filepath.Join(dir, entry.Name())
			if found := findConfigRecursiveAPI(subDir, projectName, depth+1, maxDepth); found != "" {
				return found
			}
		}
	}

	return ""
}

// recreateConfigFromDockerAPI attempts to recreate a config file from Docker storage
// This is similar to the CLI's recreateConfigFromDocker but adapted for API use
func recreateConfigFromDockerAPI(projectName string) (string, error) {
	// Get all projects from Docker
	projects, err := project.GetAllProjects()
	if err != nil {
		return "", fmt.Errorf("failed to get projects from Docker: %w", err)
	}

	// Find the matching project
	var targetProject *project.ProjectInfo
	for i, p := range projects {
		projNameLower := strings.ToLower(p.Name)
		searchNameLower := strings.ToLower(projectName)
		projBaseName := strings.TrimSuffix(projNameLower, ".aepx")
		searchBaseName := strings.TrimSuffix(searchNameLower, ".aepx")
		
		if strings.Contains(projBaseName, searchBaseName) ||
			strings.Contains(searchBaseName, projBaseName) ||
			strings.Contains(projNameLower, searchNameLower) ||
			strings.Contains(searchNameLower, projNameLower) {
			targetProject = &projects[i]
			break
		}
	}

	if targetProject == nil {
		return "", fmt.Errorf("project not found in Docker storage")
	}

	// Try to get version information from Docker
	// List versions in Docker for this project
	versionDirs, err := docker.ExecInContainer("sh", "-c", fmt.Sprintf(
		"find %s -type d -name 'v[0-9][0-9][0-9]' -mindepth 1 -maxdepth 1 | sort",
		targetProject.DockerPath))
	if err != nil {
		return "", fmt.Errorf("failed to list versions from Docker: %w", err)
	}

	versionLines := strings.Split(strings.TrimSpace(versionDirs), "\n")
	if len(versionLines) == 0 || (len(versionLines) == 1 && versionLines[0] == "") {
		return "", fmt.Errorf("no versions found in Docker for project")
	}

	// Create a minimal project config in the current directory
	currentDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	// Create .vervids directory if it doesn't exist
	vervidsDir := filepath.Join(currentDir, storage.VerVidsDir)
	if err := os.MkdirAll(vervidsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create .vervids directory: %w", err)
	}

	// Create a minimal project structure
	configPath := storage.GetConfigPath()
	
	// Try to get the latest version's .aepx file name from Docker
	latestVersionDir := strings.TrimSpace(versionLines[len(versionLines)-1])
	aepxFiles, err := docker.ExecInContainer("sh", "-c", fmt.Sprintf(
		"ls -1 %s/*.aepx 2>/dev/null | head -1", latestVersionDir))
	if err != nil {
		// If we can't find the .aepx file, use the project name
		aepxFiles = targetProject.Name
	} else {
		aepxFiles = strings.TrimSpace(aepxFiles)
		if aepxFiles != "" {
			aepxFiles = filepath.Base(aepxFiles)
		} else {
			aepxFiles = targetProject.Name
		}
	}

	// Try to reconstruct versions from Docker
	versions := []project.Version{}
	for i, versionDir := range versionLines {
		versionDir = strings.TrimSpace(versionDir)
		if versionDir == "" {
			continue
		}
		
		// Extract version number from directory name (e.g., "v000" -> 0)
		versionNum := i
		if len(versionDir) > 0 {
			versionNumStr := strings.TrimPrefix(filepath.Base(versionDir), "v")
			if num, err := strconv.Atoi(versionNumStr); err == nil {
				versionNum = num
			}
		}
		
		// Get .aepx file path in Docker
		dockerAepxPath := filepath.Join(versionDir, aepxFiles)
		if !docker.PathExistsInContainer(dockerAepxPath) {
			// Try to find any .aepx file in this version directory
			aepxList, _ := docker.ExecInContainer("sh", "-c", fmt.Sprintf(
				"ls -1 %s/*.aepx 2>/dev/null | head -1", versionDir))
			if aepxList != "" {
				dockerAepxPath = strings.TrimSpace(aepxList)
			} else {
				continue // Skip if no .aepx file found
			}
		}
		
		// Get file size from Docker (using wc -c for portability)
		sizeOutput, _ := docker.ExecInContainer("wc", "-c", dockerAepxPath)
		fileSize := int64(0)
		if sizeStr := strings.TrimSpace(sizeOutput); sizeStr != "" {
			// wc -c output format: "size filename" or just "size"
			parts := strings.Fields(sizeStr)
			if len(parts) > 0 {
				if size, err := strconv.ParseInt(parts[0], 10, 64); err == nil {
					fileSize = size
				}
			}
		}
		
		// Create version entry
		version := project.Version{
			Number:     versionNum,
			Message:    "Initial version",
			Timestamp:  time.Now(), // We don't have exact timestamp from Docker
			Size:       fileSize,
			FilePath:   filepath.Join(currentDir, aepxFiles), // Placeholder
			DockerPath: dockerAepxPath,
			Assets:     []project.AssetInfo{}, // Assets will be loaded when needed
			AssetCount: 0,
			TotalSize:  fileSize,
		}
		
		if versionNum == 0 {
			version.Message = "Initial version"
		} else {
			version.Message = fmt.Sprintf("Version %d", versionNum)
		}
		
		versions = append(versions, version)
	}

	// Create project config with reconstructed versions
	minimalProj := &project.Project{
		ProjectName:  aepxFiles,
		ProjectPath:  filepath.Join(currentDir, aepxFiles), // Placeholder path
		CreatedAt:    time.Now(), // We don't know the actual creation time
		Versions:     versions,
		UseDocker:    true,
		DockerVolume: docker.VolumeName,
	}

	// Save the config
	if err := minimalProj.Save(); err != nil {
		return "", fmt.Errorf("failed to save recreated config: %w", err)
	}

	return configPath, nil
}

// handlePullVersion handles POST /api/projects/{id}/pull
func handlePullVersion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed. Use POST")
		return
	}

	// Extract project ID from path
	// Path format: /api/projects/{id}/pull
	path := strings.TrimPrefix(r.URL.Path, "/api/projects/")
	path = strings.TrimSuffix(path, "/pull")
	path = strings.TrimSuffix(path, "/")
	projectID := path

	if projectID == "" {
		writeError(w, http.StatusBadRequest, "Project ID is required. Use: POST /api/projects/{id}/pull")
		return
	}

	// Parse request body
	var pullRequest struct {
		Version   int    `json:"version"`
		OutputDir string `json:"output_dir,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&pullRequest); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	if pullRequest.Version < 0 {
		writeError(w, http.StatusBadRequest, "Version number is required and must be >= 0")
		return
	}

	// Find the project
	projects, err := project.GetAllProjects()
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get projects: %v", err))
		return
	}

	var targetProject *project.ProjectInfo
	for i := range projects {
		p := &projects[i]
		relPath := strings.TrimPrefix(p.DockerPath, "/vervids/")
		pathParts := strings.Split(relPath, "/")
		projectIDFromPath := pathParts[len(pathParts)-1]

		if projectIDFromPath == projectID {
			targetProject = p
			break
		}
	}

	if targetProject == nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Project with ID '%s' not found", projectID))
		return
	}

	// Find and load the project config
	configPath := findProjectConfig(targetProject.Name)
	if configPath == "" {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Config file not found for project '%s'", targetProject.Name))
		return
	}

	proj, err := project.LoadFromPath(configPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to load project: %v", err))
		return
	}

	// Determine output directory
	outputDir := pullRequest.OutputDir
	if outputDir == "" {
		outputDir = "."
	}
	absOutputDir, err := filepath.Abs(outputDir)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid output directory: %v", err))
		return
	}

	// Pull the version
	restoredPath, err := proj.RestoreVersion(pullRequest.Version, absOutputDir)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to pull version: %v", err))
		return
	}

	// Return the path
	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"file_path":    restoredPath,
			"output_dir":   absOutputDir,
			"version":      pullRequest.Version,
			"project_name": proj.ProjectName,
		},
	})
}

// writeJSON writes a JSON response
func writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// writeError writes an error response
func writeError(w http.ResponseWriter, statusCode int, message string) {
	writeJSON(w, statusCode, APIResponse{
		Success: false,
		Error:   message,
	})
}

// getProjectByID finds and loads a project by its ID
func getProjectByID(projectID string) (*project.Project, string, error) {
	projects, err := project.GetAllProjects()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get projects: %w", err)
	}

	var targetProject *project.ProjectInfo
	for i := range projects {
		p := &projects[i]
		relPath := strings.TrimPrefix(p.DockerPath, "/vervids/")
		parts := strings.Split(relPath, "/")
		projectIDFromPath := parts[len(parts)-1]

		if projectIDFromPath == projectID {
			targetProject = p
			break
		}
	}

	if targetProject == nil {
		return nil, "", fmt.Errorf("project with ID '%s' not found", projectID)
	}

	configPath := findProjectConfig(targetProject.Name)
	if configPath == "" {
		return nil, "", fmt.Errorf("config file not found for project '%s'", targetProject.Name)
	}

	proj, err := project.LoadFromPath(configPath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to load project: %w", err)
	}

	return proj, configPath, nil
}

// handleListBranches handles GET /api/projects/{id}/branches
func handleListBranches(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Extract project ID from path
	path := strings.TrimPrefix(r.URL.Path, "/api/projects/")
	path = strings.TrimSuffix(path, "/branches")
	path = strings.TrimSuffix(path, "/")
	projectID := path

	if projectID == "" {
		writeError(w, http.StatusBadRequest, "Project ID is required. Use: GET /api/projects/{id}/branches")
		return
	}

	proj, _, err := getProjectByID(projectID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	// Convert branches to API format
	branches := make([]BranchItem, 0, len(proj.Branches))
	for _, branch := range proj.Branches {
		versions := proj.GetVersionsForBranch(branch.Name)
		branchItem := BranchItem{
			Name:         branch.Name,
			SourceVersion: branch.SourceVersion,
			VersionCount: len(versions),
			CreatedAt:    branch.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		if branch.SourceBranch != "" {
			sourceBranch := branch.SourceBranch
			branchItem.SourceBranch = &sourceBranch
		}
		branches = append(branches, branchItem)
	}

	response := BranchesResponse{
		Current:  proj.GetCurrentBranch(),
		Branches: branches,
	}

	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    response,
	})
}

// handleGetCurrentBranch handles GET /api/projects/{id}/branch/current
func handleGetCurrentBranch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Extract project ID from path
	path := strings.TrimPrefix(r.URL.Path, "/api/projects/")
	path = strings.TrimSuffix(path, "/branch/current")
	path = strings.TrimSuffix(path, "/")
	projectID := path

	if projectID == "" {
		writeError(w, http.StatusBadRequest, "Project ID is required. Use: GET /api/projects/{id}/branch/current")
		return
	}

	proj, _, err := getProjectByID(projectID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	currentBranchName := proj.GetCurrentBranch()
	branchInfo, err := proj.GetBranchInfo(currentBranchName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get branch info: %v", err))
		return
	}

	response := CurrentBranchResponse{
		Branch:        branchInfo.Name,
		SourceVersion: branchInfo.SourceVersion,
	}
	if branchInfo.SourceBranch != "" {
		sourceBranch := branchInfo.SourceBranch
		response.SourceBranch = &sourceBranch
	}

	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    response,
	})
}

// handleSwitchBranch handles POST /api/projects/{id}/branches/switch
func handleSwitchBranch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed. Use POST")
		return
	}

	// Extract project ID from path
	path := strings.TrimPrefix(r.URL.Path, "/api/projects/")
	path = strings.TrimSuffix(path, "/branches/switch")
	path = strings.TrimSuffix(path, "/")
	projectID := path

	if projectID == "" {
		writeError(w, http.StatusBadRequest, "Project ID is required. Use: POST /api/projects/{id}/branches/switch")
		return
	}

	proj, configPath, err := getProjectByID(projectID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	// Parse request body
	var switchRequest struct {
		Branch string `json:"branch"`
	}

	if err := json.NewDecoder(r.Body).Decode(&switchRequest); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	if switchRequest.Branch == "" {
		writeError(w, http.StatusBadRequest, "Branch name is required")
		return
	}

	// Change to project directory to save config
	configDir := filepath.Dir(filepath.Dir(configPath))
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	
	if err := os.Chdir(configDir); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to change to project directory: %v", err))
		return
	}

	// Switch branch
	if err := proj.SwitchBranch(switchRequest.Branch); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Failed to switch branch: %v", err))
		return
	}

	// Get updated branch info
	branchInfo, err := proj.GetBranchInfo(switchRequest.Branch)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get branch info: %v", err))
		return
	}

	response := CurrentBranchResponse{
		Branch:        branchInfo.Name,
		SourceVersion: branchInfo.SourceVersion,
	}
	if branchInfo.SourceBranch != "" {
		sourceBranch := branchInfo.SourceBranch
		response.SourceBranch = &sourceBranch
	}

	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    response,
	})
}


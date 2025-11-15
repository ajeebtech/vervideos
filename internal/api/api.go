package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

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
	mux.HandleFunc("/api/projects", handleListProjects)
	mux.HandleFunc("/api/projects/", handleGetProjectCommits)
	mux.HandleFunc("/health", handleHealth)
	
	http.Handle("/", mux)

	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("🌐 Starting vervids API server on http://localhost%s\n", addr)
	fmt.Printf("📡 API endpoints:\n")
	fmt.Printf("   GET /api/projects - List all projects\n")
	fmt.Printf("   GET /api/projects/{id}/commits - Get commits for a project\n")
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

// handleGetProjectCommits handles GET /api/projects/{id}/commits
func handleGetProjectCommits(w http.ResponseWriter, r *http.Request) {
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
		commits = append(commits, CommitItem{
			Number:     v.Number,
			Message:    v.Message,
			Timestamp:  v.Timestamp.Format("2006-01-02 15:04:05"),
			Size:       v.Size,
			AssetCount: v.AssetCount,
			TotalSize:  v.TotalSize,
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


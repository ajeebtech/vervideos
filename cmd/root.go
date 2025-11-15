package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ajeebtech/vervideos/internal/api"
	"github.com/ajeebtech/vervideos/internal/docker"
	"github.com/ajeebtech/vervideos/internal/project"
	"github.com/ajeebtech/vervideos/internal/storage"
	"github.com/ajeebtech/vervideos/internal/ui"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var (
	version string
	commit  string
	date    string
)

func SetVersionInfo(v, c, d string) {
	version = v
	commit = c
	date = d
}

var rootCmd = &cobra.Command{
	Use:   "vervids",
	Short: "Version control for Adobe After Effects projects",
	Long:  `vervids is a local version control system for .ae (Adobe After Effects) files.`,
	Run: func(cmd *cobra.Command, args []string) {
		printBoxedHeader()

		// Check if we have a project context
		var proj *project.Project
		var err error

		if storage.HasContext() {
			context, err := storage.LoadContext()
			if err == nil {
				proj, err = project.LoadFromPath(context.ConfigPath)
				if err == nil {
					// We have a valid project context, show commits
					showProjectCommits(proj)
					fmt.Println()
					fmt.Println(infoMsg("Available commands:"))
					fmt.Println(infoMsg("  • vervids commit \"message\" <file.aepx> - Commit a new version"))
					fmt.Println(infoMsg("  • vervids list - List all projects"))
					fmt.Println(infoMsg("  • vervids show <version> - Show version details"))
					fmt.Println(infoMsg("  • vervids pull <version> - Pull a version from Docker"))
					fmt.Println(infoMsg("  • vervids help - Show all commands"))
					return
				}
			}
		}

		// No context or invalid context - try to select
		proj, err = ensureProjectContext()
		if err != nil {
			if strings.Contains(err.Error(), "no projects available") {
				fmt.Println()
				fmt.Println(infoMsg("To get started:"))
				fmt.Println(infoMsg("  • Use 'vervids init <file.aepx>' to initialize a new project"))
				fmt.Println(infoMsg("  • Use 'vervids help' to see all available commands"))
			} else {
				fmt.Println(errorMsg(fmt.Sprintf("Error: %v", err)))
			}
			return
		}

		// After selecting a project, show its commits
		if proj != nil {
			fmt.Println()
			showProjectCommits(proj)
			fmt.Println()
			fmt.Println(infoMsg("Available commands:"))
			fmt.Println(infoMsg("  • vervids commit \"message\" <file.aepx> - Commit a new version"))
			fmt.Println(infoMsg("  • vervids list - List all projects"))
			fmt.Println(infoMsg("  • vervids show <version> - Show version details"))
			fmt.Println(infoMsg("  • vervids help - Show all commands"))
		}
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Run: func(cmd *cobra.Command, args []string) {
		printBoxedHeader()
		if commit != "none" && commit != "" {
			fmt.Printf("Commit: %s\n", commit)
		}
		if date != "unknown" && date != "" {
			fmt.Printf("Built:  %s\n", date)
		}
	},
}

// Define header styles using Lip Gloss
var (
	headerStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(0, 1).
			MarginBottom(1).
			Align(lipgloss.Center).
			Width(36)

	headerTextStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("62")).
			Bold(true)
)

// printBoxedHeader prints a nice boxed header with version info using Lip Gloss
func printBoxedHeader() {
	// Format version string - extract just the version number if it's a git version
	versionStr := version
	if strings.Contains(version, "-") {
		// If it's a git version like "v0.1.0-8-g81e2737-dirty", extract just "v0.1.0"
		parts := strings.Split(version, "-")
		if len(parts) > 0 {
			versionStr = parts[0]
		}
	}

	// Create the header text
	// Check if version already starts with 'v'
	headerText := fmt.Sprintf("🌊 vervids CLI %s", versionStr)
	if !strings.HasPrefix(versionStr, "v") && versionStr != "dev" {
		headerText = fmt.Sprintf("🌊 vervids CLI v%s", versionStr)
	}
	if version == "dev" {
		headerText = "🌊 vervids CLI (dev)"
	}

	// Style the header text and render in box
	styledText := headerTextStyle.Render(headerText)
	box := headerStyle.Render(styledText)
	fmt.Println(box)
}

// Helper functions for styled output (using shared ui package)
func successMsg(msg string) string {
	return ui.Success(msg)
}

func errorMsg(msg string) string {
	return ui.Error(msg)
}

func warningMsg(msg string) string {
	return ui.Warning(msg)
}

func infoMsg(msg string) string {
	return ui.Info(msg)
}

var initCmd = &cobra.Command{
	Use:   "init [path/to/project.aepx]",
	Short: "Initialize version control for an After Effects project",
	Long: `Initialize version control for an .aepx file. This creates a local .vervids config and stores the initial version in Docker.

Docker is required (24.0.0 or newer). Files are stored under /vervids/<projectDir>/vXXX/ in the Docker volume.

If a .vervids directory exists for a different project file, it will be automatically removed.
Use --force to re-initialize the same project file (this will delete existing version history).`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		aepxFilePath := args[0]
		force, _ := cmd.Flags().GetBool("force")

		// Check if file exists
		if _, err := os.Stat(aepxFilePath); os.IsNotExist(err) {
			fmt.Println(errorMsg(fmt.Sprintf("File '%s' does not exist", aepxFilePath)))
			os.Exit(1)
		}

		// Check if it's an .aepx file
		if filepath.Ext(aepxFilePath) != ".aepx" {
			fmt.Println(errorMsg("File must have .aepx extension"))
			fmt.Println(infoMsg("Note: vervids works with .aepx (XML) files, not binary .ae files"))
			os.Exit(1)
		}

		// Get absolute path for comparison
		absPath, err := filepath.Abs(aepxFilePath)
		if err != nil {
			fmt.Println(errorMsg(fmt.Sprintf("Error: %v", err)))
			os.Exit(1)
		}

		// Change to the directory containing the .aepx file
		// This ensures .vervids is created in the same directory as the project file
		aepxDir := filepath.Dir(absPath)
		originalDir, err := os.Getwd()
		if err != nil {
			fmt.Println(errorMsg(fmt.Sprintf("Error getting current directory: %v", err)))
			os.Exit(1)
		}
		
		// Check if we can write to the .aepx file's directory
		if err := os.Chdir(aepxDir); err != nil {
			fmt.Println(errorMsg(fmt.Sprintf("Error: Cannot access directory '%s': %v", aepxDir, err)))
			fmt.Println(infoMsg("This may be a permissions issue. Please ensure you have write access to the directory."))
			os.Exit(1)
		}
		
		// Restore original directory on exit
		defer func() {
			if err := os.Chdir(originalDir); err != nil {
				// Non-fatal, just log
				fmt.Println(warningMsg(fmt.Sprintf("Warning: Could not restore original directory: %v", err)))
			}
		}()

		// Check if already initialized
		if storage.IsInitialized() {
			// Try to load existing project to see if it's for the same file
			existingProj, err := project.Load()
			if err == nil && existingProj != nil {
				// Normalize paths for comparison
				existingPath := existingProj.ProjectPath
				if existingPath == absPath {
					// Same file - user should use commit
					if !force {
						fmt.Println(errorMsg("This project file is already initialized"))
						fmt.Printf("  Existing project: %s\n", existingProj.ProjectName)
						fmt.Println(infoMsg("  Use 'vervids commit \"message\" <file.aepx>' to save new versions"))
						fmt.Println(infoMsg("  Or use 'vervids delete <project-name>' to delete the project and start fresh"))
						os.Exit(1)
					}
				} else {
					// Different file - automatically remove old project
					fmt.Println(warningMsg("Found existing project for a different file"))
					fmt.Printf("  Existing: %s\n", existingProj.ProjectName)
					fmt.Printf("  New:      %s\n", filepath.Base(absPath))
					fmt.Println(infoMsg("  Removing old project to initialize new one..."))
				}
			} else {
				// Can't load existing project - might be corrupted or incomplete
				if !force {
					fmt.Println(warningMsg("Found .vervids directory but couldn't load project"))
					fmt.Println(infoMsg("  Removing it to start fresh..."))
				} else {
					fmt.Println(warningMsg("Force flag detected: removing existing .vervids directory..."))
				}
			}

			// Remove existing .vervids directory
			if err := os.RemoveAll(storage.VerVidsDir); err != nil {
				fmt.Println(errorMsg(fmt.Sprintf("Error removing existing .vervids directory: %v", err)))
				os.Exit(1)
			}
			fmt.Println(successMsg("Removed existing .vervids directory"))
		}

		if err := docker.EnsureDockerReady(); err != nil {
			fmt.Println(errorMsg(fmt.Sprintf("%v", err)))
			os.Exit(1)
		}

		fmt.Println(infoMsg("🚀 Initializing vervids project (Docker storage)..."))
		proj, err := project.Initialize(absPath)
		if err != nil {
			fmt.Println(errorMsg(fmt.Sprintf("Error initializing project: %v", err)))
			os.Exit(1)
		}

		fmt.Println()
		fmt.Println(successMsg("Initialized vervids project"))
		fmt.Printf("%s Project: %s\n", ui.SuccessStyle.Render("✓"), proj.ProjectName)
		fmt.Println(successMsg("Initial version stored (v000)"))

		if len(proj.Versions) > 0 {
			v := proj.Versions[0]
			fmt.Printf("%s Project file: %.2f MB\n", ui.SuccessStyle.Render("✓"), float64(v.Size)/(1024*1024))
			fmt.Printf("%s Assets tracked: %d files\n", ui.SuccessStyle.Render("✓"), v.AssetCount)
			if v.TotalSize > 0 {
				fmt.Printf("%s Total size: %.2f MB\n", ui.SuccessStyle.Render("✓"), float64(v.TotalSize)/(1024*1024))
			}
		}

		fmt.Printf("%s Storage: Docker volume '%s' under /vervids/<project>\n", ui.SuccessStyle.Render("✓"), proj.DockerVolume)

		// Save project context
		configPath := storage.GetConfigPath()
		absConfigPath, err := filepath.Abs(configPath)
		if err != nil {
			absConfigPath = configPath // Fallback to relative path
		}
		context := &storage.ProjectContext{
			ProjectName: proj.ProjectName,
			ConfigPath:  absConfigPath,
		}
		if err := storage.SaveContext(context); err != nil {
			fmt.Println(warningMsg(fmt.Sprintf("Warning: Could not save project context: %v", err)))
		} else {
			fmt.Println(successMsg("Project context saved"))
		}

		fmt.Println()
		fmt.Println(infoMsg("📝 Next steps:"))
		fmt.Println(infoMsg("  • Make changes to your .aepx file or assets"))
		fmt.Println(infoMsg("  • Use 'vervids commit \"message\" <file.aepx>' to save a new version"))
	},
}

var commitCmd = &cobra.Command{
	Use:   "commit [message] [path/to/file.aepx]",
	Short: "Save a new version of your project",
	Long: `Commit the current state of your .aepx file with a message.
This creates a new version with all assets in the Docker storage vault.

The .aepx file path must be provided - typically exported from After Effects.
Example: vervids commit "Added intro animation" "/path/to/exported.aepx"`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		message := args[0]
		aepxFilePath := args[1]

		// Get project from context (already ensured by PersistentPreRunE)
		proj, err := ensureProjectContext()
		if err != nil {
			fmt.Println(errorMsg(fmt.Sprintf("Error: %v", err)))
			os.Exit(1)
		}

		// Change to the directory containing the .vervids config file
		// This ensures we can save the config.json file correctly
		cleanup, err := changeToProjectDirectory()
		if err != nil {
			fmt.Println(errorMsg(fmt.Sprintf("Error: %v", err)))
			fmt.Println(infoMsg("Please ensure you have write access to the directory."))
			os.Exit(1)
		}
		defer cleanup()

		// Validate .aepx file
		if _, err := os.Stat(aepxFilePath); os.IsNotExist(err) {
			fmt.Println(errorMsg(fmt.Sprintf("File '%s' does not exist", aepxFilePath)))
			os.Exit(1)
		}

		if filepath.Ext(aepxFilePath) != ".aepx" {
			fmt.Println(errorMsg("File must have .aepx extension"))
			os.Exit(1)
		}

		// Get absolute path
		absPath, err := filepath.Abs(aepxFilePath)
		if err != nil {
			fmt.Println(errorMsg(fmt.Sprintf("Error: %v", err)))
			os.Exit(1)
		}

		fmt.Println(infoMsg("📦 Creating new version..."))

		// Create new version with the provided .aepx file
		v, err := proj.CommitWithPath(message, absPath)
		if err != nil {
			fmt.Println(errorMsg(fmt.Sprintf("Error committing version: %v", err)))
			os.Exit(1)
		}

		fmt.Println()
		fmt.Println(successMsg(fmt.Sprintf("Committed version %d", v.Number)))
		fmt.Printf("  Message: %s\n", v.Message)
		fmt.Printf("  Time: %s\n", v.Timestamp.Format("2006-01-02 15:04:05"))
		fmt.Printf("  Project file: %.2f MB\n", float64(v.Size)/(1024*1024))
		fmt.Printf("  Assets: %d files\n", v.AssetCount)
		if v.TotalSize > 0 {
			fmt.Printf("  Total size: %.2f MB\n", float64(v.TotalSize)/(1024*1024))
		}

		if proj.UseDocker {
			fmt.Println(infoMsg("  Storage: Docker"))
		} else {
			fmt.Println(infoMsg("  Storage: Local"))
		}
	},
}

var listCmd = &cobra.Command{
	Use:   "list [project-number]",
	Short: "List projects or commits for a project",
	Long: `List all projects stored in Docker. If a project number is provided, show commits for that project.
You can also switch to a different project by selecting it from the list.

Example:
  vervids list              # Show all projects and option to switch
  vervids list 1             # Show commits for project #1`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		projects, err := project.GetAllProjects()
		if err != nil {
			fmt.Println(errorMsg(fmt.Sprintf("Error getting projects: %v", err)))
			os.Exit(1)
		}

		if len(projects) == 0 {
			fmt.Println(infoMsg("No projects found in Docker storage."))
			fmt.Println(infoMsg("Use 'vervids init <file.aepx>' to create a project."))
			return
		}

		// If project number provided, show commits for that project
		if len(args) > 0 {
			projectNum, err := strconv.Atoi(args[0])
			if err != nil {
				fmt.Println(errorMsg("Project number must be an integer"))
				os.Exit(1)
			}
			// Convert from 1-based user input to 0-based array index
			projectIndex := projectNum - 1
			if projectIndex < 0 || projectIndex >= len(projects) {
				fmt.Println(errorMsg(fmt.Sprintf("Project number %d does not exist (1-%d)", projectNum, len(projects))))
				os.Exit(1)
			}

			selectedProj := projects[projectIndex]
			showCommitsForProject(selectedProj.Name)
			return
		}

		// Show current project context if available
		if storage.HasContext() {
			context, err := storage.LoadContext()
			if err == nil {
				if proj, err := project.LoadFromPath(context.ConfigPath); err == nil {
					fmt.Println(infoMsg(fmt.Sprintf("Current project: %s", proj.ProjectName)))
					fmt.Println()
				}
			}
		}

		// Show all projects
		fmt.Println(infoMsg("Projects in Docker storage:"))
		fmt.Println()
		fmt.Println(infoMsg("#   Project Name"))
		fmt.Println(infoMsg("--  ------------------------------"))
		for i, p := range projects {
			// Display 1-based index
			marker := "  "
			if storage.HasContext() {
				context, err := storage.LoadContext()
				if err == nil {
					if proj, err := project.LoadFromPath(context.ConfigPath); err == nil {
						if strings.Contains(strings.ToLower(proj.ProjectName), strings.ToLower(p.Name)) ||
							strings.Contains(strings.ToLower(p.Name), strings.ToLower(proj.ProjectName)) {
							marker = "→ "
						}
					}
				}
			}
			fmt.Printf("%s%s  %s\n", marker, ui.InfoStyle.Render(fmt.Sprintf("%02d", i+1)), p.Name)
		}
		fmt.Println()
		fmt.Println(infoMsg("Use 'vervids list <number>' to see commits for a project"))

		// Offer to switch project
		fmt.Println()
		reader := bufio.NewReader(os.Stdin)
		fmt.Print(infoMsg("Switch to a different project? Enter project number (or press Enter to skip): "))
		input, err := reader.ReadString('\n')
		if err == nil {
			input = strings.TrimSpace(input)
			if input != "" {
				projectNum, err := strconv.Atoi(input)
				if err != nil {
					fmt.Println(errorMsg("Invalid project number"))
					return
				}
				projectIndex := projectNum - 1
				if projectIndex < 0 || projectIndex >= len(projects) {
					fmt.Println(errorMsg(fmt.Sprintf("Project number %d does not exist (1-%d)", projectNum, len(projects))))
					return
				}

				selectedProj := projects[projectIndex]

				// Find the config file for this project using comprehensive search
				configPath, err := findProjectConfigFile(selectedProj.Name)
				if err != nil {
					fmt.Println(errorMsg(fmt.Sprintf("Could not find config file for project: %s", selectedProj.Name)))
					fmt.Println(infoMsg("Tip: Navigate to the project directory, or ensure .vervids/config.json exists."))
					fmt.Println(infoMsg("The project exists in Docker storage, but the local config file is missing."))
					return
				}

				// Load the project
				proj, err := project.LoadFromPath(configPath)
				if err != nil {
					fmt.Println(errorMsg(fmt.Sprintf("Error loading project: %v", err)))
					return
				}

				// Get absolute path for context
				absConfigPath, err := filepath.Abs(configPath)
				if err != nil {
					absConfigPath = configPath
				}

				// Save context
				context := &storage.ProjectContext{
					ProjectName: proj.ProjectName,
					ConfigPath:  absConfigPath,
				}
				if err := storage.SaveContext(context); err != nil {
					fmt.Println(errorMsg(fmt.Sprintf("Error saving context: %v", err)))
					return
				}

				fmt.Println(successMsg(fmt.Sprintf("Switched to project: %s", proj.ProjectName)))
				fmt.Println()
				// Show commits for the newly selected project
				showProjectCommits(proj)
				fmt.Println()
				fmt.Println(infoMsg("Available commands:"))
				fmt.Println(infoMsg("  • vervids commit \"message\" <file.aepx> - Commit a new version"))
				fmt.Println(infoMsg("  • vervids list - List all projects"))
				fmt.Println(infoMsg("  • vervids show <version> - Show version details"))
				fmt.Println(infoMsg("  • vervids help - Show all commands"))
				return
			}
		}

		// If no switch was made, show commits for current project if available
		if storage.HasContext() {
			context, err := storage.LoadContext()
			if err == nil {
				if proj, err := project.LoadFromPath(context.ConfigPath); err == nil {
					fmt.Println()
					showProjectCommits(proj)
					fmt.Println()
					fmt.Println(infoMsg("Available commands:"))
					fmt.Println(infoMsg("  • vervids commit \"message\" <file.aepx> - Commit a new version"))
					fmt.Println(infoMsg("  • vervids list - List all projects"))
					fmt.Println(infoMsg("  • vervids show <version> - Show version details"))
					fmt.Println(infoMsg("  • vervids pull <version> - Pull a version from Docker"))
					fmt.Println(infoMsg("  • vervids help - Show all commands"))
				}
			}
		}
	},
}

// findProjectConfigFile searches for a project's config.json file comprehensively
func findProjectConfigFile(projectName string) (string, error) {
	home := os.Getenv("HOME")
	searchDirs := []string{
		".",
		filepath.Join(home, "Documents"),
		filepath.Join(home, "Desktop"),
		filepath.Join(home, "Projects"),
		filepath.Join(home, "Downloads"),
	}

	// First, try direct search in common locations (one level deep)
	var configPath string
	for _, baseDir := range searchDirs {
		if entries, err := os.ReadDir(baseDir); err == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					potentialConfigPath := filepath.Join(baseDir, entry.Name(), storage.VerVidsDir, storage.ConfigFile)
					if _, err := os.Stat(potentialConfigPath); err == nil {
						if proj, err := project.LoadFromPath(potentialConfigPath); err == nil {
							// Check if this project matches
							projNameLower := strings.ToLower(proj.ProjectName)
							searchNameLower := strings.ToLower(projectName)
							// Remove file extensions for comparison
							projBaseName := strings.TrimSuffix(projNameLower, ".aepx")
							searchBaseName := strings.TrimSuffix(searchNameLower, ".aepx")
							
							if strings.Contains(projBaseName, searchBaseName) ||
								strings.Contains(searchBaseName, projBaseName) ||
								strings.Contains(projNameLower, searchNameLower) ||
								strings.Contains(searchNameLower, projNameLower) {
								configPath = potentialConfigPath
								break
							}
						}
					}
				}
			}
			if configPath != "" {
				break
			}
		}
		// Also check if .vervids exists directly in baseDir
		directConfigPath := filepath.Join(baseDir, storage.VerVidsDir, storage.ConfigFile)
		if _, err := os.Stat(directConfigPath); err == nil {
			if proj, err := project.LoadFromPath(directConfigPath); err == nil {
				projNameLower := strings.ToLower(proj.ProjectName)
				searchNameLower := strings.ToLower(projectName)
				projBaseName := strings.TrimSuffix(projNameLower, ".aepx")
				searchBaseName := strings.TrimSuffix(searchNameLower, ".aepx")
				
				if strings.Contains(projBaseName, searchBaseName) ||
					strings.Contains(searchBaseName, projBaseName) ||
					strings.Contains(projNameLower, searchNameLower) ||
					strings.Contains(searchNameLower, projNameLower) {
					configPath = directConfigPath
					break
				}
			}
		}
	}

	// If not found, try recursive search in Documents and Projects (max depth 3)
	if configPath == "" {
		deepSearchDirs := []string{
			filepath.Join(home, "Documents"),
			filepath.Join(home, "Projects"),
		}
		
		for _, baseDir := range deepSearchDirs {
			if found := findConfigRecursive(baseDir, projectName, 0, 3); found != "" {
				configPath = found
				break
			}
		}
	}

	if configPath == "" {
		return "", fmt.Errorf("could not find config file for project: %s", projectName)
	}

	return configPath, nil
}

// findConfigRecursive recursively searches for config.json files
func findConfigRecursive(dir string, projectName string, depth int, maxDepth int) string {
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
					projNameLower := strings.ToLower(proj.ProjectName)
					searchNameLower := strings.ToLower(projectName)
					projBaseName := strings.TrimSuffix(projNameLower, ".aepx")
					searchBaseName := strings.TrimSuffix(searchNameLower, ".aepx")
					
					if strings.Contains(projBaseName, searchBaseName) ||
						strings.Contains(searchBaseName, projBaseName) ||
						strings.Contains(projNameLower, searchNameLower) ||
						strings.Contains(searchNameLower, projNameLower) {
						return configPath
					}
				}
			}

			// Recurse into subdirectories
			subDir := filepath.Join(dir, entry.Name())
			if found := findConfigRecursive(subDir, projectName, depth+1, maxDepth); found != "" {
				return found
			}
		}
	}

	return ""
}

// selectProject prompts the user to select a project from available projects
func selectProject() (*project.Project, error) {
	projects, err := project.GetAllProjects()
	if err != nil {
		return nil, fmt.Errorf("error getting projects: %w", err)
	}

	if len(projects) == 0 {
		fmt.Println(infoMsg("No projects found."))
		fmt.Println()
		fmt.Println(infoMsg("To get started:"))
		fmt.Println(infoMsg("  • Use 'vervids init <file.aepx>' to initialize a new project"))
		fmt.Println(infoMsg("  • Use 'vervids help' to see all available commands"))
		return nil, fmt.Errorf("no projects available")
	}

	fmt.Println(infoMsg("Select a project to work with:"))
	fmt.Println()
	for i, p := range projects {
		fmt.Printf("  %s  %s\n", ui.InfoStyle.Render(fmt.Sprintf("%d", i+1)), p.Name)
	}
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print(infoMsg("Enter project number (1-" + strconv.Itoa(len(projects)) + "): "))
		input, err := reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("error reading input: %w", err)
		}

		input = strings.TrimSpace(input)
		projectNum, err := strconv.Atoi(input)
		if err != nil || projectNum < 1 || projectNum > len(projects) {
			fmt.Println(errorMsg(fmt.Sprintf("Please enter a number between 1 and %d", len(projects))))
			continue
		}

		selectedProj := projects[projectNum-1]

		// Find the config file for this project using comprehensive search
		configPath, err := findProjectConfigFile(selectedProj.Name)
		if err != nil {
			return nil, err
		}

		// Load the project
		proj, err := project.LoadFromPath(configPath)
		if err != nil {
			return nil, fmt.Errorf("error loading project: %w", err)
		}

		// Get absolute path for context
		absConfigPath, err := filepath.Abs(configPath)
		if err != nil {
			absConfigPath = configPath // Fallback to relative path
		}

		// Save context
		context := &storage.ProjectContext{
			ProjectName: proj.ProjectName,
			ConfigPath:  absConfigPath,
		}
		if err := storage.SaveContext(context); err != nil {
			return nil, fmt.Errorf("error saving context: %w", err)
		}

		fmt.Println(successMsg(fmt.Sprintf("Selected project: %s", proj.ProjectName)))
		return proj, nil
	}
}

// changeToProjectDirectory changes the working directory to the directory containing
// the .vervids config file. Returns a cleanup function to restore the original directory.
func changeToProjectDirectory() (func(), error) {
	context, err := storage.LoadContext()
	if err != nil {
		return nil, fmt.Errorf("error loading project context: %w", err)
	}
	
	// ConfigPath is the full path to .vervids/config.json
	// We need the directory containing .vervids (parent of .vervids)
	configDir := filepath.Dir(filepath.Dir(context.ConfigPath))
	originalDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("error getting current directory: %w", err)
	}
	
	// Change to the directory containing .vervids
	if err := os.Chdir(configDir); err != nil {
		return nil, fmt.Errorf("cannot access directory '%s': %w (this may be a permissions issue)", configDir, err)
	}
	
	// Return cleanup function
	return func() {
		if err := os.Chdir(originalDir); err != nil {
			fmt.Println(warningMsg(fmt.Sprintf("Warning: Could not restore original directory: %v", err)))
		}
	}, nil
}

// ensureProjectContext ensures a project context is set, prompting if needed
func ensureProjectContext() (*project.Project, error) {
	// Check if we have a context
	if storage.HasContext() {
		context, err := storage.LoadContext()
		if err != nil {
			// Context file exists but is invalid, try to select again
			storage.ClearContext()
			return selectProject()
		}

		// Verify the config file still exists
		if _, err := os.Stat(context.ConfigPath); err != nil {
			// Config file doesn't exist, clear context and select again
			storage.ClearContext()
			return selectProject()
		}

		// Load the project
		proj, err := project.LoadFromPath(context.ConfigPath)
		if err != nil {
			// Can't load project, clear context and select again
			storage.ClearContext()
			return selectProject()
		}

		return proj, nil
	}

	// No context, need to select a project
	return selectProject()
}

func init() {
	// Set custom help function to show boxed header
	originalHelpFunc := rootCmd.HelpFunc()
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		// Only show header for root command help
		if cmd == rootCmd {
			printBoxedHeader()
		}
		originalHelpFunc(cmd, args)
	})

	// Add persistent pre-run hook to check for project context
	// Commands that don't need context: init, version, help, list (when listing all), and root (when no subcommand)
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		// Skip context check for root command (handled in its Run function)
		if cmd == rootCmd {
			return nil
		}

		// Skip context check for these commands
		skipContextCommands := []string{"init", "version", "help", "list", "serve"}
		cmdName := cmd.Name()

		// Check if this is one of the skip commands
		for _, skipCmd := range skipContextCommands {
			if cmdName == skipCmd {
				// For list command, only skip if no args (listing all projects)
				if cmdName == "list" && len(args) == 0 {
					return nil
				}
				if cmdName != "list" {
					return nil
				}
			}
		}

		// For other commands, ensure project context
		_, err := ensureProjectContext()
		if err != nil {
			// If error is "no projects available", don't exit - let the command handle it
			if strings.Contains(err.Error(), "no projects available") {
				return nil
			}
			return err
		}

		return nil
	}

	rootCmd.AddCommand(versionCmd)
	initCmd.Flags().BoolP("force", "f", false, "Force re-initialization of the same project file (removes existing version history)")
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(commitCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(showCmd)
	rootCmd.AddCommand(pruneCmd)
	rootCmd.AddCommand(pullCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(serveCmd)
}

func Execute() error {
	return rootCmd.Execute()
}

// showCommitsForProject finds and displays commits for a project by name
func showCommitsForProject(projectName string) {
	// First try: look in current directory
	if storage.IsInitialized() {
		proj, err := project.Load()
		if err == nil {
			// Check if this project's directory name matches
			cwd, _ := os.Getwd()
			if strings.Contains(filepath.Base(cwd), projectName) {
				showProjectCommits(proj)
				return
			}
		}
	}

	// Use comprehensive search to find the config file
	configPath, err := findProjectConfigFile(projectName)
	if err != nil {
		fmt.Println(errorMsg(fmt.Sprintf("Could not find config.json for project '%s'", projectName)))
		fmt.Println(infoMsg("Tip: Navigate to the project directory, or ensure .vervids/config.json exists."))
		os.Exit(1)
	}

	proj, err := project.LoadFromPath(configPath)
	if err != nil {
		fmt.Println(errorMsg(fmt.Sprintf("Error loading project: %v", err)))
		os.Exit(1)
	}

	showProjectCommits(proj)
}

// showProjectCommits displays commits for a loaded project
func showProjectCommits(proj *project.Project) {

	if len(proj.Versions) == 0 {
		fmt.Printf("%s: %s\n", ui.InfoStyle.Render("Project"), proj.ProjectName)
		fmt.Println(infoMsg("No commits yet. Use 'vervids commit \"message\" <file.aepx>' to create one."))
		return
	}

	fmt.Printf("%s: %s\n", ui.InfoStyle.Render("Project"), proj.ProjectName)
	fmt.Printf("%s: %d\n\n", ui.InfoStyle.Render("Commits"), len(proj.Versions))
	fmt.Println(infoMsg("#   Time                 Size(MB)  Assets  Message"))
	fmt.Println(infoMsg("--  -------------------  -------  ------  ------------------------------"))
	for _, v := range proj.Versions {
		fmt.Printf("%02d  %s  %7.2f  %6d  %s\n",
			v.Number,
			v.Timestamp.Format("2006-01-02 15:04:05"),
			float64(v.Size)/(1024*1024),
			v.AssetCount,
			v.Message,
		)
	}
}

var showCmd = &cobra.Command{
	Use:   "show [version-number]",
	Short: "Show details for a specific version",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Get project from context (already ensured by PersistentPreRunE)
		proj, err := ensureProjectContext()
		if err != nil {
			fmt.Println(errorMsg(fmt.Sprintf("Error: %v", err)))
			os.Exit(1)
		}

		var num int
		if _, err := fmt.Sscanf(args[0], "%d", &num); err != nil {
			fmt.Println(errorMsg("Version-number must be an integer (e.g., 0, 1, 2)"))
			os.Exit(1)
		}

		v, err := proj.GetVersion(num)
		if err != nil {
			fmt.Println(errorMsg(fmt.Sprintf("%v", err)))
			os.Exit(1)
		}

		fmt.Printf("%s Version:   %d\n", ui.InfoStyle.Render("Version:"), v.Number)
		fmt.Printf("%s Message:   %s\n", ui.InfoStyle.Render("Message:"), v.Message)
		fmt.Printf("%s Time:      %s\n", ui.InfoStyle.Render("Time:"), v.Timestamp.Format("2006-01-02 15:04:05"))
		fmt.Printf("%s Proj Size: %.2f MB\n", ui.InfoStyle.Render("Proj Size:"), float64(v.Size)/(1024*1024))
		fmt.Printf("%s Assets:    %d files\n", ui.InfoStyle.Render("Assets:"), v.AssetCount)
		if v.DockerPath != "" {
			fmt.Printf("%s Docker:    %s\n", ui.InfoStyle.Render("Docker:"), v.DockerPath)
		}
		if len(v.Assets) > 0 {
			fmt.Println()
			fmt.Println(infoMsg("Assets:"))
			for _, a := range v.Assets {
				fmt.Printf("  - %s (%s)  %.2f MB\n", a.Filename, a.Extension, float64(a.Size)/(1024*1024))
			}
		}
	},
}

var pruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Remove commits whose storage is missing in Docker",
	Run: func(cmd *cobra.Command, args []string) {
		// Get project from context (already ensured by PersistentPreRunE)
		proj, err := ensureProjectContext()
		if err != nil {
			fmt.Println(errorMsg(fmt.Sprintf("Error: %v", err)))
			os.Exit(1)
		}
		
		// Change to the directory containing the .vervids config file
		cleanup, err := changeToProjectDirectory()
		if err != nil {
			fmt.Println(errorMsg(fmt.Sprintf("Error: %v", err)))
			os.Exit(1)
		}
		defer cleanup()
		
		if err := docker.EnsureDockerReady(); err != nil {
			fmt.Println(errorMsg(fmt.Sprintf("%v", err)))
			os.Exit(1)
		}
		removed, err := proj.PruneMissingDockerVersions()
		if err != nil {
			fmt.Println(errorMsg(fmt.Sprintf("Error pruning: %v", err)))
			os.Exit(1)
		}
		if removed == 0 {
			fmt.Println(successMsg("Nothing to prune; all versions present in Docker"))
		} else {
			fmt.Println(successMsg(fmt.Sprintf("Pruned %d missing version(s)", removed)))
		}
	},
}

var pullCmd = &cobra.Command{
	Use:   "pull <version> [output-dir]",
	Short: "Pull a version from Docker storage to local filesystem",
	Long: `Pull a specific version from Docker storage to your local filesystem.
The .aepx file and all assets will be copied. If assets don't exist at their
original paths, they will be copied from Docker storage and the .aepx file
will be updated to reference the new asset locations.

Requires a project to be selected. Use 'vervids list' to select a project.

Example:
  vervids pull 2              # Pull version 2 to current directory
  vervids pull 1 ./restored   # Pull version 1 to ./restored directory`,
	Args: cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		// Get project from context (already ensured by PersistentPreRunE)
		proj, err := ensureProjectContext()
		if err != nil {
			if strings.Contains(err.Error(), "no projects available") {
				fmt.Println(errorMsg("No projects available. Use 'vervids init <file.aepx>' to create a project first."))
			} else {
				fmt.Println(errorMsg(fmt.Sprintf("Error: %v", err)))
			}
			os.Exit(1)
		}

		if proj == nil {
			fmt.Println(errorMsg("No project selected. Use 'vervids list' to select a project."))
			os.Exit(1)
		}

		// Parse version number
		versionNum, err := strconv.Atoi(args[0])
		if err != nil {
			fmt.Println(errorMsg("Version must be a number"))
			os.Exit(1)
		}

		// Get output directory (default to current directory)
		outputDir := "."
		if len(args) > 1 {
			outputDir = args[1]
		}

		// Convert to absolute path
		absOutputDir, err := filepath.Abs(outputDir)
		if err != nil {
			fmt.Println(errorMsg(fmt.Sprintf("Error getting absolute path: %v", err)))
			os.Exit(1)
		}

		fmt.Println(infoMsg(fmt.Sprintf("📦 Pulling version %d...", versionNum)))

		// Pull the version
		restoredPath, err := proj.RestoreVersion(versionNum, absOutputDir)
		if err != nil {
			fmt.Println(errorMsg(fmt.Sprintf("Error pulling version: %v", err)))
			os.Exit(1)
		}

		fmt.Println()
		fmt.Println(successMsg(fmt.Sprintf("✓ Successfully pulled version %d", versionNum)))
		fmt.Printf("  Project file: %s\n", restoredPath)

		// Check if assets directory exists (only show if assets were copied)
		assetsDir := filepath.Join(absOutputDir, "assets")
		if _, err := os.Stat(assetsDir); err == nil {
			fmt.Printf("  Assets directory: %s\n", assetsDir)
		}
	},
}

var deleteCmd = &cobra.Command{
	Use:   "delete <project-name>",
	Short: "Delete a project and all its data",
	Long: `Delete removes the project from Docker storage (including all versions and assets).

⚠️  WARNING: This action cannot be undone! All versions, assets, and project history will be permanently deleted.

Example:
  vervids delete myproject`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		projectName := args[0]

		// Ensure Docker is ready
		if err := docker.EnsureDockerReady(); err != nil {
			fmt.Println(errorMsg(fmt.Sprintf("%v", err)))
			os.Exit(1)
		}

		// Get all projects to find the one to delete
		projects, err := project.GetAllProjects()
		if err != nil {
			fmt.Println(errorMsg(fmt.Sprintf("Error getting projects: %v", err)))
			os.Exit(1)
		}

		// Find project by name (case-insensitive partial match)
		var targetProject *project.ProjectInfo
		for i, p := range projects {
			// Exact match or partial match
			if strings.EqualFold(p.Name, projectName) ||
				strings.Contains(strings.ToLower(p.Name), strings.ToLower(projectName)) {
				targetProject = &projects[i]
				break
			}
		}

		if targetProject == nil {
			fmt.Println(errorMsg(fmt.Sprintf("Project '%s' not found", projectName)))
			fmt.Println()
			fmt.Println(infoMsg("Available projects:"))
			for _, p := range projects {
				fmt.Printf("  %s %s\n", ui.InfoStyle.Render("•"), p.Name)
			}
			os.Exit(1)
		}

		// Show project info
		fmt.Printf("%s Project: %s\n", ui.InfoStyle.Render("Project:"), targetProject.Name)
		fmt.Printf("%s Path: %s\n", ui.InfoStyle.Render("Path:"), targetProject.DockerPath)
		fmt.Println()

		// Confirmation prompt
		fmt.Print(warningMsg("WARNING: This will permanently delete all project data!\n"))
		fmt.Print(infoMsg("Type 'DELETE' to confirm: "))

		reader := bufio.NewReader(os.Stdin)
		confirmation, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println(errorMsg(fmt.Sprintf("Error reading input: %v", err)))
			os.Exit(1)
		}

		confirmation = strings.TrimSpace(confirmation)
		if confirmation != "DELETE" {
			fmt.Println(errorMsg("Deletion cancelled (confirmation did not match)"))
			os.Exit(1)
		}

		// Delete project
		fmt.Println()
		fmt.Println(infoMsg("🗑️  Deleting project..."))

		if err := project.DeleteProjectByName(targetProject.Name, targetProject.DockerPath); err != nil {
			fmt.Println(errorMsg(fmt.Sprintf("Error deleting project: %v", err)))
			os.Exit(1)
		}

		fmt.Println(successMsg("Project deleted successfully"))
		fmt.Println(successMsg("  • All versions removed from Docker"))
		fmt.Println(successMsg("  • All assets removed from Docker"))
		fmt.Println(successMsg("  • Local .vervids directory removed (if found)"))
	},
}

var serveCmd = &cobra.Command{
	Use:   "serve [port]",
	Short: "Start the HTTP API server for plugin access",
	Long: `Start an HTTP API server that exposes vervids data for plugins.

The server provides REST endpoints:
  GET /api/projects - List all projects with their IDs
  GET /api/projects/{id}/commits - Get commits for a specific project
  GET /health - Health check endpoint

Default port is 8080 if not specified.

Example:
  vervids serve        # Start server on port 8080
  vervids serve 3000   # Start server on port 3000`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		port := 8080
		if len(args) > 0 {
			p, err := strconv.Atoi(args[0])
			if err != nil {
				fmt.Println(errorMsg(fmt.Sprintf("Invalid port number: %v", err)))
				os.Exit(1)
			}
			if p < 1 || p > 65535 {
				fmt.Println(errorMsg("Port must be between 1 and 65535"))
				os.Exit(1)
			}
			port = p
		}

		printBoxedHeader()
		fmt.Println()

		if err := api.StartServer(port); err != nil {
			fmt.Println(errorMsg(fmt.Sprintf("Failed to start server: %v", err)))
			os.Exit(1)
		}
	},
}

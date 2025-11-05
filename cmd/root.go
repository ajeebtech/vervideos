package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ajeebtech/vervideos/internal/docker"
	"github.com/ajeebtech/vervideos/internal/project"
	"github.com/ajeebtech/vervideos/internal/storage"
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
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("vervids %s (commit: %s, built: %s)\n", version, commit, date)
	},
}

var initCmd = &cobra.Command{
	Use:   "init [path/to/project.aepx]",
	Short: "Initialize version control for an After Effects project",
    Long: `Initialize version control for an .aepx file. This creates a local .vervids config and stores the initial version in Docker.

Docker is required (24.0.0 or newer). Files are stored under /vervids/<projectDir>/vXXX/ in the Docker volume.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		aepxFilePath := args[0]

		// Check if file exists
		if _, err := os.Stat(aepxFilePath); os.IsNotExist(err) {
			fmt.Printf("❌ Error: File '%s' does not exist\n", aepxFilePath)
			os.Exit(1)
		}

		// Check if it's an .aepx file
		if filepath.Ext(aepxFilePath) != ".aepx" {
			fmt.Printf("❌ Error: File must have .aepx extension\n")
			fmt.Println("Note: vervids works with .aepx (XML) files, not binary .ae files")
			os.Exit(1)
		}

		// Check if already initialized
		if storage.IsInitialized() {
			fmt.Println("❌ Error: Project already initialized (.vervids directory exists)")
			fmt.Println("Use 'vervids commit' to save new versions")
			os.Exit(1)
		}

		// Initialize project
		absPath, err := filepath.Abs(aepxFilePath)
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			os.Exit(1)
		}

        if err := docker.EnsureDockerReady(); err != nil {
            fmt.Printf("❌ %v\n", err)
            os.Exit(1)
        }

        fmt.Println("🚀 Initializing vervids project (Docker storage)...")
        proj, err := project.Initialize(absPath)
		if err != nil {
			fmt.Printf("❌ Error initializing project: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("\n✓ Initialized vervids project")
		fmt.Printf("✓ Project: %s\n", proj.ProjectName)
		fmt.Printf("✓ Initial version stored (v000)\n")
		
		if len(proj.Versions) > 0 {
			v := proj.Versions[0]
			fmt.Printf("✓ Project file: %.2f MB\n", float64(v.Size)/(1024*1024))
			fmt.Printf("✓ Assets tracked: %d files\n", v.AssetCount)
			if v.TotalSize > 0 {
				fmt.Printf("✓ Total size: %.2f MB\n", float64(v.TotalSize)/(1024*1024))
			}
		}

        fmt.Printf("✓ Storage: Docker volume '%s' under /vervids/<project>\n", proj.DockerVolume)

		fmt.Println("\n📝 Next steps:")
        fmt.Println("  • Make changes to your .aepx file or assets")
        fmt.Println("  • Use 'vervids commit \"message\"' to save a new version")
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

		// Check if initialized
		if !storage.IsInitialized() {
			fmt.Println("❌ Error: Not a vervids project")
			fmt.Println("Run 'vervids init <file.aepx>' first")
			os.Exit(1)
		}

		// Validate .aepx file
		if _, err := os.Stat(aepxFilePath); os.IsNotExist(err) {
			fmt.Printf("❌ Error: File '%s' does not exist\n", aepxFilePath)
			os.Exit(1)
		}

		if filepath.Ext(aepxFilePath) != ".aepx" {
			fmt.Printf("❌ Error: File must have .aepx extension\n")
			os.Exit(1)
		}

		// Load project
		proj, err := project.Load()
		if err != nil {
			fmt.Printf("❌ Error loading project: %v\n", err)
			os.Exit(1)
		}

		// Get absolute path
		absPath, err := filepath.Abs(aepxFilePath)
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("📦 Creating new version...")

		// Create new version with the provided .aepx file
		v, err := proj.CommitWithPath(message, absPath)
		if err != nil {
			fmt.Printf("❌ Error committing version: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("\n✓ Committed version %d\n", v.Number)
		fmt.Printf("  Message: %s\n", v.Message)
		fmt.Printf("  Time: %s\n", v.Timestamp.Format("2006-01-02 15:04:05"))
		fmt.Printf("  Project file: %.2f MB\n", float64(v.Size)/(1024*1024))
		fmt.Printf("  Assets: %d files\n", v.AssetCount)
		if v.TotalSize > 0 {
			fmt.Printf("  Total size: %.2f MB\n", float64(v.TotalSize)/(1024*1024))
		}
		
		if proj.UseDocker {
			fmt.Println("  Storage: Docker")
		} else {
			fmt.Println("  Storage: Local")
		}
	},
}

var listCmd = &cobra.Command{
	Use:   "list [project-number]",
	Short: "List projects or commits for a project",
	Long: `List all projects stored in Docker. If a project number is provided, show commits for that project.

Example:
  vervids list              # Show all projects
  vervids list 0             # Show commits for project #0`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		projects, err := project.GetAllProjects()
		if err != nil {
			fmt.Printf("❌ Error getting projects: %v\n", err)
			os.Exit(1)
		}

		if len(projects) == 0 {
			fmt.Println("No projects found in Docker storage.")
			fmt.Println("Use 'vervids init <file.aepx>' to create a project.")
			return
		}

		// If project number provided, show commits for that project
		if len(args) > 0 {
			projectNum, err := strconv.Atoi(args[0])
			if err != nil {
				fmt.Printf("❌ Error: project number must be an integer\n")
				os.Exit(1)
			}
			if projectNum < 0 || projectNum >= len(projects) {
				fmt.Printf("❌ Error: project number %d does not exist (0-%d)\n", projectNum, len(projects)-1)
				os.Exit(1)
			}

			selectedProj := projects[projectNum]
			showCommitsForProject(selectedProj.Name)
			return
		}

		// Show all projects
		fmt.Println("Projects in Docker storage:")
		fmt.Println()
		fmt.Println("#   Project Name")
		fmt.Println("--  ------------------------------")
		for i, p := range projects {
			fmt.Printf("%02d  %s\n", i, p.Name)
		}
		fmt.Println()
		fmt.Println("Use 'vervids list <number>' to see commits for a project")
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(commitCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(showCmd)
	rootCmd.AddCommand(pruneCmd)
	rootCmd.AddCommand(deleteCmd)
}

func Execute() error {
	return rootCmd.Execute()
}

// showCommitsForProject finds and displays commits for a project by name
func showCommitsForProject(projectName string) {
	// Search for config.json files that match this project
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

	// Search common locations
	home := os.Getenv("HOME")
	searchDirs := []string{
		".",
		filepath.Join(home, "Documents"),
		filepath.Join(home, "Desktop"),
		filepath.Join(home, "Projects"),
	}

	var proj *project.Project
	for _, baseDir := range searchDirs {
		// Look for directories matching project name
		if entries, err := os.ReadDir(baseDir); err == nil {
			for _, entry := range entries {
				if entry.IsDir() && strings.Contains(entry.Name(), projectName) {
					configPath := filepath.Join(baseDir, entry.Name(), storage.VerVidsDir, storage.ConfigFile)
					if _, err := os.Stat(configPath); err == nil {
						if loaded, err := project.LoadFromPath(configPath); err == nil {
							proj = loaded
							break
						}
					}
				}
			}
			if proj != nil {
				break
			}
		}
		// Also try .vervids directly in baseDir
		configPath := filepath.Join(baseDir, storage.VerVidsDir, storage.ConfigFile)
		if _, err := os.Stat(configPath); err == nil {
			if loaded, err := project.LoadFromPath(configPath); err == nil {
				if strings.Contains(filepath.Base(baseDir), projectName) || 
				   strings.Contains(loaded.ProjectName, projectName) {
					proj = loaded
					break
				}
			}
		}
	}

	if proj == nil {
		fmt.Printf("❌ Could not find config.json for project '%s'\n", projectName)
		fmt.Println("Tip: Navigate to the project directory, or ensure .vervids/config.json exists.")
		os.Exit(1)
	}

	showProjectCommits(proj)
}

// showProjectCommits displays commits for a loaded project
func showProjectCommits(proj *project.Project) {

	if len(proj.Versions) == 0 {
		fmt.Printf("Project: %s\n", proj.ProjectName)
		fmt.Println("No commits yet. Use 'vervids commit \"message\"' to create one.")
		return
	}

	fmt.Printf("Project: %s\n", proj.ProjectName)
	fmt.Printf("Commits: %d\n\n", len(proj.Versions))
	fmt.Println("#   Time                 Size(MB)  Assets  Message")
	fmt.Println("--  -------------------  -------  ------  ------------------------------")
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
        if !storage.IsInitialized() {
            fmt.Println("❌ Error: Not a vervids project")
            fmt.Println("Run 'vervids init <file.aepx>' first")
            os.Exit(1)
        }

        var num int
        if _, err := fmt.Sscanf(args[0], "%d", &num); err != nil {
            fmt.Println("❌ Error: version-number must be an integer (e.g., 0, 1, 2)")
            os.Exit(1)
        }

        proj, err := project.Load()
        if err != nil {
            fmt.Printf("❌ Error loading project: %v\n", err)
            os.Exit(1)
        }

        v, err := proj.GetVersion(num)
        if err != nil {
            fmt.Printf("❌ %v\n", err)
            os.Exit(1)
        }

        fmt.Printf("Version:   %d\n", v.Number)
        fmt.Printf("Message:   %s\n", v.Message)
        fmt.Printf("Time:      %s\n", v.Timestamp.Format("2006-01-02 15:04:05"))
        fmt.Printf("Proj Size: %.2f MB\n", float64(v.Size)/(1024*1024))
        fmt.Printf("Assets:    %d files\n", v.AssetCount)
        if v.DockerPath != "" {
            fmt.Printf("Docker:    %s\n", v.DockerPath)
        }
        if len(v.Assets) > 0 {
            fmt.Println()
            fmt.Println("Assets:")
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
        if !storage.IsInitialized() {
            fmt.Println("❌ Error: Not a vervids project")
            fmt.Println("Run 'vervids init <file.aepx>' first")
            os.Exit(1)
        }
        if err := docker.EnsureDockerReady(); err != nil {
            fmt.Printf("❌ %v\n", err)
            os.Exit(1)
        }
        proj, err := project.Load()
        if err != nil {
            fmt.Printf("❌ Error loading project: %v\n", err)
            os.Exit(1)
        }
        removed, err := proj.PruneMissingDockerVersions()
        if err != nil {
            fmt.Printf("❌ Error pruning: %v\n", err)
            os.Exit(1)
        }
        if removed == 0 {
            fmt.Println("✓ Nothing to prune; all versions present in Docker")
        } else {
            fmt.Printf("✓ Pruned %d missing version(s)\n", removed)
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
			fmt.Printf("❌ %v\n", err)
			os.Exit(1)
		}

		// Get all projects to find the one to delete
		projects, err := project.GetAllProjects()
		if err != nil {
			fmt.Printf("❌ Error getting projects: %v\n", err)
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
			fmt.Printf("❌ Error: Project '%s' not found\n", projectName)
			fmt.Println("\nAvailable projects:")
			for _, p := range projects {
				fmt.Printf("  • %s\n", p.Name)
			}
			os.Exit(1)
		}

		// Show project info
		fmt.Printf("Project: %s\n", targetProject.Name)
		fmt.Printf("Path: %s\n", targetProject.DockerPath)
		fmt.Println()

		// Confirmation prompt
		fmt.Print("⚠️  WARNING: This will permanently delete all project data!\n")
		fmt.Print("Type 'DELETE' to confirm: ")
		
		reader := bufio.NewReader(os.Stdin)
		confirmation, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("❌ Error reading input: %v\n", err)
			os.Exit(1)
		}

		confirmation = strings.TrimSpace(confirmation)
		if confirmation != "DELETE" {
			fmt.Println("❌ Deletion cancelled (confirmation did not match)")
			os.Exit(1)
		}

		// Delete project
		fmt.Println()
		fmt.Println("🗑️  Deleting project...")
		
		if err := project.DeleteProjectByName(targetProject.Name, targetProject.DockerPath); err != nil {
			fmt.Printf("❌ Error deleting project: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("✓ Project deleted successfully")
		fmt.Println("  • All versions removed from Docker")
		fmt.Println("  • All assets removed from Docker")
	},
}



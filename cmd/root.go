package cmd

import (
	"fmt"
	"os"
	"path/filepath"

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

var (
	useDocker bool
)

var initCmd = &cobra.Command{
	Use:   "init [path/to/project.aepx]",
	Short: "Initialize version control for an After Effects project",
	Long: `Initialize version control for an .aepx file. This creates a .vervids directory
and stores the initial version of your project along with all referenced assets.

Example: 
  vervids init "new project.aepx"
  vervids init --docker "new project.aepx"  # Use Docker for storage`,
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

		fmt.Println("🚀 Initializing vervids project...")
		if useDocker {
			fmt.Println("📦 Using Docker for storage")
		}

		proj, err := project.Initialize(absPath, useDocker)
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

		if useDocker {
			fmt.Printf("✓ Storage: Docker volume '%s'\n", proj.DockerVolume)
		} else {
			fmt.Println("✓ Storage: Local (.vervids directory)")
		}

		fmt.Println("\n📝 Next steps:")
		fmt.Println("  • Make changes to your .aepx file or assets")
		fmt.Println("  • Use 'vervids commit \"message\"' to save a new version")
	},
}

var commitCmd = &cobra.Command{
	Use:   "commit [message]",
	Short: "Save a new version of your project",
	Long: `Commit the current state of your .aepx file with a message.
This creates a new version with all assets in the storage vault.

Example: vervids commit "Added intro animation"`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		message := args[0]

		// Check if initialized
		if !storage.IsInitialized() {
			fmt.Println("❌ Error: Not a vervids project")
			fmt.Println("Run 'vervids init <file.aepx>' first")
			os.Exit(1)
		}

		// Load project
		proj, err := project.Load()
		if err != nil {
			fmt.Printf("❌ Error loading project: %v\n", err)
			os.Exit(1)
		}

		// Check if the .aepx file exists
		if _, err := os.Stat(proj.ProjectPath); os.IsNotExist(err) {
			fmt.Printf("❌ Error: Project file '%s' not found\n", proj.ProjectPath)
			os.Exit(1)
		}

		fmt.Println("📦 Creating new version...")

		// Create new version
		v, err := proj.Commit(message)
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

func init() {
	// Add --docker flag to init command
	initCmd.Flags().BoolVarP(&useDocker, "docker", "d", false, "Use Docker for storage")
	
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(commitCmd)
}

func Execute() error {
	return rootCmd.Execute()
}


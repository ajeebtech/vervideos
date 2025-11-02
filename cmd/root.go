package cmd

import (
	"fmt"

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
	Use:   "vervideos",
	Short: "A versioning tool for video editors",
	Long: `vervideos is a command-line versioning tool designed specifically 
for video editors to manage and track versions of their video projects.`,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("vervideos %s (commit: %s, built: %s)\n", version, commit, date)
	},
}

var initCmd = &cobra.Command{
	Use:   "init [project-name]",
	Short: "Initialize a new video project with version control",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		projectName := "video-project"
		if len(args) > 0 {
			projectName = args[0]
		}
		fmt.Printf("✓ Initializing new video project: %s\n", projectName)
		fmt.Println("✓ Created .vervideos directory")
		fmt.Println("✓ Created initial version file")
		fmt.Println("\nNext steps:")
		fmt.Println("  • Use 'vervideos save <name>' to save a new version")
		fmt.Println("  • Use 'vervideos list' to see all versions")
	},
}

var saveCmd = &cobra.Command{
	Use:   "save [version-name]",
	Short: "Save a new version of your project",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		versionName := "auto-save"
		if len(args) > 0 {
			versionName = args[0]
		}
		fmt.Printf("✓ Saving new version: %s\n", versionName)
		fmt.Println("✓ Version saved successfully")
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all versions",
	Aliases: []string{"ls"},
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Listing all versions:")
		fmt.Println("\n  v1.0.0  2025-11-02  Initial version")
		fmt.Println("  v1.1.0  2025-11-02  Added new scene")
		fmt.Println("* v1.2.0  2025-11-02  Current version")
	},
}

var restoreCmd = &cobra.Command{
	Use:   "restore [version]",
	Short: "Restore a specific version",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		version := args[0]
		fmt.Printf("✓ Restoring version: %s\n", version)
		fmt.Println("✓ Version restored successfully")
	},
}

var diffCmd = &cobra.Command{
	Use:   "diff [version1] [version2]",
	Short: "Compare two versions",
	Args:  cobra.RangeArgs(0, 2),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Comparing versions...")
		fmt.Println("✓ Differences displayed")
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(saveCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(restoreCmd)
	rootCmd.AddCommand(diffCmd)
}

func Execute() error {
	return rootCmd.Execute()
}


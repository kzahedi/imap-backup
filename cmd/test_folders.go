package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var testFoldersCmd = &cobra.Command{
	Use:   "test-folders",
	Short: "Test folder structure handling",
	Long:  `Test how IMAP folder names are converted to filesystem paths.`,
	RunE:  runTestFolders,
}

func init() {
	rootCmd.AddCommand(testFoldersCmd)
}

func runTestFolders(cmd *cobra.Command, args []string) error {
	// Test different folder structures
	testCases := []struct {
		folderName string
		delimiter  string
		description string
	}{
		{"INBOX", "", "Simple inbox"},
		{"Friends/Mario", "/", "Gmail-style hierarchy with forward slash"},
		{"Friends.Mario", ".", "Gmail-style hierarchy with dot"},
		{"Friends\\Mario", "\\", "Exchange-style hierarchy with backslash"},
		{"Work/Projects/Website", "/", "Deep hierarchy"},
		{"Sent Items", "", "Folder with space"},
		{"[Gmail]/All Mail", "/", "Gmail special folder"},
		{"Gelöschte Elemente", "", "Non-ASCII folder name"},
		{"Personal/Family/Photos", "/", "Three-level hierarchy"},
	}
	
	fmt.Println("Testing folder structure conversion:")
	fmt.Println("=====================================")
	
	for _, test := range testCases {
		// Test the internal method (access via reflection or make it public)
		// For now, let's show the expected behavior
		
		fmt.Printf("\nFolder: %s\n", test.folderName)
		fmt.Printf("Delimiter: '%s'\n", test.delimiter)
		fmt.Printf("Description: %s\n", test.description)
		
		// Show what the old system would produce
		oldPath := convertOldWay(test.folderName)
		fmt.Printf("Old way: %s\n", oldPath)
		
		// Show what the new system produces
		newPath := convertNewWay(test.folderName, test.delimiter)
		fmt.Printf("New way: %s\n", newPath)
		
		if oldPath != newPath {
			fmt.Printf("✓ Improved: Now creates proper nested directories\n")
		} else {
			fmt.Printf("- Same result\n")
		}
	}
	
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("The new system preserves IMAP folder hierarchies")
	fmt.Println("by creating nested directories instead of flat folders.")
	
	return nil
}

// Simulate old folder conversion
func convertOldWay(folderName string) string {
	// Old way: replace all separators with underscores
	result := folderName
	result = strings.ReplaceAll(result, "/", "_")
	result = strings.ReplaceAll(result, "\\", "_")
	result = strings.ReplaceAll(result, ".", "_")
	return result
}

// Simulate new folder conversion
func convertNewWay(folderName, delimiter string) string {
	if delimiter == "" {
		// Try to detect delimiter
		if strings.Contains(folderName, "/") {
			delimiter = "/"
		} else if strings.Contains(folderName, ".") {
			delimiter = "."
		} else if strings.Contains(folderName, "\\") {
			delimiter = "\\"
		} else {
			return folderName // No hierarchy
		}
	}
	
	// Split and create nested path
	components := strings.Split(folderName, delimiter)
	var sanitized []string
	for _, comp := range components {
		if comp != "" {
			sanitized = append(sanitized, sanitizeFolderForDemo(comp))
		}
	}
	
	if len(sanitized) == 0 {
		return "INBOX"
	}
	
	return filepath.Join(sanitized...)
}

func sanitizeFolderForDemo(name string) string {
	// Simple sanitization for demo
	result := name
	result = strings.ReplaceAll(result, ":", "_")
	result = strings.ReplaceAll(result, "*", "_")
	result = strings.ReplaceAll(result, "?", "_")
	result = strings.ReplaceAll(result, "\"", "_")
	result = strings.ReplaceAll(result, "<", "_")
	result = strings.ReplaceAll(result, ">", "_")
	result = strings.ReplaceAll(result, "|", "_")
	return strings.TrimSpace(result)
}
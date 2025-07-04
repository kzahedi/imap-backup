package cmd

import (
	"fmt"
	"imap-backup/internal/charset"

	"github.com/spf13/cobra"
)

var testCharsetCmd = &cobra.Command{
	Use:   "test-charset [charset-name]",
	Short: "Test charset support",
	Long:  `Test if a specific charset is supported by the tool.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runTestCharset,
}

func init() {
	rootCmd.AddCommand(testCharsetCmd)
}

func runTestCharset(cmd *cobra.Command, args []string) error {
	charsetName := args[0]
	
	fmt.Printf("Testing charset: %s\n", charsetName)
	
	if charset.IsSupported(charsetName) {
		fmt.Printf("✓ Charset '%s' is supported\n", charsetName)
		
		// Test decoding with a simple string
		testString := "Hëllö Wörld"
		decoded, err := charset.DecodeString(testString, charsetName)
		if err != nil {
			fmt.Printf("⚠ Warning: Failed to decode test string: %v\n", err)
		} else {
			fmt.Printf("✓ Test decode successful: %s\n", decoded)
		}
	} else {
		fmt.Printf("✗ Charset '%s' is not supported\n", charsetName)
		
		// Show some supported charsets
		fmt.Println("\nSupported charsets include:")
		supportedCharsets := []string{
			"iso-8859-1", "iso-8859-15", "windows-1252", 
			"utf-8", "utf-16", "gb2312", "big5", "shift_jis",
		}
		
		for _, supported := range supportedCharsets {
			if charset.IsSupported(supported) {
				fmt.Printf("  - %s\n", supported)
			}
		}
	}
	
	return nil
}
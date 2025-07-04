package cmd

import (
	"context"
	"fmt"
	"imap-backup/internal/backup"
	"imap-backup/internal/config"

	"github.com/spf13/cobra"
)

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Backup IMAP email accounts",
	Long: `Backup emails from IMAP accounts to local storage.
Preserves folder structure, read/unread status, and extracts attachments.`,
	RunE: runBackup,
}

func init() {
	rootCmd.AddCommand(backupCmd)
	backupCmd.Flags().StringP("account", "a", "", "specific account to backup (default: all configured accounts)")
	backupCmd.Flags().BoolP("dry-run", "d", false, "show what would be backed up without actually downloading")
	backupCmd.Flags().IntP("max-concurrent", "c", 5, "maximum concurrent connections per account")
	backupCmd.Flags().Bool("ignore-charset-errors", false, "continue backup even when charset parsing fails")
}

func runBackup(cmd *cobra.Command, args []string) error {
	// Create context for the backup operation
	ctx := context.Background()
	
	// Get output directory from root command flags
	outputDir, _ := cmd.Root().Flags().GetString("output")
	if outputDir == "" {
		outputDir = "./backup"
	}
	account, _ := cmd.Flags().GetString("account")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	maxConcurrent, _ := cmd.Flags().GetInt("max-concurrent")
	ignoreCharsetErrors, _ := cmd.Flags().GetBool("ignore-charset-errors")
	verbose, _ := cmd.Root().Flags().GetBool("verbose")

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	backupConfig := backup.Config{
		OutputDir:           outputDir,
		Account:             account,
		DryRun:              dryRun,
		MaxConcurrent:       maxConcurrent,
		IgnoreCharsetErrors: ignoreCharsetErrors,
		Verbose:             verbose,
	}

	backupService := backup.NewService(cfg)
	return backupService.Run(ctx, backupConfig)
}
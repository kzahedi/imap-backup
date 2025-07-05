package cmdutil

import "github.com/spf13/cobra"

// CommonFlags represents commonly used command flags
type CommonFlags struct {
	Verbose     bool
	DryRun      bool
	Output      string
	Config      string
	MaxWorkers  int
	ShowPasswords bool
}

// AddVerboseFlag adds a verbose flag to the command
func AddVerboseFlag(cmd *cobra.Command) {
	cmd.Flags().BoolP("verbose", "v", false, "verbose output")
}

// AddDryRunFlag adds a dry-run flag to the command
func AddDryRunFlag(cmd *cobra.Command) {
	cmd.Flags().BoolP("dry-run", "d", false, "show what would be done without actually doing it")
}

// AddOutputFlag adds an output directory flag to the command
func AddOutputFlag(cmd *cobra.Command, defaultValue string) {
	cmd.Flags().StringP("output", "o", defaultValue, "output directory")
}

// AddConfigFlag adds a config file flag to the command
func AddConfigFlag(cmd *cobra.Command) {
	cmd.Flags().String("config", "", "config file path")
}

// AddMaxWorkersFlag adds a max workers flag to the command
func AddMaxWorkersFlag(cmd *cobra.Command, defaultValue int) {
	cmd.Flags().IntP("max-concurrent", "c", defaultValue, "maximum concurrent connections")
}

// AddShowPasswordsFlag adds a show passwords flag to the command
func AddShowPasswordsFlag(cmd *cobra.Command) {
	cmd.Flags().BoolP("show-passwords", "p", false, "show passwords (use with caution)")
}

// AddCommonFlags adds the most commonly used flags to a command
func AddCommonFlags(cmd *cobra.Command) {
	AddVerboseFlag(cmd)
	AddConfigFlag(cmd)
}

// AddBackupFlags adds flags commonly used by backup commands
func AddBackupFlags(cmd *cobra.Command) {
	AddCommonFlags(cmd)
	AddDryRunFlag(cmd)
	AddOutputFlag(cmd, "./backup")
	AddMaxWorkersFlag(cmd, 5)
}

// AddAccountFlags adds flags commonly used by account commands
func AddAccountFlags(cmd *cobra.Command) {
	AddCommonFlags(cmd)
	AddShowPasswordsFlag(cmd)
}

// GetCommonFlags extracts common flag values from a command
func GetCommonFlags(cmd *cobra.Command) CommonFlags {
	verbose, _ := cmd.Flags().GetBool("verbose")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	output, _ := cmd.Flags().GetString("output")
	config, _ := cmd.Flags().GetString("config")
	maxWorkers, _ := cmd.Flags().GetInt("max-concurrent")
	showPasswords, _ := cmd.Flags().GetBool("show-passwords")

	return CommonFlags{
		Verbose:       verbose,
		DryRun:        dryRun,
		Output:        output,
		Config:        config,
		MaxWorkers:    maxWorkers,
		ShowPasswords: showPasswords,
	}
}
package cmd

import (
	"github.com/spf13/cobra"
)

type args struct {
	version    string
	LogLevel   string
	ConfigPath string
	TextFormat bool
}

// InitCommands initializes and returns the root command for the application.
func InitCommands(version string) *cobra.Command {
	args := &args{
		version: version,
	}

	cmd := &cobra.Command{
		Use:   "mitbot",
		Short: "Make It Public Telegram Bot",
		Long:  "Make It Public Telegram Bot is a bot for managing your accounts and tokens.",
	}

	cmd.PersistentFlags().StringVar(&args.ConfigPath, "config", "", "config file path")
	cmd.PersistentFlags().StringVar(&args.LogLevel, "loglevel", "info", "log level (debug, info, warn, error)")
	cmd.PersistentFlags().BoolVar(&args.TextFormat, "logtext", false, "log in text format, otherwise JSON")

	return cmd
}

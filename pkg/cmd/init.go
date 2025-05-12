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
	arg := &args{
		version: version,
	}

	cmd := &cobra.Command{
		Use:   "mitbot",
		Short: "Make It Public Telegram tg",
		Long:  "Make It Public Telegram tg is a bot for managing your accounts and tokens.",
	}

	cmd.AddCommand(initRunCommand(arg))

	cmd.PersistentFlags().StringVar(&arg.ConfigPath, "config", "", "config file path")
	cmd.PersistentFlags().StringVar(&arg.LogLevel, "loglevel", "info", "log level (debug, info, warn, error)")
	cmd.PersistentFlags().BoolVar(&arg.TextFormat, "logtext", false, "log in text format, otherwise JSON")

	return cmd
}

func initRunCommand(arg *args) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run the bot",
		Long:  "Run the bot with the specified configuration.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runBot(cmd.Context(), arg)
		},
	}

	return cmd
}

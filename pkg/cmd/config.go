package cmd

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/ksysoev/make-it-public-tgbot/pkg/bot"
	"github.com/ksysoev/make-it-public-tgbot/pkg/prov"
	"github.com/ksysoev/make-it-public-tgbot/pkg/repo"
	"github.com/spf13/viper"
)

type appConfig struct {
	Bot  bot.Config  `mapstructure:"bot"`
	MIT  prov.Config `mapstructure:"mit"`
	Repo repo.Config `mapstructure:"repo"`
}

// loadConfig loads the application configuration using the provided arguments and environment variables.
// It returns a pointer to appConfig or an error if loading or unmarshalling fails.
func loadConfig(arg *args) (*appConfig, error) {
	v := viper.NewWithOptions(viper.ExperimentalBindStruct())

	if arg.ConfigPath != "" {
		v.SetConfigFile(arg.ConfigPath)

		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
	}

	var cfg appConfig

	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	slog.Debug("Config loaded", slog.Any("config", cfg))

	return &cfg, nil
}

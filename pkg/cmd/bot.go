package cmd

import (
	"context"
	"fmt"

	"github.com/ksysoev/make-it-public-tgbot/pkg/bot"
)

func runBot(ctx context.Context, arg *args) error {
	if err := initLogger(arg); err != nil {
		return fmt.Errorf("failed to init logger: %w", err)
	}

	cfg, err := loadConfig(arg)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	b, err := bot.New(&cfg.Bot)
	if err != nil {
		return fmt.Errorf("failed to create bot: %w", err)
	}

	return b.Run(ctx)
}

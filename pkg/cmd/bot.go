package cmd

import (
	"context"
	"fmt"

	"github.com/ksysoev/make-it-public-tgbot/pkg/bot"
)

func runBot(ctx context.Context, args *args) error {
	if err := initLogger(args); err != nil {
		return fmt.Errorf("failed to init logger: %w", err)
	}

	cfg, err := loadConfig(args)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	bot := bot.New(&cfg.Bot)

	slog.InfoContext(ctx, "revclient started", "server", args.Server)

	return revcli.Run(ctx)
}

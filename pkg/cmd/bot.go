package cmd

import (
	"context"
	"fmt"

	"github.com/ksysoev/make-it-public-tgbot/pkg/bot"
	"github.com/ksysoev/make-it-public-tgbot/pkg/core"
	"github.com/ksysoev/make-it-public-tgbot/pkg/prov"
	"github.com/ksysoev/make-it-public-tgbot/pkg/repo"
)

// runBot is the entry point to initialize and run the bot application with the provided context and arguments.
// It configures logging, loads the configuration, initializes dependencies, and starts the bot runtime loop.
// Returns an error if any initialization or runtime operation fails.
func runBot(ctx context.Context, arg *args) error {
	if err := initLogger(arg); err != nil {
		return fmt.Errorf("failed to init logger: %w", err)
	}

	cfg, err := loadConfig(arg)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	userRepo := repo.New(cfg.Repo)
	MITProv := prov.New(cfg.MIT)
	tokeSvc := core.New(userRepo, MITProv)

	b, err := bot.New(&cfg.Bot, tokeSvc)
	if err != nil {
		return fmt.Errorf("failed to create bot: %w", err)
	}

	return b.Run(ctx)
}

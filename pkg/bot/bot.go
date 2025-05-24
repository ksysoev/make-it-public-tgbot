package bot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"
	"github.com/ksysoev/make-it-public-tgbot/pkg/core"
)

const (
	requestTimeout = 3 * time.Second
)

// tgClient interface represents the Telegram bot API capabilities we use
type tgClient interface {
	Send(c tgbotapi.Chattable) (tgbotapi.Message, error)
	StopReceivingUpdates()
	GetUpdatesChan(config tgbotapi.UpdateConfig) tgbotapi.UpdatesChannel
}

// Config holds the configuration for the Telegram bot
type Config struct {
	TelegramToken string `mapstructure:"token"`
}

type TokenService interface {
	CreateToken(ctx context.Context, userID string) (*core.APIToken, error)
	RevokeToken(ctx context.Context, userID string) error
}

type Service struct {
	token    string
	tg       tgClient
	tokenSvc TokenService
	handler  Handler
}

// New initializes a new Service with the given configuration and returns an error if the configuration is invalid.
func New(cfg *Config, tokenSvc TokenService) (*Service, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if cfg.TelegramToken == "" {
		return nil, fmt.Errorf("telegram token cannot be empty")
	}

	bot, err := tgbotapi.NewBotAPI(cfg.TelegramToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create Telegram bot: %w", err)
	}

	s := &Service{
		token:    cfg.TelegramToken,
		tg:       bot,
		tokenSvc: tokenSvc,
	}

	s.handler = s.setupHandler()

	return s, nil
}

func (s *Service) processUpdate(ctx context.Context, update *tgbotapi.Update) {
	if update.Message == nil {
		return
	}

	// nolint:staticcheck // don't want to have dependency on cmd package here for now
	ctx = context.WithValue(ctx, "chat_id", fmt.Sprintf("%d", update.Message.Chat.ID))

	msg := update.Message

	var wg sync.WaitGroup
	defer wg.Wait()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	wg.Add(1)
	defer wg.Done() // Ensure wg.Done() is called when the function returns

	msgConfig, err := s.handler.Handle(ctx, msg)

	if errors.Is(err, context.Canceled) {
		slog.InfoContext(ctx, "Request cancelled",
			slog.Int64("chat_id", msg.Chat.ID),
		)

		return
	} else if err != nil {
		slog.ErrorContext(ctx, "Unexpected error",
			slog.Any("error", err),
		)
		return
	}

	// Skip sending if message is empty
	if msgConfig.Text == "" {
		return
	}
	cancel()

	// Send response
	if _, err := s.tg.Send(msgConfig); err != nil {
		slog.ErrorContext(ctx, "Failed to send message",
			slog.Any("error", err),
		)
	}
}

func (s *Service) Run(ctx context.Context) error {
	slog.InfoContext(ctx, "Starting Telegram bot")

	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 30

	updates := s.tg.GetUpdatesChan(updateConfig)

	var wg sync.WaitGroup

	for {
		select {
		case update, ok := <-updates:
			if !ok {
				return nil
			}

			wg.Add(1)

			go func() {
				defer wg.Done()

				reqCtx, cancel := context.WithTimeout(ctx, requestTimeout)

				// nolint:staticcheck // don't want to have dependecy on cmd package here for now
				reqCtx = context.WithValue(reqCtx, "req_id", uuid.New().String())

				defer cancel()

				s.processUpdate(reqCtx, &update)
			}()

		case <-ctx.Done():
			slog.Info("Starting graceful shutdown")
			s.tg.StopReceivingUpdates()

			// Wait for ongoing message processors with a timeout
			done := make(chan struct{})
			go func() {
				wg.Wait()
				close(done)
			}()

			select {
			case <-done:
				slog.InfoContext(ctx, "Graceful shutdown completed")
			case <-time.After(requestTimeout):
				slog.Warn("Graceful shutdown timed out after 30 seconds")
			}

			return nil
		}
	}
}

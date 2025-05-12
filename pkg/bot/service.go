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
)

const (
	requestTimeout = 3 * time.Second
)

// BotAPI interface represents the Telegram bot API capabilities we use
type BotAPI interface {
	Send(c tgbotapi.Chattable) (tgbotapi.Message, error)
	StopReceivingUpdates()
	GetUpdatesChan(config tgbotapi.UpdateConfig) tgbotapi.UpdatesChannel
}

// Config holds the configuration for the Telegram bot
type Config struct {
	TelegramToken string `mapstructure:"telegram_token"`
}

type ServiceImpl struct {
	token string
	Bot   BotAPI
}

// NewService creates a new bot service with the given configuration and AI provider
func NewService(cfg *Config) (*ServiceImpl, error) {
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

	return &ServiceImpl{
		token: cfg.TelegramToken,
		Bot:   bot,
	}, nil
}

func (s *ServiceImpl) processUpdate(ctx context.Context, update *tgbotapi.Update) {
	if update.Message == nil {
		return
	}

	// nolint:staticcheck // don't want to have dependecy on cmd package here for now
	ctx = context.WithValue(ctx, "chat_id", fmt.Sprintf("%d", update.Message.Chat.ID))

	msg := update.Message

	var wg sync.WaitGroup
	defer wg.Wait()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	wg.Add(1)

	msgConfig, err := handleMessage(ctx, msg)

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
	if _, err := s.Bot.Send(msgConfig); err != nil {
		slog.ErrorContext(ctx, "Failed to send message",
			slog.Any("error", err),
		)
	}
}

func (s *ServiceImpl) Run(ctx context.Context) error {
	slog.InfoContext(ctx, "Starting Telegram bot")

	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 30

	updates := s.Bot.GetUpdatesChan(updateConfig)

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
			s.Bot.StopReceivingUpdates()

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

func handleMessage(_ context.Context, msg *tgbotapi.Message) (tgbotapi.MessageConfig, error) {
	// Handle the message here
	// For example, you can send a reply back to the user
	reply := tgbotapi.NewMessage(msg.Chat.ID, "Hello! You said: "+msg.Text)
	return reply, nil
}

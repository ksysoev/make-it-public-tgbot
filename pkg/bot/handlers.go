package bot

import (
	"context"
	"errors"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/ksysoev/make-it-public-tgbot/pkg/bot/middleware"
	"github.com/ksysoev/make-it-public-tgbot/pkg/core"
)

const (
	welcomeMessage        = "👋 Welcome to Make It Public Bot!\n\nI help you manage API tokens for https://make-it-public.dev - a service that allows you to securely publish services hidden behind NAT.\n\nUse /help to see available commands."
	helpMessage           = "Available Commands:\n\n/start - Show welcome message\n/help - Display this help message\n/new_token - Generate a new API token\n\nAbout Make It Public:\nMake It Public allows you to securely expose services that are behind NAT or firewalls to the internet."
	unknownCommandMessage = "❓ Unknown command.\n\nUse /help to see the list of available commands."
	tokenExistsMessage    = "⚠️ You already have an active API token. You can create a new one after your current token expires."
	notCommandMessage     = "I can only respond to commands. Try /help to see what I can do."
	tokenRevokedMessage   = "🔒 Your API token has been successfully revoked.\n\nYou can create a new one using /new_token command."
)

// Handler defines the interface for processing and responding to incoming messages in a Telegram bot context.
// It handles a message by performing necessary processing and returns the configuration for the outgoing message or an error.
// ctx is the context for managing request lifecycle and cancellation.
// message is the incoming Telegram message to be processed.
// Returns a configured message object for sending a response and an error if processing fails.
type Handler interface {
	Handle(ctx context.Context, message *tgbotapi.Message) (tgbotapi.MessageConfig, error)
}

// setupHandler initializes and configures the request handler with specified middleware components.
// It applies middleware for request reduction, concurrency throttling, metric collection, and error handling,
// ensuring proper management of requests and enhanced error messages.
// Returns a Handler that processes messages with the applied middleware stack.
func (s *Service) setupHandler() Handler {
	h := middleware.Use(
		s,
		middleware.WithThrottler(30),
		middleware.WithRequestSequencer(),
		middleware.WithMetrics(),
		middleware.WithErrorHandling(),
	)

	return h
}

func (s *Service) Handle(ctx context.Context, msg *tgbotapi.Message) (tgbotapi.MessageConfig, error) {
	if msg.Command() != "" {
		resp, err := s.handleCommand(ctx, msg)
		if err != nil {
			return tgbotapi.MessageConfig{}, fmt.Errorf("failed to handle command: %w", err)
		}

		return resp, nil
	}

	return tgbotapi.NewMessage(msg.Chat.ID, notCommandMessage), nil
}

func (s *Service) handleCommand(ctx context.Context, msg *tgbotapi.Message) (tgbotapi.MessageConfig, error) {
	switch msg.Command() {
	case "start":
		return tgbotapi.NewMessage(msg.Chat.ID, welcomeMessage), nil
	case "help":
		return tgbotapi.NewMessage(msg.Chat.ID, helpMessage), nil
	case "new_token":
		resp, err := s.tokenSvc.CreateToken(ctx, fmt.Sprintf("%d", msg.From.ID))
		switch {
		case errors.Is(err, core.ErrMaxTokensExceeded):
			return tgbotapi.NewMessage(msg.Chat.ID, tokenExistsMessage), nil
		case err != nil:
			return tgbotapi.MessageConfig{}, fmt.Errorf("failed to create token: %w", err)
		default:
			message := tgbotapi.NewMessage(msg.Chat.ID, resp.Message)

			return message, nil
		}
	case "revoke_token":
		if err := s.tokenSvc.RevokeToken(ctx, fmt.Sprintf("%d", msg.From.ID)); err != nil {
			return tgbotapi.MessageConfig{}, fmt.Errorf("failed to revoke token: %w", err)
		}

		return tgbotapi.NewMessage(msg.Chat.ID, tokenRevokedMessage), nil
	default:
		return tgbotapi.NewMessage(msg.Chat.ID, unknownCommandMessage), nil
	}
}

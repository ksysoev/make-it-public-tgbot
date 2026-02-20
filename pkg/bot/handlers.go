package bot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/ksysoev/make-it-public-tgbot/pkg/bot/middleware"
	"github.com/ksysoev/make-it-public-tgbot/pkg/core"
)

const (
	welcomeMessage = `👋 Welcome to Make It Public Bot!

I help you manage API tokens for https://make-it-public.dev - a service that allows you to securely publish services hidden behind NAT.

Use /help to see available commands.`
	helpMessage = `Available Commands:

/start - Show welcome message
/help - Display this help message
/new_token - Generate a new API token (up to 3)
/my_tokens - List your active API tokens
/revoke_token - Revoke an API token
/cancel - Cancel the current question

About Make It Public:
Make It Public allows you to securely expose services that are behind NAT or firewalls to the internet.`
	unknownCommandMessage  = "❓ Unknown command.\n\nUse /help to see the list of available commands."
	notCommandMessage      = "I can only respond to commands. Try /help to see what I can do."
	tokenRevokedMessage    = "🔒 Your API token has been successfully revoked.\n\nYou can create a new one using /new_token command."
	noTokenToRevokeMessage = "❌ You don't have an active API token to revoke.\n\nUse /new_token to create one."
	noTokensMessage        = "❌ You don't have any active API tokens.\n\nUse /new_token to create one."
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

// Handle processes incoming telegram messages, handles commands, text messages, and generates appropriate responses.
func (s *Service) Handle(ctx context.Context, msg *tgbotapi.Message) (tgbotapi.MessageConfig, error) {
	slog.DebugContext(ctx, "Handling message", slog.Any("message", msg))
	if msg.Command() != "" {
		resp, err := s.handleCommand(ctx, msg)
		if err != nil {
			return tgbotapi.MessageConfig{}, fmt.Errorf("failed to handle command: %w", err)
		}

		return resp, nil
	}

	if msg.Text == "" {
		return tgbotapi.NewMessage(msg.Chat.ID, notCommandMessage), nil
	}

	resp, err := s.tokenSvc.HandleMessage(ctx, fmt.Sprintf("%d", msg.From.ID), msg.Text)
	if err != nil {
		return tgbotapi.MessageConfig{}, fmt.Errorf("failed to handle text message: %w", err)
	}

	return newMessage(msg.Chat.ID, resp), nil
}

// handleCommand handles Telegram command messages and generates an appropriate response based on the command received.
func (s *Service) handleCommand(ctx context.Context, msg *tgbotapi.Message) (tgbotapi.MessageConfig, error) {
	userID := fmt.Sprintf("%d", msg.From.ID)

	switch msg.Command() {
	case "start":
		if err := s.tokenSvc.ResetConversation(ctx, userID); err != nil {
			slog.ErrorContext(ctx, "Failed to reset conversation on start", slog.Any("error", err))
		}

		return newTextMessage(msg.Chat.ID, welcomeMessage), nil
	case "help":
		return newTextMessage(msg.Chat.ID, helpMessage), nil
	case "new_token":
		resp, err := s.tokenSvc.CreateToken(ctx, userID)
		if err != nil {
			return tgbotapi.MessageConfig{}, fmt.Errorf("failed to create token: %w", err)
		}

		return newMessage(msg.Chat.ID, resp), nil
	case "my_tokens":
		resp, err := s.tokenSvc.ListTokens(ctx, userID)

		switch {
		case errors.Is(err, core.ErrTokenNotFound):
			return newTextMessage(msg.Chat.ID, noTokensMessage), nil
		case err != nil:
			return tgbotapi.MessageConfig{}, fmt.Errorf("failed to list tokens: %w", err)
		default:
			return newMessage(msg.Chat.ID, resp), nil
		}
	case "revoke_token":
		resp, err := s.tokenSvc.RevokeToken(ctx, userID)

		switch {
		case errors.Is(err, core.ErrTokenNotFound):
			return newTextMessage(msg.Chat.ID, noTokenToRevokeMessage), nil
		case err != nil:
			return tgbotapi.MessageConfig{}, fmt.Errorf("failed to revoke token: %w", err)
		case resp != nil:
			// Multi-token case: a conversation was started to select which token to revoke.
			return newMessage(msg.Chat.ID, resp), nil
		default:
			// Single-token case: revoked directly.
			return newTextMessage(msg.Chat.ID, tokenRevokedMessage), nil
		}
	case "cancel":
		if err := s.tokenSvc.ResetConversation(ctx, userID); err != nil {
			return tgbotapi.MessageConfig{}, fmt.Errorf("failed to reset conversation: %w", err)
		}

		return newTextMessage(msg.Chat.ID, "Conversation has been reset. You can start over with /new_token."), nil
	default:
		return newTextMessage(msg.Chat.ID, unknownCommandMessage), nil
	}
}

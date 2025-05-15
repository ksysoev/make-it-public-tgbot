package bot

import (
	"context"
	"errors"
	"fmt"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/ksysoev/make-it-public-tgbot/pkg/core"
)

const (
	welcomeMessage        = "👋 Welcome to Make It Public Bot!\n\nI help you manage API tokens for https://make-it-public.dev - a service that allows you to securely publish services hidden behind NAT.\n\nUse /help to see available commands."
	helpMessage           = "Available Commands:\n\n/start - Show welcome message\n/help - Display this help message\n/new_token - Generate a new API token\n\nAbout Make It Public:\nMake It Public allows you to securely expose services that are behind NAT or firewalls to the internet."
	unknownCommandMessage = "❓ Unknown command.\n\nUse /help to see the list of available commands."
	tokenCreatedMessage   = "🔑 Your New API Token\n\n`%s`\n\n⏱ Valid until: %s\n\nKeep this token secure and don't share it with others."
	tokenExistsMessage    = "⚠️ You already have an active API token. You can create a new one after your current token expires."
	notCommandMessage     = "I can only respond to commands. Try /help to see what I can do."
)

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
		token, err := s.tokenSvc.CreateToken(ctx, fmt.Sprintf("%d", msg.From.ID))
		switch {
		case errors.Is(err, core.ErrMaxTokensExceeded):
			return tgbotapi.NewMessage(msg.Chat.ID, tokenExistsMessage), nil
		case err != nil:
			return tgbotapi.MessageConfig{}, fmt.Errorf("failed to create token: %w", err)
		default:
			expiresAt := time.Now().Add(time.Duration(token.ExpiresIn) * time.Second).Format(time.DateTime)
			tokenMsg := fmt.Sprintf(tokenCreatedMessage, token.Token, expiresAt)
			message := tgbotapi.NewMessage(msg.Chat.ID, tokenMsg)

			return message, nil
		}
	default:
		return tgbotapi.NewMessage(msg.Chat.ID, unknownCommandMessage), nil
	}
}

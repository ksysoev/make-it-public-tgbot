package bot

import (
	"context"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var errUnsupportedCommand = fmt.Errorf("unsupported command")

func (s *Service) Handle(ctx context.Context, msg *tgbotapi.Message) (tgbotapi.MessageConfig, error) {
	if msg.Command() != "" {
		resp, err := s.handleCommand(ctx, msg)
		if err != nil {
			return tgbotapi.MessageConfig{}, fmt.Errorf("failed to handle command: %w", err)
		}

		return resp, nil
	}

	return tgbotapi.MessageConfig{}, errUnsupportedCommand
}

func (s *Service) handleCommand(ctx context.Context, msg *tgbotapi.Message) (tgbotapi.MessageConfig, error) {
	switch msg.Command() {
	case "start":
		return tgbotapi.NewMessage(msg.Chat.ID, ""), nil
	case "new_token":
		token, err := s.tokenSvc.CreateToken(ctx, fmt.Sprintf("%d", msg.From.ID))
		if err != nil {
			return tgbotapi.MessageConfig{}, fmt.Errorf("failed to create token: %w", err)
		}
		return tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("Your new token: %s", token.Token)), nil
	default:
		return tgbotapi.NewMessage(msg.Chat.ID, "Unsupported command"), nil
	}
}

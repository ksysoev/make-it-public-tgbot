package bot

import (
	"context"
	"errors"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/ksysoev/make-it-public-tgbot/pkg/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name:    "nil config",
			cfg:     nil,
			wantErr: true,
		},
		{
			name:    "empty token",
			cfg:     &Config{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.cfg, &MockTokenService{})
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestHandle(t *testing.T) {
	mockTokenSvc := NewMockTokenService(t)
	svc := &Service{
		token:    "test-token",
		tg:       NewMocktgClient(t),
		tokenSvc: mockTokenSvc,
	}

	tests := []struct {
		name       string
		message    *tgbotapi.Message
		setupMocks func()
		wantText   string
		wantErr    bool
	}{
		{
			name: "start command",
			message: &tgbotapi.Message{
				Text: "/start",
				Entities: []tgbotapi.MessageEntity{
					{
						Type:   "bot_command",
						Offset: 0,
						Length: 6,
					},
				},
				Chat: &tgbotapi.Chat{
					ID: 123,
				},
			},
			setupMocks: func() {},
			wantText:   welcomeMessage,
			wantErr:    false,
		},
		{
			name: "help command",
			message: &tgbotapi.Message{
				Text: "/help",
				Entities: []tgbotapi.MessageEntity{
					{
						Type:   "bot_command",
						Offset: 0,
						Length: 5,
					},
				},
				Chat: &tgbotapi.Chat{
					ID: 123,
				},
			},
			setupMocks: func() {},
			wantText:   helpMessage,
			wantErr:    false,
		},
		{
			name: "new_token command - success",
			message: &tgbotapi.Message{
				Text: "/new_token",
				Entities: []tgbotapi.MessageEntity{
					{
						Type:   "bot_command",
						Offset: 0,
						Length: 10,
					},
				},
				Chat: &tgbotapi.Chat{
					ID: 123,
				},
				From: &tgbotapi.User{
					ID: 456,
				},
			},
			setupMocks: func() {
				token := &core.APIToken{
					KeyID:     "key123",
					Token:     "token123",
					ExpiresIn: 3600,
				}
				mockTokenSvc.EXPECT().CreateToken(mock.Anything, "456").Return(token, nil)
			},
			wantErr: false,
		},
		{
			name: "new_token command - token exists",
			message: &tgbotapi.Message{
				Text: "/new_token",
				Entities: []tgbotapi.MessageEntity{
					{
						Type:   "bot_command",
						Offset: 0,
						Length: 10,
					},
				},
				Chat: &tgbotapi.Chat{
					ID: 123,
				},
				From: &tgbotapi.User{
					ID: 456,
				},
			},
			setupMocks: func() {
				mockTokenSvc.EXPECT().CreateToken(mock.Anything, "456").Return(nil, core.ErrMaxTokensExceeded)
			},
			wantText: tokenExistsMessage,
			wantErr:  false,
		},
		{
			name: "new_token command - error",
			message: &tgbotapi.Message{
				Text: "/new_token",
				Entities: []tgbotapi.MessageEntity{
					{
						Type:   "bot_command",
						Offset: 0,
						Length: 10,
					},
				},
				Chat: &tgbotapi.Chat{
					ID: 123,
				},
				From: &tgbotapi.User{
					ID: 456,
				},
			},
			setupMocks: func() {
				mockTokenSvc.EXPECT().CreateToken(mock.Anything, "456").Return(nil, errors.New("some error"))
			},
			wantErr: true,
		},
		{
			name: "unknown command",
			message: &tgbotapi.Message{
				Text: "/unknown",
				Entities: []tgbotapi.MessageEntity{
					{
						Type:   "bot_command",
						Offset: 0,
						Length: 8,
					},
				},
				Chat: &tgbotapi.Chat{
					ID: 123,
				},
			},
			setupMocks: func() {},
			wantText:   unknownCommandMessage,
			wantErr:    false,
		},
		{
			name: "not a command",
			message: &tgbotapi.Message{
				Text: "hello",
				Chat: &tgbotapi.Chat{
					ID: 123,
				},
			},
			setupMocks: func() {},
			wantText:   notCommandMessage,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTokenSvc.ExpectedCalls = nil
			tt.setupMocks()

			msg, err := svc.Handle(context.Background(), tt.message)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			if tt.wantText != "" {
				assert.Equal(t, tt.wantText, msg.Text)
			}
		})
	}
}

func TestProcessUpdate(t *testing.T) {
	mockTg := NewMocktgClient(t)
	mockTokenSvc := NewMockTokenService(t)

	cfg := &Config{
		TelegramToken: "test-token",
	}

	svc := &Service{
		token:    cfg.TelegramToken,
		tg:       mockTg,
		tokenSvc: mockTokenSvc,
	}

	svc.handler = svc

	tests := []struct {
		name       string
		update     *tgbotapi.Update
		setupMocks func()
	}{
		{
			name: "nil message",
			update: &tgbotapi.Update{
				Message: nil,
			},
			setupMocks: func() {},
		},
		{
			name: "valid message",
			update: &tgbotapi.Update{
				Message: &tgbotapi.Message{
					Text: "/start",
					Entities: []tgbotapi.MessageEntity{
						{
							Type:   "bot_command",
							Offset: 0,
							Length: 6,
						},
					},
					Chat: &tgbotapi.Chat{
						ID: 123,
					},
				},
			},
			setupMocks: func() {
				mockTg.EXPECT().Send(mock.Anything).Return(tgbotapi.Message{}, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTg.ExpectedCalls = nil
			mockTokenSvc.ExpectedCalls = nil
			tt.setupMocks()

			svc.processUpdate(context.Background(), tt.update)
		})
	}
}

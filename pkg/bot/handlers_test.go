package bot

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/ksysoev/make-it-public-tgbot/pkg/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestSetupHandler(t *testing.T) {
	// Create a service with mocked dependencies
	mockTokenSvc := NewMockTokenService(t)
	svc := &Service{
		token:    "test-token",
		tg:       NewMocktgClient(t),
		tokenSvc: mockTokenSvc,
	}

	// Call setupHandler
	handler := svc.setupHandler()

	// Verify that the handler is not nil
	assert.NotNil(t, handler, "Handler should not be nil")
}

func TestHandleCommand(t *testing.T) {
	tests := []struct {
		name       string
		command    string
		setupMocks func(mockTokenSvc *MockTokenService)
		chatID     int64
		userID     int64
		wantText   string
		wantErr    bool
	}{
		{
			name:    "start command",
			command: "start",
			setupMocks: func(mockTokenSvc *MockTokenService) {
				// No mocks needed for start command
			},
			chatID:   123,
			userID:   456,
			wantText: welcomeMessage,
			wantErr:  false,
		},
		{
			name:    "help command",
			command: "help",
			setupMocks: func(mockTokenSvc *MockTokenService) {
				// No mocks needed for help command
			},
			chatID:   123,
			userID:   456,
			wantText: helpMessage,
			wantErr:  false,
		},
		{
			name:    "new_token command - success",
			command: "new_token",
			setupMocks: func(mockTokenSvc *MockTokenService) {
				response := &core.Response{
					Message: "🔑 Your New API Token\n\ntoken123\n\n⏱ Valid until: 2023-01-01 12:00:00\n\nKeep this token secure and don't share it with others.",
				}
				mockTokenSvc.EXPECT().CreateToken(mock.Anything, "456").Return(response, nil)
			},
			chatID:  123,
			userID:  456,
			wantErr: false,
		},
		{
			name:    "new_token command - token exists",
			command: "new_token",
			setupMocks: func(mockTokenSvc *MockTokenService) {
				response := &core.Response{
					Message: "You already have an active API token. Do you want to regenerate it?",
					Answers: []string{"Yes", "No"},
				}
				mockTokenSvc.EXPECT().CreateToken(mock.Anything, "456").Return(response, nil)
			},
			chatID:   123,
			userID:   456,
			wantText: "You already have an active API token. Do you want to regenerate it?",
			wantErr:  false,
		},
		{
			name:    "new_token command - error",
			command: "new_token",
			setupMocks: func(mockTokenSvc *MockTokenService) {
				mockTokenSvc.EXPECT().CreateToken(mock.Anything, "456").Return(nil, errors.New("some error"))
			},
			chatID:  123,
			userID:  456,
			wantErr: true,
		},
		{
			name:    "revoke_token command - success",
			command: "revoke_token",
			setupMocks: func(mockTokenSvc *MockTokenService) {
				mockTokenSvc.EXPECT().RevokeToken(mock.Anything, "456").Return(nil)
			},
			chatID:   123,
			userID:   456,
			wantText: tokenRevokedMessage,
			wantErr:  false,
		},
		{
			name:    "revoke_token command - no token to revoke",
			command: "revoke_token",
			setupMocks: func(mockTokenSvc *MockTokenService) {
				mockTokenSvc.EXPECT().RevokeToken(mock.Anything, "456").Return(core.ErrTokenNotFound)
			},
			chatID:   123,
			userID:   456,
			wantText: noTokenToRevokeMessage,
			wantErr:  false,
		},
		{
			name:    "revoke_token command - error",
			command: "revoke_token",
			setupMocks: func(mockTokenSvc *MockTokenService) {
				mockTokenSvc.EXPECT().RevokeToken(mock.Anything, "456").Return(errors.New("revoke error"))
			},
			chatID:  123,
			userID:  456,
			wantErr: true,
		},
		{
			name:    "unknown command",
			command: "unknown",
			setupMocks: func(mockTokenSvc *MockTokenService) {
				// No mocks needed for unknown command
			},
			chatID:   123,
			userID:   456,
			wantText: unknownCommandMessage,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a service with mocked dependencies
			mockTokenSvc := NewMockTokenService(t)
			svc := &Service{
				token:    "test-token",
				tg:       NewMocktgClient(t),
				tokenSvc: mockTokenSvc,
			}

			// Setup mocks
			tt.setupMocks(mockTokenSvc)

			// Create a message with the command
			msg := &tgbotapi.Message{
				Text: "/" + tt.command,
				Entities: []tgbotapi.MessageEntity{
					{
						Type:   "bot_command",
						Offset: 0,
						Length: len(tt.command) + 1,
					},
				},
				Chat: &tgbotapi.Chat{
					ID: tt.chatID,
				},
				From: &tgbotapi.User{
					ID: tt.userID,
				},
			}

			// Call handleCommand
			resp, err := svc.handleCommand(context.Background(), msg)

			// Check error
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

			// Check response
			if tt.wantText != "" {
				assert.Equal(t, tt.wantText, resp.Text)
			}

			// For new_token success, check that the response contains token information
			if tt.command == "new_token" && !tt.wantErr && tt.wantText == "" {
				assert.Contains(t, resp.Text, "Your New API Token")
				assert.Contains(t, resp.Text, "token123")
				assert.Contains(t, resp.Text, "Valid until:")
			}
		})
	}
}

func TestHandleMessage(t *testing.T) {
	tests := []struct {
		name       string
		message    *tgbotapi.Message
		setupMocks func(mockTokenSvc *MockTokenService)
		wantText   string
		wantErr    bool
	}{
		{
			name: "command message",
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
				From: &tgbotapi.User{
					ID: 456,
				},
			},
			setupMocks: func(mockTokenSvc *MockTokenService) {
				// No mocks needed for start command
			},
			wantText: welcomeMessage,
			wantErr:  false,
		},
		{
			name: "non-command message",
			message: &tgbotapi.Message{
				Text: "hello",
				Chat: &tgbotapi.Chat{
					ID: 123,
				},
				From: &tgbotapi.User{
					ID: 456,
				},
			},
			setupMocks: func(mockTokenSvc *MockTokenService) {
				// No mocks needed for non-command message
			},
			wantText: notCommandMessage,
			wantErr:  false,
		},
		{
			name: "command with error",
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
			setupMocks: func(mockTokenSvc *MockTokenService) {
				mockTokenSvc.EXPECT().CreateToken(mock.Anything, "456").Return(nil, errors.New("some error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a service with mocked dependencies
			mockTokenSvc := NewMockTokenService(t)
			svc := &Service{
				token:    "test-token",
				tg:       NewMocktgClient(t),
				tokenSvc: mockTokenSvc,
			}

			// Setup mocks
			tt.setupMocks(mockTokenSvc)

			// Call Handle
			resp, err := svc.Handle(context.Background(), tt.message)

			// Check error
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

			// Check response
			assert.Equal(t, tt.wantText, resp.Text)
		})
	}
}

// TestHandleCommandTimeFormat tests that the time format in the token message is correct
func TestHandleCommandTimeFormat(t *testing.T) {
	// Create a service with mocked dependencies
	mockTokenSvc := NewMockTokenService(t)
	svc := &Service{
		token:    "test-token",
		tg:       NewMocktgClient(t),
		tokenSvc: mockTokenSvc,
	}

	// Setup mock for CreateToken
	// Calculate expected expiration time (approximately 1 hour from now)
	expectedTime := time.Now().Add(time.Hour)
	formattedTime := expectedTime.Format(time.DateTime)

	response := &core.Response{
		Message: fmt.Sprintf("🔑 Your New API Token\n\ntoken123\n\n⏱ Valid until: %s\n\nKeep this token secure and don't share it with others.", formattedTime),
	}
	mockTokenSvc.EXPECT().CreateToken(mock.Anything, "456").Return(response, nil)

	// Create a message with the new_token command
	msg := &tgbotapi.Message{
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
	}

	// Call handleCommand
	resp, err := svc.handleCommand(context.Background(), msg)
	require.NoError(t, err)

	// Check that the response contains the token and expiration time
	assert.Contains(t, resp.Text, "token123")

	// Check that the response contains the year, month, and day
	assert.Contains(t, resp.Text, expectedTime.Format("2006"))
	assert.Contains(t, resp.Text, expectedTime.Format("01"))
	assert.Contains(t, resp.Text, expectedTime.Format("02"))
}

package core

import (
	"context"
	"errors"
	"testing"

	"github.com/ksysoev/make-it-public-tgbot/pkg/core/conv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestHandleMessage(t *testing.T) {
	tests := []struct {
		name         string
		userID       string
		message      string
		setupMocks   func(*testing.T) (*MockUserRepo, *MockMITProv, *conv.Conversation)
		expectedResp *Response
		expectedErr  string
	}{
		{
			name:    "get conversation error",
			userID:  "user123",
			message: "test message",
			setupMocks: func(t *testing.T) (*MockUserRepo, *MockMITProv, *conv.Conversation) {
				repo := NewMockUserRepo(t)
				prov := NewMockMITProv(t)

				repo.On("GetConversation", mock.Anything, "user123").Return(nil, errors.New("get conversation error"))

				return repo, prov, nil
			},
			expectedErr: "failed to get conversation: get conversation error",
		},
		{
			name:    "submit message error",
			userID:  "user123",
			message: "invalid message",
			setupMocks: func(t *testing.T) (*MockUserRepo, *MockMITProv, *conv.Conversation) {
				repo := NewMockUserRepo(t)
				prov := NewMockMITProv(t)

				conversation := conv.New("user123")
				// Setup conversation to return error on Submit
				// This is a bit tricky since we can't directly mock the conversation
				// We'll create a real conversation in a state that will cause Submit to fail

				repo.On("GetConversation", mock.Anything, "user123").Return(conversation, nil)

				return repo, prov, conversation
			},
			expectedErr: "failed to submit message: conversation is not in questions state",
		},
		{
			name:    "conversation not complete - return current question",
			userID:  "user123",
			message: "Yes",
			setupMocks: func(t *testing.T) (*MockUserRepo, *MockMITProv, *conv.Conversation) {
				repo := NewMockUserRepo(t)
				prov := NewMockMITProv(t)

				conversation := conv.New("user123")
				questions := conv.NewQuestions([]conv.Question{
					{
						Text:    "Do you want a token?",
						Answers: []string{"Yes", "No"},
					},
					{
						Text:    "Are you sure?",
						Answers: []string{"Yes", "No"},
					},
				})

				// Start the conversation with a custom state
				err := conversation.Start(StateTokenExists, questions)
				require.NoError(t, err)

				repo.On("GetConversation", mock.Anything, "user123").Return(conversation, nil)
				repo.On("SaveConversation", mock.Anything, conversation).Return(nil)

				return repo, prov, conversation
			},
			expectedResp: &Response{
				Message: "Are you sure?",
				Answers: []string{"Yes", "No"},
			},
		},
		{
			name:    "get current question error",
			userID:  "user123",
			message: "Yes",
			setupMocks: func(t *testing.T) (*MockUserRepo, *MockMITProv, *conv.Conversation) {
				repo := NewMockUserRepo(t)
				prov := NewMockMITProv(t)

				// Create a conversation that will return an error when Current() is called
				conversation := conv.New("user123")
				questions := conv.NewQuestions([]conv.Question{
					{
						Text:    "Do you want a token?",
						Answers: []string{"Yes", "No"},
					},
				})

				err := conversation.Start(StateTokenExists, questions)
				require.NoError(t, err)

				// Submit an answer to advance the conversation
				_, err = conversation.Submit("Yes")
				require.NoError(t, err)

				// Now Current() should fail because there are no more questions

				repo.On("GetConversation", mock.Anything, "user123").Return(conversation, nil)

				return repo, prov, conversation
			},
			expectedErr: "failed to submit message: conversation is not in questions state",
		},
		{
			name:    "save conversation error",
			userID:  "user123",
			message: "Yes",
			setupMocks: func(t *testing.T) (*MockUserRepo, *MockMITProv, *conv.Conversation) {
				repo := NewMockUserRepo(t)
				prov := NewMockMITProv(t)

				conversation := conv.New("user123")
				questions := conv.NewQuestions([]conv.Question{
					{
						Text:    "Do you want a token?",
						Answers: []string{"Yes", "No"},
					},
					{
						Text:    "Are you sure?",
						Answers: []string{"Yes", "No"},
					},
				})

				err := conversation.Start(StateTokenExists, questions)
				require.NoError(t, err)

				repo.On("GetConversation", mock.Anything, "user123").Return(conversation, nil)
				repo.On("SaveConversation", mock.Anything, conversation).Return(errors.New("save conversation error"))

				return repo, prov, conversation
			},
			expectedErr: "failed to save conversation: save conversation error",
		},
		{
			name:    "unsupported conversation state",
			userID:  "user123",
			message: "Yes",
			setupMocks: func(t *testing.T) (*MockUserRepo, *MockMITProv, *conv.Conversation) {
				repo := NewMockUserRepo(t)
				prov := NewMockMITProv(t)

				// Create a conversation with an unsupported state that will be completed when we submit the message
				conversation := conv.New("user123")
				questions := conv.NewQuestions([]conv.Question{
					{
						Text:    "Do you want a token?",
						Answers: []string{"Yes", "No"},
					},
				})

				// Use a custom state that's not handled in the switch statement
				err := conversation.Start("unsupportedState", questions)
				require.NoError(t, err)

				repo.On("GetConversation", mock.Anything, "user123").Return(conversation, nil)
				repo.On("SaveConversation", mock.Anything, conversation).Return(nil)

				return repo, prov, conversation
			},
			expectedErr: "unsupported conversation state: unsupportedState",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, prov, _ := tt.setupMocks(t)

			svc := New(repo, prov)

			resp, err := svc.HandleMessage(context.Background(), tt.userID, tt.message)

			if tt.expectedErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
				assert.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				assert.Contains(t, resp.Message, tt.expectedResp.Message)
				assert.Equal(t, tt.expectedResp.Answers, resp.Answers)
			}

			repo.AssertExpectations(t)
			prov.AssertExpectations(t)
		})
	}
}

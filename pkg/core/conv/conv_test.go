package conv

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	// Test creating a new conversation
	conv := New("test-id")
	assert.Equal(t, "test-id", conv.ID)
	assert.Equal(t, StateIdle, conv.State)
}

func TestConversation_Start(t *testing.T) {
	tests := []struct {
		name      string
		conv      *Conversation
		newState  State
		questions Questions
		wantErr   bool
	}{
		{
			name:      "start conversation from idle state",
			conv:      New("test-id"),
			newState:  "asking_name",
			questions: NewQuestions([]Question{{Text: "What's your name?", Answers: []string{"John", "Jane"}}}),
			wantErr:   false,
		},
		{
			name: "start conversation from non-idle state",
			conv: &Conversation{
				ID:    "test-id",
				State: "asking_name",
			},
			newState:  "asking_age",
			questions: NewQuestions([]Question{{Text: "How old are you?", Answers: []string{"20", "30"}}}),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.conv.Start(tt.newState, tt.questions)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.newState, tt.conv.State)
			assert.Equal(t, tt.questions, tt.conv.Questions)
		})
	}

	// Test panic when starting with invalid state
	assert.Panics(t, func() {
		conv := New("test-id")
		_ = conv.Start(StateIdle, NewQuestions([]Question{{Text: "Question", Answers: []string{"Answer"}}}))
	})
	assert.Panics(t, func() {
		conv := New("test-id")
		_ = conv.Start(StateComplete, NewQuestions([]Question{{Text: "Question", Answers: []string{"Answer"}}}))
	})
}

func TestConversation_Current(t *testing.T) {
	tests := []struct {
		name    string
		conv    *Conversation
		want    *Question
		wantErr bool
	}{
		{
			name: "get current question from active conversation",
			conv: func() *Conversation {
				c := New("test-id")
				c.State = "asking_name"
				c.Questions = NewQuestions([]Question{{Text: "What's your name?", Answers: []string{"John", "Jane"}}})
				return c
			}(),
			want:    &Question{Text: "What's your name?", Answers: []string{"John", "Jane"}},
			wantErr: false,
		},
		{
			name:    "get current question from idle conversation",
			conv:    New("test-id"),
			want:    nil,
			wantErr: true,
		},
		{
			name: "get current question from completed conversation",
			conv: &Conversation{
				ID:    "test-id",
				State: StateComplete,
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.conv.Current()
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConversation_Submit(t *testing.T) {
	tests := []struct {
		name      string
		conv      *Conversation
		answer    string
		wantState State
		wantErr   bool
	}{
		{
			name: "submit valid answer to active conversation",
			conv: func() *Conversation {
				c := New("test-id")
				c.State = "asking_name"
				c.Questions = NewQuestions([]Question{
					{Text: "What's your name?", Answers: []string{"John", "Jane"}},
					{Text: "How old are you?", Answers: []string{"20", "30"}},
				})
				return c
			}(),
			answer:    "John",
			wantState: "asking_name",
			wantErr:   false,
		},
		{
			name: "submit valid answer to complete conversation",
			conv: func() *Conversation {
				c := New("test-id")
				c.State = "asking_name"
				c.Questions = NewQuestions([]Question{{Text: "What's your name?", Answers: []string{"John", "Jane"}}})
				return c
			}(),
			answer:    "John",
			wantState: StateComplete,
			wantErr:   false,
		},
		{
			name: "submit invalid answer",
			conv: func() *Conversation {
				c := New("test-id")
				c.State = "asking_name"
				c.Questions = NewQuestions([]Question{{Text: "What's your name?", Answers: []string{"John", "Jane"}}})
				return c
			}(),
			answer:    "Bob",
			wantState: "asking_name",
			wantErr:   true,
		},
		{
			name:      "submit answer to idle conversation",
			conv:      New("test-id"),
			answer:    "John",
			wantState: StateIdle,
			wantErr:   true,
		},
		{
			name: "submit answer to completed conversation",
			conv: &Conversation{
				ID:    "test-id",
				State: StateComplete,
			},
			answer:    "John",
			wantState: StateComplete,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For the "submit valid answer to complete conversation" test, we need to adjust the test
			// to ensure the conversation will be completed after the answer is submitted
			if tt.name == "submit valid answer to complete conversation" {
				// This will ensure the conversation is completed after the answer is submitted
				tt.conv.Questions.Position = 0
			}

			err := tt.conv.Submit(tt.answer)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.wantState, tt.conv.State)
		})
	}
}

func TestConversation_Results(t *testing.T) {
	tests := []struct {
		name      string
		conv      *Conversation
		wantState State
		wantQA    []QuestionAnswer
		wantErr   bool
	}{
		{
			name: "get results from completed conversation",
			conv: func() *Conversation {
				c := New("test-id")
				c.State = StateComplete
				qa := []QuestionAnswer{{
					Question: Question{Text: "What's your name?", Answers: []string{"John", "Jane"}},
					Answer:   "John",
				}}
				c.Questions = Questions{QAPairs: qa, Position: 1}
				return c
			}(),
			wantState: StateComplete,
			wantQA: []QuestionAnswer{{
				Question: Question{Text: "What's your name?", Answers: []string{"John", "Jane"}},
				Answer:   "John",
			}},
			wantErr: false,
		},
		{
			name: "get results from incomplete conversation",
			conv: func() *Conversation {
				c := New("test-id")
				c.State = "asking_name"
				c.Questions = NewQuestions([]Question{{Text: "What's your name?", Answers: []string{"John", "Jane"}}})
				return c
			}(),
			wantState: "",
			wantQA:    nil,
			wantErr:   true,
		},
		{
			name: "get results - error from GetResults",
			conv: func() *Conversation {
				c := New("test-id")
				c.State = StateComplete
				// Create a Questions implementation that will return an error from GetResults
				c.Questions = Questions{
					QAPairs: []QuestionAnswer{
						{Question: Question{Text: "What's your name?", Answers: []string{"John", "Jane"}}},
					},
					Position: 0, // Position is 0 but QAPairs has 1 item, so GetResults will return ErrQuestionnaireIncomplete
				}
				return c
			}(),
			wantState: "",
			wantQA:    nil,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotState, gotQA, err := tt.conv.Results()
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.wantState, gotState)
			assert.Equal(t, tt.wantQA, gotQA)
			// After getting results, the conversation should be in idle state
			assert.Equal(t, StateIdle, tt.conv.State)
		})
	}
}

package conv

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewQuestions(t *testing.T) {
	// Test creating new questions
	questions := []Question{
		{Text: "What's your name?", Answers: []string{"John", "Jane"}},
		{Text: "How old are you?", Answers: []string{"20", "30"}},
	}

	qs := NewQuestions(questions)

	// Check that the questions were properly initialized
	assert.Equal(t, 2, len(qs.QAPairs))
	assert.Equal(t, 0, qs.Position)

	// Check that the question-answer pairs were properly created
	assert.Equal(t, "What's your name?", qs.QAPairs[0].Question.Text)
	assert.Equal(t, []string{"John", "Jane"}, qs.QAPairs[0].Question.Answers)
	assert.Equal(t, "", qs.QAPairs[0].Answer)

	assert.Equal(t, "How old are you?", qs.QAPairs[1].Question.Text)
	assert.Equal(t, []string{"20", "30"}, qs.QAPairs[1].Question.Answers)
	assert.Equal(t, "", qs.QAPairs[1].Answer)
}

func TestQuestions_GetQuestion(t *testing.T) {
	tests := []struct {
		name    string
		qs      Questions
		want    *Question
		wantErr bool
		errType error
	}{
		{
			name: "get first question",
			qs: Questions{
				QAPairs: []QuestionAnswer{
					{Question: Question{Text: "What's your name?", Answers: []string{"John", "Jane"}}},
					{Question: Question{Text: "How old are you?", Answers: []string{"20", "30"}}},
				},
				Position: 0,
			},
			want:    &Question{Text: "What's your name?", Answers: []string{"John", "Jane"}},
			wantErr: false,
		},
		{
			name: "get second question",
			qs: Questions{
				QAPairs: []QuestionAnswer{
					{Question: Question{Text: "What's your name?", Answers: []string{"John", "Jane"}}},
					{Question: Question{Text: "How old are you?", Answers: []string{"20", "30"}}},
				},
				Position: 1,
			},
			want:    &Question{Text: "How old are you?", Answers: []string{"20", "30"}},
			wantErr: false,
		},
		{
			name: "no more questions",
			qs: Questions{
				QAPairs: []QuestionAnswer{
					{Question: Question{Text: "What's your name?", Answers: []string{"John", "Jane"}}},
				},
				Position: 1,
			},
			want:    nil,
			wantErr: true,
			errType: ErrNoMoreQuestions,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.qs.GetQuestion()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errType != nil {
					assert.Equal(t, tt.errType, err)
				}
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestQuestions_ProcessAnswer(t *testing.T) {
	tests := []struct {
		name       string
		qs         Questions
		answer     string
		wantDone   bool
		wantErr    bool
		wantPos    int
		wantAnswer string
	}{
		{
			name: "process valid answer - not done",
			qs: Questions{
				QAPairs: []QuestionAnswer{
					{Question: Question{Text: "What's your name?", Answers: []string{"John", "Jane"}}},
					{Question: Question{Text: "How old are you?", Answers: []string{"20", "30"}}},
				},
				Position: 0,
			},
			answer:     "John",
			wantDone:   false,
			wantErr:    false,
			wantPos:    1,
			wantAnswer: "John",
		},
		{
			name: "process valid answer - done",
			qs: Questions{
				QAPairs: []QuestionAnswer{
					{Question: Question{Text: "What's your name?", Answers: []string{"John", "Jane"}}},
				},
				Position: 0,
			},
			answer:     "John",
			wantDone:   true,
			wantErr:    false,
			wantPos:    1,
			wantAnswer: "John",
		},
		{
			name: "process invalid answer",
			qs: Questions{
				QAPairs: []QuestionAnswer{
					{Question: Question{Text: "What's your name?", Answers: []string{"John", "Jane"}}},
				},
				Position: 0,
			},
			answer:  "Bob",
			wantErr: true,
			wantPos: 0,
		},
		{
			name: "no more questions",
			qs: Questions{
				QAPairs: []QuestionAnswer{
					{Question: Question{Text: "What's your name?", Answers: []string{"John", "Jane"}}},
				},
				Position: 1,
			},
			answer:  "John",
			wantErr: true,
			wantPos: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			done, err := tt.qs.ProcessAnswer(tt.answer)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Equal(t, tt.wantPos, tt.qs.Position)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.wantDone, done)
			assert.Equal(t, tt.wantPos, tt.qs.Position)
			assert.Equal(t, tt.wantAnswer, tt.qs.QAPairs[tt.wantPos-1].Answer)
		})
	}
}

func TestQuestions_GetResults(t *testing.T) {
	tests := []struct {
		name    string
		qs      Questions
		want    []QuestionAnswer
		wantErr bool
		errType error
	}{
		{
			name: "get results - complete",
			qs: Questions{
				QAPairs: []QuestionAnswer{
					{
						Question: Question{Text: "What's your name?", Answers: []string{"John", "Jane"}},
						Answer:   "John",
					},
				},
				Position: 1,
			},
			want: []QuestionAnswer{
				{
					Question: Question{Text: "What's your name?", Answers: []string{"John", "Jane"}},
					Answer:   "John",
				},
			},
			wantErr: false,
		},
		{
			name: "get results - incomplete",
			qs: Questions{
				QAPairs: []QuestionAnswer{
					{
						Question: Question{Text: "What's your name?", Answers: []string{"John", "Jane"}},
						Answer:   "John",
					},
					{
						Question: Question{Text: "How old are you?", Answers: []string{"20", "30"}},
						Answer:   "",
					},
				},
				Position: 1,
			},
			want:    nil,
			wantErr: true,
			errType: ErrQuestionnaireIncomplete,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.qs.GetResults()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errType != nil {
					assert.Equal(t, tt.errType, err)
				}
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

package conv

import (
	"errors"
	"fmt"
)

var (
	ErrIsNotComplete = errors.New("conversation is not complete")
)

type State string

const (
	StateIdle     State = "idle"
	StateComplete State = "complete"
)

type Question struct {
	Text    string   `json:"text"`
	Answers []string `json:"answers,omitempty"`
}

type QuestionAnswer struct {
	Answer   string   `json:"answer"`
	Field    string   `json:"field,omitempty"`
	Question Question `json:"question"`
}

type Conversation struct {
	ID        string
	State     State
	Questions Questions `json:"Questions"`
}

// New creates a new Conversation instance with the given ID and sets its state to StateIdle.
func New(id string) *Conversation {
	return &Conversation{
		ID:    id,
		State: StateIdle,
	}
}

// Start initializes the conversation with a new state and a set of questions, returning an error if the state is invalid.
func (c *Conversation) Start(newState State, questions Questions) error {
	if c.State != StateIdle {
		return errors.New("conversation is not in chat state")
	}

	if newState == StateIdle || newState == StateComplete {
		panic("invalid state for questions, canot use StateIdle or StateComplete")
	}

	c.State = newState
	c.Questions = questions

	return nil
}

// Current retrieves the current question in the conversation if it is in an active questions state, else returns an error.
func (c *Conversation) Current() (*Question, error) {
	if c.State == StateIdle || c.State == StateComplete {
		return nil, errors.New("conversation is not in questions state")
	}

	return c.Questions.GetQuestion()
}

// Submit processes the provided answer, advancing the conversation state and tracking completion or errors as appropriate.
func (c *Conversation) Submit(answer string) error {
	if c.State == StateIdle || c.State == StateComplete {
		return errors.New("conversation is not in questions state")
	}

	done, err := c.Questions.ProcessAnswer(answer)
	if err != nil {
		return err
	}

	if done {
		c.State = StateComplete
	}

	return nil
}

// Results retrieves the completed question-answer pairs of a conversation if it is in the complete state, returning an error otherwise.
func (c *Conversation) Results() (State, []QuestionAnswer, error) {
	if c.State != StateComplete {
		return "", nil, ErrIsNotComplete
	}

	r, err := c.Questions.GetResults()

	if err != nil {
		return "", nil, fmt.Errorf("failed to get question results: %w", err)
	}

	curState := c.State
	c.State = StateIdle

	return curState, r, nil
}

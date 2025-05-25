package conv

const (
	StateChat = "chat"
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
	ID            string
	State         string     `json:"state"`
	Questionnaire *Questions `json:"Questions"`
}

func New(id string) *Conversation {
	return &Conversation{
		ID:    id,
		State: StateChat,
	}
}

func (c *Conversation) StartQuestions(questions Questions) {
	c.State = questions.Kind
	c.Questionnaire = &questions
}

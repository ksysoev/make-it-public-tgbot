package conv

import (
	"errors"
)

var (
	ErrNoMoreQuestions         = errors.New("no more questions")
	ErrQuestionnaireIncomplete = errors.New("questionnaire is incomplete")
)

type Questions struct {
	QAPairs  []QuestionAnswer `json:"qa_pairs"`
	Position int              `json:"position"`
}

func NewQuestions(questions []Question) Questions {
	qaPairs := make([]QuestionAnswer, len(questions))
	for i, q := range questions {
		qaPairs[i] = QuestionAnswer{
			Question: q,
			Answer:   "",
		}
	}

	return Questions{
		QAPairs:  qaPairs,
		Position: 0,
	}
}

func (f *Questions) GetQuestion() (*Question, error) {
	if f.Position >= len(f.QAPairs) {
		return nil, ErrNoMoreQuestions
	}
	return &f.QAPairs[f.Position].Question, nil
}

func (f *Questions) ProcessAnswer(answer string) (bool, error) {
	if f.Position >= len(f.QAPairs) {
		return false, ErrNoMoreQuestions
	}

	answers := f.QAPairs[f.Position].Question.Answers

	for _, a := range answers {
		if a == answer {
			f.QAPairs[f.Position].Answer = answer
			f.Position++
			return f.Position >= len(f.QAPairs), nil
		}
	}

	return false, errors.New("invalid answer")
}

func (f *Questions) GetResults() ([]QuestionAnswer, error) {
	if f.Position < len(f.QAPairs) {
		return nil, ErrQuestionnaireIncomplete
	}

	return f.QAPairs, nil
}

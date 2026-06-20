package bedrock

import (
	"errors"
	"slices"
)

type Model string

const (
	MODEL_CLAUDE_DEFAULT   Model = "anthropic.claude-sonnet-4-20250514-v1:0"
	MODEL_CLAUDE_HAIKU     Model = "anthropic.claude-haiku-4-5-20251001-v1:0"
	MODEL_GEMINI_DEFAULT   Model = "google.gemini-2.5-flash"
	MODEL_GPT_DEFAULT      Model = "openai.gpt-oss-120b"
	MODEL_DEEPSEEK_DEFAULT Model = "deepseek-v3.1"
)

var ErrUnknownModel = errors.New("unknown bedrock model")

var validModels = map[Model]struct{}{
	MODEL_CLAUDE_DEFAULT:   {},
	MODEL_CLAUDE_HAIKU:     {},
	MODEL_GEMINI_DEFAULT:   {},
	MODEL_GPT_DEFAULT:      {},
	MODEL_DEEPSEEK_DEFAULT: {},
}

func ParseModel(s string) (Model, error) {
	m := Model(s)
	if _, ok := validModels[m]; !ok {
		return "", ErrUnknownModel
	}
	return m, nil
}

func (m Model) IsValid() bool {
	_, ok := validModels[m]
	return ok
}

func (m Model) String() string {
	return string(m)
}

func Models() []Model {
	out := make([]Model, 0, len(validModels))
	for m := range validModels {
		out = append(out, m)
	}
	slices.Sort(out)
	return out
}

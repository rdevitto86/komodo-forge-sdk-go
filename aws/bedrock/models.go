package bedrock

import (
	"errors"
	"slices"
)

// Identifies a Bedrock model by its full AWS model identifier string.
type Model string

const (
	ModelClaudeOpus4_7    Model = "anthropic.claude-opus-4-7-20260115-v1:0"
	ModelClaudeSonnet4_6  Model = "anthropic.claude-sonnet-4-6-v1:0"
	ModelClaudeHaiku4_5   Model = "anthropic.claude-haiku-4-5-20251001-v1:0"
	ModelTitanTextExpress Model = "amazon.titan-text-express-v1"
	ModelTitanTextLite    Model = "amazon.titan-text-lite-v1"
	ModelLlama3_70B       Model = "meta.llama3-70b-instruct-v1:0"
	ModelLlama3_8B        Model = "meta.llama3-8b-instruct-v1:0"
	ModelMistralLarge     Model = "mistral.mistral-large-2402-v1:0"
)

// ErrUnknownModel is returned when a model identifier is not in the supported set.
var ErrUnknownModel = errors.New("unknown bedrock model")

var validModels = map[Model]struct{}{
	ModelClaudeOpus4_7:    {},
	ModelClaudeSonnet4_6:  {},
	ModelClaudeHaiku4_5:   {},
	ModelTitanTextExpress: {},
	ModelTitanTextLite:    {},
	ModelLlama3_70B:       {},
	ModelLlama3_8B:        {},
	ModelMistralLarge:     {},
}

// Validates and returns a Model from a raw string; returns ErrUnknownModel if the value is not a known identifier.
func ParseModel(s string) (Model, error) {
	m := Model(s)
	if _, ok := validModels[m]; !ok {
		return "", ErrUnknownModel
	}
	return m, nil
}

// Reports whether m is a known Bedrock model identifier.
func (m Model) IsValid() bool {
	_, ok := validModels[m]
	return ok
}

// Returns the raw model identifier string.
func (m Model) String() string {
	return string(m)
}

// Returns all supported model identifiers in deterministic sorted order.
func Models() []Model {
	out := make([]Model, 0, len(validModels))
	for m := range validModels {
		out = append(out, m)
	}
	slices.Sort(out)
	return out
}

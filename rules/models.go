package rules

import "regexp"

const (
	LevelIgnore  = "ignore"
	LevelLenient = "lenient"
	LevelStrict  = "strict"
)

// FieldSpec describes validation rules for a single header, path param, query
// param, or body field. The compiled field is populated at config load time
// from Pattern — it is unexported so yaml ignores it during unmarshal.
type FieldSpec struct {
	Type     string   `yaml:"type,omitempty"`
	Required bool     `yaml:"required,omitempty"`
	Value    any      `yaml:"value,omitempty"`
	Pattern  string   `yaml:"pattern,omitempty"`
	Enum     []string `yaml:"enum,omitempty"`
	MinLen   int      `yaml:"min_len,omitempty"`
	MaxLen   int      `yaml:"max_len,omitempty"`
	compiled *regexp.Regexp
}

type Headers     map[string]FieldSpec
type PathParams  map[string]FieldSpec
type QueryParams map[string]FieldSpec
type Body        map[string]FieldSpec

type EvalRule struct {
	Toggle          bool        `yaml:"toggle,omitempty"`
	Level           string      `yaml:"level,omitempty"`
	OriginTypes     []string    `yaml:"originTypes,omitempty"`
	Headers         Headers     `yaml:"headers,omitempty"`
	PathParams      PathParams  `yaml:"params,omitempty"`
	QueryParams     QueryParams `yaml:"query,omitempty"`
	Body            Body        `yaml:"body,omitempty"`
	RequiredVersion int         `yaml:"requiredVersion,omitempty"`
}

type RuleConfig map[string]map[string]EvalRule

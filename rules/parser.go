package rules

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"

	logger "github.com/rdevitto86/komodo-forge-sdk-go/logging/runtime"

	"gopkg.in/yaml.v3"
)

var (
	ruleMap       map[string]map[string]EvalRule
	patternRoutes []routePattern
	mu            sync.Mutex
	configLoaded  bool
)

type routePattern struct {
	template  string
	re        *regexp.Regexp
	methods   map[string]EvalRule
	paramKeys []string
}

func LoadConfig(path ...string) bool {
	mu.Lock()
	defer mu.Unlock()

	if configLoaded {
		return true
	}

	var data []byte
	var err error
	var source string

	if len(path) > 0 && path[0] != "" {
		data, err = os.ReadFile(path[0])
		source = path[0]
	} else if envPath := os.Getenv("EVAL_RULES_PATH"); envPath != "" {
		data, err = os.ReadFile(envPath)
		source = envPath
	}

	if err != nil || data == nil {
		logger.Error("failed to load validation rules", err, logger.Attr("source", source))
		return false
	}

	rt, patterns, parseErr := parseConfigFromData(data)
	if parseErr != nil {
		logger.Error("failed to parse validation rules", parseErr)
		return false
	}

	ruleMap = rt
	patternRoutes = patterns
	configLoaded = true

	logger.Info("loaded validation rules", logger.Attr("source", source))
	return true
}

func LoadConfigWithData(data []byte) bool {
	mu.Lock()
	defer mu.Unlock()

	if configLoaded {
		return true
	}

	rt, patterns, err := parseConfigFromData(data)
	if err != nil {
		logger.Error("failed to parse validation rules from embedded config", err)
		return false
	}

	ruleMap = rt
	patternRoutes = patterns
	configLoaded = true

	logger.Info("loaded validation rules from embedded config")
	return true
}

func IsConfigLoaded() bool {
	mu.Lock()
	defer mu.Unlock()
	return configLoaded && ruleMap != nil
}

func ResetForTesting() {
	mu.Lock()
	defer mu.Unlock()
	configLoaded = false
	ruleMap = nil
	patternRoutes = nil
}

func GetRule(pKey string, method string) (*EvalRule, map[string]string) {
	if pKey == "" || method == "" || ruleMap == nil {
		return nil, nil
	}

	np := normalizePath(pKey)

	if rules, ok := ruleMap[np]; ok {
		if rule, exists := rules[method]; exists {
			return &rule, nil
		}
	}

	for _, rp := range patternRoutes {
		matches := rp.re.FindStringSubmatch(np)
		if matches == nil {
			continue
		}
		rule, exists := rp.methods[method]
		if !exists {
			continue
		}
		names := rp.re.SubexpNames()
		params := make(map[string]string, len(rp.paramKeys))
		for i, name := range names {
			if i != 0 && name != "" && i < len(matches) {
				params[name] = matches[i]
			}
		}
		return &rule, params
	}
	return nil, nil
}

func GetRules() RuleConfig { return ruleMap }

func validateAndNormalizeConfig(cfg RuleConfig) error {
	for _, methods := range cfg {
		for method, rule := range methods {
			if rule.Headers == nil {
				rule.Headers = make(Headers)
			}
			if rule.QueryParams == nil {
				rule.QueryParams = make(QueryParams)
			}
			if rule.PathParams == nil {
				rule.PathParams = make(PathParams)
			}
			if rule.Body == nil {
				rule.Body = make(Body)
			}

			methods[method] = rule
		}
	}
	return nil
}

func parseConfigFromData(data []byte) (map[string]map[string]EvalRule, []routePattern, error) {
	var root struct {
		Rules RuleConfig `yaml:"rules"`
	}

	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, nil, fmt.Errorf("invalid yaml: %w", err)
	}

	cfg := root.Rules

	if err := validateAndNormalizeConfig(cfg); err != nil {
		return nil, nil, err
	}

	patterns := make([]routePattern, 0)

	for tpl, methods := range cfg {
		if strings.Contains(tpl, ":") || strings.Contains(tpl, "{") || strings.Contains(tpl, "*") {
			reStr, keys := templateToRegex(tpl)
			re, err := regexp.Compile("^" + reStr + "$")
			if err != nil {
				return nil, nil, fmt.Errorf("invalid route pattern %s: %w", tpl, err)
			}

			patterns = append(patterns, routePattern{
				template:  tpl,
				re:        re,
				methods:   methods,
				paramKeys: keys,
			})
		}
	}

	specificityScore := func(tpl string) int {
		parts := strings.Split(strings.TrimPrefix(tpl, "/"), "/")
		literal, wild := 0, 0
		for _, p := range parts {
			if p == "*" || strings.HasPrefix(p, ":") || (strings.HasPrefix(p, "{") && strings.HasSuffix(p, "}")) {
				wild++
			} else if p != "" {
				literal++
			}
		}
		return literal*10 - wild
	}

	sort.SliceStable(patterns, func(i, j int) bool {
		return specificityScore(patterns[i].template) > specificityScore(patterns[j].template)
	})

	if err := prepareFieldSpecs(cfg); err != nil {
		return nil, nil, err
	}

	return cfg, patterns, nil
}

func prepareFieldSpecs(cfg RuleConfig) error {
	prepare := func(kind, name string, spec *FieldSpec) error {
		if spec.Pattern != "" {
			re, err := regexp.Compile(spec.Pattern)
			if err != nil {
				return fmt.Errorf("invalid %s pattern for %q: %w", kind, name, err)
			}
			spec.compiled = re
		}
		if len(spec.Enum) > 0 {
			spec.enumSet = make(map[string]struct{}, len(spec.Enum))
			for _, v := range spec.Enum {
				spec.enumSet[v] = struct{}{}
			}
		}
		return nil
	}

	for _, methods := range cfg {
		for method, rule := range methods {
			for name, spec := range rule.Headers {
				if err := prepare("header", name, &spec); err != nil {
					return err
				}
				rule.Headers[name] = spec
			}
			for name, spec := range rule.PathParams {
				if err := prepare("path param", name, &spec); err != nil {
					return err
				}
				rule.PathParams[name] = spec
			}
			for name, spec := range rule.QueryParams {
				if err := prepare("query param", name, &spec); err != nil {
					return err
				}
				rule.QueryParams[name] = spec
			}
			for name, spec := range rule.Body {
				if err := prepare("body field", name, &spec); err != nil {
					return err
				}
				rule.Body[name] = spec
			}

			methods[method] = rule
		}
	}
	return nil
}

func templateToRegex(tpl string) (string, []string) {
	if idx := strings.Index(tpl, "?"); idx != -1 {
		tpl = tpl[:idx]
	}
	tpl = strings.TrimSuffix(tpl, "/")

	parts := strings.Split(strings.TrimPrefix(tpl, "/"), "/")
	regexParts := make([]string, 0, len(parts))
	keys := make([]string, 0)

	for _, p := range parts {
		if p == "*" {
			regexParts = append(regexParts, ".*")
			continue
		}
		if key, ok := strings.CutPrefix(p, ":"); ok {
			keys = append(keys, key)
			regexParts = append(regexParts, `(?P<`+key+`>[^/]+)`)
			continue
		}
		if strings.HasPrefix(p, "{") && strings.HasSuffix(p, "}") {
			key := strings.TrimSuffix(strings.TrimPrefix(p, "{"), "}")
			keys = append(keys, key)
			regexParts = append(regexParts, `(?P<`+key+`>[^/]+)`)
			continue
		}
		regexParts = append(regexParts, regexp.QuoteMeta(p))
	}
	return "/" + strings.Join(regexParts, "/"), keys
}

func normalizePath(p string) string {
	if p == "" {
		return p
	}

	if idx := strings.Index(p, "?"); idx != -1 {
		p = p[:idx]
	}
	p = strings.TrimSpace(p)
	if p == "" {
		return "/"
	}

	if len(p) > 1 && strings.HasSuffix(p, "/") {
		p = strings.TrimSuffix(p, "/")
	}

	trimmed := strings.TrimPrefix(p, "/")
	segs := strings.Split(trimmed, "/")

	if len(segs) > 0 && len(segs[0]) > 1 && segs[0][0] == 'v' && segs[0][1] >= '0' && segs[0][1] <= '9' {
		segs = segs[1:]
		p = "/" + strings.Join(segs, "/")
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return p
}

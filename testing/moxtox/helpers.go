package moxtox

import (
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"os"
	"strings"
	"time"
)

type MoxtoxConfig struct {
	EnableMoxtox        bool                                      `yaml:"enableMoxtox"`
	Debug               bool                                      `yaml:"debug"`
	DefaultDelay        int                                       `yaml:"defaultDelay"`
	PerformanceMode     string                                    `yaml:"performanceMode"` // "default", "quick", "dynamic"
	AllowedEnvironments []string                                  `yaml:"allowedEnvironments"`
	IgnoredRoutes       []string                                  `yaml:"ignoredRoutes"`
	Mappings            map[string]any                    `yaml:"mappings"` // Raw YAML for parsing
	LookupMap           map[string]map[string][]Scenario          `yaml:"-"`        // For default/dynamic: path -> method -> []Scenario
	HashLookupMap       map[string]map[string]map[string]Scenario `yaml:"-"`        // For quick: path -> method -> hash -> Scenario
}

type Mapping struct {
	Methods map[string]Method `yaml:"methods,omitempty"`
}

type Method struct {
	Scenarios []Scenario `yaml:"scenarios,omitempty"`
}

type Scenario struct {
	Name       string                 `yaml:"name"`
	File       string                 `yaml:"file"`
	Dynamic    bool                   `yaml:"dynamic,omitempty"`
	Template   string                 `yaml:"template,omitempty"`
	Delay      int                    `yaml:"delay,omitempty"`
	Log        bool                   `yaml:"log,omitempty"`
	Priority   int                    `yaml:"priority,omitempty"`
	Conditions map[string]any `yaml:"conditions,omitempty"`
}

type MockResponse struct {
	Status  int               `json:"status"`
	Body    string            `json:"body"`
	Headers map[string]string `json:"headers,omitempty"`
}

func (cfg *MoxtoxConfig) countTotalScenarios() int {
	count := 0
	for _, rawMapping := range cfg.Mappings {
		mappingData, ok := rawMapping.(map[any]any)
		if !ok {
			continue
		}
		mapping := parseMapping(mappingData)
		for _, methodData := range mapping.Methods {
			count += len(methodData.Scenarios)
		}
	}
	return count
}

func parseMapping(data map[any]any) Mapping {
	mapping := Mapping{Methods: make(map[string]Method)}

	for key, value := range data {
		keyStr := key.(string)

		if keyStr == "methods" {
			// Handle methods if present: parse as`` map[string]Method
			if methodsData, ok := value.(map[any]any); ok {
				for methodKey, methodValue := range methodsData {
					methodStr := methodKey.(string)
					if methodMap, ok := methodValue.(map[any]any); ok {
						method := parseMethod(methodMap)
						mapping.Methods[methodStr] = method
					}
				}
			}
		} else {
			// Assume sub-paths like /token
			if subData, ok := value.(map[any]any); ok {
				subMapping := parseMapping(subData)
				maps.Copy(mapping.Methods, subMapping.Methods)
			}
		}
	}
	return mapping
}

func parseMethod(data map[any]any) Method {
	method := Method{Scenarios: []Scenario{}}

	for key, value := range data {
		keyStr := key.(string)

		if keyStr == "scenarios" {
			if scenariosData, ok := value.([]any); ok {
				for _, scenarioData := range scenariosData {
					if scenarioMap, ok := scenarioData.(map[any]any); ok {
						scenario := parseScenario(scenarioMap)
						method.Scenarios = append(method.Scenarios, scenario)
					}
				}
			}
		}
	}
	return method
}

func parseScenario(data map[any]any) Scenario {
	scenario := Scenario{Conditions: make(map[string]any)}
	for key, value := range data {
		keyStr := key.(string)

		switch keyStr {
		case "name":
			if str, ok := value.(string); ok {
				scenario.Name = str
			}
		case "file":
			if str, ok := value.(string); ok {
				scenario.File = str
			}
		case "dynamic":
			if b, ok := value.(bool); ok {
				scenario.Dynamic = b
			}
		case "template":
			if str, ok := value.(string); ok {
				scenario.Template = str
			}
		case "delay":
			if i, ok := value.(int); ok {
				scenario.Delay = i
			}
		case "log":
			if b, ok := value.(bool); ok {
				scenario.Log = b
			}
		case "priority":
			if i, ok := value.(int); ok {
				scenario.Priority = i
			}
		case "conditions":
			if condMap, ok := value.(map[any]any); ok {
				for condKey, condValue := range condMap {
					scenario.Conditions[condKey.(string)] = condValue
				}
			}
		}
	}
	return scenario
}

// Dispatches to slice or hash based on mode.
func matchesRequest(r *http.Request) (Scenario, bool) {
	if config.PerformanceMode == "quick" || (config.PerformanceMode == "dynamic" && config.countTotalScenarios() > 10) {
		return matchesRequestHash(r)
	}
	return matchesRequestSlice(r)
}

func matchesConditions(r *http.Request, cond map[string]any) bool {
	for condType, condMap := range cond {
		switch condType {
		case "body":
			if !matchesBody(r, condMap.(map[any]any)) {
				return false
			}
		case "query":
			if !matchesQuery(r, condMap.(map[any]any)) {
				return false
			}
		case "headers":
			if !matchesHeaders(r, condMap.(map[any]any)) {
				return false
			}
		case "path":
			// For path params, simple check (extend for regex if needed)
			if !matchesPath(r.URL.Path, condMap.(map[any]any)) {
				return false
			}
		}
	}
	return true
}

func matchesBody(r *http.Request, condMap map[any]any) bool {
	if r.Body == nil || !strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		return false
	}

	var body map[string]any
	json.NewDecoder(r.Body).Decode(&body)

	for k, v := range condMap {
		key := k.(string)
		if bodyVal, ok := body[key]; !ok || fmt.Sprintf("%v", bodyVal) != fmt.Sprintf("%v", v) {
			return false
		}
	}
	return true
}

func matchesQuery(r *http.Request, condMap map[any]any) bool {
	query := r.URL.Query()

	for k, v := range condMap {
		key := k.(string)
		if queryVal := query.Get(key); queryVal != fmt.Sprintf("%v", v) {
			return false
		}
	}
	return true
}

func matchesHeaders(r *http.Request, condMap map[any]any) bool {
	for k, v := range condMap {
		key := k.(string)
		if headerVal := r.Header.Get(key); headerVal != fmt.Sprintf("%v", v) {
			return false
		}
	}
	return true
}

func matchesPath(path string, condMap map[any]any) bool {
	// Placeholder: assume conditions are key-value for path segments
	for k, v := range condMap {
		key := k.(string)
		// Simple check, e.g., if key is "id", check if path contains /id/v
		if !strings.Contains(path, fmt.Sprintf("/%s/%v", key, v)) {
			return false
		}
	}
	return true
}

func injectMock(w http.ResponseWriter, r *http.Request, scenario Scenario) error {
	delay := scenario.Delay
	if delay == 0 {
		delay = config.DefaultDelay
	}
	if delay > 0 {
		time.Sleep(time.Duration(delay) * time.Millisecond)
	}

	var responseBody string
	var status int
	headers := make(map[string]string)

	if scenario.Dynamic {
		// Generate dynamic response from template
		responseBody = generateDynamicResponse(r, scenario.Template)
		status = 200 // Default for dynamic
	} else {
		// Read from file
		data, err := os.ReadFile(scenario.File)
		if err != nil {
			return fmt.Errorf("failed to read mock file %s: %v", scenario.File, err)
		}
		var mock MockResponse
		if err := json.Unmarshal(data, &mock); err != nil {
			return fmt.Errorf("failed to parse mock file %s: %v", scenario.File, err)
		}
		responseBody = mock.Body
		status = mock.Status
		headers = mock.Headers
	}

	for k, v := range headers {
		w.Header().Set(k, v)
	}

	w.WriteHeader(status)
	w.Write([]byte(responseBody))
	return nil
}

func generateDynamicResponse(r *http.Request, template string) string {
	// Extract body
	var body map[string]any
	if r.Body != nil && strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		json.NewDecoder(r.Body).Decode(&body)
	}

	// Simple replacement: {{.body.key}} -> body[key]
	result := template
	for key, val := range body {
		placeholder := fmt.Sprintf("{{.body.%s}}", key)
		result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", val))
	}
	return result
}

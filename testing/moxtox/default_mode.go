package moxtox

import (
	"fmt"
	"net/http"
	"sort"
)

func (cfg *MoxtoxConfig) buildSliceLookupMap() {
	cfg.LookupMap = make(map[string]map[string][]Scenario)

	for path, rawMapping := range cfg.Mappings {
		mappingData, ok := rawMapping.(map[any]any)
		if !ok {
			continue
		}

		mapping := parseMapping(mappingData)
		cfg.LookupMap[path] = make(map[string][]Scenario)

		for method, methodData := range mapping.Methods {
			scenarios := methodData.Scenarios
			sort.Slice(scenarios, func(i, j int) bool {
				return scenarios[i].Priority > scenarios[j].Priority
			})
			cfg.LookupMap[path][method] = scenarios
		}
	}
}

func matchesRequestSlice(req *http.Request) (Scenario, bool) {
	path := req.URL.Path
	method := req.Method

	if methodMap, ok := config.LookupMap[path]; ok {
		// Check exact method
		if scenarios, ok := methodMap[method]; ok {
			for _, scenario := range scenarios {
				if matchesConditions(req, scenario.Conditions) {
					if config.Debug || scenario.Log {
						fmt.Printf("[::Moxtox::] matched scenario '%s' for %s %s\n", scenario.Name, method, path)
					}
					return scenario, true
				}
			}
		}
		// Check "*" as user-defined method
		if scenarios, ok := methodMap["*"]; ok {
			for _, scenario := range scenarios {
				if matchesConditions(req, scenario.Conditions) {
					if config.Debug || scenario.Log {
						fmt.Printf("[::Moxtox::] matched scenario '%s' for %s %s\n", scenario.Name, "*", path)
					}
					return scenario, true
				}
			}
		}
	}
	if config.Debug {
		fmt.Printf("[::Moxtox::] no match for %s %s\n", method, path)
	}
	return Scenario{}, false
}

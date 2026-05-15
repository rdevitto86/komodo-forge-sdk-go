package rules

import (
	"fmt"
	"net/http"

	httpErr "github.com/rdevitto86/komodo-forge-sdk-go/api/errors"
	logger "github.com/rdevitto86/komodo-forge-sdk-go/logging/runtime"
)

// Enforces request validation rules based on predefined configurations.
func RuleValidationMiddleware(next http.Handler) http.Handler {
	// Ensure config is loaded
	if !LoadConfig() {
		logger.Error("validation rules failed to load", fmt.Errorf("failed to load validation rules"))
	}

	return http.HandlerFunc(func(wtr http.ResponseWriter, req *http.Request) {
		if rule := GetRule(req.URL.Path, req.Method); rule != nil {
			if !IsRuleValid(req, rule) {
				logger.Error("request does not comply with validation rule", fmt.Errorf("validation rule failed: %v", rule))
				httpErr.SendError(
					wtr, req, httpErr.Global.BadRequest, httpErr.WithDetail("request contents invalid"),
				)
				return
			}
		} else {
			logger.Error("no validation rule found", fmt.Errorf("no validation rule found for path: %s and method: %s", req.URL.Path, req.Method))
			httpErr.SendError(
				wtr, req, httpErr.Global.BadRequest, httpErr.WithDetail("failed to validate request"),
			)
			return
		}
		next.ServeHTTP(wtr, req)
	})
}

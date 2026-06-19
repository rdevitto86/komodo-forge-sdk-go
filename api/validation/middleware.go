package validation

import (
	"net/http"

	httpErr "github.com/rdevitto86/komodo-forge-sdk-go/api/errors"
	logger "github.com/rdevitto86/komodo-forge-sdk-go/logging/runtime"
	"github.com/rdevitto86/komodo-forge-sdk-go/rules"
)

type Config struct {
	RejectUnmapped bool
}

func Middleware(cfg ...Config) func(http.Handler) http.Handler {
	rejectUnmapped := false
	if len(cfg) > 0 {
		rejectUnmapped = cfg[0].RejectUnmapped
	}

	if !rules.LoadConfig() {
		logger.Error("failed to load validation rules", nil)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(wtr http.ResponseWriter, req *http.Request) {
			rule, params := rules.GetRule(req.URL.Path, req.Method)
			if rule == nil {
				if rejectUnmapped {
					httpErr.SendError(wtr, req, httpErr.Global.BadRequest, httpErr.WithDetail("failed to validate request"))
					return
				}
				next.ServeHTTP(wtr, req)
				return
			}
			if !rules.IsRuleValid(req, rule, params) {
				httpErr.SendError(wtr, req, httpErr.Global.BadRequest, httpErr.WithDetail("request contents invalid"))
				return
			}
			next.ServeHTTP(wtr, req)
		})
	}
}

func RuleValidationMiddleware(next http.Handler) http.Handler {
	return Middleware(Config{RejectUnmapped: true})(next)
}

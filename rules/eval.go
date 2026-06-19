package rules

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	headers "github.com/rdevitto86/komodo-forge-sdk-go/api/headers"
	httpReq "github.com/rdevitto86/komodo-forge-sdk-go/api/request"
	logger "github.com/rdevitto86/komodo-forge-sdk-go/logging/runtime"
)

func matchesPattern(pattern string, compiled *regexp.Regexp, val string) bool {
	if pattern == "" {
		return true
	}
	re := compiled
	if re == nil {
		var err error
		if re, err = regexp.Compile(pattern); err != nil {
			return false
		}
	}
	return re.MatchString(val)
}

func IsRuleValid(req *http.Request, rule *EvalRule) bool {
	if req == nil || rule == nil {
		logger.Error("api request or eval rule is nil", fmt.Errorf("request or rule is nil"))
		return false
	}
	if rule.Level == LevelIgnore {
		logger.Info("rule level is IGNORE - skipping all validations")
		return true
	}

	if !isValidVersion(req, rule) {
		logger.Error("request failed version validation", fmt.Errorf("version validation failed"))
		return false
	}
	if !areValidHeaders(req, rule) {
		logger.Error("request failed header validation", fmt.Errorf("headers validation failed"))
		return false
	}
	if !areValidPathParams(req, rule) {
		logger.Error("request failed path-param validation", fmt.Errorf("path params validation failed"))
		return false
	}
	if !areValidQueryParams(req, rule) {
		logger.Error("request failed query-param validation", fmt.Errorf("query params validation failed"))
		return false
	}
	if !isValidBody(req, rule) {
		logger.Error("request failed body validation", fmt.Errorf("body validation failed"))
		return false
	}

	logger.Debug("all validations passed")
	return true
}

func isValidVersion(req *http.Request, rule *EvalRule) bool {
	if rule.Level == LevelLenient {
		versionStr := httpReq.GetAPIVersion(req)
		if versionStr == "" {
			logger.Warn("version not provided in request using lenient mode - allowing")
			return true
		}

		versionStr = strings.TrimPrefix(versionStr, "/v")
		version, err := strconv.Atoi(versionStr)
		if err != nil {
			logger.Warn(fmt.Sprintf("invalid version format: %s (lenient mode - allowing)", versionStr))
			return true
		}
		if rule.RequiredVersion > 0 && version != rule.RequiredVersion {
			logger.Warn(fmt.Sprintf("version mismatch: required %d, got %d (lenient mode - allowing)", rule.RequiredVersion, version))
			return true
		}
		logger.Debug(fmt.Sprintf("version validation passed (lenient): v%d", version))
		return true
	}

	if rule.RequiredVersion <= 0 {
		logger.Error(
			"rule configuration error: requiredVersion must be >= 1 for strict validation",
			fmt.Errorf("invalid requiredVersion"),
		)
		return false
	}

	versionStr := httpReq.GetAPIVersion(req)
	if versionStr == "" {
		logger.Error(
			fmt.Sprintf("version required (v%d) but not found in request", rule.RequiredVersion),
			fmt.Errorf("version not found"),
		)
		return false
	}

	versionStr = strings.TrimPrefix(versionStr, "/v")
	version, err := strconv.Atoi(versionStr)
	if err != nil {
		logger.Error(
			fmt.Sprintf("invalid version format: %s", versionStr),
			fmt.Errorf("invalid version format"),
		)
		return false
	}

	if version != rule.RequiredVersion {
		logger.Error(
			fmt.Sprintf("version mismatch: required %d, got %d", rule.RequiredVersion, version),
			fmt.Errorf("version mismatch"),
		)
		return false
	}

	logger.Debug(fmt.Sprintf("version validation passed (strict): v%d", version))
	return true
}

func areValidHeaders(req *http.Request, rule *EvalRule) bool {
	for hName, spec := range rule.Headers {
		val := req.Header.Get(hName)

		if spec.Required && val == "" {
			logger.Error(
				fmt.Sprintf("header %q is required but missing", hName),
				fmt.Errorf("header missing"),
			)
			return false
		}
		if val == "" {
			continue
		}

		if valStr, _ := spec.Value.(string); valStr != "" {
			if valStr[len(valStr)-1] == '*' {
				prefix := valStr[:len(valStr)-1]
				if !strings.HasPrefix(val, prefix) {
					logger.Error(
						fmt.Sprintf("header %q value %q does not match required prefix %q", hName, val, prefix),
						fmt.Errorf("header value mismatch"),
					)
					return false
				}
			} else if val != valStr {
				logger.Error(
					fmt.Sprintf("header %q value %q does not match required value %q", hName, val, valStr),
					fmt.Errorf("header value mismatch"),
				)
				return false
			}
		}

		if !matchesPattern(spec.Pattern, spec.compiled, val) {
			logger.Error(
				fmt.Sprintf("header %q value %q does not match pattern %q", hName, val, spec.Pattern),
				fmt.Errorf("header pattern mismatch"),
			)
			return false
		}

		if len(spec.Enum) > 0 {
			ok := false
			for _, e := range spec.Enum {
				if e == val {
					ok = true
					break
				}
			}
			if !ok {
				logger.Error(
					fmt.Sprintf("header %q value %q not in enum %v", hName, val, spec.Enum),
					fmt.Errorf("header enum mismatch"),
				)
				return false
			}
		}

		if spec.MinLen > 0 && len(val) < spec.MinLen {
			logger.Error(
				fmt.Sprintf("header %q value length %d is less than minLen %d", hName, len(val), spec.MinLen),
				fmt.Errorf("header length mismatch"),
			)
			return false
		}
		if spec.MaxLen > 0 && len(val) > spec.MaxLen {
			logger.Error(
				fmt.Sprintf("header %q value length %d is greater than maxLen %d", hName, len(val), spec.MaxLen),
				fmt.Errorf("header length mismatch"),
			)
			return false
		}
		if ok, err := headers.ValidateHeaderValue(hName, req); !ok || err != nil {
			logger.Error(
				fmt.Sprintf("header %q failed ValidateHeaderValue check", hName),
				err,
			)
			return false
		}
	}
	logger.Debug("all headers passed validation")
	return true
}

func areValidPathParams(req *http.Request, rule *EvalRule) bool {
	_, params := matchRouteAndExtractParams(req.URL.Path)
	if params == nil {
		for k, spec := range rule.PathParams {
			if spec.Required {
				_ = k
				logger.Error(
					fmt.Sprintf("path param %q is required but missing", k),
					fmt.Errorf("path param missing"),
				)
				return false
			}
		}
		return true
	}

	for name, spec := range rule.PathParams {
		val, ok := params[name]
		if !ok || val == "" {
			if spec.Required {
				logger.Error(
					fmt.Sprintf("path param %q is required but missing", name),
					fmt.Errorf("path param missing"),
				)
				return false
			}
			continue
		}

		if !matchesPattern(spec.Pattern, spec.compiled, val) {
			logger.Error(
				fmt.Sprintf("path param %q value %q does not match pattern %q", name, val, spec.Pattern),
				fmt.Errorf("path param pattern mismatch"),
			)
			return false
		}

		// enum check
		if len(spec.Enum) > 0 {
			okEnum := false
			for _, e := range spec.Enum {
				if e == val {
					okEnum = true
					break
				}
			}
			if !okEnum {
				logger.Error(
					fmt.Sprintf("path param %q value %q not in enum %v", name, val, spec.Enum),
					fmt.Errorf("path param enum mismatch"),
				)
				return false
			}
		}

		if spec.MinLen > 0 && len(val) < spec.MinLen {
			logger.Error(
				fmt.Sprintf("path param %q value length %d is less than minLen %d", name, len(val), spec.MinLen),
				fmt.Errorf("path param length mismatch"),
			)
			return false
		}
		if spec.MaxLen > 0 && len(val) > spec.MaxLen {
			logger.Error(
				fmt.Sprintf("path param %q value length %d is greater than maxLen %d", name, len(val), spec.MaxLen),
				fmt.Errorf("path param length mismatch"),
			)
			return false
		}

		switch spec.Type {
		case "", "string":
		case "int":
			if _, err := strconv.Atoi(val); err != nil {
				logger.Error(
					fmt.Sprintf("path param %q value %q is not a valid int", name, val),
					fmt.Errorf("path param type mismatch"),
				)
				return false
			}
		case "bool":
			if val != "true" && val != "false" {
				logger.Error(
					fmt.Sprintf("path param %q value %q is not a valid bool", name, val),
					fmt.Errorf("path param type mismatch"),
				)
				return false
			}
		default:
		}
	}
	return true
}

func areValidQueryParams(req *http.Request, rule *EvalRule) bool {
	params := httpReq.GetQueryParams(req)

	for name, spec := range rule.QueryParams {
		val, ok := params[name]
		if !ok || val == "" {
			if spec.Required {
				logger.Error(
					fmt.Sprintf("query param %q is required but missing", name),
					fmt.Errorf("query param missing"),
				)
				return false
			}
			continue
		}

		if !matchesPattern(spec.Pattern, spec.compiled, val) {
			logger.Error(
				fmt.Sprintf("query param %q value %q does not match pattern %q", name, val, spec.Pattern),
				fmt.Errorf("query param pattern mismatch"),
			)
			return false
		}

		if len(spec.Enum) > 0 {
			okv := false
			for _, e := range spec.Enum {
				if e == val {
					okv = true
					break
				}
			}
			if !okv {
				logger.Error(
					fmt.Sprintf("query param %q value %q not in enum %v", name, val, spec.Enum),
					fmt.Errorf("query param enum mismatch"),
				)
				return false
			}
		}

		if spec.MinLen > 0 && len(val) < spec.MinLen {
			logger.Error(
				fmt.Sprintf("query param %q value length %d is less than minLen %d", name, len(val), spec.MinLen),
				fmt.Errorf("query param length mismatch"),
			)
			return false
		}
		if spec.MaxLen > 0 && len(val) > spec.MaxLen {
			logger.Error(
				fmt.Sprintf("query param %q value length %d is greater than maxLen %d", name, len(val), spec.MaxLen),
				fmt.Errorf("query param length mismatch"),
			)
			return false
		}
	}
	return true
}

func isValidBody(req *http.Request, rule *EvalRule) bool {
	switch req.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	}

	const maxBody = 1 << 20 // 1 MiB
	bodyBytes, err := io.ReadAll(io.LimitReader(req.Body, maxBody))
	if err != nil {
		logger.Error("failed to read request body", err)
		return false
	}

	req.Body.Close()
	req.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	if len(bodyBytes) == 0 {
		return true
	}

	var bodyMap map[string]any
	dec := json.NewDecoder(bytes.NewReader(bodyBytes))
	dec.DisallowUnknownFields()

	if err := dec.Decode(&bodyMap); err != nil {
		logger.Error("failed to decode request body as JSON", err)
		return false
	}

	for name, spec := range rule.Body {
		v, ok := bodyMap[name]
		if !ok {
			if spec.Required {
				logger.Error(
					fmt.Sprintf("body field %q is required but missing", name),
					fmt.Errorf("body field missing"),
				)
				return false
			}
			continue
		}

		switch spec.Type {
		case "", "string":
			if _, ok := v.(string); !ok {
				logger.Error(
					fmt.Sprintf("body field %q is not a string", name),
					fmt.Errorf("body field type mismatch"),
				)
				return false
			}
		case "int":
			if _, ok := v.(float64); !ok {
				logger.Error(
					fmt.Sprintf("body field %q is not a number", name),
					fmt.Errorf("body field type mismatch"),
				)
				return false
			}
		case "bool":
			if _, ok := v.(bool); !ok {
				logger.Error(
					fmt.Sprintf("body field %q is not a bool", name),
					fmt.Errorf("body field type mismatch"),
				)
				return false
			}
		}
	}
	return true
}

package rules

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	headers "github.com/rdevitto86/komodo-forge-sdk-go/api/headers"
	httpReq "github.com/rdevitto86/komodo-forge-sdk-go/api/request"
	logger "github.com/rdevitto86/komodo-forge-sdk-go/logging/runtime"
)

func validateFieldValue(spec FieldSpec, val string) error {
	if spec.compiled != nil && !spec.compiled.MatchString(val) {
		return fmt.Errorf("value %q does not match pattern %q", val, spec.Pattern)
	}

	if spec.enumSet != nil {
		if _, ok := spec.enumSet[val]; !ok {
			return fmt.Errorf("value %q not in allowed enum %v", val, spec.Enum)
		}
	}

	if spec.MinLen > 0 && len(val) < spec.MinLen {
		return fmt.Errorf("value length %d is less than minimum %d", len(val), spec.MinLen)
	}
	if spec.MaxLen > 0 && len(val) > spec.MaxLen {
		return fmt.Errorf("value length %d exceeds maximum %d", len(val), spec.MaxLen)
	}

	switch spec.Type {
	case "", "string":
	case "int":
		if _, err := strconv.Atoi(val); err != nil {
			return fmt.Errorf("value %q is not a valid int", val)
		}
	case "bool":
		if val != "true" && val != "false" {
			return fmt.Errorf("value %q is not a valid bool", val)
		}
	}

	return nil
}

func IsRuleValid(req *http.Request, rule *EvalRule, pathParams map[string]string) bool {
	if req == nil || rule == nil {
		logger.Error("received nil request or rule", fmt.Errorf("nil argument"))
		return false
	}
	if rule.Level == LevelIgnore {
		return true
	}

	if !isValidVersion(req, rule) {
		return false
	}
	if !areValidHeaders(req, rule) {
		return false
	}
	if !areValidPathParams(rule, pathParams) {
		return false
	}
	if !areValidQueryParams(req, rule) {
		return false
	}
	if !isValidBody(req, rule) {
		return false
	}

	return true
}

func isValidVersion(req *http.Request, rule *EvalRule) bool {
	if rule.Level == LevelLenient {
		versionStr := httpReq.GetAPIVersion(req)
		if versionStr == "" {
			return true
		}

		versionStr = strings.TrimPrefix(versionStr, "/v")
		version, err := strconv.Atoi(versionStr)
		if err != nil {
			return true
		}
		if rule.RequiredVersion > 0 && version != rule.RequiredVersion {
			return true
		}
		return true
	}

	if rule.RequiredVersion <= 0 {
		logger.Error("requiredVersion must be >= 1 for strict validation", fmt.Errorf("invalid configuration"))
		return false
	}

	versionStr := httpReq.GetAPIVersion(req)
	if versionStr == "" {
		logger.Error("required version not found in request",
			fmt.Errorf("missing version"),
			logger.Attr("required", rule.RequiredVersion))
		return false
	}

	versionStr = strings.TrimPrefix(versionStr, "/v")
	version, err := strconv.Atoi(versionStr)
	if err != nil {
		logger.Error("failed to parse version",
			fmt.Errorf("invalid version format"),
			logger.Attr("raw", versionStr))
		return false
	}

	if version != rule.RequiredVersion {
		logger.Error("version mismatch",
			fmt.Errorf("expected %d got %d", rule.RequiredVersion, version),
			logger.Attr("required", rule.RequiredVersion),
			logger.Attr("actual", version))
		return false
	}

	return true
}

func areValidHeaders(req *http.Request, rule *EvalRule) bool {
	for hName, spec := range rule.Headers {
		val := req.Header.Get(hName)

		if spec.Required && val == "" {
			logger.Error("required header missing",
				fmt.Errorf("missing required header"),
				logger.Attr("header", hName))
			return false
		}
		if val == "" {
			continue
		}

		if valStr, _ := spec.Value.(string); valStr != "" {
			if valStr[len(valStr)-1] == '*' {
				prefix := valStr[:len(valStr)-1]
				if !strings.HasPrefix(val, prefix) {
					logger.Error("header prefix mismatch",
						fmt.Errorf("value does not match prefix"),
						logger.Attr("header", hName),
						logger.Attr("expected_prefix", prefix))
					return false
				}
			} else if val != valStr {
				logger.Error("header value mismatch",
					fmt.Errorf("value does not match"),
					logger.Attr("header", hName),
					logger.Attr("expected", valStr))
				return false
			}
		}

		if err := validateFieldValue(spec, val); err != nil {
			logger.Error("header validation failed", err, logger.Attr("header", hName))
			return false
		}

		if ok, err := headers.ValidateHeaderValue(hName, req); !ok || err != nil {
			logger.Error("header security check failed", err, logger.Attr("header", hName))
			return false
		}
	}
	return true
}

func areValidPathParams(rule *EvalRule, params map[string]string) bool {
	if params == nil {
		for name, spec := range rule.PathParams {
			if spec.Required {
				logger.Error("required path param missing",
					fmt.Errorf("missing required path param"),
					logger.Attr("param", name))
				return false
			}
		}
		return true
	}

	for name, spec := range rule.PathParams {
		val, ok := params[name]
		if !ok || val == "" {
			if spec.Required {
				logger.Error("required path param missing",
					fmt.Errorf("missing required path param"),
					logger.Attr("param", name))
				return false
			}
			continue
		}

		if err := validateFieldValue(spec, val); err != nil {
			logger.Error("path param validation failed", err, logger.Attr("param", name))
			return false
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
				logger.Error("required query param missing",
					fmt.Errorf("missing required query param"),
					logger.Attr("param", name))
				return false
			}
			continue
		}

		if err := validateFieldValue(spec, val); err != nil {
			logger.Error("query param validation failed", err, logger.Attr("param", name))
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

	if len(rule.Body) == 0 {
		return true
	}

	const maxBody = 1 << 20
	bodyBytes, err := io.ReadAll(io.LimitReader(req.Body, maxBody))
	if err != nil {
		logger.Error("failed to read request body", err)
		return false
	}

	req.Body.Close()
	req.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	if len(bodyBytes) == 0 {
		for name, spec := range rule.Body {
			if spec.Required {
				logger.Error("required body field missing in empty body",
					fmt.Errorf("missing required body field"),
					logger.Attr("field", name))
				return false
			}
		}
		return true
	}

	var bodyMap map[string]any
	if err := json.NewDecoder(bytes.NewReader(bodyBytes)).Decode(&bodyMap); err != nil {
		logger.Error("failed to decode request body as JSON", err)
		return false
	}

	for name, spec := range rule.Body {
		v, ok := bodyMap[name]
		if !ok {
			if spec.Required {
				logger.Error("required body field missing",
					fmt.Errorf("missing required body field"),
					logger.Attr("field", name))
				return false
			}
			continue
		}

		switch spec.Type {
		case "", "string":
			s, ok := v.(string)
			if !ok {
				logger.Error("body field type mismatch",
					fmt.Errorf("expected string"),
					logger.Attr("field", name))
				return false
			}
			if err := validateFieldValue(spec, s); err != nil {
				logger.Error("body field validation failed", err, logger.Attr("field", name))
				return false
			}
		case "int":
			n, ok := v.(float64)
			if !ok {
				logger.Error("body field type mismatch",
					fmt.Errorf("expected number"),
					logger.Attr("field", name))
				return false
			}
			if err := validateFieldValue(spec, strconv.FormatFloat(n, 'f', -1, 64)); err != nil {
				logger.Error("body field validation failed", err, logger.Attr("field", name))
				return false
			}
		case "bool":
			b, ok := v.(bool)
			if !ok {
				logger.Error("body field type mismatch",
					fmt.Errorf("expected bool"),
					logger.Attr("field", name))
				return false
			}
			if err := validateFieldValue(spec, strconv.FormatBool(b)); err != nil {
				logger.Error("body field validation failed", err, logger.Attr("field", name))
				return false
			}
		default:
			if err := validateFieldValue(spec, fmt.Sprint(v)); err != nil {
				logger.Error("body field validation failed", err, logger.Attr("field", name))
				return false
			}
		}
	}
	return true
}

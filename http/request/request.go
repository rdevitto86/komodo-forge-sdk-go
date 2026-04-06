package request

import (
	"context"
	"net/http"

	ctxKeys "github.com/rdevitto86/komodo-forge-sdk-go/http/context"
	"github.com/rdevitto86/komodo-forge-sdk-go/http/request/helpers"
)

func NewRequest(method string, url string, body any, headers map[string]string, ctx context.Context) (*http.Request, error) {
	return helpers.NewRequest(method, url, body, headers, ctx)
}

func FromRequest(req *http.Request) (*http.Request, error) { return helpers.FromRequest(req) }

func GenerateRequestId() string { return helpers.GenerateRequestId() }

func GetAPIVersion(req *http.Request) string { return helpers.GetAPIVersion(req) }

func GetAPIRoute(req *http.Request) string { return helpers.GetAPIRoute(req) }

func GetPathParams(req *http.Request) map[string]string { return helpers.GetPathParams(req) }

func GetQueryParams(req *http.Request) map[string]string { return helpers.GetQueryParams(req) }

func GetClientType(req *http.Request) string { return helpers.GetClientType(req) }

func GetClientKey(req *http.Request) string { return helpers.GetClientKey(req) }

func IsValidAPIKey(apiKey string) bool { return helpers.IsValidAPIKey(apiKey) }

func GetRequestID(req *http.Request) string {
	if rid, ok := req.Context().Value(ctxKeys.REQUEST_ID_KEY).(string); ok && rid != "" {
		return rid
	}
	return "unknown"
}

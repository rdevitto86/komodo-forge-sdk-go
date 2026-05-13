package httperrors

import "fmt"

// APIError is the client-side error type for non-2xx responses from a Komodo
// service. Callers decode the response body into this type and can type-assert
// to inspect the machine-readable Code and Status.
//
// Service is set by the caller (not decoded from JSON) to identify which
// service produced the error.
type APIError struct {
	Service   string `json:"-"`
	Status    int    `json:"status"`
	Code      string `json:"code"`
	Message   string `json:"message"`
	Detail    string `json:"detail,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}

// Builds a human-readable error message combining service, code, message, and detail.
func (e *APIError) Error() string {
	if e.Detail != "" {
		return fmt.Sprintf("%s [%s] %s: %s", e.Service, e.Code, e.Message, e.Detail)
	}
	return fmt.Sprintf("%s [%s] %s", e.Service, e.Code, e.Message)
}

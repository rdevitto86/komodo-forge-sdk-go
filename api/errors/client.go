package httperrors

import "fmt"

// Represents a non-2xx error response from a Komodo service. Service is set by the caller to identify the origin.
type APIError struct {
	Service   string `json:"-"`
	Status    int    `json:"status"`
	Code      string `json:"code"`
	Message   string `json:"message"`
	Detail    string `json:"detail,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}

// Returns a human-readable error string combining service, code, message, and optional detail.
func (e *APIError) Error() string {
	if e.Detail != "" {
		return fmt.Sprintf("%s [%s] %s: %s", e.Service, e.Code, e.Message, e.Detail)
	}
	return fmt.Sprintf("%s [%s] %s", e.Service, e.Code, e.Message)
}

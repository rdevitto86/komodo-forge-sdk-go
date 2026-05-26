package httperrors

import (
	"encoding/json"
	"net/http"
	"time"

	httpReq "github.com/rdevitto86/komodo-forge-sdk-go/api/request"
)

type ErrorResponse struct {
	Status    int    `json:"status"`
	Message   string `json:"message"`
	Code      string `json:"code"`
	Detail    string `json:"detail,omitempty"`
	RequestId string `json:"request_id"`
	Timestamp string `json:"timestamp"`
}

type ErrorOverride struct {
	Message *string
	Detail  *string
	Status  *int
}

// Sends a formatted JSON error response, applying any optional field overrides.
func SendError(wtr http.ResponseWriter, req *http.Request, errCode ErrorCode, overrides ...ErrorOverride) {
	wtr.Header().Set("Content-Type", "application/json")

	status := errCode.Status
	message := errCode.Message
	detail := ""

	if len(overrides) > 0 {
		override := overrides[0]
		if override.Status != nil {
			status = *override.Status
		}
		if override.Message != nil {
			message = *override.Message
		}
		if override.Detail != nil {
			detail = *override.Detail
		}
	}

	wtr.WriteHeader(status)
	json.NewEncoder(wtr).Encode(ErrorResponse{
		Status:    status,
		Message:   message,
		Code:      errCode.ID,
		Detail:    detail,
		RequestId: httpReq.GetRequestID(req),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

// Sends a formatted JSON error response with a fully custom status, message, detail, and code.
func SendCustomError(wtr http.ResponseWriter, req *http.Request, status int, message string, detail string, code string) {
	SendError(wtr, req, ErrorCode{ID: code, Status: status, Message: message}, WithDetail(detail))
}

// Returns an ErrorOverride that replaces the default message.
func WithMessage(message string) ErrorOverride { return ErrorOverride{Message: &message} }

// Returns an ErrorOverride that sets the detail field.
func WithDetail(detail string) ErrorOverride { return ErrorOverride{Detail: &detail} }

// Returns an ErrorOverride that replaces the HTTP status code.
func WithStatus(status int) ErrorOverride { return ErrorOverride{Status: &status} }

// Returns an ErrorOverride with message, detail, and status all set at once.
func WithOverrides(message string, detail string, status int) ErrorOverride {
	return ErrorOverride{Message: &message, Detail: &detail, Status: &status}
}

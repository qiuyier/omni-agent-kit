// Package problem writes client-safe RFC 9457-style Hertz responses from
// transport-neutral faults.
package problem

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/cloudwego/hertz/pkg/app"

	"github.com/qiuyier/omni-agent-kit/fault"
)

// MediaType is the registered media type for HTTP Problem Details.
const MediaType = "application/problem+json"

// ErrStreamingResponse indicates that an SSE or other hijacked response has
// already started and can no longer be replaced with a normal HTTP problem.
var ErrStreamingResponse = errors.New("problem: response is already streaming")

// Problem is the public error response. Cause strings, stack traces, provider
// responses, and secrets are intentionally absent.
type Problem struct {
	Type   string     `json:"type"`
	Title  string     `json:"title"`
	Status int        `json:"status"`
	Code   fault.Code `json:"code"`
	Detail string     `json:"detail"`
}

// FromError maps an error to a client-safe problem.
func FromError(err error) Problem {
	if err == nil {
		err = fault.New(fault.CodeInternal, "")
	}
	status := StatusCode(err)
	return Problem{
		Type:   "about:blank",
		Title:  http.StatusText(status),
		Status: status,
		Code:   fault.CodeOf(err),
		Detail: fault.SafeMessage(err),
	}
}

// Write serializes err without exposing its internal cause. It returns
// ErrStreamingResponse if a streaming writer is already installed.
func Write(ctx *app.RequestContext, err error) error {
	if ctx.Response.GetHijackWriter() != nil {
		return ErrStreamingResponse
	}

	publicProblem := FromError(err)
	payload, marshalErr := json.Marshal(publicProblem)
	if marshalErr != nil {
		return marshalErr
	}

	ctx.Response.ResetBody()
	ctx.Abort()
	ctx.Data(publicProblem.Status, MediaType, payload)
	return nil
}

// StatusCode maps transport-neutral fault codes to HTTP status codes.
func StatusCode(err error) int {
	switch fault.CodeOf(err) {
	case fault.CodeCancelled:
		return http.StatusRequestTimeout
	case fault.CodeInvalidArgument:
		return http.StatusBadRequest
	case fault.CodeUnauthenticated:
		return http.StatusUnauthorized
	case fault.CodePermissionDenied:
		return http.StatusForbidden
	case fault.CodeNotFound:
		return http.StatusNotFound
	case fault.CodeConflict:
		return http.StatusConflict
	case fault.CodeFailedPrecondition:
		return http.StatusPreconditionFailed
	case fault.CodeResourceExhausted:
		return http.StatusTooManyRequests
	case fault.CodeDeadlineExceeded:
		return http.StatusGatewayTimeout
	case fault.CodeUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

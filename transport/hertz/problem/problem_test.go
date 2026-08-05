package problem_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/test/mock"

	"github.com/qiuyier/omni-agent-kit/fault"
	"github.com/qiuyier/omni-agent-kit/transport/hertz/problem"
)

func TestWrite(t *testing.T) {
	ctx := app.NewContext(0)
	err := fault.Wrap(fault.CodeUnavailable, "model is temporarily unavailable", errors.New("api_key=secret"))

	if writeErr := problem.Write(ctx, err); writeErr != nil {
		t.Fatalf("Write() error = %v", writeErr)
	}

	if got := ctx.Response.StatusCode(); got != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", got, http.StatusServiceUnavailable)
	}
	if got := string(ctx.Response.Header.ContentType()); got != problem.MediaType {
		t.Fatalf("content-type = %q", got)
	}
	if strings.Contains(string(ctx.Response.Body()), "secret") {
		t.Fatalf("response leaked internal cause: %s", ctx.Response.Body())
	}

	var got problem.Problem
	if err := json.Unmarshal(ctx.Response.Body(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Code != fault.CodeUnavailable || got.Detail != "model is temporarily unavailable" {
		t.Fatalf("problem = %#v", got)
	}
}

func TestWriteRejectsStreamingResponse(t *testing.T) {
	ctx := app.NewContext(0)
	ctx.Response.HijackWriter(&mock.ExtWriter{})

	err := problem.Write(ctx, fault.New(fault.CodeInternal, "failed"))
	if !errors.Is(err, problem.ErrStreamingResponse) {
		t.Fatalf("Write() error = %v, want ErrStreamingResponse", err)
	}
}

func TestUnknownErrorIsInternal(t *testing.T) {
	got := problem.FromError(errors.New("database password=secret"))
	if got.Status != http.StatusInternalServerError {
		t.Fatalf("status = %d", got.Status)
	}
	if got.Detail != "internal error" {
		t.Fatalf("detail = %q", got.Detail)
	}
}

func TestStatusCode(t *testing.T) {
	tests := []struct {
		code fault.Code
		want int
	}{
		{fault.CodeInvalidArgument, http.StatusBadRequest},
		{fault.CodeUnauthenticated, http.StatusUnauthorized},
		{fault.CodePermissionDenied, http.StatusForbidden},
		{fault.CodeNotFound, http.StatusNotFound},
		{fault.CodeConflict, http.StatusConflict},
		{fault.CodeFailedPrecondition, http.StatusPreconditionFailed},
		{fault.CodeResourceExhausted, http.StatusTooManyRequests},
		{fault.CodeDeadlineExceeded, http.StatusGatewayTimeout},
		{fault.CodeUnavailable, http.StatusServiceUnavailable},
		{fault.CodeInternal, http.StatusInternalServerError},
	}

	for _, test := range tests {
		t.Run(string(test.code), func(t *testing.T) {
			if got := problem.StatusCode(fault.New(test.code, "test")); got != test.want {
				t.Fatalf("StatusCode() = %d, want %d", got, test.want)
			}
		})
	}
}

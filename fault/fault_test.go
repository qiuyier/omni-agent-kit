package fault_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/qiuyier/omni-agent-kit/fault"
)

func TestWrapPreservesCauseAndSafeMessage(t *testing.T) {
	cause := errors.New("postgres password=secret")
	err := fault.Wrap(fault.CodeUnavailable, "database is temporarily unavailable", cause)

	if !errors.Is(err, cause) {
		t.Fatal("wrapped cause is not discoverable with errors.Is")
	}
	if got := fault.CodeOf(err); got != fault.CodeUnavailable {
		t.Fatalf("CodeOf() = %q, want %q", got, fault.CodeUnavailable)
	}
	if got := fault.SafeMessage(err); got != "database is temporarily unavailable" {
		t.Fatalf("SafeMessage() = %q", got)
	}
}

func TestUnknownErrorDoesNotLeak(t *testing.T) {
	err := fmt.Errorf("provider rejected api key: %w", errors.New("sk-secret"))

	if got := fault.CodeOf(err); got != fault.CodeInternal {
		t.Fatalf("CodeOf() = %q, want %q", got, fault.CodeInternal)
	}
	if got := fault.SafeMessage(err); got != "internal error" {
		t.Fatalf("SafeMessage() leaked an internal error: %q", got)
	}
}

func TestContextErrorsAreClassified(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code fault.Code
	}{
		{name: "cancelled", err: context.Canceled, code: fault.CodeCancelled},
		{name: "deadline", err: context.DeadlineExceeded, code: fault.CodeDeadlineExceeded},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := fault.CodeOf(test.err); got != test.code {
				t.Fatalf("CodeOf() = %q, want %q", got, test.code)
			}
		})
	}
}

func TestFaultClassificationWinsOverWrappedContext(t *testing.T) {
	err := fault.Wrap(fault.CodeUnavailable, "try later", context.DeadlineExceeded)
	if got := fault.CodeOf(err); got != fault.CodeUnavailable {
		t.Fatalf("CodeOf() = %q, want %q", got, fault.CodeUnavailable)
	}
}

func TestErrorsAsFindsFault(t *testing.T) {
	err := fmt.Errorf("outer: %w", fault.New(fault.CodeNotFound, "document not found"))
	var target *fault.Error
	if !errors.As(err, &target) {
		t.Fatal("errors.As did not find *fault.Error")
	}
	if target.Code() != fault.CodeNotFound {
		t.Fatalf("Code() = %q", target.Code())
	}
}

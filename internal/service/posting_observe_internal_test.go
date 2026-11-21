package service

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
)

func TestPostingFailureClassesAreBounded(t *testing.T) {
	for _, tc := range []struct {
		err   error
		class string
	}{
		{fmt.Errorf("wrapped: %w", context.Canceled), "cancelled"},
		{context.DeadlineExceeded, "timeout"},
		{domain.ErrInsufficientCredit, "credit_limit"},
		{domain.ErrValidation, "validation"},
		{domain.ErrAmountExceeds, "validation"},
		{domain.ErrForbidden, "authorization"},
		{domain.ErrInvalidState, "state"},
		{domain.ErrRateLimited, "rate_limited"},
		{domain.ErrUnavailable, "unavailable"},
		{errors.New("opaque storage details"), "internal"},
		{domain.Errf(domain.Code("not-a-label"), "details"), "internal"},
	} {
		if got := postingFailureClass(tc.err); got != tc.class {
			t.Errorf("%T class=%q, want %q", tc.err, got, tc.class)
		}
	}
	classes := PostingFailureClasses()
	classes[0] = "changed-copy"
	if PostingFailureClasses()[0] == "changed-copy" || NormalizePostingFailureClass("untrusted") != "internal" {
		t.Fatal("caller changed or expanded the fixed metric vocabulary")
	}
}

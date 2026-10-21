package service

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/djshellshoxxx/circula/internal/domain"
)

const (
	maxReasonLen = 500
	maxNoteLen   = 500
	maxTextLen   = 2000
)

// cleanName validates and trims a display name (FR-016).
func cleanName(n string) (string, error) {
	n = strings.TrimSpace(n)
	if c := utf8.RuneCountInString(n); c < 1 || c > 80 {
		return "", domain.Invalid("display_name", "must be 1 to 80 characters")
	}
	for _, r := range n {
		if unicode.IsControl(r) {
			return "", domain.Invalid("display_name", "must not contain control characters")
		}
	}
	return n, nil
}

// requireReason enforces a mandatory bounded reason (adjustments, overrides, suspensions...).
func requireReason(r string) error {
	r = strings.TrimSpace(r)
	if r == "" {
		return domain.Invalid("reason", "required")
	}
	if utf8.RuneCountInString(r) > maxReasonLen {
		return domain.Invalid("reason", "too long")
	}
	return nil
}

func boundedText(field, v string, max int) error {
	if utf8.RuneCountInString(v) > max {
		return domain.Invalid(field, "too long")
	}
	return nil
}

func clampPage(p Page) int {
	if p.Limit <= 0 || p.Limit > 500 {
		return 100
	}
	return p.Limit
}

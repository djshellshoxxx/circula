package domain_test

import (
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
)

func TestValidRecordID(t *testing.T) {
	const v7 = "018f0000-0000-7000-8000-abcdef000001"
	for _, id := range []string{v7, strings.ToUpper(v7), "550e8400-e29b-41d4-a716-446655440000"} {
		if !domain.ValidRecordID(id) {
			t.Errorf("valid UUID text refused: %q", id)
		}
	}
	for _, id := range []string{"", "x' OR '1'='1", "bad\x00id", "inv_secret", "town", "42",
		strings.ReplaceAll(v7, "-", ""), "{" + v7 + "}", "urn:uuid:" + v7,
		v7[:35] + "z", " " + v7, v7 + " ", strings.Replace(v7, "-", "_", 1)} {
		if domain.ValidRecordID(id) {
			t.Errorf("malformed record reference accepted: %q", id)
		}
	}
}

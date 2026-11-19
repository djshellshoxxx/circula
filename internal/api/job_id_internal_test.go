package api

import (
	"testing"

	"github.com/google/uuid"
)

// API-005 changes allocation only: an existing job's externally referenced
// identity must not be regenerated or normalized when serving its result.
func TestJobIDPreservesRecordsAndAllocatesUUIDv7(t *testing.T) {
	for _, existing := range []string{
		"a2f25fea-a94e-47ba-9a86-201104e4e477",
		"018f0000-0000-7000-8000-000000000001",
	} {
		id, err := jobID(existing)
		if err != nil || id != existing {
			t.Fatalf("existing identity changed: got %q, %v, want %q", id, err, existing)
		}
	}
	id, err := jobID("")
	if err != nil {
		t.Fatal(err)
	}
	u, err := uuid.Parse(id)
	if err != nil || u.Version() != 7 || u.Variant() != uuid.RFC4122 || u.String() != id {
		t.Fatalf("new job identifier is not canonical UUIDv7: %q, %v", id, err)
	}
}

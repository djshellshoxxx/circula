package api

import "github.com/google/uuid"

// jobID preserves an existing record's identity. New adapter job identifiers
// follow API-005/DAT-001; legacy UUIDv4 paths remain valid and are not rewritten.
func jobID(existing string) (string, error) {
	if existing != "" {
		return existing, nil
	}
	id, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	return id.String(), nil
}

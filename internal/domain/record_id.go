package domain

// ValidRecordID checks the 36-character UUID text used for record references
// (SEC-024). Legacy UUIDv4 jobs and new UUIDv7 records share this representation.
// It does not rewrite case or accept compact, braced or URN spellings. Tokens,
// network slugs and numeric sequence cursors have separate input contracts.
func ValidRecordID(id string) bool {
	if len(id) != 36 {
		return false
	}
	for i := range len(id) {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if id[i] != '-' {
				return false
			}
			continue
		}
		c := id[i]
		hexDigit := '0' <= c && c <= '9' || 'a' <= c && c <= 'f' || 'A' <= c && c <= 'F'
		if !hexDigit {
			return false
		}
	}
	return true
}

package service

import (
	"strings"
	"testing"
	"time"
)

// RB-017: the time zone validator must never panic and must never accept a
// name that could reach the file system or the process-local zone.
func FuzzValidTimeZone(f *testing.F) {
	for _, s := range []string{"", "UTC", "Europe/Berlin", "America/Argentina/Buenos_Aires", "Local", "europe/berlin", "../etc/passwd",
		"/etc/localtime", "Europe//Berlin", "Etc/GMT+5", "A", "Europe/Berlin\x00", "Ünicode/Zone", strings.Repeat("A/", 40)} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		if !ValidTimeZone(s) {
			return
		}
		if s == "" {
			return
		}
		if s == "Local" || len(s) > 64 || strings.Contains(s, "..") || strings.HasPrefix(s, "/") || strings.ContainsAny(s, "\x00\\") {
			t.Fatalf("accepted %q", s)
		}
		if _, err := time.LoadLocation(s); err != nil {
			t.Fatalf("accepted %q but it does not load: %v", s, err)
		}
	})
}

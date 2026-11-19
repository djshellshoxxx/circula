package service

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
)

// API-035/036: a page boundary is canonical, bounded public metadata. Parsing
// must not panic, normalize alternate spellings, or accept credential shapes.
func FuzzSessionPosition(f *testing.F) {
	for _, pos := range []string{
		"", "0:" + strings.Repeat("a", 64), "-1:" + strings.Repeat("f", 64),
		"9223372036854775807:" + strings.Repeat("0", 64),
		"-9223372036854775808:" + strings.Repeat("f", 64),
		"+1:" + strings.Repeat("a", 64), "01:" + strings.Repeat("a", 64),
		"1:" + strings.Repeat("A", 64), "1:" + strings.Repeat("a", 63),
		"1:" + strings.Repeat("a", 65), "1:" + strings.Repeat("a", 64) + "\x00",
		strings.Repeat("9", 4096) + ":" + strings.Repeat("a", 64),
	} {
		f.Add(pos)
	}
	f.Fuzz(func(t *testing.T, pos string) {
		got, err := parseSessionPosition(pos)
		if err != nil {
			if domain.CodeOf(err) != domain.CodeValidation {
				t.Fatalf("unexpected cursor error: %v", err)
			}
			return
		}
		if pos == "" {
			if got != nil {
				t.Fatal("empty position did not select the first page")
			}
			return
		}
		if got == nil || len(pos) > 85 || !validSessionSelector(got.PublicID) {
			t.Fatalf("accepted malformed cursor %q: %+v", pos, got)
		}
		if sessionPosition(got.CreatedAt, got.PublicID) != pos {
			t.Fatalf("accepted a noncanonical position: %q", pos)
		}
	})
}

func FuzzSessionSelector(f *testing.F) {
	for _, id := range []string{"", strings.Repeat("a", 64), strings.Repeat("f", 64),
		strings.Repeat("0", 64), strings.Repeat("A", 64), strings.Repeat("a", 63),
		strings.Repeat("a", 65), strings.Repeat("a", 63) + "\x00", strings.Repeat("é", 32)} {
		f.Add(id)
	}
	f.Fuzz(func(t *testing.T, id string) {
		want := len(id) == 64 && strings.Trim(id, "0123456789abcdef") == ""
		if got := validSessionSelector(id); got != want {
			t.Fatalf("selector acceptance disagrees with the public contract: %q", id)
		}
	})
}

func TestSessionPositionPreservesInt64Boundaries(t *testing.T) {
	for _, millis := range []int64{-1 << 63, -1, 0, 1, 1<<63 - 1} {
		pos := strconv.FormatInt(millis, 10) + ":" + strings.Repeat("a", 64)
		got, err := parseSessionPosition(pos)
		if err != nil || got == nil || !got.CreatedAt.Equal(time.UnixMilli(millis)) || sessionPosition(got.CreatedAt, got.PublicID) != pos {
			t.Fatalf("millisecond boundary %d did not round trip: %+v, %v", millis, got, err)
		}
	}
}

// Report the actual production parser's allocation cost. Oversized input must
// be rejected before strconv can copy its entire invalid numeric prefix.
func BenchmarkSessionPositionRejectedOversize(b *testing.B) {
	pos := strings.Repeat("9", 1<<20) + ":" + strings.Repeat("a", 64)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if _, err := parseSessionPosition(pos); err == nil {
			b.Fatal("oversized position accepted")
		}
	}
}

func TestSessionPositionRejectsOversizeWithoutCopyingInput(t *testing.T) {
	pos := strings.Repeat("9", 1<<20) + ":" + strings.Repeat("a", 64)
	result := testing.Benchmark(func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if _, err := parseSessionPosition(pos); err == nil {
				b.Fatal("oversized position accepted")
			}
		}
	})
	// Permit normal validation-error overhead while rejecting an allocation
	// proportional to the attacker's 1 MiB numeric prefix.
	if allocated := result.AllocedBytesPerOp(); allocated > 4096 {
		t.Fatalf("oversized cursor rejection allocated %d bytes per call, want at most 4096", allocated)
	}
}

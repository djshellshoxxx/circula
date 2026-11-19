package messages

import "testing"

// UX-047: time-zone UI text keeps IANA IDs literal and retains the existing
// display-only scope, validation guidance and save feedback.
func TestTimeZoneMessages(t *testing.T) {
	for _, tc := range []struct {
		key  string
		args []any
		want string
	}{
		{"time_zone.display", []any{"19 Nov 2026, 17:30 IST", "Asia/Kolkata"}, "Pages show times in your time zone. The API and the audit log always use UTC. Now it is 19 Nov 2026, 17:30 IST in Asia/Kolkata."},
		{"time_zone.hint", nil, "An IANA name such as Europe/Berlin, Asia/Kolkata or America/Argentina/Buenos_Aires. Leave empty for UTC."},
		{"time_zone.save", nil, "Save time zone"},
		{"time_zone.invalid_summary", nil, "Not saved. Fix the problem below. Nothing changed."},
		{"time_zone.invalid_field", nil, "Time zone: enter a name such as Europe/Berlin or America/Mexico_City, or leave it empty for UTC."},
		{"time_zone.saved", nil, "Time zone saved."},
	} {
		t.Run(tc.key, func(t *testing.T) {
			if got := Text(tc.key, tc.args...); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
	const unsafe = `Zoë 100% $& "><img src=x>` + "\x00%!"
	if got := Text("time_zone.display", "TIME", unsafe); got != "Pages show times in your time zone. The API and the audit log always use UTC. Now it is TIME in "+unsafe+"." {
		t.Fatalf("a positional value was interpreted: %q", got)
	}
}

func TestAroundTwoKeepsTimeAndZoneSlots(t *testing.T) {
	parts := AroundTwo("time_zone.display")
	if parts.Before != "Pages show times in your time zone. The API and the audit log always use UTC. Now it is " || parts.Between != " in " || parts.After != "." {
		t.Fatalf("status paragraph changed around its elements: %#v", parts)
	}
}

func TestAroundTwoRejectsMalformedSlots(t *testing.T) {
	for name, text := range map[string]string{
		"missing both":      "No elements.",
		"missing first":     secondTextSlot,
		"missing second":    firstTextSlot,
		"repeated first":    firstTextSlot + firstTextSlot + secondTextSlot,
		"repeated second":   firstTextSlot + secondTextSlot + secondTextSlot,
		"reversed":          secondTextSlot + firstTextSlot,
		"format error":      "%!d(string=" + firstTextSlot + ") " + secondTextSlot,
		"unexpected marker": firstTextSlot + textSlot + secondTextSlot,
	} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("malformed two-element message was accepted")
				}
			}()
			splitAroundTwo(text)
		})
	}
	for _, key := range []string{"account.time_zone", "freshness.snapshot", "overview.pending"} {
		t.Run(key, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("message without two ordered string arguments was accepted")
				}
			}()
			AroundTwo(key)
		})
	}
}

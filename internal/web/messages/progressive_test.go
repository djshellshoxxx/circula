package messages

import "testing"

// UX-047: the catalogue owns complete sentences and keeps substituted text
// literal. Every browser placeholder must occur exactly once.
func TestProgressiveMessages(t *testing.T) {
	const value = `Zoë 100% $& $' $` + "`" + ` $$ "><img src=x>`
	for _, tc := range []struct{ key, want string }{
		{"freshness.snapshot", "Page generated " + value + ". Figures are a snapshot and do not update automatically."},
		{"freshness.offline", "The browser reports that the connection is unavailable. The figures have not refreshed. " + value},
		{"freshness.history", "This earlier page was restored from browser history. The figures have not refreshed. " + value},
		{"session_warning.extendable", "It ends around " + value + ". Choose Stay signed in to keep working."},
		{"session_warning.fixed", "It ends around " + value + " and cannot be extended. Finish what you are doing, then sign in again."},
	} {
		t.Run(tc.key, func(t *testing.T) {
			if got := Text(tc.key, value); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
			parts := Around(tc.key)
			if parts.Before+value+parts.After != tc.want {
				t.Fatalf("single-slot sentence changed: %#v", parts)
			}
		})
	}
	for key, want := range map[string]string{
		"session_warning.ended":  "Your session has ended. Sign in again to continue. Your changes on this page have not been sent.",
		"session_warning.failed": "Could not extend the session. Sign in again.",
	} {
		if got := Text(key); got != want {
			t.Errorf("%s: got %q, want %q", key, got, want)
		}
	}
}

func TestAroundRejectsMissingRepeatedOrInvalidSlots(t *testing.T) {
	for _, formatted := range []string{
		"No placeholder.",
		"One " + textSlot + " and another " + textSlot + ".",
		"Invalid %!d(string=" + textSlot + ").",
	} {
		t.Run(formatted, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("invalid single-slot sentence was accepted")
				}
			}()
			splitAround(formatted)
		})
	}
	for _, key := range []string{"common.skip", "queue.party_proposal", "overview.pending"} {
		t.Run(key, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("message without exactly one string argument was accepted")
				}
			}()
			Around(key)
		})
	}
}

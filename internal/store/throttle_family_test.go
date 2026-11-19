package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/store"
)

// A3-F06: throttle families are literal strings separated from address keys
// by one space. SQLite substr counts characters, including for Unicode logins.
func TestDeleteThrottleFamilyUnicodeAndLiteralPrefix(t *testing.T) {
	for _, key := range []string{
		"alice@example.org",
		"élise@example.org",
		"用戶@example.org",
		"recover: élise@example.org",
		"a%_b@example.org",
	} {
		t.Run(key, func(t *testing.T) {
			s := openStore(t)
			ctx := context.Background()
			until := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
			rows := []struct {
				login string
				keep  bool
			}{
				{key, false},
				{key + " 192.0.2.1", false},
				{key + " 2001:db8::1", false},
				{key + ".suffix 192.0.2.1", true},
				{"prefix-" + key + " 192.0.2.1", true},
				{key + "\t192.0.2.1", true},
				{"a12b@example.org 192.0.2.1", true}, // % and _ must not be LIKE wildcards.
			}
			if err := s.Update(ctx, func(tx store.Tx) error {
				for _, row := range rows {
					if err := tx.PutThrottle(row.login, 5, until); err != nil {
						return err
					}
				}
				return nil
			}); err != nil {
				t.Fatal(err)
			}
			for attempt, want := range []int64{3, 0} {
				if err := s.Update(ctx, func(tx store.Tx) error {
					got, err := tx.DeleteThrottleFamily(key)
					if err == nil && got != want {
						t.Errorf("attempt %d removed %d rows, want %d", attempt, got, want)
					}
					return err
				}); err != nil {
					t.Fatal(err)
				}
			}
			if err := s.View(ctx, func(tx store.Tx) error {
				for _, row := range rows {
					failures, lockedUntil, found, err := tx.GetThrottle(row.login)
					if err != nil {
						return err
					}
					if found != row.keep {
						t.Errorf("row %q exists=%v, want %v", row.login, found, row.keep)
					}
					if found && row.keep && (failures != 5 || !lockedUntil.Equal(until)) {
						t.Errorf("unrelated throttle %q was changed", row.login)
					}
				}
				return nil
			}); err != nil {
				t.Fatal(err)
			}
		})
	}
}

package store

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
)

// A query may return a valid first row and fail on its next read. That is an
// incomplete result, not a shorter successful page. Exercise the real SQLite
// driver through the same late-read wrapper used for migration recovery; this
// qualifies driver-boundary handling, not physical disk or fsync failures.
func TestMemberReadFailureAbortsAndRecovers(t *testing.T) {
	for _, tc := range []struct {
		name string
		read func(Tx) ([]*domain.Member, error)
	}{
		{"network", func(tx Tx) ([]*domain.Member, error) {
			return tx.ListMembers("net-a", MemberFilter{})
		}},
		{"memberships", func(tx Tx) ([]*domain.Member, error) {
			return tx.ListMembershipsOfPerson("person-a")
		}},
		{"forward_page", func(tx Tx) ([]*domain.Member, error) {
			return tx.(MemberPager).PageMembers("net-a", MemberPageFilter{Limit: 10})
		}},
		{"backward_page", func(tx Tx) ([]*domain.Member, error) {
			return tx.(MemberPager).PageMembers("net-a", MemberPageFilter{BeforeID: "member-z", Limit: 10})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			path := filepath.Join(t.TempDir(), "member-read.db")
			st, faults := openFaultStore(t, path)
			// Set WAL and foreign-key enforcement before the lazy pool opens.
			// The migrated database and all statements remain real SQLite.
			faults.dsn += "?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=foreign_keys(1)&_pragma=trusted_schema(0)"
			if err := st.migrateTo(ctx, 0); err != nil {
				t.Fatal(err)
			}
			seedMemberReadFailure(t, st)
			want := errors.New("injected member row read failure")
			faults.readPrefix, faults.readErr = "SELECT "+memberCols+" FROM members m", want
			var got []*domain.Member
			err := st.Update(ctx, func(tx Tx) error {
				// Stage an ordinary write before enumeration. A partial successful
				// result would commit it; the read failure must roll it back.
				if err := tx.SetDefaultFlag("net-a", "member-a", true); err != nil {
					return err
				}
				var err error
				got, err = tc.read(tx)
				return err
			})
			if !errors.Is(err, want) {
				t.Errorf("late member read error not propagated: %v", err)
			}
			if len(got) != 0 {
				t.Errorf("incomplete member read returned %d records", len(got))
			}
			if faults.readErr != nil {
				t.Fatal("member query did not reach the armed read boundary")
			}
			var flag int
			if err := st.db.QueryRowContext(ctx, "SELECT default_flag FROM members WHERE id='member-a'").Scan(&flag); err != nil {
				t.Fatal(err)
			}
			if flag != 0 {
				t.Errorf("write committed after incomplete member read: flag=%d", flag)
			}
			checkRead := func(tx Tx) error {
				members, err := tc.read(tx)
				if err != nil {
					return err
				}
				if len(members) != 2 {
					t.Errorf("healthy read returned %d members, want 2", len(members))
				}
				for _, m := range members {
					if m.DefaultFlag {
						t.Errorf("uncommitted flag survived for member %s", m.ID)
					}
					if len(m.Roles) != 1 || m.Roles[0] != domain.RoleReviewer {
						t.Errorf("healthy member %s has roles %v", m.ID, m.Roles)
					}
				}
				return nil
			}
			if err := st.View(ctx, checkRead); err != nil {
				t.Fatalf("read did not recover on the pooled connection: %v", err)
			}
			if err := st.Close(); err != nil {
				t.Fatal(err)
			}
			reopened, err := Open(ctx, path)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = reopened.Close() })
			if err := reopened.View(ctx, checkRead); err != nil {
				t.Fatalf("read did not recover after reopen: %v", err)
			}
		})
	}
}

func seedMemberReadFailure(t *testing.T, st *SQLite) {
	t.Helper()
	at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	err := st.Update(context.Background(), func(tx Tx) error {
		for _, id := range []string{"person-a", "person-b"} {
			if err := tx.InsertPerson(&domain.Person{ID: id, Login: id, CreatedAt: at}, "unused"); err != nil {
				return err
			}
		}
		for _, id := range []string{"net-a", "net-b"} {
			if err := tx.InsertNetwork(&domain.Network{ID: id, Name: id, Slug: id, Status: domain.NetworkActive, CreatedAt: at}); err != nil {
				return err
			}
			if err := tx.AppendPolicy(id, 1, domain.DefaultPolicy(), "", "", at); err != nil {
				return err
			}
		}
		for _, m := range []domain.Member{
			{ID: "member-a", NetworkID: "net-a", PersonID: "person-a"},
			{ID: "member-b", NetworkID: "net-a", PersonID: "person-b"},
			{ID: "member-c", NetworkID: "net-b", PersonID: "person-a"},
		} {
			m.DisplayName, m.Status, m.JoinMethod, m.JoinedAt = m.ID, domain.MemberActive, "founder", at
			if err := tx.InsertMember(&m); err != nil {
				return err
			}
			if err := tx.GrantRole("role-"+m.ID, m.NetworkID, m.ID, domain.RoleReviewer, "", at); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

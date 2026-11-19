package store_test

import (
	"context"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

func TestSessionPublicIDsPersistWithoutChangingCredentials(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "v19.db")
	legacy, err := store.OpenAtVersion(ctx, path, 19)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := legacy.DB().Exec(`INSERT INTO persons(id,login,password_hash,created_at) VALUES ('owner','owner','hash',0);
		INSERT INTO sessions(id,person_id,created_at,last_seen_at,expires_at,revoked_at) VALUES
		('private-live','owner',100,200,300,NULL),('private-dead','owner',101,201,301,250)`); err != nil {
		t.Fatal(err)
	}
	if err := legacy.Close(); err != nil {
		t.Fatal(err)
	}
	want := map[string]*domain.Session{
		"private-live": {IDHash: "private-live", PersonID: "owner", CreatedAt: time.UnixMilli(100).UTC(), LastSeenAt: time.UnixMilli(200).UTC(), ExpiresAt: time.UnixMilli(300).UTC()},
		"private-dead": {IDHash: "private-dead", PersonID: "owner", CreatedAt: time.UnixMilli(101).UTC(), LastSeenAt: time.UnixMilli(201).UTC(), ExpiresAt: time.UnixMilli(301).UTC()},
	}
	revoked := time.UnixMilli(250).UTC()
	want["private-dead"].RevokedAt = &revoked
	for pass := 0; pass < 2; pass++ {
		upgraded, err := store.Open(ctx, path)
		if err != nil {
			t.Fatal(err)
		}
		if err := upgraded.View(ctx, func(tx store.Tx) error {
			seen := map[string]bool{}
			for id, original := range want {
				got, err := tx.GetSession(id)
				if err != nil {
					return err
				}
				if len(got.PublicID) != 64 || strings.Trim(got.PublicID, "0123456789abcdef") != "" || seen[got.PublicID] {
					t.Fatalf("missing, malformed or duplicated public ID: %+v", got)
				}
				seen[got.PublicID] = true
				if pass == 0 {
					original.PublicID = got.PublicID
				}
				if !reflect.DeepEqual(got, original) {
					t.Fatalf("migration or restart changed session state: got %+v want %+v", got, original)
				}
			}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		if pass == 0 {
			// Existing INSERT shape still creates complete public identities,
			// including multiple rows in one statement.
			if _, err := upgraded.DB().Exec(`INSERT INTO sessions(id,person_id,created_at,last_seen_at,expires_at)
				VALUES ('legacy-insert-a','owner',1,2,3),('legacy-insert-b','owner',1,2,3)`); err != nil {
				t.Fatal(err)
			}
			var complete int
			if err := upgraded.DB().QueryRow(`SELECT COUNT(DISTINCT public_id) FROM sessions WHERE length(public_id)=64`).Scan(&complete); err != nil || complete != 4 {
				t.Fatalf("legacy INSERT did not assign unique public IDs: %d, %v", complete, err)
			}
		}
		if err := upgraded.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestSessionPublicIDIdentityAndReplacementGuards(t *testing.T) {
	s := openStore(t)
	if _, err := s.DB().Exec(`INSERT INTO persons(id,login,password_hash,created_at) VALUES ('owner','owner','hash',0),('other','other','hash',0);
		INSERT INTO sessions(id,person_id,created_at,last_seen_at,expires_at) VALUES ('original','owner',1,2,3000)`); err != nil {
		t.Fatal(err)
	}
	var publicID string
	if err := s.DB().QueryRow(`SELECT public_id FROM sessions WHERE id='original'`).Scan(&publicID); err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		`UPDATE sessions SET public_id=NULL WHERE id='original'`,
		`UPDATE sessions SET public_id='' WHERE id='original'`,
		`UPDATE sessions SET public_id=lower(hex(randomblob(32))) WHERE id='original'`,
		`UPDATE sessions SET id='new-private-hash' WHERE id='original'`,
		`UPDATE sessions SET person_id='other' WHERE id='original'`,
		`UPDATE sessions SET created_at=4 WHERE id='original'`,
		`INSERT OR REPLACE INTO sessions(id,person_id,created_at,last_seen_at,expires_at) VALUES ('original','other',1,2,3000)`,
		`INSERT OR REPLACE INTO sessions(id,person_id,created_at,last_seen_at,expires_at,public_id)
		 SELECT 'replacement','other',created_at,last_seen_at,expires_at,public_id FROM sessions WHERE id='original'`,
		`INSERT INTO sessions(id,person_id,created_at,last_seen_at,expires_at) VALUES (NULL,'owner',1,2,3000)`,
	} {
		if _, err := s.DB().Exec(q); err == nil {
			t.Fatalf("identity rewrite accepted: %s", q)
		}
	}
	for _, id := range []any{nil, "short", strings.Repeat("A", 64), strings.Repeat("g", 64), strings.Repeat("a", 64) + "\x00", []byte(strings.Repeat("a", 64))} {
		if _, err := s.DB().Exec(`INSERT INTO sessions(id,person_id,created_at,last_seen_at,expires_at,public_id)
		 VALUES ('invalid','owner',1,2,3000,?)`, id); err == nil {
			t.Fatalf("malformed public ID accepted: %#v", id)
		}
	}
	if _, err := s.DB().Exec(`UPDATE sessions SET last_seen_at=5,revoked_at=6 WHERE id='original'`); err != nil {
		t.Fatal(err)
	}
	var gotID, owner, gotPublic string
	var created, seen, revoked, count int
	if err := s.DB().QueryRow(`SELECT id,person_id,public_id,created_at,last_seen_at,revoked_at FROM sessions WHERE id='original'`).Scan(
		&gotID, &owner, &gotPublic, &created, &seen, &revoked); err != nil {
		t.Fatal(err)
	}
	if err := s.DB().QueryRow(`SELECT COUNT(*) FROM sessions`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if gotID != "original" || owner != "owner" || gotPublic != publicID || created != 1 || seen != 5 || revoked != 6 || count != 1 {
		t.Fatal("identity guard failure or ordinary touch/revoke changed the public selector")
	}
}

package service_test

import (
	"testing"

	"github.com/djshellshoxxx/circula/internal/repository"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
	"github.com/google/uuid"
)

func persistedPersonID(t *testing.T, e *tu.Env, login string) string {
	t.Helper()
	var id string
	err := e.Store.View(e.Ctx, func(tx repository.Tx) error {
		person, _, err := tx.GetPersonByLogin(login)
		if err != nil {
			return err
		}
		id = person.ID
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return id
}

// DAT-001: a person created with the service's production default generator
// has canonical, lowercase, 36-character UUIDv7 text in persisted domain data.
func TestDefaultIDGeneratorPersistsCanonicalUUIDv7(t *testing.T) {
	e := tu.NewEnvConfig(t, nil, func(c *service.Config) { c.IDs = nil })
	created := tu.Must(e.Svc.RegisterPerson(e.Ctx, service.RegisterInput{
		Login: "default-id@example.org", Password: "correct horse battery",
	}))
	id := persistedPersonID(t, e, created.Login)
	if id != created.ID {
		t.Fatalf("persisted person ID = %q, returned %q", id, created.ID)
	}
	parsed, err := uuid.Parse(id)
	if err != nil {
		t.Fatalf("default person ID %q is not a UUID: %v", id, err)
	}
	if len(id) != 36 || parsed.String() != id || parsed.Version() != 7 || parsed.Variant() != uuid.RFC4122 {
		t.Fatalf("default person ID %q is not canonical UUIDv7 (version=%d, variant=%d)", id, parsed.Version(), parsed.Variant())
	}
}

type personIDGeneratorFunc func() string

func (f personIDGeneratorFunc) New() string { return f() }

// DAT-002: substituting Config.IDs supplies the actual persisted person ID;
// the database does not replace it with an independently allocated identifier.
func TestInjectedIDGeneratorReachesPersistedPerson(t *testing.T) {
	const want = "019bd6f6-4000-7000-8000-000000000123"
	fallback := &tu.IDs{}
	first := true
	ids := personIDGeneratorFunc(func() string {
		if first {
			first = false
			return want
		}
		return fallback.New()
	})
	e := tu.NewEnvConfig(t, nil, func(c *service.Config) { c.IDs = ids })
	created := tu.Must(e.Svc.RegisterPerson(e.Ctx, service.RegisterInput{
		Login: "injected-id@example.org", Password: "correct horse battery",
	}))
	if created.ID != want {
		t.Fatalf("returned person ID = %q, want injected %q", created.ID, want)
	}
	if got := persistedPersonID(t, e, created.Login); got != want {
		t.Fatalf("persisted person ID = %q, want injected %q", got, want)
	}
}

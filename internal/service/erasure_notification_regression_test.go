package service_test

import (
    "strings"
    "testing"

    "github.com/djshellshoxxx/circula/internal/domain"
    "github.com/djshellshoxxx/circula/internal/service"
    tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// Regression: erasing a member must redact their identifying text from a
// counterparty's trade/dispute notices, without destroying unrelated notices.
func TestErasureRedactsCounterpartyNotificationsWithoutCollateral(t *testing.T) {
    w, _ := erasureWorld(t)
    before := tu.Must(w.Svc.ListNotifications(w.Ctx, w.b, w.net.ID, service.NotificationQuery{}))
    named := make(map[string]bool)
    for _, n := range before {
        if strings.Contains(n.Body, anaLogin) {
            named[n.ID] = true
        }
    }
    if len(named) == 0 {
        t.Fatal("fixture did not create a named counterparty notice")
    }

    const unrelatedID = "erasure-unrelated-notice"
    const unrelatedBody = "Unrelated member business"
    _, err := w.Store.DB().Exec(
        "INSERT INTO notifications(id,network_id,member_id,kind,title,body,link,dedupe_key,created_at) VALUES (?,?,?,?,?,?,?,?,?)",
        unrelatedID, w.net.ID, w.bM.ID, "manual.unrelated", "Other notice", unrelatedBody, "/unrelated", "", int64(0),
    )
    if err != nil {
        t.Fatal(err)
    }

    w.depart(w.admin, w.aM)
    tu.Must(w.Svc.AnonymiseMember(w.Ctx, domain.SystemActor("cli"), w.net.ID, w.aM.ID, "erasure request"))
    after := tu.Must(w.Svc.ListNotifications(w.Ctx, w.b, w.net.ID, service.NotificationQuery{}))
    found := make(map[string]bool)
    unrelatedFound := false
    for _, n := range after {
        if named[n.ID] {
            found[n.ID] = true
            if n.Body != "" {
                t.Errorf("named counterparty notification %s retained body %q", n.ID, n.Body)
            }
        }
        if n.ID == unrelatedID {
            unrelatedFound = true
            if n.Body != unrelatedBody {
                t.Errorf("unrelated notification body = %q, want %q", n.Body, unrelatedBody)
            }
        }
    }
    for id := range named {
        if !found[id] {
            t.Errorf("named counterparty notification %s disappeared", id)
        }
    }
    if !unrelatedFound {
        t.Error("unrelated notification disappeared")
    }
}

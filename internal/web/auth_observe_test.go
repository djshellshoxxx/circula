package web

import (
	"net/url"
	"sync/atomic"
	"testing"

	"github.com/djshellshoxxx/circula/internal/web/messages"
)

type webAuthenticationCounts struct{ managementFailures atomic.Int64 }

func (*webAuthenticationCounts) Posted(int)   {}
func (*webAuthenticationCounts) Replayed(int) {}
func (o *webAuthenticationCounts) AuthenticationFailed(reason string) {
	if reason == "mfa_management" {
		o.managementFailures.Add(1)
	}
}
func (*webAuthenticationCounts) AuthenticationThrottled(string) {}

// OPS-014: a current-password refusal keeps its normal HTML 422 response and
// field message while the service observer records the failed proof.
func TestAuthenticationObserverSeesHTMLCredentialValidation(t *testing.T) {
	w := newWorld(t)
	b := w.browser(t)
	b.login("alice@example.org")
	form := b.get("/account/security/mfa/enrol")
	o := &webAuthenticationCounts{}
	w.env.Svc.SetObserver(o)
	refused := b.post("/account/security/mfa/enrol", form, url.Values{"password": {w.env.Key()}})
	if refused.Status != 422 || o.managementFailures.Load() != 1 {
		t.Fatalf("credential refusal status=%d observations=%d", refused.Status, o.managementFailures.Load())
	}
	b.mustContain(refused, "id=\"error-summary\"", messages.Text("auth.field.password_wrong"))
}

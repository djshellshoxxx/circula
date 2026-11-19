package web

import (
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
)

// UX-047: onboarding keeps accessible names and complete catalogue sentences,
// while contextual template escaping protects names, terms and return paths.
func TestOnboardingPagesCatalogue(t *testing.T) {
	body := renderAuthPage(t, "setup", &page{Title: "Create a network", D: true,
		V:      map[string]string{"name": evil, "decimals": "2"},
		Errors: []fieldErr{{"decimals", "Decimal places: choose a value from 0 to 4."}}})
	wantAll(t, "setup", body, ">Network name</label>", ">Setup secret</label>", ">Credit unit name</label>",
		">Create network</button>", "0 (whole units)", `aria-describedby="e-decimals"`,
		"You will be the first administrator.", "How far below zero a member&#39;s balance may go.")
	if strings.Contains(body, "<script>alert") {
		t.Error("setup name was not escaped")
	}

	body = renderAuthPage(t, "join_anon", &page{Title: "Accept your invitation", Next: evil})
	wantAll(t, "anonymous invitation", body, "You have been invited to join a network.", ">Sign in</a>", ">Create an account</a>")
	if strings.Contains(body, "<script>alert") {
		t.Error("invitation return path was not escaped")
	}

	terms := &domain.Terms{Version: 10000, Body: evil}
	body = renderAuthPage(t, "join", &page{Title: "Accept your invitation", D: newJoinData(evil, terms)})
	wantAll(t, "invitation", body, ">Your display name</label>", "Other members see this name.",
		"Network terms (version 10000)", "network&#39;s rules", ">Join the network</button>")
	if strings.Contains(body, "<script>alert") {
		t.Error("invitation token or terms were not escaped")
	}

	for _, mode := range []domain.JoinMode{domain.JoinInvite, domain.JoinApproval, domain.JoinOpen} {
		d := &joinNetData{NetworkInfo: &service.NetworkInfo{Name: evil, Slug: "town", JoinMode: mode}, joinData: newJoinData("", terms)}
		body = renderAuthPage(t, "joinnet", &page{Title: "Join network", D: d})
		switch mode {
		case domain.JoinInvite:
			wantAll(t, "invite-only network", body, "is invitation only. Ask an administrator for an invitation link.")
		case domain.JoinApproval:
			wantAll(t, "approval network", body, "Your request will wait for an administrator to approve it.", ">Request to join</button>", "Network terms (version 10000)")
		case domain.JoinOpen:
			wantAll(t, "open network", body, "You will join straight away.", ">Join</button>")
		}
		if strings.Contains(body, "<script>alert") {
			t.Errorf("%s: network name or terms were not escaped", mode)
		}
	}
}

package web_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func decodeBody(t *testing.T, body string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(body), &m); err != nil {
		t.Fatalf("not JSON: %v: %s", err, body)
	}
	return m
}

func TestAPILifecycle(t *testing.T) {
	e := newEnv(t)
	const p = "/api/v1/me/favorites"
	r := e.api(t, "alice", "POST", p, map[string]any{"member_id": "bob", "nickname": "Bob", "pinned": false})
	wantStatus(t, r, 201)
	if r.Header.Get("Location") != p+"/bob" || r.Header.Get("Content-Type") != "application/json" {
		t.Fatalf("headers %v", r.Header)
	}
	if m := decodeBody(t, r.Body); m["member_id"] != "bob" || m["nickname"] != "Bob" || m["display_name"] != "Member bob" {
		t.Fatalf("%v", m)
	}
	wantStatus(t, e.api(t, "alice", "POST", p, map[string]any{"member_id": "bob", "nickname": "Bob"}), 200)   // idempotent
	wantStatus(t, e.api(t, "alice", "POST", p, map[string]any{"member_id": "bob", "nickname": "Other"}), 409) // conflict
	wantStatus(t, e.api(t, "alice", "POST", p, map[string]any{"member_id": "alice"}), 422)                    // self
	wantStatus(t, e.api(t, "alice", "POST", p, map[string]any{"member_id": "nobody"}), 422)                   // unknown
	wantStatus(t, e.api(t, "alice", "POST", p, map[string]any{"member_id": "carol", "nickname": "a‮b"}), 422) // bad nick
	wantStatus(t, e.api(t, "alice", "POST", p, map[string]any{"member_id": "carol"}), 201)
	e.api(t, "alice", "POST", p, map[string]any{"member_id": "dave"})
	wantStatus(t, e.api(t, "alice", "PATCH", p+"/dave", map[string]any{"pinned": true}), 204)
	wantStatus(t, e.api(t, "alice", "PATCH", p+"/dave", map[string]any{"nickname": "D"}), 204)
	wantStatus(t, e.api(t, "alice", "PATCH", p+"/dave", map[string]any{}), 422)
	wantStatus(t, e.api(t, "alice", "PATCH", p+"/zed", map[string]any{"pinned": true}), 404)
	wantStatus(t, e.api(t, "alice", "PATCH", p+"/dave", map[string]any{"nickname": "x\u0007", "pinned": false}), 422)
	l := e.api(t, "alice", "GET", p, nil)
	wantStatus(t, l, 200)
	m := decodeBody(t, l.Body)
	data := m["data"].([]any)
	if len(data) != 3 || data[0].(map[string]any)["member_id"] != "dave" || data[0].(map[string]any)["pinned"] != true || data[0].(map[string]any)["nickname"] != "D" {
		t.Fatalf("list %v", m)
	}
	rev := m["revision"].(float64)
	// reorder
	wantStatus(t, e.api(t, "alice", "PUT", p+"/order", map[string]any{"revision": rev - 1, "member_ids": []string{"carol"}}), 409)
	wantStatus(t, e.api(t, "alice", "PUT", p+"/order", map[string]any{"revision": rev, "member_ids": []string{"carol", "carol"}}), 422)
	wantStatus(t, e.api(t, "alice", "PUT", p+"/order", map[string]any{"revision": rev, "member_ids": []string{"alice"}}), 422)
	wantStatus(t, e.api(t, "alice", "PUT", p+"/order", map[string]any{"revision": rev, "member_ids": []string{"carol", "bob"}}), 200)
	m = decodeBody(t, e.api(t, "alice", "GET", p, nil).Body)
	data = m["data"].([]any)
	if data[1].(map[string]any)["member_id"] != "carol" || data[2].(map[string]any)["member_id"] != "bob" {
		t.Fatalf("order %v", data)
	}
	wantStatus(t, e.api(t, "alice", "DELETE", p+"/carol", nil), 204)
	wantStatus(t, e.api(t, "alice", "DELETE", p+"/carol", nil), 204)
	wantStatus(t, e.api(t, "bob", "DELETE", p+"/dave", nil), 204) // bob's own (empty) list: no effect on alice
	if d := decodeBody(t, e.api(t, "alice", "GET", p, nil).Body)["data"].([]any); len(d) != 2 {
		t.Fatalf("remove scope: %v", d)
	}
}

func TestAPIPagingAndLimits(t *testing.T) {
	e := newEnv(t)
	const p = "/api/v1/me/favorites"
	for _, id := range []string{"bob", "carol", "dave", "erin"} {
		wantStatus(t, e.api(t, "alice", "POST", p, map[string]any{"member_id": id}), 201)
	}
	var seen []string
	cur := ""
	for i := 0; i < 10; i++ {
		path := p + "?limit=3"
		if cur != "" {
			path += "&cursor=" + cur
		}
		m := decodeBody(t, e.api(t, "alice", "GET", path, nil).Body)
		for _, d := range m["data"].([]any) {
			seen = append(seen, d.(map[string]any)["member_id"].(string))
		}
		if m["next_cursor"] == nil {
			break
		}
		cur = m["next_cursor"].(string)
	}
	if strings.Join(seen, ",") != "bob,carol,dave,erin" {
		t.Fatalf("%v", seen)
	}
	for _, q := range []string{"?limit=0", "?limit=201", "?limit=abc", "?cursor=!!!"} {
		wantStatus(t, e.api(t, "alice", "GET", p+q, nil), 400)
	}
}

func TestAPIRequestHygiene(t *testing.T) {
	e := newEnv(t)
	const p = "/api/v1/me/favorites"
	post := func(user string, h map[string]string, body string) resp {
		hh := map[string]string{}
		for k, v := range h {
			hh[k] = v
		}
		return e.do(t, user, "POST", p, body, hh)
	}
	good := `{"member_id":"bob"}`
	ct := map[string]string{"Content-Type": "application/json"}
	// CSRF header required for unsafe methods
	r := post("alice", ct, good)
	wantStatus(t, r, 403)
	if r.Header.Get("Content-Type") != "application/problem+json" {
		t.Fatal("problem+json expected")
	}
	if m := decodeBody(t, r.Body); m["status"] != float64(403) || m["type"] != "about:blank" || m["title"] == nil {
		t.Fatalf("%v", m)
	}
	wantStatus(t, post("alice", map[string]string{"Content-Type": "application/json", "X-CSRF-Token": "csrf-bob"}, good), 403)
	h := map[string]string{"Content-Type": "application/json", "X-CSRF-Token": "csrf-alice"}
	wantStatus(t, post("alice", map[string]string{"X-CSRF-Token": "csrf-alice", "Content-Type": "text/plain"}, good), 415)
	wantStatus(t, post("alice", map[string]string{"X-CSRF-Token": "csrf-alice"}, good), 415)
	wantStatus(t, post("alice", h, `{"member_id":"bob","owner":"bob"}`), 400) // unknown field: owner is never client-controlled
	wantStatus(t, post("alice", h, `{"member_id":"bob"} {"x":1}`), 400)
	wantStatus(t, post("alice", h, `{"member_id":`), 400)
	wantStatus(t, post("alice", h, `{"member_id":123}`), 400)
	wantStatus(t, post("alice", h, `{"member_id":"`+strings.Repeat("a", 70000)+`"}`), 413)
	wantStatus(t, post("alice", map[string]string{"X-CSRF-Token": "csrf-alice", "Content-Type": "application/json", "Origin": "https://evil.example"}, good), 403)
	wantStatus(t, post("alice", map[string]string{"X-CSRF-Token": "csrf-alice", "Content-Type": "application/json", "X-Network": "n2"}, good), 403)
	if d := decodeBody(t, e.api(t, "alice", "GET", p, nil).Body)["data"].([]any); len(d) != 0 {
		t.Fatalf("rejected requests created data: %v", d)
	}
	wantStatus(t, post("alice", map[string]string{"X-CSRF-Token": "csrf-alice", "Content-Type": "application/json; charset=utf-8", "X-Network": "n1"}, good), 201)
	// identical-looking path with different method is refused
	wantStatus(t, e.api(t, "alice", "PUT", p, good), 405)
	// errors never leak internals
	e.dir.Fail.Store(true)
	r = e.api(t, "alice", "GET", p, nil)
	wantStatus(t, r, 503)
	if strings.Contains(r.Body, "backend") {
		t.Fatal("policy error text leaked")
	}
}

func TestAPIDrafts(t *testing.T) {
	e := newEnv(t)
	const p = "/api/v1/me/favorites"
	e.api(t, "alice", "POST", p, map[string]any{"member_id": "bob"})
	r := e.api(t, "alice", "POST", p+"/drafts", map[string]any{"trade_id": "t1", "request_key": "k1"})
	wantStatus(t, r, 201)
	d := decodeBody(t, r.Body)
	if d["amount"] != nil || d["recipient_member_id"] != "bob" || d["description"] != "bike <b>repair</b>" || d["source_trade_id"] != "t1" || len(d["idempotency_key"].(string)) != 32 || d["idempotency_key"] == "t1" {
		t.Fatalf("%v", d)
	}
	again := decodeBody(t, e.api(t, "alice", "POST", p+"/drafts", map[string]any{"trade_id": "t1", "request_key": "k1"}).Body)
	other := decodeBody(t, e.api(t, "alice", "POST", p+"/drafts", map[string]any{"trade_id": "t1", "request_key": "k2"}).Body)
	if again["idempotency_key"] != d["idempotency_key"] || other["idempotency_key"] == d["idempotency_key"] {
		t.Fatal("idempotency semantics")
	}
	g := e.api(t, "alice", "GET", p+"/drafts/k1", nil)
	wantStatus(t, g, 200)
	if decodeBody(t, g.Body)["idempotency_key"] != d["idempotency_key"] {
		t.Fatal("get draft")
	}
	wantStatus(t, e.api(t, "bob", "GET", p+"/drafts/k1", nil), 404)
	wantStatus(t, e.api(t, "alice", "POST", p+"/drafts", map[string]any{"trade_id": "t1", "request_key": "k3", "member_id": "bob"}), 422)
	wantStatus(t, e.api(t, "alice", "POST", p+"/drafts", map[string]any{}), 422)
	wantStatus(t, e.api(t, "alice", "POST", p+"/drafts", map[string]any{"trade_id": "nope"}), 422)
	wantStatus(t, e.api(t, "alice", "POST", p+"/drafts", map[string]any{"trade_id": "t1", "request_key": "k1", "member_id": ""}), 201)
	fav := e.api(t, "alice", "POST", p+"/drafts", map[string]any{"member_id": "bob", "request_key": "fav1"})
	wantStatus(t, fav, 201)
	if decodeBody(t, fav.Body)["source_trade_id"] != nil {
		t.Fatal("favorite draft has no source trade")
	}
	e.dir.Block("n1", "alice", "bob")
	wantStatus(t, e.api(t, "alice", "POST", p+"/drafts", map[string]any{"trade_id": "t1"}), 422)
	wantStatus(t, e.api(t, "alice", "GET", p+"/drafts/k1", nil), 404)
	_ = fmt.Sprint
}

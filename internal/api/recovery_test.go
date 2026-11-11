package api_test

import "testing"

// SEC-008: while a person must change the password, API writes refuse with
// 403 password-change-required and reads still work.
func TestForcedPasswordChangeOnAPI(t *testing.T) {
	w := newWorld(t)
	if _, err := w.env.Store.DB().Exec(`UPDATE persons SET must_change_password = 1 WHERE id = ?`, w.alice.PersonID); err != nil {
		t.Fatal(err)
	}
	body := m{"counterparty_member_id": w.bobM.ID, "direction": "pay", "amount_minor": "2500", "description": "garden help"}
	wantProblem(t, w.post(w.aliceT, "/transactions", body, w.key()), 403, "password-change-required")
	wantStatus(t, w.get(w.aliceT, "/accounts/me"), 200)
	// an unflagged person is unaffected
	wantStatus(t, w.post(w.bobT, "/transactions", m{"counterparty_member_id": w.aliceM.ID, "direction": "pay", "amount_minor": "100"}, w.key()), 201)
}

package api

const v1 = "/api/v1"

func (s *server) routes() {
	R := func(pattern string, h handlerFunc, f ...func(*route)) {
		rt := route{pattern: pattern, h: h}
		for _, fn := range f {
			fn(&rt)
		}
		s.add(rt)
	}
	noNet := func(r *route) { r.noNet = true }
	idem := func(r *route) { r.idem = idemRequired }
	idemOpt := func(r *route) { r.idem = idemOptional }
	cookieOnly := func(r *route) { r.cookieOnly = true }

	// Network, identity, directory, account.
	R("GET "+v1+"/network", s.getNetwork)                             // API-040
	R("GET "+v1+"/network/announcements", s.notImpl("announcements")) // API-064
	R("GET "+v1+"/me", s.getMe, noNet)                                // API-041
	R("GET "+v1+"/members", s.listMembers)                            // API-042
	R("GET "+v1+"/members/{id}", s.getMember)                         // API-043
	R("GET "+v1+"/accounts/me", s.getAccount)                         // API-044
	R("GET "+v1+"/accounts/me/entries", s.listEntries)                // API-045
	R("POST "+v1+"/me/membership/departure", s.depart, idem)          // API-067
	R("POST "+v1+"/me/password", s.changePassword, noNet, cookieOnly) // AUTH-03 / SEC-003

	// Listings.
	R("GET "+v1+"/listings", s.listListings)         // API-046
	R("POST "+v1+"/listings", s.createListing, idem) // API-046
	R("PATCH "+v1+"/listings/{id}", s.patchListing)  // API-047

	// Transactions.
	R("POST "+v1+"/transactions", s.propose, idem)                // API-048
	R("GET "+v1+"/transactions", s.listTx)                        // API-049
	R("GET "+v1+"/transactions/{id}", s.getTx)                    // API-049
	R("GET "+v1+"/transactions/{id}/events", s.txEvents)          // API-050
	R("POST "+v1+"/transactions/{id}/accept", s.acceptTx, idem)   // API-051
	R("POST "+v1+"/transactions/{id}/decline", s.declineTx, idem) // API-052
	R("POST "+v1+"/transactions/{id}/cancel", s.cancelTx, idem)   // API-053

	// Disputes.
	R("POST "+v1+"/disputes", s.openDispute, idem)                 // API-054
	R("GET "+v1+"/disputes", s.listDisputes)                       // API-054
	R("GET "+v1+"/disputes/{id}", s.getDispute)                    // API-054
	R("POST "+v1+"/disputes/{id}/resolve", s.resolveDispute, idem) // API-055
	R("POST "+v1+"/disputes/{id}/review", s.reviewDispute)         // extension: start review
	R("POST "+v1+"/disputes/{id}/withdraw", s.withdrawDispute)     // extension: opener withdraws
	R("POST "+v1+"/disputes/{id}/retry-posting", s.retryDispute)   // extension: DOM-021

	// Admin.
	R("POST "+v1+"/admin/transactions/{id}/reversal", s.adminReverse, idem) // API-056
	R("POST "+v1+"/admin/adjustments", s.adminAdjust, idem)                 // API-057
	s.dualControlRoutes(R)
	R("GET "+v1+"/admin/policy", s.getPolicy)         // API-058
	R("PATCH "+v1+"/admin/policy", s.patchPolicy)     // API-058
	R("GET "+v1+"/admin/members", s.adminListMembers) // API-059
	R("POST "+v1+"/admin/members/{id}/approve", s.memberAction("approve"), idem)
	R("POST "+v1+"/admin/members/{id}/deny", s.memberAction("deny"), idem) // FR-011
	R("POST "+v1+"/admin/members/{id}/suspend", s.memberAction("suspend"), idem)
	R("POST "+v1+"/admin/members/{id}/reinstate", s.memberAction("reinstate"), idem)
	R("POST "+v1+"/admin/members/{id}/remove", s.memberAction("remove"), idem)
	R("PUT "+v1+"/admin/members/{id}/roles/{role}", s.roleAction(true))     // extension
	R("DELETE "+v1+"/admin/members/{id}/roles/{role}", s.roleAction(false)) // extension
	R("GET "+v1+"/admin/accounts/{id}", s.adminGetAccount)                  // extension: ETag source
	R("PUT "+v1+"/admin/accounts/{id}/credit-limit", s.putCreditLimit)      // API-060
	R("POST "+v1+"/admin/invitations", s.createInvitation, idem)            // API-061
	R("POST "+v1+"/invitations/{token}/accept", s.acceptInvitation, idem, noNet)
	R("GET "+v1+"/admin/audit-events", s.listAudit)                  // API-062
	R("POST "+v1+"/admin/consistency-checks", s.runCheck, idem)      // API-063
	R("GET "+v1+"/admin/consistency-checks/{id}", s.getCheck)        // API-063
	R("POST "+v1+"/admin/announcements", s.notImpl("announcements")) // API-064
	R("GET "+v1+"/admin/announcements", s.notImpl("announcements"))  // API-064

	// Exports and tokens.
	R("POST "+v1+"/exports", s.createExport, idemOpt)    // API-065
	R("GET "+v1+"/exports/{id}", s.getExport)            // API-065
	R("GET "+v1+"/exports/{id}/artefact", s.getArtefact) // API-065
	R("GET "+v1+"/tokens", s.listTokens, noNet)          // API-066
	R("POST "+v1+"/tokens", s.createToken, noNet, cookieOnly, idemOpt)
	R("DELETE "+v1+"/tokens/{id}", s.revokeToken, noNet)
}

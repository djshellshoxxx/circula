# Dispute validation catalogue completion

The second UI continuation replaces the two remaining amount-precision errors
and the re-review proof refusal in `disputeAct` with complete catalogue messages.
The amount hint and refusal correctly select `1 decimal place`; zero and larger
precision counts retain the plural. The two new keys are
`dispute.amount_invalid` and `dispute.review_again`.

`domain.Invalid(field, detail)` accepts a plain detail string. The existing field
names, amount parser, positive-amount checks, proof comparison, review steps,
service calls and error statuses remain unchanged. No accounting code changes.

## Observed focused check

On the combined source based on `e1f85a3`, the new
`TestDisputeValidationMessages` failed before implementation because
`dispute.amount_invalid` did not exist. After implementation the complete
messages package passed in 0.051s with normal vet enabled:

```
GOMAXPROCS=1 GOMEMLIMIT=250MiB GOGC=40 go test -count=1 ./internal/web/messages
```

The exact before/after output is retained beside this receipt. Existing HTTP
confirmation tests are strengthened to assert 422, the amount field association,
the complete amount/proof refusal, and unchanged dispute/re-review ledger state.
Those expanded HTTP checks require the next combined web qualification; the
messages-package result alone does not qualify them.

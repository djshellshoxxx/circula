# How Mutual Credit Works

**Status: Draft explanation based on the domain specification.**

A network defines an accounting unit and its own participation and credit policies. When a member provides value to another member and both accept a trade, the ledger records equal and opposite changes. The person providing value receives a positive entry; the person receiving value receives a negative entry.

For a trade of 10 units, one account changes by +10 and the other by -10. The network total for the trade remains zero. This accounting record does not guarantee that goods or services have a market price, that a member will accept credits, or that credits can be redeemed for money.

## Balance terms

- **Balance:** sum of an account’s posted ledger entries.
- **Credit limit:** maximum permitted negative balance under network policy.
- **Available credit:** remaining room before reaching the permitted negative limit.
- **Pending proposal:** not yet posted to balances.
- **Reversal or adjustment:** new linked event that corrects or resolves a prior posting; it does not erase history.

See the [domain model](../docs/specs/03-domain-model.md) and [data and ledger requirements](../docs/specs/06-data-ledger-audit.md), which are authoritative for this explanation.

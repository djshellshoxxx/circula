# Crypto and crypto-libertarian design options

**Status:** Conceptual explanation and future options; no cryptographic architecture selected  
**Owner:** Circula project owner  
**Last updated:** 2026-10-20  
**Scope:** Distinguish mutual credit, cryptocurrency, cryptography, and the project’s decentralization goals  
**Dependencies:** [Domain model](../specs/03-domain-model.md), [architecture](../specs/05-architecture.md), [security baseline](../specs/09-security-privacy.md)  
**Open questions:** Which trust problem, if any, justifies each cryptographic feature? What key recovery model is usable for ordinary members?

## What “crypto” means here

Circula’s core accounting model is mutual credit: when two members complete a trade, one account receives a positive entry and the other an equal negative entry. The unit tracks value exchanged under network rules. This can run in a conventional database and does not inherently create a cryptocurrency, a public token, or a claim redeemable for money.

**Cryptography** is a set of tools for proving that a message or record was approved by a key, checking that data has not changed, and protecting data from parties who lack a decryption key. **Cryptocurrency** generally adds a transferable digital asset or token model. **Blockchain** is one way to distribute and order records among parties; it is not synonymous with mutual credit or cryptography.

The strongest crypto-libertarian interpretation of Circula is an infrastructure goal: make it practical for people to associate voluntarily, govern their own networks, verify important records, operate software themselves, interoperate by consent, and exit without losing their data. The social effect would come from making parallel community coordination usable and replicable, not from the word “crypto” or the presence of a token.

## Design goals that fit this perspective

- **Voluntary association:** networks and members choose whether to participate and which networks to trust.
- **Local governance:** participating communities set their own admission, credit-limit, and dispute rules within ledger integrity requirements.
- **Exit and portability:** a network can export and reconcile its records and can change operators or software.
- **No single mandatory issuer:** the platform does not assign a universal value to network credits or require a central authority to authorize ordinary local trades.
- **Interoperability by consent:** networks choose peers, scopes, limits, and exit terms before exchanging data or obligations.
- **Verifiability:** members can check that an approval and accounting event have not been silently changed.
- **Data minimization:** verifiability should not require publishing member identities or transaction details to the world.

These are goals to test. Self-hosting alone does not eliminate administrator power, hosting dependencies, network governance, or legal obligations. Decentralization is a distribution of control and responsibility, not an absence of either.

## Candidate cryptographic capabilities

1. **Signed transaction approvals.** A member key can sign approval for a specific proposal digest, including network, participants, amount, description reference, and expiration. This can make approval verifiable across clients, but requires safe key storage, recovery, and a way to revoke compromised keys.
2. **Tamper-evident event history.** Events can be linked by hashes or periodically signed checkpoints so exported records reveal alteration or omission. This detects some tampering; it does not prevent an operator from withholding data or prove that the original event was honest.
3. **Encrypted backups and dispute evidence.** Sensitive files can be encrypted for authorized roles or stored separately from public ledger metadata. Key recovery and lawful access policies remain difficult; encryption does not eliminate the need for role-based access.
4. **Threshold administrator actions.** High-impact actions, such as a large adjustment or network closure, could require approval from multiple independently held administrator keys. This can reduce unilateral control, but needs emergency recovery, recusal, and availability rules.
5. **Member-controlled credentials.** Members could present limited proofs, such as an approved membership credential, without sharing an entire profile. Data minimization and selective disclosure should be evaluated before choosing a credential system.
6. **Signed federated messages.** Independent network operators can sign messages exchanged under an explicit federation agreement. Each receiving network can verify origin and policy context without trusting one universal server.
7. **Optional public audit anchors.** A network could publish a periodic commitment to its event history without publishing transactions themselves. This may help establish that a later export matches an earlier checkpoint, but can create metadata leakage and a false impression that the ledger is publicly endorsed.

No cryptographic feature should be added without a concrete threat model, usability study, recovery plan, and tests. Cryptographic controls are not substitutes for clear policy or accessible appeal.

## Why a blockchain is not the default

A blockchain may be useful if independent operators need a shared event order and cannot select a trusted coordinator. That need has not yet been established for the MVP. A conventional transactional database is simpler for atomic equal-and-opposite posting, privacy controls, account recovery, correction workflows, and exports.

A public chain can permanently reveal timing, amounts, addresses, and relationship patterns even if names are omitted. Keys can be lost or stolen; smart contracts can contain defects; fees and consensus rules can change; and irreversible finality can conflict with fair correction and dispute processes. A token may also suggest investment, redemption, or universal exchange value that Circula does not promise.

If future networks test a blockchain or token, that work should be separately specified and reviewed. Do not put personal data, private dispute details, or transaction details on a public chain. Do not imply a token is cash-equivalent, guaranteed, or an investment.

## Staged path to investigate

1. Start with an atomic database ledger, append-only accounting events, strong authorization, and complete exports.
2. Add deterministic export verification or signed checkpoints if members need to independently detect record alteration.
3. Test signatures for cross-client transaction approval only if ordinary authentication does not satisfy the threat model.
4. Prototype opt-in federation using signed messages and explicit network agreements before considering a common public ledger.
5. Consider an external-chain anchor only if a concrete independent-verification need outweighs privacy, cost, and permanence concerns.
6. Keep tokenization, exchange markets, redemption, and speculative value outside the foundational design until separately justified.

## Questions before any implementation

- What party is a member trying to avoid having to trust, and what evidence would reduce that trust?
- Can the goal be met with exportable records and ordinary database controls?
- Which party holds keys, and how can a member recover safely or revoke access?
- What data becomes public, linkable, or permanent?
- How does the design handle disputes, corrections, administrator succession, and network exit?
- Does a proposed token introduce expectations of cash value or investment returns?
- Which professional review is required for the actual deployment jurisdictions and product behavior?

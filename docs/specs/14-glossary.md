# 14 — Glossary

| Field | Value |
|---|---|
| Status | Draft |
| Owner role | Product architect |
| Last updated | 2026-10-21 |
| Scope | Alphabetical definitions of every canon term ([00 section 5.1](00-conventions.md)) and of the terms defined in the other specs; "not to be confused with" notes; the table of banned and discouraged wording for product copy and documentation |
| Depends on | [00 Conventions](00-conventions.md) and every spec that owns a term (links in the last column) |
| Open questions | None blocking. Discouraged-wording rationale is design guidance, not a legal conclusion; see [19](19-legal-and-regulatory-open-questions.md) |

## 1. Purpose and scope

This glossary gives one meaning to each term used in the Circula specifications, user-interface copy, help text and contributor documentation. It does **not** create requirements (the `GLO` prefix has no numbering) and it does **not** redefine anything: the canon in [00 section 5](00-conventions.md) wins on any conflict, and each term's owning document wins on detail. Where this glossary and an owner disagree, that is a defect in this glossary; fix it here and note it in the change record of the owning document.

## 2. How to read an entry

- **Term** is spelled exactly as in the canon (capitalisation matters for role and object names: *Member*, *Network*, *Trade*; lower case for generic use). Code-formatted terms (`credit_limit`) are exact parameter, state or field names.
- **Definition** is one sentence. Detail lives in the owner.
- **Not to be confused with** is given where two terms are often mixed up or where an everyday word has a different meaning in Circula.
- **Owner** is the document that defines the term normatively.
- Entries tagged *(UI)* are the user-facing words from [08 section 14](08-ux-and-accessibility.md).

Owner shorthand: [00](00-conventions.md) Conventions · [01](01-product-vision-and-scope.md) Vision · [02](02-personas-roles-and-journeys.md) Personas · [03](03-domain-model-and-mutual-credit-rules.md) Domain · [04](04-functional-requirements.md) Functional · [05](05-architecture-and-trust-boundaries.md) Architecture · [06](06-data-ledger-and-audit.md) Data · [07](07-api-and-integration-contract.md) API · [08](08-ux-and-accessibility.md) UX · [09](09-security-privacy-and-abuse.md) Security · [10](10-governance-and-network-operations.md) Governance · [11](11-deployment-operations-and-support.md) Operations · [15](15-lifecycle-state-machines.md) Lifecycle · [16](16-network-policy-catalog.md) Policy catalog · [17](17-notifications-and-communications.md) Notifications · [18](18-data-export-and-portability.md) Export · [19](19-legal-and-regulatory-open-questions.md) Legal.

## 3. Terms

### A

| Term | Definition | Not to be confused with | Owner |
|---|---|---|---|
| **Account** | The union of Member Accounts and System Accounts; the target of Postings. | A *login* (who signs in) or a *Membership* (relationship to a Network). | [06](06-data-ledger-and-audit.md) |
| **Account serialisation** | Whatever mechanism makes Postings and limit-relevant changes on one Account behave as if executed one at a time. | A user-visible feature. | [06](06-data-ledger-and-audit.md) |
| **Acting capacity** | The single role (Member, Dispute Reviewer, Network Administrator) under which a person performs an action, recorded in the Audit Event. | The person's identity. | [02](02-personas-roles-and-journeys.md) |
| **Acting Membership** | The Membership in the addressed Network under which an API request is authorised, derived server-side. | A client-supplied Network identifier. | [07](07-api-and-integration-contract.md) |
| **Active** (`active`) | Membership state in which a person can use the Network normally. | "Active" in *inactive* (a derived flag). | [15](15-lifecycle-state-machines.md) |
| **Adjustment** | Admin-initiated correcting Journal Entry (type `adjustment`) with mandatory reason, linked Audit Event and Member notification. | A *Reversal* (negates a prior entry) and a *Trade* (needs both parties' consent). Never "correction" alone or "refund" in UI. | [03](03-domain-model-and-mutual-credit-rules.md) |
| **`admin_quorum`** | Policy parameter: number of distinct Network Administrators required for sensitive actions (default 1). | The count of administrators in a Network. | [16](16-network-policy-catalog.md) |
| **Amount** | An integer number of minor units of the Network's credit unit; in the API, a string plus the Network's `unit_scale`. | A decimal or floating-point number. | [00](00-conventions.md), [07](07-api-and-integration-contract.md) |
| **Appeal** | A governance record by which an affected person asks for a Decision to be reviewed by someone who did not take it. | A *Dispute*, which concerns a Trade between Members. | [10](10-governance-and-network-operations.md) |
| **Appeal contact** | A person or body named in the Charter, outside the administrator group, who hears an Appeal when no unconflicted administrator exists. | A Platform support contact; it has no ledger power. | [10](10-governance-and-network-operations.md) |
| **Applicant** | A person with a login and a Membership in state `applicant` in a given Network. | A *Visitor* (unauthenticated). | [02](02-personas-roles-and-journeys.md) |
| **As-of** | The consistent snapshot instant (UTC) from which every dataset in an export package is read. | The time of download. | [18](18-data-export-and-portability.md) |
| **Atomic unit** | One transaction in which state change, ledger posting, Audit Event and history record commit together or not at all. | A *unit of work* (same idea at the data layer). | [15](15-lifecycle-state-machines.md) |
| **Audit Event** | Append-only record of a security- or governance-relevant action naming actor, time, reason and target. | A Journal Entry (ledger), a history record (per-object log), or the security log (operational). | [06](06-data-ledger-and-audit.md) |
| **`available_to_spend`** | `balance + credit_limit`, never shown below 0; pending Trades do not reduce it in MVP (no holds). | "Available funds" (not a UI term). | [00](00-conventions.md) |

### B

| Term | Definition | Not to be confused with | Owner |
|---|---|---|---|
| **Balance** | The sum of Posting amounts on an Account; positive means net provided, negative means net received. | A bank balance; a claim on money; a debt in national currency. | [03](03-domain-model-and-mutual-credit-rules.md) |
| **Balance panel** | The standard UI component showing balance, state label, available-to-spend, credit limit and cap headroom. | The history list. | [08](08-ux-and-accessibility.md) |
| **`balance_visibility`** | Policy parameter (`self` / `admins` / `members`) controlling who may see a Member's balance. | `directory_visibility`. | [16](16-network-policy-catalog.md) |
| **Banned flag** | A flag with reason on a `closed` or `suspended` Membership; not a lifecycle state. | A state in the Membership machine. | [15](15-lifecycle-state-machines.md) |
| **Bounce** | A delivery-status report that an email could not be delivered; hard (permanent) or soft (temporary). | A spam complaint. | [17](17-notifications-and-communications.md) |
| **Break-glass** | Time-boxed, reasoned, second-person-approved, audited Platform Operator access to Network content. | Routine operator access, which does not exist by default. | [09](09-security-privacy-and-abuse.md) |
| **Broadcast** | An administrator-authored plain-text announcement to all active Members of one Network. | A policy-change notice (system-generated, mandatory). | [17](17-notifications-and-communications.md) |

### C

| Term | Definition | Not to be confused with | Owner |
|---|---|---|---|
| **Cached balance** | The balance stored on an Account row, maintained transactionally (D-04); never an authority (INV-008). | The ledger itself. | [06](06-data-ledger-and-audit.md) |
| **Category** (notification) | The preference grouping of a notification event (for example `security`, `trade`, `announcement`). | A Listing category. | [17](17-notifications-and-communications.md) |
| **Channel** | A way a notification is delivered: `in_app` or `email` in MVP; push and SMS deferred. | A support channel. | [17](17-notifications-and-communications.md) |
| **Charter** | The Network's published, versioned governance document (the "rules text"). | The Platform's terms of service. | [10](10-governance-and-network-operations.md) |
| **Charter version** | A numbered edition of the Charter with a publication date; each Member's acceptance is recorded. | A Policy version (parameters only). | [10](10-governance-and-network-operations.md) |
| **`closed`** | Terminal Membership state after departure; personal data is anonymised per D-09. | `closing`, a Network operating status. | [15](15-lifecycle-state-machines.md) |
| **Component / Module** | A logical unit with one responsibility / its code-level realisation inside the deployable. | A separate process or service. | [05](05-architecture-and-trust-boundaries.md) |
| **`community_fund`** | System Account that can receive or absorb residual balances when `departure_settlement_mode` is `community_fund`. | A pool of money; it holds credit units only. | [03](03-domain-model-and-mutual-credit-rules.md) |
| **Copy catalogue** | The single externalised set of all user-visible strings, shared by web client and notifications. | A marketing copy deck. | [08](08-ux-and-accessibility.md) |
| **Counterparty** | In a proposal, the Member who did not propose and must accept. | The *Provider* or *Recipient* (roles in the exchange, independent of who proposed). | [00](00-conventions.md) |
| **Credit limit** (`credit_limit`) | The magnitude of negative balance permitted for a Member Account; floor is `−credit_limit`. | A loan limit or overdraft (UI never says so). | [03](03-domain-model-and-mutual-credit-rules.md) |
| **Credit unit** (unit) | The Network's accounting unit, named per Network (for example hours, seeds); not money, a security or crypto, and not redeemable for national currency by Platform design. | "Currency", "coin", "token", "points". | [00](00-conventions.md) |

### D

| Term | Definition | Not to be confused with | Owner |
|---|---|---|---|
| **Dataset / Package** | One entity-shaped table in an export / the single archive file an export job produces. | A database dump. | [18](18-data-export-and-portability.md) |
| **Decision** | A governance act affecting a named person or object, such as rejection, suspension, removal, Listing removal, limit change, Adjustment, Reversal, Dispute resolution. | A policy change (affects everyone). | [10](10-governance-and-network-operations.md) |
| **`default_credit_limit`** | Policy parameter: credit limit assigned to a Membership on activation. | `max_credit_limit`. | [16](16-network-policy-catalog.md) |
| **`departing`** | Membership state between a departure request or removal and `closed`, during which settlement happens. | `closed`. | [15](15-lifecycle-state-machines.md) |
| **`departure_settlement_mode`** | Policy parameter (`settle` / `write_off` / `community_fund`) for non-zero balances at departure. | Any payment or collection process. | [16](16-network-policy-catalog.md) |
| **Deployment** | One running installation of the Platform, hosting one or more Networks, under one Platform Operator. | A Network. | [05](05-architecture-and-trust-boundaries.md) |
| **Digest** | One message summarising several optional notifications or pending items. | A mandatory notice, which is never digested. | [17](17-notifications-and-communications.md) |
| **`directory_visibility`** | Policy parameter (`members` / `public_listings`) controlling who can see the directory and Listings. | `balance_visibility`. | [16](16-network-policy-catalog.md) |
| **Direction word** | The visible text stating which way value moved for the viewer: Earned, Spent, Added, Deducted. | A sign or colour alone. | [08](08-ux-and-accessibility.md) |
| **Dispute** | A formal challenge to a completed or proposed Trade, reviewed under Network rules. | An *Appeal* (about a Decision) or a *report* (about conduct or content). | [03](03-domain-model-and-mutual-credit-rules.md), [15](15-lifecycle-state-machines.md) |
| **Dispute Reviewer** (`dispute_reviewer`) | Member-role that can review disputes in one Network; cannot review one they are party to. | Network Administrator. | [00](00-conventions.md) |
| **`dispute_review_sla_days` / `dispute_window_days`** | Policy parameters: target days to review a Dispute / days after completion in which a Dispute can be raised. | Legal limitation periods. | [16](16-network-policy-catalog.md) |

### E

| Term | Definition | Not to be confused with | Owner |
|---|---|---|---|
| **Entry hash / Integrity checkpoint** | Digest of one Journal Entry's canonical content / sealed digest and count over all entries or events of a Network at a snapshot. | Cryptocurrency blocks; there is no chain of value. | [06](06-data-ledger-and-audit.md) |
| **Error code** | The stable machine-readable `code` member of an API Problem. | Message text, which may be localised. | [07](07-api-and-integration-contract.md) |
| **`escalated`** | Dispute state in which the matter passes to Network Administrators or an outside process. | `resolved_*`. | [15](15-lifecycle-state-machines.md) |
| **Event** (lifecycle) / **Event** (notification) | A named request to change state / a committed occurrence that may produce notifications, identified by an event key. | An Audit Event. | [15](15-lifecycle-state-machines.md) / [17](17-notifications-and-communications.md) |
| **`expired`** | Trade state reached when `expires_at` passes before acceptance. | `cancelled`. | [15](15-lifecycle-state-machines.md) |
| **Export type** | One of `member_self`, `network_admin`, `network_closure`, `network_archive`, or `audit_events`. | A file format. | [18](18-data-export-and-portability.md) |

### G–H

| Term | Definition | Not to be confused with | Owner |
|---|---|---|---|
| **Governance policy** | A rule decided by people and published in the Charter; software may record and remind but not interpret it. | Software enforcement. | [10](10-governance-and-network-operations.md) |
| **Guard** | A condition evaluated inside a transition's atomic unit; a failing guard rejects the event and changes nothing. | A UI validation. | [15](15-lifecycle-state-machines.md) |
| **History record** | Append-only per-object log of transitions (from, to, event, actor, time, reason). | An Audit Event. | [15](15-lifecycle-state-machines.md) |
| **Hosted / Self-hosted deployment** | An installation run by a Platform Operator for unrelated Networks (T3) / run by the Network's people or a host organisation (T1, T2). | A subscription product (out of MVP scope). | [11](11-deployment-operations-and-support.md) |

### I

| Term | Definition | Not to be confused with | Owner |
|---|---|---|---|
| **Idempotency key / scope** | Client-supplied key that makes retries safe / the tuple (`network_id`, Principal, key) within which it is unique. | A session or request ID. | [07](07-api-and-integration-contract.md) |
| **Inactive** (flag) | Derived flag when a Membership shows no activity for `inactivity_threshold_days`; no automatic effect. | A lifecycle state or `suspended`. | [03](03-domain-model-and-mutual-credit-rules.md) |
| **`inactivity_threshold_days`** | Policy parameter: days without activity before the inactive flag is set. | A deadline for removal. | [16](16-network-policy-catalog.md) |
| **Invitation** | A single-use, expiring token letting a person join an `invite`-mode Network. | A Trade proposal. | [15](15-lifecycle-state-machines.md) |

### J–L

| Term | Definition | Not to be confused with | Owner |
|---|---|---|---|
| **Job runner** | Component that executes asynchronous work with retries: Outbox consumption, expiry sweeps, reconciliation, export builds. | A user-visible scheduler. | [05](05-architecture-and-trust-boundaries.md) |
| **Journal Entry** | Immutable ledger record: a set of Postings that sum to zero. | An Audit Event; a Trade (a Trade results in one Journal Entry when completed). | [03](03-domain-model-and-mutual-credit-rules.md) |
| **Ledger** | The Posting interface plus Journal Entries, Postings, balances and idempotency records for a Network. | A database table alone. | [05](05-architecture-and-trust-boundaries.md) |
| **Listing** (Offer / Request) | Informational advertisement of goods or services provided (Offer) or sought (Request); no ledger effect. | A Trade proposal. In UI, "request" is reserved for Listings. | [00](00-conventions.md) |
| **Login** | The email-based identity that can hold Memberships in several Networks (D-07). | A Membership. | [06](06-data-ledger-and-audit.md) |

### M

| Term | Definition | Not to be confused with | Owner |
|---|---|---|---|
| **Maintenance mode** | A deployment-wide or per-Network state in which reads work and writes are rejected with an explicit message. | A Network `paused` status. | [11](11-deployment-operations-and-support.md) |
| **Mandatory notice** | A notification whose email cannot be turned off by preferences. | A broadcast, which is never mandatory. | [17](17-notifications-and-communications.md) |
| **`max_balance`** | Optional policy parameter: positive cap on a Member Account balance (null = none). | `max_credit_limit`. | [16](16-network-policy-catalog.md) |
| **`max_credit_limit` / `max_trade_amount` / `min_trade_amount`** | Policy parameters: highest credit limit an admin may set / largest and smallest allowed Trade amount. | `default_credit_limit`. | [16](16-network-policy-catalog.md) |
| **Member** | A person (post-MVP, also an organisation) with a Membership in a Network. | A login or a Visitor. | [00](00-conventions.md) |
| **Member Account** | The ledger account owned by exactly one Membership, created with it. | The Member's profile. | [03](03-domain-model-and-mutual-credit-rules.md) |
| **Membership** | A person's relationship to one Network, with a lifecycle and roles. | A login, or an Account. | [03](03-domain-model-and-mutual-credit-rules.md) |
| **`membership_mode`** | Policy parameter: `invite`, `approval` or `open`. | The Membership state. | [16](16-network-policy-catalog.md) |

### N

| Term | Definition | Not to be confused with | Owner |
|---|---|---|---|
| **Network** | One independent mutual-credit community with its own unit, members, rules and ledger; an isolated tenant. | A group, tenant (architecture only), or the Platform. | [00](00-conventions.md) |
| **Network Administrator** (`network_admin`) | Member-role with rule-configuration and admin powers in one Network. | Platform Operator. | [00](00-conventions.md) |
| **Network admin console / Operator console** | The administrator's section of the web client / the Platform Operator's tooling. | Each other: separate identity spaces. | [05](05-architecture-and-trust-boundaries.md) |
| **Network context** | The Network identifier derived server-side from the authenticated session and Membership. | A client-supplied value. | [05](05-architecture-and-trust-boundaries.md) |
| **`network_clearing`** | System Account that balances Adjustments and openings. | A bank account. | [00](00-conventions.md) |
| **Network operating status** | Network-level status: `setup`, `open`, `paused`, `closing`, `closed`. | Membership, Trade or Operator hold. | [10](10-governance-and-network-operations.md) |
| **Notice period** | Interval between approval of a policy change and its effective time, at least `policy_notice_days`. | Notification latency. | [10](10-governance-and-network-operations.md) |
| **Notification** | One message to one recipient for one event, shown in-product and optionally by email. | An Audit Event. | [17](17-notifications-and-communications.md) |
| **Notification intent** | The Outbox record written in the same atomic step as the originating change. | The delivered message. | [17](17-notifications-and-communications.md) |

### O

| Term | Definition | Not to be confused with | Owner |
|---|---|---|---|
| **Offer / Request** | See *Listing*. | | [00](00-conventions.md) |
| **Operator hold** | A deployment-level restriction (`read_only` or `suspended`) placed on a Network by the Platform Operator. | `paused`, which a Network Administrator sets. | [10](10-governance-and-network-operations.md) |
| **Outbox** | Durable list of events written in the same atomic step as the state change and consumed asynchronously. | A user's outbox. | [05](05-architecture-and-trust-boundaries.md) |
| **Over limit / over cap** | A Member Account whose balance is below `−credit_limit` / above `max_balance` after a Reversal, Adjustment or limit change; it cannot be debited / credited by a Trade until back in range. | A violation by the Member. | [04](04-functional-requirements.md) |

### P

| Term | Definition | Not to be confused with | Owner |
|---|---|---|---|
| **Party** | A Member who is the Provider or Recipient of a Trade, or the Proposer or Counterparty of a proposal. | A Dispute party (raiser and respondent). | [02](02-personas-roles-and-journeys.md) |
| **Platform** | The Circula software plus the party operating a deployment. | A Network. | [00](00-conventions.md) |
| **Platform Operator** (`platform_operator`) | Staff of the deployment operator; has no ledger-posting powers and no default access to Network content. | Network Administrator. | [00](00-conventions.md) |
| **Policy** / **Policy version** | The Network's set of policy parameters / an immutable numbered snapshot of it with an effective time. | Charter text. | [16](16-network-policy-catalog.md) |
| **`policy_notice_days`** | Policy parameter: days between announcing and applying a policy change (default 14). | Safety fixes, which may be immediate. | [16](16-network-policy-catalog.md) |
| **Posting** | One signed line of a Journal Entry against one account. | A forum post. | [03](03-domain-model-and-mutual-credit-rules.md) |
| **Posting interface** | The single internal interface through which all Journal Entries are created. | A public API. | [05](05-architecture-and-trust-boundaries.md) |
| **Principal** | The authenticated caller: a logged-in person or, post-MVP, an integration token. | A Member (a person may have several). | [07](07-api-and-integration-contract.md) |
| **Problem** | An API error body per RFC 9457 with Circula extension members. | A support ticket. | [07](07-api-and-integration-contract.md) |
| **`proposal_expiry_days`** | Policy parameter: days a Trade proposal stays `proposed` before it expires. | `dispute_window_days`. | [16](16-network-policy-catalog.md) |
| **Proposal origin** (`initiated_by`) | Whether the Provider or Recipient created the proposal; the other party accepts. | Who is credited (always the Provider). | [00](00-conventions.md) |
| **Proposer** | Member who creates a Trade proposal. | Provider. | [00](00-conventions.md) |
| **Provider / Recipient** | In a Trade: the Member who supplies goods or services and is credited / who receives them and is debited. | Buyer and seller (not used). | [00](00-conventions.md) |

### Q–R

| Term | Definition | Not to be confused with | Owner |
|---|---|---|---|
| **Quorum action** | An action subject to `admin_quorum`, approved by distinct administrators. | A vote of Members. | [04](04-functional-requirements.md) |
| **Reason** | Mandatory non-empty free text plus, where defined, a reason code. | A rationale in a spec. | [04](04-functional-requirements.md) |
| **Reconciliation** | The job that recomputes balances from Postings and compares them with cached balances. | A bank reconciliation. | [11](11-deployment-operations-and-support.md) |
| **Reconciliation proof / Verifier** | Manifest blocks and algorithm allowing independent recomputation of an export / the program that checks it. | Proof of identity. | [18](18-data-export-and-portability.md) |
| **Report** | A Member's category-plus-text complaint about a Member, Listing or Trade to the administrators. | A Dispute. | [09](09-security-privacy-and-abuse.md) |
| **Review signal** | An advisory indicator to administrators that a pattern may merit human review; never an automatic sanction. | A rating or score. | [09](09-security-privacy-and-abuse.md) |
| **Review step** | The mandatory screen between entering a Trade and submitting it, or between viewing and accepting a proposal. | A reviewer's work on a Dispute. | [08](08-ux-and-accessibility.md) |
| **Reversal** | A new Journal Entry that negates a prior completed one; never an edit or delete. | An Adjustment or a "refund". | [03](03-domain-model-and-mutual-credit-rules.md) |
| **`reversed`** | Terminal Trade state reflecting a linked Reversal entry. | `cancelled`, which happens before posting. | [15](15-lifecycle-state-machines.md) |
| **RPO / RTO** | Maximum tolerable data loss / restore time, in time units. | SLA. | [11](11-deployment-operations-and-support.md) |

### S

| Term | Definition | Not to be confused with | Owner |
|---|---|---|---|
| **Safety fix** | A policy change that takes effect before the notice period because delay would increase harm; only for listed tightening changes. | Any urgent preference of an administrator. | [10](10-governance-and-network-operations.md) |
| **Security log** | Operational log of authentication and security events. | An Audit Event; the ledger. | [09](09-security-privacy-and-abuse.md) |
| **Sensitive action** | An action protected by `admin_quorum`, step-up authentication and an Audit Event. | Any admin action. | [09](09-security-privacy-and-abuse.md) |
| **Settlement window** | Period in `departing` during which a departing Member's non-zero balance is resolved per `departure_settlement_mode`. | A payment deadline. | [15](15-lifecycle-state-machines.md) |
| **Side effect** | Everything a successful transition does besides changing state: posting, Audit Event, history record, notification. | An error side effect. | [15](15-lifecycle-state-machines.md) |
| **SLI / SLO** | Service level indicator / objective. | A contractual SLA. | [11](11-deployment-operations-and-support.md) |
| **Source system** | A non-Circula system from which an import originates. | A Network. | [18](18-data-export-and-portability.md) |
| **Status message** | A message that reports an action's result without taking focus (WCAG 4.1.3). | A notification. | [08](08-ux-and-accessibility.md) |
| **Step-up** | Fresh re-authentication required before a sensitive operation. | Sign-in. | [09](09-security-privacy-and-abuse.md) |
| **`suspended`** | Membership state in which the Member is restricted but keeps read-only access to their own data; reversible. | `closed`, or a ban flag. | [15](15-lifecycle-state-machines.md) |
| **System Account** | Per-Network non-member account that balances Adjustments, openings and settlements (`network_clearing`, `community_fund`); exempt from limits. | A Member Account. | [00](00-conventions.md) |

### T–Z

| Term | Definition | Not to be confused with | Owner |
|---|---|---|---|
| **Template / Template key** | A versioned localisable message definition / its stable identifier. | A policy template or Charter template. | [17](17-notifications-and-communications.md) |
| **Tenant** | One Network, as the unit of isolation (architecture wording only). | UI term for a Network. | [05](05-architecture-and-trust-boundaries.md) |
| **Terminal state** | A state with no outgoing transitions. | Deleted. | [15](15-lifecycle-state-machines.md) |
| **Tombstone** | A fixed placeholder (for example `[removed]`) replacing a redacted field. | Deletion of the row. | [06](06-data-ledger-and-audit.md) |
| **Trade** | A proposed or completed exchange of value between exactly two Member Accounts (domain: `Transaction` of type `trade`). | A sale, payment or transfer of money. | [00](00-conventions.md) |
| **Transaction** | Business-level record with a `type`: `trade`, `reversal`, `adjustment`, `opening_balance`, `departure_settlement`. | A bank transaction. UI says "trade". | [03](03-domain-model-and-mutual-credit-rules.md) |
| **Transport** | The external message-delivery service behind the message-transport interface. | Network transport (TLS). | [17](17-notifications-and-communications.md) |
| **Trust boundary / Zone** | A line between regions of differing trust / a region (Z0 to Z5) with one trust level. | A network firewall only. | [05](05-architecture-and-trust-boundaries.md) |
| **`unit_name` / `unit_symbol` / `unit_scale`** | Policy parameters: name, symbol and decimal places (0 to 4, default 2) of the credit unit; immutable after the first Journal Entry. | Currency code. | [16](16-network-policy-catalog.md) |
| **Undeliverable address** | A sign-in email address that has permanently failed delivery. | A blocked or unverified address. | [17](17-notifications-and-communications.md) |
| **Visitor** | An unauthenticated person using public pages. | An Applicant. | [02](02-personas-roles-and-journeys.md) |
| **Web client** | The first-party UI for Visitors, Applicants, Members, Reviewers and Administrators. | The API. | [05](05-architecture-and-trust-boundaries.md) |
| **WCAG 2.2 AA** | The accessibility target (D-14). | AAA, used only as guidance. | [08](08-ux-and-accessibility.md) |

## 4. Banned and discouraged terms

This table is used by UX copy review, the catalogue lint ([08](08-ux-and-accessibility.md), UX-029, UX-109), notification templates ([17](17-notifications-and-communications.md)), marketing pages and developer documentation. It is **design guidance to keep the product's description accurate and consistent with the canon (INV-014)**; it asserts no legal conclusion. Whether any of this wording would matter legally in a jurisdiction is a question for counsel ([19](19-legal-and-regulatory-open-questions.md), notably LEG-Q-06 to LEG-Q-09, LEG-Q-18).

Status: **Banned** = must not appear in member-facing or marketing copy, except inside the single statement that credits are not money. **Discouraged** = avoid; use only with a recorded reason. **Restricted** = allowed in a defined narrow use.

| Term(s) | Status | Preferred wording | Reason (UX and legal-risk design) |
|---|---|---|---|
| currency, money, cash, dollars, euros (as the unit) | Banned | "credits", `{unit_name}`, "the network's unit" | The unit is an accounting unit, not money; the word invites the assumption it is redeemable (INV-014). Allowed once in the standing statement "{unit_name} are not money." |
| pay, payment, paid, pay back | Banned | "spend", "record a trade", "you received", "you provided" | Implies settlement in funds and may suggest a regulated payment activity (LEG-Q-06, LEG-Q-07). |
| price, cost, fee, charge | Banned (as applied to credits) | "amount", "the agreed amount" | Prices imply an exchange rate to money; no fees on Trades in MVP (LEG-G-04, LEG-Q-16). |
| wallet, funds, deposit, withdraw, withdrawal, top up, cash out, redeem, convert, exchange rate | Banned | "your balance", "your account", "you can spend up to" | Suggests stored value and redeemability (LEG-Q-09, INV-014). |
| coin, token, mint, stake, yield, airdrop, blockchain | Banned | "credit", "recorded in the ledger" | Crypto vocabulary signals a different product and regulatory category (LEG-Q-19). |
| interest, rate, APR, yield, return, appreciation, invest, investment | Banned | none; the Platform applies no interest (LEG-G-03) | Implies lending or securities (LEG-Q-18). |
| loan, lend, borrow, borrowing limit, overdraft, credit line | Banned | "credit limit" (canon term) | Lending vocabulary; the negative balance is an obligation to provide goods or services (canon 5.2). |
| debt, debtor, creditor, owe, owed, owing, arrears, overdue, default, collections | Banned | "your balance is {amount} below zero", "within your limit", "you can spend up to ..." | A negative balance is not a debt in national currency (canon 5.2); collection language invites enforcement expectations (LEG-Q-08, LEG-Q-23). Governance forbids Platform dunning (GOV-039). |
| balance owed, amount due | Banned | "balance", "amount" | As above. |
| refund, chargeback, reimbursement | Banned | "reversal" (only after an agreed or upheld outcome) | Implies the card-payment dispute model and a return of money (LEG-Q-22). |
| correction (alone) | Discouraged | "adjustment" or "reversal" | Hides which mechanism was used and who can appeal it. |
| transaction (member-facing) | Discouraged | "trade", "adjustment", "reversal" | Implies a bank transaction. Allowed in domain and API documentation. |
| buy, sell, bought, sold, income, revenue, profit | Discouraged | "received", "provided", "earned" / "spent" in the defined UX use | Implies a market sale and may suggest a tax-relevant event (LEG-Q-02, LEG-Q-03). |
| earn / earned, spend / spent | Restricted | Only as direction words with the unit shown and the unit explained on first use (UX-036) | Required for direction clarity (UX-OD-4). |
| group, organisation (for Network), owner, moderator (for Network Administrator) | Discouraged | "network", "network administrator" | UI term consistency (UX-109). |
| tenant, tenancy | Restricted | Architecture documents only | Not a member-facing concept. |
| complaint | Discouraged | "dispute" (about a Trade) or "report" (about conduct or a Listing) | Different processes with different outcomes. |
| ban, blacklist, whitelist | Discouraged | "suspension"; "allow list" / "block list" in technical text | Avoids public shaming and biased terminology. "Ban" remains a recorded flag, never a headline. |
| guarantee, guaranteed, insured, safe as money | Banned | State what is true: "balances are recorded in the ledger and checked daily" | Avoids implying protection schemes. |
| "Circula holds your credits" / "Circula pays" | Banned | "Your network records your balance" | Operator is not a custodian of value (INV-014, LEG-G-01). |
| admin override, god mode | Discouraged | "adjustment with reason" | Admin changes are visible Adjustments (INV-012). |

## 5. Assumptions and open decisions

- **ASM-G1.** The glossary lists terms defined in documents 00 to 11, 15, 17 to 19 as of 2026-10-21. Terms from [12](12-quality-strategy-and-test-plan.md), [13](13-assumptions-risks-decisions-roadmap.md) and [16](16-network-policy-catalog.md) are included where they appear in the canon (policy parameters) or in other specs; their own local terms must be added when those documents are baselined.
- **ASM-G2.** Canon observation: the canon lists `community_fund` only as a value of `departure_settlement_mode` in 5.6, while [03](03-domain-model-and-mutual-credit-rules.md) also defines a System Account of that name. This glossary treats them as the same object under two uses; the canon text in 5.1 names only `network_clearing`.
- **ASM-G3.** UX-029 lists "price" and "pay" as banned; the canon's proposal wording ("please accept a charge of N") in 5.3 is internal vocabulary and **MUST** be shown to users as "an amount of N" (this table). Recorded for the owner of [08](08-ux-and-accessibility.md).
- **DEC-G1 (proposed).** Whether the banned-term lint applies to developer documentation and code comments, or only to member-facing copy and marketing. Recommended default: member-facing copy, notifications and marketing are linted; developer docs follow the table as a guideline (the research note recommends avoiding crypto vocabulary in code comments as well).

## 6. References

- [00 — Conventions and shared canon](00-conventions.md), sections 5.1 to 5.6.
- Owning documents as linked in the Owner column.
- Research input on terminology guardrails: [regulatory and privacy questions](../notes/research-regulatory-privacy-questions.md); [prior art in mutual credit](../notes/research-prior-art-mutual-credit.md).
- RFC 9457, Problem Details for HTTP APIs: <https://www.rfc-editor.org/rfc/rfc9457> (not re-fetched on 2026-10-21; the network proxy blocked access).

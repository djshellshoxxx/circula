# Circula

**Community mutual credit, built for voluntary exchange and local autonomy.**

Circula is an open-source-capable platform for communities that want to coordinate the exchange of goods and services through mutual credit. Members earn credits by providing value and spend them with other members. Every completed trade creates equal-and-opposite accounting entries, so the network can keep a shared, auditable record without requiring every exchange to pass through a bank, payment platform, or centralized marketplace.

Circula is software for operating a network. A balance is an accounting position under that network’s rules; Circula does not guarantee monetary value, acceptance by other members, or redemption for national currency.

## Why Circula could matter

Most economic platforms make participation depend on access to money, centralized payment rails, or a platform’s permission. Mutual credit offers another coordination tool: a member can obtain value from one participant and later provide value to someone else in the network. The accounting unit tracks the exchange; the community decides who may participate, what limits apply, and how disputes are handled.

Circula’s disruptive potential comes from making this model practical, understandable, and replicable:

- **More ways to coordinate:** communities can organize useful exchange when cash is scarce or when conventional marketplaces do not serve them well.
- **Voluntary association:** each network can set its own participation and governance rules, with transparent records for members.
- **User and community control:** an open, portable design can let groups choose how they operate and where their instance runs.
- **Reduced platform dependence:** members can discover one another and transact within a community without making a large marketplace the sole intermediary.
- **Replicable infrastructure:** shared software and open protocols could let many independent networks form, learn, and interoperate by choice.

This is an experiment in economic coordination, not a promise to replace existing money or institutions. Participation is voluntary, and each network’s accounting unit is not a claim on cash.

## Project principles

1. **Voluntary exchange:** members choose whether to join, offer, request, and accept trades.
2. **Community governance:** networks define membership, limits, and dispute processes; the software records and enforces the selected rules.
3. **Accounting integrity:** completed trades are balanced, atomic, and reconstructable from a traceable ledger.
4. **Member agency:** balance direction, credit limits, pending activity, and consequences are clear before acceptance.
5. **Portability and self-determination:** exports and self-hosting are architectural goals; exact release support remains an open decision.
6. **Privacy by design:** collect the minimum information needed and make visibility and export permissions explicit.
7. **Interoperability by consent:** future network connections should be opt-in, transparent, and governed by participating networks.

## The crypto and crypto-libertarian angle

“Crypto” can mean cryptography and distributed protocols, or it can mean a tradable cryptocurrency. Circula’s mutual-credit model does not require a coin, blockchain, mining, or token speculation. Its crypto-libertarian potential is the ability for people to form voluntary networks, set local rules, operate their own infrastructure, connect by consent, and leave with portable records. This reduces dependence on a single issuer or platform; it does not make a network independent of governance, infrastructure, or law.

Cryptographic tools could support that direction: signed transaction approvals, verifiable append-only audit records, encrypted backups and dispute evidence, member-controlled credentials, and multi-administrator recovery. These can help detect tampering and limit unilateral control, but they do not make false claims true, prevent coercion, guarantee privacy, or replace a fair dispute process. A signature also proves control of a key, not necessarily a person’s real-world identity or informed consent.

A conventional database with atomic balanced postings is likely the simplest initial ledger. Cryptographic signatures or tamper-evident event proofs could be layered on when a clear member need justifies them. A public blockchain is an optional future experiment, not a foundation: public ledgers can expose durable transaction metadata, fees and irreversible errors can harm users, and token framing can invite speculation that conflicts with mutual-credit accounting. See [Crypto and crypto-libertarian design options](docs/brainstorming/crypto-and-crypto-libertarian-design.md) for the distinctions, candidate features, and tradeoffs.

## Initial MVP direction

The draft MVP focuses on network setup, membership, an offers and requests directory, transaction proposals and acceptance, balance and history views, administration, disputes, audit history, and basic data exports. The [specifications index](docs/specs/README.md) links to the detailed requirements and unresolved decisions.

Ideas such as cross-network exchange, community reserves, member-controlled credentials, and participatory governance experiments are intentionally kept in the [future brainstorming list](docs/brainstorming/disruptive-future-ideas.md) until the project owner and prospective networks choose what belongs in a later roadmap.

## Sustainable business and profit options

Circula can pursue commercial success by selling useful operations and expertise around open, community-controlled software. The strongest fit to explore is a free, inspectable core with paid convenience, reliability, and support. That can create a viable business while allowing communities to self-host, export their records, and avoid paying a toll on every unit of mutual credit.

Potential revenue paths include hosted network subscriptions, managed self-hosted deployments, backup and recovery services, organization-level administration features, migration and onboarding, training, integrations, and paid support. Federation or clearing services could be considered later if participating networks define transparent limits and governance. The product should earn from operating the software and helping networks succeed, not from controlling or speculating on their balances.

The business model is undecided. Pricing, software licensing, company structure, and any paid network services need separate evaluation. The detailed options, tradeoffs, and staged approach are in [Business model options](docs/brainstorming/business-model-options.md).

## What Circula does not claim

- Credits are not presumed to be cryptocurrency, a security, or a national-currency equivalent.
- Circula does not guarantee that a member will accept a credit or deliver an agreed good or service.
- A network balance is not automatically redeemable for cash.
- The software does not decide a community’s values or replace the need for clear governance.
- No legal, tax, privacy, or financial-regulatory status is asserted by these draft documents.

Any future feature involving conversion, redemption, payment processing, or other legally significant behavior requires separate product definition and qualified jurisdiction-specific review.

## What is in the repository

The project is currently in its documentation and design stage. The repository contains draft foundational specifications and future concepts for community review. It does not yet contain a working application.

- [Foundational specifications](docs/specs/README.md): purpose, scope, roles, domain rules, MVP requirements, architecture, data, API, UX, security, governance, operations, and verification.
- [Future disruptive ideas](docs/brainstorming/disruptive-future-ideas.md): possible extensions for increasing Circula’s reach and social impact. These are concepts for discussion, not approved requirements.
- [Business model options](docs/brainstorming/business-model-options.md): possible ways to make the software and related services commercially sustainable.
- [Wiki framework](wiki/README.md): source pages for the future GitHub Wiki, clearly marked as draft until features are implemented.
- [Specification authoring prompt](prompts/create-foundational-specs.md): original scope and completion checklist.

## Contributing and next steps

The first step is to review the [foundational specifications](docs/specs/README.md), especially the open decisions on hosting, pending credit reservations, membership, privacy retention, accessibility, and service targets. The brainstorming lists are places to propose ideas and record benefits, risks, and questions before turning concepts into requirements.

As implementation begins, contribution guidance, supported deployment targets, and release criteria should be added here and linked from the specs index.

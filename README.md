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

## What is in the repository

The project is currently in its documentation and design stage. The repository contains draft foundational specifications and a list of future concepts for community review. It does not yet contain a working application.

- [Foundational specifications](docs/specs/README.md): purpose, scope, roles, domain rules, MVP requirements, architecture, data, API, UX, security, governance, operations, and verification.
- [Future brainstorming ideas](docs/brainstorming/disruptive-future-ideas.md): possible extensions for increasing Circula’s reach and social impact. These are concepts for discussion, not approved requirements.
- [Specification authoring prompt](prompts/create-foundational-specs.md): original scope and completion checklist.

## Initial MVP direction

The draft MVP focuses on network setup, membership, an offers and requests directory, transaction proposals and acceptance, balance and history views, administration, disputes, audit history, and basic data exports. The [specifications index](docs/specs/README.md) links to the detailed requirements and unresolved decisions.

Ideas such as cross-network exchange, community reserves, member-controlled credentials, and participatory governance experiments are intentionally kept in the brainstorming list until the project owner and prospective networks choose what belongs in a later roadmap.

## What Circula does not claim

- Credits are not presumed to be cryptocurrency, a security, or a national-currency equivalent.
- Circula does not guarantee that a member will accept a credit or deliver an agreed good or service.
- A network balance is not automatically redeemable for cash.
- The software does not decide a community’s values or replace the need for clear governance.
- No legal, tax, privacy, or financial-regulatory status is asserted by these draft documents.

Any future feature involving conversion, redemption, payment processing, or other legally significant behavior requires separate product definition and qualified jurisdiction-specific review.

## Contributing and next steps

The first step is to review the [foundational specifications](docs/specs/README.md), especially the open decisions on hosting, pending credit reservations, membership, privacy retention, accessibility, and service targets. The brainstorming list is a place to propose ideas and record benefits, risks, and questions before turning any concept into a requirement.

As implementation begins, contribution guidance, supported deployment targets, and release criteria should be added here and linked from the specs index.

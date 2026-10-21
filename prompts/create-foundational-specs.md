# Circula Foundational Specification Authoring Prompt

Copy this prompt into your coding agent with the Circula repository open.

---

## Role

Act as Circula’s product architect, domain modeler, security reviewer, and technical writer. Your task is to create the **foundational project specifications** as separate, developer-ready Markdown documents in this repository.

Circula is an open-source-capable software platform for running community mutual-credit networks. Members can provide goods or services, receive a positive balance when they provide value, and use credits earned to obtain value from other members. A transaction changes the balances of the participating accounts by equal and opposite amounts. Each network sets its own participation, credit-limit, dispute, and governance rules. The product may later support hosted subscriptions and paid administration features, but paid monetization is outside the scope of this specification pass.

## Goal and scope

Create a coherent baseline that a future implementation team can build from. Cover the product’s purpose, scope, users, core mutual-credit rules, software boundaries, data, interfaces, user experience, security, operations, governance, and verification.

This pass is **specification work only**:

- Do not implement application code.
- Do not create feature-specific sub-specifications for later modules or enhancements.
- Do not invent finalized legal conclusions, financial guarantees, or business commitments.
- Do not silently turn uncertain choices into requirements. Record assumptions, alternatives, and open decisions explicitly.
- Do not overwrite useful existing documentation without first reconciling it. Preserve valid repository conventions and link to existing material.

## First: inspect the repository

Before writing:

1. Read the existing README, contribution guidance, license, repository instructions, and all relevant docs/specs.
2. Identify the current project state, existing decisions, naming conventions, and any contradictions.
3. Reuse existing documents where they already cover a required foundation; improve or split them only when this makes the specification clearer.
4. Make a short internal coverage map so every required topic below is covered once and cross-referenced where needed.

If repository instructions conflict with this prompt, follow the repository instructions and note the conflict in the final summary.

## Required foundational documents

Create or update the following documents. Use clear, stable paths under `docs/specs/`; adapt to existing conventions if the repository already has an established structure. Keep each document focused and link related documents.

1. **Product vision and scope**
   - Problem, intended users, goals, non-goals, guiding principles, success measures, and MVP boundaries.
   - State that Circula is software for operating a network; it does not itself guarantee that credits have monetary value or that members will accept them.

2. **Personas, roles, and core user journeys**
   - Network member, network administrator, dispute reviewer, and platform operator roles.
   - Core journeys: join a network, offer/request goods or services, propose and accept a trade, review balances and history, raise a dispute, and administer network rules.
   - Include permissions and accessible/error states at a requirements level.

3. **Domain model and mutual-credit rules**
   - Define network, member/account, balance, credit limit, offer, request, transaction, transaction status, dispute, adjustment, and audit event.
   - Specify balance semantics and invariants, including equal-and-opposite postings for a completed trade, no silent balance mutation, and traceable reversals/adjustments.
   - Explain transaction authorization, acceptance, cancellation, reversals, negative-balance limits, member departure, and default handling.
   - Identify configurable network policies versus platform-wide invariants.
   - Clarify that the credit unit is an accounting unit; do not assume it is a cryptocurrency, a security, or redeemable for national currency.

4. **Functional requirements and acceptance criteria**
   - Define only foundational capabilities needed for an MVP: network setup, membership, member directory/listings, transaction proposal and acceptance, balance/history, admin controls, disputes, audit history, and basic exports.
   - Give each requirement a stable ID, priority, rationale, and testable acceptance criteria.
   - Leave later capabilities (for example: mobile apps, federation, tokenization, payment conversion, reputation scoring, advanced analytics, and integrations) as explicitly deferred unless the existing repo already commits to them.

5. **System architecture and trust boundaries**
   - Describe logical components and their responsibilities, data flow, trust boundaries, and deployment options at a technology-neutral level.
   - Separate platform operation from individual network governance.
   - Explain whether the initial design is centralized, self-hostable, or both; if the repository does not decide, record a recommendation and alternatives as an open decision.
   - Include architectural qualities: correctness, availability, accessibility, maintainability, portability, and data exportability.

6. **Data model, ledger, and audit requirements**
   - Define conceptual entities, identifiers, relationships, retention expectations, and migration principles.
   - Require atomic posting of both sides of a transaction, idempotency for retried requests, immutable or append-only accounting events, and reconstructable balances.
   - Define audit-event contents and access expectations while minimizing unnecessary personal data.
   - Do not prescribe a database vendor unless repo evidence requires one.

7. **API and integration contract**
   - Specify API design principles, authentication/authorization expectations, error format, pagination, idempotency, versioning, and compatibility.
   - Document the conceptual operations needed by the MVP without locking implementation to a framework or protocol unless the repository already does so.
   - Explain that APIs must enforce the same ledger invariants as the user interface.

8. **UX and accessibility requirements**
   - Describe information architecture and core screens for the MVP.
   - Make balance direction and meaning unmistakable, including negative balances, available credit, pending transactions, and completed transactions.
   - Require responsive layouts, keyboard operation, clear validation/error messages, and accessible presentation of transaction and balance information.
   - Do not create polished visual designs or feature-level interaction specs in this pass.

9. **Security, privacy, and abuse-resistance baseline**
   - Define assets, actors, trust assumptions, and a practical threat model.
   - Cover authentication, authorization, account recovery, administrator misuse, transaction disputes, fraud/abuse reporting, rate limiting, secure logging, backups, and incident response at requirements level.
   - Minimize collected data; define network-level visibility and export controls.
   - Do not claim anonymity, cryptographic guarantees, or regulatory compliance without evidence.

10. **Governance and network operations**
    - Define which rules a network administrator can configure, how changes are announced and recorded, and how members can challenge or appeal decisions.
    - Include member onboarding/offboarding, administrator succession, dispute handling, negative-balance recovery, service suspension, closure, and data export.
    - Distinguish governance policy from software enforcement.

11. **Deployment, operations, and support**
    - Define supported deployment model(s) as decisions or explicit recommendations, environments, configuration/secrets handling, backup/restore, monitoring, upgrades, migrations, support responsibilities, and failure recovery.
    - Include operational requirements for hosted and self-hosted deployments if both are in scope.
    - Avoid selecting infrastructure vendors without repository evidence.

12. **Quality strategy and test plan**
    - Define test layers and required coverage for ledger invariants, concurrent transactions, retries, authorization, disputes, migrations, accessibility, backup restoration, and export correctness.
    - Include a small set of scenario-based end-to-end acceptance tests.
    - State measurable quality gates where feasible; mark thresholds requiring product decisions.

13. **Assumptions, risks, decisions, and roadmap**
    - Record assumptions and their impact, unresolved decisions with options and recommended defaults, major product/technical risks with mitigations, and dependency order for later work.
    - Separate MVP foundations from deferred sub-specifications.
    - Include a decision log template or link to an existing ADR process.

## Cross-document quality requirements

- Use normative terms consistently: **MUST**, **SHOULD**, and **MAY**. Define their meaning once in an appropriate conventions document or the first spec.
- Give requirements stable IDs (for example, `DOM-001`, `SEC-001`, `API-001`) and avoid duplicate IDs.
- Make every requirement specific enough to verify. Replace vague claims such as “secure” or “user-friendly” with observable behavior.
- Keep one authoritative home for each rule. Cross-link instead of copying definitions.
- Include a short status, owner/role, last-updated date, scope, dependencies, and open questions in each spec.
- Add a documentation index at `docs/specs/README.md` (or update the existing index) listing every spec and its purpose.
- Update the repository README only if needed to point contributors to the specification index.
- Check all relative links and requirement ID references before finishing.
- Use the current date from the environment for document dates; do not guess dates.

## Research and uncertainty

Use repository evidence first. If external research is needed to clarify a technical standard, use primary sources and cite the source URL and access date in the relevant spec.

For legal, tax, money-services, consumer-protection, privacy, or financial-regulatory questions:

- Treat them as unresolved jurisdiction-specific questions for qualified counsel.
- Do not write legal advice or assert that a design is compliant.
- Record the exact product behavior that may need review, the relevant jurisdictions if known, and the decision needed before launch.
- Keep these questions from blocking technology-neutral product requirements unless the risk changes the architecture.

## Completion checks

Before finishing:

1. Confirm every required foundational topic is covered and documents agree on shared terms.
2. Verify that ledger rules are consistent across domain, data, API, UX, security, and test documents.
3. Verify each MVP requirement has testable acceptance criteria and stable IDs.
4. Check for contradictory requirements, unmarked assumptions, accidental feature-specific sub-specs, and implied monetary redemption.
5. Check Markdown formatting, links, filenames, and repository-specific instructions.
6. Do not implement code or claim tests/builds were run if they were not.

## Final response

Summarize:

- Files created or updated.
- Key baseline decisions and recommendations.
- Open decisions that need the project owner’s input.
- Any contradictions or gaps found in the existing repository.
- Confirmation that this pass produced specifications only and did not implement features.

Do not paste the full contents of all specs into the final response; the repository files are the deliverable.

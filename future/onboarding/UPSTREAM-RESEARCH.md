# FUX-106 upstream research and adaptation handoff

Research phase: 2026-11-16. **Research complete; implementation and independent audit pending. HOLD integration.** No upstream application code is added by this document. Read alongside SPEC-SNAPSHOT.md and README.md. The owner now requests research and isolated adaptation, without authorizing main integration.

## Decision

Use **GOV.UK Frontend's task-list and warning-text components** as the concrete upstream source for a no-JavaScript, resumable checklist and unmistakable practice banner. Translate their small Nunjucks markup into Go `html/template`, adapting only the necessary styles and preserving their semantic relationships. Keep persistent progress, synthetic trade simulation, authorization and practice isolation as original Circula domain code. No searched upstream supplies a compatible mutual-credit practice engine, and copying a tour library cannot supply those invariants.

Avoid a JavaScript tour engine as the primary implementation. Circula's requirement is a skippable, server-rendered learning flow, not a mandatory spotlight overlay. A pinned source snapshot is preferable to importing an entire npm runtime or changing the root build.

## Verified candidates

The three repositories below were cloned and their checked-out source/license files read. SHA pins are actual inspected commits, not guessed release-to-commit mappings. Package versions are the values at those commits; a package version does not assert that HEAD is the release tag.

| Candidate | Exact inspected pin / package version | License evidence | Maturity and fit | Decision |
|---|---|---|---|---|
| [alphagov/govuk-frontend](https://github.com/alphagov/govuk-frontend) | `a11bd3316363ab4702fcef81d6ce06a947c464ef`; package `6.6.0`; commit 2026-11-12 | [LICENSE.txt](https://github.com/alphagov/govuk-frontend/blob/a11bd3316363ab4702fcef81d6ce06a947c464ef/LICENSE.txt), MIT, Crown Copyright 2017 | Established government design system; maintained component source, JSDOM tests, Puppeteer accessibility tests, published research. Task list introduced in v5.0.0 (2023), fixes documented in v5.5.0. Supports plain HTML without Nunjucks runtime. | **Selected for adapted markup/styles and test cases.** |
| [nilbuild/driver.js](https://github.com/nilbuild/driver.js) (older `kamranahmedse/driver.js` URL redirects) | `7c442ae7e2722585de297ffe97b6e3e99e951b45`; `packages/driver/package.json` says `1.14.0`; commit 2026-11-14 | [license](https://github.com/nilbuild/driver.js/blob/7c442ae7e2722585de297ffe97b6e3e99e951b45/license), MIT, Kamran Ahmed | Maintained TypeScript tour engine with unit and Playwright suites. Newer checklist/headless APIs are concrete reusable features, but DOM/JS-centric and do not provide secure persistence. Mature parent library does not make every newer subfeature independently mature. | Keep as an optional later enhancement candidate; **do not import now**. |
| [alphagov/govuk-prototype-kit](https://github.com/alphagov/govuk-prototype-kit) | `a5e0766d38846a534a9203c7b7437fa6264bf7d1`; package `14.0.2`; commit 2026-11-02 | [LICENCE.txt](https://github.com/alphagov/govuk-prototype-kit/blob/a5e0766d38846a534a9203c7b7437fa6264bf7d1/LICENCE.txt), MIT, Crown Copyright 2014 | Long-lived interactive prototype system; maintained by Design System team with minimal support. README explicitly says prototypes do not have production security provisions. Node/Express session/form-data mechanics do not match Circula. | **Reject production code migration.** Use its isolated synthetic-data concept only; never port blanket POST/query-to-session storage. |

Other search exclusion: [Intro.js](https://github.com/usablica/intro.js) advertises AGPL/commercial licensing; it is not selected for copying. [GOV.UK prototype task-list plugin](https://github.com/alphagov/govuk-prototype-kit-task-list) is archived/unmaintained and points users to the modern Frontend component; do not use it as the source.

## Exact selected reuse surface

All paths below are under `alphagov/govuk-frontend` at `a11bd3316363ab4702fcef81d6ce06a947c464ef`:

- `packages/govuk-frontend/src/govuk/components/task-list/template.njk`: port the unordered list, link-to-hint/status `aria-describedby`, unique per-row IDs, name/hint/status structure, and plain-text completion status into `templates/checklist.html`. Remove arbitrary attributes, HTML-valued parameters, and Nunjucks `safe` handling. Admin text is always escaped Go template data.
- `packages/govuk-frontend/src/govuk/components/task-list/_mixin.scss`: adapt list layout, visible focus/hover and clear status presentation into locally namespaced CSS. The upstream SCSS calls GOV.UK base mixins and uses tag styles; **it is not standalone CSS**. Either compile its verified dependency closure or deliberately port a minimal derivative with a modification record. Do not blindly copy `_index.scss` and claim it works.
- `packages/govuk-frontend/src/govuk/components/warning-text/template.njk`: adapt visible strong text plus an assistive-technology fallback for an always-present “Practice — synthetic credits, no real trade” marker. The icon is decorative; do not let it be the sole distinction.
- `packages/govuk-frontend/src/govuk/components/task-list/template.jsdom.test.js` and `accessibility.puppeteer.test.mjs`: port relevant assertions into Circula's Go HTML tests and a local browser fixture suite: task list semantics, unique IDs, accessible link names/status association, absent optional hint nodes, wrapped text, and keyboard access.
- `packages/govuk-frontend/src/govuk/components/warning-text/template.jsdom.test.js`: source for banner markup assertions.

Source example: [task-list template at the pin](https://github.com/alphagov/govuk-frontend/blob/a11bd3316363ab4702fcef81d6ce06a947c464ef/packages/govuk-frontend/src/govuk/components/task-list/template.njk).

Do not copy GOV.UK logos, royal arms, fonts or branding assets. Retain a full upstream MIT notice in `third_party/govuk-frontend/LICENSE.txt`; add a provenance manifest with upstream URL, SHA, original path, destination, local checksum and modification explanation for every adapted file. This preserves upstream licensing without choosing a global license for Circula. Check any imported dependency's own license separately. This research imports no upstream source and makes no vulnerability-free claim.

## Official guidance used

- [Task list component](https://design-system.service.gov.uk/components/task-list/): tasks should support flexible completion order and multi-session use; use ordinary HTML. Circula must explicitly deviate from the default service-completion pattern by allowing **Skip** and never gating core trade access on tutorial completion.
- [Complete multiple tasks](https://design-system.service.gov.uk/patterns/complete-multiple-tasks/): keep descriptions concise and progress understandable.
- [Accessibility](https://design-system.service.gov.uk/accessibility/): adopting the design system does not automatically make the resulting service accessible.
- [Modifying components](https://design-system.service.gov.uk/get-started/extending-and-modifying-components/): re-test changes; namespace local modifications and maintain an update boundary.
- [Prototype Kit README at pin](https://github.com/alphagov/govuk-prototype-kit/blob/a5e0766d38846a534a9203c7b7437fa6264bf7d1/README.md): prototype data and security are not production substitutes.

## Existing Circula seams inspected

Read-only inspection confirms the core has `internal/web/templates`, `internal/web/web.go` constructors/routes, `internal/service/join.go` (`JoinNetwork`, `JoinNetworkTerms`), `internal/service/gov_terms.go` (`AcceptTerms`), membership/directory services, listing services and proposal/acceptance services. The future module currently has only `Progress.Complete`, `Progress.Next` and checked synthetic `PreviewTrade` arithmetic. It has no persistent state, browser flow, demo runtime, or authorization adapter.

The actual joining, terms acceptance, listings and trade completion must remain authoritative core operations. Tutorial checkboxes must never serve as evidence that the user accepted current terms or completed a real trade. The later core adapter may derive completion from authorized facts; the module may also track non-authoritative learning/skip state.

## Full-spec adaptation sequence for the next agent

1. **Domain contract and progress persistence.** Make progress keyed by network + member + curriculum version. Validate loaded states as well as writes (`Next` currently does not reject an invalid Version/mask). Represent skipped versus completed explicitly; never overwrite a newer curriculum with a stale request. Define a small repository transaction interface, a standalone SQLite adapter or `database/sql` adapter compatible with the core's existing driver, and an in-memory test double. Atomic update/CAS prevents two tabs losing completed steps. Provide additive schema SQL inside the module only; no root migration registration.
2. **Accessible renderer from selected upstream.** Build an embedded Go template, locally bundled CSS and render/view-model API. Links/routes come from trusted configuration; admin copy is bounded plain text. Validate URL schemes/origins/path prefixes rather than accepting arbitrary `template.URL`. Use normal forms/links for Continue, Skip and Resume. No modal overlay, forced tour, browser local storage, external assets, telemetry or mandatory JavaScript. Never let progress gate normal Circula functionality.
3. **Practice isolation boundary.** Create a standalone constructor that only opens an isolated temporary or in-memory store and seeds fixed synthetic members/listings. It must not accept a production `*sql.DB`, production session token, core service or arbitrary DSN. Make practice IDs/types visibly separate and nonconvertible to real transaction inputs. Configure a distinct origin/session cookie name and no credential sharing at its eventual server boundary. Do not claim that a network ID in the same live database provides equivalent isolation.
4. **Scripted learning flow.** Present join/terms explanation, credit signs/limits/reservations, synthetic listing, partner, and proposal/acceptance in no more than five guided actions to the listing/proposal milestone. A proposal is pending until explicit synthetic acceptance; use `PreviewTrade` to show before/after balances. Carry clear practice markers through all pages, exports and errors. Allow reset and exit without transferring practice state. Keep arithmetic limits/overflow handling and add state-machine/idempotency tests.
5. **Admin copy/link configuration.** Permit bounded per-network instructional text and trusted governance/terms links via a distinct configuration adapter. Do not permit custom script, HTML, arbitrary remote embed, policy override, or tutorial content to modify credit rules. Bind configuration to the network and curriculum version.
6. **Metrics default off.** No event sender by default. If opt-in aggregate completion counting is included, design a narrow anonymous counter interface and ensure events do not contain member identifiers, balances, learning errors or creditworthiness inferences. Resolve minimum cohort/privacy policy before offering metrics externally; leaving optional metrics unimplemented is a recorded scope gap, not completion.
7. **Adapters prepared, activation deferred.** Supply an optional no-JS HTTP adapter with explicit caller-provided current-auth middleware, POST CSRF checks, no-store headers, bounded bodies and tenant scoping, plus command/test harness. Keep it entirely under `future/onboarding`; no runtime registration or root dependency changes. Production GET/PATCH contracts, CLI wiring, main templates and service wiring are later integration artifacts, not silently active code.
8. **Provenance and self-review.** Record exactly what was adapted and test evidence; update README/feature manifest honestly. The next independent reviewer must inspect the copied code, isolation boundary and adapters. No global application or feature-complete label until all applicable acceptance criteria have evidence.

## Required test matrix and completion gates

| Boundary | Automated evidence to add | Human/platform gate |
|---|---|---|
| Persistence | reopen/restart; concurrent updates; version/mask validation; network/member isolation; account switch; skip/resume; additive schema on historical data | Core backup/restore integration qualification |
| Rendering | XSS payloads in admin/member strings; link scheme/open-redirect rejection; unique IDs; semantic labels; normal-form no-JS flow; no external resources | Keyboard + screen reader, 320px/400% zoom, contrast and translation review |
| Practice | synthetic state machine; duplicate acceptance; credit/reservation boundary; overflow fuzzing; constructor cannot accept production handles; reset and session isolation | Separate origin deployment/cookie validation; verify the production ledger and audit remain byte-identical across full practice flow |
| Authorization | missing/currently suspended actor; wrong network/member; stale terms facts; CSRF failure; overlarge body; no-store responses | Review the actual core adapter before any integration |
| Metrics | default-off produces zero events; opt-in only; payload contains aggregate counts only | Privacy/cohort policy approval if shipped |
| Learning outcome | fixtures distinguish negative balance, reserved credit, pending proposal and completed transfer | User testing confirms comprehension and guided action count |

Run standalone `go test -race ./...`, `go vet ./...`, parser/state-machine fuzzing and browser fixture tests; run full core regression only when an adapter is integrated. A template rendering correctly does not prove accessibility, a memory store does not prove persistence, and tests against a fake auth provider do not qualify the real service adapter.

## Handoff inventory

Local inspected clones (scratch, not deliverables): `/tmp/onboarding-upstream/govuk`, `/tmp/onboarding-upstream/driver`, `/tmp/onboarding-upstream/prototype`. Re-fetch by the recorded commit if missing. No implementation or third-party code was committed in this research phase. All feature and core integration holds remain in effect.

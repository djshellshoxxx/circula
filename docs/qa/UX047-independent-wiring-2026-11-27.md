# UX-047 independent wiring investigation — 2026-11-27 Pacific

This is a bounded investigation, not a feature-completion or runtime-qualification record.

## Source and ownership

Fresh Git fetch and connected-provider reads agreed on main
`3faeed76c41467a75b09bc75e7185a98a92f8219`. The isolated detached worktree was
clean at that revision. No product files were changed. No future module was
inspected, changed or executed.

The final live read still showed #142 open/draft at
`78e254f4f1b8283c124274e0a6a1ba2b82959a0f` and #143 open/draft at
`bfbfb79020942394457966c1e72b329e65bb06dd`, both based on that main. Neither
separate owner merge authorization was present. Their product paths, shared
catalogue/test paths and source handoff files were left untouched. No CI rerun,
ready-for-review action or merge was requested.

Completed qualification was reused only as the published historical evidence:

- [#142 provider record](https://github.com/djshellshoxxx/circula/blob/fdb476e0d9c25b47110b37b51dfff88691f991fa/docs/qa/UX047-platform-notices-provider-2026-11-27.md).
- [#143 provider record](https://github.com/djshellshoxxx/circula/blob/6c2b653d1e6a43a332cc21142ea61723785e97f0/docs/qa/UX047-session-warning-provider-2026-11-27.md).

Those records are source qualification, not actual-main Windows/macOS evidence
for either draft. Required integration checks remain after any later authorized
merge.

## Result

Two native investigators found no confirmed eligible wrong-number, wrong-plural
or raw-label wiring defect in this scan outside the owned paths. The independent
investigator's scope and eliminated candidates are preserved in the companion
[investigation notes](UX047-independent-wiring-notes-2026-11-27.md).

The scan covered all 37 current one/other message entries and their production
callers, raw-looking dispute/transaction/member/break-glass status fields, integer
formatting callers and direct date/time presentation helpers. Existing count-first
callers, display-label mappings and exact fixed-point/identifier formatting were
consistent. No failing regression was fabricated, and no speculative product
repair was made.

This does not establish whole-UI localization or UX-024/UX-047 acceptance.

## Concrete residual assessment boundaries

The current pending-adjustment requested-by cell still directly composes a name,
comma and timestamp in
`internal/web/templates/admin_adjust_pending.html`:

```gotemplate
{{.Proposer}}, {{$.Time .Created}}
```

This is a candidate for a complete catalogue-owned presentation pattern, not a
confirmed current-English output error. There is no matching existing complete
catalogue entry. Implementing a new entry in the shared `en.json` would overlap
#142's explicitly excluded owned path. No new catalogue loader/file architecture
was introduced merely to avoid that boundary. Any future repair must retain the
escaped arbitrary proposer name and semantic time element; arbitrary names must
not enter `messages.Around`'s trusted-label arguments. A reordered synthetic
catalogue test should prove the selected structure before implementation.

Other separately bounded candidates remain unaudited for full acceptance:
service-supplied validation details/blocker lists and catalogue ownership of role
list punctuation. Their presence is not a claim that their current behavior is
incorrect or that every string needs a new translation. API/audit/protocol values,
unknown legacy fallbacks and user-authored text must remain literal data.

The broader nil-viewer selected-zone assessment remains distinct from #143's
qualified explicit-UTC fallback. It was not silently changed in this run.

## Execution and continuation

During the bounded investigation, only native Git/filesystem checks, connected
read-only provider requests and source inspections ran. Publishing these two
documentation files uses separate provider writes to an isolated audit branch.
No Go test, Node test, HTTP test, browser test, race suite
or new CI run was executed. The earlier SQLite compiler-killed attempts were not
repeated and are not passing test evidence. The official SDK's availability is
not a test result. GitHub CLI authentication was unavailable; connected provider
tools were available.

Next work must not repeat this same plural/status scan or #142/#143 qualification.
Fresh-check refs and owner messages, then assess one of the concrete residual
presentation boundaries with a failing functional regression before editing.
Coordinate shared-catalogue ownership before changes to that excluded path. If
an independently scoped defect outside the owned paths is found, implement only
that coherent repair. This investigation alone does not justify pausing all Core
engineering or declaring that only external gates remain.

UX-047 remains partial. Preserve locale/product scope, six explicit test skips,
manual screen-reader/external security review, bounded fuzz/load/reproducibility
and packaging limits. #83/#85/#86/#94/#107 and SEC-026/GOV-021 policies remain
held. All fifteen future modules remain out of scope. No licence, deployment,
release, force-push, history rewrite or branch deletion is authorized.

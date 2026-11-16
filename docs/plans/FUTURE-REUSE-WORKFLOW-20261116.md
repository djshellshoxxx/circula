# Future-feature reuse workflow — 2026-11-16

Owner authorized a team to search mature GitHub projects, reuse suitable code, adapt each of the 15 independent modules and continue development. Every phase uses a fresh agent for each feature. Integration remains on HOLD until each feature and Circula core are qualified and the owner selects one feature at a time.

## Phase separation

1. Research agent: inspect the feature specification and current core seams, compare upstreams, verify exact commits/releases and per-file licenses, document actual maturity and deficiencies. Commit UPSTREAM-RESEARCH.md.
2. Fresh implementation agent: consume that handoff, reuse compatible pinned components, preserve notices, implement and test the complete isolated feature scope. Track any unimplemented acceptance criteria explicitly. Commit implementation and evidence.
3. Fresh audit agent: inspect implementation against every requirement, reproduce tests, test adversarial inputs, concurrency and restart where applicable, fix findings and rerun relevant tests. Commit AUDIT.md and accurate status.

A phase can uncover additional work. It must not mark a feature complete while unresolved required behavior or acceptance tests remain. Research alone is not implementation; unit coverage alone is not qualification. Upstream popularity does not establish suitability or security.

## Reuse boundaries

- Prefer maintained, pinned permissive libraries and small auditable components over transplanting an entire application's architecture.
- Record original repository, exact commit/version, source paths, license and copyright notices, local changes, and any bundled transitive licenses.
- Do not copy copyleft code into the undecided-license project; document incompatible candidates and use compatible alternatives or original code.
- Do not add analytics, external endpoints, mandatory services, online asset imports, or new ledger authority from upstream examples.
- Preserve independent nested modules. No module imports another unfinished feature. No root application configuration, migrations, route registration, release workflow or main-branch edits.
- Browser/manual/platform tests and final core adapter qualification remain explicit gates if they cannot be performed in the isolated environment.

## Publishing

Research and implementation checkpoints go to the existing feature branch. Updates use a branch-head lease; verify the published tree equals the tested local tree. Do not force-overwrite concurrent work. Do not open auto-merge PRs, merge these branches, tag releases or deploy.

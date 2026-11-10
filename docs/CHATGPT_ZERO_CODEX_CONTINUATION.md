# ChatGPT Zero-Codex Continuation Specification

Status: proposed coordination policy. Does not itself create a timer or detect account credit balances.

## Executor and costs
Use ChatGPT direct GitHub connector and available execution tools only. No Codex, Codex CLI/cloud, paid OpenAI API, or credit-consuming agents. GitHub Actions and ChatGPT usage have independent limits and potential charges.

## Ownership
Use `chatgpt/zero-codex/` branches for implementation and `docs/handoffs/CHATGPT_ZERO_CODEX_LATEST.md` for checkpoints. This is advisory ChatGPT-only ownership, not technical enforcement. Never overwrite, rebase, close, merge or edit another agent's active branches/PRs. Before each run inspect repository default branch (not assumed to be main), AGENTS.md and nested instructions, specifications, recent commits, open PRs, source-pinned handoffs and CI. Do not treat draft work as merged.

## Sequential work queue
1. Reconcile current application development in PR #19 with prompt-only main and source-pinned handoffs; qualify its exact-head tests and avoid duplicate integration.
2. Reproduce and fix remaining P0/P1 defects and independently validated A1/A3 security findings without changing financial rules or weakening tests.
3. Improve invitation/login return-link handling, no-script and accessible browser flows, API contracts and Go service correctness.
4. Harden ledger accounting invariants, historical balances, dual-control approvals and recovery with property/negative tests.
5. Complete staged operations documentation, cross-platform Go/Playwright CI, reproducible builds and release readiness evidence.
6. Record license, human accessibility, external security review, financial-policy and deployment gates as owner-owned.

## Run protocol
1. Inspect prior automation handoff and exact current PR/CI status. If the prior task or its CI is still active, do not duplicate; defer to next hourly invocation.
2. Select the highest-priority unblocked independent task and finish one slice at a time. If approval is blocked, document it and work on another safe task.
3. Follow test-driven development: reproduce a defect with a failing test, implement a minimal change, refactor and run appropriate focused and broad checks. Never invent passing tests or security reviews.
4. Preserve existing security, privacy, financial and architectural invariants. Do not weaken lint, security gates, test assertions, or fail-closed behavior.
5. Commit to isolated branch and create/update draft PR. Record exact base/head SHA, changed paths, tests, CI URLs, blockers and next steps in the handoff.
6. Continue on future scheduled runs until features are complete or as near-complete as available tools and approvals permit. Do not claim continuous execution between runs.
7. No main/default-branch merge, release, deployment, credential use, financial-policy change, production trust-path change, or security-signoff claim without required separate authorization.

## Acceptance
Complete means source, tests, docs, exact-head CI and required approvals are satisfied. Near-complete means every remaining gate has an evidence-based blocker and next action. Do not invent completion percentages.

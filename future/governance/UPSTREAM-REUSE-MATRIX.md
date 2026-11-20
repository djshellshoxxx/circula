# Governance reuse matrix

| Component | Exact version/revision | Reused portion | Adaptation | License evidence |
|---|---|---|---|---|
| alphagov/govuk-frontend | 6.6.0, a11bd3316363ab4702fcef81d6ce06a947c464ef | Radio/fieldset/legend/hint/label HTML structure | Go html/template, three choices, no selected default, native forms; no JS/branding/framework copied | third_party/govuk-frontend/LICENSE.txt, third_party/PROVENANCE.md |
| modernc.org/sqlite | v1.60.1 | Test-only database/sql driver | Unmodified module dependency; runtime receives explicit host DB | third_party/dependency-notices/modernc.org_sqlite@v1.60.1/ |
| Imported Go dependency graph | Exact go.mod/go.sum versions | Test SQLite dependencies | No modified dependency source | third_party/dependency-notices/README.md |

Original research and candidate comparisons remain in UPSTREAM-RESEARCH.md. No GPL/AGPL project source was copied or translated. No new dependency or source transplant was added by this continuation. Popularity or license availability is not security qualification. Root product license and future release packaging remain separate decisions.

# Upstream adaptation and dependency notices

The static ballot radio/fieldset/hint/label patterns in web.go adapt the native HTML component structure of alphagov/govuk-frontend commit a11bd3316363ab4702fcef81d6ce06a947c464ef, package 6.6.0:

- packages/govuk-frontend/src/govuk/components/radios/template.njk
- packages/govuk-frontend/src/govuk/components/fieldset/template.njk
- packages/govuk-frontend/src/govuk/components/radios/template.test.js (structural-test pattern)

Adaptation: Go html/template escaping, three fixed choices, no default selection, server-owned IDs/attributes, native forms, system fonts and original minimal styles. No branding, icons, assets, Sass framework, JavaScript, conditional controls or Nunjucks safe-HTML injection API is copied. The complete selected MIT license is included. GOV.UK does not endorse this project. AGPL/GPL governance applications remain design references; their source was not copied or translated.

modernc.org/sqlite v1.60.1 is the test-only SQLite driver. Runtime code accepts an explicit host database/sql connection and never opens a production DB. All module pins/checksums are in go.mod/go.sum. Direct/transitive license files for actually imported dependencies are included in dependency-notices/. No dependency source is modified. Preserve applicable upstream notices in any distributed dependency bundle. This document does not choose a root product license or approve integration/release.

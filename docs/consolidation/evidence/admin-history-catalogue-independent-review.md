# Independent source review: administrator history catalogue

## Findings and scope

**No behavior or escaping findings identified.** Reviewed immutable production
and test changes in `4ed93b3b0d148303ef2225b12088cfad3a6669e2` against
`fefaf0124dc4270205b1f38220e362868b71476d` on 2026-11-19. This records the
previously completed read-only review; no additional builds were run and no
production or test source was changed by the reviewer.

The scope covers announcements, the remaining administrator-overview and stale
queue labels, and read-only operator history. Catalogue comparison found exactly
74 added entries, explicit positional arguments in their formats, and no
changes to existing values.

## Reviewed boundaries

- **Announcement deletion:** `admin_announcements.html` uses the complete
  ordinary-string accessible name `Delete announcement <title>`, including the
  visible `Delete` verb. Contextual attribute escaping protects user-entered
  titles. Delete actions, record IDs and CSRF fields are unchanged; create
  scheduling, replacement selection, input values and idempotency remain intact.
- **Announcement title and time:** `msgAroundTwo` receives only a catalogue
  key. User titles stay in the template's escaped slot, while the existing
  strong/time markup remains template-owned. Scheduled and already-effective
  rows preserve their original conditions and displayed meaning.
- **Stored operator reasons:** the close-time/reason message likewise invokes
  `msgAroundTwo` without user arguments. Stored operator labels and reasons,
  including `ClosedReason`, remain ordinary escaped data. They cannot collide
  with the helper's fixed marker validation. Known statuses are adapted only
  for display; unknown status strings remain escaped fallback data.
- **Controls and navigation:** operator history remains read-only, with the
  same continuation reference and no grant/close/extension controls. Overview
  links, zero-count action gating, frozen-state explanation, queue timestamps,
  exact monetary strings and pagination remain intact. Queue grammar uses the
  existing counted-message helper; its zero/one/many display does not change
  the threshold or queried data.

The reviewed source includes `h_admin_f2.go`, `h_breakglass.go`, the four
corresponding administrator templates, catalogue additions and focused tests.
The actual-template test source checks hostile names/reasons, complete delete
names, input preservation, route/CSRF/idempotency controls, status fallbacks,
read-only history, queue amounts and message escaping. The small
support-boundaries template loader adds its required `msgAround` registration;
the production helper registration already exists.

Runtime passes reported by the implementing worker belong to that worker's
receipts. This review independently inspected the source and assertions; it
does not reattribute those runs or claim combined/full-suite qualification.

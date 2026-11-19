# Upstream reuse matrix — FUX-105

| Component | Pin and license | Adopted boundary | Decision |
|---|---|---|---|
| modernc SQLite | v1.60.1, GitLab `b122d0417c01508beb55158faedeb64a0c5bfd8a`; BSD notices and bundled notices in third_party | SQLite engine, transactions, durable storage, exact sequence selection and literal `instr` search | Reuse the complete driver as the existing core-compatible nested dependency; do not transplant another application's private-message authorization |
| coder/websocket | v1.8.15, GitHub `9c8faadccd1b679e811a79ce506f8a10237251ad`; ISC | Evaluated for optional hint transport | Not adopted in the selected newer implementation: durable request/response and unread state meet required behavior without an extra transport or dependency |
| centrifugal/centrifuge | v0.40.1, GitHub `79c893e8692d5caf29e4c1e5983191e6e5ee3530`; MIT | Evaluated realtime server | Not adopted: additional broker and session scope is unnecessary |
| mattn/go-sqlite3 | v1.14.53, GitHub `93c7e0a2ec6762a38ab371755d49a2b3f4a12e67`; MIT | Evaluated/test-only driver in older outage implementation | Not adopted in the selected implementation; default modernc tests now execute successfully |

The detailed candidate maintenance, source and license observations remain in UPSTREAM-RESEARCH.md. Those are dated observations, not a new security certification. Native messaging policy, privacy, retention and rendering stay Circula-specific. No other future module is imported. Dependency archives and complete runtime notices are preserved; the project-wide product license is unchanged.

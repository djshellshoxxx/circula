---
title: Requirements traceability matrix
generated: true
generator: scripts/gen-traceability.py
---

# Traceability matrix

> **Generated file - do not edit by hand.** Regenerate with `python3 scripts/gen-traceability.py`. Source: `docs/specs/*.md`. Output is deterministic (no timestamps).

Method and heuristics are in the script docstring. Citation = the ID appears on a line owned by another ID (table row / list item) or inside a `QA-E2E` scenario block. This is a *citation* map, not proof of test coverage; the authoritative gate is QA-GATE-11 once covers-tags exist in code.

## 1. Summary

- Spec files scanned: 20; defined IDs: 1589; FR requirements: 150 (P0 125, P1 25, P2 0, no priority parsed 0); INV: 16; QA-E2E scenarios: 24.
- P0/P1 FR with **no doc 12 citation at all** (hard coverage gaps): **65**.
- P0/P1 FR with no QA-E2E scenario citation (may be covered at lower layers; informational): 71.
- IDs cited but not defined anywhere in `docs/specs`: **1**.

## 2. Coverage gaps

### 2.1 P0/P1 FR with no doc 12 citation (no QA-E2E, no QA-nnn row, no section 5/6 mention)

| FR | Pri | Defined in | Area |
|---|---|---|---|
| FR-ADM-011 | P0 | doc 04 line 181 | FR-ADM |
| FR-ADM-012 | P0 | doc 04 line 182 | FR-ADM |
| FR-ADM-013 | P0 | doc 04 line 183 | FR-ADM |
| FR-ADM-016 | P1 | doc 04 line 186 | FR-ADM |
| FR-ADM-020 | P1 | doc 04 line 190 | FR-ADM |
| FR-AUD-003 | P0 | doc 04 line 224 | FR-AUD |
| FR-AUD-007 | P0 | doc 04 line 228 | FR-AUD |
| FR-AUD-008 | P1 | doc 04 line 229 | FR-AUD |
| FR-AUD-009 | P1 | doc 04 line 230 | FR-AUD |
| FR-BAL-002 | P0 | doc 04 line 154 | FR-BAL |
| FR-BAL-005 | P0 | doc 04 line 157 | FR-BAL |
| FR-BAL-007 | P0 | doc 04 line 159 | FR-BAL |
| FR-BAL-008 | P0 | doc 04 line 160 | FR-BAL |
| FR-DSP-003 | P0 | doc 04 line 200 | FR-DSP |
| FR-DSP-004 | P0 | doc 04 line 201 | FR-DSP |
| FR-DSP-005 | P0 | doc 04 line 202 | FR-DSP |
| FR-DSP-011 | P0 | doc 04 line 208 | FR-DSP |
| FR-DSP-014 | P0 | doc 04 line 211 | FR-DSP |
| FR-DSP-015 | P1 | doc 04 line 212 | FR-DSP |
| FR-EXP-002 | P0 | doc 04 line 239 | FR-EXP |
| FR-EXP-006 | P0 | doc 04 line 243 | FR-EXP |
| FR-LST-002 | P0 | doc 04 line 100 | FR-LST |
| FR-LST-004 | P0 | doc 04 line 102 | FR-LST |
| FR-LST-006 | P0 | doc 04 line 104 | FR-LST |
| FR-LST-007 | P0 | doc 04 line 105 | FR-LST |
| FR-LST-008 | P0 | doc 04 line 106 | FR-LST |
| FR-LST-009 | P0 | doc 04 line 107 | FR-LST |
| FR-LST-010 | P1 | doc 04 line 108 | FR-LST |
| FR-LST-011 | P0 | doc 04 line 109 | FR-LST |
| FR-LST-012 | P0 | doc 04 line 110 | FR-LST |
| FR-LST-013 | P1 | doc 04 line 111 | FR-LST |
| FR-LST-014 | P1 | doc 04 line 112 | FR-LST |
| FR-LST-015 | P0 | doc 04 line 113 | FR-LST |
| FR-MEM-007 | P0 | doc 04 line 75 | FR-MEM |
| FR-MEM-008 | P1 | doc 04 line 76 | FR-MEM |
| FR-MEM-009 | P0 | doc 04 line 77 | FR-MEM |
| FR-MEM-016 | P1 | doc 04 line 84 | FR-MEM |
| FR-MEM-018 | P0 | doc 04 line 86 | FR-MEM |
| FR-MEM-020 | P1 | doc 04 line 88 | FR-MEM |
| FR-MEM-021 | P0 | doc 04 line 89 | FR-MEM |
| FR-MEM-022 | P1 | doc 04 line 90 | FR-MEM |
| FR-MEM-023 | P0 | doc 04 line 91 | FR-MEM |
| FR-NET-001 | P0 | doc 04 line 46 | FR-NET |
| FR-NET-002 | P0 | doc 04 line 47 | FR-NET |
| FR-NET-003 | P0 | doc 04 line 48 | FR-NET |
| FR-NET-006 | P0 | doc 04 line 51 | FR-NET |
| FR-NET-007 | P1 | doc 04 line 52 | FR-NET |
| FR-NET-008 | P0 | doc 04 line 53 | FR-NET |
| FR-NET-009 | P0 | doc 04 line 54 | FR-NET |
| FR-NET-010 | P1 | doc 04 line 55 | FR-NET |
| FR-NET-011 | P1 | doc 04 line 56 | FR-NET |
| FR-NET-013 | P0 | doc 04 line 58 | FR-NET |
| FR-NET-014 | P0 | doc 04 line 59 | FR-NET |
| FR-NET-015 | P0 | doc 04 line 60 | FR-NET |
| FR-NET-016 | P1 | doc 04 line 61 | FR-NET |
| FR-NTF-006 | P1 | doc 04 line 258 | FR-NTF |
| FR-NTF-007 | P1 | doc 04 line 259 | FR-NTF |
| FR-TRD-005 | P0 | doc 04 line 125 | FR-TRD |
| FR-TRD-010 | P0 | doc 04 line 130 | FR-TRD |
| FR-TRD-014 | P0 | doc 04 line 134 | FR-TRD |
| FR-TRD-017 | P0 | doc 04 line 137 | FR-TRD |
| FR-TRD-018 | P0 | doc 04 line 138 | FR-TRD |
| FR-TRD-019 | P1 | doc 04 line 139 | FR-TRD |
| FR-TRD-024 | P0 | doc 04 line 144 | FR-TRD |
| FR-TRD-025 | P0 | doc 04 line 145 | FR-TRD |

### 2.2 Gap counts by FR area (P0/P1 with no doc 12 citation / no QA-E2E citation / total P0+P1)

| Area | No doc 12 citation | No QA-E2E | Total P0/P1 |
|---|---|---|---|
| FR-ADM | 5 | 6 | 20 |
| FR-AUD | 4 | 6 | 9 |
| FR-BAL | 4 | 5 | 11 |
| FR-DSP | 6 | 8 | 15 |
| FR-EXP | 2 | 2 | 8 |
| FR-LST | 12 | 12 | 15 |
| FR-MEM | 9 | 9 | 23 |
| FR-NET | 13 | 13 | 16 |
| FR-NTF | 2 | 2 | 8 |
| FR-TRD | 8 | 8 | 25 |

### 2.3 P0/P1 FR with no QA-E2E scenario (but cited elsewhere in doc 12)

FR-ADM-010, FR-AUD-002, FR-AUD-005, FR-BAL-010, FR-DSP-012, FR-DSP-013

## 3. Undefined IDs (cited but not defined)

Non-breaking hyphens are used below on purpose (see script docstring). Location is `doc:line`.

| ID | Cited at |
|---|---|
| FR‑ADM‑027 | 04:90 |

## 4. FR matrix

Columns: JRN/PER = journeys/personas cited by the FR row or citing the FR (docs 02/04); QA-E2E = scenarios citing it; Doc 12 other = doc 12 rows (QA-nnn etc.) citing it; Cited by = API/UX/SEC/DAT/ARC/OPS/GOV/NTF/PRT/LIF/POL rows citing it.

### FR-ADM

| FR | Pri | JRN/PER | QA-E2E | Doc 12 other | Cited by (API/UX/SEC/DAT/...) |
|---|---|---|---|---|---|
| FR-ADM-001 | P0 | - | QA-E2E-12 | - | POL-003 |
| FR-ADM-002 | P0 | - | QA-E2E-11, QA-E2E-12 | - | GOV-008, GOV-013, POL-026 |
| FR-ADM-003 | P0 | - | QA-E2E-11 | - | GOV-047, POL-022 |
| FR-ADM-004 | P0 | - | QA-E2E-12 | - | - |
| FR-ADM-005 | P0 | - | QA-E2E-12 | - | - |
| FR-ADM-006 | P0 | - | QA-E2E-21 | - | - |
| FR-ADM-007 | P0 | - | QA-E2E-09 | - | UX-045 |
| FR-ADM-008 | P0 | - | QA-E2E-09, QA-E2E-21 | (prose) | - |
| FR-ADM-009 | P0 | - | QA-E2E-08 | - | - |
| FR-ADM-010 | P0 | - | - | INV-012 | UX-020 |
| FR-ADM-011 | P0 | - | - | - | - |
| FR-ADM-012 | P0 | - | - | - | GOV-018 |
| FR-ADM-013 | P0 | - | - | - | - |
| FR-ADM-014 | P0 | - | QA-E2E-12 | - | POL-005 |
| FR-ADM-015 | P0 | - | QA-E2E-08, QA-E2E-09, QA-E2E-11 | (prose) | GOV-044 |
| FR-ADM-016 | P1 | - | - | - | - |
| FR-ADM-017 | P0 | - | QA-E2E-22 | INV-012 | - |
| FR-ADM-018 | P0 | - | QA-E2E-11 | - | GOV-008, UX-103 |
| FR-ADM-019 | P1 | - | QA-E2E-11 | - | - |
| FR-ADM-020 | P1 | - | - | - | GOV-024 |

### FR-AUD

| FR | Pri | JRN/PER | QA-E2E | Doc 12 other | Cited by (API/UX/SEC/DAT/...) |
|---|---|---|---|---|---|
| FR-AUD-001 | P0 | - | QA-E2E-08 | INV-010 | DAT-054, PRT-032 |
| FR-AUD-002 | P0 | - | - | INV-010 | DAT-055 |
| FR-AUD-003 | P0 | - | - | - | - |
| FR-AUD-004 | P0 | - | QA-E2E-09 | - | UX-098 |
| FR-AUD-005 | P0 | - | - | QA-032 | - |
| FR-AUD-006 | P0 | - | QA-E2E-22 | - | - |
| FR-AUD-007 | P0 | - | - | - | DAT-056 |
| FR-AUD-008 | P1 | - | - | - | - |
| FR-AUD-009 | P1 | - | - | - | DAT-063, PRT-032 |

### FR-BAL

| FR | Pri | JRN/PER | QA-E2E | Doc 12 other | Cited by (API/UX/SEC/DAT/...) |
|---|---|---|---|---|---|
| FR-BAL-001 | P0 | - | QA-E2E-01 | - | UX-015 |
| FR-BAL-002 | P0 | - | - | - | - |
| FR-BAL-003 | P0 | - | QA-E2E-19 | INV-008 | UX-027 |
| FR-BAL-004 | P0 | - | QA-E2E-01 | - | UX-030 |
| FR-BAL-005 | P0 | - | - | - | UX-033, UX-064 |
| FR-BAL-006 | P0 | - | QA-E2E-08, QA-E2E-09 | - | UX-017, UX-031, UX-098 |
| FR-BAL-007 | P0 | - | - | - | UX-028 |
| FR-BAL-008 | P0 | - | - | - | DAT-091 |
| FR-BAL-009 | P0 | - | QA-E2E-02, QA-E2E-21 | - | UX-018, UX-025 |
| FR-BAL-010 | P0 | - | - | QA-093 | UX-032 |
| FR-BAL-011 | P1 | - | QA-E2E-13 | QA-071 | - |

### FR-DSP

| FR | Pri | JRN/PER | QA-E2E | Doc 12 other | Cited by (API/UX/SEC/DAT/...) |
|---|---|---|---|---|---|
| FR-DSP-001 | P0 | - | QA-E2E-08 | - | - |
| FR-DSP-002 | P0 | - | QA-E2E-08 | - | - |
| FR-DSP-003 | P0 | - | - | - | - |
| FR-DSP-004 | P0 | - | - | - | - |
| FR-DSP-005 | P0 | - | - | - | - |
| FR-DSP-006 | P0 | - | QA-E2E-08 | - | - |
| FR-DSP-007 | P0 | - | QA-E2E-08 | QA-033 | - |
| FR-DSP-008 | P0 | - | QA-E2E-08 | - | - |
| FR-DSP-009 | P0 | - | QA-E2E-08 | - | - |
| FR-DSP-010 | P0 | - | QA-E2E-08 | QA-033 | - |
| FR-DSP-011 | P0 | - | - | - | - |
| FR-DSP-012 | P0 | - | - | QA-033 | - |
| FR-DSP-013 | P1 | - | - | QA-033 | - |
| FR-DSP-014 | P0 | - | - | - | - |
| FR-DSP-015 | P1 | - | - | - | - |

### FR-EXP

| FR | Pri | JRN/PER | QA-E2E | Doc 12 other | Cited by (API/UX/SEC/DAT/...) |
|---|---|---|---|---|---|
| FR-EXP-001 | P0 | - | QA-E2E-13 | - | PRT-001 |
| FR-EXP-002 | P0 | - | - | - | PRT-023, PRT-028 |
| FR-EXP-003 | P0 | - | QA-E2E-13, QA-E2E-16 | - | PRT-002 |
| FR-EXP-004 | P0 | - | QA-E2E-13 | QA-021, QA-037, QA-070 | PRT-017 |
| FR-EXP-005 | P0 | - | QA-E2E-13 | QA-073 | DAT-072, PRT-036 |
| FR-EXP-006 | P0 | - | - | - | - |
| FR-EXP-007 | P1 | - | QA-E2E-13 | QA-074 | PRT-011 |
| FR-EXP-008 | P1 | - | QA-E2E-13 | QA-073 | PRT-036 |

### FR-LST

| FR | Pri | JRN/PER | QA-E2E | Doc 12 other | Cited by (API/UX/SEC/DAT/...) |
|---|---|---|---|---|---|
| FR-LST-001 | P0 | - | QA-E2E-01 | - | - |
| FR-LST-002 | P0 | - | - | - | UX-076 |
| FR-LST-003 | P0 | - | QA-E2E-01 | - | - |
| FR-LST-004 | P0 | - | - | - | - |
| FR-LST-005 | P0 | - | QA-E2E-01 | - | - |
| FR-LST-006 | P0 | - | - | - | - |
| FR-LST-007 | P0 | - | - | - | - |
| FR-LST-008 | P0 | - | - | - | - |
| FR-LST-009 | P0 | - | - | - | - |
| FR-LST-010 | P1 | - | - | - | - |
| FR-LST-011 | P0 | - | - | - | - |
| FR-LST-012 | P0 | - | - | - | - |
| FR-LST-013 | P1 | - | - | - | - |
| FR-LST-014 | P1 | - | - | - | - |
| FR-LST-015 | P0 | - | - | - | - |

### FR-MEM

| FR | Pri | JRN/PER | QA-E2E | Doc 12 other | Cited by (API/UX/SEC/DAT/...) |
|---|---|---|---|---|---|
| FR-MEM-001 | P0 | - | QA-E2E-01 | - | - |
| FR-MEM-002 | P0 | - | QA-E2E-01 | - | - |
| FR-MEM-003 | P0 | - | QA-E2E-01 | - | - |
| FR-MEM-004 | P0 | - | QA-E2E-16 | - | UX-002 |
| FR-MEM-005 | P0 | - | QA-E2E-01 | - | - |
| FR-MEM-006 | P0 | - | QA-E2E-01 | - | - |
| FR-MEM-007 | P0 | - | - | - | - |
| FR-MEM-008 | P1 | - | - | - | - |
| FR-MEM-009 | P0 | PER-013 | - | - | - |
| FR-MEM-010 | P0 | - | QA-E2E-20 | - | - |
| FR-MEM-011 | P0 | - | QA-E2E-20 | - | UX-018 |
| FR-MEM-012 | P0 | - | QA-E2E-10 | - | - |
| FR-MEM-013 | P0 | - | QA-E2E-10 | - | - |
| FR-MEM-014 | P0 | - | QA-E2E-10 | - | - |
| FR-MEM-015 | P0 | - | QA-E2E-10 | - | - |
| FR-MEM-016 | P1 | - | - | - | - |
| FR-MEM-017 | P0 | - | QA-E2E-10 | - | - |
| FR-MEM-018 | P0 | - | - | - | - |
| FR-MEM-019 | P0 | - | QA-E2E-15 | - | - |
| FR-MEM-020 | P1 | - | - | - | - |
| FR-MEM-021 | P0 | - | - | - | - |
| FR-MEM-022 | P1 | - | - | - | - |
| FR-MEM-023 | P0 | PER-014 | - | - | - |

### FR-NET

| FR | Pri | JRN/PER | QA-E2E | Doc 12 other | Cited by (API/UX/SEC/DAT/...) |
|---|---|---|---|---|---|
| FR-NET-001 | P0 | - | - | - | - |
| FR-NET-002 | P0 | - | - | - | - |
| FR-NET-003 | P0 | - | - | - | POL-039 |
| FR-NET-004 | P0 | - | QA-E2E-24 | - | - |
| FR-NET-005 | P0 | - | QA-E2E-24 | - | GOV-002 |
| FR-NET-006 | P0 | - | - | - | - |
| FR-NET-007 | P1 | - | - | - | GOV-042 |
| FR-NET-008 | P0 | - | - | - | NTF-078 |
| FR-NET-009 | P0 | - | - | - | - |
| FR-NET-010 | P1 | PER-057 | - | - | GOV-073 |
| FR-NET-011 | P1 | - | - | - | GOV-079 |
| FR-NET-012 | P0 | - | QA-E2E-16 | INV-011 | - |
| FR-NET-013 | P0 | - | - | - | - |
| FR-NET-014 | P0 | - | - | - | - |
| FR-NET-015 | P0 | PER-057 | - | - | - |
| FR-NET-016 | P1 | - | - | - | - |

### FR-NTF

| FR | Pri | JRN/PER | QA-E2E | Doc 12 other | Cited by (API/UX/SEC/DAT/...) |
|---|---|---|---|---|---|
| FR-NTF-001 | P0 | - | QA-E2E-21 | - | NTF-001, UX-010 |
| FR-NTF-002 | P0 | - | QA-E2E-12 | - | NTF-074 |
| FR-NTF-003 | P0 | - | QA-E2E-18 | QA-038 | NTF-041, NTF-044 |
| FR-NTF-004 | P0 | - | QA-E2E-09, QA-E2E-15 | QA-038 | NTF-005 |
| FR-NTF-005 | P0 | - | QA-E2E-15, QA-E2E-18 | QA-038 | NTF-024 |
| FR-NTF-006 | P1 | - | - | - | UX-021 |
| FR-NTF-007 | P1 | - | - | - | NTF-022, NTF-059 |
| FR-NTF-008 | P1 | - | QA-E2E-18 | QA-038 | NTF-045 |

### FR-TRD

| FR | Pri | JRN/PER | QA-E2E | Doc 12 other | Cited by (API/UX/SEC/DAT/...) |
|---|---|---|---|---|---|
| FR-TRD-001 | P0 | - | QA-E2E-01, QA-E2E-17 | - | UX-043 |
| FR-TRD-002 | P0 | - | QA-E2E-01 | QA-010 | UX-044, UX-089 |
| FR-TRD-003 | P0 | - | QA-E2E-02 | - | UX-005 |
| FR-TRD-004 | P0 | - | QA-E2E-01, QA-E2E-17 | - | UX-035, UX-037 |
| FR-TRD-005 | P0 | - | - | - | UX-026 |
| FR-TRD-006 | P0 | - | QA-E2E-01, QA-E2E-07 | - | - |
| FR-TRD-007 | P0 | - | QA-E2E-01, QA-E2E-02, QA-E2E-03 | - | - |
| FR-TRD-008 | P0 | - | QA-E2E-02, QA-E2E-03 | - | UX-078 |
| FR-TRD-009 | P0 | - | QA-E2E-02 | - | UX-078 |
| FR-TRD-010 | P0 | - | - | - | - |
| FR-TRD-011 | P0 | - | QA-E2E-07 | - | - |
| FR-TRD-012 | P0 | - | QA-E2E-06 | - | UX-016, UX-104, UX-105 |
| FR-TRD-013 | P0 | - | QA-E2E-06 | - | - |
| FR-TRD-014 | P0 | - | - | - | UX-038 |
| FR-TRD-015 | P0 | - | QA-E2E-04, QA-E2E-05, QA-E2E-19 | - | UX-040 |
| FR-TRD-016 | P0 | - | QA-E2E-03 | INV-005, QA-030, QA-100 | - |
| FR-TRD-017 | P0 | - | - | - | UX-017 |
| FR-TRD-018 | P0 | - | - | - | - |
| FR-TRD-019 | P1 | - | - | - | - |
| FR-TRD-020 | P0 | - | QA-E2E-10 | - | - |
| FR-TRD-021 | P0 | - | QA-E2E-20 | - | - |
| FR-TRD-022 | P0 | - | QA-E2E-01 | - | - |
| FR-TRD-023 | P0 | - | QA-E2E-08 | - | - |
| FR-TRD-024 | P0 | - | - | - | - |
| FR-TRD-025 | P0 | - | - | - | - |

## 5. INV coverage in doc 12

"Section 5 row" = doc 12 owns a row whose first cell is the INV (the ledger-invariant coverage table). QA-E2E and QA rows are citations elsewhere in doc 12.

| INV | Section 5 row | QA-E2E | Other doc 12 owners | Cited by other specs (count) |
|---|---|---|---|---|
| INV-001 | yes | QA-E2E-01 | (prose), QA-001, QA-011, QA-040 | 13 |
| INV-002 | yes | QA-E2E-01, QA-E2E-07 | - | 12 |
| INV-003 | yes | QA-E2E-14, QA-E2E-22, QA-E2E-23 | QA-053 | 20 |
| INV-004 | yes | QA-E2E-08, QA-E2E-09 | - | 15 |
| INV-005 | yes | QA-E2E-02, QA-E2E-03, QA-E2E-21 | (prose), QA-011, QA-100 | 10 |
| INV-006 | yes | - | - | 7 |
| INV-007 | yes | QA-E2E-01, QA-E2E-03, QA-E2E-07, QA-E2E-13, QA-E2E-14 | (prose), QA-040 | 10 |
| INV-008 | yes | QA-E2E-03, QA-E2E-13, QA-E2E-14, QA-E2E-23 | (prose), QA-011 | 16 |
| INV-009 | yes | QA-E2E-04, QA-E2E-05, QA-E2E-18, QA-E2E-19 | (prose), QA-031 | 18 |
| INV-010 | yes | QA-E2E-08, QA-E2E-09, QA-E2E-12 | - | 24 |
| INV-011 | yes | QA-E2E-16 | INV-007, QA-011 | 17 |
| INV-012 | yes | QA-E2E-09, QA-E2E-20, QA-E2E-22 | QA-038 | 29 |
| INV-013 | yes | QA-E2E-24 | - | 8 |
| INV-014 | yes | - | (prose), QA-025 | 22 |
| INV-015 | yes | QA-E2E-01, QA-E2E-10, QA-E2E-20 | (prose), QA-011 | 3 |
| INV-016 | yes | QA-E2E-10 | (prose), QA-001, QA-011 | 6 |

## 6. QA-E2E scenario index

| Scenario | Title | FR cited | INV cited |
|---|---|---|---|
| QA-E2E-01 | Join and first trade | 16 | INV-001, INV-002, INV-007, INV-015 |
| QA-E2E-02 | Credit-limit rejection and recovery | 5 | INV-005 |
| QA-E2E-03 | Concurrent accepts on the same account | 3 | INV-005, INV-007, INV-008 |
| QA-E2E-04 | Retry with the same idempotency key | 1 | INV-009 |
| QA-E2E-05 | Same key, changed payload rejected | 1 | INV-009 |
| QA-E2E-06 | Expiry race at the boundary | 2 | - |
| QA-E2E-07 | Cancel versus accept race | 2 | INV-002, INV-007 |
| QA-E2E-08 | Dispute upheld to Reversal with limit-breach flag | 12 | INV-004, INV-010 |
| QA-E2E-09 | Adjustment visibility | 6 | INV-004, INV-010, INV-012 |
| QA-E2E-10 | Departure with negative balance | 6 | INV-015, INV-016 |
| QA-E2E-11 | Admin quorum | 5 | - |
| QA-E2E-12 | Policy change with notice | 6 | INV-010 |
| QA-E2E-13 | Export reconciliation | 7 | INV-007, INV-008 |
| QA-E2E-14 | Restore drill | 0 | INV-003, INV-007, INV-008 |
| QA-E2E-15 | Recovery flow | 3 | - |
| QA-E2E-16 | Cross-Network isolation | 3 | INV-011 |
| QA-E2E-17 | Keyboard-only trade with screen-reader checks | 2 | - |
| QA-E2E-18 | Notification deduplication | 3 | INV-009 |
| QA-E2E-19 | Unknown outcome after timeout | 2 | INV-009 |
| QA-E2E-20 | Suspension versus accept race | 3 | INV-012, INV-015 |
| QA-E2E-21 | Over-limit account after limit lowered | 4 | INV-005 |
| QA-E2E-22 | Operator has no ledger or content power; break-glass | 2 | INV-003, INV-012 |
| QA-E2E-23 | Migration preserves the ledger | 0 | INV-003, INV-008 |
| QA-E2E-24 | Unit identity locked by first entry | 2 | INV-013 |

## 7. Journeys and personas to QA-E2E (doc 12 QA-105 starting map)

| ID | Scenarios citing it | FR citing it (doc 02/04 rows) |
|---|---|---|
| JRN-001 | - | 0 |
| JRN-002 | - | 0 |
| JRN-003 | - | 0 |
| JRN-004 | - | 0 |
| JRN-005 | - | 0 |
| JRN-006 | - | 0 |
| JRN-007 | - | 0 |
| JRN-008 | QA-E2E-15 | 0 |
| JRN-009 | - | 0 |
| JRN-010 | - | 0 |
| JRN-011 | - | 0 |


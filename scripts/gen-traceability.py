#!/usr/bin/env python3
"""Generate docs/plans/traceability.md from docs/specs/*.md (stdlib only).

Usage: python3 scripts/gen-traceability.py [--check]

Heuristics (documented so the output can be trusted or challenged):
  * A requirement is DEFINED where a table row / list item / heading begins with its ID.
    Its priority is the first P0/P1/P2 cell on that row.
  * A line CITES every other ID on it. The "owner" of a line is the ID it defines (if any).
  * Ranges written as "FR-TRD-001 to 016", "FR-TRD-001..016" or "FR-TRD-001 to FR-TRD-016"
    are expanded for FR-* only.
  * QA-E2E scenarios: every ID inside the scenario block (header line to the next scenario
    or heading) counts as cited by that scenario.
  * IDs cited but not defined are printed with U+2011 (non-breaking hyphen) instead of '-'
    so that scripts/check-specs.py does not flag this generated file itself.
--check exits 1 if there are undefined IDs or P0/P1 FR with no doc-12 citation.
"""
import re, sys, pathlib, collections

ROOT = pathlib.Path(__file__).resolve().parent.parent
SPECS = ROOT / 'docs' / 'specs'
OUT = ROOT / 'docs' / 'plans' / 'traceability.md'

ID = re.compile(r'\b((?:FR-[A-Z]{3}|[A-Z]{2,4})-\d{3})\b')
DEF = re.compile(r'^(?:\|\s*(?:<a [^>]*><\/a>)?[*`]*|\|\s*[*`]*|[-*]\s+[*`]*|#+\s+[*`]*|[*`]+)((?:FR-[A-Z]{3}|[A-Z]{2,4})-\d{3})\b')
PFX = {'VIS', 'PER', 'JRN', 'DOM', 'INV', 'ARC', 'DAT', 'API', 'UX', 'SEC', 'GOV', 'OPS', 'QA', 'ASM', 'RSK',
       'DEC', 'LIF', 'POL', 'NTF', 'PRT', 'LEG', 'FR'}
SKIP = re.compile(r'-OD-|-E2E-|-Q-|-G-|-GATE-')
E2E_HDR = re.compile(r'^\*\*(QA-E2E-\d{2})\b')
E2E_ID = re.compile(r'\bQA-E2E-\d{2}\b')
RANGE = re.compile(r'\b(FR-[A-Z]{3})-(\d{3})\s*(?:to|–|—|\.\.|\.\.\.)\s*(?:\1-)?(\d{3})\b')


def pfx(i):
    return i.split('-')[0]


def area(i):
    return i.rsplit('-', 1)[0]


def expand(line):
    """Return all IDs on a line, expanding FR ranges."""
    ids = []
    for m in ID.finditer(line):
        i = m.group(1)
        if pfx(i) in PFX and not SKIP.search(i):
            ids.append(i)
    for m in RANGE.finditer(line):
        a, lo, hi = m.group(1), int(m.group(2)), int(m.group(3))
        if lo < hi and hi - lo < 200:
            ids.extend(f'{a}-{n:03d}' for n in range(lo, hi + 1))
    return ids


def docnum(path):
    return path.name.split('-')[0]


def main():
    files = sorted(SPECS.glob('[0-9][0-9]-*.md'))
    defs = {}      # id -> (doc, line, priority)
    cites = collections.defaultdict(lambda: collections.defaultdict(set))  # id -> {citer_id_or_doc: set(docs)}
    # structures
    cited_by = collections.defaultdict(set)   # id -> set of (owner_id, doc)
    e2e_cites = collections.defaultdict(set)  # id -> set of QA-E2E ids
    e2e_title = {}
    e2e_defined = set()
    all_refs = []  # (id, doc, line)
    doc12_cites = collections.defaultdict(set)  # id -> set(owner ids or 'prose') in doc 12 outside E2E blocks

    for f in files:
        dn = docnum(f)
        cur_e2e = None
        for n, line in enumerate(f.read_text().splitlines(), 1):
            m = DEF.match(line)
            owner = None
            if m and pfx(m.group(1)) in PFX and not SKIP.search(m.group(1)):
                owner = m.group(1)
                if owner not in defs:
                    pr = re.search(r'\|\s*(P[012])\s*\|', line)
                    defs[owner] = (dn, n, pr.group(1) if pr else None)
            h = E2E_HDR.match(line)
            if dn == '12':
                if h:
                    cur_e2e = h.group(1)
                    e2e_defined.add(cur_e2e)
                    e2e_title[cur_e2e] = re.sub(r'^\*\*QA-E2E-\d{2}\s*[—-]\s*', '', line).strip('* ')
                elif line.startswith('#'):
                    cur_e2e = None
            ids = expand(line)
            for i in ids:
                all_refs.append((i, dn, n))
                if i != owner:
                    if owner:
                        cited_by[i].add((owner, dn))
                    else:
                        cited_by[i].add(('(prose)', dn))
                if dn == '12':
                    if cur_e2e and not h:
                        e2e_cites[i].add(cur_e2e)
                    elif cur_e2e and h:
                        pass
                    else:
                        if i != owner:
                            doc12_cites[i].add(owner or '(prose)')
            # E2E index table rows (section 12.2): | QA-E2E-nn | title | P0 | persona/journeys |
            if dn == '12' and re.match(r'^\|\s*QA-E2E-\d{2}\s*\|', line):
                e2e_defined.add(E2E_ID.search(line).group(0))

    # ---- helpers
    frs = sorted(i for i in defs if pfx(i) == 'FR')
    invs = sorted(i for i in defs if pfx(i) == 'INV')
    undefined = collections.defaultdict(set)
    for i, dn, n in all_refs:
        if i not in defs:
            undefined[i].add(f'{dn}:{n}')

    def j(items, limit=14):
        items = sorted(items)
        s = ', '.join(items[:limit])
        return s + (f' (+{len(items) - limit})' if len(items) > limit else '') if items else '-'

    def safe(i):
        return i.replace('-', '‑')

    fr_rows = []
    gaps_no_doc12, gaps_no_e2e = [], []
    for fr in frs:
        dn, ln, pr = defs[fr]
        src = {o for o, d in cited_by[fr] if pfx(o) in ('JRN', 'PER')}
        # FR row citing JRN/PER
        # (handled via cited_by reverse below)
        fr_rows.append(fr)
    # reverse: FR rows citing JRN/PER: need FR as owner citing JRN/PER
    fr_cites_src = collections.defaultdict(set)
    for tgt, owners in cited_by.items():
        if pfx(tgt) in ('JRN', 'PER'):
            for o, d in owners:
                if pfx(o) == 'FR':
                    fr_cites_src[o].add(tgt)

    other_pfx = ('API', 'UX', 'SEC', 'DAT', 'ARC', 'OPS', 'GOV', 'NTF', 'PRT', 'LIF', 'POL')
    out = []
    w = out.append
    w('---')
    w('title: Requirements traceability matrix')
    w('generated: true')
    w('generator: scripts/gen-traceability.py')
    w('---')
    w('')
    w('# Traceability matrix')
    w('')
    w('> **Generated file - do not edit by hand.** Regenerate with `python3 scripts/gen-traceability.py`. '
      'Source: `docs/specs/*.md`. Output is deterministic (no timestamps).')
    w('')
    w('Method and heuristics are in the script docstring. Citation = the ID appears on a line owned by another ID '
      '(table row / list item) or inside a `QA-E2E` scenario block. This is a *citation* map, not proof of test '
      'coverage; the authoritative gate is QA-GATE-11 once covers-tags exist in code.')
    w('')
    npri = collections.Counter(defs[f][2] for f in frs)
    w('## 1. Summary')
    w('')
    w(f'- Spec files scanned: {len(files)}; defined IDs: {len(defs)}; FR requirements: {len(frs)} '
      f'(P0 {npri["P0"]}, P1 {npri["P1"]}, P2 {npri["P2"]}, no priority parsed {npri[None]}); '
      f'INV: {len(invs)}; QA-E2E scenarios: {len(e2e_defined)}.')

    # classify gaps
    for fr in frs:
        pr = defs[fr][2]
        if pr in ('P0', 'P1'):
            in_e2e = bool(e2e_cites.get(fr))
            in12 = bool(doc12_cites.get(fr))
            if not in_e2e:
                gaps_no_e2e.append(fr)
            if not in_e2e and not in12:
                gaps_no_doc12.append(fr)
    w(f'- P0/P1 FR with **no doc 12 citation at all** (hard coverage gaps): **{len(gaps_no_doc12)}**.')
    w(f'- P0/P1 FR with no QA-E2E scenario citation (may be covered at lower layers; informational): {len(gaps_no_e2e)}.')
    w(f'- IDs cited but not defined anywhere in `docs/specs`: **{len(undefined)}**.')
    w('')

    w('## 2. Coverage gaps')
    w('')
    w('### 2.1 P0/P1 FR with no doc 12 citation')
    w('')
    w('No QA-E2E scenario, no QA-nnn row and no other doc 12 mention cites these requirements.')
    w('')
    if gaps_no_doc12:
        w('| FR | Pri | Defined in | Area |')
        w('|---|---|---|---|')
        for fr in gaps_no_doc12:
            w(f'| {fr} | {defs[fr][2]} | doc {defs[fr][0]} line {defs[fr][1]} | {area(fr)} |')
    else:
        w('None.')
    w('')
    w('### 2.2 Gap counts by FR area')
    w('')
    w('Columns: P0/P1 with no doc 12 citation; P0/P1 with no QA-E2E citation; total P0/P1.')
    w('')
    w('| Area | No doc 12 citation | No QA-E2E | Total P0/P1 |')
    w('|---|---|---|---|')
    ar = collections.defaultdict(lambda: [0, 0, 0])
    for fr in frs:
        if defs[fr][2] in ('P0', 'P1'):
            ar[area(fr)][2] += 1
            if fr in gaps_no_doc12:
                ar[area(fr)][0] += 1
            if fr in gaps_no_e2e:
                ar[area(fr)][1] += 1
    for a in sorted(ar):
        w(f'| {a} | {ar[a][0]} | {ar[a][1]} | {ar[a][2]} |')
    w('')
    w('### 2.3 P0/P1 FR with no QA-E2E scenario (but cited elsewhere in doc 12)')
    w('')
    w(', '.join(sorted(set(gaps_no_e2e) - set(gaps_no_doc12))) or 'None.')
    w('')

    w('## 3. Undefined IDs (cited but not defined)')
    w('')
    w('Non-breaking hyphens are used below on purpose (see script docstring). Location is `doc:line`.')
    w('')
    if undefined:
        w('| ID | Cited at |')
        w('|---|---|')
        for i in sorted(undefined):
            w(f'| {safe(i)} | {j(undefined[i], 6)} |')
    else:
        w('None.')
    w('')

    w('## 4. FR matrix')
    w('')
    w('Columns: JRN/PER = journeys/personas cited by the FR row or citing the FR (docs 02/04); QA-E2E = scenarios '
      'citing it; Doc 12 other = doc 12 rows (QA-nnn etc.) citing it; Cited by = API/UX/SEC/DAT/ARC/OPS/GOV/NTF/PRT/LIF/POL rows citing it.')
    w('')
    for a in sorted({area(f) for f in frs}):
        w(f'### {a}')
        w('')
        w('| FR | Pri | JRN/PER | QA-E2E | Doc 12 other | Cited by (API/UX/SEC/DAT/...) |')
        w('|---|---|---|---|---|---|')
        for fr in [f for f in frs if area(f) == a]:
            src = fr_cites_src.get(fr, set()) | {o for o, d in cited_by[fr] if pfx(o) in ('JRN', 'PER')}
            oth = sorted(o for o, d in cited_by[fr] if pfx(o) in other_pfx)
            d12 = sorted(o for o in doc12_cites.get(fr, set()) if o != '(prose)' and not o.startswith('QA-E2E'))
            if doc12_cites.get(fr) and not d12:
                d12 = ['(prose)']
            w(f'| {fr} | {defs[fr][2] or "?"} | {j(src, 6)} | {j(e2e_cites.get(fr, set()), 8)} | {j(d12, 6)} | {j(oth, 8)} |')
        w('')

    w('## 5. INV coverage in doc 12')
    w('')
    w('"Section 5 row" = doc 12 owns a row whose first cell is the INV (the ledger-invariant coverage table). '
      'QA-E2E and QA rows are citations elsewhere in doc 12.')
    w('')
    w('| INV | Section 5 row | QA-E2E | Other doc 12 owners | Cited by other specs (count) |')
    w('|---|---|---|---|---|')
    for inv in invs:
        row = any(re.match(r'^\|\s*' + inv + r'\b', l) for l in (SPECS / next(p.name for p in files if docnum(p) == '12')).read_text().splitlines())
        d12 = sorted(o for o in doc12_cites.get(inv, set()) if not o.startswith('QA-E2E'))
        ext = len({o for o, d in cited_by[inv] if d != '12' and d != '03'})
        w(f'| {inv} | {"yes" if row else "NO"} | {j(e2e_cites.get(inv, set()), 8)} | {j(d12, 6)} | {ext} |')
    w('')

    w('## 6. QA-E2E scenario index')
    w('')
    w('| Scenario | Title | FR cited | INV cited |')
    w('|---|---|---|---|')
    for e in sorted(e2e_defined):
        fr = sorted(i for i, s in e2e_cites.items() if e in s and pfx(i) == 'FR')
        iv = sorted(i for i, s in e2e_cites.items() if e in s and pfx(i) == 'INV')
        w(f'| {e} | {e2e_title.get(e, "")[:60]} | {len(fr)} | {j(iv, 8)} |')
    w('')

    w('## 7. Journeys and personas to QA-E2E (doc 12 QA-105 starting map)')
    w('')
    w('| ID | Scenarios citing it | FR citing it (doc 02/04 rows) |')
    w('|---|---|---|')
    for i in sorted(k for k in defs if pfx(k) in ('JRN',)):
        frs_c = sorted(o for o, d in cited_by[i] if pfx(o) == 'FR')
        w(f'| {i} | {j(e2e_cites.get(i, set()), 8)} | {len(frs_c)} |')
    w('')
    OUT.parent.mkdir(parents=True, exist_ok=True)
    OUT.write_text('\n'.join(out) + '\n')
    print(f'wrote {OUT.relative_to(ROOT)}: {len(frs)} FR, {len(invs)} INV, {len(gaps_no_doc12)} hard gaps, '
          f'{len(gaps_no_e2e)} without E2E, {len(undefined)} undefined')
    if '--check' in sys.argv and (undefined or gaps_no_doc12):
        sys.exit(1)


if __name__ == '__main__':
    main()

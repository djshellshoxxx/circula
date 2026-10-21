#!/usr/bin/env python3
"""Spec lint: relative links, duplicate/dangling requirement IDs. Usage: python3 scripts/check-specs.py"""
import re, sys, pathlib, collections
root = pathlib.Path(__file__).resolve().parent.parent
files = sorted(root.glob('docs/**/*.md')) + [p for p in [root/'README.md'] if p.exists()]
ID = re.compile(r'\b((?:FR-[A-Z]{3}|[A-Z]{2,4})-\d{3})\b')
# definitions: table rows or list items beginning with an ID, or headings
DEF = re.compile(r'^(?:\|\s*(?:<a [^>]*><\/a>)?[*`]*|\|\s*[*`]*|[-*]\s+[*`]*|#+\s+[*`]*|[*`]+)((?:FR-[A-Z]{3}|[A-Z]{2,4})-\d{3})\b')
PFX = {'VIS','PER','JRN','DOM','INV','ARC','DAT','API','UX','SEC','GOV','OPS','QA','ASM','RSK','DEC','LIF','POL','NTF','PRT','LEG','FR'}
defs = collections.defaultdict(list); refs = []
errs = []
for f in files:
    txt = f.read_text()
    rel = f.relative_to(root)
    for n, line in enumerate(txt.splitlines(), 1):
        m = DEF.match(line)
        if m and m.group(1).split('-')[0] in PFX and rel.parts[:2] == ('docs','specs'): defs[m.group(1)].append((str(rel), n))
        for i in ID.findall(line):
            if i.split('-')[0] in PFX and not re.search(r'-OD-|-E2E-|-Q-|-G-', i): refs.append((i, str(rel), n))
    for m in re.finditer(r'\]\(([^)#\s]+)(#[^)]*)?\)', txt):
        t = m.group(1)
        if re.match(r'^[a-z]+:', t): continue
        if not (f.parent / t).resolve().exists(): errs.append(f'BROKEN LINK {rel}: {t}')
# ranges like SEC-001..SEC-010 or SEC-001–010 are expanded as definitions-by-range references; ignore for dangling
for i, locs in defs.items():
    # duplicates within the same doc are the real problem (tables repeat IDs legitimately across docs only when citing)
    byf = collections.Counter(l[0] for l in locs)
    for f, c in byf.items():
        if c > 1 and not i.startswith('INV'): errs.append(f'DUP ID {i} defined {c}x in {f}')
seen = set()
for i, f, n in refs:
    if i not in defs and (i, f) not in seen:
        seen.add((i, f)); errs.append(f'DANGLING {i} in {f}:{n}')
print('\n'.join(errs) or 'OK'); print(f'{len(defs)} defined IDs, {len(refs)} refs, {len(errs)} problems')
sys.exit(1 if errs else 0)

#!/usr/bin/env python3
"""Link, anchor, ID, and convention checker for docs/specs.

Usage: python3 tools/check_specs.py docs/specs
Exits non-zero on duplicate or undefined requirement IDs, broken relative links or
anchors, missing header/open-questions/change-history sections, or banned adjectives.
Money-word mentions are listed for review only (most are negations or prohibition lists).
"""
import re, sys, os, collections
root = sys.argv[1]
files = sorted(f for f in os.listdir(root) if f.endswith('.md'))
raw = {f: open(os.path.join(root, f), encoding='utf-8').read() for f in files}
def strip_code(t):
    t = re.sub(r'```.*?```', '', t, flags=re.S)
    return re.sub(r'`[^`\n]*`', '', t)
text = {f: strip_code(t) for f, t in raw.items()}
anchors = {f: set(re.findall(r'<a id="([^"]+)"></a>', t)) for f, t in text.items()}
# heading anchors (GitHub style, approximate)
def slug(h):
    h = re.sub(r'<[^>]+>', '', h).strip().lower()
    h = re.sub(r'[^\w\- ]', '', h)
    return h.replace(' ', '-')
for f, t in text.items():
    for m in re.finditer(r'^#{1,6}\s+(.+?)\s*$', t, re.M):
        anchors[f].add(slug(m.group(1)))
idre = re.compile(r'\b(VIS|PER|JRN|DOM|FR|ARCH|DATA|API|UX|SEC|GOV|OPS|QA|E2E|ASM|RISK|DEC)-(\d{3})\b')
defined = collections.defaultdict(list)   # id -> [files]
for f, t in text.items():
    for a in re.findall(r'<a id="([a-z0-9]+-\d{3})"></a>', t):
        defined[a.upper()].append(f)
owner = {'VIS':'01','PER':'02','JRN':'02','DOM':'03','FR':'04','ARCH':'05','DATA':'06','API':'07','UX':'08','SEC':'09','GOV':'10','OPS':'11','QA':'12','E2E':'12','ASM':'13','RISK':'13','DEC':'13'}
problems = []
# duplicates and wrong-owner definitions
for i, fs in defined.items():
    if len(fs) > 1: problems.append(f'DUP-ID {i} defined in {fs}')
    p = i.split('-')[0]
    for f in fs:
        if not f.startswith(owner[p]): problems.append(f'WRONG-OWNER {i} defined in {f}')
# referenced-but-undefined
refs = collections.defaultdict(set)
for f, t in text.items():
    for line in t.splitlines():
        line = re.sub(r'[A-Z0-9]+-\d{3}\s+to\s+[A-Z0-9]+-\d{3}', '', line)
        for p, n in idre.findall(line):
            refs[f'{p}-{n}'].add(f)
for i, fs in sorted(refs.items()):
    if i not in defined: problems.append(f'UNDEFINED-ID {i} referenced in {sorted(fs)}')
# links
linkre = re.compile(r'\[[^\]]*\]\(([^)\s]+)\)')
for f, t in text.items():
    for target in linkre.findall(t):
        if target.startswith(('http://', 'https://', 'mailto:')): continue
        path, _, frag = target.partition('#')
        if path == '':
            tf = f
        else:
            tf = os.path.normpath(os.path.join(os.path.dirname(f), path))
            if tf not in text:
                full = os.path.normpath(os.path.join(root, path))
                if not os.path.exists(full): problems.append(f'BROKEN-LINK {f}: {target}')
                continue
        if frag and frag.lower() not in anchors.get(tf, set()):
            problems.append(f'BROKEN-ANCHOR {f}: {target}')
# conventions
banned = re.compile(r'\b(user-friendly|intuitive|robust|scalable)\b', re.I)
money = re.compile(r'\b(money|currency|cash|funds|deposit|payment|pay)\b', re.I)
for f, t in text.items():
    if f in ('README.md', '00-conventions.md'): continue
    if '| Status |' not in t: problems.append(f'NO-HEADER-TABLE {f}')
    if not re.search(r'^##+\s+\d*\.?\s*Open questions', t, re.M): problems.append(f'NO-OPEN-QUESTIONS {f}')
    if not re.search(r'^##+\s+\d*\.?\s*Change history', t, re.M): problems.append(f'NO-CHANGE-HISTORY {f}')
    for m in banned.finditer(t):
        ln = t.count('\n', 0, m.start()) + 1
        problems.append(f'BANNED-WORD {f}:{ln} "{m.group(0)}"')
    for m in money.finditer(t):
        ln = t.count('\n', 0, m.start()) + 1
        line = t.splitlines()[ln-1]
        problems.append(f'MONEY-WORD {f}:{ln} "{m.group(0)}" :: {line.strip()[:110]}')
counts = {f: len(re.findall(r'<a id="[a-z0-9]+-\d{3}"></a>', t)) for f, t in text.items()}
print('ID counts:', counts)
sev = [p for p in problems if not p.startswith('MONEY-WORD')]
print(f'{len(sev)} hard problems, {len(problems)-len(sev)} money-word mentions to review')
for p in problems: print(p)
sys.exit(1 if sev else 0)

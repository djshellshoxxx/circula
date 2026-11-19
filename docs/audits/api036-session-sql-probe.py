#!/usr/bin/env python3
"""Measure the checked-in session SQL with SQLite's VM progress counter.

This supplements, rather than replaces, production Go-driver tests and
allocation benchmarks. All input identities and credentials are synthetic. The probe uses only an
in-memory database and standard-library Python modules.

Usage: python3 docs/audits/api036-session-sql-probe.py [REPOSITORY_ROOT]
The default root is derived from this file, independent of the working directory.
"""

import json
import re
import sqlite3
import subprocess
import sys
import time
from pathlib import Path

if len(sys.argv) > 2:
    raise SystemExit('usage: api036-session-sql-probe.py [REPOSITORY_ROOT]')
root = (Path(sys.argv[1]) if len(sys.argv) == 2 else Path(__file__).resolve().parents[2]).resolve()
sources = '\n'.join(root.joinpath(name).read_text(encoding='utf-8') for name in (
    'internal/store/sqlite_person.go',
    'internal/store/sqlite_session_others.go',
))


def constant(name):
    match = re.search(r'const ' + re.escape(name) + r' = ([^\n]+(?:\n\t[^\n]+)*)', sources)
    if not match:
        raise AssertionError(f'missing production SQL constant: {name}')
    return ''.join(literal if literal else constant(identifier)
                   for literal, identifier in re.findall(r'`([^`]*)`|([A-Za-z_]\w*)', match.group(1)))


def measured(db, sql, args):
    steps = 0

    def progress():
        nonlocal steps
        steps += 1
        return 0

    changed = db.total_changes
    db.set_progress_handler(progress, 1)
    started = time.perf_counter()
    rows = db.execute(sql, args).fetchall()
    elapsed = time.perf_counter() - started
    db.set_progress_handler(None, 0)
    return dict(rows=len(rows), changed=db.total_changes-changed,
                vm_instructions=steps, milliseconds=round(elapsed * 1000, 4))


def probe(retained):
    with sqlite3.connect(':memory:') as db:
        db.execute('PRAGMA foreign_keys=ON')
        db.execute('PRAGMA trusted_schema=OFF')
        migrations = sorted(root.joinpath('internal/store/migrations').glob('*.sql'))
        for migration in migrations:
            db.executescript('BEGIN IMMEDIATE;\n' + migration.read_text(encoding='utf-8') + '\nCOMMIT;')
        db.execute("INSERT INTO persons(id,login,password_hash,created_at) VALUES ('owner','owner','hash',0),('foreign','foreign','hash',0)")
        db.executemany('INSERT INTO sessions(id,person_id,created_at,last_seen_at,expires_at,revoked_at,public_id) VALUES (?,?,?,?,?,?,?)', (
            (f'private-{i}', 'owner', i // 3, 0, 0 if i % 3 else 999999,
             None if i % 10 == 0 else 1, f'{i:064x}')
            for i in range(1, retained + 1)
        ))
        db.executemany('INSERT INTO sessions(id,person_id,created_at,last_seen_at,expires_at,public_id) VALUES (?,?,?,?,?,?)', (
            ('current', 'owner', -1, 0, 999999, f'{retained+1:064x}'),
            ('foreign-current', 'foreign', retained, 0, 999999, f'{retained+2:064x}'),
        ))
        db.commit()
        anchor = retained // 2
        cases = [
            ('first', 'sessionListFirstSQL', ('owner', 25)),
            ('deep', 'sessionListAfterSQL', ('owner', anchor // 3, f'{anchor:064x}', 25)),
            ('previous', 'sessionListBeforeSQL', ('owner', anchor // 3, f'{anchor:064x}', 25)),
            ('direct', 'sessionOwnedSQL', (f'{retained:064x}', 'owner')),
            ('bulk_initial', 'sessionRevokeOthersSQL', (1000, 'owner', 'current', 'current', 'owner')),
            ('bulk_repeat', 'sessionRevokeOthersSQL', (1000, 'owner', 'current', 'current', 'owner')),
            ('bulk_foreign_current', 'sessionRevokeOthersSQL', (1000, 'owner', 'foreign-current', 'foreign-current', 'owner')),
        ]
        results = {}
        for name, query_name, args in cases:
            query = constant(query_name)
            plan = [row[3] for row in db.execute('EXPLAIN QUERY PLAN ' + query, args)]
            assert not any('SCAN ' in part or 'TEMP B-TREE' in part for part in plan), plan
            result = measured(db, query, args)
            result['plan'] = plan
            results[name] = result
        assert all(results[name]['rows'] == 25 for name in ('first', 'deep', 'previous'))
        assert results['direct']['rows'] == 1
        assert results['bulk_initial']['changed'] == retained // 10
        assert results['bulk_repeat']['changed'] == 0
        assert results['bulk_foreign_current']['changed'] == 0
        assert db.execute("SELECT COUNT(*) FROM sessions WHERE id IN ('current','foreign-current') AND revoked_at IS NULL").fetchone()[0] == 2
        assert db.execute('PRAGMA integrity_check').fetchone()[0] == 'ok'
        assert not db.execute('PRAGMA foreign_key_check').fetchall()
        return dict(retained_history=retained, migrations=len(migrations), results=results)


report = dict(
    sqlite_version=sqlite3.sqlite_version,
    source_commit=subprocess.check_output(['git', '-C', str(root), 'rev-parse', 'HEAD'], text=True).strip(),
    note='SQLite VM counts from actual production SQL; Python adapter, not Go allocation qualification.',
    samples=[probe(n) for n in (100, 10000, 100000)],
)
print(json.dumps(report, indent=2))

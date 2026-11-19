"""SQL-level smoke for the final session query union; not Go-driver qualification."""
import hashlib
import json
from pathlib import Path
import re
import sqlite3
import subprocess

ROOT = Path(__file__).resolve().parents[4]
OUT = Path(__file__).with_suffix(".json")
source = (ROOT / "internal/store/sqlite_person.go").read_text() + "\n" + (ROOT / "internal/store/sqlite_session_others.go").read_text()

def sql_constant(name):
    match = re.search(r"^\s*(?:const\s+)?" + re.escape(name) + r"\s*=(?:[ \t]*//[^\n]*)?\s*((?:`[^`]*`|[A-Za-z_][A-Za-z_0-9]*)(?:\s*\+\s*(?:`[^`]*`|[A-Za-z_][A-Za-z_0-9]*))*)", source, re.MULTILINE)
    if not match:
        raise AssertionError("missing production SQL " + name)
    return "".join(part[1:-1] if part.startswith("`") else sql_constant(part)
                   for part in re.findall(r"`[^`]*`|[A-Za-z_][A-Za-z_0-9]*", match.group(1)))

db = sqlite3.connect(":memory:", isolation_level=None)
db.execute("PRAGMA foreign_keys=ON")
migrations = sorted((ROOT / "internal/store/migrations").glob("*.sql"))
for path in migrations:
    db.executescript(path.read_text())
db.executemany("INSERT INTO persons(id,login,password_hash,created_at) VALUES (?,?,?,0)",
               [(person, person, "fixture") for person in ("owner", "foreign")])
db.execute("BEGIN")
db.executemany("INSERT INTO sessions(id,person_id,created_at,last_seen_at,expires_at,revoked_at) VALUES (?,?,?,?,?,?)",
    [(f"private-{i}", "owner", i//3, 0, 9999999999999, 1 if 2 <= i <= 9000 else None)
     for i in range(1, 10001)] +
    [(f"foreign-{i}", "foreign", i, 0, 9999999999999, None) for i in range(1, 1001)])
db.executemany("INSERT INTO api_tokens(id,person_id,name,token_hash,scopes,created_at) VALUES (?,?,?,?,?,0)",
    [(f"token-{i:06d}", "owner", "fixture", f"fixture-{i}", '["read"]') for i in range(10000)])
db.execute("COMMIT")
selector = db.execute("SELECT public_id FROM sessions WHERE id='private-1'").fetchone()[0]
cases = [
    ("first", "sessionListFirstSQL", ("owner", 100), "SEARCH sessions USING INDEX sessions_owner_page (person_id=?)"),
    ("after", "sessionListAfterSQL", ("owner", 2000, "a"*64, 100), "SEARCH sessions USING INDEX sessions_owner_page (person_id=? AND (created_at,public_id)<(?,?))"),
    ("before", "sessionListBeforeSQL", ("owner", 2000, "a"*64, 100), "SEARCH sessions USING INDEX sessions_owner_page (person_id=? AND (created_at,public_id)>(?,?))"),
    ("owned", "sessionOwnedSQL", (selector, "owner"), "SEARCH sessions USING INDEX sessions_public_id (public_id=?)"),
    ("bulk", "sessionRevokeOthersSQL", (1000, "owner", "private-1", "private-1", "owner"), "SEARCH sessions USING INDEX sessions_owner_unrevoked (person_id=?)"),
    ("tokens", "tokenListSQL", ("owner",), "SEARCH api_tokens USING INDEX api_tokens_person_order (person_id=?)"),
]
plans = []
for label, name, args, expected in cases:
    query = sql_constant(name)
    details = [row[3] for row in db.execute("EXPLAIN QUERY PLAN " + query, args)]
    combined = "\n".join(details)
    assert expected in combined and "SCAN " not in combined and "TEMP B-TREE" not in combined, details
    rows = None if label == "bulk" else len(db.execute(query, args).fetchall())
    if label in ("first", "after", "before", "tokens"):
        assert rows == 100
    if label == "owned":
        assert rows == 1
    plans.append({"case": label, "production_constant": name, "sql_sha256": hashlib.sha256(query.encode()).hexdigest(), "rows": rows, "plan": details})
bulk = sql_constant("sessionRevokeOthersSQL")
args = cases[4][2]
db.execute("BEGIN")
changed = db.execute(bulk, args).rowcount
assert changed == 1000, changed
db.execute("ROLLBACK")
assert db.execute("SELECT count(*) FROM sessions WHERE person_id='owner' AND revoked_at IS NULL").fetchone()[0] == 1001
assert db.execute(bulk, args).rowcount == 1000
assert db.execute(bulk, args).rowcount == 0
assert db.execute("SELECT revoked_at FROM sessions WHERE id='private-1'").fetchone()[0] is None
assert db.execute("SELECT count(*) FROM sessions WHERE person_id='foreign' AND revoked_at IS NULL").fetchone()[0] == 1000
assert db.execute("SELECT count(*) FROM api_tokens WHERE revoked_at IS NULL").fetchone()[0] == 10000
assert db.execute(bulk, (2000, "foreign", "private-1", "private-1", "foreign")).rowcount == 0
db.execute("UPDATE sessions SET revoked_at=1 WHERE id='private-1'")
assert db.execute(bulk, args).rowcount == 0
integrity = db.execute("PRAGMA integrity_check").fetchone()[0]
foreign_keys = db.execute("PRAGMA foreign_key_check").fetchall()
assert integrity == "ok" and not foreign_keys
report = {
    "scope": "Python SQLite SQL/query/transaction smoke; not Go/modernc or service authorization qualification",
    "source_commit": subprocess.check_output(["git", "-C", str(ROOT), "rev-parse", "HEAD"], text=True).strip(),
    "sqlite": sqlite3.sqlite_version, "migrations": [p.name for p in migrations],
    "session_rows": 11000, "token_rows": 10000, "queries": plans,
    "bulk_changed": changed, "rollback_preserved_state": True, "repeat_changed_zero": True,
    "current_foreign_tokens_preserved": True, "foreign_or_revoked_current_guarded": True,
    "integrity": integrity, "foreign_key_violations": len(foreign_keys)
}
OUT.write_text(json.dumps(report, indent=2) + "\n")
print(json.dumps({"result": "pass", "migrations": len(migrations), "query_plans": len(plans), "integrity": integrity, "report": str(OUT)}))


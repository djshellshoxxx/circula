#!/usr/bin/env python3
"""Reference reader for a Circula export package (DAT-041).

Standard library only; does not need Circula. Format: docs/api/export-schema.md.

    python3 scripts/read-export.py <package.zip | directory>

Verifies size, SHA-256 and row count of every file against manifest.json, then
checks the reconciliation summary when present. Older packages without that
optional summary remain supported. Exit status 0 only when every check passes.
"""
import csv
import hashlib
import io
import json
import os
import re
import sys
import zipfile


def open_package(path):
    if os.path.isdir(path):
        def read(name):
            with open(os.path.join(path, os.path.basename(name)), "rb") as f:
                return f.read()
        return read
    z = zipfile.ZipFile(path)
    names = {os.path.basename(n): n for n in z.namelist()}

    def read(name):
        return z.read(names[name])
    return read


def integer(value, label, csv_cell=False):
    """Parse exact signed 64-bit integers; never round floats or accept bools."""
    if csv_cell and isinstance(value, str) and re.fullmatch(r"-?(0|[1-9][0-9]*)", value):
        value = int(value)
    if type(value) is not int or not -(1 << 63) <= value < (1 << 63):
        raise ValueError("%s must be an integer in the signed 64-bit range" % label)
    return value


def identifier(value, label):
    if not isinstance(value, str) or not value or any(c in value for c in ",\r\n"):
        raise ValueError("%s must be a nonempty identifier without commas or newlines" % label)
    return value


def check_reconciliation(man, tables):
    """Recompute GOV-022 from the exported rows, independent of Circula."""
    if "reconciliation" not in man:
        return
    rec = man["reconciliation"]
    if not isinstance(rec, dict):
        raise ValueError("reconciliation must be an object")
    scope = man["scope"]
    if scope not in ("mine", "ledger", "full"):
        raise ValueError("reconciliation is not valid for scope %s" % scope)
    count = integer(rec["entries"], "reconciliation.entries")
    entry_sum = integer(rec["entry_sum"], "reconciliation.entry_sum")
    balance_total = integer(rec["balance_total"], "reconciliation.balance_total")
    snapshot = integer(man["ledger_seq"], "ledger_seq")
    if count < 0 or snapshot < 0:
        raise ValueError("entry count and ledger_seq must not be negative")
    entries, csv_cells = tables["entries"]
    per_account, per_tx, seen = {}, {}, set()
    total = 0
    for row in entries:
        entry_id = identifier(row["id"], "entries.id")
        if entry_id in seen:
            raise ValueError("duplicate entry id %s" % entry_id)
        seen.add(entry_id)
        tx = identifier(row["tx_id"], "entries.tx_id")
        amount = integer(row["amount"], "entries.amount", csv_cells)
        seq = integer(row["ledger_seq"], "entries.ledger_seq", csv_cells)
        if not 1 <= seq <= snapshot:
            raise ValueError("entry %s is outside the ledger snapshot" % entry_id)
        total += amount
        # Self-exports deliberately omit account_id. Only network scopes
        # carry the account identifiers needed for per-account comparison.
        if scope != "mine":
            account = identifier(row["account_id"], "entries.account_id")
            per_account[account] = per_account.get(account, 0) + amount
        per_tx[tx] = per_tx.get(tx, 0) + amount
    if count != len(entries) or entry_sum != total:
        raise ValueError("reconciliation entry count or sum does not match exported entries")
    if scope == "mine":
        if "accounts" in rec:
            raise ValueError("mine reconciliation must not contain an accounts digest")
        if balance_total != total:
            raise ValueError("mine balance does not match exported entries")
        return
    if total != 0 or any(per_tx.values()):
        raise ValueError("network or transaction entries do not balance to zero")
    accounts, csv_cells = tables["accounts"]
    balances = {}
    for row in accounts:
        account = identifier(row["id"], "accounts.id")
        if account in balances:
            raise ValueError("duplicate account id %s" % account)
        balance = integer(row["balance"], "accounts.balance", csv_cells)
        if balance != per_account.get(account, 0):
            raise ValueError("account %s balance does not match its entries" % account)
        balances[account] = balance
    if per_account.keys() - balances.keys():
        raise ValueError("entries refer to an account missing from the export")
    if balance_total != sum(balances.values()) or balance_total != 0:
        raise ValueError("reconciliation balance total does not match exported accounts")
    digest = rec["accounts"]
    if not isinstance(digest, dict):
        raise ValueError("reconciliation.accounts must be an object")
    nonzero = integer(digest["nonzero"], "reconciliation.accounts.nonzero")
    sha = digest["sha256"]
    if not isinstance(sha, str) or not re.fullmatch(r"[0-9a-f]{64}", sha):
        raise ValueError("reconciliation.accounts.sha256 must be a lowercase SHA-256 digest")
    ids = sorted((a for a, b in per_account.items() if b), key=lambda a: a.encode("utf-8"))
    lines = "".join("%s,%d\n" % (a, per_account[a]) for a in ids).encode("utf-8")
    if nonzero != len(ids) or sha != hashlib.sha256(lines).hexdigest():
        raise ValueError("reconciliation account count or digest does not match exported entries")


def verify(argv):
    if len(argv) != 2:
        print(__doc__, file=sys.stderr)
        return 2
    read = open_package(argv[1])
    man = json.loads(read("manifest.json"))
    if man.get("schema_version") != 1:
        print("unsupported schema_version %r" % man.get("schema_version"), file=sys.stderr)
        return 1
    bad = 0
    exported = {}
    for f in man["files"]:
        data = read(f["name"])
        rows = 0
        tables = {}
        if f["name"].endswith(".csv"):
            recs = list(csv.reader(io.StringIO(data.decode("utf-8"), newline="")))
            rows = max(len(recs) - 1, 0)
            table = f["name"][:-4]
            tables[table] = rows
            if "reconciliation" in man and table in ("entries", "accounts"):
                if not recs or len(set(recs[0])) != len(recs[0]):
                    raise ValueError("%s has a missing or duplicate header" % f["name"])
                if any(len(row) != len(recs[0]) for row in recs[1:]):
                    raise ValueError("%s has inconsistent CSV row widths" % f["name"])
                if table in exported:
                    raise ValueError("duplicate exported table %s" % table)
                exported[table] = ([dict(zip(recs[0], row)) for row in recs[1:]], True)
        else:
            doc = json.loads(data)
            for t, items in doc["tables"].items():
                tables[t] = len(items)
                rows += len(items)
                if "reconciliation" in man and t in ("entries", "accounts"):
                    if t in exported:
                        raise ValueError("duplicate exported table %s" % t)
                    if not isinstance(items, list) or any(not isinstance(row, dict) for row in items):
                        raise ValueError("%s must be an array of row objects" % t)
                    exported[t] = (items, False)
        problems = []
        if len(data) != f["bytes"]:
            problems.append("size %d != %d" % (len(data), f["bytes"]))
        if hashlib.sha256(data).hexdigest() != f["sha256"]:
            problems.append("sha256 mismatch")
        if rows != f["rows"]:
            problems.append("rows %d != %d" % (rows, f["rows"]))
        for p in problems:
            print("FAIL %s: %s" % (f["name"], p))
            bad += 1
        for t, n in sorted(tables.items()):
            print("%-28s %d rows" % (t, n))
    try:
        check_reconciliation(man, exported)
    except (ValueError, KeyError, TypeError) as exc:
        print("FAIL reconciliation: %s" % exc)
        bad += 1
    if bad:
        print("export verification failed: %d problem(s)" % bad)
        return 1
    print("export ok: scope %s, ledger_seq %d, %d file(s)" % (man["scope"], man["ledger_seq"], len(man["files"])))
    return 0


def main(argv):
    try:
        return verify(argv)
    except (OSError, ValueError, KeyError, TypeError, zipfile.BadZipFile, csv.Error) as exc:
        print("FAIL export verification failed: %s" % exc, file=sys.stderr)
        return 1


if __name__ == "__main__":
    sys.exit(main(sys.argv))

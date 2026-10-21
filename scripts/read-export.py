#!/usr/bin/env python3
"""Reference reader for a Circula export package (DAT-041).

Standard library only; does not need Circula. Format: docs/api/export-schema.md.

    python3 scripts/read-export.py <package.zip | directory>

Verifies size, SHA-256 and row count of every file against manifest.json, then prints
one line per table with its row count. Exit status 0 only when every check passes.
"""
import csv
import hashlib
import io
import json
import os
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


def main(argv):
    if len(argv) != 2:
        print(__doc__, file=sys.stderr)
        return 2
    read = open_package(argv[1])
    man = json.loads(read("manifest.json"))
    if man.get("schema_version") != 1:
        print("unsupported schema_version %r" % man.get("schema_version"), file=sys.stderr)
        return 1
    bad = 0
    for f in man["files"]:
        data = read(f["name"])
        rows = 0
        tables = {}
        if f["name"].endswith(".csv"):
            recs = list(csv.reader(io.StringIO(data.decode("utf-8"), newline="")))
            rows = max(len(recs) - 1, 0)
            tables[f["name"][:-4]] = rows
        else:
            doc = json.loads(data)
            for t, items in doc["tables"].items():
                tables[t] = len(items)
                rows += len(items)
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
    if bad:
        print("export verification failed: %d problem(s)" % bad)
        return 1
    print("export ok: scope %s, ledger_seq %d, %d file(s)" % (man["scope"], man["ledger_seq"], len(man["files"])))
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))

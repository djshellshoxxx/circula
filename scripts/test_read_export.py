"""GOV-022 / QA-030 regression tests for the standalone export verifier.

Run: python3 -m unittest discover -s scripts -p 'test_read_export.py'
"""
import copy
import csv
import hashlib
import io
import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
import zipfile


READER = Path(__file__).with_name("read-export.py")
EMPTY_DIGEST = hashlib.sha256(b"").hexdigest()


def fixture(scope="ledger"):
    # Greater than 2**53: a float round-trip must not silently alter money.
    amount = 9007199254740993
    entries = [
        {"id": "e1", "tx_id": "t1", "account_id": "z", "amount": -amount, "ledger_seq": 1},
        {"id": "e2", "tx_id": "t1", "account_id": "a", "amount": amount, "ledger_seq": 1},
    ]
    accounts = [{"id": "z", "balance": -amount}, {"id": "a", "balance": amount}, {"id": "zero", "balance": 0}]
    rec = {"entries": 2, "entry_sum": 0, "balance_total": 0,
           "accounts": {"nonzero": 2, "sha256": hashlib.sha256(("a,%d\nz,%d\n" % (amount, -amount)).encode()).hexdigest()}}
    tables = {"entries": entries, "accounts": accounts}
    if scope == "mine":
        # sqlite_export.go removes account_id from the self-export projection.
        own_entry = {key: value for key, value in entries[0].items() if key != "account_id"}
        tables = {"entries": [own_entry]}
        rec = {"entries": 1, "entry_sum": -amount, "balance_total": -amount}
    elif scope in ("members", "audit"):
        tables = {scope: []}
        rec = None
    man = {"schema_version": 1, "scope": scope, "ledger_seq": 1, "files": []}
    if rec is not None:
        man["reconciliation"] = rec
    return man, tables


class ReferenceExportTests(unittest.TestCase):
    def verify(self, man, tables, fmt="json", zipped=False, corrupt=None):
        man = copy.deepcopy(man)
        man["format"] = fmt
        files = {}
        if fmt == "json":
            name = man["scope"] + ".json"
            files[name] = json.dumps({"tables": tables}).encode()
            counts = {name: sum(len(rows) for rows in tables.values())}
        else:
            counts = {}
            for table, rows in tables.items():
                buf = io.StringIO(newline="")
                columns = list(rows[0]) if rows else {"entries": ["id", "tx_id", "account_id", "amount", "ledger_seq"],
                                                     "accounts": ["id", "balance"]}.get(table, ["id"])
                if table == "entries" and man["scope"] == "mine":
                    columns = [column for column in columns if column != "account_id"]
                writer = csv.DictWriter(buf, fieldnames=columns)
                writer.writeheader()
                writer.writerows(rows)
                name = table + ".csv"
                files[name] = buf.getvalue().encode()
                counts[name] = len(rows)
        man["files"] = [{"name": name, "rows": counts[name], "bytes": len(data),
                         "sha256": hashlib.sha256(data).hexdigest()} for name, data in files.items()]
        if corrupt:
            corrupt(man, files)
        files["manifest.json"] = json.dumps(man).encode()
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp)
            if zipped:
                path /= "export.zip"
                with zipfile.ZipFile(path, "w") as package:
                    for name, data in files.items():
                        package.writestr(name, data)
            else:
                for name, data in files.items():
                    (path / name).write_bytes(data)
            result = subprocess.run([sys.executable, "-I", str(READER), str(path)],
                                    capture_output=True, text=True, timeout=10)
        self.assertNotIn("Traceback", result.stderr)
        return result.returncode, result.stdout + result.stderr

    def assert_valid(self, man, tables, **kwargs):
        code, output = self.verify(man, tables, **kwargs)
        self.assertEqual(code, 0, output)
        self.assertIn("export ok:", output)

    def assert_invalid(self, man, tables, **kwargs):
        code, output = self.verify(man, tables, **kwargs)
        self.assertEqual(code, 1, output)
        self.assertNotIn("export ok:", output)
        self.assertIn("export verification failed", output)

    def test_valid_scopes_formats_and_package_types(self):
        for scope in ("ledger", "full", "mine", "members", "audit"):
            for fmt in ("json", "csv"):
                for zipped in (False, True):
                    with self.subTest(scope=scope, fmt=fmt, zipped=zipped):
                        self.assert_valid(*fixture(scope), fmt=fmt, zipped=zipped)

    def test_old_packages_without_optional_summary(self):
        for scope in ("ledger", "full", "mine"):
            for fmt in ("json", "csv"):
                man, tables = fixture(scope)
                del man["reconciliation"]
                with self.subTest(scope=scope, fmt=fmt):
                    self.assert_valid(man, tables, fmt=fmt)

    def test_empty_ledger_and_zero_balance_accounts(self):
        for scope in ("ledger", "full", "mine"):
            man, tables = fixture(scope)
            tables["entries"] = []
            man["reconciliation"].update(entries=0, entry_sum=0, balance_total=0)
            if scope != "mine":
                tables["accounts"] = [{"id": "zero", "balance": 0}]
                man["reconciliation"]["accounts"] = {"nonzero": 0, "sha256": EMPTY_DIGEST}
            for fmt in ("json", "csv"):
                with self.subTest(scope=scope, fmt=fmt):
                    self.assert_valid(man, tables, fmt=fmt)

    def test_rejects_inconsistent_summary_fields(self):
        for field in ("entries", "entry_sum", "balance_total"):
            for fmt in ("json", "csv"):
                man, tables = fixture()
                man["reconciliation"][field] += 1
                with self.subTest(field=field, fmt=fmt):
                    self.assert_invalid(man, tables, fmt=fmt)
        for field, wrong in (("nonzero", 3), ("sha256", "0" * 64)):
            man, tables = fixture()
            man["reconciliation"]["accounts"][field] = wrong
            self.assert_invalid(man, tables)

    def test_rejects_malformed_summary(self):
        replacements = [None, [], "summary", {}, {"entries": 2},
                        {"entries": True, "entry_sum": 0, "balance_total": 0},
                        {"entries": 2, "entry_sum": 0.0, "balance_total": 0},
                        {"entries": 2, "entry_sum": 0, "balance_total": "0"}]
        for rec in replacements:
            man, tables = fixture()
            man["reconciliation"] = rec
            with self.subTest(rec=rec):
                self.assert_invalid(man, tables)
        for digest in (None, [], {}, {"nonzero": 2, "sha256": 123},
                       {"nonzero": True, "sha256": EMPTY_DIGEST}):
            man, tables = fixture()
            man["reconciliation"]["accounts"] = digest
            with self.subTest(digest=digest):
                self.assert_invalid(man, tables)

    def test_rejects_ledger_rows_with_valid_file_checksums(self):
        mutations = {
            "wrong account balance": lambda t: t["accounts"][0].update(balance=-5),
            "unknown account": lambda t: t["entries"][0].update(account_id="missing"),
            "duplicate account": lambda t: t["accounts"].append(t["accounts"][0]),
            "duplicate entry": lambda t: t["entries"][1].update(id="e1"),
            "future entry": lambda t: t["entries"][0].update(ledger_seq=2),
            "invalid entry sequence": lambda t: t["entries"][0].update(ledger_seq=0),
            "unbalanced transaction": lambda t: t["entries"][0].update(tx_id="other"),
            "missing accounts": lambda t: t.pop("accounts"),
            "missing entries": lambda t: t.pop("entries"),
            "bad amount": lambda t: t["entries"][0].update(amount=0.5),
            "oversize amount": lambda t: t["entries"][0].update(amount=1 << 63),
        }
        for name, mutate in mutations.items():
            for fmt in ("json", "csv"):
                man, tables = fixture()
                mutate(tables)
                with self.subTest(case=name, fmt=fmt):
                    self.assert_invalid(man, tables, fmt=fmt)

    def test_mine_summary_rules(self):
        man, tables = fixture("mine")
        man["reconciliation"]["balance_total"] += 1
        self.assert_invalid(man, tables)
        for digest in (None, {"nonzero": 1, "sha256": EMPTY_DIGEST}):
            man, tables = fixture("mine")
            man["reconciliation"]["accounts"] = digest
            self.assert_invalid(man, tables)
        for fmt in ("json", "csv"):
            man, tables = fixture("mine")
            self.assertNotIn("account_id", tables["entries"][0])
            self.assertNotIn("accounts", tables)
            self.assert_valid(man, tables, fmt=fmt)
            tables["entries"][0]["amount"] += 1
            self.assert_invalid(man, tables, fmt=fmt)

    def test_unrelated_scopes_reject_summary(self):
        for scope in ("members", "audit", "unknown"):
            man, tables = fixture()
            man["scope"] = scope
            self.assert_invalid(man, tables)

    def test_malformed_csv_reports_failure_without_traceback(self):
        def corrupt(man, files):
            files["entries.csv"] += b"x\n"
        code, output = self.verify(*fixture(), fmt="csv", corrupt=corrupt)
        self.assertEqual(code, 1, output)
        self.assertIn("FAIL", output)

    def test_integrity_checks_remain_enforced(self):
        for field, wrong in (("rows", 99), ("bytes", 99), ("sha256", "0" * 64)):
            def corrupt(man, files):
                man["files"][0][field] = wrong
            with self.subTest(field=field):
                self.assert_invalid(*fixture(), corrupt=corrupt)


if __name__ == "__main__":
    unittest.main()

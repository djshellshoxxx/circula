"""GOV-022 / QA-030 regression tests for the standalone export verifier.

Run: python3 -m unittest discover -s scripts -p 'test_read_export.py'
"""
import copy
import csv
import hashlib
import importlib.util
import io
import json
from pathlib import Path
import subprocess
import struct
import sys
import tempfile
import unittest
import warnings
import zipfile
import zlib


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
    def verify(self, man, tables, fmt="json", zipped=False, corrupt=None,
               extra_zip_members=None, transform_package=None):
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
                with warnings.catch_warnings():
                    warnings.simplefilter("ignore", UserWarning)
                    with zipfile.ZipFile(path, "w") as package:
                        for name, data in extra_zip_members or []:
                            package.writestr(name, data)
                        for name, data in files.items():
                            package.writestr(name, data)
            else:
                for name, data in files.items():
                    (path / name).write_bytes(data)
            if transform_package:
                transform_package(path)
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

    def test_rejects_duplicate_zip_members(self):
        for name, data in (("members.json", b'{"tables": {"members": []}}'),
                           ("members.json", b"conflicting payload"),
                           ("manifest.json", b"{}")):
            with self.subTest(name=name, data=data):
                self.assert_invalid(*fixture("members"), zipped=True,
                                    extra_zip_members=[(name, data)])

    def test_rejects_zip_path_aliases(self):
        for name in ("nested/members.json", "nested/manifest.json",
                     "../members.json", "nested\\members.json", "C:members.json"):
            with self.subTest(name=name):
                self.assert_invalid(*fixture("members"), zipped=True,
                                    extra_zip_members=[(name, b"unlisted")])

    def test_rejects_unsafe_manifest_names(self):
        for name in ("../members.json", "nested/members.json", "nested\\members.json",
                     "C:members.json", "/members.json", "", ".", "..", "manifest.json"):
            def corrupt(man, files):
                man["files"][0]["name"] = name
            for zipped in (False, True):
                with self.subTest(name=name, zipped=zipped):
                    self.assert_invalid(*fixture("members"), zipped=zipped, corrupt=corrupt)

    def test_rejects_duplicate_manifest_declarations(self):
        def corrupt(man, files):
            man["files"].append(copy.deepcopy(man["files"][0]))
        for zipped in (False, True):
            with self.subTest(zipped=zipped):
                self.assert_invalid(*fixture("members"), zipped=zipped, corrupt=corrupt)

    def test_rejects_unlisted_package_members(self):
        def corrupt(man, files):
            files["unlisted.csv"] = b"id\nextra\n"
        for zipped in (False, True):
            with self.subTest(zipped=zipped):
                self.assert_invalid(*fixture("members"), zipped=zipped, corrupt=corrupt)

    def test_rejects_nonregular_zip_members(self):
        for name, mode in (("notes/", 0o40755), ("outside", 0o120777), ("pipe", 0o10600)):
            info = zipfile.ZipInfo(name)
            info.create_system = 3
            info.external_attr = mode << 16
            with self.subTest(name=name):
                self.assert_invalid(*fixture("members"), zipped=True,
                                    extra_zip_members=[(info, b"target")])

    def test_malformed_manifest_is_reported_without_traceback(self):
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp)
            for value in (None, [], "manifest"):
                (path / "manifest.json").write_text(json.dumps(value))
                result = subprocess.run([sys.executable, "-I", str(READER), str(path)],
                                        capture_output=True, text=True, timeout=10)
                with self.subTest(value=value):
                    self.assertEqual(result.returncode, 1)
                    self.assertIn("export verification failed", result.stderr)
                    self.assertNotIn("Traceback", result.stderr)

    def test_rejects_raw_nul_zip_names_before_zipfile_normalization(self):
        for target in ("manifest.json", "members.json"):
            def transform(path):
                with zipfile.ZipFile(path) as package:
                    entries = [(info.filename, package.read(info)) for info in package.infolist()]
                raw = io.BytesIO()
                with zipfile.ZipFile(raw, "w") as package:
                    for name, data in entries:
                        package.writestr(name + "XA" if name == target else name, data)
                # Patch both local and central names without changing lengths.
                # ZipInfo.filename silently truncates these raw names at NUL.
                path.write_bytes(raw.getvalue().replace((target + "XA").encode(),
                                                        target.encode() + b"\x00A"))
            with self.subTest(target=target):
                self.assert_invalid(*fixture("members"), zipped=True, transform_package=transform)

    def test_dos_zip_file_types(self):
        for creator in (0, 3, 10, 14, 19):
            for directory in (False, True):
                def transform(path):
                    with zipfile.ZipFile(path) as package:
                        entries = [(info.filename, package.read(info)) for info in package.infolist()]
                    with zipfile.ZipFile(path, "w") as package:
                        for name, data in entries:
                            info = zipfile.ZipInfo(name)
                            info.create_system = creator
                            info.external_attr = 0x10 if directory else 0x20
                            package.writestr(info, data)
                with self.subTest(creator=creator, directory=directory):
                    check = self.assert_invalid if directory else self.assert_valid
                    check(*fixture("members"), zipped=True, transform_package=transform)

    def test_schema_version_requires_an_integer(self):
        for version in (True, 1.0, "1", None, 0, 2):
            man, tables = fixture("members")
            man["schema_version"] = version
            code, output = self.verify(man, tables)
            with self.subTest(version=version):
                self.assertEqual(code, 1, output)
                self.assertNotIn("export ok:", output)

    def test_member_budget_rejects_unchecked_suffixes(self):
        spec = importlib.util.spec_from_file_location("export_reader_budget", READER)
        reader = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(reader)
        # Exercise the production limit with a tiny budget; no giant allocation.
        reader.MAX_EXPORT_FILE = 8
        self.assertEqual(reader.read_member(io.BytesIO(b"12345678")), b"12345678")
        with self.assertRaisesRegex(ValueError, "too large"):
            reader.read_member(io.BytesIO(b"12345678unchecked"))
        for zipped in (False, True):
            with tempfile.TemporaryDirectory() as tmp:
                path = Path(tmp)
                if zipped:
                    path /= "export.zip"
                    with zipfile.ZipFile(path, "w") as package:
                        package.writestr("members.csv", b"123456789")
                else:
                    (path / "members.csv").write_bytes(b"123456789")
                with self.subTest(zipped=zipped):
                    with reader.open_package(path) as (read, names):
                        self.assertEqual(names, {"members.csv"})
                        with self.assertRaisesRegex(ValueError, "too large"):
                            read("members.csv")

    def test_zip_sizes_match_the_complete_member(self):
        for compression in (zipfile.ZIP_STORED, zipfile.ZIP_DEFLATED):
            for mismatch in ("understated_prefix_crc", "understated_full_crc", "overstated"):
                def transform(path):
                    with zipfile.ZipFile(path) as package:
                        files = {info.filename: package.read(info) for info in package.infolist()}
                    prefix = files["members.json"]
                    payload = prefix if mismatch == "overstated" else prefix + b" \n " * 8
                    files["members.json"] = payload
                    if mismatch == "understated_full_crc":
                        man = json.loads(files["manifest.json"])
                        man["files"][0]["bytes"] = len(payload)
                        man["files"][0]["sha256"] = hashlib.sha256(payload).hexdigest()
                        files["manifest.json"] = json.dumps(man).encode()
                    raw = io.BytesIO()
                    with zipfile.ZipFile(raw, "w", compression=compression) as package:
                        for name, data in files.items():
                            package.writestr(name, data)
                    with zipfile.ZipFile(io.BytesIO(raw.getvalue())) as package:
                        local = package.getinfo("members.json").header_offset
                        central = package.start_dir
                    data = bytearray(raw.getvalue())
                    self.assertEqual(data[central:central + 4], b"PK\x01\x02")
                    name_size = struct.unpack_from("<H", data, central + 28)[0]
                    self.assertEqual(data[central + 46:central + 46 + name_size], b"members.json")
                    declared = len(prefix) + 1 if mismatch == "overstated" else len(prefix)
                    crc_data = prefix if mismatch == "understated_prefix_crc" else payload
                    crc = zlib.crc32(crc_data)
                    # Preserve the complete physical/compressed member, but
                    # falsify its declared uncompressed length in both headers.
                    struct.pack_into("<I", data, local + 14, crc)
                    struct.pack_into("<I", data, local + 22, declared)
                    struct.pack_into("<I", data, central + 16, crc)
                    struct.pack_into("<I", data, central + 24, declared)
                    path.write_bytes(data)
                with self.subTest(compression=compression, mismatch=mismatch):
                    self.assert_invalid(*fixture("members"), zipped=True, transform_package=transform)


if __name__ == "__main__":
    unittest.main()

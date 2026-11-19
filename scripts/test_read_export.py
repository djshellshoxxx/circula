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
from unittest import mock
import warnings
import zipfile
import zlib


READER = Path(__file__).with_name("read-export.py")
EMPTY_DIGEST = hashlib.sha256(b"").hexdigest()


def duplicate_manifest_field(data, name, earlier, alias=None):
    """Insert a real duplicate member before its existing final value."""
    marker = (json.dumps(name) + ": ").encode()
    if marker not in data:
        raise AssertionError("manifest field %s is missing" % name)
    earlier_marker = (json.dumps(alias or name) + ": ").encode()
    return data.replace(marker, earlier_marker + json.dumps(earlier).encode() + b", " + marker, 1)


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
               extra_zip_members=None, transform_package=None, transform_manifest=None):
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
        manifest = json.dumps(man).encode()
        files["manifest.json"] = transform_manifest(manifest) if transform_manifest else manifest
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

    def test_manifest_file_counts_require_integer_tokens(self):
        for field, convert in (("rows", bool), ("rows", float), ("bytes", float)):
            for fmt in ("json", "csv"):
                def corrupt(man, files):
                    entry = man["files"][0]
                    entry[field] = convert(entry[field])
                with self.subTest(field=field, convert=convert.__name__, fmt=fmt):
                    self.assert_invalid(*fixture("members"), fmt=fmt, corrupt=corrupt)
        man, tables = fixture("members")
        tables["members"] = [{"id": "member-1"}]
        self.assert_invalid(man, tables, corrupt=lambda m, f: m["files"][0].update(rows=True))

    def test_manifest_boolean_size_does_not_match_one_byte(self):
        def corrupt(man, files):
            entry = man["files"][0]
            files[entry["name"]] = b"x"  # a one-field CSV header; zero data rows
            entry.update(bytes=True, sha256=hashlib.sha256(b"x").hexdigest())
        self.assert_invalid(*fixture("members"), fmt="csv", zipped=True, corrupt=corrupt)

    def test_manifest_metadata_rejects_wrong_types(self):
        cases = (("ledger_seq", False), ("ledger_seq", 1.0), ("ledger_seq", 1 << 63),
                 ("decimals", True), ("decimals", 2.0), ("decimals", 1 << 63),
                 ("scope", False), ("format", 7), ("network_id", []), ("generated_at", 0))
        for name, value in cases:
            with self.subTest(name=name, value=value):
                self.assert_invalid(*fixture("members"),
                                    corrupt=lambda m, f: m.update({name: value}))

    def test_manifest_timestamps_reject_malformed_values(self):
        for value in ("not-a-time", "2026-02-29T00:00:00Z", "2026-13-01T00:00:00Z",
                      "2026-11-19T24:00:00Z", "2026-11-19T00:00:60Z",
                      "2026-11-19T00:00:00", "2026-11-19 00:00:00Z",
                      "2026-11-19T00:00:00+25:00", "2026-11-19T00:00:00+00:61",
                      "２０２６-11-19T00:00:00Z"):
            with self.subTest(value=value):
                self.assert_invalid(*fixture("members"),
                                    corrupt=lambda m, f: m.update(generated_at=value))

    def test_manifest_metadata_preserves_supported_values(self):
        for value in ("2026-11-19T12:34:56Z", "2026-11-19T12:34:56.123456789Z",
                      "2026-11-19T12:34:56+05:30", "2026-11-19T12:34:56-03:30",
                      "0000-02-29T00:00:00Z",
                      # Go's time.Time decoder also accepts these legacy spellings.
                      "2026-11-19T1:02:03,123+24:60"):
            man, tables = fixture("members")
            man.update(network_id="réseau-雪", decimals=2, generated_at=value)
            tables["members"] = [{"id": "member-1", "display_name": "Élodie 雪"}]
            with self.subTest(value=value):
                self.assert_valid(man, tables, zipped=True)

    def test_manifest_preserves_existing_missing_and_null_behavior(self):
        # These optional metadata fields were not required by the reader, and
        # Go's decoder accepts null for its scalar fields. Do not add presence
        # requirements while repairing type errors.
        self.assert_valid(*fixture("members"))
        self.assert_valid(*fixture("members"), corrupt=lambda m, f: m.update(
            network_id=None, decimals=None, generated_at=None, format=None, scope=None))
        # These nulls already failed the reader's final checks.
        for field in ("ledger_seq", "files"):
            with self.subTest(field=field):
                self.assert_invalid(*fixture("members"), corrupt=lambda m, f: m.update({field: None}))
        for field in ("rows", "bytes"):
            with self.subTest(file_field=field):
                self.assert_invalid(*fixture("members"),
                                    corrupt=lambda m, f: m["files"][0].update({field: None}))

    def test_manifest_rejects_shadowed_type_errors(self):
        cases = (("schema_version", True), ("rows", False), ("rows", 1 << 63),
                 ("bytes", 0.0), ("sha256", 7), ("files", False), ("files", [True]),
                 ("scope", False), ("format", 7), ("network_id", []), ("decimals", False),
                 ("ledger_seq", 1.0), ("generated_at", 0), ("generated_at", "not-a-time"),
                 ("reconciliation", False), ("entries", True), ("entry_sum", 0.0),
                 ("balance_total", "0"), ("accounts", []), ("nonzero", True))
        for name, earlier in cases:
            man, tables = fixture()
            man.update(network_id="network", decimals=2, generated_at="2026-11-19T00:00:00Z")
            with self.subTest(name=name, earlier=earlier):
                self.assert_invalid(man, tables, transform_manifest=lambda data: duplicate_manifest_field(
                    data, name, earlier))

    def test_manifest_valid_duplicates_keep_final_values(self):
        cases = (("schema_version", 2), ("schema_version", None), ("rows", -1),
                 ("bytes", 99), ("sha256", "earlier string"), ("files", None),
                 ("generated_at", "2025-01-01T00:00:00+01:00"),
                 ("reconciliation", None), ("entries", 99), ("accounts", None), ("nonzero", 99))
        for name, earlier in cases:
            man, tables = fixture()
            man["generated_at"] = "2026-11-19T00:00:00Z"
            with self.subTest(name=name, earlier=earlier):
                self.assert_valid(man, tables, transform_manifest=lambda data: duplicate_manifest_field(
                    data, name, earlier))

    def test_manifest_unknown_fields_remain_ignored(self):
        man, tables = fixture()
        man["extension"] = {"rows": False, "schema_version": True, "files": 7,
                            "generated_at": "not-a-time", "reconciliation": {"entries": False}}
        man["reconciliation"]["extension"] = {"nonzero": False, "sha256": 7}
        man["reconciliation"]["accounts"]["extension"] = {"ledger_seq": False}
        def corrupt(man, files):
            man["files"][0]["extension"] = {"rows": False, "bytes": 0.0}
        self.assert_valid(man, tables, corrupt=corrupt,
                          transform_manifest=lambda data: duplicate_manifest_field(data, "extension", False))

    def test_manifest_case_aliases_reject_shadowed_type_errors(self):
        cases = (("schema_version", "SCHEMA_VERSION", True),
                 ("ledger_seq", "LEDGER_SEQ", False), ("scope", "ſCOPE", False),
                 ("network_id", "networK_id", []), ("decimals", "DECIMALS", False),
                 ("format", "FORMAT", 7), ("generated_at", "GENERATED_AT", False),
                 ("files", "FILES", False), ("name", "NAME", 7),
                 ("rows", "ROWſ", False), ("bytes", "BYTES", 0.0),
                 ("sha256", "SHA256", False), ("reconciliation", "RECONCILIATION", False),
                 ("entries", "ENTRIES", True), ("entry_sum", "ENTRY_SUM", 0.0),
                 ("balance_total", "BALANCE_TOTAL", "0"),
                 ("accounts", "ACCOUNTS", []), ("nonzero", "NONZERO", True))
        for name, alias, earlier in cases:
            man, tables = fixture()
            man.update(network_id="network", decimals=2, generated_at="2026-11-19T00:00:00Z")
            with self.subTest(name=name, alias=alias):
                self.assert_invalid(man, tables, zipped=True,
                                    transform_manifest=lambda data: duplicate_manifest_field(
                                        data, name, earlier, alias))

    def test_manifest_case_aliases_apply_later_values(self):
        cases = (("manifest", "LEDGER_SEQ", 0),
                 ("manifest", "ſCOPE", "mine"), ("manifest", "FILES", []),
                 ("file", "ROWS", 1), ("file", "BYTES", 0),
                 ("file", "SHA256", "0" * 64), ("file", "NAME", "missing.json"),
                 ("reconciliation", "ENTRIES", 0), ("reconciliation", "ENTRY_SUM", 1),
                 ("reconciliation", "BALANCE_TOTAL", 1),
                 ("accounts", "NONZERO", 0), ("accounts", "SHA256", "0" * 64))
        for target, alias, later in cases:
            def corrupt(man, files):
                objects = {"manifest": man, "file": man["files"][0],
                           "reconciliation": man["reconciliation"],
                           "accounts": man["reconciliation"]["accounts"]}
                objects[target][alias] = later
            with self.subTest(target=target, alias=alias):
                self.assert_invalid(*fixture(), corrupt=corrupt)
        # Unsupported versions have their own existing diagnostic.
        code, output = self.verify(*fixture(), corrupt=lambda m, f: m.update(SCHEMA_VERSION=2))
        self.assertEqual(code, 1, output)
        self.assertIn("unsupported schema_version 2", output)
        self.assertNotIn("export ok:", output)

    def test_manifest_case_aliases_accept_valid_names(self):
        aliases = {"schema_version": "SCHEMA_VERSION", "scope": "ſCOPE",
                   "ledger_seq": "LEDGER_SEQ", "network_id": "networK_id",
                   "decimals": "DECIMALS", "generated_at": "GENERATED_AT",
                   "format": "FORMAT", "files": "FILES", "name": "NAME",
                   "rows": "ROWſ", "bytes": "BYTES", "sha256": "SHA256",
                   "reconciliation": "RECONCILIATION", "entries": "ENTRIES",
                   "entry_sum": "ENTRY_SUM", "balance_total": "BALANCE_TOTAL",
                   "accounts": "ACCOUNTS", "nonzero": "NONZERO"}
        def transform(data):
            for name, alias in aliases.items():
                data = data.replace((json.dumps(name) + ":").encode(),
                                    (json.dumps(alias) + ":").encode())
            return data
        man, tables = fixture()
        man.update(network_id="réseau-雪", decimals=2, generated_at="2026-11-19T12:34:56+05:30")
        for fmt in ("json", "csv"):
            with self.subTest(fmt=fmt):
                self.assert_valid(man, tables, fmt=fmt, zipped=True, transform_manifest=transform)

    def test_manifest_case_aliases_preserve_null_rules(self):
        # Go ignores scalar nulls after an existing value, including when the
        # later spelling is an alias. Slice/pointer nulls do clear the field.
        def scalar_nulls(man, files):
            objects = (man, man["files"][0], man["reconciliation"],
                       man["reconciliation"]["accounts"])
            for obj in objects:
                for name, value in list(obj.items()):
                    if type(value) in (str, int):
                        obj[name.upper()] = None
        man, tables = fixture()
        man.update(network_id="network", decimals=2, generated_at="2026-11-19T00:00:00Z")
        self.assert_valid(man, tables, corrupt=scalar_nulls)
        for name in ("FILES", "RECONCILIATION", "ACCOUNTS"):
            def corrupt(man, files):
                target = man["reconciliation"] if name == "ACCOUNTS" else man
                target[name] = None
            with self.subTest(name=name):
                self.assert_invalid(*fixture(), corrupt=corrupt)

    def test_manifest_case_aliases_do_not_fold_ligatures(self):
        # Python full casefold/upper would treat some of these as ASCII keys;
        # Go's simple-fold matching leaves all three as unknown extensions.
        for alias in ("ﬁles", "fıles", "fİles"):
            with self.subTest(alias=alias):
                self.assert_valid(*fixture("members"),
                                  transform_manifest=lambda data: duplicate_manifest_field(
                                      data, "files", False, alias))

    def test_json_constants_rejected_in_manifest(self):
        for value in (float("nan"), float("inf"), float("-inf")):
            for zipped in (False, True):
                with self.subTest(value=value, zipped=zipped):
                    self.assert_invalid(*fixture("members"), zipped=zipped,
                                        corrupt=lambda m, f: m.update(extension={"nested": [value]}))
        # Valid JSON extension values and strings with those names remain valid.
        self.assert_valid(*fixture("members"), corrupt=lambda m, f: m.update(
            extension={"names": ["NaN", "Infinity", "-Infinity"], "finite": 1.5, "optional": None}))

    def test_json_constants_rejected_in_members(self):
        for value in (float("nan"), float("inf"), float("-inf")):
            for nested_row in (False, True):
                def corrupt(man, files):
                    entry = man["files"][0]
                    document = json.loads(files[entry["name"]])
                    target = document["tables"]["members"][0] if nested_row else document
                    target["extension"] = {"nested": [value]}
                    data = json.dumps(document).encode()
                    files[entry["name"]] = data
                    entry.update(bytes=len(data), sha256=hashlib.sha256(data).hexdigest())
                man, tables = fixture("members")
                tables["members"] = [{"id": "member-1"}]
                with self.subTest(value=value, nested_row=nested_row):
                    self.assert_invalid(man, tables, zipped=True, corrupt=corrupt)
        man, tables = fixture("members")
        tables["members"] = [{"id": "member-1", "extension": ["NaN", "Infinity", "-Infinity", 1.5, None]}]
        self.assert_valid(man, tables, zipped=True)

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

    def recompress_package(self, path, compression, incomplete=None):
        """Keep every data byte/checksum while removing stream completion."""
        with zipfile.ZipFile(path) as package:
            files = [(info.filename, package.read(info)) for info in package.infolist()]
        try:
            rewritten = zipfile.ZipFile(path, "w", compression=compression)
        except RuntimeError as exc:
            self.skipTest("stdlib compression method %d unavailable: %s" % (compression, exc))
        with rewritten as package:
            for name, data in files:
                package.writestr(name, data)
        if incomplete is None:
            return
        with zipfile.ZipFile(path) as package:
            info = package.getinfo(incomplete)
            central = package.start_dir
        raw = bytearray(path.read_bytes())
        local = info.header_offset
        name_size, extra_size = struct.unpack_from("<HH", raw, local + 26)
        start = local + 30 + name_size + extra_size
        if compression == zipfile.ZIP_DEFLATED:
            # This small fixture has one final block. Clear BFINAL: its data
            # and CRC stay intact, but a further block is now required.
            self.assertEqual(raw[start] & 1, 1)
            raw[start] &= 0xfe
        else:
            # Remove the final compressed byte from this member's declared
            # extent. Its old byte becomes padding, preserving other offsets.
            while True:
                self.assertEqual(raw[central:central + 4], b"PK\x01\x02")
                name_size, extra_size, comment_size = struct.unpack_from("<HHH", raw, central + 28)
                name = raw[central + 46:central + 46 + name_size].decode()
                if name == incomplete:
                    break
                central += 46 + name_size + extra_size + comment_size
            struct.pack_into("<I", raw, local + 18, info.compress_size - 1)
            struct.pack_into("<I", raw, central + 20, info.compress_size - 1)
        path.write_bytes(raw)

    def test_zip_compression_valid_streams(self):
        for compression in (zipfile.ZIP_STORED, zipfile.ZIP_DEFLATED, zipfile.ZIP_BZIP2, zipfile.ZIP_LZMA):
            with self.subTest(compression=compression):
                def transform(path):
                    self.recompress_package(path, compression)
                self.assert_valid(*fixture("members"), zipped=True, transform_package=transform)

    def test_zip_compression_requires_complete_streams(self):
        for compression in (zipfile.ZIP_DEFLATED, zipfile.ZIP_BZIP2, zipfile.ZIP_LZMA):
            for target in ("members.json", "manifest.json"):
                with self.subTest(compression=compression, target=target):
                    def transform(path):
                        self.recompress_package(path, compression, target)
                    self.assert_invalid(*fixture("members"), zipped=True, transform_package=transform)

    def test_zip_compression_completion_state_must_be_available(self):
        spec = importlib.util.spec_from_file_location("export_reader_completion", READER)
        reader = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(reader)

        # Simulate a stdlib reader that still returns valid bytes and checks
        # CRC, but no longer exposes its decompressor's completion state.
        class CompletionBlindReader:
            def __init__(self, stream):
                self.stream = stream

            def read(self, size=-1):
                return self.stream.read(size)

            def __enter__(self):
                return self

            def __exit__(self, *exc):
                self.stream.close()

        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "complete.zip"
            with zipfile.ZipFile(path, "w", compression=zipfile.ZIP_DEFLATED) as package:
                package.writestr("members.json", b'{"tables":{"members":[]}}')
            real_open = zipfile.ZipFile.open

            def open_without_completion(package, *args, **kwargs):
                return CompletionBlindReader(real_open(package, *args, **kwargs))

            with mock.patch.object(zipfile.ZipFile, "open", open_without_completion):
                with reader.open_package(path) as (read, _):
                    with self.assertRaisesRegex(ValueError, "completion state"):
                        read("members.json")


if __name__ == "__main__":
    unittest.main()

#!/usr/bin/env python3
"""Reference reader for a Circula export package (DAT-041).

Standard library only; does not need Circula. Format: docs/api/export-schema.md.

    python3 scripts/read-export.py <package.zip | directory>

Verifies size, SHA-256 and row count of every file against manifest.json, then
checks the reconciliation summary when present. Older packages without that
optional summary remain supported. Exit status 0 only when every check passes.
"""
import calendar
import copy
import csv
from contextlib import contextmanager
import hashlib
import io
import json
import os
import re
import stat
import sys
import zipfile


MAX_EXPORT_FILE = 1 << 32


def read_member(stream):
    """Reject oversized data instead of accepting an unchecked prefix."""
    with io.BytesIO() as data:
        while True:
            chunk = stream.read(min(1 << 20, MAX_EXPORT_FILE + 1 - data.tell()))
            if not chunk:
                return data.getvalue()
            data.write(chunk)
            if data.tell() > MAX_EXPORT_FILE:
                raise ValueError("export member is too large")


def package_name(name):
    """Use one flat, portable identity for each member; never normalize paths."""
    if (not isinstance(name, str) or name in ("", ".", "..")
            or any(c in name for c in "/\\\x00:")):
        raise ValueError("unsafe package member name %r" % name)
    return name


@contextmanager
def open_package(path):
    if os.path.isdir(path):
        names = set()
        with os.scandir(path) as entries:
            for entry in entries:
                name = package_name(entry.name)
                if not entry.is_file(follow_symlinks=False):
                    raise ValueError("package member %r is not a regular file" % name)
                names.add(name)

        def read(name):
            if name not in names:
                raise ValueError("package member %r is missing" % name)
            with open(os.path.join(path, name), "rb") as f:
                if os.fstat(f.fileno()).st_size > MAX_EXPORT_FILE:
                    raise ValueError("export member is too large")
                return read_member(f)

        yield read, names
        return
    with zipfile.ZipFile(path) as z:
        members = {}
        for info in z.infolist():
            name = package_name(info.orig_filename)
            if name != info.filename:
                raise ValueError("normalized ZIP member name %r" % name)
            if name in members:
                raise ValueError("duplicate ZIP member %r" % name)
            mode = stat.S_IFMT(info.external_attr >> 16)
            dos_directory = info.external_attr & 0x10
            if info.is_dir() or dos_directory or mode not in (0, stat.S_IFREG):
                raise ValueError("package member %r is not a regular file" % name)
            members[name] = info

        def read(name):
            info = members[name]
            if info.file_size > MAX_EXPORT_FILE:
                raise ValueError("export member is too large")
            # ZipExtFile otherwise truncates at the declared uncompressed size
            # before callers can notice an understated length. Read through a
            # bounded copy so CRC covers the complete physical member, then
            # compare its actual length with the original declaration.
            complete = copy.copy(info)
            complete.file_size = MAX_EXPORT_FILE + 1
            with z.open(complete) as f:
                data = read_member(f)
                if info.compress_type != zipfile.ZIP_STORED:
                    # ZipExtFile can stop at the compressed byte count before
                    # its decoder reaches the stream end. Its public API has
                    # no completion signal, so this small adapter deliberately
                    # uses private state. Fail closed if Python changes it;
                    # valid bytes and CRC alone do not prove stream completion.
                    decoder = getattr(f, "_decompressor", None)
                    if getattr(decoder, "eof", None) is not True:
                        raise ValueError("ZIP member compression stream is incomplete or completion state is unavailable")
            if len(data) != info.file_size:
                raise ValueError("ZIP member size does not match its complete contents")
            return data

        yield read, set(members)


def integer(value, label, csv_cell=False):
    """Parse exact signed 64-bit integers; never round floats or accept bools."""
    if csv_cell and isinstance(value, str) and re.fullmatch(r"-?(0|[1-9][0-9]*)", value):
        value = int(value)
    if type(value) is not int or not -(1 << 63) <= value < (1 << 63):
        raise ValueError("%s must be an integer in the signed 64-bit range" % label)
    return value


class ManifestObject(dict):
    """Keep every occurrence until known fields are validated and resolved."""
    def __init__(self, pairs):
        super().__init__(pairs)
        self.pairs = pairs


MANIFEST_FIELDS = {
    "manifest": (("schema_version", "decimals", "ledger_seq"), ("network_id", "scope", "format")),
    "file": (("rows", "bytes"), ("name", "sha256")),
    "reconciliation": (("entries", "entry_sum", "balance_total"), ()),
    "accounts": (("nonzero",), ("sha256",)),
}

# All known tags are ASCII. These are the only extra Unicode equivalents in
# Go's simple-fold comparison of ASCII letters; full casefold would also
# expand ligatures and incorrectly turn unknown extensions into known fields.
MANIFEST_FIELD_FOLD = str.maketrans("ABCDEFGHIJKLMNOPQRSTUVWXYZſK", "abcdefghijklmnopqrstuvwxyzsk")
MANIFEST_NULLABLE_FIELDS = {"manifest": ("files", "reconciliation"), "reconciliation": ("accounts",)}


def reject_json_constant(value):
    """Go rejects NaN and infinities even in otherwise ignored JSON fields."""
    raise ValueError("invalid JSON constant %s" % value)


def manifest_timestamp(value, label):
    """Validate time.Time's RFC 3339 input without changing its precision."""
    if not isinstance(value, str):
        raise ValueError("%s must be a timestamp string" % label)
    # Keep Go's existing legacy allowances: a one-digit hour, comma fraction,
    # and timezone components of 24 hours or 60 minutes. Its strict RFC 3339
    # checks are currently disabled; do not narrow accepted exports here.
    match = re.fullmatch(r"([0-9]{4})-([0-9]{2})-([0-9]{2})T"
                         r"([0-9]{1,2}):([0-9]{2}):([0-9]{2})"
                         r"(?:[.,][0-9]+)?(?:Z|[+-]([0-9]{2}):([0-9]{2}))", value)
    if match:
        year, month, day, hour, minute, second = map(int, match.groups()[:6])
        zone_hour, zone_minute = match.groups()[6:]
        if (1 <= month <= 12 and 1 <= day <= calendar.monthrange(year, month)[1]
                and hour < 24 and minute < 60 and second < 60
                and (zone_hour is None or (int(zone_hour) <= 24 and int(zone_minute) <= 60))):
            return
    raise ValueError("%s must be an RFC 3339 timestamp" % label)


def validate_manifest_object(value, kind="manifest", label="manifest.json", previous=None):
    """Validate Go fields in member order, retaining reused nested struct state."""
    # Go accepts null while decoding these scalar, struct, slice and pointer
    # fields. Preserve that at this stage; existing final version/files/count/
    # reconciliation checks still decide whether the resulting package is valid.
    if value is None:
        return
    if not isinstance(value, ManifestObject):
        raise ValueError("%s must be an object" % label)
    integer_fields, string_fields = MANIFEST_FIELDS[kind]
    nullable_fields = MANIFEST_NULLABLE_FIELDS.get(kind, ())
    resolved = {}
    # A repeated non-null reconciliation/accounts object decodes into Go's
    # existing struct pointer. Retain its known fields, including nested
    # pointers, but do not repeatedly copy ignored extension data. A null
    # pointer is stored as None below and therefore starts fresh next time.
    if previous is not None:
        for name in integer_fields + string_fields + nullable_fields:
            if name in previous:
                resolved[name] = previous[name]
    # Go reuses the backing elements of a nonempty []ManifestFile across
    # duplicate fields. Keep populated slots after a temporary shrink; an
    # empty or null whole array resets that state. Only known fields survive.
    file_slots = []
    for original, item in value.pairs:
        name = original.translate(MANIFEST_FIELD_FOLD)
        field = "%s.%s" % (label, original)
        if name in integer_fields:
            if item is not None:
                integer(item, field)
        elif name in string_fields:
            if item is not None and not isinstance(item, str):
                raise ValueError("%s must be a string" % field)
        elif kind == "manifest" and name == "generated_at":
            if item is not None:
                manifest_timestamp(item, field)
        elif kind == "manifest" and name == "files":
            if item is None:
                file_slots = []
            else:
                if not isinstance(item, list):
                    raise ValueError("%s must be an array" % field)
                if not item:
                    file_slots = []
                for index, entry in enumerate(item):
                    prior = file_slots[index] if index < len(file_slots) else {}
                    if entry is None:
                        # Null into a struct leaves that element unchanged.
                        item[index] = dict(prior)
                    else:
                        validate_manifest_object(entry, "file", "%s[%d]" % (field, index), prior)
                    state = {key: item[index][key] for group in MANIFEST_FIELDS["file"]
                             for key in group if key in item[index]}
                    if index < len(file_slots):
                        file_slots[index] = state
                    else:
                        file_slots.append(state)
        elif kind == "manifest" and name == "reconciliation":
            validate_manifest_object(item, "reconciliation", field, resolved.get(name))
        elif kind == "reconciliation" and name == "accounts":
            validate_manifest_object(item, "accounts", field, resolved.get(name))
        else:
            # Unknown additive fields stay ignored, including nested values.
            resolved[original] = item
            continue
        # Scalar nulls leave an already decoded value unchanged in Go. Keep a
        # lone null explicit for the reader's existing final presence checks;
        # slice/pointer nulls always replace an earlier value.
        if item is not None or name in nullable_fields or name not in resolved:
            resolved[name] = item
    value.clear()
    value.update(resolved)


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
    with open_package(argv[1]) as package:
        return verify_package(*package)


def verify_package(read, names):
    man = json.loads(read("manifest.json"), object_pairs_hook=ManifestObject,
                     parse_constant=reject_json_constant)
    if not isinstance(man, dict):
        raise ValueError("manifest.json must contain an object")
    validate_manifest_object(man)
    if type(man.get("schema_version")) is not int or man["schema_version"] != 1:
        print("unsupported schema_version %r" % man.get("schema_version"), file=sys.stderr)
        return 1
    if not isinstance(man["files"], list):
        raise ValueError("manifest files must be an array")
    listed = {"manifest.json"}
    for f in man["files"]:
        name = package_name(f["name"])
        integer(f["rows"], "manifest file rows")
        integer(f["bytes"], "manifest file bytes")
        if name in listed:
            raise ValueError("duplicate or reserved manifest member %r" % name)
        listed.add(name)
    extra = sorted(names - listed)
    if extra:
        raise ValueError("package members not listed in manifest: %s" % ", ".join(extra))
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
            doc = json.loads(data, parse_constant=reject_json_constant)
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

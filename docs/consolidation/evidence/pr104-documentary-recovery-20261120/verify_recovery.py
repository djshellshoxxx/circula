#!/usr/bin/env python3
"""Verify historical evidence in memory; never extract or execute archive contents."""
import argparse
import hashlib
import io
import json
from pathlib import Path, PurePosixPath
import stat
import sys
import tarfile
import zipfile

ROOT = Path(__file__).resolve().parent
PREFIX = "docs/audits/master-consolidation-20261119/"


def check(ok, label):
    if not ok:
        raise ValueError(label)


def digest(data):
    return hashlib.sha256(data).hexdigest()


def safe(name):
    path = PurePosixPath(name)
    return bool(name) and name == path.as_posix() and not path.is_absolute() and not any(
        part in (".", "..") for part in path.parts
    ) and "\\" not in name and "\0" not in name and ":" not in path.parts[0]


def verify():
    for line in (ROOT / "SHA256SUMS").read_text().splitlines():
        expected, name = line.split("  ", 1)
        check(safe(name) and "/" not in name, "checksum path")
        check(digest((ROOT / name).read_bytes()) == expected, "package checksum: " + name)
    manifest = json.loads((ROOT / "payload-manifest.json").read_bytes())
    entries = {entry["archive_entry"]: entry for entry in manifest["entries"]}
    check(len(entries) == len(manifest["entries"]) == 102, "original file count")
    info = manifest["payload_archive"]
    check(info["path"] == "historical-evidence.tar.gz", "wrapper name")
    raw = (ROOT / info["path"]).read_bytes()
    check(len(raw) == info["bytes"] and digest(raw) == info["sha256"], "wrapper identity")
    originals = {}
    with tarfile.open(fileobj=io.BytesIO(raw), mode="r:gz") as archive:
        members = archive.getmembers()
        check(len(members) == 102, "wrapper member count")
        for member in members:
            name = member.name
            check(safe(name) and name.startswith(PREFIX) and name in entries and name not in originals, "wrapper path")
            check(member.isfile() and member.mode == 0o644, "wrapper member type/mode")
            entry = entries[name]
            check(member.size == entry["bytes"], "wrapper size: " + name)
            data = archive.extractfile(member).read()  # Read stream only; no filesystem extraction.
            check(len(data) == entry["bytes"] and digest(data) == entry["sha256"], "payload identity: " + name)
            blob = hashlib.sha1(b"blob " + str(len(data)).encode() + b"\0" + data).hexdigest()
            check(blob == entry["git_blob"], "original Git blob: " + name)
            originals[name] = data
    check(set(originals) == set(entries), "original path set")
    check(sum(len(data) for data in originals.values()) == 5206983, "original byte count")
    external = originals[PREFIX + "verification/qualification-evidence-20261119.manifest.json"]
    listed = json.loads(external)["entries"]
    expected = {entry["path"]: entry for entry in listed}
    check(len(expected) == len(listed) == 205, "original ZIP manifest count")
    nested = {}
    with zipfile.ZipFile(io.BytesIO(originals[PREFIX + "verification/qualification-evidence-20261119.zip"])) as archive:
        check(len(archive.infolist()) == 206, "original ZIP entry count")
        for member in archive.infolist():
            name = member.filename
            check(safe(name) and name not in nested, "ZIP member path")
            check(not member.is_dir() and not stat.S_ISLNK(member.external_attr >> 16) and not member.flag_bits & 1, "ZIP member type/encryption")
            check(member.file_size <= 5_000_000, "ZIP member size bound")
            data = archive.read(member)  # Validates ZIP CRC as well.
            if name == "EVIDENCE-MANIFEST.json":
                check(data == external, "internal/external manifest equality")
            else:
                check(name in expected, "unlisted ZIP member")
                entry = expected[name]
                check(len(data) == entry["bytes"] and digest(data) == entry["sha256"], "ZIP payload identity: " + name)
            nested[name] = data
    check(set(nested) == set(expected) | {"EVIDENCE-MANIFEST.json"}, "ZIP path set")
    counts = []
    for payload in (originals, nested):
        names = [name for name in payload if name.endswith(".json")]
        for name in names:
            json.loads(payload[name])
        counts.append(len(names))
    return originals, nested, {"scope": "artifact integrity only; no product execution", "original_files": 102,
        "original_bytes": 5206983, "nested_payloads": 205, "original_json_parsed": counts[0], "nested_json_parsed": counts[1]}


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    group = parser.add_mutually_exclusive_group()
    group.add_argument("--show-original", metavar="PATH", help="print one verified original UTF-8 document")
    group.add_argument("--show-zip", metavar="PATH", help="print one verified original ZIP text member")
    args = parser.parse_args()
    originals, nested, result = verify()
    if args.show_original or args.show_zip:
        name = args.show_original or args.show_zip
        payload = originals if args.show_original else nested
        check(name in payload, "requested path is not a verified member")
        sys.stdout.write(payload[name].decode("utf-8"))
    else:
        print(json.dumps(result, indent=2))

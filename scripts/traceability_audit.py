#!/usr/bin/env python3
"""QA-042: audit coverage of specification status-table IDs in TRACEABILITY.md.

Read-only by default; this is a drift detector, not an implementation-status
classifier. A matching ID does not establish that its code or tests pass.
"""
import argparse
import json
import re
from pathlib import Path

ID = re.compile(r"^(?:[A-Z]{2,8}-\d{3}|LED-\d+)$")
ROW = re.compile(r"^\|\s*([^|]+?)\s*\|")
SPECS = "docs/specs"
TRACE = "docs/TRACEABILITY.md"


def table_ids(content):
    """Extract ID-bearing markdown table rows, not prose references."""
    found = set()
    for line in content.splitlines():
        match = ROW.match(line)
        if match and ID.fullmatch(match.group(1).strip()):
            found.add(match.group(1).strip())
    return found


def audit(root):
    root = Path(root)
    specs = sorted((root / SPECS).glob("[0-1][0-9]-*.md"))
    if not specs:
        raise ValueError("no numbered specification files found")
    target = root / TRACE
    if not target.is_file():
        raise ValueError("traceability matrix missing")
    sources = {}
    for spec in specs:
        for req in table_ids(spec.read_text(encoding="utf-8")):
            sources.setdefault(req, []).append(spec.relative_to(root).as_posix())
    matrix = table_ids(target.read_text(encoding="utf-8"))
    return {
        "spec_files": len(specs),
        "source_ids": len(sources),
        "matrix_ids": len(matrix),
        "missing": sorted(set(sources) - matrix),
        "matrix_only": sorted(matrix - set(sources)),
        "duplicate_sources": {k: v for k, v in sorted(sources.items()) if len(v) > 1},
    }


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", default=".", help="repository root")
    parser.add_argument("--check", action="store_true", help="exit 1 if source IDs are absent from matrix")
    args = parser.parse_args()
    try:
        report = audit(args.root)
    except (ValueError, OSError) as exc:
        parser.error(str(exc))
    print(json.dumps(report, indent=2, sort_keys=True))
    if args.check and report["missing"]:
        raise SystemExit(1)


if __name__ == "__main__":
    main()

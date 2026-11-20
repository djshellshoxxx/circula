#!/usr/bin/env python3
"""Validate recorded decisions in the QA-040 threshold table, not its prose."""

import argparse
import datetime
from pathlib import Path
import re
import sys


HEADER = ["Threshold", "Value", "Source", "Measured on this repo", "Decision"]
# These are the existing core rows in THRESHOLDS.md. A missing row must not
# silently remove a release condition; additional rows still need decisions.
ABSOLUTE_THRESHOLDS = {
    "Ledger invariant failures in any run",
    "Authorization matrix mismatches",
    "Failing P0 tests",
}
REQUIRED_THRESHOLDS = ABSOLUTE_THRESHOLDS | {
    "Property-test long run (merge/release)",
    "Interleavings per CI configuration",
    "100 disjoint concurrent trades finish within",
    "Branch coverage, posting/balance/credit-limit code",
    "Throughput / latency targets",
    "Login timing difference, existing vs missing account",
    "Migration of fixture with >10,000 entries",
}


def cells(line):
    line = line.strip()
    if not (line.startswith("|") and line.endswith("|")):
        return None
    return [cell.strip() for cell in line[1:-1].split("|")]


def check_thresholds(text):
    lines = text.splitlines()
    headers = [i for i, line in enumerate(lines) if cells(line) == HEADER]
    if len(headers) != 1:
        return ["expected exactly one threshold table with a Decision column"]
    start = headers[0] + 1
    separator = cells(lines[start]) if start < len(lines) else None
    if not separator or len(separator) != len(HEADER) or any(
        re.fullmatch(r":?-{3,}:?", cell) is None for cell in separator
    ):
        return ["threshold table has a missing or malformed separator"]

    errors = []
    names = set()
    count = 0
    for i in range(start + 1, len(lines)):
        line = lines[i]
        if not line.strip():
            continue
        row = cells(line)
        # Prose after the table is not a decision. Continue checking later
        # rows so a blank line cannot hide an unconfirmed threshold.
        if row is None and "|" not in line:
            continue
        if row is None or len(row) != len(HEADER) or not all(row):
            errors.append(f"line {i + 1}: missing or malformed threshold row")
            continue
        count += 1
        name, value, source, _, decision = row
        if name in names:
            errors.append(f"line {i + 1}: duplicate threshold {name!r}")
        names.add(name)
        if (
            name in ABSOLUTE_THRESHOLDS and decision == "n/a (absolute)"
            and value == "0 (absolute)" and source == "QA-040"
        ):
            continue
        approved = re.fullmatch(r"DEC-023 ([0-9]{4}-[0-9]{2}-[0-9]{2})", decision)
        if approved:
            try:
                datetime.date.fromisoformat(approved.group(1))
                continue
            except ValueError:
                pass
        errors.append(
            f"line {i + 1}: {name!r} has no dated DEC-023 decision ({decision!r})"
        )
    if count == 0:
        errors.append("threshold table has no decision rows")
    for name in sorted(REQUIRED_THRESHOLDS - names):
        errors.append(f"threshold table is missing required row {name!r}")
    return errors


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("path", nargs="?", default="docs/qa/THRESHOLDS.md")
    args = parser.parse_args()
    try:
        errors = check_thresholds(Path(args.path).read_text(encoding="utf-8"))
    except (OSError, UnicodeError) as exc:
        errors = [f"cannot read threshold decisions: {exc}"]
    for error in errors:
        print(f"thresholds: {error}", file=sys.stderr)
    return 1 if errors else 0


if __name__ == "__main__":
    sys.exit(main())

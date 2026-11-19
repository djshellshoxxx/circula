#!/usr/bin/env python3
"""QA-037: flag destructive migration statements, including append-only updates.

This lexical guard complements migration execution tests. Comments and quoted
values cannot hide statements or manufacture review exemptions. A reviewed
exception needs a nonempty -- allow-destructive: reason on a statement line.
"""
import re
import sys
from pathlib import Path


PROTECTED = {
    "entries", "audit_events", "audit_subjects", "transaction_status_events",
    "dispute_events", "consistency_runs", "network_policies", "network_terms",
    "terms_acceptances", "member_erasures", "break_glass_grants",
    "transaction_status_event_order", "dispute_event_order",
}
TOKEN = re.compile(
    r"(?P<comment>--[^\n]*|/\*[\s\S]*?\*/)"
    r"|(?P<quoted>'(?:''|[^'])*'|\"(?:\"\"|[^\"])*\"|`(?:``|[^`])*`|\[[^\]]*\])"
    r"|(?P<word>[A-Za-z_][A-Za-z_0-9$]*)|(?P<other>\S)"
)
EXEMPTION = re.compile(r"^--\s*allow-destructive:\s*\S.*$", re.IGNORECASE)


def violations(sql):
    tokens, allowed = [], set()
    line, offset = 1, 0
    for match in TOKEN.finditer(sql):
        line += sql.count("\n", offset, match.start())
        value, kind = match.group(), match.lastgroup
        end_line = line + value.count("\n")
        if kind == "comment":
            if EXEMPTION.fullmatch(value):
                allowed.add(line)
        else:
            if kind == "quoted":
                delimiter = value[0]
                value = value[1:-1]
                if delimiter != "[":
                    value = value.replace(delimiter * 2, delimiter)
            tokens.append((value.lower(), kind, line, end_line))
        line, offset = end_line, match.end()

    def keyword(index, value):
        return 0 <= index < len(tokens) and tokens[index][:2] == (value, "word")

    def table_at(index):
        # SQLite permits both quoted names and schema-qualified names.
        if index + 2 < len(tokens) and tokens[index + 1][0] == ".":
            index += 2
        return index

    found, insert_target, insert_start = [], None, None
    for index, (value, kind, start, _) in enumerate(tokens):
        if kind == "other" and value == ";":
            insert_target, insert_start = None, None
        if kind != "word":
            continue
        end, reason = index, None
        if value == "insert":
            target = index + 1
            if keyword(target, "or"):
                target += 2
            if keyword(target, "into"):
                target = table_at(target + 1)
                if target < len(tokens):
                    insert_target, insert_start = tokens[target][0], start
        elif value == "alter" and keyword(index + 1, "table"):
            # SQLite accepts DROP column_name as well as DROP COLUMN name.
            target = table_at(index + 2)
            if keyword(target + 1, "drop") and not keyword(target + 2, "column"):
                end, reason = target + 2, "DROP COLUMN"
        elif value == "drop" and index + 1 < len(tokens):
            if tokens[index + 1][1] == "word" and tokens[index + 1][0] in {"table", "column", "trigger"}:
                end, reason = index + 1, "DROP " + tokens[index + 1][0].upper()
                # Include the object name (and IF EXISTS) in exemption scope.
                end = table_at(end + (3 if keyword(end + 1, "if") and keyword(end + 2, "exists") else 1))
        elif value == "delete" and keyword(index + 1, "from"):
            end, reason = table_at(index + 2), "DELETE FROM"
        elif value == "update" or (value == "replace" and keyword(index + 1, "into")):
            end = index + (2 if value == "replace" else 1)
            if value == "update" and keyword(end, "or"):
                end += 2
            end = table_at(end)
            if end < len(tokens) and tokens[end][0] in PROTECTED:
                reason = value.upper() + " append-only " + tokens[end][0]
            elif value == "update" and keyword(index - 1, "do") and keyword(index + 1, "set") and insert_target in PROTECTED:
                end, reason = index + 1, "UPSERT append-only " + insert_target
                start = insert_start
        if reason:
            last = tokens[min(end, len(tokens) - 1)][3]
            if not any(number in allowed for number in range(start, last + 1)):
                found.append((start, reason))
    return found


def main(paths):
    failed = False
    for path in paths:
        for line, reason in violations(Path(path).read_text(encoding="utf-8")):
            print(f"{path}:{line}: QA-037: {reason} requires '-- allow-destructive: reason'")
            failed = True
    return int(failed)


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))

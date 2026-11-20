"""Exercise evidence exceptions with the real scanner, using only temporary copies.

Run with the CI-pinned command, or an equivalent prebuilt executable:
    python3 scripts/check_gitleaks_config.py -- go run github.com/zricethezav/gitleaks/v8@v8.28.0
    python3 scripts/check_gitleaks_config.py -- /path/to/gitleaks

No downloads or builds are performed except those requested by the supplied command.
Reports and diagnostics are redacted. Synthetic credentials are generated at runtime.
"""

from collections import Counter
import base64
import hashlib
import json
import math
from pathlib import Path
import re
import subprocess
import sys
import tempfile
import tomllib


ROOT = Path(__file__).resolve().parents[1]
BEGIN = "# BEGIN verified evidence exceptions"
END = "# END verified evidence exceptions"
RULE = "generic-api-key"
CURSOR = "docs/audits/evidence/post-snapshot-20261119/core-cursor-security-merged-a50f90b.execution.json"
CATALOGUE = "docs/audits/evidence/post-snapshot-20261119/core-catalogue-traceability-merged-12af211.execution.json"
INVENTORY = "docs/audits/BRANCH_INVENTORY_2026-11-19.json"
PROSE = "docs/consolidation/BRANCH_REFRESH_20261119.md"
REMOTE = "docs/audits/evidence/closing-publication-20261119/remote-heads.json"
SHARED = "docs/consolidation/evidence/shared-pages-receipt.json"
LINES = {
    CURSOR: [66, 67, 89, 94, 100], CATALOGUE: [36], INVENTORY: [1] * 4,
    PROSE: [133], REMOTE: [18, 36, 65, 87], SHARED: [102, 255, 298],
}
# Main preserves JSON-escaped digests in these immutable evidence files. The
# scanner reads their encoded bytes, while exception controls must also exercise
# the decoded values. Keep both representations under explicit assertions.
ESCAPED_DOCUMENTS = frozenset((CURSOR, CATALOGUE, INVENTORY))
RAW_LINES = {path: lines for path, lines in LINES.items() if path not in ESCAPED_DOCUMENTS}
JSON_STRING = re.compile(r'"(?:[^"\\]|\\.)*"')
BRANCHES = (
    "chatgpt/circula-ledger-auth-guard-20261115", "claude/aud4-auth-fixes",
    "claude/ux-047-auth-catalogue", "dependabot/go_modules/github.com/getkin/kin-openapi-0.153.0",
)


def require(condition, message):
    if not condition:
        raise AssertionError(message)


def known_values(documents):
    cursor = json.loads(documents[CURSOR])["files_sha256"]
    catalogue = json.loads(documents[CATALOGUE])["files_sha256"]
    heads = json.loads(documents[INVENTORY])["dependency_node_shas"]
    remote = json.loads(documents[REMOTE])["heads"]
    shared = json.loads(documents[SHARED])
    return {
        CURSOR: [cursor[name] for name in (
            "internal/api/api.go", "internal/api/auth.go", "internal/api/h_password.go",
            "internal/api/h_tokens.go", "internal/api/replay_auth.go",
        )],
        CATALOGUE: [catalogue["internal/web/messages/auth_test.go"]],
        INVENTORY: [heads[name] for name in BRANCHES],
        PROSE: ["CSRF/idempotency"],
        REMOTE: [remote[name] for name in BRANCHES],
        SHARED: [shared["before"]["source_blobs"]["internal/web/h_auth.go"],
                 shared["source_sha256"]["internal/web/h_auth.go"]],
    }


def check_scope(config, values):
    require(config.count(BEGIN) == 1 and config.count(END) == 1, "exception section must be unique")
    before, section = config.split(BEGIN, 1)
    section, after = section.split(END, 1)
    # The marker's description is a comment, not TOML content.
    section = section.split("\n", 1)[1]
    parsed = tomllib.loads(config)
    require(parsed.get("extend") == {"useDefault": True}, "all default scanner rules must remain enabled")
    scoped = tomllib.loads(section)
    require(set(scoped) == {"rules"} and len(scoped["rules"]) == 1, "exceptions must extend exactly one rule")
    rule = scoped["rules"][0]
    require(set(rule) == {"id", "allowlists"} and rule["id"] == RULE,
            "exceptions must extend only the observed rule, without overriding its matcher")
    blocks = rule["allowlists"]
    require(len(blocks) == len(values), "expected six narrowly scoped exception blocks")
    remaining = dict(values)
    for block in blocks:
        require(set(block) == {"description", "condition", "paths", "regexTarget", "regexes"},
                "unexpected exception predicate")
        require(block["condition"] == "AND", "file and value predicates must both match")
        require(block["regexTarget"] == "secret", "exceptions must match a value, not a whole line")
        path = next((name for name in remaining
                     if block["paths"] == ["^" + re.escape(name).replace(r"\-", "-") + "$"]), None)
        require(path is not None, "exception must name exactly one observed file with both anchors")
        require(set(block["regexes"]) == {"^" + re.escape(value) + "$" for value in remaining.pop(path)},
                "exceptions must contain only the exact observed values, with both anchors")
    require(not remaining, "missing exception scope")
    return before + after


def synthetic(label, length, hexadecimal=True):
    """Deterministic, public test data with enough entropy for the real rule."""
    for attempt in range(100):
        digest = hashlib.sha256(f"circula scanner control {label} {attempt}".encode()).digest()
        value = (digest.hex() if hexadecimal else base64.b64encode(digest).decode())[:length]
        entropy = -sum((count / len(value)) * math.log2(count / len(value)) for count in Counter(value).values())
        if entropy > 3.6 and value[0].isalnum() and any(character.isdigit() for character in value):
            return value
    raise AssertionError("could not construct synthetic scanner control")


def expected(rename=lambda name: name, lines=LINES):
    return Counter((rename(path), RULE, line) for path, numbers in lines.items() for line in numbers)


def decoded_documents(documents):
    """Decode only temporary JSON string copies without moving physical lines."""
    decoded = dict(documents)
    for path in ESCAPED_DOCUMENTS:
        source = documents[path]
        decoded[path] = JSON_STRING.sub(
            lambda match: json.dumps(json.loads(match.group()), ensure_ascii=False), source,
        )
        require(json.loads(decoded[path]) == json.loads(source),
                f"{path}: temporary normalization changed the JSON values")
        require(decoded[path].count("\n") == source.count("\n"),
                f"{path}: temporary normalization changed physical lines")
    return decoded


def scan(command, temporary, label, documents, config):
    case = temporary / label
    source = case / "input"
    source.mkdir(parents=True)
    for name, content in documents.items():
        destination = source / name
        destination.parent.mkdir(parents=True, exist_ok=True)
        destination.write_text(content, encoding="utf-8")
    config_path = case / "scanner.toml"
    config_path.write_text(config, encoding="utf-8")
    report = case / "report.json"
    result = subprocess.run(
        command + ["dir", ".", "--config", str(config_path), "--no-banner", "--no-color",
                   "--redact", "--report-format", "json", "--report-path", str(report)],
        cwd=source, capture_output=True, text=True, timeout=120,
    )
    require(report.is_file(), f"{label}: scanner did not produce a report (exit {result.returncode})")
    findings = json.loads(report.read_text(encoding="utf-8"))
    require(isinstance(findings, list), f"{label}: invalid scanner report")
    require(result.returncode == (1 if findings else 0), f"{label}: unexpected scanner exit {result.returncode}")
    # Never print matched text or secret fields, including on a failed assertion.
    return Counter((finding["File"].replace("\\", "/"), finding["RuleID"], finding["StartLine"])
                   for finding in findings)


def check(label, actual, wanted):
    require(actual == wanted, f"{label}: finding metadata differs; expected {dict(wanted)}, observed {dict(actual)}")
    print(f"PASS {label}: {sum(actual.values())} findings", flush=True)


def main(command):
    require(bool(command), "supply the pinned scanner command after --")
    documents = {name: (ROOT / name).read_text(encoding="utf-8") for name in LINES}
    values = known_values(documents)
    config = (ROOT / ".gitleaks.toml").read_text(encoding="utf-8")
    baseline = check_scope(config, values)
    with tempfile.TemporaryDirectory(prefix="circula-gitleaks-controls-") as name:
        temporary = Path(name)
        check("original encoded documents before exceptions",
              scan(command, temporary, "raw-baseline", documents, baseline), expected(lines=RAW_LINES))
        check("original documents after exceptions", scan(command, temporary, "fixed", documents, config), Counter())

        # Only the temporary controls are normalized. Original documents and
        # their SHA-256 manifests remain untouched; the default scanner and
        # exact rule/path/value allowlists are unchanged.
        documents = decoded_documents(documents)
        for path, observed in values.items():
            require(all(value in documents[path] for value in observed),
                    f"{path}: decoded controls must contain every excepted value")
        check("decoded documents before exceptions",
              scan(command, temporary, "decoded-baseline", documents, baseline), expected())
        check("decoded documents after exceptions",
              scan(command, temporary, "decoded-fixed", documents, config), Counter())

        changed = dict(documents)
        for path, observed in values.items():
            for index, value in enumerate(observed):
                replacement = synthetic(f"{path}-{index}", len(value), hexadecimal=path != PROSE)
                require(replacement not in observed, "synthetic control must differ from observed values")
                changed[path] = changed[path].replace(value, replacement)
        check("synthetic replacements in original contexts", scan(command, temporary, "replaced", changed, config), expected())

        extended = dict(documents)
        for path, observed in values.items():
            for value in observed:
                extended[path] = extended[path].replace(value, value + "9")
        check("extended values are not excepted", scan(command, temporary, "extended", extended, config), expected())

        for label, rename in (("prefixed-paths", lambda path: "outside/" + path),
                              ("suffixed-paths", lambda path: path + ".copy")):
            moved = {rename(path): content for path, content in documents.items()}
            check(label, scan(command, temporary, label, moved, config), expected(rename))

        adjacent = {}
        wanted = Counter()
        for path, content in documents.items():
            lines = content.splitlines(keepends=True)
            line = LINES[path][0]
            ordinary = synthetic(path + "-adjacent", 40)
            provider = "ghp" + "_" + synthetic(path + "-provider", 36)
            lines[line - 1] = lines[line - 1].rstrip("\r\n") + f' api_key = "{ordinary}"; probe = "{provider}"\n'
            adjacent[path] = "".join(lines)
            wanted[(path, RULE, line)] += 1
            wanted[(path, "github-pat", line)] += 1
        check("same-line generic and provider controls", scan(command, temporary, "adjacent", adjacent, config), wanted)

        # A test-only rule matching those same nonsecret values verifies rule scope
        # with the scanner itself; the real provider controls above cover default rules.
        representative = {path: observed[0] for path, observed in values.items()}
        expression = "(" + "|".join(re.escape(value) for value in representative.values()) + ")"
        probe_config = config + "\n[[rules]]\nid = 'scope-control'\nregex = '''" + expression + "'''\n"
        probe_documents = {path: f'api_key = "{value}"\n' for path, value in representative.items()}
        wanted = Counter((path, "scope-control", 1) for path in documents)
        check("exceptions affect only their target rule", scan(command, temporary, "rule-scope", probe_documents, probe_config), wanted)
    print("PASS: all ten real-scanner checks; original repository documents were not modified.")


if __name__ == "__main__":
    arguments = sys.argv[1:]
    if arguments[:1] == ["--"]:
        arguments = arguments[1:]
    try:
        main(arguments)
    except (AssertionError, OSError, ValueError, subprocess.TimeoutExpired) as error:
        print(f"FAIL: {error}", file=sys.stderr)
        sys.exit(1)

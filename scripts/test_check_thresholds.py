"""Decision-table and full release-gate regressions for QA-040."""

from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest

from check_thresholds import check_thresholds


ROOT = Path(__file__).resolve().parents[1]
# Keep fixtures independent of future owner decisions in the real document.
PROPOSED = """# Quality thresholds

| Threshold | Value | Source | Measured on this repo | Decision |
|---|---:|---|---|---|
| Ledger invariant failures in any run | 0 (absolute) | QA-040 | tests green | n/a (absolute) |
| Authorization matrix mismatches | 0 (absolute) | QA-040 | tests green | n/a (absolute) |
| Failing P0 tests | 0 (absolute) | QA-040 | tests green | n/a (absolute) |
| Property-test long run (merge/release) | 100,000 sequences | recommended | recorded | UNCONFIRMED |
| Interleavings per CI configuration | 100 | recommended | not 100 yet | UNCONFIRMED |
| 100 disjoint concurrent trades finish within | 60 s, zero deadlocks | recommended | measured | UNCONFIRMED |
| Branch coverage, posting/balance/credit-limit code | >= 90 % | recommended | not measured | UNCONFIRMED |
| Throughput / latency targets | none claimed | QA-040 | reference hardware | UNCONFIRMED |
| Login timing difference, existing vs missing account | per security requirement | QA-040 | not measured | UNCONFIRMED |
| Migration of fixture with >10,000 entries | time recorded | recommended | test exists | UNCONFIRMED |

How to confirm: replace each
UNCONFIRMED with `DEC-023 <date>`.
"""


def confirmed_fixture():
    # Only the Decision cells change. Instructional prose still says UNCONFIRMED.
    return "\n".join(
        line.replace("| UNCONFIRMED |", "| DEC-023 2026-11-18 |")
        if line.startswith("|") else line
        for line in PROPOSED.splitlines()
    ) + "\n"


def hidden_decision_fixtures():
    lines = PROPOSED.splitlines()
    def is_owner_row(line):
        return line.startswith("|") and "| UNCONFIRMED |" in line
    return {
        "owner rows deleted": "\n".join(line for line in lines if not is_owner_row(line)),
        "owner rows hidden as prose": "\n".join(
            line.replace("|", "\t") if is_owner_row(line) else line for line in lines
        ),
        "unknown absolute only": "\n".join(lines[:4])
        + "\n| Invented threshold | 0 (absolute) | QA-040 | test | n/a (absolute) |\n",
        "unknown absolute appended": confirmed_fixture()
        + "\n| Invented threshold | 0 (absolute) | QA-040 | test | n/a (absolute) |\n",
        "one required row deleted": "\n".join(
            line for line in confirmed_fixture().splitlines()
            if not line.startswith("| Authorization matrix mismatches |")
        ),
    }


class ThresholdDecisionsTest(unittest.TestCase):
    def test_confirmed_decisions_ignore_instructional_marker(self):
        text = confirmed_fixture()
        self.assertIn("UNCONFIRMED with", text)
        self.assertEqual(check_thresholds(text), [])

    def test_unconfirmed_decisions_refuse(self):
        self.assertTrue(check_thresholds(PROPOSED))

    def test_missing_or_malformed_decisions_refuse(self):
        text = confirmed_fixture()
        cases = {
            "missing table": "# No decision table\n",
            "missing decision column": text.replace("| Decision |", "| Notes |"),
            "missing decision value": text.replace("| DEC-023 2026-11-18 |", "| |", 1),
            "missing cell": text.replace("| DEC-023 2026-11-18 |", "|", 1),
            "unconfirmed row": text.replace("DEC-023 2026-11-18", "UNCONFIRMED", 1),
            "unconfirmed after blank": PROPOSED.replace("| Property-test", "\n| Property-test", 1),
            "undated decision": text.replace("DEC-023 2026-11-18", "DEC-023", 1),
            "invalid date": text.replace("DEC-023 2026-11-18", "DEC-023 2026-02-30", 1),
            "wrong decision": text.replace("DEC-023 2026-11-18", "APPROVED 2026-11-18", 1),
            "absolute bypass": text.replace("DEC-023 2026-11-18", "n/a (absolute)", 1),
            "empty table": "\n".join(text.splitlines()[:4]) + "\n\n",
            "duplicate required row": text + "\n| Failing P0 tests | 0 (absolute) | QA-040 | tests green | n/a (absolute) |\n",
        }
        for name, candidate in cases.items():
            with self.subTest(name=name):
                self.assertTrue(check_thresholds(candidate))

    def test_required_rows_cannot_be_hidden_or_replaced(self):
        for name, candidate in hidden_decision_fixtures().items():
            with self.subTest(name=name):
                self.assertTrue(check_thresholds(candidate))

    def test_extra_rows_need_dated_decisions(self):
        extra = "\n| Additional threshold | 10 | proposed | recorded | DEC-023 2026-11-18 |\n"
        self.assertEqual(check_thresholds(confirmed_fixture() + extra), [])
        self.assertTrue(check_thresholds(
            confirmed_fixture() + extra.replace("DEC-023 2026-11-18", "UNCONFIRMED")
        ))

    def run_gate(self, text):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            (root / "scripts").mkdir()
            (root / "docs/qa").mkdir(parents=True)
            for name in ("release-gate.sh", "check_thresholds.py"):
                shutil.copyfile(ROOT / "scripts" / name, root / "scripts" / name)
            (root / "docs/qa/THRESHOLDS.md").write_text(text, encoding="utf-8")
            # There are no defect rows in this isolated classification fixture.
            (root / "docs/qa/DEFECTS.md").write_text("# Test fixture\n", encoding="utf-8")
            result = subprocess.run(
                ["bash", "scripts/release-gate.sh", "-s",
                 "fmt,vet,race,lint,contract,golden,e2e,load,vuln,pins,hygiene,drill",
                 "-o", str(root / "evidence")],
                cwd=root, capture_output=True, text=True, timeout=30,
            )
            records = list((root / "evidence").glob("gate-*.md"))
            self.assertEqual(len(records), 1, result.stdout + result.stderr)
            return result, records[0].read_text(encoding="utf-8")

    def test_full_gate_accepts_only_confirmed_fixture(self):
        result, record = self.run_gate(confirmed_fixture())
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertIn("| thresholds | PASS |", record)
        self.assertIn("| screen-reader | MANUAL |", record)
        self.assertIn("| race | SKIPPED |", record)

    def test_full_gate_retains_unconfirmed_decision_block(self):
        result, record = self.run_gate(PROPOSED)
        self.assertEqual(result.returncode, 1, result.stdout + result.stderr)
        self.assertIn("| thresholds | FAIL |", record)

    def test_full_gate_refuses_hidden_or_replaced_decisions(self):
        for name, candidate in hidden_decision_fixtures().items():
            with self.subTest(name=name):
                result, record = self.run_gate(candidate)
                self.assertEqual(result.returncode, 1, result.stdout + result.stderr)
                self.assertIn("| thresholds | FAIL |", record)


if __name__ == "__main__":
    unittest.main()

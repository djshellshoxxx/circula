"""QA-042 traceability audit regression tests."""
import importlib.util
import tempfile
import unittest
from pathlib import Path

MODULE = Path(__file__).with_name("traceability_audit.py")
spec = importlib.util.spec_from_file_location("traceability_audit", MODULE)
audit_module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(audit_module)


class TraceabilityAuditTests(unittest.TestCase):
    def test_reports_missing_requirements_without_confusing_prose_mentions(self):
        with tempfile.TemporaryDirectory() as d:
            root = Path(d)
            (root / "docs/specs").mkdir(parents=True)
            (root / "docs/specs/04-functional.md").write_text(
                "| ID | Priority | Status |\n|---|---|---|\n"
                "| FR-001 | P0 | built |\n| FR-002 | P0 | partial |\n"
                "Mention FR-003 in prose only.\n", encoding="utf-8"
            )
            (root / "docs/TRACEABILITY.md").write_text(
                "| ID | Pri | Requirement |\n|---|---|---|\n"
                "| FR-001 | P0 | example |\n", encoding="utf-8"
            )
            report = audit_module.audit(root)
            self.assertEqual(report["missing"], ["FR-002"])
            self.assertEqual(report["source_ids"], 2)
            self.assertEqual(report["matrix_ids"], 1)

    def test_no_missing_rows_when_matrix_covers_source(self):
        with tempfile.TemporaryDirectory() as d:
            root = Path(d)
            (root / "docs/specs").mkdir(parents=True)
            (root / "docs/specs/12-quality.md").write_text(
                "| ID | Priority |\n|---|---|\n| QA-042 | P0 |\n", encoding="utf-8"
            )
            (root / "docs/TRACEABILITY.md").write_text(
                "| ID | Pri |\n|---|---|\n| QA-042 | P0 |\n", encoding="utf-8"
            )
            self.assertEqual(audit_module.audit(root)["missing"], [])

    def test_rejects_missing_inputs(self):
        with tempfile.TemporaryDirectory() as d:
            with self.assertRaisesRegex(ValueError, "no numbered specification"):
                audit_module.audit(d)


if __name__ == "__main__":
    unittest.main()

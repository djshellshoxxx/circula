"""QA-042 traceability generator tests (scripts/traceability.py)."""
import importlib.util
import tempfile
import textwrap
import unittest
from pathlib import Path

MODULE = Path(__file__).with_name("traceability.py")
spec = importlib.util.spec_from_file_location("traceability", MODULE)
tr = importlib.util.module_from_spec(spec)
spec.loader.exec_module(tr)

SPEC04 = textwrap.dedent("""\
    # 04 Functional requirements

    | ID | Priority | Status | Note |
    |---|---|---|---|
    | FR-001 | P0 | built | Create network |
    | FR-002 | P1 | partial | Setup secret only |
    | FR-003 | P0 | built | New one |

    ### FR-003 · P0 · Close a network

    Mentioning FR-009 in prose does not create a row.
    """)

SPEC07 = textwrap.dedent("""\
    # 07 API contract

    | ID | Operation | Role | Idem. | Pri. | Acceptance (testable) |
    |---|---|---|---|---|---|
    | API-001 | `GET /network` | Member | No | P0 | Returns name |
    """)

MATRIX = textwrap.dedent("""\
    # Traceability matrix

    Old header text.

    ## 04 Functional requirements

    | ID | Pri | Requirement (short) | Status | Code path(s) | Test name(s) | Note |
    |---|---|---|---|---|---|---|
    | FR-001 | P0 | Create a network with policy | partial | `internal/svc/net.go` (CreateNetwork)<br>`internal/svc/gone.go` (Nope) | `internal/svc/net_test.go:TestCreate`<br>`internal/svc/net_test.go:TestVanished` | old note |
    | FR-002 | P1 | Setup secret | partial | - | - | old |

    ## 07 API contract

    | ID | Pri | Requirement (short) | Status | Code path(s) | Test name(s) | Note |
    |---|---|---|---|---|---|---|
    | API-001 | P0 | Network read | built | `internal/svc/net.go` | - | curated API note |
    """)

CODE = "package svc\n\nfunc CreateNetwork() {}\n"
TESTS = textwrap.dedent("""\
    package svc

    import "testing"

    func TestCreate(t *testing.T) {}

    // FR-002 and FR-003: closing keeps the secret used.
    // Second comment line.
    func TestCloseKeepsSecret(t *testing.T) {}

    // Unrelated helper mentioning FR-001.
    func helper() {}
    """)


def make_repo(d, spec04=SPEC04, matrix=MATRIX):
    root = Path(d)
    (root / "docs/specs").mkdir(parents=True)
    (root / "docs/specs/04-functional.md").write_text(spec04, encoding="utf-8")
    (root / "docs/specs/07-api.md").write_text(SPEC07, encoding="utf-8")
    (root / "docs/TRACEABILITY.md").write_text(matrix, encoding="utf-8")
    (root / "internal/svc").mkdir(parents=True)
    (root / "internal/svc/net.go").write_text(CODE, encoding="utf-8")
    (root / "internal/svc/net_test.go").write_text(TESTS, encoding="utf-8")
    return root


def row(text, rid):
    for line in text.splitlines():
        if line.startswith("| %s |" % rid):
            return [c.strip() for c in line.strip().strip("|").split("|")]
    return None


class GeneratorTests(unittest.TestCase):
    def test_status_and_note_come_from_spec_title_from_matrix(self):
        with tempfile.TemporaryDirectory() as d:
            text, _ = tr.generate(make_repo(d))
            r = row(text, "FR-001")
            self.assertEqual(r[1], "P0")
            self.assertEqual(r[2], "Create a network with policy")
            self.assertEqual(r[3], "built")
            self.assertEqual(r[6], "Create network")

    def test_missing_id_is_added_with_heading_title(self):
        with tempfile.TemporaryDirectory() as d:
            text, report = tr.generate(make_repo(d))
            r = row(text, "FR-003")
            self.assertIsNotNone(r)
            self.assertEqual(r[2], "Close a network")
            self.assertIn("FR-003", report["added"])
            self.assertIsNone(row(text, "FR-009"))

    def test_stale_citations_are_dropped_and_reported(self):
        with tempfile.TemporaryDirectory() as d:
            text, report = tr.generate(make_repo(d))
            r = row(text, "FR-001")
            self.assertNotIn("gone.go", r[4])
            self.assertNotIn("TestVanished", r[5])
            self.assertIn("`internal/svc/net.go` (CreateNetwork)", r[4])
            stale = "\n".join(report["stale"])
            self.assertIn("internal/svc/gone.go", stale)
            self.assertIn("TestVanished", stale)

    def test_a_citation_may_name_several_symbols(self):
        with tempfile.TemporaryDirectory() as d:
            m = MATRIX.replace("(CreateNetwork)<br>", "(CreateNetwork, CreateNetwork)<br>`internal/svc/net.go` (CreateNetwork, Nope)<br>")
            text, report = tr.generate(make_repo(d, matrix=m))
            self.assertIn("(CreateNetwork, CreateNetwork)", row(text, "FR-001")[4])
            self.assertIn("FR-001 code `internal/svc/net.go` (CreateNetwork, Nope)", report["stale"])

    def test_tests_citing_the_id_in_their_doc_comment_are_discovered(self):
        with tempfile.TemporaryDirectory() as d:
            text, _ = tr.generate(make_repo(d))
            self.assertIn("internal/svc/net_test.go:TestCloseKeepsSecret", row(text, "FR-002")[5])
            self.assertIn("internal/svc/net_test.go:TestCloseKeepsSecret", row(text, "FR-003")[5])
            # helper() is not a test, so FR-001 gains nothing from its comment.
            self.assertEqual(row(text, "FR-001")[5], "`internal/svc/net_test.go:TestCreate`")

    def test_rows_only_in_matrix_keep_curated_status(self):
        with tempfile.TemporaryDirectory() as d:
            text, _ = tr.generate(make_repo(d))
            r = row(text, "API-001")
            self.assertEqual(r[3], "built")
            self.assertEqual(r[6], "curated API note")

    def test_output_is_deterministic_and_check_detects_drift(self):
        with tempfile.TemporaryDirectory() as d:
            root = make_repo(d)
            first, _ = tr.generate(root)
            (root / "docs/TRACEABILITY.md").write_text(first, encoding="utf-8")
            second, _ = tr.generate(root)
            self.assertEqual(first, second)
            self.assertEqual(tr.check(root), [])
            (root / "docs/specs/04-functional.md").write_text(
                SPEC04.replace("| FR-002 | P1 | partial |", "| FR-002 | P1 | built |"), encoding="utf-8")
            self.assertNotEqual(tr.check(root), [])

    def test_check_fails_when_a_cited_test_is_removed(self):
        with tempfile.TemporaryDirectory() as d:
            root = make_repo(d)
            first, _ = tr.generate(root)
            (root / "docs/TRACEABILITY.md").write_text(first, encoding="utf-8")
            (root / "internal/svc/net_test.go").write_text(
                TESTS.replace("func TestCreate(t *testing.T) {}\n", ""), encoding="utf-8")
            self.assertTrue(any("TestCreate" in line for line in tr.check(root)))

    def test_header_counts_statuses(self):
        with tempfile.TemporaryDirectory() as d:
            text, _ = tr.generate(make_repo(d))
            self.assertIn("built 3, partial 1", text)

    def test_duplicate_status_rows_are_an_error(self):
        with tempfile.TemporaryDirectory() as d:
            dup = SPEC04.replace("| FR-003 | P0 | built | New one |",
                                 "| FR-003 | P0 | built | New one |\n| FR-001 | P0 | built | again |")
            with self.assertRaisesRegex(ValueError, "FR-001"):
                tr.generate(make_repo(d, spec04=dup))


if __name__ == "__main__":
    unittest.main()

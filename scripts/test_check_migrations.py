"""QA-037 / OPS-022 migration protection regressions (A3-F14)."""
import unittest

from check_migrations import violations


class MigrationSafetyTests(unittest.TestCase):
    def test_recorded_bypass(self):
        self.assertEqual(len(violations("""
DROP TRIGGER audit_no_update;
UPDATE audit_events SET summary='rewritten';
UPDATE entries SET amount=0;
""")), 3)

    def test_multiline_comments_conflict_modes_and_quoted_names(self):
        for sql in (
            'UPDATE\n/* bypass */ main."audit_events" SET summary=\'x\';',
            'UPDATE OR IGNORE [entries] SET amount=0;',
            'UPDATE `audit_subjects` SET person_id=\'x\';',
            "UPDATE 'entries' SET amount=0;",
            'DROP\nTRIGGER IF EXISTS main."audit_no_update";',
            'DELETE /* gap */ FROM entries;',
            'INSERT OR REPLACE INTO entries(id) VALUES (1);',
            'REPLACE INTO audit_events(id) VALUES (1);',
            'ALTER TABLE entries DROP COLUMN amount;',
            'ALTER TABLE "scratch" DROP obsolete;',
            'ALTER TABLE main.[scratch] DROP obsolete;',
            "INSERT INTO audit_events(id) VALUES(1) ON CONFLICT(id) DO UPDATE SET id=excluded.id;",
            "INSERT OR IGNORE INTO main.[entries](id) VALUES(1) ON CONFLICT(id) DO UPDATE SET id=excluded.id;",
        ):
            with self.subTest(sql=sql):
                self.assertTrue(violations(sql))

    def test_additive_schema_and_literal_text_are_allowed(self):
        self.assertEqual(violations("""
-- UPDATE entries SET amount=0;
/* DROP TRIGGER audit_no_update; */
SELECT 'UPDATE entries', 'DROP TABLE', 'DELETE FROM';
CREATE TRIGGER entries_no_update BEFORE UPDATE ON entries
BEGIN SELECT RAISE(ABORT, 'append-only'); END;
CREATE TABLE new_metadata(id TEXT);
INSERT INTO new_metadata VALUES ('x');
UPDATE persons SET display_name='changed';
INSERT INTO persons(id) VALUES(1) ON CONFLICT(id) DO UPDATE SET id=excluded.id;
"""), [])

    def test_exemption_requires_real_comment_reason_on_statement_line(self):
        self.assertEqual(violations("DROP TABLE scratch; -- allow-destructive: temporary table"), [])
        self.assertEqual(violations("DROP\nTABLE scratch; -- allow-destructive: temporary table"), [])
        self.assertEqual(violations("INSERT INTO audit_events(id) VALUES(1) -- allow-destructive: reviewed repair\nON CONFLICT(id) DO UPDATE SET id=excluded.id;"), [])
        for suffix in (
            " -- allow-destructive", " -- allow-destructive:   ",
            " /* allow-destructive: unsafe */", "\n-- allow-destructive: unrelated",
            " SELECT '-- allow-destructive: fake';",
        ):
            with self.subTest(suffix=suffix):
                self.assertTrue(violations("DROP TABLE scratch;" + suffix))


if __name__ == "__main__":
    unittest.main()

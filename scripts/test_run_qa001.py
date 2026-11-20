"""Focused tests for the manual QA-001 runner; no long sequence run."""
import contextlib
import io
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest import mock

import run_qa001 as runner


class QA001RunnerTests(unittest.TestCase):
    def test_strict_range_and_replay_arguments(self):
        invalid = [[], ["--sequences", "bad"], ["--sequences", "0"],
                   ["--sequences", "1000001"], ["--seed", "-1"],
                   ["--seed", "1", "--sequences", "1"],
                   ["--seed", "1", "--start-seed", "2"],
                   ["--sequences", "2", "--start-seed", str(runner.MAX_SEED)],
                   ["--sequences", "1", "--shards", "9"],
                   ["--sequences", "1", "--timeout-seconds", "19801"]]
        for argv in invalid:
            with self.subTest(argv=argv), contextlib.redirect_stderr(io.StringIO()):
                with self.assertRaises(SystemExit) as error:
                    runner.arguments(argv)
                self.assertEqual(error.exception.code, 2)
        args = runner.arguments(["--sequences", "2", "--start-seed", "200000", "--shards", "8"])
        self.assertEqual((args.start_seed, args.sequences, args.shards), (200000, 2, 2))
        args = runner.arguments(["--seed", "0"])
        self.assertEqual((args.start_seed, args.sequences, args.shards), (0, 1, 1))

    def fake_go(self, directory):
        executable = directory / "fake-go"
        executable.write_text(f"#!{sys.executable}\n" + '''import json, os, pathlib, sys
if sys.argv[1:] == ["version"]:
    print("go version go-fixture linux/amd64")
    sys.exit(0)
controls = {k: v for k, v in os.environ.items() if k.startswith("CIRCULA_QA001_")}
print(json.dumps({"Action": "output", "controls": controls, "argv": sys.argv[1:]}), flush=True)
if os.environ.get("QA001_FAKE_FAILURE") == "1":
    path = pathlib.Path(controls["CIRCULA_QA001_FAILDIR"]) / ("qa001-seed-" + controls["CIRCULA_QA001_START_SEED"] + ".json")
    path.write_text(json.dumps({"seed": int(controls["CIRCULA_QA001_START_SEED"]), "failure": "fixture"}))
    sys.exit(7)
''', encoding="utf-8")
        executable.chmod(0o755)
        return executable

    def test_actual_log_metadata_controls_and_failure_fixtures(self):
        for failing in (False, True):
            with self.subTest(failing=failing), tempfile.TemporaryDirectory() as directory:
                base = Path(directory)
                out = base / "evidence"
                args = runner.arguments(["--sequences", "3", "--start-seed", "200000", "--shards", "2",
                                         "--timeout-seconds", "10", "--race", "--output", str(out),
                                         "--go", str(self.fake_go(base))])
                inherited = {"CIRCULA_QA001_SEED": "999", "CIRCULA_QA001_SEQUENCES": "999",
                             "CIRCULA_QA001_FAILDIR": "/wrong/directory", "CIRCULA_QA001_CAPTURE_TEST": "fatal",
                             "QA001_FAKE_FAILURE": "1" if failing else "0"}
                with mock.patch.dict(os.environ, inherited), contextlib.redirect_stdout(io.StringIO()):
                    self.assertEqual(runner.run(args), 7 if failing else 0)
                metadata = json.loads((out / "metadata.json").read_text())
                self.assertEqual(metadata["status"], "failed" if failing else "passed")
                self.assertEqual(metadata["go_version"], "go version go-fixture linux/amd64")
                self.assertRegex(metadata["revision"], r"^[0-9a-f]{40}$")
                self.assertIsInstance(metadata["dirty"], bool)
                self.assertFalse(metadata["release_threshold_confirmed"])
                self.assertEqual((metadata["start_seed"], metadata["last_seed"], metadata["sequences"]), (200000, 200002, 3))
                record = json.loads((out / "go-test.jsonl").read_text())
                controls = record["controls"]
                self.assertNotIn("CIRCULA_QA001_SEED", controls)
                self.assertNotIn("CIRCULA_QA001_CAPTURE_TEST", controls)
                self.assertEqual(controls["CIRCULA_QA001_SEQUENCES"], "3")
                self.assertEqual(controls["CIRCULA_QA001_SHARDS"], "2")
                self.assertEqual(controls["CIRCULA_QA001_FAILDIR"], str(out / "failures"))
                self.assertEqual(controls["CIRCULA_QA001_REVISION"], metadata["revision"])
                self.assertEqual(record["argv"], metadata["command"][1:])
                self.assertIn("-race", record["argv"])
                self.assertIn("-parallel=2", record["argv"])
                self.assertIn("-timeout=10s", record["argv"])
                self.assertEqual(metadata["failure_fixtures"], ["qa001-seed-200000.json"] if failing else [])
                if failing:
                    self.assertEqual(json.loads((out / "failures/qa001-seed-200000.json").read_text())["seed"], 200000)
                # Never mix a second run into a prior artifact directory.
                with self.assertRaises(FileExistsError):
                    runner.run(args)

    @unittest.skipUnless(os.name == "posix", "manual workflow uses Linux process groups")
    def test_hard_timeout_kills_group_and_preserves_metadata(self):
        with tempfile.TemporaryDirectory() as directory:
            out = Path(directory) / "evidence"
            args = runner.arguments(["--seed", "42", "--timeout-seconds", "1", "--output", str(out)])
            proc = mock.Mock(pid=1234)
            proc.wait.side_effect = [subprocess.TimeoutExpired("go", 61), -9]
            with mock.patch.object(runner.subprocess, "check_output", side_effect=["a" * 40, "", "go version fixture"]), \
                    mock.patch.object(runner.subprocess, "Popen", return_value=proc) as start, \
                    mock.patch.object(runner.os, "killpg") as kill, \
                    contextlib.redirect_stdout(io.StringIO()):
                self.assertEqual(runner.run(args), 124)
            proc.wait.assert_any_call(timeout=61)
            kill.assert_called_once_with(1234, runner.signal.SIGKILL)
            self.assertTrue(start.call_args.kwargs["start_new_session"])
            metadata = json.loads((out / "metadata.json").read_text())
            self.assertEqual((metadata["status"], metadata["exit_code"]), ("timed_out", 124))
            self.assertTrue((out / "go-test.jsonl").exists())

    def test_missing_toolchain_retains_setup_failure(self):
        with tempfile.TemporaryDirectory() as directory:
            out = Path(directory) / "evidence"
            args = runner.arguments(["--seed", "42", "--go", str(Path(directory) / "missing-go"), "--output", str(out)])
            with contextlib.redirect_stdout(io.StringIO()):
                self.assertEqual(runner.run(args), 2)
            metadata = json.loads((out / "metadata.json").read_text())
            self.assertEqual((metadata["status"], metadata["exit_code"]), ("failed", 2))
            self.assertIn("missing-go", (out / "runner-error.txt").read_text())


if __name__ == "__main__":
    unittest.main()

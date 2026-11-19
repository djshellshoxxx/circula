#!/usr/bin/env python3
"""Run an explicit QA-001 seed range and preserve reproducible test evidence."""
import argparse
import datetime as dt
import json
import os
from pathlib import Path
import signal
import subprocess
import sys
import time

ROOT = Path(__file__).resolve().parent.parent
MAX_SEED = 2**63 - 1


def bounded(low, high):
    def parse(raw):
        try:
            value = int(raw)
        except ValueError as exc:
            raise argparse.ArgumentTypeError("must be an integer") from exc
        if not low <= value <= high:
            raise argparse.ArgumentTypeError(f"must be between {low} and {high}")
        return value
    return parse


def arguments(argv=None):
    p = argparse.ArgumentParser(description=__doc__)
    mode = p.add_mutually_exclusive_group(required=True)
    mode.add_argument("--sequences", type=bounded(1, 1000000), help="explicit workload size; not a release threshold")
    mode.add_argument("--seed", type=bounded(0, MAX_SEED), help="replay one isolated seed")
    p.add_argument("--start-seed", type=bounded(0, MAX_SEED), help="first seed of the range; default 100000")
    p.add_argument("--shards", type=bounded(1, 8), default=2, help="concurrent workers (1..8, default 2)")
    p.add_argument("--timeout-seconds", type=bounded(1, 19800), default=18000, help="Go test timeout; hard process limit is 60 seconds longer")
    p.add_argument("--output", type=Path, help="new artifact directory; existing directories are refused")
    p.add_argument("--go", default=os.environ.get("GO", "go"), help="Go executable")
    p.add_argument("--race", action="store_true", help="enable race instrumentation")
    args = p.parse_args(argv)
    if args.seed is not None and args.start_seed is not None:
        p.error("--seed cannot be combined with --start-seed")
    args.start_seed = args.seed if args.seed is not None else (args.start_seed if args.start_seed is not None else 100000)
    args.sequences = 1 if args.seed is not None else args.sequences
    if args.start_seed + args.sequences - 1 > MAX_SEED:
        p.error("seed range overflows int64")
    args.shards = min(args.shards, args.sequences)
    return args


def utcnow():
    return dt.datetime.now(dt.timezone.utc).isoformat()


def run(args):
    stamp = dt.datetime.now(dt.timezone.utc).strftime("%Y%m%dT%H%M%S.%fZ")
    out = (args.output or ROOT / "bin" / ("qa001-" + stamp)).resolve()
    out.mkdir(parents=True, exist_ok=False)
    fixtures = out / "failures"
    fixtures.mkdir()
    command = [args.go, "test", "-json", "-count=1", "-p=1", f"-parallel={args.shards}",
               f"-timeout={args.timeout_seconds}s", "-run=^TestQA001InvariantSequences$"]
    if args.race:
        command.append("-race")
    command.append("./internal/service")
    metadata = {"started_at": utcnow(), "status": "preparing", "command": command,
                "start_seed": args.start_seed, "last_seed": args.start_seed + args.sequences - 1,
                "sequences": args.sequences, "shards": args.shards, "race": args.race,
                "timeout_seconds": args.timeout_seconds, "release_threshold_confirmed": False}
    meta_path = out / "metadata.json"
    def save():
        meta_path.write_text(json.dumps(metadata, indent=2) + "\n", encoding="utf-8")
    save()
    began = time.monotonic()
    code = 2
    try:
        metadata["revision"] = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=ROOT, text=True).strip()
        metadata["dirty"] = bool(subprocess.check_output(["git", "status", "--porcelain"], cwd=ROOT, text=True).strip())
        metadata["go_version"] = subprocess.check_output([args.go, "version"], cwd=ROOT, text=True).strip()
        env = os.environ.copy()
        # Never let an inherited replay/range silently replace the requested run.
        for key in list(env):
            if key.startswith("CIRCULA_QA001_"):
                del env[key]
        env.update({"CIRCULA_QA001_START_SEED": str(args.start_seed),
                    "CIRCULA_QA001_SEQUENCES": str(args.sequences),
                    "CIRCULA_QA001_SHARDS": str(args.shards),
                    "CIRCULA_QA001_FAILDIR": str(fixtures),
                    "CIRCULA_QA001_REVISION": metadata["revision"],
                    "GOMAXPROCS": str(args.shards)})
        metadata["status"] = "running"
        save()
        print(f"QA-001: {args.sequences} seeds from {args.start_seed}, {args.shards} workers; evidence {out}", flush=True)
        # Stream to disk, not an unbounded in-memory buffer. A hard timeout also
        # kills the Go command's test subprocesses on the Linux manual runner.
        with (out / "go-test.jsonl").open("w", encoding="utf-8") as log:
            proc = subprocess.Popen(command, cwd=ROOT, env=env, stdout=log, stderr=subprocess.STDOUT,
                                    start_new_session=(os.name == "posix"))
            try:
                code = proc.wait(timeout=args.timeout_seconds + 60)
                metadata["status"] = "passed" if code == 0 else "failed"
            except (subprocess.TimeoutExpired, KeyboardInterrupt) as exc:
                metadata["status"] = "timed_out" if isinstance(exc, subprocess.TimeoutExpired) else "interrupted"
                if os.name == "posix":
                    os.killpg(proc.pid, signal.SIGKILL)
                else:
                    proc.kill()
                proc.wait()
                code = 124 if metadata["status"] == "timed_out" else 130
    except (OSError, subprocess.SubprocessError) as exc:
        metadata["status"], metadata["error"] = "failed", str(exc)
        (out / "runner-error.txt").write_text(str(exc) + "\n", encoding="utf-8")
    finally:
        metadata.update({"finished_at": utcnow(), "elapsed_seconds": round(time.monotonic() - began, 3),
                         "exit_code": code, "failure_fixtures": sorted(p.name for p in fixtures.glob("*.json"))})
        save()
        print(f"QA-001 {metadata['status']}: {out}", flush=True)
    return code


if __name__ == "__main__":
    try:
        sys.exit(run(arguments()))
    except OSError as exc:
        print(f"QA-001 runner: {exc}", file=sys.stderr)
        sys.exit(2)

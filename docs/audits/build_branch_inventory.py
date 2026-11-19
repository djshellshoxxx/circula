#!/usr/bin/env python3
"""Build a source-pinned, read-only branch inventory from local Git evidence.

Inputs are explicit snapshots, not a potentially stale tracking-ref listing.
This tool writes reports only; merge-tree writes unreferenced Git objects but
never changes a branch, index or checkout. No historical tests are rerun here.
"""
import argparse
import collections
import copy
import datetime
import hashlib
import json
import pathlib
import re
import subprocess
import sys


def compact_inventory(expanded):
    """Normalize repeated evidence without dropping any original fact.

    Schema 2 stores commit metadata once by SHA, comparisons once by their
    source/target pair, changed-file lists once, and ancestry edges as pairs of
    branch names. expand_inventory reconstructs schema 1 exactly.
    """
    result = copy.deepcopy(expanded)
    if result["schema_version"] != 1:
        raise ValueError("expected expanded schema 1")
    comparisons, commits, file_sets, file_keys = {}, {}, {}, {}
    for row in result["remote_branches"] + result["recovered_local_branches"]:
        for name, base in (("against_main", result["base_sha"]),
                           ("against_candidate", result["candidate_sha"])):
            key = base + "..." + row["sha"]
            comparison = row[name]
            for item in comparison["unique_commits"]:
                if item["sha"] in commits and commits[item["sha"]] != item:
                    raise ValueError("inconsistent commit metadata")
                commits[item["sha"]] = item
            if key in comparisons and comparisons[key] != comparison:
                raise ValueError("inconsistent comparison metadata")
            comparisons[key] = comparison
            row[name] = key
    for comparison in comparisons.values():
        comparison["unique_commits"] = [item["sha"] for item in comparison["unique_commits"]]
    for pr in result["pull_requests"]:
        for item in pr["unique_commits_outside_main"]:
            if item["sha"] in commits and commits[item["sha"]] != item:
                raise ValueError("inconsistent PR commit metadata")
            commits[item["sha"]] = item
        pr["unique_commits_outside_main"] = [item["sha"] for item in pr["unique_commits_outside_main"]]
    for item in list(comparisons.values()) + result["pull_requests"]:
        paths = item["changed_files_since_merge_base"]
        key = json.dumps(paths, separators=(",", ":"))
        if key not in file_keys:
            file_keys[key] = str(len(file_sets))
            file_sets[file_keys[key]] = paths
        item["changed_files_since_merge_base"] = file_keys[key]
    nodes = {row["branch"]: row["sha"] for row in result["remote_branches"]}
    for edge in result["dependencies"]:
        if edge != dict(source=edge["source"], target=edge["target"],
                        source_sha=nodes[edge["source"]], target_sha=nodes[edge["target"]],
                        relation="ancestor"):
            raise ValueError("dependency cannot be represented losslessly")
    result["dependencies"] = [[edge["source"], edge["target"]] for edge in result["dependencies"]]
    result.update(schema_version=2, expanded_schema_version=1, comparisons=comparisons,
                  commits=commits, file_sets=file_sets, dependency_node_shas=nodes,
                  dependency_relation="ancestor")
    return result


def expand_inventory(compact):
    """Restore the complete, human-readable schema 1 evidence structure."""
    result = copy.deepcopy(compact)
    if result["schema_version"] == 1:
        return result
    if result["schema_version"] != 2 or result["expanded_schema_version"] != 1:
        raise ValueError("unsupported inventory schema")
    comparisons, commits = result.pop("comparisons"), result.pop("commits")
    file_sets = result.pop("file_sets")
    for comparison in comparisons.values():
        comparison["unique_commits"] = [copy.deepcopy(commits[sha]) for sha in comparison["unique_commits"]]
        comparison["changed_files_since_merge_base"] = copy.deepcopy(file_sets[comparison["changed_files_since_merge_base"]])
    for row in result["remote_branches"] + result["recovered_local_branches"]:
        row["against_main"] = copy.deepcopy(comparisons[row["against_main"]])
        row["against_candidate"] = copy.deepcopy(comparisons[row["against_candidate"]])
    for pr in result["pull_requests"]:
        pr["unique_commits_outside_main"] = [copy.deepcopy(commits[sha]) for sha in pr["unique_commits_outside_main"]]
        pr["changed_files_since_merge_base"] = copy.deepcopy(file_sets[pr["changed_files_since_merge_base"]])
    nodes, relation = result.pop("dependency_node_shas"), result.pop("dependency_relation")
    result["dependencies"] = [dict(source=a, target=b, source_sha=nodes[a], target_sha=nodes[b], relation=relation)
                              for a, b in result["dependencies"]]
    result["schema_version"] = result.pop("expanded_schema_version")
    return result


def main():
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--repo", type=pathlib.Path, required=True)
    ap.add_argument("--refs", type=pathlib.Path, required=True)
    ap.add_argument("--prs", type=pathlib.Path, required=True)
    ap.add_argument("--runs", type=pathlib.Path, nargs="+", required=True)
    ap.add_argument("--base", required=True)
    ap.add_argument("--candidate", required=True)
    ap.add_argument("--output", type=pathlib.Path, required=True)
    args = ap.parse_args()
    args.output.mkdir(parents=True, exist_ok=True)

    def git(*argv, allow=(0,)):
        p = subprocess.run(["git", "-C", str(args.repo), *argv], text=True,
                           stdout=subprocess.PIPE, stderr=subprocess.PIPE, check=False)
        if p.returncode not in allow:
            raise RuntimeError(f"git {argv!r}: exit {p.returncode}: {p.stderr}")
        return p.returncode, p.stdout

    def out(*argv):
        return git(*argv)[1].rstrip("\n")

    def lines(*argv):
        return [x for x in out(*argv).splitlines() if x]

    def ancestor(a, b):
        return git("merge-base", "--is-ancestor", a, b, allow=(0, 1))[0] == 0

    base, candidate = out("rev-parse", args.base), out("rev-parse", args.candidate)
    source_refs = [line.split("\t", 1) for line in args.refs.read_text().splitlines()]
    remote_refs = [(ref.removeprefix("refs/heads/"), sha) for sha, ref in source_refs
                   if ref.startswith("refs/heads/")]
    prs = json.loads(args.prs.read_text())
    runs = {r["id"]: r for path in args.runs for r in json.loads(path.read_text())}
    prior_path = args.repo / "docs/audits/BRANCH_CONSOLIDATION_MATRIX.md"
    prior = prior_path.read_text()
    prior_rows = {}
    for line in prior.splitlines():
        if not line.startswith("| `"):
            continue
        cells = re.split(r"(?<!\\)\|", line)[1:-1]
        match = re.match(r"\s*`([^`]+)`.*?/commit/([0-9a-f]{40})", cells[0])
        if match and len(cells) == 6:
            prior_rows[match[1]] = dict(sha=match[2], purpose=cells[1].strip(),
                                      prior_unique_work=cells[2].strip(), prior_conflicts=cells[3].strip(),
                                      prior_readiness=cells[4].strip(), prior_action=cells[5].strip())

    commit_cache, compare_cache, reach_cache = {}, {}, {}

    def commit(sha):
        if sha not in commit_cache:
            fields = out("show", "-s", "--format=%H%x00%P%x00%cI%x00%s", sha).split("\0")
            commit_cache[sha] = dict(sha=fields[0], parents=fields[1].split(),
                                     committed_at=fields[2], subject=fields[3])
        return commit_cache[sha]

    def reach(sha):
        if sha not in reach_cache:
            reach_cache[sha] = set(lines("rev-list", sha))
        return reach_cache[sha]

    def compare(target, sha):
        if (target, sha) in compare_cache:
            return compare_cache[target, sha]
        merge_base = out("merge-base", target, sha)
        behind, ahead = map(int, out("rev-list", "--left-right", "--count", f"{target}...{sha}").split())
        cherry = lines("cherry", target, sha)
        files = []
        for line in lines("diff", "--name-status", "--no-renames", merge_base, sha):
            status, path = line.split("\t", 1)
            files.append(dict(status=status, path=path))
        if ahead:
            rc, simulation = git("merge-tree", "--write-tree", "--name-only", target, sha, allow=(0, 1))
            first, _, rest = simulation.partition("\n")
            conflict_paths = rest.split("\n\n", 1)[0].splitlines() if rc else []
            delta = lines("diff", "--name-only", target, first) if not rc else None
            merge = dict(status="clean" if rc == 0 else "conflicts", tree=first,
                         conflict_paths=conflict_paths, resulting_changed_paths=delta)
        else:
            merge = dict(status="already_ancestor", tree=out("rev-parse", target+"^{tree}"),
                         conflict_paths=[], resulting_changed_paths=[])
        value = dict(merge_base=merge_base, behind=behind, ahead=ahead, ancestor=(ahead == 0),
                     unique_commits=[commit(s) for s in lines("rev-list", "--reverse", f"{target}..{sha}")],
                     unique_patch_commits=[s[2:] for s in cherry if s.startswith("+ ")],
                     equivalent_patch_commits=[s[2:] for s in cherry if s.startswith("- ")],
                     changed_files_since_merge_base=files,
                     changed_test_files=[x["path"] for x in files if re.search(r"(_test\.go$|(^|/)e2e/|\.spec\.|(^|/)test[^/]*[/.])", x["path"])],
                     source_diff_sha256=hashlib.sha256(git("diff", "--binary", merge_base, sha)[1].encode()).hexdigest(),
                     merge_simulation=merge)
        compare_cache[target, sha] = value
        return value

    def classify(name, sha, bm, cm, source):
        if bm["ancestor"]:
            return "already_integrated_main", "No commit recovery needed. Preserve source ref."
        if cm["ancestor"]:
            return "integrated_candidate", "Already in PR #90 ancestry. Qualify repaired combined candidate."
        if out("rev-parse", sha+"^{tree}") == out("rev-parse", candidate+"^{tree}"):
            return "duplicate_candidate_tree", "Exact candidate tree with alternate local commit provenance. Preserve."
        if "bounded-session" in name:
            return "requires_cherry_pick_and_qualification", "Recover 853a388 and adapter completion; review and execute migration, service, API, browser and race suites."
        if any(x in name for x in ("gov021", "gov-021", "sec026", "sec-026", "aud4-f02", "future-cli-specs")):
            return "requires_product_decision", "Retain isolated; authority, privacy, interface or governance policy remains unresolved."
        if source == "recovered_local" and name == "audit/ledger-features-20261119":
            return "requires_product_decision", "Mixed GOV-021 and SEC-026 local topic; never merge the generic-name branch wholesale into core."
        if name.startswith("future/") or "/future-" in name or "fnet-204" in name or "feed-signed-cursors" in name or "favorites-owner-scoped" in name:
            return "future_only", "Preserve separate module path; never merge module runtime into core."
        if cm["merge_simulation"]["status"] == "clean" and not cm["merge_simulation"]["resulting_changed_paths"]:
            return "duplicate_content", "Clean merge produces no candidate tree changes. Preserve without redundant source integration."
        if not cm["unique_patch_commits"]:
            return "duplicate_patch", "All nonmerge patches equivalent to candidate; preserve later candidate edits."
        old = prior_rows.get(name)
        if old:
            return "superseded_or_recovered", old["prior_action"]
        if source == "recovered_local":
            return "superseded_local_topic", "Source changes are represented by reviewed publication commits in PR90; do not reimport older files or stale status notes."
        return "manual_review_required", "No inherited disposition; inspect complete source changes before integration."

    def row(name, sha, source):
        bm, cm = compare(base, sha), compare(candidate, sha)
        category, action = classify(name, sha, bm, cm, source)
        related = [p for p in prs if p["head"] == name or p["sha"] == sha]
        latest_by_workflow = {}
        for r in sorted((r for r in runs.values() if r["head_sha"] == sha), key=lambda x:x["created_at"], reverse=True):
            latest_by_workflow.setdefault(r["name"], r)
        old = prior_rows.get(name)
        return dict(branch=name, sha=sha, tree=out("rev-parse", sha+"^{tree}"), source=source,
                    tip=commit(sha), classification=category, action=action,
                    snapshot_change="new" if not old else "unchanged" if old["sha"] == sha else "advanced",
                    prior_disposition=old, against_main=bm, against_candidate=cm,
                    pull_requests=[{k:p.get(k) for k in ("number", "state", "title", "head", "sha", "base", "base_sha", "merged_at", "merge_commit_sha", "url")} for p in related],
                    ci={"latest_exact_head_workflows":list(latest_by_workflow.values()),
                        "coverage": "exact-head provider workflows observed" if latest_by_workflow else "no exact-head workflow record in 564-run repository history snapshot; not a pass",
                        "historical_test_suites_rerun_in_inventory":False})

    rows = [row(name, sha, "remote") for name, sha in remote_refs]
    local = []
    for line in lines("for-each-ref", "--format=%(refname:strip=3)%09%(objectname)", "refs/remotes/recovered-local/"):
        name, sha = line.split("\t", 1)
        local.append(row(name, sha, "recovered_local"))
    # Every PR head, including deleted remote branches and closed unmerged PRs,
    # remains independently addressable in the fetched pull/*/head refs.
    pr_rows = []
    for p in prs:
        item = {k:p.get(k) for k in ("number", "state", "title", "head", "sha", "base", "base_sha", "merged_at", "merge_commit_sha", "url")}
        item["head_present_in_remote_snapshot"] = any(name == p["head"] and sha == p["sha"] for name, sha in remote_refs)
        item["head_ancestor_main"] = ancestor(p["sha"], base)
        item["head_ancestor_candidate"] = ancestor(p["sha"], candidate)
        item["changed_files_since_merge_base"] = compare(base, p["sha"])["changed_files_since_merge_base"]
        item["unique_commits_outside_main"] = compare(base, p["sha"])["unique_commits"]
        item["current_branch_classification"] = next((r["classification"] for r in rows if r["branch"] == p["head"]), None)
        if not item["current_branch_classification"] and p["number"] in (49, 50):
            item["current_branch_classification"] = "dependency_version_already_recovered"
        pr_rows.append(item)

    # Ancestry dependencies are exact. Keep all edges in JSON; nearest edges
    # and declared PR targets make the human graph readable.
    edges = []
    for dependent in rows:
        ancestors = [r for r in rows if r["branch"] != dependent["branch"] and r["sha"] in reach(dependent["sha"])]
        for upstream in ancestors:
            edges.append(dict(source=upstream["branch"], target=dependent["branch"],
                              source_sha=upstream["sha"], target_sha=dependent["sha"], relation="ancestor"))
        nearest = [r for r in ancestors if not any(r["sha"] != other["sha"] and r["sha"] in reach(other["sha"]) for other in ancestors)]
        dependent["nearest_ancestor_branches"] = [{"branch":r["branch"], "sha":r["sha"]} for r in nearest]
        dependent["all_ancestor_remote_branches"] = [r["branch"] for r in ancestors]

    result = dict(schema_version=1, generated_at_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                  base_sha=base, candidate_sha=candidate, remote_refs_source=args.refs.name,
                  remote_refs_sha256=hashlib.sha256(args.refs.read_bytes()).hexdigest(),
                  limitations=["Git merge simulation is not a test pass or approval.",
                               "Patch-ID equivalence does not establish semantic or squash equivalence.",
                               "Provider workflow inventory is a source-pinned snapshot, not the status of later commits.",
                               "Historical tests were not rerun by this inventory.",
                               "Local recovered refs are separate from the count of real remote branches."],
                  counts=dict(remote_branches=len(rows), recovered_local_refs=len(local), pull_requests=len(prs),
                              closed_unmerged_prs=sum(p["state"] == "closed" and not p["merged_at"] for p in prs),
                              tags=sum(ref.startswith("refs/tags/") for _, ref in source_refs),
                              provider_workflow_records=len(runs),
                              classifications=dict(collections.Counter(r["classification"] for r in rows))),
                  remote_branches=rows, recovered_local_branches=local, pull_requests=pr_rows,
                  dependencies=edges,
                  declared_pr_targets=[dict(number=p["number"],source=p["head"],source_sha=p["sha"],target=p["base"],target_sha=p["base_sha"],state=p["state"]) for p in prs])
    path = args.output / "BRANCH_INVENTORY_2026-11-19.json"
    compact = compact_inventory(result)
    if expand_inventory(compact) != result:
        raise AssertionError("inventory normalization lost evidence")
    path.write_text(json.dumps(compact, separators=(",", ":"))+"\n")
    (args.output / "BRANCH_REFS_2026-11-19.tsv").write_text(args.refs.read_text())
    print(json.dumps({"output":str(path), "counts":result["counts"],
                      "new_or_advanced":[dict(branch=r["branch"],sha=r["sha"],change=r["snapshot_change"]) for r in rows if r["snapshot_change"] != "unchanged"]}, indent=2))


if __name__ == "__main__":
    if len(sys.argv) == 3 and sys.argv[1] == "--expand-json":
        print(json.dumps(expand_inventory(json.loads(pathlib.Path(sys.argv[2]).read_text())), indent=2))
    else:
        main()

# Pending-CI progress guard addendum

Independent read-only source review approved the narrow progress-handling
change in `finalize_consolidation_evidence.py`, SHA-256
`cd788d2eb58139405c0e7ba82356074f7247eb9bd56346faf51f34b208b8bd7c`.
The retained `evidence/report-generator.py.txt` has the same hash.

The existing completed-merge proof requirements remain in place: exact PR,
head and run identities, completed-success provider run, unique named jobs,
and all seven or nine required jobs completed successfully. The pending mode
still keeps the integrated main and locally qualified candidate separate.

For each open core candidate (#82, #84 and #87), the revised guard preserves
the exact reviewed head/run pins and unique required job names. Each required
job must be either completed successfully or queued/in progress with a null
conclusion. Failed, cancelled, skipped and unknown required-job outcomes are
rejected. At least one required job must remain incomplete for every pending
PR; a fully successful PR cannot be retained through this pending-mode path.

The retained, manifest-hashed provider snapshot at
`2026-11-18T10:24:32.909Z` supports four passed, two running and fifteen queued
required jobs across those three PRs. The counts are derived from job status,
so a stale provider run summary of `queued` cannot hide jobs already running.
The additional skipped placeholder `test (${{ matrix.os }})` is outside the
required named-job set; no required skipped job is accepted.

The generated README and current HANDOFF identify a pending checkpoint, name
the dated job totals, and preserve the main/candidate distinction. The remaining
gates document accurately identifies #84's four passed/two running/one queued
jobs and seven queued jobs for each of the other four reviewed open PRs at
that observation time. Earlier wholly unassigned observations remain dated
historical evidence. No cause or later execution outcome is inferred.

No additional correction was found in this narrow source-and-wording review.
No publication artifact was edited and no test, scanner, generator or live
provider query was run. The separate representation addendum records the
successful seven-transform verification and corrected PR #84 prose.

# Recurring upstream reuse matrix

| Component | Pin | Reused portion | Changes | License/notice evidence |
|---|---|---|---|---|
| [robfig/cron](https://github.com/robfig/cron) | v3.0.1, ccba498c397bb90a9c84945bbb0f7af2d72b6309 | NewParser and Schedule.Next for bounded structured calendars | None; imported module; no runner started | MIT, third_party/github_com_robfig_cron_v3/LICENSE |
| [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite) | v1.60.1 | Synthetic standalone SQL tests only | None; runtime library accepts caller-owned database/sql connection | Exact graph notices retained in THIRD_PARTY_NOTICES.md and third_party/ |

The continuation adds no dependency or source transplant. Module graph checksums are in go.sum. Runtime scheduling policy, authorization, transaction atomicity and lifecycle rules remain native code; upstream parser reuse does not qualify these. Original research and maintenance limitations remain in UPSTREAM-RESEARCH.md. Production license approval is not inferred.

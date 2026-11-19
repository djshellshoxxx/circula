# Master consolidation branch inventory

Starting main: `ead63b5ce24b2bcf8352d75fd0d1ff7be711f625`. Live remote snapshot: **114 branches**, **0 tag refs**. **96 pull-request metadata records** examined.

API-036 published integration advanced to `1e2f369a541963540d93c3b7f455b052827839c7` during discovery. Its separate cursor-hardening branch and unpublished local worktrees are protected. FNET-204 remains exclusively owned.

## Current disposition

| Classification | Branches |
|---|---:|
| Already integrated | 52 |
| Blocked by active worker | 11 |
| Blocked by policy or design decision | 5 |
| Duplicate or superseded | 29 |
| Experimental future module | 17 |

No additional independently eligible Core branch was found. Existing security, dispute, audit and documentation candidates are either already contained in main or carried by the active PR90 stream. Future modules remain isolated; unresolved governance and signal policy is not inferred from broad engineering authorization.

## Method and evidence limits

The companion JSON records every exact head, merge base, ahead/behind count, unique commit, changed path, changed test, complete-diff digest, tree identity, merge simulation, patch-equivalence count, observed Actions runs, PR association and active-worker overlap. Historical semantic assessments retain their own source SHA and an explicit current-head match flag.

Actions observations cover the first page of PR-triggered workflow runs for each starting head. An empty response does not mean passing CI. Successful historical runs do not qualify an integration commit or satisfy unknown approvals/protections. No runtime test suite was executed by this inventory task.

Merge simulation creates Git objects but does not check out, merge, commit, update, delete or rewrite any branch. No original branch or existing worker file was changed.

## Remote branches

| Branch | Exact head | Ahead / behind | Disposition | Observed PR workflow |
|---|---|---:|---|---|
| `audit/future-feed-cursors-20261119` | `8638f3d30f27f4882c4c6424d4c8b248208b6f3d` | 38 / 0 | Blocked by active worker | [44491232322](https://github.com/djshellshoxxx/circula/actions/runs/44491232322): queued; [44491232304](https://github.com/djshellshoxxx/circula/actions/runs/44491232304): success |
| `audit/future-notifications-binding-20261119` | `9c794a5b9e6ff7aa4eaa57c6e8fb5e2b145307ae` | 37 / 0 | Experimental future module | [44491239020](https://github.com/djshellshoxxx/circula/actions/runs/44491239020): queued; [44491239025](https://github.com/djshellshoxxx/circula/actions/runs/44491239025): success |
| `audit/future-pwa-browser-20261119` | `f25d037c3cc53fbf1ebbf687831300be9bdb749f` | 39 / 0 | Experimental future module | [44496951449](https://github.com/djshellshoxxx/circula/actions/runs/44496951449): queued; [44496951525](https://github.com/djshellshoxxx/circula/actions/runs/44496951525): queued |
| `audit/gov021-fixture-20261119` | `d922d4f880059f939aeca40cbcb77c60201ff0c7` | 9 / 15 | Duplicate or superseded | [44486475798](https://github.com/djshellshoxxx/circula/actions/runs/44486475798): success |
| `audit/sec026-independent-20261119` | `26f350b8aa437d315c993c5555cb0b9fa2b7eba4` | 33 / 0 | Blocked by policy or design decision | [44491136235](https://github.com/djshellshoxxx/circula/actions/runs/44491136235): in_progress |
| `chatgpt/circula-consolidation-review-20261111` | `02f9f0a7bb87f54611e2fe0b0837a519d2bcca9e` | 22 / 187 | Duplicate or superseded | [43167493278](https://github.com/djshellshoxxx/circula/actions/runs/43167493278): failure |
| `chatgpt/circula-departure-pagination-20261115` | `8abff32c1b4a30507497d1e43dc4f88d0c7e6eab` | 4 / 155 | Duplicate or superseded | [43786301780](https://github.com/djshellshoxxx/circula/actions/runs/43786301780): failure |
| `chatgpt/circula-erasure-notification-20261115` | `26b57163d96534e79e3d35d471573d609af8a415` | 1 / 155 | Duplicate or superseded | [43786018052](https://github.com/djshellshoxxx/circula/actions/runs/43786018052): failure |
| `chatgpt/circula-expiry-starvation-20261115` | `2eacf7718f5bb3da3be5a288024af879076a9597` | 0 / 151 | Already integrated | No PR-triggered run observed |
| `chatgpt/circula-ledger-auth-guard-20261115` | `29e46e5016a05a0b633a4cb23cab6a4dea38e1ca` | 3 / 155 | Duplicate or superseded | [43785743515](https://github.com/djshellshoxxx/circula/actions/runs/43785743515): failure |
| `chatgpt/circula-mfa-decider-20261115` | `4ffb5c78d6b44e6f964f1018ed2fea16cd742abd` | 4 / 155 | Duplicate or superseded | [43785989156](https://github.com/djshellshoxxx/circula/actions/runs/43785989156): failure |
| `chatgpt/circula-qa042-traceability-audit-20261114` | `fe832faced826ed2234fcd9ba0ce7f7321f04df5` | 37 / 161 | Duplicate or superseded | [43663449911](https://github.com/djshellshoxxx/circula/actions/runs/43663449911): success |
| `chatgpt/circula-secure-json-write-20261115` | `e02085a6aa1a1394f874b304fdf0961cc0888423` | 8 / 156 | Duplicate or superseded | [43785303060](https://github.com/djshellshoxxx/circula/actions/runs/43785303060): failure |
| `chatgpt/circula-security-audit-20261115` | `36a80335abf54ec8d965cb350b157443ef7f2fec` | 45 / 161 | Duplicate or superseded | [43712392795](https://github.com/djshellshoxxx/circula/actions/runs/43712392795): success |
| `chatgpt/consolidation-delta-qualification-20261114` | `8f742877fb53cc39aea5e189a5452d924d8c08c8` | 1 / 161 | Duplicate or superseded | [43580764126](https://github.com/djshellshoxxx/circula/actions/runs/43580764126): success |
| `chatgpt/favorites-owner-scoped-collection-20261116` | `372a8f97ae01c0af3b4a96c4a5ce8b1bcc75cd8d` | 4 / 151 | Duplicate or superseded | [44011276748](https://github.com/djshellshoxxx/circula/actions/runs/44011276748): failure |
| `chatgpt/feed-signed-cursors-20261116` | `7a528c611bffdf92ef9f8cbddb98e8b088eb5036` | 4 / 151 | Blocked by active worker | [44011402381](https://github.com/djshellshoxxx/circula/actions/runs/44011402381): failure |
| `chatgpt/zero-codex-continuation-spec-20261109` | `3d9c56196625fc0c158b4652cbe2b9716c7af97b` | 1 / 466 | Duplicate or superseded | No PR-triggered run observed |
| `chatgpt/zero-codex/audit-20261110-a` | `2e729346d6297075b6ff1f7828d48681d48320b6` | 0 / 246 | Already integrated | [42934807230](https://github.com/djshellshoxxx/circula/actions/runs/42934807230): cancelled |
| `chatgpt/zero-codex/ci-macos-race-budget-20261110` | `2078bf0964353892e80214b4601211e36818b889` | 0 / 255 | Already integrated | [42880245458](https://github.com/djshellshoxxx/circula/actions/runs/42880245458): success |
| `chatgpt/zero-codex/ci-macos15-retirement-20261110` | `1760551f5b03a89ec64d09269d631e72f2e95959` | 0 / 256 | Already integrated | [42869647736](https://github.com/djshellshoxxx/circula/actions/runs/42869647736): failure |
| `chatgpt/zero-codex/cookie-policy-adapter-20261110-b` | `7e8481004801c01aea5d22f4720e1ead7381c4f7` | 0 / 251 | Already integrated | [42934571187](https://github.com/djshellshoxxx/circula/actions/runs/42934571187): success |
| `chatgpt/zero-codex/decidable-expiry-20261110-a` | `b61a8e719030ff56044d71fee632b5ba824d7698` | 0 / 253 | Already integrated | [42908897181](https://github.com/djshellshoxxx/circula/actions/runs/42908897181): success |
| `chatgpt/zero-codex/docs-20261110-a` | `a376a4ed592451d90d2ff7b4d3db1489a2231f96` | 0 / 227 | Already integrated | [42934941672](https://github.com/djshellshoxxx/circula/actions/runs/42934941672): cancelled |
| `chatgpt/zero-codex/qualify-pr19-20261110-a` | `63b033d842a2fa40b56ff9d79e69c259db951c83` | 0 / 245 | Already integrated | [42935092231](https://github.com/djshellshoxxx/circula/actions/runs/42935092231): cancelled |
| `chatgpt/zero-codex/terms-return-20261110-b` | `7620d7f51913363dd013b80cc682eb670eeff3a5` | 0 / 257 | Already integrated | [42838956843](https://github.com/djshellshoxxx/circula/actions/runs/42838956843): cancelled |
| `checkpoint/circula-handoff-20261025` | `3d0080a9c43680f69f81217af3b42374e7160bf8` | 6 / 320 | Duplicate or superseded | No PR-triggered run observed |
| `claude/aud4-auth-fixes` | `82dc18e07f6b22b9c34ef026f833391705b30067` | 9 / 0 | Blocked by active worker | [44288786538](https://github.com/djshellshoxxx/circula/actions/runs/44288786538): success |
| `claude/aud4-f02-reviewer-reset-policy` | `af54887aa0cd63ba080e2944aa52a5b759b008ca` | 2 / 15 | Blocked by policy or design decision | [44273207384](https://github.com/djshellshoxxx/circula/actions/runs/44273207384): success |
| `claude/aud4-f03-dispute-timelines` | `b6275747df66ede81ee4267e2cd6a2595f3bd680` | 13 / 0 | Blocked by active worker | [44290387452](https://github.com/djshellshoxxx/circula/actions/runs/44290387452): success |
| `claude/awesome-darwin-y6eaiz` | `827e651fe151f49659cc20aefe85d4af67bdaa97` | 0 / 300 | Already integrated | No PR-triggered run observed |
| `claude/backup-path-hardening` | `09ef8a5aa25f11ee4d96f043e7d686e97598b7e3` | 0 / 12 | Already integrated | [44264463917](https://github.com/djshellshoxxx/circula/actions/runs/44264463917): success |
| `claude/charming-gauss-19fbfa` | `8b4c8ba23c99d4418348d76ff8cb38a72e7b958d` | 0 / 161 | Already integrated | [43784908906](https://github.com/djshellshoxxx/circula/actions/runs/43784908906): success; [43157071772](https://github.com/djshellshoxxx/circula/actions/runs/43157071772): success |
| `claude/circula-beta-completion-1npjw6` | `bffcede8c7dca0d702e1a5cd5b5e8b5d363c7c3b` | 0 / 168 | Already integrated | [42966603001](https://github.com/djshellshoxxx/circula/actions/runs/42966603001): failure |
| `claude/circula-build` | `fc488445af5f43165a7103f1c1fd12483b39d448` | 0 / 430 | Already integrated | [39597594942](https://github.com/djshellshoxxx/circula/actions/runs/39597594942): success |
| `claude/circula-completeness` | `fcd9fc5878482e1e4d340a3448a0d2294a33eb5b` | 1 / 340 | Duplicate or superseded | [42934849953](https://github.com/djshellshoxxx/circula/actions/runs/42934849953): cancelled |
| `claude/circula-consolidation-audit-0sydm1` | `0e5296a71f3f3802f36e14d847c0abf7b1dbf882` | 10 / 0 | Blocked by active worker | [44288798706](https://github.com/djshellshoxxx/circula/actions/runs/44288798706): success |
| `claude/circula-finish` | `d350474e49522c34213d90a66904b296b9509f6c` | 0 / 423 | Already integrated | [39601241527](https://github.com/djshellshoxxx/circula/actions/runs/39601241527): success |
| `claude/circula-pr-consolidation-q6q18n` | `163fe1596bd7cfb8c2a22891629fc4459e6b639c` | 0 / 40 | Already integrated | [44226365456](https://github.com/djshellshoxxx/circula/actions/runs/44226365456): success |
| `claude/circula-security-features` | `f8278a047392541a077d57c560823471053d1e1d` | 0 / 438 | Already integrated | [39594488621](https://github.com/djshellshoxxx/circula/actions/runs/39594488621): success |
| `claude/compassionate-cannon-txdlz1` | `047e92e88a0ccf7ed8b4c6ed1a1df78a90cc523a` | 6 / 56 | Duplicate or superseded | [44029146812](https://github.com/djshellshoxxx/circula/actions/runs/44029146812): success |
| `claude/cool-gauss-vdxlod` | `64fa4f685780b93a349a862e3741df5b4db067cc` | 0 / 144 | Already integrated | [43876355471](https://github.com/djshellshoxxx/circula/actions/runs/43876355471): failure |
| `claude/cool-ramanujan-n2peag` | `c033cfa448fb89d1a5cc0ca87458b40d2709fa2e` | 40 / 161 | Duplicate or superseded | [43685188544](https://github.com/djshellshoxxx/circula/actions/runs/43685188544): success |
| `claude/fnet-204-signed-cursors` | `0952c94a5dbab4044d7673ce9d51cec9063ab97f` | 5 / 151 | Blocked by active worker | [44261331550](https://github.com/djshellshoxxx/circula/actions/runs/44261331550): failure |
| `claude/gov-021-admin-recovery` | `0d480a5b348f3d385df0dc9c9a3e43a0b03a8332` | 10 / 15 | Blocked by policy or design decision | No PR-triggered run observed |
| `claude/gracious-ptolemy-dscc7v` | `1d165989e2176e214536dd22b05f7b3a4cb2f48f` | 3 / 254 | Duplicate or superseded | [42966428723](https://github.com/djshellshoxxx/circula/actions/runs/42966428723): success |
| `claude/happy-babbage-dcumwz` | `dc97f7d5b0ae64306a8e2b1e57ec83f233d3d09d` | 0 / 98 | Already integrated | No PR-triggered run observed |
| `claude/happy-noether-444cum` | `34ccb81a54ab63875d4a5305076437d6b3221049` | 0 / 255 | Already integrated | [42938376400](https://github.com/djshellshoxxx/circula/actions/runs/42938376400): cancelled |
| `claude/kind-einstein-v5ycss` | `3bfb2f78fb0aad03ec3533af5985c394d64c1898` | 1 / 466 | Duplicate or superseded | No PR-triggered run observed |
| `claude/lucid-archimedes-upisa8` | `d2c1c6090a053b62dc75002464dc96bb542d6c78` | 15 / 466 | Duplicate or superseded | No PR-triggered run observed |
| `claude/quirky-babbage-y9fpmm` | `af69702be08aaf14cbe9a705ddf7f08796d5e976` | 41 / 161 | Duplicate or superseded | No PR-triggered run observed |
| `claude/quirky-einstein-l9ni8m` | `4a8db9e80509d23d3937c68575c632f4650576aa` | 0 / 110 | Already integrated | [43793810240](https://github.com/djshellshoxxx/circula/actions/runs/43793810240): success |
| `claude/relaxed-hopper-8q4vhq` | `2da1a293ebf2f28f349fcae9bf32184737bc8619` | 22 / 466 | Duplicate or superseded | No PR-triggered run observed |
| `claude/sec-026-abuse-signals` | `4d349c4d03bfb81760d471a2a7ef4ac4338d9c45` | 13 / 9 | Blocked by policy or design decision | [44274333610](https://github.com/djshellshoxxx/circula/actions/runs/44274333610): failure |
| `claude/sharp-archimedes-tu76rh` | `eb4bb08b15d41295aa2c4b9f93040124ce4d66e9` | 0 / 7 | Already integrated | [44272452596](https://github.com/djshellshoxxx/circula/actions/runs/44272452596): success |
| `claude/sweet-turing-m3ftja` | `01b6e5c7cfc5bd9d660b80aaafe2b35de3912904` | 13 / 466 | Duplicate or superseded | No PR-triggered run observed |
| `claude/ux-047-auth-catalogue` | `dd3e82e27ddb3d1cbf03460e6a5fc879f16691db` | 0 / 28 | Already integrated | [44260832164](https://github.com/djshellshoxxx/circula/actions/runs/44260832164): success |
| `claude/vibrant-brahmagupta-3uofnj` | `33b200e00c088c86ffb0d0d8176f1575376a9b49` | 0 / 465 | Already integrated | No PR-triggered run observed |
| `codex/api036-cursor-hardening-20261119` | `81f203555e562bc45d86a53341b9b5c44d0b550f` | 32 / 0 | Blocked by active worker | [44502959201](https://github.com/djshellshoxxx/circula/actions/runs/44502959201): queued |
| `codex/circula-checkpoint-recovery-20261116` | `12175ec348d5234c27e1c72dc8df96a091cd111c` | 0 / 75 | Already integrated | [43930580504](https://github.com/djshellshoxxx/circula/actions/runs/43930580504): success |
| `codex/circula-common-ui-catalog-20261116` | `a112c979932bc7b9e209455d3df14dfebc60243a` | 0 / 68 | Already integrated | [43963324717](https://github.com/djshellshoxxx/circula/actions/runs/43963324717): success |
| `codex/circula-consolidate-20261030` | `7643fb616208cc318939ded809e6cada66bb7415` | 0 / 285 | Already integrated | [41047238538](https://github.com/djshellshoxxx/circula/actions/runs/41047238538): success |
| `codex/circula-core-checkpoint-20261116` | `15375a8dc87544849b4a85fa128ce77be1103814` | 0 / 88 | Already integrated | No PR-triggered run observed |
| `codex/circula-core-completion-20261116` | `e3c60a62c065567ff67fc2e219a724b6fd91ca76` | 0 / 86 | Already integrated | [43907084340](https://github.com/djshellshoxxx/circula/actions/runs/43907084340): success |
| `codex/circula-evidence-20261116` | `f6ea3213e663c7e37d678b018861b6703947e3e0` | 0 / 80 | Already integrated | [43922903011](https://github.com/djshellshoxxx/circula/actions/runs/43922903011): success |
| `codex/circula-financial-messages-20261116` | `cf5abf5cfb00638e1772a7e34014a352048d6b35` | 0 / 70 | Already integrated | [43952719310](https://github.com/djshellshoxxx/circula/actions/runs/43952719310): success |
| `codex/circula-integration-plan-20261030` | `127943ba7aa2cb154d3e65d13279e739b901416f` | 1 / 461 | Duplicate or superseded | No PR-triggered run observed |
| `codex/circula-message-catalogs-20261116` | `96059bb4b8ae23ad6e4e45428bf69288bec4c182` | 0 / 72 | Already integrated | [43942206927](https://github.com/djshellshoxxx/circula/actions/runs/43942206927): success |
| `codex/circula-noscript-qa` | `ba57512918195423b25bc7958ff0b8af6df7ed7a` | 0 / 269 | Already integrated | No PR-triggered run observed |
| `codex/circula-qualification-20261116` | `dc89212a8bf1e1207e5443ffc719670623de70e1` | 0 / 82 | Already integrated | [43914378342](https://github.com/djshellshoxxx/circula/actions/runs/43914378342): success |
| `codex/circula-responsive-qa` | `04e11989b811b4541a20b4e06e7a020beabb6af3` | 0 / 279 | Already integrated | No PR-triggered run observed |
| `codex/circula-resume-20261024` | `f4bfe6bc3f81c94b859abeae902ae862ea6e143e` | 0 / 303 | Already integrated | No PR-triggered run observed |
| `codex/circula-staging-refresh-docs` | `9c0bc500cb19f11f18694c76d134f4bfebf896e7` | 0 / 283 | Already integrated | No PR-triggered run observed |
| `codex/circula-support-boundaries-20261028` | `837d51b0d5374822b28f1b3a9206f9e2d23bd60c` | 0 / 305 | Already integrated | No PR-triggered run observed |
| `codex/dashboard-amounts-meter-20261116` | `d53196cc6907f27222661246a7767bb0aa123a76` | 1 / 67 | Duplicate or superseded | [43981603571](https://github.com/djshellshoxxx/circula/actions/runs/43981603571): success |
| `consolidation/http-persistence-regressions-20261116` | `050798c9cf94598efd0fff15f7a8a74bb3c4f4a1` | 0 / 61 | Already integrated | [43982223745](https://github.com/djshellshoxxx/circula/actions/runs/43982223745): failure |
| `consolidation/recover-future-specifications` | `3a776e6304551c21ff372a8ae962606108dde4ff` | 0 / 63 | Already integrated | [43981789097](https://github.com/djshellshoxxx/circula/actions/runs/43981789097): success |
| `dependabot/go_modules/github.com/getkin/kin-openapi-0.153.0` | `3fa76cdb590182eaff42fc4044cf3538275eea6b` | 0 / 155 | Already integrated | [43785332133](https://github.com/djshellshoxxx/circula/actions/runs/43785332133): success |
| `dependabot/go_modules/golang.org/x/text-0.44.0` | `4a6ea9ab3dff3bdf1f3597e98abd449b203afb11` | 1 / 56 | Duplicate or superseded | [43992058080](https://github.com/djshellshoxxx/circula/actions/runs/43992058080): failure |
| `docs/fnet-203-optional-trade-receipts` | `3d6ed55b2e44d31f7350814d87ba44564e8344c0` | 0 / 158 | Already integrated | [43784932244](https://github.com/djshellshoxxx/circula/actions/runs/43784932244): success; [43682010099](https://github.com/djshellshoxxx/circula/actions/runs/43682010099): success |
| `docs/foundational-spec-shells` | `d96f2e75634bff074d26965ae93c2fe00c6f7f24` | 1 / 466 | Duplicate or superseded | No PR-triggered run observed |
| `docs/foundational-specifications` | `22e9202f19f55aa46be4b115edbe84178e9cee0b` | 31 / 466 | Duplicate or superseded | No PR-triggered run observed |
| `docs/future-usability-network-specs-20261111` | `f4a051c1c1af815e7bb6a902929108e155145c97` | 0 / 165 | Already integrated | [42971313261](https://github.com/djshellshoxxx/circula/actions/runs/42971313261): failure |
| `docs/readme-and-wiki` | `db7826495ec2e8bd3f5459afa5f619cdc8fe8def` | 1 / 466 | Duplicate or superseded | No PR-triggered run observed |
| `future/fnet-201-matching-20261116` | `12850290c6bca7720853c33689662d751aced160` | 2 / 151 | Experimental future module | No PR-triggered run observed |
| `future/fnet-202-favorites-20261116` | `efc9d696eea2dd3260ecb6c5091802ef31e20572` | 9 / 151 | Experimental future module | No PR-triggered run observed |
| `future/fnet-203-receipts-20261116` | `bd2fbdbd3adf77b27fa12335ff2827359423aea3` | 2 / 151 | Experimental future module | No PR-triggered run observed |
| `future/fnet-204-feed-20261116` | `5d3819c3b3af031b75eceb5251da0226a4ecee77` | 2 / 151 | Blocked by active worker | No PR-triggered run observed |
| `future/fnet-205-analytics-20261116` | `0b2965101e1d5d1e3e6447e7c24197efc7f0cb9f` | 2 / 151 | Experimental future module | No PR-triggered run observed |
| `future/fnet-206-areas-20261116` | `8b444b20139e3e687db494b3fe36bb3643148e5b` | 2 / 151 | Experimental future module | No PR-triggered run observed |
| `future/fnet-207-recurring-20261116` | `0437e54897c6d9cfb3f2577807b82dd038493850` | 2 / 151 | Experimental future module | No PR-triggered run observed |
| `future/fnet-208-governance-20261116` | `b4f511372ec26dc63d005bb50d87c56e18fc1a88` | 2 / 151 | Experimental future module | No PR-triggered run observed |
| `future/fnet-209-federation-20261116` | `1b3c92e03007e35e3f190e719a1ea78a807d4102` | 2 / 151 | Experimental future module | No PR-triggered run observed |
| `future/fux-101-pwa-20261116` | `a6326b8725fe6d7270997b4a284cbbc129022dd1` | 7 / 151 | Experimental future module | No PR-triggered run observed |
| `future/fux-102-qr-20261116` | `9db8a323d7dcf430d1a58a4a4d10ca3628aeb401` | 8 / 151 | Experimental future module | No PR-triggered run observed |
| `future/fux-103-discovery-20261116` | `34d0f889631e2d133149a89a276cf9c71fd261a9` | 7 / 151 | Experimental future module | No PR-triggered run observed |
| `future/fux-104-notifications-20261116` | `3bdf8e92d84af2c660c615bcf956cf76752e86ae` | 4 / 151 | Experimental future module | No PR-triggered run observed |
| `future/fux-105-messaging-20261116` | `7f92251cdb0d6ed5d9559730045a021a173a278b` | 2 / 151 | Experimental future module | No PR-triggered run observed |
| `future/fux-106-onboarding-20261116` | `473cab2fb45a7b67ea0001f9fdc7833ffda45eb9` | 6 / 151 | Experimental future module | No PR-triggered run observed |
| `future/module-catalog-20261116` | `dfa4779d05a47bc0397110ecfbb14659ce3884d7` | 3 / 151 | Experimental future module | No PR-triggered run observed |
| `haiku/circula-consolidation` | `6fbfa13fb696d38e8e120377c0eee09af1ac44f0` | 16 / 187 | Duplicate or superseded | No PR-triggered run observed |
| `integration/admin-grant-notice-pagination-20261117` | `cb25b41febd43273cd4aa856072406b210089d81` | 0 / 38 | Already integrated | [44247617122](https://github.com/djshellshoxxx/circula/actions/runs/44247617122): success |
| `integration/backup-private-staging-20261117` | `faed4906f5419f573181bd7cf3ffdf0bdfb9992b` | 0 / 22 | Already integrated | [44256964805](https://github.com/djshellshoxxx/circula/actions/runs/44256964805): success |
| `integration/browser-parity-clock-20261117` | `2f6b3e4f1b70d8c787c4de886cf085f5a3295002` | 0 / 29 | Already integrated | [44256413287](https://github.com/djshellshoxxx/circula/actions/runs/44256413287): success |
| `integration/comprehensive-audit-20261119` | `1e2f369a541963540d93c3b7f455b052827839c7` | 32 / 0 | Blocked by active worker | [44502969014](https://github.com/djshellshoxxx/circula/actions/runs/44502969014): pending |
| `integration/consolidation-execution-record-20261118` | `73244357ca136874452a6cd826ffa91ac8c3c434` | 1 / 9 | Blocked by active worker | [44286700090](https://github.com/djshellshoxxx/circula/actions/runs/44286700090): success |
| `integration/erasure-fixture-stability-20261117` | `fc31dc9a6b8c102fe612f47bc543e94e8114df0f` | 0 / 38 | Already integrated | [44249505223](https://github.com/djshellshoxxx/circula/actions/runs/44249505223): success |
| `integration/release-threshold-check-20261117` | `ec98caba7328790ef36f0f2c144644bc05155d6b` | 0 / 38 | Already integrated | [44248576511](https://github.com/djshellshoxxx/circula/actions/runs/44248576511): success |
| `integration/store-untrusted-schema-20261117` | `b1cd8f38450ae966cfea99d59a131175d7b798dd` | 0 / 25 | Already integrated | [44256822964](https://github.com/djshellshoxxx/circula/actions/runs/44256822964): success |
| `integration/throttle-documentation-20261117` | `83de536ee373a7c4997222cf58ff563c53c8cbbc` | 0 / 38 | Already integrated | [44248706120](https://github.com/djshellshoxxx/circula/actions/runs/44248706120): success |
| `main` | `ead63b5ce24b2bcf8352d75fd0d1ff7be711f625` | 0 / 0 | Already integrated | No PR-triggered run observed |
| `nodata/doc-sweep-20261118` | `866adc8f2c30abd294843e43a2ba42a6fe5f7b67` | 3 / 0 | Blocked by active worker | [44288311142](https://github.com/djshellshoxxx/circula/actions/runs/44288311142): success |
| `planning/future-cli-specs-20261110` | `8e2372c064258fb8c53554ae8f6b81957932e885` | 1 / 466 | Blocked by policy or design decision | No PR-triggered run observed |
| `work/spec-completion` | `d39bfef12d4ac1430815fa49ad799fb1c4bf9964` | 0 / 322 | Already integrated | [39750077129](https://github.com/djshellshoxxx/circula/actions/runs/39750077129): failure |

## Active-work preservation

The JSON also records all local branches and worktrees in the three existing Circula repository families discovered before this pass. In addition to API-036 and FNET-204, storage recovery (`internal/store/migrate.go`, `internal/store/sqlite.go`, `internal/store/recovery_internal_test.go`), token inventory, shared repository interfaces, migration numbering, OpenAPI, generated traceability and the six shared audit reports were already active elsewhere. Those paths are reserved.

The separate current task owns only its new report directory and the independently assigned source/module paths listed in the master ownership registry. A clean historical merge is not permission to import another worker’s unfinished changes.

## Specific preservation decisions

- PR91’s GOV-021 fixture head is already an ancestor of its original isolated GOV-021 target. Keep the original policy hold; do not merge GOV-021 into main.
- The owner-scoped favorites follow-up is already an ancestor of the favorites module branch. Its functionality does not need another recovery merge.
- The ledger-auth, MFA-decider and dashboard-amount historical branches produce a clean merge with exactly the current main tree; they contain no missing product change.
- The old x/text update is semantically present in current dependencies. Historical merge conflicts are not evidence that its version is missing.
- Foundational draft specifications, archived QA frontiers, superseded Python traceability tools and old process prompts remain preserved in source history. Reintroducing them would duplicate or contradict the current architecture.
- All fifteen future features retain their disabled/HOLD boundary. Module-local repair is eligible only with separate ownership and verification; successful root Core tests cannot qualify nested Go modules.

Snapshot refresh is required before any final merge. This inventory does not certify release readiness or authorize deployment.

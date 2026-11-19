# Integration HOLD — FUX-105

No production integration, activation, migration registration, release or deployment is authorized. Only `future/messaging/` changes. No dependency on another future module exists.

CONTRACT.md defines the explicit Store/Authorizer and Handler/HTTPAuth boundaries. A future host adapter must resolve active network membership, conversation policy, reference visibility, role authority and retention inside the same transaction; authenticate every request; validate session-bound CSRF; expose only opt-in HTTPS routes; and record privileged operations in the host audit system without message bodies. The standalone fixture authorizer is not production authentication.

After separate feature selection, verify root contract drift, schema adoption, core ledger regression, operational backup/retention, accessibility, platform behavior and feature disablement. Do not infer approval from standalone test results. Search's POST routes must retain no-store and CSRF checks, request-body logging exclusion and tenant scope. See MIGRATION-PLAN.md for restart/restore requirements.

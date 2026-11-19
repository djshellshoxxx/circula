# Proposed isolated interface

No route below is registered in Circula. The host strips a selected prefix before invoking `Web`.

| Method/path | Behavior |
|---|---|
| GET `catalog` | Current visible named areas; HTML with selection/load/filter forms or JSON |
| GET `catalog?listing=ID` | Owner-authorized current selection/revision prefilled in native form |
| GET `assignment?listing=ID` | Current owner-private association JSON; hidden/archived area projects as unset |
| GET `filter?area=ID&cursor=TOKEN` | Fresh authorized keyset page, HTML or JSON; limit 100 |
| GET `admin` | Fresh admin/MFA-authorized catalog and sharing forms, including archived records |
| POST `admin` | Strict JSON `Edit` or native admin form; create revision 0, update exact revision |
| POST `assign` | Strict JSON `Assign` or native owner form; empty AreaID unsets |
| POST `sharing` | Strict JSON `{Enabled, Revision}` or admin form; defaults false |

JSON mutations require an `Idempotency-Key`; forms carry `key`. Host authentication supplies identity and verifies mutation CSRF. Optional `X-Network`, if present, must equal authenticated network. Request bodies are limited to 64 KiB before authentication/CSRF reads. Unknown JSON/form fields, trailing JSON and duplicated form fields are rejected. RFC 9457-style generic errors avoid reflecting sensitive policy details. Results use `Cache-Control: no-store`, `nosniff` and restrictive CSP. No image or geometry input exists.

Trusted service-only operations: `CleanupListing` for departure/archive; `ExportSharing` for an approved federation adapter, consulting durable default-off network policy. Federation integration is absent.

`RunPagedCLI(ctx, service, authenticatedIdentity, cursorKey, args, output)` accepts:

- `catalog`
- `current LISTING`
- `filter AREA [CURSOR]`
- `assign LISTING AREA REVISION KEY` (an empty area unsets)
- `edit ID LABEL REVISION VISIBLE ARCHIVED KEY`
- `sharing`
- `share ENABLED REVISION KEY`

The host obtains identity/current authorization and stores secrets. Command arguments never establish user/admin authority. A standalone alternate credential store is deliberately absent.

## Independent audit additions

`DisableSharing(ctx, identity)` is an admin/MFA-checked, naturally idempotent safety operation with no retry allocation. POST `disable-sharing` accepts a native CSRF-protected form (revision0 field for the common form parser) or strict empty JSON object; CLI `unshare` uses the same service. This operation stays available when the ordinary retry quota is exhausted. Current sharing/assignment state must still match cached replies; changed/archived/retired objects reject stale retries. Listing cleanup removes association payloads from old retries while retaining fingerprint tombstones.

JSON requests require application/json (UTF-8), unique semantic keys, valid Unicode, object roots, bounded nesting/bytes and no trailing data. Native forms retain the existing strict field policy. Every mutation requires foreign keys enabled on its supplied connection; migration rejects incompatible owned schema objects before creating anything. Canonical local routes reject encoded/protocol-relative/backslash/trailing-slash forms.

## Master continuation HTTP query and callback contract

Only GET catalog/root/assignment `listing` and filter `area`/`cursor` fields are accepted, at most one value each. Unknown or malformed fields, duplicates and any POST query fields fail before host callbacks. Raw query bytes are capped at8192; a single empty cursor/listing remains supported.

Form token callbacks run before a second read-only HostAuth check requiring the same identity. Private catalog/assignment/admin reads follow that check and still enforce current Policy inside their database transaction. HostAuth must return current authority and honor cancellation. This is a bounded callback check, not a global atomic transaction across arbitrary external session stores.

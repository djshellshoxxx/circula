# Receipts offline numeric fidelity repair

## Frozen source

Base restoration `7e2df8e435f3d7000c791ad46ea6f397a98c26dd`. Local repair `abaa1e8942f0997f86ce9e3bbc6d80b2dd65d056`. Tree `4f9fa8e7a9067a3c9f4535990658d3c65f38d376`. Only `future/receipts/canonical.go` and `future/receipts/canonical_numeric_fidelity_test.go` change. The clean isolated branch is `master/receipts-numeric-fidelity-20261119`; nothing was pushed or published.

## Proven defect

`DecodeDocument` invoked JCS before typed integer decoding. The pinned JCS parser uses `strconv.ParseFloat(token,64)`, so a nonintegral decimal could first round to an integer and pass the receipt schema. The actual production entry point, with all other receipt fields valid, accepted all11 malformed-value fixtures before this patch:

- `amount_minor:1125899906842624.01` became the allowed maximum `1125899906842624`.
- Fractional values immediately above or below one, including exponent notation, became `1` for amount/revision.
- `precision:8.0000000000000001` became `8`.
- Positive and negative nonzero `1e-400` precision values underflowed to `0`.

This violates the documented exact-integer contract of the plain offline receipt parser. It is not a demonstrated signature forgery or authorization bypass. The certificate verifier separately requires the signed payload to be exactly canonical and remains unchanged.

## Repair and compatibility

After existing syntax/schema/bounds validation, the parser compares each original amount, precision and revision token with the exact integer value in the DTO. It strips decimal presentation zeros, compares significant digits and checks the required decimal-point shift. The comparison performs bounded string work within the existing16KiB document cap and never computes a large power or allocates an attacker-sized integer. Nonzero exponents outside an `int64` cannot equal the bounded required shift; zero is handled directly.

The repair preserves the existing parser's exact integral decimal/exponent forms such as `1.0`, `10e-1`, and a scaled representation of `2^50`, rather than banning those spellings. All18 positive controls passed before and after, including both integer upper bounds, precision0/8, negative zero, and zero with a very large exponent. Failure returns an empty `Document`. Existing unknown/duplicate/Unicode/array/schema handling and all signed certificate behavior stay in their prior paths.

## Actual validation

| Check | Result |
|---|---|
| Unchanged-source targeted numeric regression | Actual failure:11 negative subcases;18 exact controls passed. Package0.014s; wall130.327s including fresh dependency/standard-library build. No compiler/resource failure. |
| `go test -p 1 -count=1 -cover -timeout=90s ./...` | Full current receipts module PASS,0.044s and84.6% coverage; wall5.148s. Includes existing certificate/signing/verification regressions. |
| `go vet -p 1 ./...` | PASS; wall13.845s. |
| `go mod verify` | PASS; wall0.117s. |
| gofmt and `git diff --check` | Clean. |

Exact command/environment/output and every Go/module file hash are preserved in the four JSON/text receipts. The frozen commit matches the green source hashes. Dependencies remain the original pinned JCS and go-jose versions; no driver, dependency, signature profile or cryptographic policy was substituted. The shared test slot was explicitly returned after these commands terminated. This patch has no new race pass; original archive race evidence applies only to its original source, and future current-head combined/provider gates remain necessary.

## Review and ownership

Independent root review is requested before composition/publication. Root retains workflow/docs ownership. No Core path, PR106 README, PR110 handoff, activation or integration policy was modified. The portable patch and its SHA-256 are recorded in `qualification.json`.

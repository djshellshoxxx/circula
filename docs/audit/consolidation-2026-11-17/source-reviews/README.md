# Preserved workstream reviews

These are source-pinned workstream records, including findings before their repair or merge. Their pending-work statements describe their own snapshots. The parent [execution record](../README.md) and final disposition ledgers provide the resulting state.

The integration lead copied these records without rewriting the reviewers’ conclusions. Selected logs are preserved under [../evidence](../evidence), sometimes with clearer filenames. The [evidence manifest](../evidence-manifest.json) maps each retained record to its original workstream path and SHA-256. Records with whitespace-only normalization are explicitly marked with original and retained hashes. Other cited scratch files are not claimed to be retained; production regression tests and focused audit notes are committed with each fix. Review records may retain original scratch paths as provenance, not persistent links.

The full initial branch inventory is [../branch-inventory-initial.json](../branch-inventory-initial.json); the historical PR inventory is [../pr-disposition-initial.json](../pr-disposition-initial.json). These reviews are AI-assisted engineering checks and do not satisfy the external SEC-033 review gate.

Selected structured digest records use explicit `git_object_sha1` or `sha256_digest` wrappers to distinguish object identities/checksums from credentials. Replacing each sole-key wrapper with its value reconstructs the original JSON data. Stored unified diffs use a JSON `original_utf8` field; decoding it recovers exact original patch bytes. The evidence manifest records every such transformation, original/retained hashes and recovery instructions. Scanner configuration and scope are unchanged.

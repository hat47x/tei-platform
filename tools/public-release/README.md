# Public source candidate preparation

This is an **offline staging helper**, not a publication or approval mechanism.

`prepare-candidate.sh` reads immutable Git blobs from an **exact 40-character commit SHA** and copies only **explicitly reviewed, individual files** into a new directory outside the source repository. It does not carry Git history, remote URLs, tags, branch names, GitHub issues/PRs, or GitHub Actions history. It does not create commits, initialize a repository, connect to a network, or publish anything.

## Local usage

In a trusted private checkout:

```bash
sha=$(git rev-parse HEAD)
git ls-tree -r --name-only "$sha" > /private/review-all-paths.txt
# Independently review every file and prepare a narrowed, exact path allowlist:
# /private/reviewed-public-paths.txt
bash tools/public-release/prepare-candidate.sh \
  "$PWD" "$sha" \
  /private/reviewed-public-paths.txt \
  /private/tei-public-candidate
bash tools/public-release/verify-candidate.sh \
  "$PWD" "$sha" \
  /private/reviewed-public-paths.txt \
  /private/tei-public-candidate
bash tools/public-release/prepare-candidate-test.sh
```

`verify-candidate.sh` independently verifies the bytes, exact Git blob OIDs, file modes, permitted filenames and absence of extra regular files or symlinks in the exported candidate. Both scripts return nonzero if these checks fail. This is a **local integrity gate**, not a security clearance or review signature.

The allowlist must contain one exact repository-relative file path per line (ASCII letters, numbers, `_`, `-`, `.`, `/`). Blank lines and `#` comments are ignored. Entire directories, Git symlinks/submodules, duplicate paths, path traversal and all non-regular blobs are rejected. If the candidate lacks the root `README.md` and `LICENSE`, or lacks the explicit AI-development/human-review warning near the start of the README, preparation fails.

**Do not publish `/private/review-all-paths.txt`, the source checkout, or the audit report.** Keeping output outside the source repository prevents accidentally adding it to the private tree; it is not itself a confidentiality guarantee.

## Independent publication gates

- Exact source commit selected and recorded privately; no reference to a mutable branch as the proof of exported bytes.
- Every candidate file's content and its intellectual-property provenance reviewed. Confirm all required notices and the Reference Model boundary (#941).
- Run version-pinned secret/PII/license scanning on the *candidate bytes* and triage results separately from the private full-history audit (#1188); never print findings in public logs.
- Audit README, Markdown links, source-to-document references and the public-tree file inventory. An allowlist is not proof of content safety.
- Execute source-level compilation, full Go tests, CLI/reference validation and other appropriate E2E tests **against the complete approved source tree**. A partial candidate that lacks dependencies is not considered ready.
- Maintain a private record of exact source revision, selected file paths, original blob OIDs, candidate digests, reviewer decision and resulting public commit SHA. Do not automatically publish private commit IDs or internal report contents.
- Create a **fresh public Git history** only after review; publish by a separately authorized workflow. Enable a read-only CI, signed/qualified release process and branch protection. No automatic push from this helper.
- For subsequent releases, compare both public-tree diffs and changes to the published Git history; an initially clean public tree may later acquire private information through additions or commit messages.

The helpers use GNU/Linux Bash/Git/coreutils (`realpath`, `stat`, and `mv -T`) and are intended for the trusted WSL2/Linux execution host. Run the synthetic test from this host before attempting the full candidate. A candidate may be reproducible and still carry confidential or third-party material, so every accepted file needs independent human and license review.

A new public repository changes Git lineage and does **not** preserve private Issues/PR numbers, exact-head qualification receipts or past product Evidence. No implied qualification transfer.

Runbook: `plan/PUBLIC_RELEASE_READINESS_2026-10.md`; decision: #1190; overall gate: #1187.

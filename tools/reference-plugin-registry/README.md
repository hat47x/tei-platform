# Plugin Release Evidence Binding Reference (P0-A, test-only)

Tracking: [TEI #2589](https://github.com/hat47x/tei-platform/issues/2589), design: [`plan/design/PLUGIN_REGISTRY_PUBLICATION_VERIFICATION_DESIGN_2026-11.md`](../../plan/design/PLUGIN_REGISTRY_PUBLICATION_VERIFICATION_DESIGN_2026-11.md).

This small, independently readable fixture tests **exact Release bytes → publisher declaration signature → complete required Artifact bytes → scoped verifier statement signature**. It intentionally does **not** implement the Registry, Contract Resolver, current issuer directory, artifact store, OCI/GitHub, SLSA provenance, actual Conformance evaluation, or Runtime Authority.

Run from this directory:

```sh
go test ./...
node --test verify.test.mjs
node verify.mjs
```

The Go and Node readers do not share production code or a parser. They use standard-library SHA-256 and Ed25519 verification against two **hardcoded, test-only public keys**. The signatures in `fixture.json` were generated once offline with throwaway test credentials; no private key is included and the keys must never be trusted for live publication. The expected owner epoch and fresh/allowed Issuer policies are **mock inputs outside the untrusted release**. A positive result proves only that these fixture bytes and mocked policy satisfy this narrow reference contract.

The Release bytes are an intentionally fixed six-line ASCII frame, **not** a proposed Canonical TEI JSON or registration schema. It binds `plugin`, `version`, `owner epoch`, and **two mandatory artifact roles** (`core`, `ui`) to exact SHA-256 digests. Physical download locators are not part of its digest. Attestation text binds the exact Release digest, a sample Profile digest, one Target, and result `passed`. Signing prefixes separate publication and verification purposes. No claims are made about historical issuance time or current revocation.

Negative cases independently reject tampered or missing members, duplicate roles, malformed base64, changed Release bytes, borrowed/wrong target or Profile attestation, invalid publisher/verifier signatures, a changed owner epoch, and mocked stale/revoked Trust. These are bounded tests; they do not establish a Registry-wide atomic `(Plugin ID, Version)` uniqueness transaction, repo ownership, Package Conformance, simultaneous consumer readback, or TOCTOU-safe execution (#1370).

**Scope ceiling:** This is P0-A **partial binding evidence** only. A second independent implementation is *not* a second Publisher Authority or Resolver. The signature check is real Ed25519 against synthetic keys, but current trusted status and Conformance result are **fixture assertions**. Next: connect to #1104 actual Trust evidence, #1105/#2604 GitHub Release real readback, #951 Contract/Provider and #2390 independent Host. Keep the E0/E1/E2/E3 gates of #2589 open until each exact target is observed.

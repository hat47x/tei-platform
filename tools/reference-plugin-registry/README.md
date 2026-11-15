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


## Shared raw-wire interoperability reference (stacked candidate)

The test-only fixture envelope is deliberately **not** a production Registry
serialization format. Both readers now parse the same raw UTF-8 JSON bytes
before validating signed Release/Artifact content. The small wire guard rejects
duplicate object keys (including escape-equivalent keys), unknown or case-alias
fields, non-string fields, malformed nesting, and noncanonical Base64 containing
LF/CRLF. The signed Release framing and hardcoded test public keys do not change.

`wire-cases.json` contains **14 shared raw-wire cases**: three positive
(including a whitespace variant and an escaped-property-name variant) and
eleven rejected variants. Each case has a SHA-256 of the mutated serialized
JSON, which the independent Go/Node tests check **before** fixture parsing
and signature verification. This is stronger than running language-specific
mutations that happen to share a case name.

The limit is **E0-W reference interoperability only**. It does not establish
canonical JCS/DSSE/COSE, a secure issuer directory, current Publisher Trust,
GitHub Release readback, the release-version CAS, provider semantic Conformance,
or a Runtime/Host grant. No bundled test key may be used for real publication.


## E1-C independent signed HTTP Release readback (test-only)

E1-BではConsumerはHTTP経由でArtifactを取得する一方、署名済みReleaseの入力をローカルの固定Fixtureから得ていた。E1-Cの`http_consumer.mjs`は**別OSプロセスのNode ConsumerがHTTPから未信頼のRaw JSONを取得**して、既存`wire.mjs`の厳密なJSON検査と`verify.mjs`のtest-only Ed25519 Publisher/Verifier署名・Profile・Target・Release Digest検査を実行する。その後、検証済みReleaseの必須`core/ui` Role/Digestに対して**各Memberを個別のHTTP GET**で実取得しSHA-256を照合する。

対応するGoテスト`../reference-build-artifact-store/registry_e1_signed_http_test.go`は、**既存CASを使うPublisher側の試験用loopback Server**であり、新しいRegistry APIやWriterを実装しない。正例と9件の負例（署名・版・Target改変、Raw JSON同名キー重複、署名EnvelopeのRedirect/過大Body/欠落、正しいR1署名に対する別Member Body、Publisher読戻し成功後のMember欠落）をGoの子プロセス起動から実証する。ConsumerへPublisherのCAS Receipt・Byte・Roleの期待値を引数で渡さない。

固定テスト鍵による**signed subject verification**とLoopback HTTP member transportの合成はReference限定のClaimであり、実Publisher Authority/Owner委任、現在のIssuer鍵失効、実GitHub Releases、遠隔TLS／別Host、Version CAS、同時世代切替（ABA）、秘密Evidenceの開示・Runtime Effectは**未実証**。HTTP serverがPublisherであるという認証も行わない。テスト鍵・固定Policyを本番で再利用しない。製品用JSON Canonical Profileを制定するものでもない。

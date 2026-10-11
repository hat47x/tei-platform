# Build Attempt Failure Evidence Reference

GitHub #1110 の最小Referenceです。

exact BuildPlanとBuilderを確定し、actual build executionを開始した後にcompile / link / builder実行が失敗したケースを、CLI errorだけで失わず構造化Evidenceとして保持します。

## Failureの分離

このReferenceは次を同じ状態へ潰しません。

```text
Resolution Failure
  exact BuildPlanを確定できない

Build Attempt Failure
  exact BuildPlan / Builderでexecutionを開始したが失敗

Reproducibility Failure
  二つのactual outputを得たが一致しない

Successful Build Result
  candidate artifactを得た
```

ここで扱うのはBuild Attempt Failureだけです。

invalid / unresolved BuildPlanからAttempt Evidenceを作りません。

## exact binding

Attempt Evidenceはcurrent production modelの:

- BuildPlanFingerprint
- exact BuilderIdentity
- Target
- Profile

へbindします。

#1080 / #1106のdomain-neutral BuildExecution primitiveがproduction化された場合は、同じ責務をExecutionPlan fingerprintへ移す候補です。このReferenceだけを理由に新Core型を追加しません。

## phase / failure class

phaseとfailure classはReference-local文字列です。

例:

- phase: compile / link / builder-run
- failure class: compile-error / link-error / builder-start-failure

production enumは、このReferenceから先回りして固定しません。

## diagnostics

raw stdout / stderrをEvidence JSONへinlineしません。

Referenceはfull streamについて:

- SHA-256 digest
- observed byte count
- bounded retained byte count
- truncated flag
- access class = restricted-build-diagnostic

だけをEvidenceへ残します。

bounded retained bytesは、将来#1105 Artifact Store等へrestricted artifactとして保管するpressureです。このReferenceでは永続保存しません。

raw diagnosticにはprivate pathやtokenが含まれ得るため、Evidence本文・Build semantic identity・公開可能metadataへ無条件に入れません。

Diagnostic digestが変わればAttempt Evidence identityは変わり得ますが、BuildPlan fingerprintは変えません。

## retry history

同じBuildPlanを複数回attemptできます。

```text
attempt 1 failed
attempt 2 failed
attempt 3 succeeded
```

成功resultを記録しても過去failure fingerprintを削除しません。

Attempt ordinalはEvidence instanceの順序であり、BuildPlan identityには入りません。

## Build graph

failed nodeとblocked downstreamを分離します。

```text
generator = failed + Attempt Evidence
consumer  = blocked-by generator
```

blocked nodeへAttempt Evidence fingerprintを付けて「実行して失敗した」ように見せることをrejectします。

## Gate

Reference CLIは固定fixtureのfake compile failureを実行し、Evidence JSONをstdoutへ出した後に**意図的にnon-zero exit**します。

```bash
go run ./tools/reference-build-attempt-evidence \
  > /tmp/tei-build-attempt-evidence.json
```

成功exitへ丸めません。

## Authority

Attempt Evidenceは次を明示します。

- failure Evidence != build success
- failure EvidenceからArtifact Lockを作らない
- failure Evidence != reproducibility result
- diagnostic digest != Build identity
- diagnostic retention != publication authority
- AI repair input != Acceptance Authority

## 非目標

- compiler diagnostic schemaの標準化
- raw stderrの無制限保存
- secret redaction engine完成
- failed artifact execution
- production logging / retention
- actual compilerを故意に壊すintegration test

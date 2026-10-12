# ビルド試行失敗の参照実装

GitHub #1110 に対応する、模擬データを使った参照実装です。

実際のビルドでコンパイルやリンク、ビルダーの起動が失敗した場合を想定し、確定済みのビルド計画とビルダーに結び付けて失敗情報を保存する構造を検証します。現行の参照コマンドは実際のコンパイラやビルダーを起動しません。

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

成功結果を記録しても過去の失敗記録の指紋は削除しません。ただし、参照コマンドでは失敗だけを模擬し、成功したビルド結果は記録しません。テスト中に成功結果の参照を組み立てる場合も、それ自体は実際の成功を証明しません。

試行番号は記録の順序を表し、ビルド計画の識別には使いません。履歴に追加する失敗記録には、有効な内容指紋が必要です。終了コード0でシグナルもない試行を失敗として記録しません。

## Build graph

failed nodeとblocked downstreamを分離します。

```text
generator = failed + Attempt Evidence
consumer  = blocked-by generator
```

blocked nodeへAttempt Evidence fingerprintを付けて「実行して失敗した」ように見せることをrejectします。

## Gate

参照コマンドは固定データから模擬的なコンパイル失敗の記録を組み立て、標準出力にJSONを出した後、**意図的に終了コード1**で終了します。実際にコンパイル失敗を発生させたという検証結果ではありません。

```bash
go run ./tools/reference-build-attempt-evidence \
  > /tmp/tei-build-attempt-evidence.json
```

終了コードを成功扱いに変更しません。

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

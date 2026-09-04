# ISSUE-000043: Gate B utility subprocessからPostgreSQL DSNを隔離する

- 状態: Ready to verify
- GitHub Issue: #433
- 親Issue: `ISSUE-000037` / GitHub Issue #387
- 関連Issue: `ISSUE-000031` / #308、`ISSUE-000040` / #424、`ISSUE-000042` / #430、`ISSUE-000029` / #273

## 目的

Gate B final runnerが保持する`TEI_TEST_POSTGRES_DSN`を、実際にPostgreSQLへ接続する#308 child以外のsubprocessへ不要に渡さないようにします。

DSNにはcredentialを含み得るため、「通常Evidenceへ値を保存しない」だけでなく、利用しないprocessへそもそも継承しないleast-exposure境界を一貫させます。

## 背景

PR #418以降、final runnerは三laneのうち#308だけへDSNを渡し、#310 / #255 childからは除去していました。

一方、runner内部では`commandOutput`を通じて次のutility subprocessも起動します。

```text
go env GOWORK
go env GOFLAGS
go version
```

PR #434以前の`commandOutput`は親process environmentを通常継承していたため、これらのGo probeにもDSNが渡っていました。Go probeはDSNを必要としません。

## 実装 — PR #434

DSN除去を`environmentWithoutPostgresDSN`へ共通化しました。

```text
final runner process
  ├─ go env / go version probe ─ DSNなし
  ├─ #310 child              ─ DSNなし
  ├─ #255 child              ─ DSNなし
  └─ #308 child              ─ DSNあり
```

### 共通helper

`environmentWithoutPostgresDSN`は次を行います。

- `TEI_TEST_POSTGRES_DSN`をcase-insensitiveに識別する
- 同名keyが複数あってもすべて除去する
- PATH等の無関係なenvironmentは保持する
- 元のenvironment sliceを破壊しない

### #308は維持

`real-postgres-ephemeral-fixture` laneだけはDSNを必要とするため、base environmentのcopyをそのまま渡します。

本変更は#308接続契約を変更するものではありません。

## actual subprocess test

PR #434ではpure helper testに加え、実child process継承も確認しました。

親test processへ次を設定します。

```text
TEI_TEST_POSTGRES_DSN=postgres://user:secret@example.invalid/evidence
```

その状態で`commandOutput`から現在のtest binaryをchildとして起動し、childで`TEI_TEST_POSTGRES_DSN`が存在しないことを確認します。

期待結果:

```text
dsn-absent
```

standard-libraryだけの隔離package結果:

```text
ok  tei-probe-dsn  0.003s
```

これは通常TEI Repository checkout上のpackage testやactual #387 Evidenceではありません。

## secret非保存との関係

本Issueは新しいEvidence fieldを追加しません。

次は引き続き保存しません。

- `TEI_TEST_POSTGRES_DSN`
- credential / secret
- PostgreSQL Database名

「値を保存しない」と「不要なprocessへ渡さない」は別の境界です。本Issueは後者を補強します。

## #424 / #430との関係

```text
#430 / ISSUE-000042
  final runner compile前から exact GOENV=off

#424 / ISSUE-000040
  child / nested Go環境を
  GOWORK=off / GOFLAGS= / GOENV=offへ固定

#433 / ISSUE-000043
  DSNを必要な#308 childだけへ露出
```

Go environmentの再現性境界とsecret exposure境界を混同しません。

## 完了条件

- [x] utility subprocessからDSNを除去する（PR #434）
- [x] non-PostgreSQL laneと同じ共通helperへ収束する（PR #434）
- [x] case-insensitive duplicate除去のpure testを追加する
- [x] actual subprocess継承testを追加する
- [x] Repository正本を追加する（本書）
- [ ] tool READMEへutility probeのleast-exposureを同期する
- [ ] 実TEI checkout上で`go test ./tools/reference-gate-b-final-evidence -count=1`を再確認する
- [ ] actual #387 final runを同境界で実行する

最後のactual run自体は#387の完了条件です。本Issue単独の成功をGate B Readyへ格上げしません。

## 非目標

- #308からDSNを除去すること
- DSN値をEvidenceへ追加すること
- OS process全体のsecret sandbox化
- Go toolchain / module cacheのvendoring
- GitHub Actionsの再有効化
- #308 Provider Evidenceの代替

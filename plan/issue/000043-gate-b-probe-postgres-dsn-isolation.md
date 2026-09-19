# ISSUE-000043: Gate B utility subprocessからPostgreSQL secretを隔離する

- 状態: Done — Gate B Ready（R = `e5231839`）のactual #387 final runで確認済み（2026-09-20）
- GitHub Issue: #433
- 親: `ISSUE-000037` / #387
- 関連: `ISSUE-000044` / #436、`ISSUE-000045` / #441、GitHub #444

## 目的

final runner内部のGo utility subprocessへ、PostgreSQL接続に不要なsecretを渡しません。

「Evidenceへ値を保存しない」と「利用しないprocessへsecretを継承しない」は別の安全境界です。

## 現行secret model

現行final runではlaunch environmentへDSN本文を置きません。

```text
TEI_TEST_POSTGRES_DSN_FILE = Repository外original secret file path
TEI_TEST_POSTGRES_DSN = absent
```

final runner runtimeはsecret-free source preflight後にoriginal secret fileをstable single-openし、normalized DSN本文をmemoryへ保持します。その後、Repository外0700 temporary directory / 0600 handoff fileを作成します。

production経路ではbase environmentから少なくとも次の双方をcase-insensitiveに除去して、secretを必要としないsubprocess environmentを構築します。

- `TEI_TEST_POSTGRES_DSN`
- `TEI_TEST_POSTGRES_DSN_FILE`

original secret file pathとtemporary handoff pathをGo utilityへ伝播しません。

## utility subprocess

少なくとも次へDSN本文、original secret file path、temporary handoff pathを渡しません。

```text
go env GOWORK
go env GOFLAGS
go version
```

最終的な露出境界は次です。

```text
Go environment / version probe
  -> DSN本文なし
  -> original secret pathなし
  -> handoff pathなし

#310 / #255
  -> DSN本文なし
  -> original secret pathなし
  -> handoff pathなし

#308 go run / compiler / linker
  -> DSN本文なし
  -> original secret pathなし
  -> handoff pathのみ

#308 runtime
  -> handoff fileからDSN本文を読込
  -> DSN / file path environmentをsource probe前に消去
  -> 以後DSN本文はmemoryのみ
```

#308だけは実PostgreSQL接続を必要としますが、必要なのはruntimeで読めるhandoff pathです。`go run` compile environmentへDSN本文を注入しません。

## helper contract

secret除去helperは次を満たします。

- DSN本文keyをcase-insensitiveに除去
- DSN file path keyもcase-insensitiveに除去
- duplicate keyをすべて除去
- PATH等の無関係environmentを維持
- callerのenvironment sliceを破壊しない

handoff pathを#308 laneへ追加する処理は、secret-free base environmentの構築後に#308だけへ限定して行います。

## 成立済み

- [x] Go utility subprocessからDSN本文を除去（PR #434）
- [x] non-PostgreSQL laneとleast-exposure helperを共有
- [x] duplicate / case-insensitive除去test
- [x] actual child subprocess継承test
- [x] `ISSUE-000045`でoriginal secret file pathも除去対象へ拡張
- [x] PR #473で#308 compile環境をtemporary path-only handoffへ変更
- [x] Go utilityへtemporary handoff pathを伝播しない
- [x] tool READMEへGo utility least-exposureを同期
- [x] `plan/GATE_B_EVIDENCE_RUNBOOK.md`へ同期

## 残作業

- [x] 実TEI full checkoutで次を再確認する

```bash
GOENV=off \
GOWORK=off \
GOFLAGS= \
go test ./tools/reference-gate-b-final-evidence -count=1
```

- [x] actual #387 final runで同じsecret exposure境界を使用する
- [x] actual #308 child compile環境にDSN本文が存在しないことを確認する

本Issue単独の成功をGate B Readyへ格上げしません。

## 非目標

- #308 runtimeから必要なDSN利用を除去すること
- DSN / original secret path / handoff pathをEvidence fieldへ追加すること
- OS process全体のsecret sandbox化
- secret manager製品の標準化
- CI greenをGate B Product Evidenceへ格上げすること

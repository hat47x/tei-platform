# ISSUE-000043: Gate B utility subprocessからPostgreSQL secretを隔離する

- 状態: Ready to verify — 実装 / isolated test / tool README / Runbook同期済み、actual checkout / final run待ち
- GitHub Issue: #433
- 親: `ISSUE-000037` / #387
- 関連: `ISSUE-000044` / #436、`ISSUE-000045` / #441

## 目的

final runner内部のGo utility subprocessへ、PostgreSQL接続に不要なsecretを渡しません。

「Evidenceへ値を保存しない」と「利用しないprocessへsecretを継承しない」は別の安全境界です。

## 現行secret model

現行final runではlaunch environmentへDSN本文を置きません。

```text
TEI_TEST_POSTGRES_DSN_FILE = Repository外secret file path
TEI_TEST_POSTGRES_DSN = absent
```

runner起動後にsecret fileを一度だけ読み、DSN本文はmemoryへ保持します。

production経路ではbase environmentから次の双方を除去してsubprocess environmentを構築します。

- `TEI_TEST_POSTGRES_DSN`
- `TEI_TEST_POSTGRES_DSN_FILE`

## utility subprocess

少なくとも次へDSN本文もsecret file pathも渡しません。

```text
go env GOWORK
go env GOFLAGS
go version
```

最終的な露出境界は次です。

```text
Go environment / version probe -> DSN本文なし / file pathなし
#310                           -> DSN本文なし / file pathなし
#255                           -> DSN本文なし / file pathなし
#308                           -> in-memory DSN本文のみ / file pathなし
```

#308は実PostgreSQL接続が必要なので、runner memory上のDSN本文だけをchild environmentへ明示注入します。

## helper contract

secret除去helperは次を満たします。

- DSN本文keyをcase-insensitiveに除去
- DSN file path keyもcase-insensitiveに除去
- duplicate keyをすべて除去
- PATH等の無関係environmentを維持
- callerのenvironment sliceを破壊しない

## 成立済み

- [x] Go utility subprocessからDSN本文を除去（PR #434）
- [x] non-PostgreSQL laneとleast-exposure helperを共有
- [x] duplicate / case-insensitive除去test
- [x] actual child subprocess継承test
- [x] `ISSUE-000045`でsecret file pathも除去対象へ拡張
- [x] tool READMEへGo utility least-exposureを同期
- [x] `plan/GATE_B_EVIDENCE_RUNBOOK.md`へ同期

## 残作業

- [ ] 実TEI checkoutで次を再確認する

```bash
GOENV=off \
GOWORK=off \
GOFLAGS= \
go test ./tools/reference-gate-b-final-evidence -count=1
```

- [ ] actual #387 final runで同じsecret exposure境界を使用する

本Issue単独の成功をGate B Readyへ格上げしません。

## 非目標

- #308から必要なDSN本文を除去すること
- DSN / secret file pathをEvidence fieldへ追加すること
- OS process全体のsecret sandbox化
- secret manager製品の標準化
- GitHub Actions再有効化

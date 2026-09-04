# ISSUE-000036: Gate B 3-lane scopeを既存長文へ内容保持で同期する

- 状態: In Progress
- GitHub Issue: #383
- 前段判断: `ISSUE-000035` / GitHub Issue #380（Done）
- 親Issue: `ISSUE-000006` / GitHub Issue #69
- Gate B監査: `ISSUE-000029` / GitHub Issue #273

## 目的

`ISSUE-000035` / #380で確定したGate B scopeを、既存の長い計画文書へ**設計履歴とfail-closed条件を落とさず**同期します。

scope判断そのものは完了しています。

Gate Bの必須Evidenceは次の三laneです。

1. #310 deterministic Repository Evidence
2. #255 generated artifact reproducibility
3. #308 real PostgreSQL Provider Evidence

#214 actual Oracle / #371 / #376はGate B必須から外しますが削除しません。Milestone 2の既存外部資産対応とMilestone 8の第二Provider圧力試験として継続し、未完了ならHandover Baselineの`unresolved`へ引き継ぎます。

## このIssueを分ける理由

scope再監査の途中で、複数のGate B長文をまとめて3-lane化する差分を一度作りました。

しかし、差分が大きくなり、次まで圧縮・削除する危険がありました。

- #214で成立済みのruntime / Evidence intake境界
- #308 fixture安全性の設計履歴
- #255 generated artifactのrepair / check-only分離
- #310 source-fixed runnerのfail-closed条件
- Evidence natureを分けてきた経緯
- candidate revisionを固定する理由

そのため一括置換は採用しません。

現行本文を保持し、**scopeに関係する文だけを小さいPRで更新する**ことを原則とします。

## 同期対象

### 判定とEvidence契約

- `plan/GATE_B_READINESS_AUDIT.md`
- `plan/GATE_B_EVIDENCE_ACQUISITION.md`
- `plan/GATE_B_EVIDENCE_RUNBOOK.md`
- `plan/GATE_B_TO_C_HANDOVER_CONTRACT.md`

### v1主線と導線

- `plan/V1_REFERENCE_APPLICATION_PLAN.md`
- `plan/README.md`

### Repository Issue正本

- `plan/issue/000006-application-driver-vertical-slice.md`
- `plan/issue/000029-gate-b-readiness-audit.md`

### GitHub Issue本文

必要に応じて次を同期します。

- #69
- #214
- #273
- #291
- #70

scope判断自体は既にコメントで同期済みです。本文更新は、古い4-lane表現が現在の必須条件として誤読される箇所に限定します。

## 同期原則

### 1. 既存のEvidence設計を消さない

#214をGate B必須から外しても、次の成立済み成果は履歴として残します。

- Contract Query runtime
- external Evidence intake
- source-fixed acceptance receipt
- raw列 / driver metadata観測境界
- read-only Evidence計画
- driver integration計画

「Gate Bで必須ではない」と「不要・失敗だった」を混同しません。

### 2. #214を#308で代替しない

#308はreal PostgreSQL engine + ephemeral fixture Evidenceです。

#214 actual Oracle Evidenceとは性質が異なります。

Gate B必須から#214を外す理由は、#308が同じものを証明するからではなく、**Gate Bの証明対象をApplication / Driverへ戻すため**です。

### 3. candidate revisionは三laneへ収束する

Gate Bのcandidate revisionを`R`とする場合、完了判定へ使うsource-dependent Evidenceを次へ揃えます。

```text
#310 -> R
#255 -> R
#308 -> R
```

異なるrevisionの成功runを寄せ集めてReadyにはしません。

### 4. #214未完了はHandoverのunresolvedへ残す

少なくとも次を明示します。

- actual physical facts未観測
- actual Oracle read未実証
- driver integration未確定
- read-only実効Evidence未取得

第一Web Targetがこの不足をWeb固有実装で埋めることは禁止します。

### 5. 実行手順の詳細を守る

Runbookでは、#310 / #255 / #308について既存の正規tool、source-fixed条件、safe-replace、fixture safety、Event Trigger guard等を保持します。

Oracle固有のactual Evidence手順はGate B必須の実行順から分離しますが、後段Evidenceとして追跡可能な状態を保ちます。

### 6. 日本語を自然にする

古い文へ注記を機械的に積み重ねるだけでなく、現在地から読んで意味が自然につながるように整えます。

ただし、自然さのために設計履歴や条件を削りすぎません。

## 作業順

まず判断への影響が大きい文書から進めます。

1. `GATE_B_READINESS_AUDIT.md`
2. `GATE_B_EVIDENCE_ACQUISITION.md`
3. `GATE_B_TO_C_HANDOVER_CONTRACT.md`
4. `GATE_B_EVIDENCE_RUNBOOK.md`
5. `ISSUE-000029`
6. `ISSUE-000006`
7. `V1_REFERENCE_APPLICATION_PLAN.md`
8. `plan/README.md`
9. GitHub Issue本文の残存表現を監査

Runbookは長く安全条件が多いため、scope判断だけを理由に全文を短縮しません。

## 完了条件

- [ ] Readiness Auditを内容保持で3-lane scopeへ同期する
- [ ] Evidence Acquisitionを内容保持で同期する
- [ ] Handover Contractへ#214 unresolvedを明示する
- [ ] Runbookを内容保持で同期する
- [ ] ISSUE-000029を同期する
- [ ] ISSUE-000006を同期する
- [ ] V1 Reference Application Planを同期する
- [ ] Planning READMEを同期する
- [ ] GitHub Issue本文に古い必須4-lane表現が残っていないか監査する
- [ ] #380 / ISSUE-000035のscope判断と矛盾しないことを確認する
- [ ] Gate Bを実EvidenceなしにReadyへ変更していないことを確認する

## 非目標

- #214 / #371 / #376を削除・完了扱いすること
- #308をOracle Evidenceの代替とすること
- Gate B Ready前に第一Web Targetを実装すること
- GitHub Actionsを再有効化すること
- 文書を短くすること自体を目的にすること

## 関連

- `ISSUE-000035` / #380
- #69 / #70 / #214 / #255 / #273 / #291 / #308 / #310 / #371 / #376 / #383
- PR #379 / #382

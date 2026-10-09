# ISSUE-000086: REF-SPEC-01のレビュー結果を一つのpacketへ束ねる

- 状態: In Progress
- 起点: GitHub Issue #915 / `plan/TECH_RESEARCH_INTEGRATION_2026-10.md`
- 関連: ISSUE-000027 / ISSUE-000047 / ISSUE-000076 / ISSUE-000075
- 対象価値: 実装前に仕様の不足・曖昧さ・矛盾・古い前提へ気づけること

## 目的

REF-SPEC-01では、同じ変更要求を通常のSource探索とTEIの依存・差分・未解決表示でレビューし、見落としや確認負担を比較する。

最初から新しい仕様判定エンジンは作らない。すでにmainにある`knowledge-context`、Business Procedure checker、Shared Acceptanceのmachine-readable出力を一つのread-only packetへ束ねる。元toolの診断意味やauthorityは変えない。

## 最初の縦切り

既存のcustomer edit変更要求を使う。

```text
Desired change
  order_entry.customer_reference を更新したい

Accepted context
  Pattern -> datasource master
  owning Application -> master access = read

Cross-view result
  BUSINESS_FUNCTION_DATASOURCE_ACCESS_CONFLICT
```

このcaseでは、`knowledge-context`で対象PatternとSourceを絞り、既存checkerが返す衝突を同じpacketへ載せる。Shared Acceptance出力がある場合は、基準revisionが古いProposalも別findingとして載せる。

## Review packet

Reference-onlyのschemaは`tei.reference.spec-review/v0`とする。

packetは次を区別する。

```text
coverage
  knowledge-context
  cross-view
  shared-acceptance

finding
  ambiguity
  inconsistency
  cross-view-difference
  stale-premise
  source-diagnostic

missing input
  未確認
  != 問題なし

evidence
  existing subject / source / fingerprint / role
  -> findingから元Sourceへ戻る導線
```

`evidence`は既存toolが持つSource traceを引き継ぐだけで、新しい根拠を生成しない。完全一致する重複行だけを正規化し、入力に無いfingerprintやAuthorityは補完しない。

`status=no-finding-in-provided-evidence`も仕様の完全性を意味しない。確認した三つの入力の範囲でfindingが無かったことだけを表す。

## PR #1040との境界

PR #1040は、Business Procedure側のValidation参照とApplication側の受理済みValidation集合との差を読み取り専用で表示する作業である。

本Issueでは、その比較ロジックを複製しない。cross-view JSONに`validationScopeReview`が含まれる場合、そこで確定した`review-needed`や`unresolved-*`をpacketへ写すだけにする。PR #1040が未統合の間も、既存のcustomer edit diagnosticだけで本Issueの最小Referenceは成立する。

## 実装

`tools/reference-spec-review`を追加する。

最初のtestでは次を固定する。

1. Knowledge Contextの`unresolved`、customer edit conflict、stale revisionを別classで保持する。
2. 実Repositoryの`order_entry.customer_reference`とcustomer edit probeを、既存CLIの出力から一つのpacketへ束ねる。
3. 入力が不足した場合は`partial-no-finding`とし、不在をnegative Evidenceへ変換しない。
4. 未知のcross-view diagnosticを独自解釈せず`source-diagnostic`として保持する。
5. PR #1040型のValidation scope差を入力できても、その意味の所有者はBusiness Procedure checker側に残す。
6. JSONの後ろに追加valueがある不正入力を拒否する。
7. Knowledge ContextとShared Acceptanceが持つSource Evidenceをpacketへ保持し、完全一致する重複だけを除く。

## 非目標

- 新しいCanonical kindや仕様モデルを追加すること
- Business Procedure checkerの診断ロジックを複製すること
- Shared Acceptanceのstale判定を再実装すること
- findingを自動修正すること
- Review packetをAcceptance authorityへすること
- REF-SPEC-01の比較実験が完了したと主張すること

## 完了条件

- [ ] 実customer edit fixtureから一つのreview packetを再現できる
- [ ] ambiguity / inconsistency / stale premiseを混同せず表示できる
- [ ] 未確認の入力をcoverage gapとして残せる
- [ ] 元toolの診断意味を上書きしない
- [ ] findingから既存Source pathとfingerprintへ再到達できる
- [ ] read-onlyでReference Workspaceを変更しない
- [ ] exact headのclean checkoutで`go test ./tools/reference-spec-review -count=1`がPASSする
- [ ] exact headで`tools/repository-health-check.sh`が全項目PASSする
- [ ] 検証前後のworktreeがcleanである

## 次段

このpacketが成立した後に、REF-SPEC-01のA/B比較へ進む。測定対象は見落とし、誤警告、開いたSource数、最初の意味あるfindingまでの操作数、確認時間、findingからSourceへ再到達できるかである。

packet自体の存在やfinding件数をProduct Valueの達成とみなさない。

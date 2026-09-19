# ISSUE-000013: Gate Aの到達性を横断監査する

- 状態: Done
- GitHub Issue: #124
- 前段: `ISSUE-000005` / GitHub Issue #67（完了）
- 後段: `ISSUE-000006` / GitHub Issue #69（Application Driver）
- Gate A判定: Ready
- Gate AでReadyになった関連Issue: #68（既存認証Compatibility） / #77（Capability Outcome） / #125（業務Validation） / #126（Capability Binding）
- 監査結果: [`GATE_A_READINESS_AUDIT.md`](../GATE_A_READINESS_AUDIT.md)

## 目的

Application Pattern候補5種類がすべて実Referenceまで成立した後、Driverへ渡すApplication上の意味を横断的に確認しました。

Gate Aでは、Web画面やDriver専用設定を見なければ分からない情報を残さないことを重視します。

Driver #69が、Repository上のCanonicalな意味、Contract、Bindingから少なくとも次を解決できるかを監査対象としました。

- browse / record / reference / header-detail / commandの対象
- Datasource read / write境界
- 業務Validation
- Authentication / Authorization
- Capability Binding
- Business Result / Execution Error
- Command / Observationの境界

## 最終判定

**Gate AはReadyです。Driver #69の主実装を開始できます。**

最後まで残っていた#68について、PR #148、#149、#151で必要なEvidenceが揃いました。

### Readyと判断した領域

- Application Pattern 5種類
- Datasource境界
- Capability Binding #126 / PR #135
- 業務Validation #125 / PR #136 / #139 / #140
- Capabilityのnamed Outcome / Execution Error #77 / PR #132 / #141
- 既存単一output Capabilityとの互換性
- Authentication / Authorization #68 / PR #148 / #149
- 既存認証の実現方式交換 #68 / PR #151
- Secret非保存境界
- Command / Observation分離の基本境界

## #68からGate Aへ渡せるようになったもの

### Authentication

`security.authenticate`を`order_entry`から明示的なApplication Bindingで解決できます。

Referenceでは、user ID、disabled、validity、login failure count、lock、organization、role等の既存情報を保持したまま認証adapterへ渡せます。TEI標準User schemaへ先に作り替える必要はありません。

認証成功と正常な拒否は`authenticated` / `denied`というBusiness Resultとして扱い、store、adapter、既存関数等の実行障害はExecution Errorとして分離します。

### Authorization

`security.authorize`を次のApplication上の操作へ接続できています。

```text
browse  / order_entry.recent_orders
edit    / order_entry.order_record
execute / order_entry.submit_order
```

resourceにはWeb画面名やURLではなくApplication Pattern identityを使います。

organization、role、既存ruleはadapter側へ残し、ApplicationやDriverへ同じ判断条件を書き写しません。

### 実現方式の交換

PR #151では、`order_entry/security.authenticate`という一つのBindingを保ったまま、次の二方式を交換できることを確認しました。

1. 既存の利用者情報を読む方式
2. 既存認証関数、stored procedure、外部API等に相当する処理を呼ぶ方式

どちらも同じCanonical `security.authenticate`契約とauthenticated subjectを返します。

この実証では`reference/workspace`と`reference/world.json`を変更していません。CI #374のReference World再現性検査もGolden更新なしで成功しており、実現方式の違いがCanonical Worldへ入り込んでいないことを確認できました。

## Secretの境界

実行時credentialや実secretを次へ保存しない境界が成立しています。

- Canonical World
- Source Index
- Business Result
- Git管理するReference fixture
- 通常Evidence

`security.credential`は入力の意味を表すCanonicalな型であり、credential値そのものの保存場所ではありません。

Driverでも、Scenarioの再現性を理由にsecretを平文で記録しないことをGate Bの条件として引き継ぎます。

## 判定原則

Gate Aは、関連Issueがすべてcloseしたかどうかだけでは判定していません。

Driverへ渡すために必要な意味をRepository上のReference / Conformanceから再現できるかを基準に、一項目ずつ再評価しました。

その結果、Gate Aの独立したBlockerはなくなりました。

## Driver #69へ引き継ぐ条件

Gate AがReadyになったことは、Driver自体の設計が完了したことを意味しません。

Driverでは、今回成立した境界を**利用すること**を確認します。

- PatternをDriver専用設定へ複製しない
- Datasource read / write境界を独自定義しない
- Validation rule本文をDriverへ再実装しない
- legacy schema、organization、role、return codeへDriverが直接依存しない
- Capability Bindingをfile pathやGo hookから暗黙に推測しない
- expectedな`denied`や`NotSatisfied`をDriver障害として扱わない
- Execution Errorを業務上のOutcomeへ変換しない
- commandの応答とObservationによる外部状態を混同しない
- secretをScenario Evidenceへ保存しない

これらを満たしながら、UIなしで主要Scenarioを再現できた時点でGate Bを評価します。

## 非目標

- Driver自体の主実装
- 第一Web Target
- production-ready IdP
- production transaction coordinator
- 5 Patternの標準Catalog昇格
- Gate A以後のOutcome Projection
- 汎用Validation rule engine
- 認証製品やlegacy schemaの標準化
- Runtime上の汎用implementation選択機構

## 完了条件の評価

- [x] Gate Aの入力項目ごとにEvidenceを説明できる
- [x] Application Pattern Stage完了とGate A全体完了を混同していない
- [x] Gate Aで必要なAuthentication / Authorization境界が成立している
- [x] DriverがWebや専用設定から業務意味を補わなくてよい
- [x] Driver #69を開始してよいと明示的に判定できる

## 関連

- GitHub Issue #124: Gate A到達性監査
- GitHub Issue #68: 既存認証Compatibility
- GitHub Issue #69: Application Driver
- GitHub Issue #77: Capability Outcome / Execution Error（完了）
- GitHub Issue #125: 業務Validation（完了）
- GitHub Issue #126: Capability Binding（完了）
- GitHub PR #132 / #135 / #136 / #139 / #140 / #141 / #143 / #148 / #149 / #151
- ADR-0019 / ADR-0020
- RQ-005, RQ-008, RQ-009, RQ-012, RQ-014, RQ-017, RQ-018, RQ-019, RQ-021

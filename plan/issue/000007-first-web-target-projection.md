# ISSUE-000007: 第一Web TargetへのProjection境界を設計する

- 状態: 提案中
- GitHub Issue: #70
- 依存: Gate B

## 背景

Product Requirements Gap Auditでは、RQ-013「Target-neutralなPresentation / Interaction境界を持つこと」が未着手であることを確認しました。

ただし、最初からWeb / Desktop / Mobile / API / MCP / VUIを一つの抽象モデルへ押し込むと、実証のない巨大なPresentation IRを作る危険があります。

そのため、`ISSUE-000005`のApplication Patternと`ISSUE-000006`のDriverを先に成立させ、Gate B「UIなしで主要な業務シナリオを再現できる」を確認してから、その同じ`order_entry`を第一TargetとしてWebへProjectionします。

## 関連するProduct Requirement

- RQ-009 既存認証・認可互換
- RQ-010 Editor
- RQ-011 Projection
- RQ-012 Driver
- RQ-013 Target-neutral Presentation
- RQ-014 Application Pattern
- RQ-017 二重管理の最小化
- RQ-021 Reference / Conformance
- RQ-024 Target別性能

## 目的

Web画面を作ること自体ではなく、**Application PatternからWebへ落とすために本当に必要なPresentation / Interaction上の意味を、具体的なReferenceから抽出すること**を目的とします。

Driverで確認済みのApplication上の意味を利用することで、Application側の不足をWeb固有コードで補ってしまうことを避けます。

## Gate Bから受け取るもの

第一Web Targetでは、少なくとも次がUIなしで実行・説明できる状態から開始します。

- browse
- record
- reference
- header-detail
- Validation
- Authorization
- command
- Observation

Webはこれらの意味を再定義せず、利用者が操作できるPresentation / InteractionへProjectionします。

## 対象Scenario

`order_entry`の次の5 PatternをWebへProjectionします。

- browse
- record
- reference
- header-detail
- command

加えて、Driverと同じValidation、Authorization、Observation境界へ接続します。

Web Targetだけのために別のApplication modelを作りません。

## 最初に扱うPresentation上の意味

具体的なUI部品名をCoreへ固定する前に、少なくとも次の区別が必要かをReferenceから検討します。

- collection / item
- read / edit
- field grouping
- relation / reference selection
- detail collection
- validation feedback
- command invocation
- authorizationによる利用可否
- read-only / unavailable
- navigation / focus target

これらをすべて共通IRへ入れるとは限りません。

Application上の意図、Projection情報、Web固有実装のどこに属するかを分けます。

## Custom UI

標準Patternから生成できる画面だけをTEIの世界とはしません。

第一Web Targetの段階から、少なくとも一箇所はCustom UIを接続できる逃げ道を用意し、標準ProjectionとCustom implementationの境界を確認します。

Custom UIを使う場合も、Application上の意味やValidation、Authorization、Capabilityとの接続を維持します。

## Web Adapter

Web固有の実装はCoreへ入れず、独立したAdapter / Packageとして組み立てます。

具体的なframeworkは、このIssueの着手時点で最新の技術条件と、生成物の検査しやすさ、Wails / WebViewとの再利用可能性などを比較して決めます。

frameworkの選定結果をPresentation上の意味の一部にはしません。

## Interaction

Web Targetでは、表示だけでなく次の操作をDriverと同じ意味へ接続します。

- search
- open
- edit
- add / remove detail
- reference lookup
- validate
- authorize
- submit command
- observed completionの表示

Target内で独自の業務ruleや認可ruleを再実装しません。

## Error / Diagnostic

Application validationとauthoring diagnosticを混同しません。

- end-user向けvalidation feedback
- authorization failureなどの利用者向け状態
- developer向けSource / Diagnostic

は、それぞれ意味の異なるProjectionとして扱います。

## 性能

RQ-024の最終contractは第二Target以降で比較して決めますが、第一Web Targetから測定点を用意します。

少なくともReference Scenarioについて次を測れる構造を持たせます。

- initial projection / load
- browse result render
- record open
- reference lookup
- validation
- authorization
- command response
- observed completion

このIssueでは、根拠のない絶対値を固定しません。

## Gate Cへ進む条件

第一Web Targetが表示できたことだけでは完了としません。

次をEvidenceから確認できたとき、Gate C「第一Targetとの差を説明できる」を通過したとみなします。

- 同じApplication / PatternをDriverとWebで共有する
- Web固有fieldをCanonical Semantic Modelへ追加しない
- native SQLやCapability implementationをWeb側へ複製しない
- read-only Datasource制約をTargetが破らない
- Validationの意味をWebで再実装しない
- Authorizationの意味をWebで再実装しない
- standard ProjectionとCustom UIを併存できる
- Web Adapterを外してもCanonical Worldが変わらない
- Source / Application / Targetの来歴を追跡できる
- 複数Targetで共有できそうなPresentation上の意味と、Web固有差分を明示できる
- Reference Scenarioの主要な測定点を取得できる

## Editor / AIへの引き継ぎ

Gate Cを通過すると、Application上の意味、Driver実行結果、第一Targetとの差分が実資産として揃います。

この時点で`ISSUE-000008`の保留を解除するか判断します。

EditorやAIが何を表示・探索・確認すべきかを、抽象的な想像ではなく、実際のApplication / Driver / Target差から具体化できる状態にします。

## 第二Targetへの引き継ぎ

このIssueの完了時点では「Target-neutral設計が完成した」と主張しません。

Webで得られた共通候補とWeb固有差分を記録し、第二Targetを実装したときに初めて、何が本当にTarget-neutralだったかを圧力試験します。

第二Targetで境界修正が必要になっても、それ自体を失敗とはみなしません。第一Web Targetの都合を誤って共通意味へ持ち上げていなかったかを検査するEvidenceとして扱います。

## 非目標

- 全Target対応
- design systemの完成
- pixel-perfectなGUI Editor
- Web frameworkをTEI標準として固定すること
- B2C大規模負荷のproduction保証
- Custom UIを禁止すること
- 第一TargetだけでTarget-neutral設計の完成を宣言すること

## 完了条件

`order_entry`の5 PatternとValidation / Authorization / Commandを、Driverと同じApplication上の意味から第一Web TargetへProjectionできること。

その過程で得た「複数Targetで共有できそうなPresentation上の意味」と「Web固有の実装」を明示的に分離し、Gate Cを評価できるEvidenceと、Editor / AI支援および第二Targetへ渡せる差分を残します。

## 関連文書

- `plan/V1_REFERENCE_APPLICATION_PLAN.md`
- `plan/adr/0017-application-pattern-ownership.md`
- `plan/issue/000006-application-driver-vertical-slice.md`
- `plan/issue/000008-editor-and-ai-followup-note.md`

# ISSUE-000007: 第一Web TargetへのProjection境界を設計する

- Status: Proposed

## 背景

Product Requirements Gap Auditでは、RQ-013「Target-neutralなPresentation / Interaction境界を持つこと」が未着手であることを確認しました。

ただし、最初からWeb / Desktop / Mobile / API / MCP / VUIを一つの抽象モデルへ押し込むと、実証のない巨大なPresentation IRを作る危険があります。

そのため、`ISSUE-000005`のApplication Patternと`ISSUE-000006`のDriverを先に成立させ、その同じ`order_entry`を第一TargetとしてWebへProjectionします。

## 関連するProduct Requirement

- RQ-010 Editor
- RQ-011 Projection
- RQ-012 Driver
- RQ-013 Target-neutral Presentation
- RQ-014 Application Pattern
- RQ-017 二重管理の最小化
- RQ-021 Reference / Conformance
- RQ-024 Target別性能

## 目的

Web画面を作ること自体ではなく、**Application PatternからWebへ落とすために本当に必要なPresentation / Interactionの意味を、具体的なReferenceから抽出すること**を目的とします。

## 対象Scenario

`order_entry`の次の5 PatternをWebへProjectionします。

- browse
- record
- reference
- header-detail
- command

Driverで確認済みの業務意味を再利用し、Web Targetだけのために別のApplication modelを作りません。

## 最初に扱うPresentation意味

具体的なUI部品名をCoreへ固定する前に、少なくとも次の区別が必要かをReferenceから検討します。

- collection / item
- read / edit
- field grouping
- relation / reference selection
- detail collection
- validation feedback
- command invocation
- read-only / unavailable
- navigation / focus target

これらをすべて共通IRへ入れるとは限りません。

Application intent、Projection information、Web固有implementationのどこに属するかを分けます。

## Custom UI

標準Patternから生成できる画面だけをTEIの世界とはしません。

第一Web Targetの段階から、少なくとも一箇所はcustom UIを接続できるescape hatchを用意し、標準Projectionとcustom implementationの境界を確認します。

Custom UIの存在によってCanonical meaningやApplication Patternを失わないことを条件とします。

## Web Adapter

Web固有の実装はCoreへ入れず、独立したAdapter / Packageとして組み立てます。

具体的なframeworkは、このIssueの開始時点で最新の技術条件と、生成物の検査しやすさ、Wails / WebViewとの再利用可能性等を比較して決めます。

framework選定をPresentation meaningの一部にはしません。

## Interaction

Web Targetでは、表示だけでなく次の操作をDriverと同じmeaningへ接続します。

- search
- open
- edit
- add / remove detail
- reference lookup
- validate
- submit command

Target内で独自のbusiness ruleを再実装しません。

## Error / Diagnostic

Application validationとauthoring diagnosticを混同しません。

- end-user向けvalidation feedback
- developer向けSource / Diagnostic

は別Projectionとして扱います。

## 性能

RQ-024の最終contractは第二Target以降で比較して決めますが、第一Web Targetから測定点を用意します。

少なくとも、Reference Scenarioについて次を測れる構造を持たせます。

- initial projection / load
- browse result render
- record open
- validation
- command response / observed completion

このIssueでは根拠のない絶対値を固定しません。

## Conformance

- 同じApplication / PatternをDriverとWebで共有する
- Web固有fieldをCanonical Semantic modelへ追加しない
- native SQLやCapability implementationをWeb側へ複製しない
- read-only Datasource制約をTargetが破らない
- Validation semanticsをWebで再実装しない
- standard Projectionとcustom UIを併存できる
- Web Adapterを外してもCanonical Worldが変わらない
- Source / Application / Targetのprovenanceを追跡できる

## 第二Targetへの引き継ぎ

このIssueの完了時点では「Target-neutral設計が完成した」と主張しません。

Webで得られた共通候補とWeb固有差分を記録し、第二Targetを実装したときに初めて、何が本当にTarget-neutralだったかを圧力試験します。

## 非目標

- 全Target対応
- design systemの完成
- pixel-perfectなGUI Editor
- Web frameworkをTEI標準として固定すること
- B2C大規模負荷のproduction保証
- custom UIを禁止すること

## 完了条件

`order_entry`の5 Patternを、Driverと同じApplication meaningから第一Web TargetへProjectionできること。

その過程で得た「共通Presentation meaning」と「Web固有implementation」を明示的に分離し、第二Targetによる検証へ渡せること。

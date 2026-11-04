# AI-first Pipeline Candidate Fixture（テスト専用）

このディレクトリは、生成AIが**既存Go Language Plugin Referenceの構文・Operator pin・型連続性・source span**について、Goコードを変更せずテスト候補を追加するための最小の足場です。**正式なTEI DSL、業務意味の正本、永続Scenario、Coverage DB、新Runnerではありません。**

## 追加方法

1. `baseline.json`は固定された基準線です。既存ケースを変更せず、新しい`*.json`ファイルをこのディレクトリへ追加します。
2. `contracts`に、テストで使用する明示的なOperator契約（id/version/fingerprint/effect/inputType/outputType）を記載します。
3. `cases`に一意な`id`、`source`、`pins`、`catalog`と`expected`を記載します。契約名の配列はそのファイルの`contracts`への参照です。
4. `expected.stage`は`ok`／`parse_error`／`lower_error`のいずれかです。`ok`には`output_type`、エラーには`error_code`（`syntax`／`invalid`／`missing`／`ambiguous`／`effect`）を明記します。必要なら`step`、`span_token`、`first_operator_byte_start`を指定できます。
5. Repository rootで`go test -count=1 ./reference/dsl-language-plugin -run '^TestAIPipelineCandidateCases$'`を実行します。正規CIがexact-headでPASSするまでは新しい候補を合格済みとしません。

**位置**は元`source`のUTF-8バイト半開区間です。たとえば`　rows/order |> std.where`の最初のOperator開始位置は17バイトです。表示上の文字幅・UTF-16 editor offsetと混同しません。

## 既存Scenario CLIとの境界

mainには既に `cmd/tei/scenario.go` の `scenario verify / generate / run / check` と、`reference/driver-scenarios/order-entry-pattern-smoke.json` の6-Step保存済みReferenceがあります（`tei.driver-scenario/v0`）。本ファイル群は**そのScenarioとは独立した、Language Pluginの構文・型・Pinだけのtest-only入力**です。Pipeline候補を業務Scenarioへ自動挿入したり、ScenarioのExpected・Canonicalへ昇格させたりしません。既存Scenarioの対象範囲はPattern/Browse/Validation/合成Observationであり、本テストのPipelineはその新しい実行Operationではありません。

JSONの重複メンバーは既存 `scenariotest.RejectDuplicateJSONMembers` を再利用して拒否します（単なる `json.Decoder.DisallowUnknownFields` は重複キーを拒否しません）。AIが同じ `source`/`expected` を二度書いて見かけの合格を作る入力を許しません。

## 証拠上の注意

- Goテストは既存`ParsePipelineSyntax`→`LowerPinnedPipeline`→`ComposePinned`を使います。実Goコードで受理していない表層をテスト内で「対応済み」と仮定しません。
- `baseline.json`にある主要4件はGoテスト内でSourceと期待結果を別途固定し、AIがファイルを削除・結果を書き換えて見かけのPASSを作ることを防ぎます。残るケースや新規ケースのExpectedも**実装のActualからコピーしただけでは独立Oracleになりません**。#2514の独立Expectedや既知Mutantとの照合とOwner Reviewが必要です。
- `std.invoke`の例は**効果付きOperatorが拒否される負例**であり、実Capability/Grant/Provider Writeの権限はありません。
- 全ケースが通っても、金額600の業務計算、NULL/未取得/partialの実データ評価、Decision Table Hit Policy、Unicode identifier lexer、Source Writer、CLI/Web/Studio、実DB Effect、全Source import closureを検証したことにはなりません。
- 新Caseを増やしてもCoverage分母が自動的に増えた・全件を検査したと主張しません。Coverage Obligationと独立Expected、実Scenario/Targetは[#2811](https://github.com/hat47x/tei-platform/pull/2811)、[#1180](https://github.com/hat47x/tei-platform/issues/1180)、[#2514](https://github.com/hat47x/tei-platform/issues/2514)の既存担当で管理します。

**次の小さい改善候補**：AIが追加した1ケースを別ファイルから読ませて正常・負例を1つずつ検証し、その後、候補の独立Expected出典と既知Mutant検出・候補レビューを既存のテスト工程へ接続する。新しい`tei test` CLIやAI Agent Runtimeを先行実装しません。

# AIが増やせるDriver Scenario候補（test-only）

このReferenceは既存の `tei.driver-scenario/v0`、`cmd/tei/scenario.go`、`scenariotest.Parse/Generate`を**そのまま使用**し、JSON Scenarioを新規追加するだけで既存CLIの検証をGoテストから実行する最小の足場です。正式な業務意味の正本、新Runner/CLI、独立Oracle、Coverage Store、外部Agent Hostではありません。

## テスト候補を追加する

1. `reference/driver-scenarios/`に`ai-candidate-<unique-id>.json`を新規追加します。既存`order-entry-pattern-smoke.json`と`ai-candidate-order-validation.json`、独立Expected、固定Anchorを変更しません。
2. `schemaVersion=tei.driver-scenario/v0`、一意のScenario ID/Step ID、実在するApplication/Pattern/Operationとその期待値を記述します。現行Schemaが扱うのは`resolve-pattern`、合成Browse、Canonical Validation、合成Observationの4 Operationだけです。
3. FixtureはScenarioと同じディレクトリにある`fixtures/`など**内部の相対パス**で指定してください。未知Operation、未対応値型（NULL/missing/decimalなど）、実Provider Write/認可/実Web/GUIはサポート対象外です。ケースの追加だけでそれらの機能を実装したことにはなりません。
4. 候補を追加後、完全なcheckoutと必要なGo toolchainがある環境で、`go test -count=1 ./cmd/tei -run '^TestAI'`を実行してください。テストはglobした全JSONをParseし、現行Scenarioの`verify → generate → check → pinned run`を実行します。固定2 Anchorの欠落・Step ID衝突を拒否します。
5. 正常/拒否の既知Anchorと、候補Expectedを改ざんした場合に`run`が拒否し古いplanが失効する負例はGo側で固定しています。追加CaseのExpected自体は**AIの提案**であり、実装Actualをコピーしただけでは独立Oracleとは認めません。

## 成立する証拠の限界

- `generated-pinned`かつ`verified=true`の結果は、**当該合成Fixture、Canonical/World、Plugin Lock、Scenario byte**に対する限定した実行結果を示します。実DB、外部Effect、業務Authoritative Owner Meaning Acceptance、Web認可、CLI/Web等価性、全必要SourceがLoadedだったことは示しません。
- `scenario verify`や`tei validate -json`の成功のみから対象の完全性を証明しません（#2822〜#2824）。`Declared/Discovered/Loaded/Applicable/Executed/Qualified`の根拠は#2811の設計票で別々に記録します。
- CLI実行計画とFixtureの指紋照合が成功しても、生成器の**宣言Sourceと実際のBuild Input**の差（#2595/#2819）、独立Expectedの共有由来（#2514）、対象外のCoverageは解消しません。
- 新候補はAgentごとの独立branchで増やします。機密データ、非公開Source/holdout、実Provider Credentialを公開のJSON/ログへ混入させません。公開・採用済みSuiteの世代一式性とConsumer受領は#2604の別資格です。

## 実行・資格

- 本ファイルとGoテストは**test-only候補**です。PR exact-head CI/Go結果が取得できるまで実行成功と記載しません。
- Goのテストコードを編集せずに新規`ai-candidate-*.json`を追加できることを検証しますが、Candidateが新規の未知バグを見つけたことは保証しません。#2514の独立Expected、固定Mutant、事前Coverage分母を組み合わせた実測が次段です。
- 所有責務は#1180 Scenario・CLI/Web、#2811 AI-first検証足場、#2514独立Oracle、#2621 Authoring、#2515公開/隔離、#1163 Evidenceの既存レーンを維持します。

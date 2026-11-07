# Verified Composition BOMとVersion Matrix：標準構造・テスト・Artifactからの導出（設計候補）

- **製品方針**：検証済みPlugin構成パターンのBOM提供と、複数版組合せ試験をTEIの**基本機能**に位置付ける（2026-11-07ユーザー方針）。
- **採用状態**：本書の具体的Schema、公開CLIコマンド、発行権限、認証済み実装は未採用・未実装。最小のread-only ReferenceとIssueの受入条件のみ。
- **管理Issue**：[標準Verified Composition BOM #3267](https://github.com/hat47x/tei-platform/issues/3267)。関連：#1016 #951 #1251 #943 #1097 #1180 #2514 #2600 #2589 #3262 #2831 #1174 #2390。
- **公開方式の比較**：[Draft #3265](https://github.com/hat47x/tei-platform/pull/3265)のOpen/unpromised（上流はStable保証だけ明示、下流の依存/Adapter追随責任）。BOMはOpen依存にも適用し得るが、Contract公開とHost効果の資格を統合しない。
- **本書の責務**：既存の正本からBOM/Matrix/検証履歴を**導出する方法**。第二のPackage Manager、Resolver、Lock、Runner、Artifact Store、Acceptance、認可/Trust Engineは作らない。

## 1. BOMという言葉の二つの異なる意味

Jenkins Plugin BOMは、複数Pluginの動作実績がある組合せをVersion付きで提供する。同じBill of Materialsでも、CycloneDX/SPDXの**SBOM**は主に部品と依存関係・由来の記述であり、Plugin同士の実行時互換性の試験合格とは違う。TEIは同じ標準データから**構成上のBOM／実績付きのQualified Composition BOM／SBOM輸出**をそれぞれ投影する。出力用途を混同しない。

| 意味 | 必要な主語 | 保証しないこと |
| --- | --- | --- |
| Composition Pattern | 必要Contract/Operator/Role、許容Version範囲、required/optional、Function/Target等 | 実Providerが存在・選択・Link済み |
| Resolved Composition | #951/#1016で固定した選択Provider、Contract fingerprint、実行Target/Scope、Lock | 実BuildとConformanceが成功したこと |
| Built Composition | 実Build入力閉包、Builder/Toolchain、ArtifactSet、各logical PluginのSubject binding（#943/#1097） | 実呼出、意味等価、現在Grant |
| Tested Composition | 同一対象・実Artifact/Attempt/TargetのConformance＋Independent Expected＋negative/coverageのRun Evidence | 業務MeaningのOwner受理、他Targetや将来版の適合 |
| Qualified BOM（発行候補） | 発行Subject、対象Scope、基準・適用版、検証した個別組合せのexact Lock/Evidence、claim ceiling、現Trust | 無期限・無条件の互換性やProduction許可 |
| Export SBOM | 対象実Artifactに含まれるDependency/Version/Provenance、公開制限 | Plugin統合試験のPASS |

**BOMを新しい設計正本にしない**。元Pattern/Accepted Meaning/Lock/Run/Artifactのread-only materialized view＋証拠を伴う推奨記録とする。Patternの可能な組合せ全体と、実際にPASSしたtupleは別集合。

## 2. 標準データ構造の最小写像

既存の標準Record/Relation/Sequence、Source Ref、Version/Validity Facetだけで表現する。

```text
CompositionPattern --requires--> ServiceContract@version-range
  --allows--> ProviderCandidate@version-range
  --applies-to--> Function/Target/Scope/Feature
  --requires-test--> Scenario/IndependentExpectedRef
ResolvedCase --selects--> Contract+Provider+Artifact exact identities
  --materializes--> Build/ArtifactSet
TestAttempt --observes--> ResolvedCase & real Consumer/Target
  --uses--> Scenario@rev & independent Expected@rev
  --produces--> pass / fail / skip / unknown / partial
QualifiedBOMEntry --cites--> ResolvedCase + TestAttempt(s) + Coverage
```

Relationは**異なる型の辺**とOccurrence/Scope/Versionを保持。単一の`depends-on`Edgeや「合格」の一ビットにまとめない。Provider AとBの個別Conformance PASSをA+Bの統合PASSへ合算しない。多Tenantや期限付きGrantでは実呼出の権限を現在Hostで別判定する。

### VersionとArtifactの分離

`Contract semantic version` ≠ `Contract fingerprint` ≠ `Provider version` ≠ `Bundle release` ≠ `exact resolved Lock` ≠ `Builder/Toolchain` ≠ `ArtifactSet digest`。依存側の範囲指定は候補収集の宣言で、BOM上のPASSは**完全なexact tuple**にだけ付く。前後版のRangeを横断した推論はTest Coverageと具体的なCompatibility Law/Qualificationの根拠がある場合のみ段階的に提案する。

## 3. Version Matrixの標準利用体験

通常の利用者は「受注分析に使える検証済みのPlugin構成を提案」「今回の更新で影響する組合せだけ検証」と頼める。AI・CLI/MCP・Studioは同一のtyped Query/Matrix/Receiptを別の投影として使う。新しいコマンド名は未採用、製品で実行可能なCLIとして宣伝しない。

**入力例（設計説明。採用済みDSLではない）**

| 軸 | 候補 | 種類 |
| --- | --- | --- |
| order.query Provider | 1.4.2、1.5.0 | Contract適合済み候補 |
| report.render Provider | 2.0.1、2.1.0 | 任意Provider／`none`も明示的候補 |
| Target | Go static-linux、Go static-windows | 物理Target条件 |
| Test profile | order-integration-v2 | 対象に独立して固定した期待値と必要負例 |

**順序**：①Design Context Queryで実SourceとRequiredを検査→②既存ResolverでTargetごとのexact候補を解決→③Matrix計画（少数は全直積、大きいときはpairwise/境界/変更差分）→④既存#1180 Driver Scenarioを実行→⑤same-attempt actual Test+Build/Artifact/SourceのEvidence join→⑥#2514の独立性/判別力/coverage→⑦限定的Qualified BOM行へ物化→⑧発行・Consumer Disclosureは別判断。

**組合せ爆発を避ける**：3軸で各10候補なら全直積1,000。対象Scope・互換/非互換制約の妥当な事前除外、pairwise Coverage、前回BOMの差分、境界値、変更した依存Closure、危険な3-way/4-way依存の明示追加を組み合わせる。pairwise実行の25行にPASSしても、未実行の残り975行や高次相互作用をPASSと記録しない。候補除外は`infeasible-by-constraint` / `unsupported` / `not-evaluated`を分けて理由を保持。

## 4. 最小2×2の独立ExpectedとReference

**既存の`tools/reference-verified-composition-bom/`**では、解決済みLockを受け取る範囲に限ってVersion 2×2を安定順に列挙し、Check Witnessをread-only照合する。自ら解決・ビルド・実テスト・Trusted Evidence発行を行わない。

| Consumer | Provider | Fixtureでの観測状態 | BOMへ移してよい主張 |
| --- | --- | --- | --- |
| 2.0 | 1.4 | 必須Check双方`pass` | `observed-pass`（Reference照合だけ、資格ではない） |
| 2.0 | 1.5 | 一つが`fail` | `failed`。推薦不可 |
| 2.1 | 1.4 | 一つが`skip` | `unverified`。推薦不可 |
| 2.1 | 1.5 | exact Lockなし | `unresolved`。推薦不可 |

Fixture自体は合成。Go標準ライブラリで有限の直積・上限・版ごとのLock一意性・独立Expected参照/欠落・SKIP/FAILED/UNKNOWNを検査し、`qualified:false`を固定。実TEI Conformanceの証明には使わない。最初のProduct最小縦切りでは同じパターンを#951/#1180の**実Consumer**に置き換える。

### 合格にしない反例

`validate.ok`やPlugin Catalog非空、単独Go Build成功、個別Plugin Testの横並び、Test runner exit0だがテスト名0件、SKIP、Tool result途中切断、Fixture Expectedを実装から自己生成したもの、古いSource/Artifact/Lock/Host Grant、Case名だけ同じ別Target、Replica delay/部分Read、先行Commit後ACK-lossは、いずれも該当上位Claimの根拠にならない。Source/BOMの公開が許可されてもRuntime Permissionや業務Effectを保証しない。

## 5. BOMの再利用・発行・更新

- **利用**：配布者が検証したBOMは推奨候補としてCatalogに置ける。案件ProfileではContract範囲とTarget/Scopeの適合を確認し、現行Resolverからexact Lockを得て利用する。利用者は標準Patternを再利用し、技術的なProvider/Versionの詳細はAdvanced表示で編集できる。
- **Open/unpromised**：公開者がStableを保証していないAPIへ下流が依存する場合、下流独自Adapterと検証行をBOMへ登録してよい。ただし公開者本人がその互換性を保証したとは表示しない。上流の新リリースによる影響は当該下流Tupleだけ再試験。
- **部分利用**：BOM 10個の中で3個だけ必要なら、対象のSelection/実Build Closureへ含まれない7個を自動リンクしない。BOMは全Bundle導入や余分なinitを強制しない。既存#1251 PBNで現物検証。
- **複数版**：論理Contractの二Major併存と、Go一Binary内の二版同一packageの実Link可否を区別。不可能なら該当Targetはunsupported、Process/Wasm等の別方式は別Runで判定。
- **履歴**：旧版で検証したBOMを変更・上書きせずimmutableなReleaseとして保存。新BOMを別Revisionで発行し、現在のPublisher/Trust/撤回を別照合する。Source A→B→A、Publisher失効、Bundle更新でも過去Runは歴史として残る。
- **差分再資格化**：#2600の実消費入力と独立Evidence Requirementsから、変化したTupleだけを未資格へ戻す。無関係Plugin追加やUI説明文変更で無条件に全BOMを再試験しない。
- **出力**：Matrix Table、Dependency/Graph、Version Compatibility Map、BOM JSON、必要時CycloneDX/SPDXの在庫表現を**同じ標準構造から派生**。機密Artifact URL/Scope/Hidden Dependentsの数は、実Consumer Disclosureによって表示を制約する。

## 6. Ownerと段階的な製品化

| 段階 | 必要最小限 | 正本/Owner | 非目標 |
| --- | --- | --- | --- |
| P-1 | 一FunctionのRequired/Loaded/Resolved/Artifact/Evidenceの現況と不足 | #3262 | 欠落を空集合PASS |
| P0-A | 2×2のMatrixとfixture projection、全状態の負例 | #3267とこのReference | 第二Resolver/Runner |
| P0-B | 実Provider/Consumer 2×2の正負Integration Test、same-attempt exact Lock | #951/#1016/#1180/#2514 | 合成FixtureをProduction資格化 |
| P0-C | Target実Build/Artifact/実Invokeの照合から限定BOM候補導出 | #943/#1251/#2600 | 全Target/実DB完了 |
| P1 | changed-based/pairwise＋境界＋3-way既知Mutantの組合せ | #1180/#2514/#2600 | Pairwise=全組合せ互換との推論 |
| P1 | Public Registry/外部Consumerへ受理済み限定BOM公開 | #2589/#2390/#2516 | 第三者発行権限の自己申告 |
| P2 | 実DB Effect/current Host Authorityと組合せ再資格化 | #1174/#1119 | BOM単体で本番の権限付与 |

## 7. 測定・撤回条件

測定項目＝初回Matrix作成指示数／全ケース決定論的再現性／Fixture版とIndependent Expectedの混同件数／wrong Provider/TargetへのEvidences誤結合／未実施の誤PASS率／Matrix拡張によるCI時間とキャッシュ効果／BOM再利用の手動設定削減量／Open依存版更新の下流修復量。

既存LockとScenarioから同一テスト束縛を導けず、本BOMのために第二の意味正本/Resolver/Runnerが必要になった場合は方式を再検討し、BOM機能を薄いMaterialized Projectionへ戻す。原型は**検証済み製品BOMではない**。本Issue #3267の受入を経て初めて実製品のQualified BOM発行を判断する。

## 参照

[Jenkins Plugin BOM](https://www.jenkins.io/doc/developer/plugin-development/dependency-management/)／[Jenkins BOMのCompatibility Testing](https://github.com/jenkinsci/bom)／[Gradle Platformと版制約](https://docs.gradle.org/current/userguide/platforms.html)／[GitHub Actions Matrix](https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-syntax)／[Nix exact lock](https://nix.dev/concepts/flakes)／[CycloneDX v1.7](https://ecma-international.org/publications-and-standards/standards/ecma-424/)。

**確認の上限**：本件のSource/Version Matrix Reference、Docs-only設計、隔離Go試験4件以外の製品実装・GitHub exact-head CI・二Plugin実Target統合・Conformance issuer/Host資格は未実施。

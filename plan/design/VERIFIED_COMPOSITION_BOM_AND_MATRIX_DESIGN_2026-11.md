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


## 8. 横断統合：Pairwise計画・Cache/SnapshotとBOMの利用資格（2026-11-07追補）

本節は「BOMを標準機能として届ける」ために、別チャットで進んだテスト設計、Design Context完全性、Cache/Snapshot Provider、大プラグイン主義（Open/unpromised）の議論を再接続する。**BOMはContextやCacheの真実を保存する第二正本ではなく、Sourceと実装のVersionから現在導出できる構成・試験結果の証拠付き投影**である。

### 8.1 Matrix計画と実行資格を二段に分ける

[Go Pairwise限定Reference](../../tools/reference-verified-composition-bom/README.md)には従来の2×2全直積に加え、**複数軸に対して全2項相互作用を含む行だけを決定論的に選ぶ**Planning-only実装を追加した。3軸×各2値の全直積8から4行を選択し、4軸×各3値の全直積81から2項全組を含む少数行を生成する。**このReferenceが証明するのは「無制約の指定軸に対してすべてのペアが計画に含まれた」までであり、PluginのVersion適合、実Targetで実行可能、テスト結果PASSを意味しない。** `FullCartesianSize`、`TotalPairs`、`CoveredPairs`、`Cases`と`qualified=false`を別フィールドにする。

Product化で追加すべき要件は次の通り。

| 問い | 必須の処理 | 誤って主張してはいけないこと |
| --- | --- | --- |
| Required Closureは完全か | #3262が対象Function/Targetの要求・Source/Loader/Disclosure・必要Pluginの欠落を示す | 見えていない依存が存在しない |
| 組合せは実現可能か | #951/#1016が各exact tupleのProvider/Contract/Scopeを解決し、解決不能・曖昧・非互換を理由付きで示す | Pairwiseに含まれた組合せなら実行可能 |
| どの組合せを試すか | small full、pairwise、boundary、changed-impact、既知高次interaction、negative constraintsを独立に選択 | Pairwiseの2項100%を全直積の検証済み100%と呼ぶ |
| 実際に何を使ったか | 実Build/Toolchain/Artifact/ProviderのSource/Runtime consumptionを確定したLockに束縛 | Lockがあるだけで実Link/Invocation成立 |
| Test/Expectedは独立か | #1180/#2514のScenario/独立Expected/Mutant、実Run/断片・SKIP・Tool切断を照合 | CI exit0や別AttemptのPASSの合算 |
| BOMの現在資格は何か | 実受理Owner/Publisher/Target/Grant/DisclosureをProduct側で必要に応じて照合 | 過去Evidence＝現在も許可/業務的に正しい |

**Constraint付きPairwiseの要点**：除外を行った後は、元の無制約ペア集合が全て実現可能とは限らない。適用不能なペアを`infeasible-with-reason`、Source/Resolver不足を`unknown`、解決済みで未試験を`not-run`、検査済みのペアを`covered`と分け、**到達可能な全ペア集合の証拠がある場合だけ**制約相対のCoverageを確定する。ReferenceはConstraintsを解かないため、Pairwise生成候補をただ除去して「100%」へ再計算する方式は不採用。

### 8.2 BOMとCache/Snapshotは相互利用するが資格を共有しない

別チャットで成熟した「CacheをBackend別メモ化ではなく正本からの派生Snapshotとして統合する」案は、既存[#3271](https://github.com/hat47x/tei-platform/issues/3271)へ管理Ownerを置く（2026-11-07作成済み）。BOM側は**実Provider Instanceを選ぶ構成要素・試験軸として含める**。

- **Build cache**：実Consumed Source/Generator/ToolchainとTargetが一致するときにだけBuild省略候補。Content digest一致でも別Toolchain/別ExpectedのTest Witnessは共有しない。#2600/#2831がOwner。
- **Test result / Qualification cache**：過去のConformance Evidenceは**そのexact tuple・Fixture/Independent Expected Revision・実Target・Build/Source closure**にだけ紐付く。旧PASSは歴史として残し、Test Profile変更やSource ABAからCurrent qualifiedを自動復活させない。#2514/#3267がOwner。
- **Derived Dataset Snapshot**：業務データに適用する場合、入力Source/時点/粒度/NULL・Partial/権限Scope/整合保証とそのProvider Instanceを保持。単にSnapshot byteが同じでも、別Tenant、別有効時点、非開示Scopeでは共有しない。#1641/#1708/#2582/#2622/#3271がOwner。
- **Provider Instance**：Memory、Local File、tmpを先行検証し、Redis/S3/DynamoDB/CDNは必要が示された場合のみ追加。Storage Plugin版と選択された**Instance/Region/Tenant/Capabilities/Retention**を別IdentityとしてBOMに含める。CDN purge ACKを全Edge消滅・権限取消とは呼ばない。
- **Persisted business fact**：Ownerが確定した受注金額等は派生キャッシュではない。Cache eviction、BOM再試験、Rule改版で暗黙に業務保存値を変更しない。#1641/#1174がOwner。

**状態の分離**：Cache hit／byte verified／Snapshot freshness／Test result valid／Artifact included／Conformance qualified／Host currently authorized／Business acceptedを一つの`valid=true`にしない。Reuseできる部分だけ再利用し、範囲不足・Unknown・Permission deniedの状態を保存する。

### 8.3 新たに導く否定試験（正式SSOTは#1180/#2514）

| ID | 変化/操作 | 独立Expected候補・棄却条件 |
| --- | --- | --- |
| VCB-P1 | 3軸各2版の8候補からPairwise計画 | 計画4行、2項12/12、qualified=false、Test未実行をPASSとしない |
| VCB-P2 | 4軸各3版の81候補 | 2項54/54、全直積より少ない行、固定順。実組合せ適否は未判定 |
| VCB-P3 | Pairwiseの候補行がConstraint上不可能 | 実行不能理由を残し、候補行を消しただけで制約相対Coverage100%としない |
| VCB-S1 | Providerは同版だが選択したCache InstanceのScope/Storage変化 | 対象TupleのActual-Selected Instance/Artifact/Expectedを再検査。無関係BOMを不要失効させない |
| VCB-S2 | Cache hitだが実Source A→B→A、旧Test Evidence付き | Hitを現在Conformance資格へ昇格しない。過去テスト記録は改ざんせず保持 |
| VCB-S3 | Partial sourceからDerived Snapshotを作り、BOM側では全件と解釈 | SnapshotのPartial/unknownを維持、BOM全件テストPASSを許さない |
| VCB-S4 | 別Tenantと同じByteでCacheを共有、Grant失効 | 参照候補の発見と実Data開示・実呼出のGrantを分離 |
| VCB-S5 | Optional Open APIが上流更新で破壊 | 必要な下流Adapterと該当Functionだけ資格を失う。Stable保証を上流へ偽帰属しない |
| VCB-S6 | 高凝集Bundleの未選択Pluginに変更 | 選択Closureが不変で実Artifactも非干渉ならBOM不要失効なし。Go initが混入したらBuild側失格 |

**実装順**：①今回のPairwise pure plannerの局所Reference → ②#951/#1016の実合法Tuple・#3262のRequired Closureを投影 → ③#1180のRunner/Independent Expectedと実Artifactへ結合 → ④#2600の正しいCache/Build省略＋#3271のProvider Instance比較 → ⑤Qualified BOM発行・外部Host・異Tenantは必要な独立実証後。#1119 G1/G2/G4に新しいHard Gateを足さず、Production資格は個別のOwnerの受入条件に従う。

**今回の証拠**：隔離Go 1.23.2でPairwise追加6テストと従来4テストの計10件PASS、`go vet`PASS、`gofmt`差分なし。現在のGitHub branchのexact-head CIと実Resolver/実テストRunner/Cache Providerは**NOT RUN**。新Core kind、BOM発行権限、独立Cache Host、CDN機能の採用を意味しない。

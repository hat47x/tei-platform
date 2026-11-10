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


## 9. 組合せ制約のあるVersion MatrixとBOM同士の合成（2026-11-07追加／最小Referenceあり）

### 9.1 「存在するLockだけでPairwise」が解決できる範囲

§8の`PlanPairwise`は宣言軸に含まれる値の**無制約直積**を前提とする。現実のPlugin依存は「Provider 2.1はContract 1.x非対応」「Go static Targetでは同一package二Majorを同時Linkできない」「あるProviderはTenant Bで使用不可」などの**制約付き候補空間**である。無制約Plannerで選んだ行から不成立行を後で削除すると、元の組合せ全体に必要なペアが脱落する可能性がある。

これを薄く確認するため、[PR #3269](https://github.com/hat47x/tei-platform/pull/3269)の限定Go Referenceに**`PlanResolvedPairwise`**を追加した。新しいResolverを作るのではなく、既存#951/#1016等から将来提供される**解決済みと申告された exact Lock の有限集合**を選択可能な行候補として入力する。候補間の2項ペアについて決定論的にgreedy set coverを計画し、どの選択行がどのLock Fingerprint由来かを保持する。

**ただし候補一覧自体は完全と検証していない**。実装は次の資格の境界を固定する。

| フィールド／主張 | 正確に言えること | 言えないこと |
| --- | --- | --- |
| `fullCartesianSize` | 宣言軸から算出した全直積のサイズ | そのすべてが実現可能 |
| `candidateCount` | 提供されたexact Lockの件数 | Catalog全件、実Provider起動可能件数 |
| `declaredPairs` | 無制約宣言軸の全2項候補数 | そのすべてが実現可能 |
| `candidatePairs` | 提供されたLock行に現れた2項集合の数 | 他の未発見候補を含む全実現可能ペア数 |
| `coveredCandidatePairs` | 選択したLock行がカバーする**提供候補内の**ペア数 | 全可能なPlugin組合せの網羅率 |
| `unrepresentedDeclaredPairs` | 宣言軸には存在しても、入力候補に現れないペアの数 | そのペアが「不可能」「拒否」「存在しない」と確定 |
| `candidateUniverseCompleteness: unknown` | Source/Registry/Resolver列挙の完全性を未証明 | 完全な可行候補列挙 |
| `qualified:false, claimCeiling:planning-only` | Test実行前の試験計画 | 検証済みBOM、実Contract/Runtime適合 |

**3軸×2候補**で宣言全直積8行、うち4行だけがexact Lockとして提供される負例では、宣言ペア12組のうち候補に現れた9組だけを検出する。Plannerが9/9を覆っても、残り3組は`unrepresented`であり「infeasible」とは言わない。入力のLock配列順によらず結果を固定。異TargetのLockを混在させるなら`target`軸を明示し、Lock.Targetと不一致なら拒否する。**Scope/Host Grant/契約意味/元Source等をこの文字列Lockの存在で証明しない**。

このReferenceは供給された候補の2項集合をgreedyに被覆するだけで、最少ケース数を保証しないし、全互換性制約の解決もしない。選択しなかった合法候補の個別重要Caseを必要Scenarioから落としてはならない。

### 9.2 BOMを合成するときに「PASSの和≠合成PASS」を維持する

BOM-A（Query Provider＋DB Access）とBOM-B（Report Renderer＋Export）が各々検証済みでも、A+Bの組合せが検証済みとは限らない。二つのBOMが同一Contractの別Versionを要求したり、同一DB/Host Effect/Memoryの共有、組織Scope、非可逆Data Projection、Target別ABI等で相互干渉する場合がある。**BOMをBOMへ依存させることは許すが、BOM-A ∪ BOM-B のQualificationは自動では継承しない。**

1. **Pattern上の合成**：必要Contract/Facet/Version rangeと負のConstraint、Open/unpromised依存を結合し、衝突を診断。これ自体はCandidateであり、推奨済み完成構成ではない。
2. **Exact Lock合成**：#951/#1016の**既存解決**を再実行し、選択対象・Scope/Targetの一致と二重採用／ambiguous Providerを検出。旧Lockの単純連結は禁止。
3. **Build/Artifact合成**：#1097の一物理Artifact多Plugin binding、#1251の同梱だが未選択要素非干渉、#2600の実Build closureに合わせる。別Artifactを同時配備した時のLink/Load/init問題を見落とさない。
4. **試験合成**：BOM-A/Bで独立に実施したテストを履歴として保ち、**跨BOM辺**のContract/Effect/Function/Scopeを含む同一構成の統合Scenarioを#1180/#2514に要求する。共通部分は再利用可能でも未検証の横断部分は`unverified`にする。
5. **BOM発行**：対象がexactに一致し、独立Test/Target/Source/Artifact/Consumer Disclosureの証拠が揃った部分だけ限定Claimとして発行する。発行者／Conformance Issuer/Runtime Host/OwnerのAuthorityは別。

特に、BOMを**依存解決の優先順位（暗黙のProvider選択）として使わない**。BOMは推奨された既知組合せを示す選択ヒントであり、Consumerの要求・実Providerの互換性・Host現在権限に優先する第二Resolverではない。

### 9.3 受入条件と現実の実装順

| ID | 独立Expected候補 | 不合格にする過大主張 |
| --- | --- | --- |
| VCB-C01 | 8候補中4 exact Lock、Candidate Pair 9/12、covered=9、unrepresented=3 | 9/9を全宇宙100%や欠けた3ペアの不可能証明とする |
| VCB-C02 | Lock候補の順序を入れ替えても同じ生成行・Lock ID | Loader順序をBOM推薦優先度に流用 |
| VCB-C03 | 255種の非空部分集合（3軸2値の8 tuple）で、選択行が提供された全2項を覆う | 未観測Tupleを合成・捏造、Case上限を超えて黙示省略 |
| VCB-C04 | 二TargetのLockを混ぜる際はTarget軸必須、宣言値と実Lock.Target不一致拒否 | 論理Contract一致だけで異Target互換と称する |
| VCB-C05 | BOM-A/Bの両方PASSだが跨BOMの互換/副作用Case未実施 | 二つのQualifiedを合算してQualified Union |
| VCB-C06 | 同一Contractの2 Major要件衝突、Go static TargetでLink不可 | 解決済みPlugin ID列だけで実Binary合格 |
| VCB-C07 | Open API依存のAdapter改版、上流未保証・対象Functionだけ再資格化 | 第三者BOMにPublisherのStable保証を虚偽付与 |
| VCB-C08 | Scope/Source/Expected欠損、partial Catalog、古いArtifact/Evidence | 未発見を不可能、不完全試験をPASSへ昇格 |

**実装済み範囲**：Goの`supplied-locks-pairwise`は構文上のLock候補集合と全2項を照合する**試験計画のみ**。入力の不変性とTarget混在の棄却を含む。孤立したGo試験で追加4 Test（うち255部分集合の全列挙）がPASS。**GitHub exact-head CI、#951/#1016実Resolver/実Provider、#1180実Runner、Constraintの完全Source Closure、Qualified BOMの真正発行、BOM-A∪BOM-B実ConformanceはNOT RUN。**

**次の最短縦断**：#3262 Design Contextで対象範囲と未収集を確定 → #951の既存解決結果をJSON候補として取得 → `supplied-locks-pairwise`でLock付き試験計画を生成 → #1180の既存Test Scenarioを各exact Target/Artifactへ実行 → #2514で完結度と独立Expectedを検査 → #3267で限定的BOM行へ物化。別Planner/Resolver/Runner/Publisher Authorityは追加しない。


## 10. Trait義務・設計Coverageと同一AttemptのWitness束縛（2026-11-07／横断・限定Reference）

### 10.1 Atomic TraitがBOMに追加する「検証前の義務」

[#3279](https://github.com/hat47x/tei-platform/issues/3279)ではTraitを、Provider/Pluginに実装を強制する**原子的・合成可能な契約**として管理する。Trait Profileは集合の便宜的な投影であって単一の巨大契約ではない。Required Traitを宣言しても実Providerの存在を保証せず、`generated`・`provided`・`manual-required`・`external`のようなFulfillment方式と対象Target、実Source/Artifact、Conformanceを別に照合する。**manual-requiredなのに実装がない状態**は設計／開発環境で明示的に拒否できることを基本要件にする。

BOMは既存#951のContract解決結果に対し、Traitごとに`Required → Bound Implementation → Fulfillment Check → Conformance Evidence`という**追加の依存／検査義務**をread-onlyに投影する。Trait名一致、Interfaceの型一致、標準実装の存在、Open APIへのアクセス可能性は、当該組合せで実動作検証済みであることを意味しない。

BOM再利用では、次の変更を異なる再資格条件として扱う。

| 変更 | BOMへの影響 |
| --- | --- |
| Open/unpromised上流APIの更新 | 依存下流Adapterと該当Trait/Contract/Scenarioの再検査。PublisherのStable保証を偽造しない |
| 手動実装が欠落したTrait | 該当Target/Plugin構成を実行前にunfulfilled。別のデフォルト実装へ無断fallbackしない |
| 実装は存在するがIndependent Expected未取得 | 構造的なFulfillment候補まで。Conformance未検証を維持 |
| Generator/Toolchainのみ変化 | 実消費ClosureとArtifactに基づく当該Tupleの再資格化。無関係Pluginは非干渉 |
| Tenant/Grantのみ失効 | 過去BOM/Conformance履歴は保持。現在の開示・Operation/Effectを拒否 |
| 新CLI/Studio Adapterだけ追加 | 選択された実Consumer/Buildへの影響を実測。全Targetを無条件に再ビルドしない |

### 10.2 Coverageは少なくとも四つの異なる分母を持つ

| 指標 | 分母 | 誤って推論してはいけないこと |
| --- | --- | --- |
| **Version Pairwise Coverage** | 宣言軸の二項、または存在が確認された候補Universeの二項 | 2項100%を全組合せの相互作用/実PASSとする |
| **Requirement/Scenario Coverage** | #3262で対象Scopeに必要と確定した機能・非機能・負例・Trait obligation | 未Loaded要件やunknown依存を「不要」とする |
| **Code C0/C1 Coverage** | 実BuildしたSource/Target/Package・計測器で観測可能な命令/分岐 | C0/C1が高ければ正しいIndependent Expected/意味を検証したとする |
| **Decision Graph/Client Reaction Coverage** | #3274 HTTP Graphの必須Node/Edge/GuardとClient/Recovery分岐 | HTTPレスポンス受信や描画ノードだけでServer Guard/Effectを実証したとする |

[#3273](https://github.com/hat47x/tei-platform/issues/3273)のIssue→Design Context→未カバー義務→AI候補→独立Expected→実CLI TestというDogfoodを、#3267のMatrix planと合成する。**Versionだけの差分で変化する義務**と**Source/要件/Rule/Client Contract変更で増える義務**を分ける。両方のRequired Closureが閉じたと説明できない場合は「未計測／未知」を保持する。AIがテスト生成しただけ、静的Skillが存在しただけでBOMのEvidenceを発行しない。

### 10.3 合成したTest PASSが別Attempt・Artifact/Scopeのものなら発行しない

既存の`Project`は外部から与えられた`Check`を`lockFingerprint`で単純束縛する限定Referenceであり、当初から`qualified:false`だった。ここに別の純粋関数`AuditWitnessBindings`（[witness_audit.go](../../tools/reference-verified-composition-bom/witness_audit.go)）を追加し、**同じAttemptオブジェクトの中で**全必須Checkが完了していることを確認する。

- 固定した`Lock fingerprint`、`Source fingerprint`、`Artifact digest`、`Target`、`Scope`、`Scenario digest`、`Independent Expected digest`と、送られてきたAttempt記録が一致するかを照合する。
- 必須CheckはCandidate Attempt側の報告ではなく、**要求側で先に固定**する。0 Step、途中打切り、SKIP、TIMEOUT、Partial、Unknown、Evidence Ref欠落をPASSにしない。
- 同じLockへ複数Attemptが送られた場合は`ambiguous-attempt`として拒否し、最新時刻・配列順・最大PASS数による自動選択を行わない。実行元が特定AttemptをPinして再投入する必要がある。
- 一つのEvidence Refを異なるAttemptへ再利用した報告、未知Lock、重複Checkを受理しない。同一Attempt内でのみ複数Checkが一つの実Runを構成する。
- 結果は`not-run / ambiguous-attempt / identity-mismatch / incomplete / reported-failure / reported-pass`。**`reported-pass`は入力JSON相互の一致にすぎず、独立Expectedの真正性、実Providerのロード/Invoke、Conformance Issuer、現在Grantを証明しない**。常に`qualified:false,claimCeiling:fixture-identity-consistency-only`。

この関数はBOM「発行」ではない。#1180/#2514の実Driver/Independent Expected/Run Evidence、#943/#2600の実Build Closure、#951のResolver、#1174のHost Effect/Receiptを同一Attemptに接続して初めて、Qualified-in-Scope BOM発行を検討できる。

### 10.4 HTTP/Client Reactionを同じPlugin組合せの追加テスト対象にする

[#3274](https://github.com/hat47x/tei-platform/issues/3274)が所有するREST例外系では、OpenAPIで示した成功/例外レスポンス型だけでは**HTTP応答喪失・Client Reaction・副作用確定不明・回復観測**を定められない。BOM対象Version/Targetが変わった際、同じServer/Client/HTTP Adapterの組合せで次のDriver負例を再実行する義務を設ける。ClientがGo/Web/Desktopそれぞれで別Transportを持つ場合はTarget別に評価する。

| ケース | Check/受入 |
| --- | --- |
| API Response/Problem Type版変更 | Problem Details/typed Outcome/Client Reactionの写像。HTTP `detail`文字列解析に依存しない |
| 更新後のACK喪失・Commit unknown | Idempotent Operation identity/Receipt/独立Readの経路。無条件再送で二重書込みしない |
| 401/403/409/412/422/429/503とRetry-After | 機能/Host Grant/情報秘匿を守り、Retry Hintをwrite許可へ昇格させない |
| Graph Node/Edge/Guard更新 | Source ID/条件/必須Guardを保ち、Graph表示済みを実Guard動作済みとしない |

Server/ClientのGraph構造、HTTP Adapter、Conformance EvidenceをBOMに**リンク**するだけで、専用HTTP Runtimeや別RunnerをBOM側へ導入しない。具体的な実践は#3274と#1180へ委譲する。

### 10.5 次に実装すべき縦断（P0の昇格条件）

1. #3262の一Function read-only Contextに、Required Contract/Atomic Trait/Test Obligation/Targetを**不足とともに**表示する。未Loadedと不在を混同しない。
2. #951/#1016の実Resolverが返した**同一Revision**の候補Lockを`supplied-locks-pairwise`へ渡す。候補Universe不足は`unknown`とする。
3. #1180既存Scenario/Independent Expectedと#943実Target Builderで、Provider 2版×Consumer 2版の実Runを取得する。C0/C1、Version Pairwise、要求、HTTP ReactionのCoverageを別票にする。
4. 今回の`AuditWitnessBindings`相当のjoinを**実Execution Receiptの正本から**読み出し、実Source/Artifact/Target/Scope/Scenario/Expected/Attempt一致を再確認する。別RunのPASS合算を拒否する。
5. 当該SubjectのOwner受理・Trust/Grant/Disclosureが別に成立した場合だけ#3267の限定BOMを発行。単独Reference PASSや履歴Cache hitは資格条件を満たさない。

**実施状況**：設計は本節へ反映。Trait/HTTP Graph/Test Coverage/C0/C1の製品統合は未実施。`witness_audit.go`および合成テストは新規追加したが、最新GitHub HEADのCI結果を確認するまでは実装上の合格を主張しない。Trusted/actual E2EとOwner受理はNOT RUN。


## 11. 履歴Capability EvidenceのProducer規則変更をVersion Matrixに組み込む（2026-11-08／限定検査）

### 11.1 実際に発生した差分：表示上の`v0`は同じでも選択規則が変わる

[main merged #3337](https://github.com/hat47x/tei-platform/pull/3337)の初回実記録は`reference/workspace`で12個の宣言Capabilityのうち6個が`unselected`だった。これは`database-instance`の技術名`postgres`とCapability ID `database.postgres`を直接文字列照合する**当時の限定計測規則に依存した記録**であり、実Providerが使われていない事実ではない。その後の[main merged #3350](https://github.com/hat47x/tei-platform/pull/3350)・[#3363](https://github.com/hat47x/tei-platform/pull/3363)では`SupportsDatabaseProvider`に基づく、既存Validatorに合わせた選択判定に変更された。同じ`Schema`と`RuleVersion:v0`の**ラベル**でも`ruleFingerprint`が変わる。

[closed/unmerged #3379](https://github.com/hat47x/tei-platform/pull/3379)を引き継いだ[main統合 #3490](https://github.com/hat47x/tei-platform/pull/3490)は保存済み歴史的raw JSONと現行`workspace.Load`→`plugincoverage.Derive`を同じPlugin/Version/Kind+ID単位で照合し、古い規則に由来する`unselected`→新規則の`selected`とProducer Rule差を反証するテストを追加した。旧rawを現在の選択資格へ書き換えない。

### 11.2 BOMに再利用可能な証拠の判定表（新Coreスキーマではなく受入規則）

| 歴史的Claim | 変更後の判断 | 不正な推論 |
| --- | --- | --- |
| Plugin/Capabilityが`unselected`（旧Ruleで観測） | 当時の歴史的Observationを保持し、現行World/Rule/Environment/Scopeで再導出 | 未選択が永続的に必要能力欠落／欠陥 |
| 同じPlugin Package/Version文字列 | `Kind+Capability ID+Contract/Provider Implementation`、Target/Scopeを再照合 | 同Packageだから全Trait/Capabilityが使われた |
| `Schema`/`RuleVersion`表示が同じ | ルール内容の`ruleFingerprint`と実Producer Implementation、Source Closureを照合 | 表示上同じ`v0`ならTest Witness流用可 |
| 新Ruleで`selected`になった | 合法なCandidate/Bindingとして選択済みの**範囲**のみ。#951のProvider一意解決、#1251実Load/Invokeへ別送 | `selected`→実Provider呼出→Conformance PASS |
| Build Report／Stage Driverの自己報告 | #3336のinvoker/invokedと実ディスクArtifactの再取得、#2600の実消費Closureで照合 | Self-report、Tool Exit0、cache hitが正式BOMを認定 |
| 同じ2×2 Version Matrixを再計画 | 先にProducer/Required Context/Source/Target/Rule再資格。合法な候補2項集合と全4行の実独立Testを別票へ | 既存Pairwise計画100%がRule変更後も互換性100% |

**対象相対の失効**：宣言や選択数が変化したからといって全PluginのSemantic Source/Build Artifact/過去Test結果を無条件に失効させない。変わったProducer Ruleを実際に用いた**Selection/Eligibilityの派生Claim**を再導出し、実選択・実Build入力・必要Testの依存経路で影響範囲を限定する。対象依存が不完全なら`unknown`に保つ。旧`unselected=6`と現行の値を「改善KPI」へ自動変換しない（測定規則が違うため）。

**P0の接続**：#3262からRequired Source/Rule/Relation/Loader/Targetの閉包を得る→main #3337/#3350/#3363の現行`Capability Report`を**現在の規則で再生成**→#951の合法解決/Version Lock→#1251の実Inspector Provider A/B→#1180/#2514の独立Expectedと同一Attempt/Artifact→#3267の限定BOM行。BOM発行は本節を読んだだけでは可能にならない。

**資格上限**：main #3490は過去/現在のReference Ruleの**読取結果を比べるだけ**。実Provider選択、Go実Link/Invoke、CLI/Studioの二Consumer、Runtime/Grant、Trusted Conformance、BOM配布は**NOT RUN**。#3297自体もmainからdivergeしたDraft・未統合であり、新Targetや新Hard Gate、第二Resolver/Runner/Cache/Authorityを追加しない。


## 12. BOM再利用の三つの鮮度境界と実Call-site依存（2026-11-08／現mainへの引継ぎ）

[main統合 #3490](https://github.com/hat47x/tei-platform/pull/3490)ではPlugin-first計画§18とCapability Coverageの歴史Rule比較Go回帰を限定実証した。旧[#3379](https://github.com/hat47x/tei-platform/pull/3379)はbaseが古く、Goテスト/vetが通っても`coverage_test.go`のgofmt差でHealth失敗し、現mainの計画更新を上書きしないようclosed/unmergedで引継いだ。歴史raw Evidenceは変更しない。

BOMの`reusable/needs-recheck`は少なくとも**三つの異なる鮮度判断**を含む。

1. **Consumer Managed Source**：main統合[#3477](https://github.com/hat47x/tei-platform/pull/3477)はWorld同値でもSource Index FPが変わり得るSource-only改訂を、実TEI CLI／Scenario Contextで有限実証した。外部Tool完全配送・全Source原子的Snapshotは別。
2. **Imported Provider Source**：[main統合 #3479](https://github.com/hat47x/tei-platform/pull/3479)はConsumer WorldとSource Indexが不変でも外部Providerのraw byteとLocateImportedSource fingerprintが変わり得る実Go負例を提案。exact-headの合格・Provider Nested Load原子性・Build Inputの実消費はまだ必要。Consumer内FPを外部Plugin依存全体のハッシュと扱わない。
3. **Producer Rule/Implementation**：main統合#3337の履歴Capability Coverage（当時6/12未選択）は、main #3350/#3363のDB Capability選択Rule変更後の`selected/unselected`と同じ意味の資格として使えない。Schema/RuleVersionラベルが同じでも`ruleFingerprint`差分に基づき再導出し、影響する正確なProvider/Environment/Targetだけ再検査する。

さらにmain統合[#3446](https://github.com/hat47x/tei-platform/pull/3446)の静的Caller調査では、`servicehost.NewSelectedServiceCarrierInvoker`の非試験呼出し元が0だった。既存Consumerへ選択済みProviderが実配線された証拠がない以上、上の三つのFPが全部一致しても、実CLI/Studio Invoke資格・Verified BOMには達しない。

**VCB-F01〜F05（#1180/#2514へ渡す、NOT RUN）**：
- **F01** Consumer内Sourceだけ変更・World不変→Source IndexとScenario Requiredを再照合。
- **F02** Consumer World/Index不変、Imported Provider rawのみ変更→必要Reference Source occurrence/fingerprint/actual consumptionを再照合。ない場合`unverified`。
- **F03** RuleVersion文字列不変、Rule Fingerprint/選択結果変更→旧Policy由来SelectionをBOMに流用しない。
- **F04** 無関係な未選択Plugin資産だけ変更→実Closureが非干渉なら全組合せを誤失効させない。
- **F05** CLI/Studio双方のtyped Result一致・旧Builtin直呼び→実Selected Provider/Host側Callsite未観測ならBOM実呼出しのClaimは成立しない。

**組込み順**：#3262の目的相対Required/Source Closure→#951/#1016 exact SelectionとRule Producer鮮度→#1251 D1実Read-only Inspector Provider/実Host-origin Call→#943/#2600実Build/Toolchain/依存Closure→#1180/#2514同Target/Attempt・独立Expected→#3267限定BOM。外部Referenceの意味受理・現在Grant/Disclosure・Publisher保証は引き続き別。**D1と4組合せ実E2E、Conformance、BOM発行は未実施**。#3297はmainとdivergedしたDraftなので、現main §18で先に受入判断し、同内容を競合した長命PRから無確認mergeしない。


## 13. RQ-037現行mainへの前方移植と資格境界（2026-11-08）

本設計とGo Referenceは、旧[Draft #3297](https://github.com/hat47x/tei-platform/pull/3297)の限定成果として作られ、その後現mainにもBOM固有のGoコードと本設計の§1〜10が存在する。旧PRが変更した`requirements/TEI_PRODUCT_REQUIREMENTS.md`、PAS、Product Gap Auditと大量のDesign Index差分を一括移植していない。理由は、現在mainの**RQ-036＝ドメイン全容Model Map**が旧PRの**RQ-036＝BOM**と衝突するためである。現mainで旧RQ-036のBOM監査混入が判明し、[修復Draft #3592](https://github.com/hat47x/tei-platform/pull/3592)でRQ-037の要求・PAS・Audit・Masterの正確な対応を修復中（旧Draft #3538は移植元）。**RQ-035 Workbench／RQ-036 Model Map／RQ-037 Verified BOM**をそれぞれ独立のOwner／Evidenceで扱う。

### 13.1 今回のReferenceができること

- Full Cartesianの宣言版列挙、決定的Pairwise Coverage、既に解決済みのexact Lock候補集合に**限定した**Pairwise計画。
- 外部提供されたSource/Artifact/Target/Scope/Scenario/Independent Expected/Attemptおよび必須Checkの整合性照合。
- 被覆不足・複数Attempt・結果欠落・指紋不一致・部分取得／SKIP／TIMEOUT／UNKNOWN・Evidence再利用の誤報告拒否。
- `qualified:false`を維持し、サンプル入力がすべて一致しても正式Conformanceの発行・実Permission付与を行わない。

このGoコードは自前のPlugin Resolver、Builder、Runtime Invoker、Scenario Runner、Evidence Issuer、Artifact Storeを持たず、上位が渡した値を**純粋な入力データとしてチェックするReference**である。型付き`PlanPairwise`の二項網羅は完全なConstraint Solverや3-way/full Cartesian保証ではない。ReferenceのRule/Lockの実在やPublisher署名もここでは保証できない。

### 13.2 次の製品受入の最短経路

1. **S2**：main統合[#3381](https://github.com/hat47x/tei-platform/pull/3381)の局所Go選択Provider B呼出を、通常`tei inspect`のDispatchと実Studio Bridgeから**別々に**呼ぶ。CLI Helper/Studio Host APIの同一Goテストだけなら製品二Consumer資格にしない。選択B→実Aが同じ結果を返してもreject。
2. **Source/Manifestの束縛**：main[#3477](https://github.com/hat47x/tei-platform/pull/3477)（Consumer Source-only）、[#3479](https://github.com/hat47x/tei-platform/pull/3479)（外部Provider raw/Manifest-only）、[#3490](https://github.com/hat47x/tei-platform/pull/3490)（Capability Rule Fingerprint）のそれぞれをProviderの実消費Closureと照合。各局所ReferenceのPASSを合算して全Source完全性としない。
3. **S3**：#951のexact Selectionと#943/#2600の実Build/Artifact、#1180/#2514の独立Expected/実Runnerの**同一Scope/Target/Attempt**へ束縛。その実証されたCaseごとにBOMの適格範囲を示す。今回の`AuditWitnessBindings`のJSON上の一致はTrusted Witness発行ではない。
4. **2×2**：実Provider二版×実Consumer二版を4行個別実行し、未実行/不可能/未知を残す。元来S2の「CLIとStudioの2 Surface」と混同しない。Host Grant/Disclosure/Effect/Owner Meaningは同様に独立。

**状態**：本節は現在の要求修復Draft #3592と、mainに存在する限定Referenceツールの実装／受入を混同しないための設計追補である。新ブランチのexact-head CI、現mainとの変更閉包、Reviewer/Owner判断は別検収が必要。実2×2 Conformance、通常CLI＋実Studioの製品S2、BOM正式発行・本番権限は**NOT RUN**。

**次の検証の区別（2026-11-09）**：[Draft #3540](https://github.com/hat47x/tei-platform/pull/3540)の通常CLIとStudio Hostへの静的参照Provider接続（S2-reference）は、正式な別Providerに差し替えた製品S2でも、Publisher保証・外部Host Grantでもない。また[main #3583](https://github.com/hat47x/tei-platform/pull/3583)が示すC-1配備CommandとC-2 Service Requestの異質性を前提とし、二者択一を先行固定しない。C-4外部Host委譲を選ぶ場合もReceipt未受領をConformance PASSへ変換しない。

# Model Viewの横断資格：読取・検証・同一性・被覆・操作を混線させない（2026-11-09）

- **状態**：研究・受入候補（未採用）。製品Schema、API、標準State Enum、共通Host、第二Source/Reader/Policy/Proof Registryを導入しない。
- **固定基準main**：`3c8bebefef30a9407fef71c0e6a5a76680f46719`。本文はmain実装と別Draftの限定実証を区別する。実装現況の正本は[docs/status.md](../../docs/status.md)。
- **既存の正本**：[複合図表・トポロジー設計](../design/TOPOLOGY_COMPOSITE_VIEWS_AND_PIPELINE_2026-10.md)、[統合研究（第10〜15節）](TECH_RESEARCH_INTEGRATION_2026-10.md)、[三視点のSource・Reader対応](../design/THREE_VIEW_DIAGRAM_CROSS_REFERENCE_PROJECTION_2026-11.md)、[#1124 Studio](https://github.com/hat47x/tei-platform/issues/1124)、[#2497 写像の保存・編集逆写像](https://github.com/hat47x/tei-platform/issues/2497)、[#2516 現在の開示](https://github.com/hat47x/tei-platform/issues/2516)。他Ownerのテスト・図面/Source実装や未統合PRを本書に複製しない。

## 1. 新たに判明した実証上の問題

同じWorkspaceで`model catalog`を読めても、結果の輸送上の`ok`や`worldFingerprint`は**Validation合格を意味しない**。[PR #3728](https://github.com/hat47x/tei-platform/pull/3728)は2026-11-09に**main統合済み**。`model catalog`は検証NGでも読取に成功し、`validation.ok/errorCount/warningCount`を別表示する。現mainの`internal/operations/model_catalog.go`と`model_catalog_validation_test.go`で確認した。診断本文は`validate -json`を正とし、読取成功・World FP一致・Model Validation OKを別の判定にする。

[Draft #3321](https://github.com/hat47x/tei-platform/pull/3321)の実CLI反例では、Manifest名だけが異なる二WorkspaceでWorld・SourceIndex・Snapshot fingerprintが一致することがある。#3321 exact head `701cb4ad197091f6268379fa68dbdce95013516f`の[専用CI #42435108689](https://github.com/hat47x/tei-platform/actions/runs/42435108689)は`completed/success`、PR記録上Node 17/17件＋Go実CLI 3 Traceが成功。比較関数の`same-observed-load-context-only`も**同一物理Root・同一Reader Attempt・同一Principalの証明ではない**。

[Draft #3543](https://github.com/hat47x/tei-platform/pull/3543)は、実CLIの曖昧候補Aと一意候補Jを同じloopback Workbenchへ渡し、`Initial A: Inspect 0 / List A: 0 / List J: 1`のRPCを実DOMイベントで観測した。[専用CI #42487892720](https://github.com/hat47x/tei-platform/actions/runs/42487892720)は`completed/success`。この成功は実ProductionのGrant・Policy、Graph/Command・別scope・ABA・全Source closureの保証ではない。

[Draft #3518](https://github.com/hat47x/tei-platform/pull/3518)の型付きNative Model Map M0は、既存World/Catalog/SourceIndexからGraphをread-only導出する試み。`resolved`は**Catalogの宣言対象が見つかったという局所的な意味**であり、Coverageは`partial-projection`。Event/State/Lifecycle/Effect・Provider/Current Disclosureなどを含む全構造の列挙は証明しない。別Ownerの[PR #3492](https://github.com/hat47x/tei-platform/pull/3492)は2026-11-09に**main統合済み**。`reference/diagram-crossref/real_workspace_admission_test.go`は実WorkspaceでQuery/RelationのCatalog→Inspect→Trace、およびRelation親Data/Target Dataを限定照合している。ただし複数operationの原子的Read・全Diagram生成・独立Meaning受理は未実証。

上記は互いに異なるProducer/Consumer・Head・操作である。**成功したCIの件数を合算して「完成したVerified Model Map」とは呼ばない。**

## 2. 一つのViewの利用資格は五つの直交した問いに分解する

| 独立の問い | 本文に表示／記録すべき根拠 | 誤った短絡 |
| --- | --- | --- |
| **Read**：その対象を読めたか | 実Reader/CLI・Attempt、Root/Workspace Manifest、Source/Index版、対象のkind/id、結果0件と読取失敗 | CLIの`ok=true`をモデルが正しいと扱う |
| **Validation**：そのSourceは要求された検証に合格したか | 必要Rule集合、診断のseverity、Source-root findings、現在の検証版。未提供なら`unexamined` | Model Catalog/Graphが描画できたので`valid` |
| **Identity**：他View/操作と同じ対象・同じ読取文脈か | Source-owned vs caller-supplied scope、qualified subject、Manifest identity、Source Index/Reader Attempt、View/Consumer epoch | `worldFingerprint`や`Snapshot token`だけが一致すれば同じWorkspace |
| **Coverage**：問い合わせに必要な関係を網羅したか | 対象Q、Role/Relation宣言scope、Reader closure、omitted/opaque、`partial-projection` | Map上に線が無いので業務Relationが存在しない |
| **Eligibility**：今その操作・公開・実行をしてよいか | 一意のSource anchor、現行Grant/Disclosure、#267受理、必要ならEffect/Target/Artifact独立観測 | 候補`resolved`、Inspect成功、CI成功からWrite/Business Effectの許可を推定 |

この5行は**評価票の見出し**であり製品の一括`verified` enum/APIではない。異なるConsumerが同一文字列・同一Fingerprintを持っても、必要な証拠が不足する軸は`unknown`／非公開扱いに留める。内部`unknown`の詳細を許可前の公開Viewへ漏らさない。

### ClaimとUI出力の関係

- **Catalogの読取成功**は、編集中・検証NGのモデルを閲覧可能にするため有用。`read succeeded`と`validation failed`は矛盾しない。#3728の検証欄が採用される場合も、診断本文を別の正本へ重複コピーせず既存`validate -json`へ参照する。
- **Mapのリンク`resolved`** は特定の投影内のReference候補の解決であり、全Relationの被覆、現在のSource Jump、Owner意味受理ではない。M0のGapを表示できることは、M1以降の情報が不存在である証拠ではない。
- **Inspectの0件** は曖昧対象に対する拒否の成功を示し得る。呼出しが0件だったことは、Source自体の不在や別入口の安全性ではない。**Inspectの1件**も正しい現在Source/Grantへの操作を保証する追加証拠を必要とする。
- **Workspace tokenの一致** と **Workspace identityの一致** と **同じ実Reader Attempt** は別。完全コピーされたManifestやABA（A→B→A）では必要な条件を再資格化し、過去の成功を現在の資格に自動再利用しない。

## 3. 既存実装へ渡す独立Expectedと負例

新しい共通Acceptance/Scenario/Source Registryを作らず、各Ownerが既存のテスト・実Consumerへ引き渡せる**評価用判別票**を記す。結果欄は実行して初めて記入し、本節の記載はPASSではない。

| ケース | 固定する入力／意図的な変更 | 独立Expected（問い合わせの上限） | 現在の基線・Owner |
| --- | --- | --- | --- |
| **MV-01 Transport≠Validation** | 未定義TypeRefを含む同じWorkspaceで`model catalog`と`validate` | Catalogは閲覧可能でも`validation=NG`。World FP一致を検証成否へ転用しない。現mainに存在するCatalogの`validation`欄と`validate`結果を、別読取として突き合わせる | #3728/#3674、CLI JSON契約 |
| **MV-02 Snapshot≠Workspace** | 同SourceのコピーでManifest名のみ変化、二つの`tei trace` | Fingerprint/Token一致を許してもWorkspace同一断定を拒否。Manifest不一致を別観測し`unknown`維持 | #3321の実Go CLI＋Node |
| **MV-03 同Manifest≠同Attempt** | Manifest/Source bytes同一の二Root、または同RootでReaderを再実行 | manifest+FP一致だけでは物理Root・Open handle・同一Attemptを未証明とする | #3321、#3181、#2505 |
| **MV-04 Partial Mapの偽不在** | M0図表にEvent/Effectの線が無い | `partial-projection`を維持しEvent/Effect不存在を主張しない。受理済み対象なら独立Source/Role Readerへ | #3518、RQ-036、#3492 |
| **MV-05 曖昧→一意の実操作** | 同じloopback Workbench内 A→JをInitial/List操作 | AのInspect増分0/0、Jが1。受理したRPC引数・Source対応を独立比較 | #3543（focused runのみ既証拠） |
| **MV-06 stale/ABAへの再資格化** | 同Source内容へA→B→A、Source Index/Grant/Projection epoch変更 | 同じHashでも旧Inspect・Source Jump・Current Grantを自動復活させない | #1124/#3327/#2516 |
| **MV-07 Relationとdirect edge** | Traceの直接Edge 2本と、RelationのSource宣言ID／roleが未取得 | `direct-edge-candidates`は表示可、独立Relation・bounded absenceは`unknown` | #3321/#2881/#2497 |
| **MV-08 数理最適性≠業務適合** | 固定Owner・direct write・must-linkの合成Partition最適値 | 制約した既知入力上の最小値を示すのみ。実CRUD閉包・Effective Effect・Owner受理は未証明 | [Draft #3662](https://github.com/hat47x/tei-platform/pull/3662)／#3250/#3255 |
| **MV-09 公開Sliceの非干渉** | 同じ許可済Public Slice、異なるPrivate Relation/検証診断 | 公開Graph・件数・Tool/ARIA・診断がPrivateの有無を漏らさないか別検証 | #2516/#2972 |

### 先行・後続の作業と撤回

最初の**P0**は、既存#3543の実Workbench A/J、#3321のSource/Manifest二世界、#3728のCatalog/Validateを**各担当の固定head・同一元Source条件で**個別に資格化する。別CLI/Runnerを新設しない。**P1**は#3518のM0 typed link/Gapsを#3492の実Catalog/Inspect/TraceとSource Indexへ束縛し、未解決・部分被覆を安全に表示する。**P2**で初めてGraph/Table/関連表の横断と#3255の分割評価に接続し、変更の局所性・偽の不存在・認知負荷を[#2018](https://github.com/hat47x/tei-platform/issues/2018)の評価へ渡す。実Studioの正式Grant、Production Effect/Headless、実Build/ETL、Owner受理は独立Gate。

**撤回条件**：この区別を保つために巨大な共通State機械・第二Registry/Reader/Proof Store・専用Hostが必要になる、あるいは既存の検証・Source探索費用やFalse Qualifiedが増える場合は、共通UI機構を先行採用せず、限定read-onlyの表示・報告手順へ戻す。

**現状の資格**：本書は公開されたGitHubのPR・固定head CI・現mainの静的照合に基づく**受入計画**。#3543/#3321の既存限定CI成功は参照したが、この9ケースの一括E2E、実Policy/Production、実Owner Source受理、Graph/Table/Map M1、Go/Node新規試験、認知負荷実測はすべて **NOT RUN / NOT PROVEN**。#3728/#3518/#3662などの別Draft内容を製品mainの実装済みとしない。

## 4. 2026-11-09 夜：Catalog検証結果と実Source Navigationの最小接合

**事実の更新**：本節の対象mainでは、#3728のCatalog Validationと#3492の実Workspace Query/Relation Source Admissionは**統合済み**。一方、#3518のModel Map M0、#3543のWorkbench A/J結合、#3321の実CLI Load-context比較は各Draftであり、局所試験が成功しても現在mainの単一Consumer動作とは扱わない。

MV-01の次の一手として、既存`internal/operations/model_catalog_validation_test.go`に`TestUnsoundWorkspaceCatalogInspectAndTraceKeepValidationDistinct`を追加する候補を同一PRで提出。**元のSourceは未定義型参照を含み、Loaderの構文処理は成功する**という小さい実Workspaceを使い、次の条件を同じfixtureで確かめる。

| 対照・対象 | 期待する事実 | 断定してはいけないこと |
| --- | --- | --- |
| `CatalogWorkspace` と `ValidateWorkspace` | Catalogはread-onlyで成功、`validation.ok=false/errorCount>0`、`ValidateWorkspace.HasErrors()=true`。Workspace/World/Indexの観測値は一致 | Catalogの`ok`を検証合格とする／Fingerprint一致から全Required closureを主張 |
| `Catalog`の`semantic-data scn.X` | Preludeではない唯一のindexed declarationとして探索可能 | 無効TypeRefがあるDataに業務上の意味受理・実Target適合が成立 |
| `InspectSubject` と `TraceSubjectSources` | Canonical Subjectを読めて、Ref/Artifactの観測が一致 | Source Indexだけから追加のCanonical Entityや全Relationを採用 |
| 複数operationの連続実行 | 同じ固定fixtureで各操作が対応するSource/Index FPとWorkspace Manifestを観測 | **同一Read Attempt／原子的な複数operation Snapshot**。別々に`workspace.Load`する現APIでは未証明 |

この回帰試験は既存の`workspace.Load`／Catalog／Inspect／Source Traceを**そのまま呼ぶ**。新たなVerifier／Proof Schema／Source Index／Transport Adapterを追加しない。正例は「未検証モデルも観察・修正候補の探索が可能」であり、「欠陥モデルを安全に実行できる」ではない。Workspaceのこの単純な未定義TypeRef検証だけでは、Rule Closure、Provider、Target、Owner Acceptance、Current Disclosure/Grant、UI上の再表示/修正を証明しない。

**次の判別優先順位**：P0は上記実Go局所回帰のexact-head結果確認。P1は既存#3492の実Relation Parent/Targetと#3518のtyped Map M0を**同一World/Catalog/Index生成経路**で照合するが、M0の`partial-projection`を変更せず、Source Index由来のLocatorを意味上のRelation IDに代用しない。P2は#3543の実Workbench A/Jに検証NGの選択時表示・現在Source変更・Graph/Command入口を追加する検討を#1124 Ownerへ返す。#3728の判定欄がStudioへ自動表示済みとは主張しない。

**検証資格**：本節はmainに存在するコードの静的照合と**新しい単一Go回帰試験候補の追加**。新規試験の実行/CI/Review結果は当該PR exact headを参照し、結果未確認の時点では **NOT RUN / NOT PROVEN** と記す。M0 GUI、実Provider/DB、Production Grant/Policy、複数Consumerの同一Attempt、Owner意味受理、認知負荷測定は引き続き未検証。

### 4.1 Source-rootとin-process Catalogの追加反証候補（静的所見）

今回のGo回帰は**未定義TypeRefのSemantic診断**だけを対象とする。mainのコードを別途静的に確認すると、関連する別の論点がある。

- `CatalogWorkspace`は`CatalogWorld`を作成した**後で**`catalogValidation(loaded)`を設定する。したがって`CatalogWorld(world, idx, ...)`を直接利用するin-process Consumerでは、`CatalogValidation.OK`がGoの初期値`false`のままでも、**Validationを実行して失敗したという証拠にはならない**。現実の不適切なConsumer到達経路は未確認。`false`だけからfailedとする誤表示は止め、個別Caller/用途を調べる。
- `catalogValidation`は`SourceRootFindings`のerrorを数える一方、`WorkspaceValidation.HasErrors()`は確認したmainコードでは`Direct`／`Diagnostics`のerrorを確認し、`SourceRootFindings`を走査しない。**Source-root-onlyのerrorがあるWorkspace**について、これらをそのまま合否として比べれば不一致の可能性がある。実CLIの`validate -json`がどの規則を使うか、発火可能なRoot負例とともに[#2823](https://github.com/hat47x/tei-platform/issues/2823)／[#3674](https://github.com/hat47x/tei-platform/issues/3674) Ownerへ返して調べる。現在のProduct CLIが誤判定したという実証結果ではない。

これらは本PRの既存4 operationテストの成否とは**独立の未検証問題**。正式なValidation状態Schemaや新検証器を先に増やさず、同一Rootの`CatalogWorkspace`／`ValidateWorkspace`／CLI JSONを実Source-root-only負例で照合した上で最小対応を判断する。**NOT RUN / NOT PROVEN**。

## 5. 2026-11-11：Source-root-onlyエラーを跨ぐValidation資格の整合（小さい実装修正候補）

### 2026-11-13 実装追跡と負例の追加

- この節の **MV-10/11の修正候補は実装済み**。[#4392](https://github.com/hat47x/tei-platform/pull/4392)をmainに統合し（merge `a8e7e01dccca4f5feed60707122ddb52236832b3`）、**統合コミットに紐づく16/16 checks成功**を確認した。現行mainの`WorkspaceValidation.HasErrors()`はSource-root errorを含み、`MapWorkspace`は同一`workspace.Load`結果で`catalogValidation(loaded)`を埋込Catalogへ渡す。下の2026-11-11の「候補」「未設定」は**当時の発見履歴**であり現行実装状態ではない。
- **MV-10/11の接続負例（独立Draft）**：[PR #4557](https://github.com/hat47x/tei-platform/pull/4557)は、検証NGでも実Workspaceの`CatalogWorkspace`／`MapWorkspace`／`LocateWorkspace`が同じ不合格・read-only・partial/NotDerivedを保つかを**未定義型**と**欠落したmanifest Root**の2条件で照合。最初のhead `8e01a54b7588ab860a68d23a49d4cc51eb72dc00`は、`RuleCandidate`を誤ったCanonical IDでLocateしたテストfixtureの指定ミスによりRepository Health失敗。新head `bb08dbda934382b3174b2aa7bcc7d2e871905b7b`へ明示的に宣言した`scn.x`へ直したfixtureを載せ、現mainに非強制マージして**CI再検証中**。失敗した旧headや#4392の成功を新headの合格に代用しない。
- [#4521](https://github.com/hat47x/tei-platform/pull/4521)は実公開Relationの正常Map／Locate対応を担当する別Draftであり、この検証NG負例と混合しない。両者の結果の合算はStudio/Provider／Meaning／編集資格の統合証拠ではない。低水準`CatalogWorld`が直接呼ばれた場合の未実施Validationも引き続き別契約。

**更新した事実**：[#3808](https://github.com/hat47x/tei-platform/pull/3808)の検証NG WorkspaceによるCatalog／Validate／Inspect／Source Trace回帰と[#3543](https://github.com/hat47x/tei-platform/pull/3543)の実Workbench A/J限定結線は**main統合済み**。[#4239](https://github.com/hat47x/tei-platform/pull/4239)の歴史Source A/B→実CLI/MCP受領、[#4279](https://github.com/hat47x/tei-platform/pull/4279)のOperationSurface代表Bindingの非完全性負例もmain統合済み。ただし成功した各試験は同一Provider/Attempt/Owner業務意味のEnd-to-end証拠ではない。[旧Draft #3518](https://github.com/hat47x/tei-platform/pull/3518)はそれ自体**未統合でclosed**だが、後継[#3866](https://github.com/hat47x/tei-platform/pull/3866)の型付きModel Map M0は[統合PR #4079](https://github.com/hat47x/tei-platform/pull/4079)でmainへ移植済み（`internal/operations/model_map.go`）。M0の`Coverage=partial-projection`は維持され、業務三視点・実Effect・現在権限の完成を意味しない。

### MV-10：同じ実Workspaceで、manifestの存在しないRootだけがerror

[#2823](https://github.com/hat47x/tei-platform/issues/2823)にあるRoot処理の実装を2026-11-11のmain `2202baa96e5e845c996da62c8f8df27e00d45c43`で照合した。現行の`WorkspaceValidationReportFrom`（CLI/MCP共通投影）と`catalogValidation`は、`SourceRootFindings`の`error`を不合格に含める。一方`WorkspaceValidation.HasErrors()`は`Direct`と`Diagnostics`だけを見るため、**Source-root-only errorを見落とす静的な規則差**があった。これは新しいValidationの意味判断ではなく、既存の不合格判定への一致問題である。

最小修正候補は、`WorkspaceValidation.HasErrors()`へ`SourceRootFindings`のerrorの検査を追加すること。診断本体の複製や第二Validator／Schemaを作らず、既存APIの利用者がCatalog/CLI報告と異なる不合格判定をしないようにする。

| 実Workspaceの固定対照 | 型付きHasErrors | Catalog Validation | CLI/MCP共通Report |
| --- | --- | --- | --- |
| `semantic`正常＋宣言した`nonexistent` Rootが存在しない | `true`が必要 | `ok=false`、error件数≥1 | `ok=false` |
| `semantic`正常＋存在する未知`future` Root | `false`が必要（warningのみ） | `ok=true`、warning件数≥1 | `ok=true` |

本変更の実Goテストは`internal/operations/validate_test.go`へ、既存`writeWorkspaceFixture`を使って上記の**二つの実ファイル系Workspace**を作る。Loader→Source Index→Catalog/Validate→共通Reportの既存経路を使用し、入力全件やHostによる外部実行の保証はしない。異なるOperationの実行は別々のLoadであり、結果のFingerprint一致から**原子的な同一Attempt**を推論しない。

### MV-11：Native Model Map内のCatalog Validationは「未実行」と「不合格」を区別する

現行mainの`MapWorkspace`は`CatalogWorld`を直接呼んで`ModelMap.Catalog`に格納するが、`CatalogWorkspace`とは異なり`catalogValidation(loaded)`を設定していなかった。そのため**健全な実Workspaceでも埋込Catalogの`validation.ok=false`がGoの初期値で残る**。これはModel Mapを「Validation NG」と表示した場合に実害となる、具体的なProducer差である。

最小修正は、`MapWorkspace`が持つ**同一Load結果**で`catalogValidation(loaded)`を計算して埋込Catalogへセットすること。新しいValidation正本/Readerは作らない。実ファイル系Workspaceの「健全／未定義TypeRef」「missing source root／未知root warning」を既存`CatalogWorkspace`・`ValidateWorkspace`・`WorkspaceValidationReportFrom`と対照する回帰試験を提出する。Graphの`resolved`が業務Relation受理でないこと、`partial-projection`であることは変えない。

**停止線**：この修正によって、低水準の`CatalogWorld`自体へ「未検証」の第三状態は追加しない。Read-only Mapの内部で検証済み結果を作るだけで、現在のSource・Relation全件閉包、Map Renderer、Studio受領・編集Grant、Owner Meaningは依然 **NOT PROVEN**。

**独立の残課題**：`CatalogWorld(world, idx, ...)`を直接呼ぶ際の`Validation.OK`初期値は、`CatalogWorkspace`の検証済み結果と別物。今回の修正はこの低水準関数に検証資格を付与しない。また将来の三視点図やCRUD表で「Rootが未取得」と「対象が存在しない」を混同してはいけない。

**実証資格**：この節は提案した最小製品修正とその二つの回帰条件を記録するもの。変更のGo試験、実CLI/MCP、Studio・Model Map・Runtime/Provider、完全なSource Closure、Current Grant/Owner MeaningとCIが**exact HEADで確認されるまでNOT PROVEN**。新Core/Host/Reader/Policyを増設しない。

## 6. 2026-11-11：Validation一致からSourceの現行性・Claim再利用へ飛躍しない受入線

**追加の横断所見**：MV-10/11で修正するのは、`HasErrors`・Catalog・Mapに渡る**同一Workspace内容に対する検証成否の表現**である。これにより、別Root／別時点での同じSourceの消費や業務Claimの再利用が保証されるわけではない。

- [#4329](https://github.com/hat47x/tei-platform/pull/4329)は、ConsumerのWorld/Source Index Fingerprintや**実CLI応答13件すべてが不変**でも、Import先Provider raw／Manifestの内容が変わる二世界を実CLIで観測した、と記録している。Consumerが必要とする1-hop依存がIndex外または未観測なら`unknown`に止め、旧Fingerprintを現行資格として配布しない設計候補。**同PRは現時点でOpen／未統合**、Providerの真正な宣言と全Closureは未観測。
- [#4393](https://github.com/hat47x/tei-platform/pull/4393)は、独立したRequired依存の閉包と、実際に消費した入力集合の閉包を分離している。ただし`dependencies_closed`と`consumption_closed`の真偽値は**合成fixtureの作者が与えた前提**であり、実Providerから二閉包を観測した証拠ではない。
- [#4335](https://github.com/hat47x/tei-platform/pull/4335)は、DerivedIRと宣言されたData Context束縛でread-only変更影響を`retain/recheck/unknown`に分類するDraft。`retain`は既存の資格維持候補であって、`Read`・`Validation`・`Eligibility`の新しい合格ではない。実SQL Worker/Provider/DBの閉包は未証明。
- [#4396](https://github.com/hat47x/tei-platform/pull/4396)の実CLI Catalog/Validate→独立Node三要素Consumerには、固定headの専用試験成功の記録がある。これはReaderのValidator判定を**別プロセスへ渡せる**限定根拠であり、Source Attemptの原子性、実三要素Role、Current Disclosure/編集権限まで移すものではない。

### MV-12：五つのView資格と、必要・消費・観測Closureの対応表

| Consumerが利用したい判断 | このPRが改善する範囲 | 追加で必要な独立証拠・拒否条件 |
| --- | --- | --- |
| **Read/Validation**：このWorkspaceは読めた・不合格か | MV-10/11：Root error／warningと型エラーの局所判定をCatalog/Map/Typedで一致させる | 各OperationのReader Attempt・実使用byte。低水準`CatalogWorld`を呼ぶだけならValidationは未実施 |
| **Identity**：このModel Mapと別Consumerは同じ対象か | 同じfixtureでWorld/Source Index fingerprint・Manifestの観測一致を試験する | 別Root／別Attempt／Import先Providerの真正なSourceは別照合。Fingerprint・表示Key一致だけでSource-owned同一性に昇格しない |
| **Coverage**：必要なRole/Source・実Consumedの集合が閉じたか | Mapの`partial-projection`と未資格Roleを保持する | 独立Required集合と実Consumed集合のSource/Revision/Providerを各Ownerが観測。未知や未取得を0件／不在に変換しない |
| **Change impact**：このClaimは再利用可能か | 変更前後のValidator判定だけを再確認する候補 | Required+Consumedの閉包、影響経路と非単射Mappingを独立Expectedで検査。変更を観測しただけでCommitや業務Effectを主張しない |
| **Eligibility**：今Viewから編集・公開・実行できるか | **このPRによる新規資格なし** | 現Source/Consumer epoch、Ownerが受理した意味、現在のDisclosure/Grant、一意Source anchor、#267のtyped Preview/Acceptanceを独立に確認 |

**次に引き渡す反例**：同じ基礎Workspaceで`validation.ok=true`／Map links不変の状態から、Consumer Fingerprint外のProviderを改版する。Catalog/MapのValidation表示は直ちに「業務Claim現在有効」へ昇格せず、RequiredにProviderが含まれる場合は#4329の`unknown`、実Consumedを未観測なら#4393の`unresolved-impact`相当として**別々に停止**させる。公開UIに出す内訳はCurrent Disclosureに従い、Private Providerの存在を表示漏洩しない。実Studioでの失効／旧DOM拒否は#1124のOwnerに残す。

**計画上の優先順（2026-11-13更新）**：P0の#4392はmain統合・統合コミット16/16成功まで確定。次は#4557の検証NGの二負例と#4521の正常Relationの各exact-head結果を別々に確認し、失敗したテストFixtureをProductの失敗と誤認しない。P1は#4329/#4393のRequired・Consumed証拠を各Ownerが独立に測定できる場合だけConsumerへ移す。P2は同じ条件で#4396の実CLI→Node/三視点と#1124の実Workbenchへ接続し、正しいUnknown・旧証拠失効・操作拒否を検証する。P3はセルフビルドの選択／実消費ArtifactやETL/MFTのManifest／Receiptに類似の証拠要件が反復した場合に、個別実行のOwnerへ**評価票だけ**を引き渡す。第二Reader/Runner/Host/Proof Storeや共通の`verified`状態は追加しない。

**資格**：この節は既存の他担当Draftの記録・コード・CI上限を整理した研究計画。横断MV-12実行、Providerの真正な閉包、現行Policy、Studioの実UI、実Build/ETL業務Effect、Owner受理および認知負荷比較は **NOT RUN / NOT PROVEN**。

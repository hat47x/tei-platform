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

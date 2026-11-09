# Model Viewの横断資格：読取・検証・同一性・被覆・操作を混線させない（2026-11-09）

- **状態**：研究・受入候補（未採用）。製品Schema、API、標準State Enum、共通Host、第二Source/Reader/Policy/Proof Registryを導入しない。
- **固定基準main**：`3c8bebefef30a9407fef71c0e6a5a76680f46719`。本文はmain実装と別Draftの限定実証を区別する。実装現況の正本は[docs/status.md](../../docs/status.md)。
- **既存の正本**：[複合図表・トポロジー設計](../design/TOPOLOGY_COMPOSITE_VIEWS_AND_PIPELINE_2026-10.md)、[統合研究`10〜15](TECH_RESEARCH_INTEGRATION_2026-10.md)、[三視点のSource・Reader対応](../design/THREE_VIEW_DIAGRAM_CROSS_REFERENCE_PROJECTION_2026-11.md)、[#1124 Studio](https://github.com/hat47x/tei-platform/issues/1124)、[#2497 写像の保存・編集逆写像](https://github.com/hat47x/tei-platform/issues/2497)、[#2516 現在の開示](https://github.com/hat47x/tei-platform/issues/2516)。他Ownerのテスト・図面/Source実装や未統合PRを本書に複製しない。

## 1. 新たに判明した実証上の問題

同じWorkspaceで`model catalog`を読めても、結果の輸送上の`ok`や`worldFingerprint`は**Validation合格を意味しない**。[Open PR #3728](https://github.com/hat47x/tei-platform/pull/3728)は、検証NGでもCatalogを閲覧できる性質を残し、`validation.ok/errorCount/warningCount`を別表示する実装候補。PR記録では、Windows 11／Go1.27.0の実Binary・局所テストで`validate`との対応を確認した。ただし**本評価文書はそのPRのコードを採用・統合せず、現mainが当該フィールドを提供すると仮定しない**。

[Draft #3321](https://github.com/hat47x/tei-platform/pull/3321)の実CLI反例では、Manifest名だけが異なる二WorkspaceでWorld・SourceIndex・Snapshot fingerprintが一致することがある。#3321 exact head `701cb4ad197091f6268379fa68dbdce95013516f`の[専用CI #42435108689](https://github.com/hat47x/tei-platform/actions/runs/42435108689)は`completed/success`、PR記録上Node 17/17件＋Go実CLI 3 Traceが成功。比較関数の`same-observed-load-context-only`も**同一物理Root・同一Reader Attempt・同一Principalの証明ではない**。

[Draft #3543](https://github.com/hat47x/tei-platform/pull/3543)は、実CLIの曖昧候補Aと一意候補Jを同じloopback Workbenchへ渡し、`Initial A: Inspect 0 / List A: 0 / List J: 1`のRPCを実DOMイベントで観測した。[専用CI #42487892720](https://github.com/hat47x/tei-platform/actions/runs/42487892720)は`completed/success`。この成功は実ProductionのGrant・Policy、Graph/Command・別scope・ABA・全Source closureの保証ではない。

[Draft #3518](https://github.com/hat47x/tei-platform/pull/3518)の型付きNative Model Map M0は、既存World/Catalog/SourceIndexからGraphをread-only導出する試み。`resolved`は**Catalogの宣言対象が見つかったという局所的な意味**であり、Coverageは`partial-projection`。Event/State/Lifecycle/Effect・Provider/Current Disclosureなどを含む全構造の列挙は証明しない。別Ownerの[Draft #3492](https://github.com/hat47x/tei-platform/pull/3492)は実Catalog/Inspect/Traceと三視点Source対応の個別試験を進める。

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
| **MV-01 Transport≠Validation** | 未定義TypeRefを含む同じWorkspaceで`model catalog`と`validate` | Catalogは閲覧可能でも`validation=NG`。World FP一致を検証成否へ転用しない。現mainに新欄が無ければ現在の`validate`から別照合 | #3728/#3674、CLI JSON契約 |
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

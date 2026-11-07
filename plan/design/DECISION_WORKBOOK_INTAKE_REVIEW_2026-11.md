# Validation・Derivation・FlowのExcel決定表調査：TEI適合判定と段階導入の追補（2026-11-03）

- 状態：**調査受領・一部現物検査・設計差分案**。Schema／DSL／State Machine Role／Workbook Editor／Runtime採用の決定ではない。
- 入力：ユーザー提供 `Excel決定表_Validation_Derivation_Flow_枠組み調査.md`（S2Buri調査第3期）、`Buri_全モジュール解析.md`、`decision_tables_report.md`、`TEI_決定表テンプレート.xlsx`。この文書は調査結果を再掲するためではなく、**調査で確かめられた範囲と現行TEIとの差分を検収する**ためのもの。
- 対応：[Buri設計継承](BURI_TO_TEI_LIFECYCLE_TRANSFER_2026-11.md)、[状態機械と図表交差・三形態](STATE_MACHINE_CROSS_DOMAIN_DESIGN_PRESSURE_2026-11.md)、[Lifecycle v1](BUSINESS_LIFECYCLE_V1_DESIGN.md)。
- Buri固定参照：`seasarorg/s2buri@fbaf32dd3ec2a815dfe3f4426f10d6b0ce53547d`。Buriの古い実動環境を復元することはこの設計調査の完了条件ではない。

## 1. 採り入れるべき成果――Buriの歴史的実装とTEIの提案を分ける

添付調査は、Buriリポジトリ内のExcel利用を①`ExcelBaseParticipantProvider`／権限階層・有効期間、②ガス料金表 `rate_table.xls` の `TablePickup` 参照、③開発時 `ExcelImportCodeDataImpl` のスキーマ取込、④試験用Excelに分けた。XPDLの遷移条件・ToolのOGNL・Java計算と表ルックアップは存在するが、**Validation・Derivation・Flowを同じ正式な決定表エンジンで評価するBuri Core実装を当該調査範囲では確認していない**。この限定的な不在を「S2Coffeeを含む歴史的な全関連製品に機構が存在しない」証明へ拡張しない。

業務ユーザーが図／Excel表を直接読み書きできること、そこから既存のDTO・DAO・BAOや生成コードへ接続することは参考にする。一方、BuriのRuntime Excelをそのまま正本にする、XPDL/OGNL/Java生成器を移植する、無制限な式・スクリプト実行を認める、といった結論は出さない。

外部比較：DMNのDecision Table/DRD/Hit Policy、OpenL TabletsのExcel-first Authoring・型検査・Test・Trace、Drools/KIEの表アセット、DMN TCKの独立適合Caseは**比較材料**。汎用DMN/FEEL/Rule Engine全体の採用提案ではない。OMGの[DMN 1.5](https://www.omg.org/spec/DMN/1.5/)は正式仕様として有効だが、照合時点では[DMN 1.6](https://www.omg.org/spec/DMN/1.6)も正式版、[DMN 1.7 beta](https://www.omg.org/spec/DMN/1.7/Beta1)が進行中である。元調査で用いた1.5を「最新版」とは呼ばず、比較時に対象版を固定する。

## 2. 添付Workbookの独立した現物確認と証拠の限界

`TEI_決定表テンプレート.xlsx`を受領し、Workbookのセルと構造をプログラムで読んだ。

| 対象 | 現物で確認した値 | 結論の上限 |
| --- | --- | --- |
| シート | 12シート。Manifest/Glossary/Views/Traceability/Lifecycles/Validation/Derivation/Lookup/Flow/Tests/Changelog/Readme | 試作の構成と添付レポートの対応を確認 |
| 行数 | Validation 3、Derivation 3、Lookup 1、Flow 4、Tests 14 | 14件は**Expect記述**であり評価PASSではない |
| Manifest | `workbook_schema=tei.decision-workbook/v0`、`status=draft`、`base_revision`空、Target文字列は`semantic/sales/order_decision.tei` | 現時点で変更提案／Owner受理へ進める資格はない |
| xlsxパッケージ | `vbaProject`／`externalLinks`パーツなし、worksheetの実formulaタグなし | 添付**このファイル**についての静的パッケージ所見。全Excelを安全と保証しない |
| 添付生成レポート | lint error 0／warning 0、表・Mermaid・行列の出力を記載 | `render_decision_tables.py`・`decision_tables_diagnostics.json`が今回の個別添付に含まれず、同じlint実行の再現は**NOT RUN** |
| TEI接続 | 正式Semantic／Shared Acceptance／実Provider／Go Targetへの登録・使用を示すEvidenceなし | 設計・Workbook試作であってTEI製品でのConformanceではない |

テストベクトルは一例として`T-V-02`: quantity=0→not-satisfied、`T-D-02`: null→null、`T-F-03`: submittedからsubmit→拒否を含む。**これらをWorkbookの記述と同じロジックから自己採点すれば独立Oracleとならない**。既存[#2514](https://github.com/hat47x/tei-platform/issues/2514)の独立Expected／適用対象束縛／判別力を使う。

## 3. 互換性の衝突・過剰一般化を補正する

| 論点 | 調査・試作の表現 | 現行TEIとの照合と修正要求 |
| --- | --- | --- |
| Authoritative Source | Workbook提案では「正本は `.tei`、Excelは提案だけ」 | TEIの正本は**受理済Meaningに結びつく権威あるSource**であり、Sourceは`.tei`／Markdown／明示DSL／Native SQL等の用途別契約を持つ。Excelを一律に新正本とする必要も、全DomainのSourceを`.tei`へ強制移行する必要もない。既存[#2621](https://github.com/hat47x/tei-platform/issues/2621)／Shared AcceptanceのSource Indexを使用 |
| Validation | `positive`に`Operator`、`Operand`、`Message`、`Severity`、`When`等の列を付与、`amount_non_negative >= 0`を記述 | 現行`internal/semantic/model.go`の`Validation`は`ID/Symbol/Kind/Attribute/Relation`で、Kindは`positive`／`relation-unique`のみ。比較演算子・閾値等は**そのままCanonicalへ表現できない**。特に`>=0`を`positive`が表現すると決めない。現行2種へ意味保持変換可能かを評価し、それ以外は新意味Rule候補／非対応として停止 |
| Derivation | `multiply/add/subtract/divide/sum/lookup/coalesce`の許可演算子、Scope別一意とDAG | `internal/semantic/derivationref`は実験的Reference。現物コードでは`Multiply`と`ValidateGraph`が確認でき、汎用Lookup、丸め、演算群の正式契約はない。WorkbookのOperator全件を実装済みと報告しない。型・NULL・missing・Relation cardinality・時間、Operator pinを分離 |
| Lookup | `order_date=2026-06-01`から`Key=2026`／`rate=0.10`を得ると期待 | `date→year`のキー導出、Key Type、適用Scope、範囲重複／Gap、税率区分、版／改訂時点は未定義。T-D-03は**対応するOperatorとBindingを別定義**しない限り意味を実証しない。税率0.10は試作の合成値であり実税務要件として採用しない |
| Derived依存 | `tax_from_amount`の`Depends On`は`tax_rate_from_date`のみ | 入力`amount`は`amount_from_quantity`の出力でもある。Dependency欄にない計算依存・Field参照・Context/Provider取得・RuleFootprintを同じDAG閉包に含めないと、変更時のBuild／Consumer資格を誤る |
| Flow | `(Lifecycle, From State, Capability)`一意、必須の初期/終端、`command/none/external/reopen` | 受注Lifecycleの**簡易Profile**なら検討可能。ただしUI非同期、通信Session、ゲーム直交Region、組込FaultにはEvent payload／Guard競合・時間・Region・終端なし・Historyを扱うProfileが必要。State Machine一般契約として一律に固定しない。`Effect=none`で状態保存が無効果になるとは推論しない |
| Period / Lifecycle | `05_Lifecycles`と`30_Flow`に両者を載せる。全既存Instanceは旧定義版固定 | `Period`の実装とLifecycle ADR-0024/25の提案状態を混同しない。単純な業務status列、Period、長時間Workflow Instanceを分け、`effective_from`・既存Instanceの自動版固定を全体ルールにしない |
| Round-trip | `reference-list-roundtrip`を厳密base revisionの先行例に位置付ける | 既存`roundtrip.go`の行検査は`base_revision`が**非空のときのみ**staleを拒否し、空欄は通る。Workbookで必須化するなら**強化提案**であって実装済み機能の援用ではない。さらにID同一でも対象Source/Meaning/Row revisionとScopeを照合する |
| Shared Acceptance | Validationの`add`をPhase 2で新設予定、`set-symbol`も同じ受理経路にあるように読める | 既存[#1935](https://github.com/hat47x/tei-platform/issues/1935)では`add-validation`候補は列挙済み。欠けるのは**既存Ruleのmodify/deprecateやその他の更新系**である。`reference-list-roundtrip`の`set-symbol`候補生成とShared Acceptanceの正式な更新可能操作を区別する |
| Diagram / Table | Workbook表→Mermaidだけを生成し、図からは書き戻さない | これは安全な**Phase 0の片方向Projection**。利用者ヒアリングの図上編集・表上編集の双方から、差分Preview/Sourceに戻る能力は[#2497](https://github.com/hat47x/tei-platform/issues/2497)の**別の可逆性条件付き**計画であり、生成図を編集済みSourceとみなさない |
| Tests / lint | `0 error/0 warning`と14 Expected | Workbook schemaの局所条件は確認できても、TEI受理済Meaning・実演算・各Target・Provider／Host権限・外部Effect・実Consumerの資格までは証明しない。少なくとも独立Expectedと現行Referenceの差分を別表にする |

## 4. 提案に残す最小の設計原則

**採用候補として成熟しているのは、Workbookの固定スキーマではなく変更・証拠の責務分離。**

1. **共通の表示／編集Surface**：Validation・Derivation・Lookup・FlowとTestsを、同じAuthoritative Meaning／Typed Data／Source spanの上に表・図・Matrixとして投影する。Excelは有力なAuthoring/Review候補だが、Markdown／Studio／SHOも同列のUI/Source経路。表の数・名称・記法はProfileとDomain別に変え得る。
2. **書き戻しはProposal**：安定Element ID、Source/Meaning Revision、対象Scope、変更内容、変更根拠、Review fingerprintと差分Previewを介す。Excel内の`accepted`文字列だけではOwner受理とならない。Shared Acceptanceへ渡すまで`applied=false`。
3. **構造lintと意味診断を分ける**：欄の欠落・重複・循環・到達不能・式未対応は診断できるが、有限Event集合／全Input Domain／処理優先／権限現在性／実Effectを未観測のまま「完全」としない。警告0でも該当Claimの情報が欠ければ`unknown`。
4. **二方向の図表編集に到達可能な契約**：Table→Graph→Sourceへの選択／移動はまずRead-only。編集は1対1対応する変更だけ逆写像し、非可逆の集計・折畳み／同じedge上の複数GuardはCandidate＋レビューへ戻す。Source位置／Worksheet列・行番号はIdentityにしない。
5. **選択的再生成**：表示位置・書式、Definition/Rule式、Operator/Plugin版、Provider Binding、独立Expected、Target Generator templateを別々に差分化。Source→Meaning→Dependency→Artifact→Consumerの資格を再確認し、単一の「新版」「PASS」を作らない。
6. **実行Effectは独立**：Flow遷移候補・Validation成立・Derivation結果と、現在Grant／Provider Commit／Receipt／外部発火完了を混同しない。Business Workflow/業務状態は本体行を第一候補、複雑なUI/通信/ゲーム/組込は各Target Runtimeで管理する。

## 5. 既存計画への非重複の接続

| Owner | 本稿から渡す成熟した差分 | 実装・判断を保留するもの |
| --- | --- | --- |
| [#2621](https://github.com/hat47x/tei-platform/issues/2621) / [#2620](https://github.com/hat47x/tei-platform/issues/2620) | Excel表・Markdown表・Studioの同じRole候補とSource span／Glossary ID。Workbookシートは**Projection Profile** | 正本ファイル拡張子の固定、新Schema/Core |
| [#2497](https://github.com/hat47x/tei-platform/issues/2497) / [#1124](https://github.com/hat47x/tei-platform/issues/1124) / [#1803](https://github.com/hat47x/tei-platform/issues/1803) | 表↔Graph↔状態×Event/Capability Matrix↔SHO文書の可逆性条件とレビューUI | 巨大専用Excel Editor／自動編集適用 |
| [#1935](https://github.com/hat47x/tei-platform/issues/1935) / [#267](https://github.com/hat47x/tei-platform/issues/267) | 既存`add-validation`と未対応modify/deprecateを区別し、base revisionとreview fingerprintの再使用 | 新たな全種類Patch DSL／第二Acceptance |
| [#988](https://github.com/hat47x/tei-platform/issues/988) / [#2622](https://github.com/hat47x/tei-platform/issues/2622) / [#1129](https://github.com/hat47x/tei-platform/issues/1129) | Validation/Derivationの型付きOperator適用・NULL／missing/Relation cardinality、Rule phase／Effect fence | 任意Excel式の無検査実行、汎用Rule Engine |
| [#2619](https://github.com/hat47x/tei-platform/issues/2619) / [#1121](https://github.com/hat47x/tei-platform/issues/1121) | 三形態の共通Source→Role→Accepted Meaning→Target-native生成→SHO文書の比較 | 一律のFlow Runtime／Go/Java/組込の同一実行保証 |
| [#2514](https://github.com/hat47x/tei-platform/issues/2514) / [#1131](https://github.com/hat47x/tei-platform/issues/1131) | 14ベクトルを*入力候補*とし、現行Referenceで支持できる列と未実装列、独立Oracleへ分解 | 元WorkbookのExpectを自動で独立真理へ昇格させる |

### 5.1 Flow表がDerived Attributeの変化を契機とする場合

[仮想項目・Data Script・イベント起動・Batch Pipelineの設計案](DATA_DERIVED_TRANSITION_TRIGGER_PIPELINE_2026-11.md)に接続する。添付Workbookの`(Lifecycle, From, Capability)`形式だけでは、**子明細の更新で親レコードの仮想statusが変わった際にJobを起動する**という要求を表現し切れない。

Data Modelに`virtual/read-only`と宣言したAttributeを、Data Scriptで型付き依存付きに導出し、さらに**そのData Script上の変更時Trigger宣言をFlow／Capabilityへ参照Bindingする**構成を評価する。ただし仮想項目の再計算＝永続Event発生ではない。Flow表で`physical-write`／`derived-edge`／`level`／`domain-commit`／`timer`を区別し、Scope・Snapshot・Clock、未知・重複・再送・Rollbackの扱いを検査する。正式なWorkbook列や構文は未採用。

- サンプルの受注`billing_ready`が`false→true`へ変わる場合、子明細の全件取得／請求不存在のread-setと前後評価の整合がないならEdgeを確定しない。
- 大規模Jobは#1130の非UI Batch／Pipeline、更新と副作用は#1129/#1174のPhase/Effect Fence、Dependencyと鮮度は#2582/#1708がOwner。WorkbookのFlow列に`command`と書いただけで起動権限・一度限りの実行を保証しない。
- Flow図、Decision表、Data Modelの仮想項目、Data Script、SHO成果物を同じIDで投影するが、JobのJournal/Receiptという**独立した実行事実**は元データの導出状態へ消去しない。

## 6. 最小受入順序と停止条件

- **D0 調査還元（今回）**：一次観察／3報告とWorkbook現物／現行TEIの適合・不適合を分離。Buri実動、TEI Runtime、第三者Rule Engineの実行は不要。**本稿は設計レビュー入力であり、正式Schema受入ではない。**
- **D1 Read-onlyの同一性**：現行の受理済みSourceからValidation/Derivation/Flowの**実在する部分集合だけ**をTyped ProjectionとしてCSV/Markdown/Excelへ出力し、Stable ID／Source／Version／Coverageを照合。Unsupportedは埋めない。
- **D2 一本の提案往復**：ID不変・行の並べ替え・一行変更・重複ID・空/古いrevision・Scope違い・同時編集・見えない列を対照とし、Preview/Impactだけで止める。実Acceptは#1935の対応可能操作に限定し、Ownerの判断を経る。
- **D3 三形態の意味圧力**：`positive`厳密意味と非負値、`Multiply`と`Lookup`の差、`date→year`、型・null/missing、全入力依存、`state×capability`の空欄、Event/Region/時間を独立Expectedで比較。必要なRule/Role/Operator追加はOwner判断へ戻す。
- **D4 図表の交差編集**：Validation表、Derivation DAG、Flow図/Matrix、Tests、SHO説明を同一Meaning IDで相互選択。逆写像は可逆な差分のみに限定し、不確実な編集を自動適用しない。
- **D5 Target／実Effect（別資格）**：既存Lowering／Go・外部Host向け生成を適合したDomainだけで評価。生成物の実消費、Provider/Runtime、現在Auth、Commit後Receiptは独立Gate。Buri旧Runtimeの動作確認は不要。

**停止**：Managed Sourceが未解決／Role・Type・Operator未受理／Workbook base revision未固定／図表が対象Scopeを消失／独立Expectedなし／Providerのpartialをcompleteへ誤昇格／Lookup・時間・業務Keyが未定義／現在Grantを過去候補へ置換／Commit unknownを成功扱い、のいずれかに当たれば当該変換・実行を止め、何を理解できたかだけread-onlyで表示する。

## 7. 今回の検証状態

- **添付Workbook現物確認**：12シート・各規則件数・Manifest・4 Flow遷移・14ベクトルを読み取り、パッケージ内にVBA/External Link/Formulaタグがないことを静的確認。**書き換えなし**。
- **TEI現行ソース**：`internal/semantic/model.go`のValidation型、`internal/semantic/derivationref/contract.go`のMultiply/ValidateGraph、`tools/reference-list-roundtrip/roundtrip.go`のbase revisionチェックとProposal、`tools/reference-shared-acceptance/README.md`の受入境界、#1935／#2621／#2497／#2514等の現行Issueを読取。
- **未確認**：添付元Pythonのlint再実行・Diagnostics JSON照合、TEI Semantic Loaderへの取込、テストベクトルの実行、実Excel UI描画・共同編集、外部Decision Engine実動、Provider/Host/Target/SHO実消費、実Owner受理。すべて**NOT RUN／NOT ACCEPTED**。
- **計画の影響**：設計資料だけをDraft PRで更新し、main／既存ADR／Owner決定／Core／Goコードには変更を加えない。

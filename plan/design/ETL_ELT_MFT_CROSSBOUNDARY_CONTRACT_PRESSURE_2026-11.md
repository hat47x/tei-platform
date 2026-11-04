# ETL/ELT・MFT/ファイルリレー横断契約の圧力試験（設計候補・2026-11-04）

- **状態**: 外部一次資料・既存TEI Issueに基づく **設計・Research / NOT RUN / 未採用**。新Core型、正式Plugin API、Host、Scheduler、業務受理、Target実行保証を採用した文書ではない。
- **調査対象**: ODIP（モデル駆動処理・業務基準日・リカバリ）、Apache Hop（Pipeline/Workflow）、SeaTunnel（Logical/Physical Plan）、HULFT-HUB（蓄積・追越禁止）、AS2（受領証拠）、Apache Camel（ファイル公開/重複取込み）、OpenLineage（Job/Run/Dataset）。
- **照合したmain**: `dd773c1ad8735a8c6aeb67ef51f8437ac3320e17`（GitHub branch APIで2026-11-04に取得）。
- **文書の責務**: 「既存Issue/契約に不足する意味の区別」と「反証可能なHeadless受入圧力」を記録する。要件正本は `requirements/TEI_PRODUCT_REQUIREMENTS.md`、採用判断はADR、実装資格は各Owner。新しい第二のPipeline/Scenario/Authority/Resolver/Lock正本は作らない。

## 1. 問題と全体判断

多くのローコード画面系はデータ加工・大量処理・企業間ファイル転送を別ツールで補完する。TEIでは、標準データ構造と型付き関係を共通基盤として **同じ受理済み意味から、UI/APIと独立した非画面Batch・ETL/ELT・Transfer Adapterなど異なる実装成果物を構成する** ことを検討する。

ただし「ひとつのパイプライン図」はひとつの意味論ではない。少なくとも次の責務を区別する。

| 観点 | 対象・契約 | 成功の意味と限界 |
| --- | --- | --- |
| Data Transform | Dataset, schema, grain, completeness, relation, operator, input/output | 値の導出。NULL/欠損/重複/順序/精度/粒度・時点を保持 |
| Workflow / Trigger | Step dependency, schedule, calendar, business period, retry | 起動・分岐・依存。Dataflowの矢印を実行順序と同一視しない |
| Materialization / Publication | Artifact, snapshot, storage, visibility, commit protocol | 読取可能な成果物への公開。アップロード終了≠公開・可視性保証 |
| Transfer / Relay | Sender, receiver, route/hop, attempt, spool, receipt | Byte/Artifactの配送。転送完了≠後続取込み・業務受理 |
| Ingestion / Reconciliation | FileSpec, import unit, event identity, watermark, reject | データ・事象の照合。重複配送≠重複業務効果 |
| Business Effect / Acceptance | Operation intent, authority, provider commit, independent read, owner acceptance | 実際の更新・受理。Job成功/MDN/ACKからは推論しない |
| Build / Deployment / Observation | target-neutral plan, target-bound binding, artifact set, run/attempt evidence | 実行対象の生成と証拠。生成成功≠実配備≠現在権限 |

**非目標:** ODIP/HULFTを複製すること、すべてを必ずJob化すること、TEIが常駐サーバやJob Scheduler/Resource Authorityになること、Plugin実装やAgentの自己申告のみで業務効果を確定すること。

## 2. 外部製品から抽出する相補的な設計知見

- **ODIP**：入力・加工・出力仕様から中間データとジョブネットを導き、処理基準日・業務カレンダー・期・前回結果を利用するサイクリック処理・リカバリを扱う。**業務基準期間 ≠ 物理実行時刻**。ベンダーが述べる自動化や品質の程度は独立検証済みでない。実行はトランスフォーマ/リポジトリ方式であり、同じ仕様から任意の独立バイナリが生成できる事実ではない。
- **Apache Hop**：Dataflow Pipeline（Transformが並行に動く）とWorkflow（Action実行・完了・分岐）が異なる意味を持つ。GUIの図の見た目と実行トポロジーは同一ではない。
- **SeaTunnel**：Source/Sink/Transform API、エンジン別Translation、LogicalDagとPhysicalPlanの分離。TEIが対応するなら論理契約→Provider能力/Targetに依存する物理計画を可視化し、unsupportedを隠さない。
- **HULFT-HUB**：蓄積（Store-and-Forward）で送信側/受信側の稼働差を吸収。追越禁止は基本的に同一ファイルID内の順序制約で、**多段中継の全ホップ設定が条件**。順序確保は並列性に影響し、蓄積異常・蓄積開始前失敗等の例外を考慮する。これをグローバルな業務イベント順序に拡張しない。
- **AS2 RFC 4130**：MDNと署名付きReceipt/MIC照合は受領証拠に使えるが、受信業務データの意味・Ownerの受理とは異なる。署名なしと署名済み・同期と非同期・異常を区別する。
- **Camel File**：書込み中ファイルの誤取込み対策としてreadLock/done marker/公開先へのmove、冪等ConsumerのKey/Storeが重要。完了マーカーやrenameの強度はファイルシステム/ストレージ/Host依存。
- **OpenLineage**：Job/Run/Datasetとfacetを分ける。TEI Source、Accepted Meaning、Plan、Build Artifact、Transfer Attempt、Provider Effect、Independent Readの来歴を同一の成功フラグに畳まない。

## 3. 標準データ構造への最小対応付け（名前と配置は未採用）

Coreは `Type / Instance / Port / Binding / Graph / Conformance` などの普遍的構造に留め、ドメイン語彙は `tei.std.*` またはドメインPluginにより**型付き役割/facet/関係**として組み立てる。Principal/Account/Business Entity、特定DB・特定Host・固定の実行エンジンを最小バッチに必須化しない。

| 意味上の役割（仮称） | 既存責務に接続する点 | 必須の区別 |
| --- | --- | --- |
| `DatasetRole` / `TransformRole` | #2622 Data Operations / Derived Dataset | Row/Bag/Sequence, grain, completeness, input scope, data time |
| `FlowStepRole` / `DependencyRole` | #1130 Flow、#2744 Buri/TEI/EKI案 | data dependency / temporal order / control / causal edge |
| `TransferContractRole` / `HopRole` | #2542 ファイル・メッセージPlugin面 | endpoint/route/能力・再送・順序の範囲・期限・容量 |
| `ArtifactRole` / `PublicationRole` | #943/#1102 ArtifactSet、#1807 import | logical dataset vs materialized byte object vs published version |
| `ReceiptRole` / `AttemptRole` | #1174 Effect/Receipt、#2542 import unit | transferred/verified/published/ingested/acceptedの独立観測 |
| `PeriodRole` / `RecoveryRole` | #1130、#1129、#2622 | business effective period vs physical run/observedAt、checkpoint vs effect |
| `LineageRole` | #1121 Lowering、#2619 Everything is Data | plan/build/run/attempt/independent observation各版の参照 |

この表は正式な標準型の提案でもない。既存Type/Relation/Plugin/ModelRefで表現できるなら新しいCore種別は不要。固定 `Pipeline` 型・新DSL・第二のArtifact Storeを先行させない。

## 4. 独立して保持すべき契約と反例

### 4.1 同一性の軸

- Semantic Dataset ID と Source snapshot/revision
- DataArtifact version と物理ファイルのbyte digest（algorithm/encoding/length）
- ファイル名/パスと配置先のversion/generation（同名上書き・ABA）
- Transfer contract と配送要求、個々のAttempt（retry/relay hopを区別）
- Import Unit とBusiness Event identity、Provider Effect id/receipt

内容ハッシュで同じbyteでも、別の取引・別宛先・別の期・別送達要求が同一とは限らない。逆に輸送時に圧縮・暗号化・改行変換・文字コード正規化されれば、論理内容が等価でもbyte digestは異なる。何に対するDigestか、変換前後のBoundaryを明示する。ハッシュの桁数・alphabet・algorithm選択は既存ID研究へ接続し、具体的な不可逆変換や衝突安全性を推定しない。

### 4.2 保証を直積として表現し、強い保証へ自動昇格しない

| 独立facet | 例 | 禁止する推論 |
| --- | --- | --- |
| Completeness | complete / partial / unknown | 0件受信⇒0件存在 |
| Order | per-file-id/per-partition/per-destination/unknown | 1ホップの順序⇒多段/全業務順序 |
| Delivery | at-most-once / at-least-once / uncertain | 通信ACK⇒永続受領 |
| Integrity | transferred bytes verified / signed receipt / unknown | byte一致⇒正当な業務内容 |
| Publication | temp/spooled / committed-visible / unknown | Put成功⇒Consumerが完成版を見る |
| Replay | restartable from offset / application replay / unsupported | byte resume⇒Effectを安全に再実行 |
| Business effect | committed / rejected / unknown | Job成功⇒Provider Commit Known |
| Authorization | contract declared / runtime grant observed / expired | PluginのCapability宣言⇒現在実行可能 |

`exactly-once` は多義的であるため、何の境界（配信試行、Import Unit、業務Effect、受領済み事象）に対する一回性かを必要条件として固定する。一般の分散ファイル配送＋外部DB更新で普遍的な原子Commitを主張しない。unknown effectでは照合/Owner回復へ停止。

### 4.3 時点と物理資源

Event time、business/effective period、batch reference date、processing time、transfer/spool time、publish time、observedAt、receipt time は別値。基準日変更は同じデータ変換ジョブの意味を変え得る。Spool budget、storage lifetime、file size、bandwidth/egress、backpressure、CPU/Memory、deadlineとrecovery slackは別単位で示す。実Nodeの資源Admission/Lease/RecoveryはEKI Owner側、TEIは制約・必要な能力と確認結果を受ける（#1130/#2744）。単一ローカルGo executable、OS Scheduler、外部MFT、EKIを選択可能な実行/連携先とし、どれも必須Hostにはしない。

### 4.4 Publish/Receive境界

Writerが最終ファイル名へ直接書くことを既定安全とみなさず、暫定配置・検証・公開条件を明示する。renameの原子性、オブジェクトストレージの世代条件、completion marker、SFTP remote path、後続Consumerのpoll方式の差をProvider capabilityとして検証する。転送/配信側の「完了」、受信側の「検証」、Importerの「受理」、業務Ownerの「確定」は別のReceipt/Observationとして関連付ける。取込み元削除・永久保留レコードの更正には別Authority/Retention/監査契約が必要。

## 5. Headless-firstの最小縦切り（新たなScenario SSOTを作らない）

**入力例**: 売上CSV（処理基準日D）＋店舗マスタから集計Datasetを生成し、CSVとSQLiteのELT/独立Worker-ETLへ実現、SFTP-compatible/in-processの合成Relayを通して取込み・照合する。既存#2145の `FileSpec` / `ImportFile` と #1130のBatch、#1180のScenarioを再利用し、未存在の実MFT Adapterはあるものと扱わない。

以下は**新規の受入Pressure提案・全件 NOT RUN**。独立Expected/実行Script/TargetとOwnerは、既存#1131/#1180/#2514に集約し、本文は単なる比較票とする。

| ID | 入力・反例 | 期待する最小の判定 |
| --- | --- | --- |
| DM-01 | 同じ受理済み意味をSQLite ELTとWorker ETLへ | null/bag/順序/decimal/grainを比較し、異なる場合は不適合。両実装のBuild digestとSourceを別照合 |
| DM-02 | 処理実行日D+1、業務基準日D | 実行日が変わっても選択した業務期・依存するデータ版を誤変更しない |
| DM-03 | 再実行、前日結果からのサイクリック再計算 | 入力版とチェックポイントを識別。前回出力を現在正本として黙って再利用しない |
| DM-04 | 最終名へ書込み中に受信側がpoll | 不完全ファイルをPublished/Accepted扱いしない。Providerが公開原子性を証明できなければunsupported |
| DM-05 | Relay 2ホップ、片方のみ順序制御 | 全路線の順序保証を拒否/unknown。per-file-id順序をglobal orderへ昇格しない |
| DM-06 | spool full / expiry / crash-restart | hold/failed/unknownと実送出証拠を分離。勝手にBusiness Lost/Acceptedとしない |
| DM-07 | timeout後に二重配送、Import再送 | Transfer Attemptは2件、Import Unit/Business Eventはidempotency規則に従って1回のみ確定 |
| DM-08 | 配送成功後、Importerの検証失敗 | Delivered/VerifiedとIngested/BusinessAcceptedの別ステータスを維持 |
| DM-09 | 同一名・異byte、同一byte・別期間 | path/hash/period/transport idを混同せず、再取り込み・上書きの可否を型付き診断 |
| DM-10 | 送信/中継で圧縮・暗号化・文字コード変換 | Digest検証の対象byte・normalization/semantic equivalence・lossを分離 |
| DM-11 | 外部DB commit応答喪失、Retry要求 | unknown effectは自動再実行せず、Receipt/独立Readへ引き渡す |
| DM-12 | 同一FlowをCLI/OS timer/外部MFT/EKIに接続 | 意味・Scenarioは同じ。失われたCapability（順序、保存、Scope、Grant）はunsupported |
| DM-13 | 図のfilter/collapse・不透明な外部MFT・動的Provider | TEI管理範囲外を「図に無いので存在しない」と断定しない（#1124/Topology文書） |
| DM-14 | 転送先/外部Provider権限失効・秘密Field | Capability/Planは実Grantに昇格しない。確認不足時の送出・開示を保留 |

**初回の実装可能な縦切り**: DM-04, DM-07, DM-08, DM-09 を既存限定Referenceへ追加できるか静的確認 → 実File/SQLite独立Read → DM-01/02 の二生成戦略 → 多段転送DM-05/06。その後に実SFTP/HULFT相当への接続候補を選定する。性能、スケール、実取引先、実Windows/HULFT、E2Eを達成済みとしない。

## 6. 既存Ownerとの境界と反映順

| Owner | 今回の追補との関係 |
| --- | --- |
| [#2619](https://github.com/hat47x/tei-platform/issues/2619), [#2622](https://github.com/hat47x/tei-platform/issues/2622) | 標準データ構造とData Operations/Derived Dataset、用途相対の意味同等性 |
| [#1130](https://github.com/hat47x/tei-platform/issues/1130), [Draft #2788](https://github.com/hat47x/tei-platform/pull/2788) | UI非依存Batch、Invocation/Build Unit/ArtifactSet、非常駐実行。ここで新Scheduler/Job Engineを作らない |
| [#2542](https://github.com/hat47x/tei-platform/issues/2542), [closed #2145](https://github.com/hat47x/tei-platform/issues/2145), [#1807](https://github.com/hat47x/tei-platform/issues/1807) | FileSpec、Import Unit、同一事象照合の既存成果。MFT/relayのプラグイン面は**Owner未決** |
| [#1121](https://github.com/hat47x/tei-platform/issues/1121), [#943](https://github.com/hat47x/tei-platform/issues/943), [#1102](https://github.com/hat47x/tei-platform/issues/1102) | Target-neutral plan→実Target/physical Artifact/Build provenance。第二Build Lock不要 |
| [#2744](https://github.com/hat47x/tei-platform/pull/2744), [#1174](https://github.com/hat47x/tei-platform/issues/1174), [#1129](https://github.com/hat47x/tei-platform/issues/1129) | TEI Flow meaning／EKI Task resource／実Provider EffectのOwnerを分ける。新Authority不要 |
| [#2495](https://github.com/hat47x/tei-platform/issues/2495), [#2501](https://github.com/hat47x/tei-platform/issues/2501), [Topology設計](TOPOLOGY_COMPOSITE_VIEWS_AND_PIPELINE_2026-10.md) | Bag/Sequence/Stream/GraphとOrder、opaqueと観測範囲のPressure。新Core型の確定に転用しない |
| [#1180](https://github.com/hat47x/tei-platform/issues/1180), [#1131](https://github.com/hat47x/tei-platform/issues/1131), [#2514](https://github.com/hat47x/tei-platform/issues/2514) | 受入・試験Expectedの正本と独立Oracle。別のテストの正本を作らない |

段階: (P0) 現行Referenceと契約の静的Fit/Gap → (P1) Headless合成 negative fixtures → (P2) 実SQLite/ファイル系別processでSource/実Artifact/Independent Read → (P3) 別Transport/ProviderでCapabilitiesと未知を比較 → (P4) 実MFT/EDI連携・復旧/運用を対象Ownerの権限で実証。各段階で試験済み範囲・未実行・意思決定を独立記録する。

## 7. 一次資料（参照先と検証範囲）

- ODIP 機能: https://www.imkk.jp/product/function
- ODIP コンポーネント/実行まで: https://www.imkk.jp/product/component
- ODIP Tutorial・メタデータ: https://www.imkk.jp/product/tutorial
- Apache Hop pipeline/workflow: https://hop.apache.org/manual/latest/pipeline/pipelines.html / https://hop.apache.org/manual/latest/workflow/workflows.html
- SeaTunnel Architecture: https://seatunnel.apache.org/docs/architecture/overview/
- SeaTunnel Engine Logical/Physical: https://seatunnel.apache.org/docs/architecture/engine/engine-architecture/
- HULFT-HUB蓄積: https://www.hulft.com/help/ja-jp/HUB-V3/COM-HUB/Content/HUB_HUB/Fun_HUB/Acc.htm
- HULFT-HUB追越禁止: https://www.hulft.com/help/ja-jp/HUB-V3/COM-HUB/Content/HUB_HUB/Fun_HUB/Phbt_OVTK.htm
- AS2 RFC 4130: https://www.rfc-editor.org/rfc/rfc4130.html
- Apache Camel File: https://camel.apache.org/components/next/file-component.html
- OpenLineage facets: https://openlineage.io/docs/spec/facets/

**証拠制約:** 公開マニュアルの製品動作はその文書の版・条件に限定。ODIP/HULFT/AS2/Camel/SeaTunnel等の実機実行、コード監査、性能・セキュリティ評価、TEI生成Targetとの接続、実企業取引先、正式Owner受理、全Repository Healthは本書では **NOT RUN**。現在のmain/関連PRが変更されたら再照合する。

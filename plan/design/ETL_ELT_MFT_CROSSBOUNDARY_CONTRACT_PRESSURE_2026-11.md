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


## 8. 2026-11-04 再照合：公開・配送・受理の「保証の合成」と停止線（設計追加／NOT RUN）

本節は既存[#2604](https://github.com/hat47x/tei-platform/issues/2604)の複数ファイル公開、[#1130](https://github.com/hat47x/tei-platform/issues/1130)のJob入出力、[#2542](https://github.com/hat47x/tei-platform/issues/2542)のファイル取込み/連携を**別Ownerのまま**つなぐ。新しい成果物Publisher/Runner、第二の受理State Machine、全Provider横断Txは作らない。HULFT-HUBの配信順・S3通知を、業務イベント全順序や取引の完全性に昇格させない。

### 8.1 Five boundaries: 同じ「成功」を使わない

| 証拠境界 | 証明可能な主張（要独立の証拠） | そこからは**導けない**こと／Owner |
| --- | --- | --- |
| **B0 Semantic / Plan** | Accepted Source/意味/Function/Period、型・入力・出力・必要な能力、Target-neutral plan | 未実行の生成計画から実Byteや実権限。#2619/#2622/#1121 |
| **B1 Materialization / Publication** | 物理MemberのByte、Digest algorithm、版、Manifestの一致、Consumerの実Readerが同一世代のMemberを取得可能 | Manifest Complete≠一式atomic公開≠Business Accepted。#2604/#943/#1102 |
| **B2 Transfer / Relay** | Route/Hop、Contractと各Attempt、Spoolの保存/期限、Byte整合性、配送/受領Receiptの範囲 | 配送Ack≠永続公開≠Importer確認。HULFT相当実Adapterは未実装。#2542 |
| **B3 Ingestion / Reconciliation** | Import unit/file spec/検証結果、変換失敗/保留、Event identity、重複適用の結果、Consumer側確認 | 同一File digest≠同一Business Event；Record ingest≠業務意味受理。#2145/#1807/#2542 |
| **B4 Provider Effect / Owner** | 現在Authority、実Request/Commit Receipt、別Read、許可されるBusiness Result、Ownerの意味受理 | Worker exit=0、EKI Task completed、MDN、S3通知だけでCommitを推論しない。#1174/#1129/#1119 |

**契約の合成規則（候補）**:
1. 必要な保証をConsumer/Operation/Period/宛先/データ分類で宣言する。各Providerは対象操作・版・実環境・Scopeに限定した能力を返す。意図した保証より弱い場合は \`unsupported\`、未照合なら \`unverified\` とし、勝手に「より弱い成功」へ代替しない。
2. \`Transferred\`、\`IntegrityVerified\`、\`Published\`、\`Ingested\`、\`BusinessAccepted\`を一列の単一enumの状態機械に強制しない。各境界に独立したObservation・unknown/partial・時点・Authorityを持たせ、順不同の到着を許す。全段階を必須化せず、不要な段階は \`not-applicable\` と \`unverified\` を区別。
3. 互いの証拠は\`subject qualified identity + version + attempt + scope + provider + time\`で参照関係を張る。路線・内容・業務のIDを単一キーへ自動統合せず、正確に解決できない場合はambiguous/unknown。既存Hash ID方式（#2814/ Draft #2815）はScheme/用途別の候補であり、Artifact Digestや正本IDを置き換えない。
4. \`unknown\`（ACK失踪・途中失敗・独立Reader不可）時は同じ副作用Jobを自動再送しない。必要に応じHostの現在GrantとEffect/Receipt再照合を要求。純粋な計算の再試行能力を業務Commitへ転用しない。

### 8.2 Publicationに必要なCapabilityは配送能力と別

| Provider/方式（例） | 明示できる可能性のある保証 | 未検証のまま推定しない保証 |
| --- | --- | --- |
| ローカルFile stage→rename | 同一FS/OSでの個別ファイル公開候補。Consumerが期待版のbyteを再読取 | Windows/POSIX/ネットワークFSすべての一式原子公開、永続化までの保証 |
| S3の単一Object PUT | **単一Key**でのatomicな旧版/新版観測、read-after-write、条件付きPutによる上書き競合防止 | 複数Keyの一式公開、通知順序、業務取込み、ETag=内容Digest |
| Immutable generation prefix + manifest pointer | 成果物を別世代のKeyへ保存し、公開Pointerと各Memberの版を照合する実装候補 | Pointerだけで未到達Member/実権限/Readerが読んだByteの保証 |
| HULFT-HUB Store-and-Forward | 蓄積、任意送出、経由Serverの条件付き順序制御 | 受信側Ingestion/業務受理、全イベント欠落ゼロ、全路線に無条件で順序保証 |

**一次資料で確認した限定的な違い**:
- S3は単一Keyの更新に原子性・強いread-after-write整合性を提供し、\`If-None-Match\` / \`If-Match\`で単一Key更新を制約できる。**ETagは条件付き更新のTokenであって一般のContent Digestではない**。特にMultipartや一部の暗号化方式ではETagは全Object MD5にならない。独立内容Digestは別のアルゴリズム・検証対象Byteを指定する。
- S3 Event Notificationsは**at-least-once**であり、重複と配達順序逆転があり得る。\`sequencer\`は同一Object Keyでのイベント比較に使えるが、異なるKeyを跨ぐ全順序や取引の完全性には使えない。Event通知成功をPublication/Reader/Importer受理へ転用しない。
- HULFT-HUB追越禁止は**同一File IDの蓄積開始順**に関する条件付き送出制御。多段の場合は全HUBでの設定が必要。ただし先行データが蓄積異常で削除された、または蓄積開始前にエラーになった場合、後続は待機せず送出され得る。従って\`ordered\`と\`gap-free\`は直交する。追越禁止で並列度が下がる条件も、Jobのdeadline/bandwidth/cost評価へ返す。
- AS2 signed MDN/MICは元メッセージと受領証跡の関連付けを支えるが、後続ERP取込み・事業者の業務承認を証明しない。無署名MDN、非同期MDN、保持された元メッセージとの照合有無を区別。
- Camel File Consumerの既定冪等Keyが物理Pathの場合、同名別版ファイルを誤って二度目と判定し得る。TEIは物理ファイル名をImporterの唯一のBusiness Eventキーにしない。

この違いから、**Provider capabilityは単なる \`transferSupported=true\` でなく、\`guarantee + subject + ordering key + scope + version + required evidence\` の候補表**で表す。現行TEIにこれらの正式APIがあるとの主張ではない。Capability不一致時の実行前の拒否やTarget fallbackは既存#1121/#951のOwnerへ委譲する。

## 9. 実行経路における「一式性・順序・完全性」の直交チェック

同じ報告用CSV一件だけでは、**物理一式公開**と**ビジネス入力の完全性**を区別できない。

例: 11月3日締めの売上結果 \`D\` がCSV \`A\`・補助マスタ \`B\`・Manifest \`M\` からなる。新世代 \`R2\` の \`A2\` を公開した時点で旧世代 \`R1\` の \`B1\` と \`M1\` をReaderが見た場合、一式公開は成立しない。全Byteがそろっても、受注の連番 \`1,2,4\`（欠落3）なら業務期の入力完全性は成立しない。さらに配送先が\`R2\`のCSVを受信しても、受理対象日の意味や改訂済みMasterとの関連付けは別。

**一式性の確認**は[#2604](https://github.com/hat47x/tei-platform/issues/2604) AP-01〜07（と[PR #2817](https://github.com/hat47x/tei-platform/pull/2817)の限定Characterization）を再利用し、新しいPublisher/同名Fixtureを作らない。**Business Datasetの期間/粒度/期待する全件性**は#2622、**Transfer/Relayを跨ぐ列挙・欠落**だけを本ノートの固有Pressureとする。

- **順序（order）**：どのPartition/File ID/宛先/路線で、どの操作の先行制約を保証するか。全ホップが必要保証を満たすかを確認。
- **完全性（coverage / gap-free）**：そのScopeで期待される全件数・連番範囲・締め時刻・Manifest member集合・Import件数やChecksum等の独立の根拠があるか。先行Spool失効・欠落を「順序通り」によって隠さない。
- **一式公開（atomic visibility）**：同じ世代のMember集合をConsumerが再読取し、任意の一つのMemberのみ新世代を参照する状態を拒否するか。S3単一Keyの原子性と複数Key一式性は異なる。
- **鮮度（freshness）**：Source/Meaning、Job Period、Provider snapshot、物理Object version、Receipt/Grant時点が現在の要求条件を満たすか。0→1→0のABAや同名上書きを同値としない。
- **効果（effect）**：取込み後の検証成功、外部DB Commit、OwnerのBusiness Acceptanceの別Receiptを持つか。

## 10. 実証計画を追加する場合の既存Scenarioとの接続（設計／全件NOT RUN）

新たなDMケース番号や第二Scenario正本を設けず、前節DM-01〜14を既存Ownerのテストへ投影する**比較観測列**を追加する。

| 既存Pressureへ重ねる反例 | 必要な独立Observation | 必ず保持する拒否/unknown |
| --- | --- | --- |
| DM-04 + #2604 AP-01/06 | Published manifest revision、Member actual bytes、Readerの実版、切替時刻 | R1/R2混在、旧Manifestを読んだ後のR2上書きをSuccessとしない |
| DM-05 + HULFT-HUB欠落先行 | Hop設定と各蓄積Attempt・失効、同一File ID順序、期待する配信集合 | 先行データ欠落にもかかわらず\`ordered and complete\`と称しない |
| DM-07/09 + S3 Event通知 | Key/version、通知Event id/sequencer、受信Byte Digest、Import Unit identity | 重複通知/逆順、ETagをSHA-256と解釈、同名別版を一つのBusiness Eventと誤認しない |
| DM-08 + AS2 signed MDN | 送信元Message ID、署名・MIC/元Message照合、Importer実結果、Provider側別Read | MDN verifiedだけでBusiness Acceptedを主張しない |
| DM-02/03 + Period再処理 | 業務基準期、Source/参照Master版、前回Artifact/Checkpointと本日Attempt | 旧期・旧Master混在、同じJob名での意味的ABAを合格としない |
| DM-11 + 外部Effect不明 | OperationID、Host Grant、Provider Commit Receiptまたは独立Readのunknown | EKI Task Result / exit0 でCommit確認済みへ昇格しない |

### 同一試行の最小Witness（**正式Wire Schemaではない**）

- \`acceptedMeaningRef + revision + selectedPeriod + functionOrOperationRef\`
- \`target/build artifact actual digest + runtime provider/capability evidence\`
- \`logical dataset + resolved input snapshot + output manifest/member byte digests\`
- \`publication generation / actual reader version / read byte result\`
- \`route + hop + transfer contract + attempt + ordered key + receipt scope/time\`
- \`import unit + business event identity + validation/reconciliation result\`
- \`effect/commit receipt or unknown + separate provider read + owner business acceptance (if applicable)\`

すべてを常に生成物に埋め込む要求ではない。各欄の**Owner・出典・時点・開示可否・未観測**を保持し、利用者に公開不能な秘密・他TenantのデータをGraph/LLM/ログへ無条件出力しない。SHO/Studio/CUIはこの比較票のread-only Projection候補であり、見えるから実Authorityがあるとの判断はしない。

**実証順序（対象Ownerへ引渡す計画）**:
1. #2145の実PostgreSQL 16限定取込みReference、#2604/#2817のArtifact混合世代反例、#2666の実CLI→Artifact byte Reader（限定CI）を**再利用可能か静的点検**。歴史的PASSの証拠資格と現在Source版は分ける。
2. Headlessで単一ローカルBuild/ArtifactSetの\`R1→R2\` Publisher/Consumerを構成し、途中失敗/Member欠落/旧世代読取を固定。**#2604 Ownerの既存AP試験を主張元とする**。
3. 別のDisposable File/SQLite Consumerで、同一実行AttemptのPeriod・Manifest・Route・Import・独立Readを結ぶ。SFTP/HULFT/EKI/クラウドに依存しない最小の正常/負例を得る。
4. S3やMFT等の**第二Provider**で、実能力の差分（conditional write、multipart checksum、Event再送、同一File ID順序、Spool expiry）を表示。保証できないものをunsupported/unknownに残す。
5. 実HULFT製品や企業間EDI/本番Hostへは許諾・実環境・Owner判断後に進む。製品仕様比較のみで商用運用適合、実同期/配信保証を宣言しない。

### 複数レーンの2026-11-04現状との分担

- #1130 の更新にある **Meaning/Invocation、Build membership、Entry/Adapter、Effect/Authority、Result/Consumerの五Claim**は既存受入を尊重。本ノートB0〜B4は**物理データ配送に特化した分解**であり、第二のGate体系ではない。
- Draft #2788のFunction/Build Unit選択はUI+Jobの物理構成に関する未採用案。本ノートは配信契約を新たなFunction型として確定しない。最初は単一Build/非常駐Goでもよい。
- #2604は設計ArtifactSet一式公開がOwnerであり、本ノートは**その公開結果を別Transport/別Consumerへ引き渡すときの保証の合成**のみを扱う。
- Draft #2744とEKIはResource/Task/Attempt/Lease・Job委任のOwnerを区別。現EKI P0の\`parameters\`/Effect/Artifact bodyの制約を型付き拒否へ接続する際は#1130のBD-01〜07を再利用。実TEI↔EKI Job I/Oは未実証。
- #2814/Draft #2815のHashIdScheme/Wireの候補は、ProviderのETagや移送Checksumと**目的・バイト対象が違う**。既存Artifact Digest/署名/Canonical IDを変更しない。
- #1180の保存Scenarioと独立Expected、#2514のOracleを正本とする。上記観測票はFixture/Scenarioの第二正本にはしない。

### 追加根拠（一次資料、対象版に限定）

- HULFT-HUB追越禁止と例外: https://www.hulft.com/help/ja-jp/HUB-V3/COM-HUB/Content/HUB_HUB/Fun_HUB/Phbt_OVTK.htm
- HULFT-HUB蓄積・送出: https://www.hulft.com/help/ja-jp/HUB-V3/COM-HUB/Content/HUB_HUB/Fun_HUB/Acc.htm
- S3 単一Keyの整合性: https://docs.aws.amazon.com/AmazonS3/latest/userguide/Welcome.html
- S3 conditional writes: https://docs.aws.amazon.com/AmazonS3/latest/userguide/conditional-writes.html
- S3 ETag / multipart integrity: https://docs.aws.amazon.com/AmazonS3/latest/userguide/checking-object-integrity-upload.html
- S3 events at-least-once/out-of-order: https://docs.aws.amazon.com/AmazonS3/latest/userguide/notification-how-to-event-types-and-destinations.html
- S3 sequencer per object key: https://docs.aws.amazon.com/AmazonS3/latest/userguide/notification-content-structure.html
- AS2 MDN/MIC: https://www.rfc-editor.org/rfc/rfc4130.html
- Camel File idempotent/readLock: https://camel.apache.org/components/next/file-component.html
- OpenLineage Run/Job/Dataset facet: https://openlineage.io/docs/spec/facets/

**新しい実行・実証はNOT RUN**。本節は既存main/Issue/外部一次資料を使った静的設計追補であり、正式API、受理済みSchema、HULFT/AS2/S3/TEI実E2E、安全な本番送受信、TEI Resource Scheduler、複数Object Atomic Transactionを実装した主張ではない。

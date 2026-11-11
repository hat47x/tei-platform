# ETL・Batch処理方式の汎用化：既存Function・Data Operations・Operatorの合成による要件・基本設計・実装計画

- **状態**: 実装前設計・**Owner未採用**（2026-11-11）。本文は要件の解釈と実装計画の提案であり、製品受入・正式API/DSL・Scheduler・Hostの採用ではない。
- **役割**: 要求の解釈・基本設計・実装計画。設計判断の正本はADR、作業条件の正本はGitHub Issue、現行実装の説明は[現行構成](../../docs/architecture.md)と[実装状況](../../docs/status.md)にある。本文はそれらを置き換えない。
- **対象Issue**: [#1130](https://github.com/hat47x/tei-platform/issues/1130)（Batch/帳票/一括処理）、[#2622](https://github.com/hat47x/tei-platform/issues/2622)（Data Operations/Provider）、[#2542](https://github.com/hat47x/tei-platform/issues/2542)（ファイル・メッセージ連携）、[#2545](https://github.com/hat47x/tei-platform/issues/2545)（段階的データ移行の未決・closed）、[#1129](https://github.com/hat47x/tei-platform/issues/1129)（副作用規則）、[#2499](https://github.com/hat47x/tei-platform/issues/2499)（限定Grant）、[#4194](https://github.com/hat47x/tei-platform/issues/4194)（TEI→EKI→Provider縦断）、[#4384](https://github.com/hat47x/tei-platform/issues/4384)（TEI-EKI統合計画）。
- **基準版**: TEI `origin/main` `7fe5de4d9b8f611d7f945693a6383eaa59cba035`（2026-11-11）、EKI `hat47x/eki-fabric` `e15d3351f333a7cd7229acf001ff83ae5dce8120`。
- **本文が決めないこと**: 新しいETL専用DSL、第二のデータモデル、第二のWorkflow Engine、TEI専用の常駐Scheduler/共通実行Host、正式なCLI文法、`effect_mode`の拡張。これらは§15のOwner判断事項として残す。

## 1. 問題と結論

TEIには既に、型付き意味（`semantic.Data`・`Attribute`・`Relation`）、操作（`semantic.Capability`）、実行計画（`semantic.RunPlan`）、業務規則（`semantic.Validation`・`EffectRule`）、期間（`semantic.Period`）、DB取得（`internal/query`の`RelationQuery`・`DerivedSelect`・`standard`Query IR）、実DB実行（`internal/driver/queryruntime`）、実行同定と再開（`internal/runlog`・`internal/execution/runplan.go`）、成果物とManifest（`internal/artifact`）、期間状態（`internal/periodstate`）がある。**足りないのは能力の総量ではなく、それらをETL/Batchの一連の流れとして結ぶ契約と、実行入口・予定・場所・権限・結果の分離である。**

結論は次の五つである。

1. **ETL専用の第二DSL・第二データモデル・第二Workflow Engineを作らない。** Batchは「型付きData＋Function（Capability/RunPlan/Validation/EffectRule）＋Data Operations＋Operator/Plugin＋Entry」の合成として表す。新しい抽象は、既存の抽象で表現できないことが実例で示された場合にだけ追加する。
2. **BatchのFunction定義と、実行入口・実行予定・実行場所・実際の権限・実行結果を分離する。** 一つの宣言が五つの顔を同時に確定したとみなさない（§2）。TEI専用の常駐Schedulerも共通実行Hostも必須にしない。
3. **実行方式（一括・チャンク・ストリーミング・差分・増分）は、変換の意味ではなく取得と出力の契約として選ぶ。** 方式の違いでIdentity・粒度・順序・重複・NULL・欠損・部分取得・精度・Source Revisionが変わるなら、それは意味の変更であり、黙って方式だけを替えない（§4・§6）。
4. **再実行の安全性はCheckpointの有無では決まらない。** 純粋変換・冪等な書込み・非冪等な外部作用・Commit結果不明を別の級に分け、級ごとに許可される再実行だけを許す（§9）。
5. **最初の縦断実装は副作用のない決定論的Batchに限定する。** 固定した小さな入力ファイルから型付きデータを読み、Filter・Convert・Aggregateを実行し、決定的な出力ファイルを生成する。同じ意味を直接Go実行とCLI実行の二入口から呼び、Input・Expected・実Output・生成Artifact・実行版を一つの票で追跡する（§12）。

## 2. 五つの顔の分離：Batch Function定義・入口・予定・場所・権限・結果

| 顔 | 決めること | 決めないこと | 既存の置き場（基準版で確認） |
| --- | --- | --- | --- |
| **意味（Batch Function定義）** | 対象DataとRelation path、入力の型・必須・NULL可否、出力の型・粒度、Effect級、冪等級、再開時に期待する同一性 | いつ起動するか、どこで動くか、誰が許したか | `semantic.Capability`（inputs/output/outcomes/effect）、`semantic.RunPlan`（operations/key/concurrency/resume/outputs/autonomy）、`semantic.Validation`、`semantic.EffectRule` |
| **入口（Entry）** | 到達可能な呼出し形（CLI argv、HTTP route、timer設定、EKI Task、Host内関数呼出し）と、その入口が運べる引数・成果物・時点 | 入口が公開されていること＝必要な意味を運べること | `cmd/tei`の`acceptedCommands`、`scenario`のverify/generate/run/check、`cmd/tei/cli_relation_query.go` |
| **予定（Schedule）** | 起動の期待（毎日・締め日・イベント・手動）と時刻の出所 | 誰が実際に起動するか。業務日付の意味は`Period`/`Calendar`側 | 宣言は`semantic.Period`・`Calendar`の時刻意味のみ。実起動は外部 |
| **場所（Place）** | 実ロードされるBinary/Artifact、OS、作業ディレクトリ、同時実行の範囲 | 意味や権限。場所が変わっても意味IDは変えない | `internal/plugin/build.go`の`BuildPlan`・`BuildArtifactLock`、`internal/buildintent`、[ADR-0043](../adr/0043-target-native-artifacts-without-tei-host.md) |
| **権限（Authority）** | 呼出し時点のPrincipal・Scope・Grant、操作対象、Expected Version | 宣言に書いたRole名の自己申告 | `internal/permission`、`internal/execution/permissions.go`の`GuardFromGrants`・`PermissionPreconditions` |
| **結果（Result）** | Run/Attempt/Fence、項目ごとの状態、件数、失敗行、Artifact、Evidence、Receipt、Unknown | Job終了コード0＝業務Commit | `internal/runlog.Record`、`internal/execution.StepResult`、`internal/artifact.Manifest`、`internal/failure` |

この分離の実務的な意味は、「宣言した」「入口として到達可能」「今この時点で許可されている」「実際に実行した」「成果物が実在する」「業務として確定した」が**別の証拠**になることである。一つのPASSを隣へ昇格させない。

`autonomy unattended`の扱いは既存の設計に従う。`internal/execution/runplan.go`の`RunPlan`は前提検査が未供給なら停止し、`blocked`を記録する（`internal/runlog.Blocked`）。ETL/Batchでも、無人実行の宣言だけでなければ実行しない。

## 3. 取得：DB・CSV・TSV・JSON・JSONL・固定長・REST/OData・メッセージ

取得元を一つの「Reader」型に潰さない。取得元ごとに、**宣言できること・実測できること・失われること**が違うためである。

| 取得元 | 既存の実装（基準版） | 宣言に必要 | 黙って主張してはいけないこと |
| --- | --- | --- | --- |
| RDB | `internal/query`の`RelationQuery`（Root/Relations/Filters/Sorts/PageSize/PageOffset/Depth/Mode/FetchMode/BatchSize/ResultColumns）、`standard`Query IR（`ParseSQLToStandard`/`StandardToSQL`）、`internal/driver/queryruntime/{postgres,sqlite}`の`ExecuteBrowseQuery`/`ExecuteRelationQuery`/`Runner.ExecuteSQL`、`internal/database/provider`の`Provider`能力（metadata-read/query/data-mutation/schema-change/explain/transaction/cancel） | Data Source aliasとDB種別、Read Snapshot、NULL可否、順序の有無、ページ境界、取得行数の上限 | 一つのSELECT結果の行数＝母集団。`FetchMode`/`BatchSize`の変更＝意味不変 |
| CSV/TSV | 参照実装`tools/reference-data-import-check`（ヘッダー照合・型照合・キー欠落/重複・参照切れ・`initial`/`operational`の境界）。製品パッケージには専用のCSV読取契約はない | 文字コード、区切り、引用規則、ヘッダー有無、列名→Attributeの明示対応、型の解釈、空文字とNULLの区別 | 同名ヘッダー＝同じ意味。空文字＝NULL。行番号＝Identity |
| JSON/JSONL | `internal/jsonl`（追記・読取の規則を一箇所に集約。owner-only 0600、壊れた行はエラーとし、その行より前は返す） | 1行=1レコードの境界、キーの重複規則、深さ・項目数の上限、数値の精度表現 | JSON objectの同名キーをMapとして読んだ結果＝元Source |
| 固定長 | 参照実装`tools/reference-month-end-close-correction`の`FileSpec`（`SpecDigest`・`ParseFile`・`EncodeRecord`・`EmitFile`・`FileSHA256`・`NormalizeForCompare`、CP932/windows-31j、項目長、詰め文字、後書き件数/合計、変換不能文字のレコード保留） | レコード長、項目の開始位置と長さ、符号、詰め、件数・合計の後書き、比較等価規則 | 参照実装の`FileSpec`が製品パッケージへ移植済みであること。バイト同一の再出力が全文字集合で成立すること |
| REST/OData | 製品パッケージに標準Adapterはない。[#1656](https://github.com/hat47x/tei-platform/issues/1656)が担当 | 操作・対象・展開・フィルタ・ページ方式・ETag/版・Batch実行の原子性・Requirement側の`$select`/`$expand`対応 | Batch要求＝原子的。`$expand`の結果＝Relationの完全な取得 |
| メッセージ | 製品パッケージに標準Broker Adapterはない。[#2542](https://github.com/hat47x/tei-platform/issues/2542)が担当 | 配信保証（at-most-once/at-least-once/不明）、重複の判定キー、順序キー、ACK主体、保持期限 | ACK受領＝業務取込完了。順序保証＝欠落なし |

**取得の共通契約（提案）**: 取得結果は「値の列」ではなく、**対象集合（範囲）・取得時点・完全性（complete/partial/unknown）・順序保証（none/per-key/global）・NULL/欠損の区別・Source Revision**を伴うレコード列として返す。これは新しい型を先に固定する提案ではなく、既存の`RelationQuery`・`DerivedSelect`・`FileSpec`・`Manifest`が既に部分的に持っている情報を、ETLの一連の流れで落とさないための要求である。

## 4. 変換：Filter・Map・Join・Aggregate・Lookup・Validate・Convert・Enrich

変換を新しい演算子群として宣言しない。**既存の表現へ落とし、落とせないものを`unsupported`として返す。**

| 変換 | 落とす先（既存） | 明示が必要な条件 | 落とせない場合 |
| --- | --- | --- | --- |
| Filter | `standard`Query IRの`Filter`ノード、`RelationQuery.Filters`、`execution/outputs.go`の投影前条件 | NULL を含む述語の意味（三値論理）、比較の等価規則（照合順序・正規化）、絞り込み後の完全性 | 条件が Provider 側で評価できない、または評価結果が同一でない場合は`unsupported` |
| Map（項目変換） | `application.Pattern`の列対応、`query.ResultColumnMapping`、`semantic.Attribute.TypeRef` | 型変換の定義域（変換不能値）、精度・丸め・単位、Locale | 型の定義域が未宣言なら`unresolved` |
| Join | `RelationQuery.Relations`（`RelationStep`の`JoinKind`/`Cardinality`）、`standard`Query IRの`Join`ノード | 結合キーの一意性、多対多の重複増殖、外部結合の欠損表現 | 結合キーが一意でない状態で「1行=1件」を主張しない |
| Aggregate | `standard`Query IRの`Aggregate`、`application.Pattern.Aggregates`（`Name`/`Attribute`/`Function`）、`query.DerivedSelect`（`subtract`/`multiply`の純粋演算に限定） | 粒度（Group key）、空集合とNULLの区別、合計の精度、Windowの完全性 | `DerivedSelect`が受ける形（単表・派生前の物理操作なし・`MaxRows`上限あり）を超える集計は`ErrDerivedUnsupported` |
| Lookup | `semantic.Relation`の`Keys`による参照、参照実装`tools/reference-partial-import-plan`の`conflict`/`unresolved`判定 | 参照先の存在・一意性・時点、参照切れの扱い（保留か拒否か） | 参照先の版・時点が未確定なら`unknown` |
| Validate | `semantic.Validation`（`positive`/`relation-unique`）、`internal/validation.Validate`、参照実装`tools/reference-data-import-check`の行別結果 | 拒否（業務Reject）と実行不能（技術失敗）の区別、項目ごとの理由コード | 評価不能を「適合」に丸めない |
| Convert（形式・単位・コード） | 型の`Underlying`、`FileSpec`の`NormalizeForCompare`、`internal/formatversion`（形式版の変換は別契約） | 変換表の版と有効期間、非可逆な変換（丸め・切捨て・コード縮約）の明示 | 可逆でない変換を往復同値として扱わない |
| Enrich（外部参照による補完） | `semantic.Capability`の呼出し（`hook.Declared.Call`の宣言済み能力のみ）、`EffectRule`の`stage` | 外部参照の時点、取得失敗時の部分結果、再取得の冪等性 | 外部参照が非冪等ならEffect級を上げる |

**変換の合成規則**: 変換は`RunPlan.Operations`の順序に従う。各操作の入力は前段の出力であり、`hook.Hook`の`Reads`/`Uses`/`Produces`宣言を超える参照を許さない（`internal/hook/declared.go`の宣言外呼出し拒否と同じ規則）。項目ごとの途中結果は`execution.StepResult`として残し、途中で止まった場合に「どこまで変換したか」を件数で示す。

## 5. 実行方式の比較：一括・チャンク・ストリーミング・差分・増分

実行方式は、**取得の境界**と**出力の境界**の組み合わせとして選ぶ。方式の名前だけを選び、意味の保存条件を選ばないことは許さない。

| 方式 | 単位 | 成立する条件 | 失われやすいもの | 最初の適用範囲 |
| --- | --- | --- | --- | --- |
| **一括** | 対象集合全体 | 対象集合を列挙できる（または上限`MaxRows`で有界）、全体がメモリまたは一時領域に収まる | 巨大集合での所要メモリ、長い単一トランザクション | **最初の縦断実装の既定** |
| **チャンク** | 部分集合（ページ・区切り） | チャンク間で独立に変換でき、集計がチャンクをまたがない、または境界をまたぐ集計を明示的に別操作とする | チャンク境界をまたぐ集計・順序・重複排除、途中失敗時の部分出力 | 集計を含む場合は「境界をまたぐ集計の禁止」を宣言 |
| **ストリーミング** | 到着順の連続 | 各レコードが独立に変換でき、窓（event-time/処理時刻）と遅延許容が宣言されている | 全体の完全性、リプレイ可能性、順序保証、後着による再計算 | 窓・水位・遅延許容を宣言できる場合のみ |
| **差分** | 前回との差 | 前回の基準（Source Revision・スナップショット・水位）が同一で、削除・再出現を区別できる | 削除、キーの再利用、時刻の巻き戻し | 基準Revisionの同一性を検査できる場合のみ |
| **増分** | 未処理範囲 | 進捗（Checkpoint）が入力の位置と意味の版に束縛され、再開点の不一致を検出できる | 途中でのSource改版、追記と更新の混在、順序の入替 | §9の再開検査が通る場合のみ |

**方式選択の規則（提案）**:
1. 既定は**一括**。対象集合が有界であることを`MaxRows`または明示の上限で示せない場合は、一括を名乗らない。
2. チャンクは、**集計の窓がチャンク境界と一致するか、集計を境界の外（全件取得後）へ移す**ことを条件とする。境界をまたぐ`SUM`を、チャンクごとの`SUM`の合計として黙って置き換えない。
3. ストリーミングは、窓と遅延許容を宣言できない限り採用しない。宣言できない場合、到着順の処理は「一括の逐次実行」であり、完全性の主張を伴わない。
4. 差分・増分は、**前回の基準Revisionを入力として受け取り、今回のSource Revisionと一致しなければ拒否**する。進捗だけを信用しない。

## 6. 業務処理の類型：単純変換・定期バッチ・締め・移行・バックフィル・再集計

類型ごとに、必要になる証拠と停止条件が違う。類型を一つの「Batch」にまとめると、締めの不可逆性と単純変換の可逆性が同じ扱いになる。

| 類型 | 代表例 | 必要な追加契約 | 停止条件 |
| --- | --- | --- | --- |
| 単純なファイル変換 | 固定長→TSV、TSV→JSONL | 入出力の形式宣言、決定的な出力、副作用なし | 出力が入力の意味を失う変換を含む場合は全体拒否 |
| 定期バッチ | 日次集計、通知生成 | 対象期間、実行同定（`runKey`）、同一runKeyの重複起動の拒否 | 期間・対象集合が未確定なら実行しない |
| 業務締め処理 | 月次締め、確定帳票 | `semantic.Period`の状態（open/closing/closed）、`close-requires`の充足、訂正（`correction`）の別操作 | `closed`の宣言だけを業務確定とみなさない。[#2545](https://github.com/hat47x/tei-platform/issues/2545)の未決（観測の窓・縮約の実行）が残る範囲は`NOT RUN` |
| データ移行 | 旧DB→新DB、段階移行 | 区切り・再開地点・照合規則（件数・金額合計・キー集合・行ハッシュ）、旧版書込みの観測 | 照合規則のない移行、旧版書込みを観測できない縮約 |
| バックフィル | 過去期間の再生成 | 対象期間の列挙、Source Revisionの固定、出力の世代管理 | 「全期間」を列挙できないまま全件再生成を主張する |
| 再集計 | 集計規則の改版後の再計算 | 旧出力との比較（件数だけでなく合計・キー集合）、Consumerの版`pin` | 件数一致を内容一致として扱う（§13 N-08） |

## 7. 出力先：DB・ファイル・オブジェクトストレージ・外部API・メッセージ

出力先ごとに、**冪等性の作り方**と**失敗時に分かること**が違う。出力を一つの「Writer」に潰さない。

| 出力先 | 冪等性の作り方 | 失敗時に確定できること | 既存の接続先 |
| --- | --- | --- | --- |
| DB | 業務キー＋Operation ID＋Expected Version（CAS） | 適用件数、競合、Commit不明 | `internal/command`の`Lifecycle`、`internal/security/effectcommit`、`internal/database/provider`の`data-mutation` |
| ファイル | 一時ファイルへの書込み→検証→原子的rename、`Manifest`の世代 | 書込み完了、digest、世代の混在 | `cmd/tei/cli_operations_artifacts.go`の`writeArtifactFiles`（同名衝突を拒否）、`internal/artifact.Manifest`（`InputRevision`混在を拒否）、`internal/artifact`の`StageArtifacts`/`PublishStage` |
| オブジェクトストレージ | Put＋digest＋世代、上書き禁止または版付き | 転送完了、内容一致、公開可視 | 標準Adapterなし。参照実装の範囲を超える主張をしない |
| 外部API | 冪等キーがある場合のみ再送可。ない場合は`lookup_first` | 応答、Receipt、Commit不明 | `internal/failure`の`EffectMayBeUnknown`/`IsRetryable`、[#1174](https://github.com/hat47x/tei-platform/issues/1174) |
| メッセージ | 業務イベントIDによる重複排除、at-least-once前提 | 送信受付、ACK主体ごとの到達、不明 | [#2542](https://github.com/hat47x/tei-platform/issues/2542)。ACK受領＝全宛先到達ではない |

**出力の共通規則（提案）**: 出力は必ず「何を・どの世代で・どの消費者のために」を持つ。`artifact.Manifest`の`Complete`は「未解決項目がない」ことを示すだけで、Consumerが実体を読んだことを示さない。よって出力のPASSは、(a) 生成側のManifest、(b) 実ファイルのdigest、(c) Consumer側の読取、を別の列で記録する。

## 8. 依存関係・条件分岐・並列化

- **依存関係**: `RunPlan.Operations`の順序が第一の依存表現である。データ依存（前段の出力を後段が読む）と制御依存（前段の成否で後段を実行するか決める）を区別する。既存の`RunPlan`は順序のみを持ち、制御依存を明示しない。条件分岐を追加する場合も、`RunPlan`の意味を変えずに「実行しない理由」を`StepResult`へ残す形を先に検討する（新しいBranch型を先行導入しない）。
- **条件分岐**: 分岐の条件は業務規則（`semantic.Validation`）または入力の有無で表し、分岐の結果は「実行した／しない（理由）」の二値で記録する。分岐しなかったことを「成功」と混同しない。
- **並列化**: `RunPlan.Concurrency`（`singleton`/`per-key`）が既存の同時実行の範囲を定める。並列化の単位は**runKeyの異なる実行**であり、同一runKey内の操作を並列化しない。`internal/runlog.Owns`のfence epochが古い担当の書込みを拒否する。
- **部分失敗**: 並列化した場合、一部のrunKeyだけ成功し得る。全体を一つの成否に畳まず、runKeyごとの結果を残す。

## 9. 再実行・Checkpoint・再開・失敗回復

### 9.1 四つの級を区別する

| 級 | 例 | 再実行 | 必要な証拠 |
| --- | --- | --- | --- |
| **純粋変換** | Filter・Convert・Aggregate、決定的なファイル出力 | 同じ入力・同じ意味版なら安全（出力は同一digestになるべき） | 入力digest、意味版、出力digest |
| **冪等な書込み** | 業務キー指定のUPSERT、同一Operation IDの再送 | 同じキー・同じ内容なら安全。適用0件を成功として区別 | Operation ID、業務キー、適用件数、Receipt |
| **非冪等な外部作用** | メール送信、外部APIのPOST、メッセージ送出 | **自動再実行しない。**照会→Owner回復 | 送信Attempt、応答、Receiptまたは不明 |
| **Commit結果不明** | タイムアウト、ACK喪失、接続断 | **自動再実行しない。**照会してから判断 | `internal/failure`のEffectUnknown、Receipt照会結果 |

`internal/failure`は既に`Kind`（`business_rejection`/`technical_failure`/`partial_success`）、`RetryPolicy`（`retryable`/`not_retryable`/`lookup_first`）、`HasEffectUnknown`を持つ。ETL/Batchはこの分類を再利用し、第二の失敗分類を作らない。

### 9.2 Checkpointが保証しないこと

Checkpointは「どこまで進んだか」の記録であり、**「同じ意味のまま再開できる」ことの証明ではない。** 再開の安全性は次の一致で決まる。

```text
再開可能 =
  runKey 一致
  ∧ Plan（Function意味）の revision 一致
  ∧ 入力の Source Revision 一致
  ∧ 入力digest（または処理済み範囲のdigest）一致
  ∧ 出力世代が未公開、または公開済みなら冪等
  ∧ attempt/fenceEpoch が現在の所有と一致
```

一つでも不一致なら再開せず、`CHECKPOINT_*`の理由付きで拒否する。特に**Source改版後の古いCheckpoint再利用**は拒否する（§13 N-06）。`internal/runlog.Decide`は未完了かつ`resume checkpoint`なら`resume`を返すが、それは進捗の記録に基づく判定であり、Source Revisionの一致を検査しない。ETL側でその検査を足す。

### 9.3 途中失敗と部分出力

- 入力の途中切断（固定長のレコード境界、CSVの引用未閉じ、JSONLの壊れた行）は、**部分を成功として扱わず全体を拒否**する。参照実装`FileSpec`の`RejectError`と`import_unit`の全体拒否がこの規則の先例である。
- 出力は一時領域へ書き、検証後に公開する。途中までの出力を公開領域へ置かない。
- 途中で止まった実行の再開では、**処理済みの範囲**と**未処理の範囲**を件数で示す。再開地点の不一致（`offset`のずれ、Sourceの入れ替え）は拒否する。

### 9.4 失敗回復の順序

1. 失敗の級を判定する（`internal/failure`）。
2. 純粋変換なら、入力を固定して再実行する。
3. 冪等な書込みなら、同じOperation IDで再送し、適用件数0を「既に適用済み」として区別する。
4. 非冪等・Commit不明なら、Receipt照会を行う。照会できない場合は`unknown`のまま止め、成功にも失敗にも丸めない。
5. 回復の結果は元のRunとは別のAttemptとして記録し、元のRunの記録を書き換えない。

## 10. 実行入口・予定・場所・権限・結果の接続

- **ローカル実行**: `tei`の単一バイナリを直接実行する。既定の入口であり、EKIを必要としない（[ADR-0043](../adr/0043-target-native-artifacts-without-tei-host.md)、[#1130](https://github.com/hat47x/tei-platform/issues/1130) BD-01）。最初の縦断実装はこの経路だけを使う。
- **既存Scheduler**: OSのtimer・既存のバッチ基盤・CIなどが`tei`を起動する。TEIは起動の主体を持たず、起動された実行の同定と記録だけを持つ。
- **EKI委任**: 重い処理・資源制約・期限・再試行を外部へ委ねる場合の経路。EKIの外部契約（`spec/external/openapi.json`の`/v1/tasks`・`/v1/tasks/{id}`・`/v1/tasks/{id}/cancel`・`/v1/tasks/{id}/result`、`spec/worker/v1/worker.proto`のセッションと`ResultClaim`）を参照する。**基準版の`TaskRequest`は`parameters`をWorkerへ配送せず、`reliability.effect_mode`は`none`のみ**（`spec/external/README.md`）。したがって、パラメータを運ぶBatchや書込みを伴うBatchを、この版のEKI Taskへ黙って載せない。これは[#4194](https://github.com/hat47x/tei-platform/issues/4194)のBD-04/BD-05と同じ停止線である。
- **外部Hostへの組込み**: 既存の.NET/Java/Webプロセスから関数として呼ぶ経路。成果物の形はTarget側が決め、TEIは意味と生成物の追跡を持つ（[ADR-0043](../adr/0043-target-native-artifacts-without-tei-host.md)）。
- **権限**: 入口ごとに現在のPrincipal・Scope・Grantを判定する。宣言に書いたRole名や能力名を、実行許可の代わりにしない。`internal/execution/permissions.go`の`RequirementsFromCatalog`と`GuardFromGrants`を再利用し、ETL専用の権限エンジンを作らない。
- **結果**: 入口が違っても、同じ意味ID・同じ入力・同じ意味版なら、生成物のdigestは一致すべきである。一致しない場合は、入口・Binary・Artifact・環境の差を記録し、意味の差と環境の差を混同しない。

## 11. 実行状況・件数・部分成功・失敗行・処理時間・Evidence・Receipt

観測は「実行の記録」と「成果物の証拠」と「外部作用の受領」を分ける。

| 観測対象 | 記録する内容 | 記録しない内容 | 既存の置き場 |
| --- | --- | --- | --- |
| 実行 | runKey、attempt、fenceEpoch、項目ごとの状態（resolved/blocked/failed）、開始・終了時刻、実行したBinaryの版 | 顧客データの値、秘密 | `internal/runlog.Record`、`internal/execution.StepResult` |
| 件数 | 入力件数、選択件数、変換件数、出力件数、拒否件数、保留件数 | 件数だけを内容一致の根拠にすること | `internal/execution/outputs.go`の`ProjectOutputs`/`RenderOutputs`は件数の投影であり、母集団の完全性を示さない |
| 部分成功 | `partial_success`、成功範囲、未処理範囲、理由 | 全体成功への丸め | `internal/failure`の`Kind` |
| 失敗行 | 行番号、理由コード、項目名 | 行の値 | 参照実装`tools/reference-data-import-check`の`RowOutcome`（`Row`/`Status`/`Codes`）が値を持たない設計の先例 |
| 処理時間 | 入力取得・変換・出力・検証の区間時間 | 一つの合計時間だけ | 計測は実行記録の項目として持つ（第二の計測基盤を作らない） |
| Evidence | 入力digest、意味版、出力digest、Manifest、実行版、検証結果 | 検証していない項目の「適合」 | `internal/artifact`の`EvidenceBundle`/`WriteEvidenceBundle`（基準版で追加済み） |
| Receipt | Operation ID、対象、Expected Version、応答、照会結果、不明 | Jobの終了コードからの推測 | `internal/security/effectcommit`、[#1174](https://github.com/hat47x/tei-platform/issues/1174) |

**部分成功の扱い**: 部分成功は独立した状態であり、成功でも失敗でもない。集計・帳票・確定を伴うBatchでは、部分成功のまま次の段へ進まない。進む場合は、業務Ownerが許した範囲だけを入力にする。

## 12. 最初の縦断実装（副作用のない決定論的Batch）

### 12.1 形

固定した小さな入力ファイルから型付きデータを読み、Filter・Convert・Aggregateを実行し、決定的な出力ファイルを生成する。DB書込み・外部API・メッセージ送出を行わない。同じ意味を**直接Go実行**と**CLI実行**の二入口から呼び、結果を比較する。

```text
入力: examples/etl-first-slice/input/orders.tsv（固定・合成データ）
      ヘッダー: order_id  customer  status  quantity  unit_price  currency
      型宣言: order_id→std.identifier, customer→std.identifier,
              status→std.code, quantity→std.integer,
              unit_price→std.decimal, currency→std.code
      Filter: status == "accepted"
      Convert: unit_price を JPY へ（宣言した変換表の版を固定）
      Aggregate: customer ごとの quantity 合計 と amount 合計
出力: order-summary.tsv（列順・改行LF・小数桁を宣言し、内容で決定的）
      run.json（runKey/attempt/fenceEpoch/項目ごとの状態/件数/所要時間/意味版）
      manifest.json（出力ファイルのdigestと世代）
```

### 12.2 既存APIの使い方（新規に作らないもの）

| 用途 | 使う既存API | 作らないもの |
| --- | --- | --- |
| 入力の型付き読取 | `encoding/csv`（TSV読取）、`semantic.Data`/`Attribute`の型宣言、参照実装`tools/reference-data-import-check`のヘッダー・型・キー照合の規則 | 第二のReader型、第二の型システム |
| Filter/Convert/Aggregate | `standard`Query IRの`Filter`/`Aggregate`ノード相当の意味、`query.DerivedSelect`（純粋演算の範囲）、`derivationref` | 第二の演算子DSL、第二の集計エンジン |
| 実行同定と再開 | `internal/runlog`の`RunKey`/`Decide`/`Owns`/`File` | 第二のJournal形式 |
| 実行ループ | `internal/execution.RunPlan`（`StepFunc`で各段を実行、`StepResult`で項目ごとの状態） | 第二のRunner |
| 出力の世代とdigest | `internal/artifact.Manifest`または同形の独自Manifest（`InputRevision`混在の拒否規則を写す） | 第二のArtifact Store |
| 実行版の記録 | `cmd/tei/cli_version.go`の`cliBuildIdentity`（`VCSRevision`/`VCSModified`/`DeclaredRevision`の区別） | 第二の版記録 |
| 直接Go実行 | 同じ変換を`tools/reference-etl-batch-slice`（新規参照実装）内の関数として呼ぶ | 変換ロジックの二重実装 |

### 12.3 二入口の対応付け

「同じ意味のFunctionを直接Go実行とCLI実行へ対応付ける」とは、次の四点を一つの票で比較できることである。

| 比較列 | 直接Go実行 | CLI実行 | 一致しない場合に言うこと |
| --- | --- | --- | --- |
| Input | 固定入力ファイルのdigest | 同じdigest | 入力が違えば比較しない |
| Expected | 独立に用意した期待出力（実装出力から生成しない） | 同じ期待出力 | 期待が実装由来なら独立でない（§13の原則） |
| 実Output | 出力ファイルのdigest | 同じdigest | 入口で結果が変われば意味または環境の差を記録 |
| 生成Artifact | Manifestのdigest | 同じdigest | 片方だけがManifestを作る状態を成功としない |
| 実行版 | 実行した関数を含むBinaryの版 | CLIの`cliBuildIdentity` | 版が違えば同一性を主張しない |

実装の完了条件は「二つの入口が同じ期待出力を再現し、**入力・意味版・出力digest・実行版を別の列で記録できる**」ことである。二入口の一致は、意味の受理でも業務確定でもない。

### 12.4 CLI入口の候補

新しいCLIサブコマンド名（候補: `batch`）と引数の形は**未採用**である。既存の`scenario run`（`-scenario`・`-workspace`・`-step`とpin必須）と`relation-query`の形を踏襲できるかを、W2で比較してから決める。CLI名の追加は`cmd/tei/cli_commands.go`の`acceptedCommands`と`cmd/tei/cli_flag_contract.go`の許可リストの両方に触れるため、宣言と実装の一致試験を同時に置く。

## 13. 独立Expected（負例）

いずれも**実装出力から生成しない**独立の期待値として固定する。参照実装の既存の先例がある場合は、その規則を再利用し、第二の期待値正本を作らない。

| ID | 仕込み | 独立Expected |
| --- | --- | --- |
| **N-01** NULL・欠損・不正型・変換不能 | 金額が空欄、数量が`"abc"`、通貨が未宣言のコード、NULLと欠損を混在 | 正常なNULLは失敗ではない。不正型は行番号と理由コードで拒否。変換不能な値を0や空文字へ置換しない。`0`と`NULL`と`未取得`を区別 |
| **N-02** 重複行・順序変更・部分取得 | 同一`order_id`の行を2件、入力順を入れ替え、後半のページ取得だけ失敗 | 重複は業務キーの重複として検出し、暗黙にlast-winsにしない。行順を意味の順序とみなさない。部分取得は`partial`として返し、全件の集計を主張しない |
| **N-03** チャンク境界をまたぐ集計 | 100行を25行×4チャンクで処理し、`customer`別の合計をチャンクごとに計算 | チャンクごとの合計の和を全体の合計として出すことを拒否。境界をまたぐ集計は全件取得後または明示のWindowで行う |
| **N-04** 入力途中切断・途中失敗・再開地点不一致 | 固定長ファイルの最終レコードが途中で切れている。実行が3段目で停止。再開時に前回の位置と入力の入れ替え | 切断は全体拒否。3段目までを`resolved`として記録し、再開は4段目から。位置の不一致は再開拒否（`unknown`へは丸めない） |
| **N-05** 同じ入力の再処理・重複書込み | 完了したrunKeyで再実行。同じOperation IDで再送。別Operation IDで同じ対象を更新 | 完了済みrunKeyは`refuse`。同一Operation IDの再送は適用0件／Receipt照会で「既に適用済み」と区別。別IDの再適用は重複Effectとして検出（件数だけで判定しない） |
| **N-06** Source改版後の古いCheckpoint再利用 | Checkpoint取得後に入力ファイルが差し替わり、Source Revisionが変わる | 再開を拒否し、古いCheckpointと新しい入力の組合せを実行しない。進捗だけを信用しない |
| **N-07** 外部書込みのCommit結果不明 | 外部APIがタイムアウト、応答なし、ACK喪失 | 成功にも失敗にも丸めない。Receipt照会を行い、照会できない場合は`unknown`のまま停止。自動再送しない |
| **N-08** 件数一致でも金額合計・Identity不一致 | 出力件数は12件で一致するが、金額合計が1円異なり、キー集合が1件ずれている | 件数一致を内容一致として扱わない。件数・合計・キー集合・行ハッシュを別々に比較し、不一致を成功としない |
| **N-09** 入口の取り違え | CLIと直接Go実行で同じ意味を実行し、片方だけ古いBinaryを使う | 出力digestが一致しても実行版が違えば同一性を主張しない。版が違う場合は差を記録し、意味の一致と環境の一致を分ける |
| **N-10** 部分成功の全成功への丸め | 全12件中10件が変換・出力され、2件が拒否 | `partial_success`として残し、全体成功の表示にしない。拒否2件の理由と行番号を残し、確定・帳票の段へ進まない |
| **N-11** 空集合と未取得 | 対象0件（正しい空）と、Reader失敗（未取得）と、Reader未設定 | 三つを別の結果にする。空集合を「0件の成功」に、Reader失敗を「0件」に丸めない（`internal/execution/outputs.go`の`ProjectOutputs`が件数と理由を別に返す先例） |
| **N-12** 帳票・合計の精度 | 単価`300.005`を小数2桁の通貨へ丸め、合計を集計 | 丸めの時点（行ごとか合計後か）を宣言し、丸めの前後で合計が変わることを記録。丸めを無宣言の既定にしない |

## 14. 作業単位と実装計画

実装を次の八つの作業単位に分ける。各単位は独立に検証でき、前段の完了を待たずに設計・参照実装を進められる範囲を明示する。**変更ファイルは候補**であり、着手時に最新mainと再照合する。

### W1 取得（Source Adapter）— 固定入力の型付き読取

- **対象API**: `encoding/csv`、`semantic.Data`/`Attribute`の型宣言、参照実装`tools/reference-data-import-check`の照合規則、`internal/query`の`RelationQuery`（DB経路）
- **入力・出力**: 宣言された型のTSV/CSV→型付きレコード列（値・完全性・順序保証・Source Revision付き）
- **テスト**: ヘッダー重複・未知列・型不一致・キー欠落・空文字とNULL・引用未閉じ・文字コード違い
- **依存先**: なし（DB経路は`internal/driver/queryruntime`に依存）
- **想定変更ファイル**: `tools/reference-etl-batch-slice/`（新規）、`examples/etl-first-slice/`（新規の固定入力）
- **完了条件**: N-01・N-02・N-11が独立Expectedで通る。値を持たない失敗行記録ができる
- **停止条件**: 製品パッケージのReaderを新設する必要が出た時点で止め、`#2622`/`#2542`へ戻す

### W2 変換（Operator合成）

- **対象API**: `standard`Query IRの`Filter`/`Join`/`Aggregate`、`query.DerivedSelect`と`ValidateDerivedSelect`（`ErrDerivedUnsupported`/`ErrDerivedIncomplete`）、`derivationref`、`semantic.Validation`
- **入力・出力**: 型付きレコード列→変換後の型付きレコード列＋項目ごとの途中結果
- **テスト**: 三値論理のFilter、多対多Joinの重複増殖、空集合のAggregate、`MaxRows`超過、派生不能な集計の`unsupported`
- **依存先**: W1
- **想定変更ファイル**: `tools/reference-etl-batch-slice/contract.go`、`internal/query/standard/`（既存のまま使えるかの確認に限る）
- **完了条件**: 八つの変換すべてについて「落とす先」「明示が必要な条件」「落とせない場合の返し方」が表で埋まり、N-03・N-08・N-12が通る
- **停止条件**: 新しい演算子型・第二の集計エンジンが必要になった時点で止める

### W3 出力（Writer）と世代

- **対象API**: `internal/artifact.Manifest`（`InputRevision`混在拒否）、`cmd/tei/cli_operations_artifacts.go`の同名衝突拒否、一時ファイル＋原子的rename
- **入力・出力**: 変換後のレコード列→決定的な出力ファイル＋`manifest.json`
- **テスト**: 途中失敗で公開領域に部分ファイルが残らない、同名の別内容を拒否、LF/列順/小数桁の決定性
- **依存先**: W2
- **想定変更ファイル**: `tools/reference-etl-batch-slice/output.go`、`examples/etl-first-slice/expected/`
- **完了条件**: 同じ入力と同じ意味版で出力digestが再現し、N-04・N-05（出力側）が通る
- **停止条件**: `artifact.Kind`の拡張が必要になった時点で止め、[#2482](https://github.com/hat47x/tei-platform/issues/2482)へ戻す

### W4 Job生成（Batch Function定義）

- **対象API**: `semantic.Capability`、`semantic.RunPlan`、`semantic.EffectRule`、`internal/validation/runplan.go`の`RUN_PLAN_*`診断
- **入力・出力**: 意味宣言→検証済みのBatch定義（操作順・key・concurrency・resume・autonomy・Effect級・冪等級）
- **テスト**: 操作なし、keyなし、未知のconcurrency/resume/autonomy、checkpointかつoutputなし、Effect級と冪等級の矛盾
- **依存先**: W2（意味の形が固まる）
- **想定変更ファイル**: `internal/validation/`（既存診断の再利用）、`plan/design/`（本文）
- **完了条件**: 五つの顔（§2）がそれぞれ別の欄で宣言でき、`RUN_PLAN_*`で表現できない欄（Effect級・冪等級）の扱いがOwner判断事項として記録される
- **停止条件**: 新しいDSLキーワードが必要になった時点で止め、[#1130](https://github.com/hat47x/tei-platform/issues/1130)のOwner判断へ戻す

### W5 CLI実行と二入口対応

- **対象API**: `cmd/tei/main.go`のdispatch、`cmd/tei/cli_commands.go`の`acceptedCommands`、`cmd/tei/cli_flag_contract.go`の`selectedReadOnlyFlagAllowlist`、`cmd/tei/cli_version.go`の`cliBuildIdentity`
- **入力・出力**: 固定入力パス・出力先・意味版→`run.json`＋API/CLI JSON
- **テスト**: 引数不足、未知のフラグ、`-json`の出力契約、`cliBuildIdentity`の`VCSRevision`/`DeclaredRevision`の区別、直接Go実行との出力digest一致
- **依存先**: W1〜W4
- **想定変更ファイル**: `cmd/tei/cli_etl_batch.go`（新規候補）、`cmd/tei/cli_commands.go`、`cmd/tei/cli_flag_contract.go`
- **完了条件**: §12.3の五列が一つの票で埋まり、N-09が通る
- **停止条件**: CLI名の採用判断が無いまま`acceptedCommands`へ追加しようとした時点で止める

### W6 検証・Evidence

- **対象API**: `internal/artifact`の`EvidenceBundle`/`WriteEvidenceBundle`/`StageArtifacts`/`PublishStage`、`internal/scenariotest`の`Parse`/`Generate`と`scenario`のpin付き実行
- **入力・出力**: 実行記録＋出力digest＋意味版→Evidence票（検証した項目と未検証の項目を分ける）
- **テスト**: 期待出力が実装出力から生成されていないこと（独立Expectedの検査）、未検証項目を「適合」にしないこと
- **依存先**: W5
- **想定変更ファイル**: `plan/evidence/`（実行記録）、`tools/reference-etl-batch-slice/evidence.go`
- **完了条件**: §11の表の各行が証拠で埋まり、N-08・N-10が通る
- **停止条件**: 第二のEvidence Storeや第二の検証基準が必要になった時点で止める

### W7 再開・回復

- **対象API**: `internal/runlog`（`RunKey`/`Decide`/`Owns`/`File`）、`internal/execution.RunPlan`、`internal/failure`、`internal/security/effectcommit`
- **入力・出力**: Checkpoint（runKey・意味版・Source Revision・入力digest・処理済み範囲・attempt/fence）→再開可否の判定と再開実行
- **テスト**: N-04・N-05・N-06・N-07。Source Revision不一致の拒否、Commit不明で自動再送しないこと
- **依存先**: W4・W6
- **想定変更ファイル**: `tools/reference-etl-batch-slice/resume.go`、`plan/design/`の§9追補
- **完了条件**: §9.2の五条件すべてに検査があり、Checkpointの存在だけを安全の根拠にしないことが試験で固定される
- **停止条件**: 外部Effectの再実行可否をTEI単独で確定しようとした時点で止め、[#1174](https://github.com/hat47x/tei-platform/issues/1174)/[#2499](https://github.com/hat47x/tei-platform/issues/2499)へ戻す

### W8 外部実行連携（EKI・既存Host）

- **対象API**: EKI外部契約（`/v1/tasks`ほか）、`semantic.Capability`の入出力、`internal/plugin/build.go`の`BuildPlan`/`BuildArtifactLock`
- **入力・出力**: Batch Function意味＋入力Artifact→外部Task要求（または既存Host呼出し）と、結果・Receipt・不明の記録
- **テスト**: `parameters`が運ばれないことの診断、`effect_mode=none`以外を要求するBatchの拒否、Result body取得可否の区別
- **依存先**: W5・W7、EKI側のOwner
- **想定変更ファイル**: 参照実装の範囲（`tools/reference-etl-batch-slice/external.go`は**候補であり、この版のEKIへ接続しない**）
- **完了条件**: 意味・入口・実行・Receiptを別列で記録し、`wire-shape-only`を実行済みと表示しない
- **停止条件**: EKIの契約版（OpenAPIのGit blob）を固定できないまま接続を主張する時点で止める（[#4194](https://github.com/hat47x/tei-platform/issues/4194)の停止線）

**実装順**: W1 → W2 → W3 → W5（W4をW2と並行）→ W6 → W4確定 → W7 → W8。最初の縦断実装（§12）はW1〜W3＋W5で成立し、W6で証拠を付ける。W7・W8は縦断実装の成立後に拡張する。

## 15. Owner判断事項（未受理の実行契約・副作用規則）

以下は本文が決めない。決めるまで実装の対象にしない、または既定案を選んで覆す条件を記録する。

| ID | 判断事項 | 既定案（本文が提案するもの） | 覆す条件 | 判断の場所 |
| --- | --- | --- | --- | --- |
| O-01 | Batch定義を`run-plan`の拡張にするか、独立の宣言（候補`batch`）にするか | `run-plan`＋`capability`の合成で表し、新キーワードを追加しない | 既存の`RUN_PLAN_*`で表現できない欄が二つ以上の実例で示された場合 | [#1130](https://github.com/hat47x/tei-platform/issues/1130) |
| O-02 | CLIサブコマンド名と引数形 | `scenario run`の形を踏襲する候補`batch` | 既存の`scenario`と意味が衝突する場合 | W5 |
| O-03 | Checkpointの保存場所 | 実行ディレクトリ内のJSON Lines（`internal/runlog.File`と同形） | 複数Hostからの同時実行が必要になった場合 | W7 |
| O-04 | 出力Manifestを`internal/artifact.Manifest`へ統合するか | 同形の独自Manifestから開始し、統合は別Issue | 統合で`Kind`の意味が曖昧になる場合 | [#2482](https://github.com/hat47x/tei-platform/issues/2482) |
| O-05 | チャンク境界と集計の禁止規則をDSLで表すか診断で表すか | 診断（実行前拒否）で表す | 実行時の動的な境界決定が必要になった場合 | W2 |
| O-06 | Effect級・冪等級を意味宣言へ持たせるか | 別欄として宣言し、CapabilityのEffectと混ぜない | 既存`semantic.EffectRule`で表現できることが示された場合 | [#1129](https://github.com/hat47x/tei-platform/issues/1129) |
| O-07 | EKI `TaskRequest.parameters`の未配送をどう扱うか | 意味損失として診断し、EKIへ載せない | EKIがパラメータ配送の契約版を公開した場合 | [#4194](https://github.com/hat47x/tei-platform/issues/4194) |
| O-08 | `effect_mode=none`以外を要求するBatchの経路 | EKIへ載せず、既存Host/Providerの別経路へ戻す | EKIが書込みプロファイルを公開した場合 | [#4194](https://github.com/hat47x/tei-platform/issues/4194) |
| O-09 | 移行の観測窓・縮約実行・異種DB | 観測窓は`NOT RUN`、縮約は実行しない | [ADR](../adr/README.md)とOwnerの判断が揃った場合 | [#2545](https://github.com/hat47x/tei-platform/issues/2545) |
| O-10 | 無人の締め処理の許可範囲 | 前提（期間・Effect Observer・Grant）が揃わなければ停止 | 運用側の期限と`latest_useful`の意味が確定した場合 | [#1130](https://github.com/hat47x/tei-platform/issues/1130) |

## 16. 主張しないこと

- 本文は設計と計画であり、実装・受入の完了ではない。§14のW1〜W8の**実Go実装・実CLI・実DB・実EKI・実MFT・実Providerは、この版では`NOT RUN`**である。
- 参照実装（`tools/reference-*`）の局所試験のPASSを、製品パッケージでの成立として扱わない。特に`FileSpec`の文字コード・外字・バイト同一再出力、`batch-completeness`の完全性判定は、合成入力に対する狭いcharacterizationである。
- EKIの外部契約は**読取り専用の参照**である。`TaskRequest.parameters`の未配送と`effect_mode=none`は、この版の契約記述に基づく制約であり、EKIの全実装能力の断定ではない。
- 件数の投影（`execution.RenderOutputs`）は母集団の完全性・Data Quality・Consumer受領を示さない。
- `artifact.Manifest.Complete`は未解決項目がないことを示すだけで、Consumerが実体を読んだことを示さない。
- 期間が`closed`であることは、対象Data・Recipient・Effectの完了を示さない。
- 本設計は、新しいETL専用DSL・第二のデータモデル・第二のWorkflow Engine・TEI専用Scheduler/共通Hostを**採用していない**。

## 17. 引継ぎ（下位モデル向け）

- **基準コミット**: TEI `origin/main` `7fe5de4d9b8f611d7f945693a6383eaa59cba035`、EKI `e15d3351f333a7cd7229acf001ff83ae5dce8120`。着手時に`git fetch origin`で再確認する。
- **実装順**: W1（取得）→W2（変換）→W3（出力）→W5（CLI二入口）。W4（Job定義）はW2と並行。W6（Evidence）→W4確定→W7（再開）→W8（外部連携）。
- **最初の縦断実装**: §12。固定TSV→Filter（`status=="accepted"`）→Convert（JPY換算、変換表の版を固定）→Aggregate（`customer`別の`quantity`/`amount`）→決定的TSV＋`run.json`＋`manifest.json`。副作用なし。直接Go実行とCLI実行で同じ出力digestを再現する。
- **独立Expected**: §13のN-01〜N-12。実装出力から生成せず、`examples/etl-first-slice/expected/`へ固定する。
- **未決**: §15のO-01〜O-10。決めるまで実装の対象にしない、または既定案を選んで覆す条件をPR本文へ書く。
- **停止線**: §14の各Wの停止条件と、§1の「第二のDSL/データモデル/Workflow Engineを作らない」規則。
- **次の作業**: 本設計の起票（Issue更新と子Issue起票）までで止め、大規模実装は開始しない。実装はW1の独立Expectedから始める。

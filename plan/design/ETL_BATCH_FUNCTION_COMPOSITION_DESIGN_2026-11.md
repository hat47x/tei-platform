# ETL・Batch処理方式の汎用化：既存Function・Data Operations・Operatorの合成による要件・基本設計・実装計画

- **状態**: 実装前設計・**Owner未採用**（2026-11-11）。本文は要件の解釈と実装計画の提案であり、製品受入・正式API/DSL・Scheduler・Hostの採用ではない。
- **役割**: 要求の解釈・基本設計・実装計画。設計判断の正本はADR、作業条件の正本はGitHub Issue、現行実装の説明は[現行構成](../../docs/architecture.md)と[実装状況](../../docs/status.md)にある。本文はそれらを置き換えない。
- **対象Issue**: [#1130](https://github.com/hat47x/tei-platform/issues/1130)（Batch/帳票/一括処理）、[#2622](https://github.com/hat47x/tei-platform/issues/2622)（Data Operations/Provider）、[#2542](https://github.com/hat47x/tei-platform/issues/2542)（ファイル・メッセージ連携）、[#2545](https://github.com/hat47x/tei-platform/issues/2545)（段階的データ移行の未決・closed）、[#1129](https://github.com/hat47x/tei-platform/issues/1129)（副作用規則）、[#2499](https://github.com/hat47x/tei-platform/issues/2499)（限定Grant）、[#4194](https://github.com/hat47x/tei-platform/issues/4194)（TEI→EKI→Provider縦断）、[#4384](https://github.com/hat47x/tei-platform/issues/4384)（TEI-EKI統合計画）。子Issueは[#4410](https://github.com/hat47x/tei-platform/issues/4410)・[#4411](https://github.com/hat47x/tei-platform/issues/4411)、残課題は[#4434](https://github.com/hat47x/tei-platform/issues/4434)・[#4435](https://github.com/hat47x/tei-platform/issues/4435)。
- **基準版**: TEI `origin/main` `7fe5de4d9b8f611d7f945693a6383eaa59cba035`（2026-11-11）、EKI `hat47x/eki-fabric` `e15d3351f333a7cd7229acf001ff83ae5dce8120`。
- **本文が決めないこと**: 新しいETL専用DSL、第二のデータモデル、第二のWorkflow Engine、TEI専用の常駐Scheduler/共通実行Host、正式なCLI文法、`effect_mode`の拡張。これらは§15のOwner判断事項として残す。

## 1. 問題と結論

TEIには既に、型付き意味（`semantic.Data`・`Attribute`・`Relation`）、操作（`semantic.Capability`）、実行計画（`semantic.RunPlan`）、業務規則（`semantic.Validation`・`EffectRule`）、期間（`semantic.Period`）、DB取得（`internal/query`の`RelationQuery`・`DerivedSelect`・`standard`Query IR）、実DB実行（`internal/driver/queryruntime`）、実行同定と再開（`internal/runlog`・`internal/execution/runplan.go`）、成果物とManifest（`internal/artifact`）、期間状態（`internal/periodstate`）がある。**足りないのは能力の総量ではなく、それらをETL/Batchの一連の流れとして結ぶ契約と、実行入口・予定・場所・権限・結果の分離である。**

結論は次の五つである。

1. **ETL専用の第二DSL・第二データモデル・第二Workflow Engineを作らない。** Batchは「型付きData＋Function（Capability/RunPlan/Validation/EffectRule）＋Data Operations＋Operator/Plugin＋Entry」の合成として表す。新しい抽象は、既存の抽象で表現できないことが実例で示された場合にだけ追加する。
2. **BatchのFunction定義と、実行入口・実行予定・実行場所・実際の権限・実行結果を分離する。** 一つの宣言が六つの顔を同時に確定したとみなさない（§2）。TEI専用の常駐Schedulerも共通実行Hostも必須にしない。
3. **実行方式（一括・チャンク・ストリーミング・差分・増分）は、変換の意味ではなく取得と出力の契約として選ぶ。** 方式の違いでIdentity・粒度・順序・重複・NULL・欠損・部分取得・精度・Source Revisionが変わるなら、それは意味の変更であり、黙って方式だけを替えない（§4・§6）。
4. **再実行の安全性はCheckpointの有無では決まらない。** 純粋変換・冪等な書込み・非冪等な外部作用・Commit結果不明を別の級に分け、級ごとに許可される再実行だけを許す（§9）。
5. **最初の縦断実装は副作用のない決定論的Batchに限定する。** 固定した小さな入力ファイルから型付きデータを読み、Filter・Convert・Aggregateを実行し、決定的な出力ファイルを生成する。同じ意味を直接Go実行とCLI実行の二入口から呼び、Input・Expected・実Output・生成Artifact・実行版を一つの票で追跡する（§12）。

## 2. 六つの顔の分離：Batch Function定義・入口・予定・場所・権限・結果

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
| **チャンク** | 部分集合（ページ・区切り） | Source集合が重複なく完全被覆され、結合可能な中間状態とGroup Key・NULL・Decimal・丸め規則が保存される場合のみ部分集計をmergeできる | チャンク欠落/重複、同一Keyの別chunk出現、中間丸め、COUNT DISTINCTの不正な合算 | W2で正負例を固定してから適用 |
| **ストリーミング** | 到着順の連続 | 各レコードが独立に変換でき、窓（event-time/処理時刻）と遅延許容が宣言されている | 全体の完全性、リプレイ可能性、順序保証、後着による再計算 | 窓・水位・遅延許容を宣言できる場合のみ |
| **差分** | 前回の基準Revision→今回の対象Revision | 新しいRunでbase→targetの連続性と変更・削除・遅着の捕捉が検証できる | CDC欠落、watermarkの飛び、削除・Key再利用 | Sourceの差分能力を証明できなければunsupported/unknown |
| **増分** | 未処理範囲 | 進捗（Checkpoint）が入力の位置と意味の版に束縛され、再開点の不一致を検出できる | 途中でのSource改版、追記と更新の混在、順序の入替 | §9の再開検査が通る場合のみ |

**方式選択の規則（提案）**:
1. 既定は**一括**。対象集合が有界であることを`MaxRows`または明示の上限で示せない場合は、一括を名乗らない。
2. チャンク集計は**無条件許可でも無条件禁止でもない**。入力全件の重複ない被覆、Group Keyの同一性、結合可能な中間状態、NULL・Decimal・丸めの保存が検証できる場合のみmergeする。SUMは精度が保存される加算、AVGは(sum,count)、COUNT DISTINCTは集合や正確なDistinct状態等の別条件を要求する。
3. ストリーミングは、窓と遅延許容を宣言できない限り採用しない。宣言できない場合、到着順の処理は「一括の逐次実行」であり、完全性の主張を伴わない。
4. **同一Runの再開**ではcheckpointに束縛したSource snapshot/revision・意味版・範囲digestが一致しなければ拒否する。一方**新規差分Run**ではbaseRevisionとtargetRevisionの不一致は通常の入力であり、base→targetの連続性、変更・削除捕捉、watermark/遅着処理を個別検査する。差分取得の能力が無い場合、全件比較による代替を明示・検証できなければunsupported/unknown。

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

### 9.2 Checkpointが保証しないこと（同一Runの再開・Attemptの切替を含む）

Checkpointは「どこまで進んだか」の記録であり、**「同じ意味のまま再開できる」ことの証明ではない。** 再開の安全性は次の一致で決まる。

```text
再開可能 =
  runKey 一致
  ∧ Plan（Function意味）の revision 一致
  ∧ 入力の Source Revision 一致
  ∧ 入力digest（または処理済み範囲のdigest）一致
  ∧ 出力世代が未公開、または公開済みなら冪等
  ∧ 旧Attempt/fenceEpochの来歴が識別可能で、再開主体の現在所有と旧主体の競合拒否が確認できる
```

一つでも不一致なら再開せず、`CHECKPOINT_*`の理由付きで拒否する。特に**Source改版後の古いCheckpoint再利用**は拒否する（§13 N-06）。再起動後は新しいAttempt/fenceEpochになり得るため、**旧Checkpointのepochと現在epochの数値一致を要求するのではない**。旧Attemptの記録、新しい実行主体の現在所有、旧主体の競合書込み拒否を別々に検査する。`internal/runlog.Decide`は未完了かつ`resume checkpoint`なら`resume`を返すが、それは進捗に基づく判定であり、Source Revisionの一致・外部Providerの古いWriterの実効的な停止を証明しない。共有Host/Providerでのfencingが未実証なら`unsupported/unknown`で止める。

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

「同じ意味のFunctionを直接Go実行とCLI実行へ対応付ける」とは、次の五つの列を一つの票で比較できることである。

| 比較列 | 直接Go実行 | CLI実行 | 一致しない場合に言うこと |
| --- | --- | --- | --- |
| Input | 固定入力ファイルのdigest | 同じdigest | 入力が違えば比較しない |
| Expected | 独立に用意した期待出力（実装出力から生成しない） | 同じ期待出力 | 期待が実装由来なら独立でない（§13の原則） |
| 実Output | 出力ファイルのdigest | 同じdigest | 入口で結果が変われば意味または環境の差を記録 |
| 生成Artifact | Data output memberのdigestとManifest member集合・世代を別記 | 同じData output digestを要求し、Manifestは構成memberごとに検査 | run.json/attempt時刻を含むManifest全体digestの一致は要求しない。Consumer実読取の資格はさらに別 |
| 実行版 | 実行した関数を含むBinaryの版 | CLIの`cliBuildIdentity` | 版が違えば同一性を主張しない |

実装の完了条件は「二つの入口が同じ期待出力を再現し、**入力・意味版・出力digest・実行版を別の列で記録できる**」ことである。二入口の一致は、意味の受理でも業務確定でもない。

### 12.4 CLI入口の候補

新しいCLIサブコマンド名（候補: `batch`）と引数の形は**未採用**である。既存の`scenario run`（`-scenario`・`-workspace`・`-step`とpin必須）と`relation-query`の形を踏襲できるかを、W5で比較してから決める。CLI名の追加は`cmd/tei/cli_commands.go`の`acceptedCommands`と`cmd/tei/cli_flag_contract.go`の許可リストの両方に触れるため、宣言と実装の一致試験を同時に置く。

### 12.5 Target・実行単位の拡張（Go単一バイナリを標準とし、他Targetへは同じ意味を再生成する）

**既定はGoの単一バイナリ**である。ETL/Batchの意味は「型付きData＋Function＋Data Operations＋Operator＋Entry」の合成として定義し、Targetはその**落とし先**でしかない。Targetが変わっても意味ID・入力・期待出力は変えない（[ADR-0043](../adr/0043-target-native-artifacts-without-tei-host.md)）。

| Target | 基準版の実装 | ETL/Batchでの扱い | 落とせない場合 |
| --- | --- | --- | --- |
| **Go（既定）** | `tools/reference-application-binary`（`internal/appgen`の`Build`/`Generator`、`templates/*.tmpl`、P0-A縦断試験）、`cmd/tei`単一バイナリ | そのまま実行単位とする。`internal/plugin/build.go`の`BuildPlan`/`BuildArtifactLock`で依存閉包と成果物を束縛 | Build closureが検証できない場合は未実装 |
| **Java** | `tools/reference-generator-selection`の`templates/*.java.tmpl`（`Main.java`・`LineAmounts.java`・`TeiFailure.java`）と`java_test.go`・`java_failure_contract_test.go` | 契約・fixture・同じ独立Expectedで候補を生成・選択・検証する。失敗契約（`TeiFailure`）もTarget側で同一の区分を保つ | build入力の対象が基準版で解消できない場合は`unsupported` |
| **Python / JavaScript** | 同`templates/*.py.tmpl`・`*.js.tmpl`と`python_test.go`・`javascript_test.go`、`wasi_node.js` | 同一契約の別Target候補として扱う。interpreterのsnapshot digest・file同一性・size・更新時刻を記録し、実行中にsourceが変わったらcaseを拒否する（`source changed during execution`） | snapshot digestが取れない場合は実行として扱わない |
| **Wasm / WASI** | 同`wasi_test.go`・`wasm_test.go`、`templates/wasi_node.js` | 同一bytesを再現してからbuildする。実行はsandbox/measurementの範囲で行い、OS権限の境界を`reference-generator-selection`側の隔離に委ねる | build入力・OS承認・権限境界が未確認なら実行しない |
| **.NET / C#** | **基準版にGenerator・テンプレートは確認できない** | 未評価。[#4435](https://github.com/hat47x/tei-platform/issues/4435)が「追加できる／追加できない（制約付き）」を証拠付きで判定する。ETL/Batchの設計はTarget非依存なので、`.NET`実装の不在はblocking条件ではない | 候補が無ければ`unsupported`。黙ってGoへ読み替えない |

**Target追加の規則（提案）**:
1. 新しいTargetは「Generator候補」として契約・fixture・同一独立Expectedで検証してから実行単位にする。意味側にTarget名を持ち込まない（`#2510`/`#988`/`#1018`の境界）。
2. Targetごとの実行は、`execution.measurement`（build・処理・完了の経過時間、`userCpuNanos`/`systemCpuNanos`、`goos`/`goarch`/`harnessGoVersion`）とSource/binary digest・Receiptを別に記録する。測定値は性能比較であり、意味の同一性の証明ではない。
3. 実行中にSource/binaryが変わったcaseは拒否し、その前に読み取ったsnapshotのdigestを残す。`execution.sourceDigest`とそのcaseの直前のsnapshot digestを別の値として扱う。
4. 実行中にSource/binaryが変わったcaseは拒否し、その前に読み取ったsnapshotのdigestを残す。`execution.sourceDigest`とそのcaseの直前のsnapshot digestを別の値として扱う。
5. EKI・外部Hostへの載せ替え（W8）もTargetの一種として扱い、`wire-shape-only`を「Target拡張済み」と表示しない。
6. `.NET`のようにGeneratorが存在しないTargetは、`unsupported`として記録し、Goで代替した事実を独立Expectedの不一致として残す。

**縦断実装との関係**: 最初の縦断実装は**Go単一バイナリのみ**で成立させる（§12.1〜12.4）。Java/Wasm等への拡張は、同一の`examples/etl-first-slice/expected/`を用いて候補生成・選択・検証できることを別Issueで示してから行う。二Targetの成功を、一つの成功へ畳まない。

## 13. 独立Expected（負例）

いずれも**実装出力から生成しない**独立の期待値として固定する。参照実装の既存の先例がある場合は、その規則を再利用し、第二の期待値正本を作らない。

| ID | 仕込み | 独立Expected |
| --- | --- | --- |
| **N-01** NULL・欠損・不正型・変換不能 | 金額が空欄、数量が`"abc"`、通貨が未宣言のコード、NULLと欠損を混在 | 正常なNULLは失敗ではない。不正型は行番号と理由コードで拒否。変換不能な値を0や空文字へ置換しない。`0`と`NULL`と`未取得`を区別 |
| **N-02** 重複行・順序変更・部分取得 | 同一`order_id`の行を2件、入力順を入れ替え、後半のページ取得だけ失敗 | 重複は業務キーの重複として検出し、暗黙にlast-winsにしない。行順を意味の順序とみなさない。部分取得は`partial`として返し、全件の集計を主張しない |
| **N-03** チャンク集計の有資格Merge | 同一customerが複数chunkに出現。正例は全件を重複なく被覆しDecimal中間状態を保持、負例はchunk欠落/重複・中間丸め・AVGの平均値をそのまま平均 | 正例のみ全件集計と同一値/Key/digest。負例は拒否/unknown。SUMとAVGの結合条件を検査し、COUNT DISTINCTを無条件加算しない |
| **N-04** 入力途中切断・Step途中失敗・完了直後停止・再開点不一致 | 固定長の最終record切断。Step3開始後失敗した場合とStep3完了を永続観測後Step4前に停止した場合を分離 | 切断は全体拒否。前者はStep1/2のみresolved、Step3 failed/unknownとして再実行資格を独立判定。後者はStep3完了の耐久証拠がある場合に限りStep4から再開。位置/Sourceの不一致は拒否 |
| **N-05** 同じ入力の再処理・重複書込み | 完了したrunKeyで再実行。同じOperation IDで再送。別Operation IDで同じ対象を更新 | 完了済みrunKeyは`refuse`。同一Operation IDの再送は適用0件／Receipt照会で「既に適用済み」と区別。別IDの再適用は重複Effectとして検出（件数だけで判定しない） |
| **N-06** Source改版後の古いCheckpoint再利用 | Checkpoint取得後に入力ファイルが差し替わり、Source Revisionが変わる | 再開を拒否し、古いCheckpointと新しい入力の組合せを実行しない。進捗だけを信用しない |
| **N-07** 外部書込みのCommit結果不明 | 外部APIがタイムアウト、応答なし、ACK喪失 | 成功にも失敗にも丸めない。Receipt照会を行い、照会できない場合は`unknown`のまま停止。自動再送しない |
| **N-08** 件数一致でも金額合計・Identity不一致 | 出力件数は12件で一致するが、金額合計が1円異なり、キー集合が1件ずれている | 件数一致を内容一致として扱わない。件数・合計・キー集合・行ハッシュを別々に比較し、不一致を成功としない |
| **N-09** 入口の取り違え | CLIと直接Go実行で同じ意味を実行し、片方だけ古いBinaryを使う | 出力digestが一致しても実行版が違えば同一性を主張しない。版が違う場合は差を記録し、意味の一致と環境の一致を分ける |
| **N-10** 部分成功の全成功への丸め | 全12件中10件が変換・出力され、2件が拒否 | `partial_success`として残し、全体成功の表示にしない。拒否2件の理由と行番号を残し、確定・帳票の段へ進まない |
| **N-11** 空集合と未取得 | 対象0件（正しい空）と、Reader失敗（未取得）と、Reader未設定 | 三つを別の結果にする。空集合を「0件の成功」に、Reader失敗を「0件」に丸めない（`internal/execution/outputs.go`の`ProjectOutputs`が件数と理由を別に返す先例） |
| **N-12** 通貨・合計の精度と適用時点 | 単価300.005、複数量・複通貨、異なるrate適用日、未登録通貨。rate表の版/Source digestと業務基準期を固定 | quantity×unit_price×rateの演算順、Decimal scale、行別/合計後の丸め、NULL・未登録通貨の拒否を独立Expectedに固定。暗黙0円・無宣言丸めを認めない |

## 14. 作業単位と実装計画

実装を次の八つの作業単位に分ける。各単位は独立に検証でき、前段の完了を待たずに設計・参照実装を進められる範囲を明示する。**変更ファイルは候補**であり、着手時に最新mainと再照合する。

### W1 取得（Source Adapter）— 固定入力の型付き読取

- **対象API**: `encoding/csv`、`semantic.Data`/`Attribute`の型宣言、参照実装`tools/reference-data-import-check`の照合規則、`internal/query`の`RelationQuery`（DB経路）
- **入力・出力**: 宣言された型のTSV/CSV→型付きレコード列（値・完全性・順序保証・Source Revision付き）
- **テスト**: ヘッダー重複・未知列・型不一致・キー欠落・空文字とNULL・引用未閉じ・文字コード違い。rate表の有効日・業務基準期・Source digest・未登録通貨と独立Expectedを先に固定
- **依存先**: なし（DB経路は`internal/driver/queryruntime`に依存）
- **想定変更ファイル**: `tools/reference-etl-batch-slice/`（新規）、`examples/etl-first-slice/`（新規の固定入力）
- **完了条件**: N-01・N-02・N-11が独立Expectedで通る。値を持たない失敗行記録ができる
- **停止条件**: 製品パッケージのReaderを新設する必要が出た時点で止め、`#2622`/`#2542`へ戻す

### W2 変換（Operator合成）

- **対象API**: `standard`Query IRの`Filter`/`Join`/`Aggregate`、`query.DerivedSelect`と`ValidateDerivedSelect`（`ErrDerivedUnsupported`/`ErrDerivedIncomplete`）、`derivationref`、`semantic.Validation`
- **入力・出力**: 型付きレコード列→変換後の型付きレコード列＋項目ごとの途中結果
- **テスト**: 三値論理のFilter、多対多Joinの重複増殖、空集合のAggregate、`MaxRows`超過、派生不能な集計の`unsupported`。N-03ではSUM/AVGの有資格Merge正常系と欠落/重複/中間丸めの負例を同じ独立Expectedで比較
- **依存先**: W1
- **想定変更ファイル**: `tools/reference-etl-batch-slice/contract.go`、`internal/query/standard/`（既存のまま使えるかの確認に限る）
- **完了条件**: 八つの変換すべてについて「落とす先」「明示が必要な条件」「落とせない場合の返し方」が表で埋まり、N-03・N-08・N-12が通る
- **停止条件**: 新しい演算子型・第二の集計エンジンが必要になった時点で止める

### W3 出力（Writer）と世代

- **対象API**: `internal/artifact.Manifest`（`InputRevision`混在拒否）、`cmd/tei/cli_operations_artifacts.go`の同名衝突拒否、一時ファイル＋原子的rename
- **入力・出力**: 変換後のレコード列→決定的な出力ファイル＋`manifest.json`
- **テスト**: 途中失敗で公開領域に部分ファイルが残らない、同名の別内容を拒否、LF/列順/小数桁の決定性。単一Fileのrenameと多member一式の公開資格を別々に判定し、Consumer pin・実byte読取は#2604へ接続
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
- **完了条件**: 六つの顔（§2）がそれぞれ別の欄で宣言でき、`RUN_PLAN_*`で表現できない欄（Effect級・冪等級）の扱いがOwner判断事項として記録される
- **停止条件**: 新しいDSLキーワードが必要になった時点で止め、[#1130](https://github.com/hat47x/tei-platform/issues/1130)のOwner判断へ戻す

### W5 CLI実行と二入口対応

- **対象API**: `cmd/tei/main.go`のdispatch、`cmd/tei/cli_commands.go`の`acceptedCommands`、`cmd/tei/cli_flag_contract.go`の`selectedReadOnlyFlagAllowlist`、`cmd/tei/cli_version.go`の`cliBuildIdentity`
- **入力・出力**: 固定入力パス・出力先・意味版→`run.json`＋API/CLI JSON
- **テスト**: 引数不足、未知のフラグ、`-json`の出力契約、`cliBuildIdentity`の`VCSRevision`/`DeclaredRevision`の区別、直接Go実行との**Data output digest**一致。Attempt/時刻を含む`run.json`とManifestの全byte一致は要求しない
- **依存先**: W1〜W4
- **想定変更ファイル**: `cmd/tei/cli_etl_batch.go`（新規候補）、`cmd/tei/cli_commands.go`、`cmd/tei/cli_flag_contract.go`
- **完了条件**: §12.3の五列が一つの票で埋まり、N-09が通る
- **停止条件**: CLI名の採用判断が無いまま`acceptedCommands`へ追加しようとした時点で止める

### W6 検証・Evidence

- **対象API**: `internal/artifact`の`EvidenceBundle`/`WriteEvidenceBundle`/`StageArtifacts`/`PublishStage`、`internal/scenariotest`の`Parse`/`Generate`と`scenario`のpin付き実行
- **入力・出力**: 実行記録＋出力digest＋意味版→Evidence票（検証した項目と未検証の項目を分ける）
- **テスト**: 期待出力が実装出力から生成されていないこと（独立Expectedの検査）、未検証項目を「適合」にしないこと。Data output、実Attempt Evidence、Manifest member構成、Consumer readbackを別列で採否する
- **依存先**: W5
- **想定変更ファイル**: `plan/evidence/`（実行記録）、`tools/reference-etl-batch-slice/evidence.go`
- **完了条件**: §11の表の各行が証拠で埋まり、N-08・N-10が通る
- **停止条件**: 第二のEvidence Storeや第二の検証基準が必要になった時点で止める

### W7 再開・回復

- **対象API**: `internal/runlog`（`RunKey`/`Decide`/`Owns`/`File`）、`internal/execution.RunPlan`、`internal/failure`、`internal/security/effectcommit`
- **入力・出力**: Checkpoint（runKey・意味版・Source Revision・入力digest・処理済み範囲・attempt/fence）→再開可否の判定と再開実行
- **テスト**: N-04・N-05・N-06・N-07。同一Run再開はSnapshot pin不一致拒否、新規差分Runはbase→target連続性と削除/遅着捕捉を分離。Source Revision不一致の拒否、Commit不明で自動再送しないこと
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

### 実装前反証レビューの採用候補（W1〜W7横断、NOT RUN）

この段落は[PR #4409の反証レビュー](https://github.com/hat47x/tei-platform/pull/4409#issuecomment-6793680605)を作業条件へ戻す**設計候補**であり、Ownerの正式採用や実装試験の結果ではない。

- **有資格な部分集計**: 演算子ごとにmergeに必要な状態を定める。SUMは同じGroup Key・単位・正確な加算状態、AVGは`(sum,count)`、MIN/MAXはNULL規則、COUNT DISTINCTは正確な重複排除可能状態などが必要。全件被覆と重複排除の独立証拠が無ければ、部分結果が計算できても「全件完了」と主張しない。浮動小数点の非結合性や中間丸めがある場合は全件計算とbyte同値を自動主張しない。
- **差分と再開の別Claim**: `resume(same run, pinned snapshot, checkpoint)` と `delta(new run, base revision → target revision, CDC completeness)` を別資格とする。前者ではsnapshot同一性、後者では変更・削除の完全性と範囲の連続性を証明する。未知のwatermarkとlate eventは推定で補完しない。
- **再現性の三層**: (1)固定入力・rate/意味版に対する決定的な`order-summary.tsv`、(2)Attempt時刻・実Binary・fenceを含む`run.json`、(3)実Member集合を結びつけるManifest/Publicationと独立Consumer readback。二入口で(1)の等値を要求するが、(2)(3)の全byte等値は仕様にしない。Manifest単体はmulti-member atomic publicationの証拠ではない。
- **数値意味の固定**: rate適用日・業務基準期・rate Source digest、`quantity × unit_price × rate`の計算順序、各Decimal scaleと行別/集計後の丸め、NULL/未登録通貨の拒否を独立Expectedへ固定する。実行時刻からrate適用日を推測しない。
- **Step復旧**: 「Step3が失敗した」のか「Step3完了の耐久証拠を観測した後に停止した」のかを別入力で試す。ログにresolvedと書かれたことだけで外部Effectの不存在・Commitを推定しない。

**受入順**: W1でrate/入力の独立Expected→W2で有資格Mergeの正負例→W3で決定的単一Data memberと多member公開の別資格→W5/W6で二入口のData digestとAttempt evidence比較→W7で再開/新規差分・unknown effectの別資格。第二のPublisher・Manifest正本・ETL Runner・Schedulerを作らない。

## 15. Owner判断事項（未受理の実行契約・副作用規則）

以下は本文が決めない。決めるまで実装の対象にしない、または既定案を選んで覆す条件を記録する。

| ID | 判断事項 | 既定案（本文が提案するもの） | 覆す条件 | 判断の場所 |
| --- | --- | --- | --- | --- |
| O-01 | Batch定義を`run-plan`の拡張にするか、独立の宣言（候補`batch`）にするか | `run-plan`＋`capability`の合成で表し、新キーワードを追加しない | 既存の`RUN_PLAN_*`で表現できない欄が二つ以上の実例で示された場合 | [#1130](https://github.com/hat47x/tei-platform/issues/1130) |
| O-02 | CLIサブコマンド名と引数形 | `scenario run`の形を踏襲する候補`batch` | 既存の`scenario`と意味が衝突する場合 | W5 |
| O-03 | Checkpointの保存場所 | 実行ディレクトリ内のJSON Lines（`internal/runlog.File`と同形） | 複数Hostからの同時実行が必要になった場合 | W7 |
| O-04 | 出力Manifestの表現と既存Artifact契約への接続 | 既存`internal/artifact.Manifest`/`EvidenceBundle`/Stage・Publishを再利用し、第二の同形Manifest正本は作らない | 既存契約で表現できない意味が独立実例で示された場合はunsupportedとして#2482のOwnerに戻す | [#2482](https://github.com/hat47x/tei-platform/issues/2482) |
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

- **基準コミット**: TEI `origin/main` `7fe5de4d9b8f611d7f945693a6383eaa59cba035`（設計時点）、EKI `e15d3351f333a7cd7229acf001ff83ae5dce8120`。着手時に`git fetch origin`で再確認する。
- **関連Issue**: 親[#1130](https://github.com/hat47x/tei-platform/issues/1130)、子[#4410](https://github.com/hat47x/tei-platform/issues/4410)（W1/W2/W3/W5の縦断実装）、[#4411](https://github.com/hat47x/tei-platform/issues/4411)（W4/W6/W7/W8の再開・Evidence・外部境界）、[#4434](https://github.com/hat47x/tei-platform/issues/4434)（設計検証で確認したmain由来の失敗9件）、[#4435](https://github.com/hat47x/tei-platform/issues/4435)（`.NET` Target評価）。
- **実装順**: W1（取得）→W2（変換）→W3（出力）→W5（CLI二入口）。W4（Job定義）はW2と並行。W6（Evidence）→W4確定→W7（再開）→W8（外部連携）。
- **最初の縦断実装**: §12。固定TSV→Filter（`status=="accepted"`）→Convert（JPY換算、変換表の版を固定）→Aggregate（`customer`別の`quantity`/`amount`）→決定的TSV＋`run.json`＋`manifest.json`。副作用なし。直接Go実行とCLI実行で同じ出力digestを再現する。
- **独立Expected**: §13のN-01〜N-12。実装出力から生成せず、`examples/etl-first-slice/expected/`へ固定する。
- **未決**: §15のO-01〜O-10。決めるまで実装の対象にしない、または既定案を選んで覆す条件をPR本文へ書く。
- **停止線**: §14の各Wの停止条件と、§1の「第二のDSL/データモデル/Workflow Engineを作らない」規則。
- **次の作業**: 本設計の起票（Issue更新と子Issue起票）までで止め、大規模実装は開始しない。実装はW1の独立Expectedから始める。`.NET`評価は[#4435](https://github.com/hat47x/tei-platform/issues/4435)、main由来の失敗は[#4434](https://github.com/hat47x/tei-platform/issues/4434)が担当。

## 18. 入力閉包と業務期間の再集計：Claim相対の完全性・遅着・訂正（2026-11-12、設計候補）

- **状態**：既存 #1130 / #4410 / #4411 のW1〜W8、N-01〜12の受入を明確化する設計候補。新Core/正式API、第二DSL、専用ETL Engine、Runner、Authority、Manifestの採用を意味しない。実装・正式Owner判断は別資格。
- **目的**：物理ファイルの読了と、業務期間の全Fact取得を区別する。Source revisionと業務期・Capture時点・実行Attempt・公開Generationを分離し、遅着と訂正を無条件のCheckpoint再開で処理しない。

### 18.1 Claim相対のInput Closure

| 独立したClaim | 最低限必要な証拠 | そこから推論できないこと |
| --- | --- | --- |
| **Physical Read Complete** | pinned物理ObjectのEOF、length/digest/形式照合、Reader attempt | 物理Objectに業務全件が入っている |
| **Required Source Closure** | 問いに必要なSource/Relation/FX表/Rule/Partition/業務期の列挙、受理済みMeaning・Source identityとの対応 | 要求を宣言しただけで実際に読めたこと |
| **Population Complete** | 対象Factの粒度・母集団・重複規則・Partition/snapshot coverageと、必要なら独立件数・Key集合など | EOFや件数一致から業務全件・未知Readerの不存在 |
| **Temporal Cut Qualified** | 有効業務期、capture as-of、event/ingestion時刻、source snapshot・watermark範囲、遅延許容・観測時刻 | 単なるRun開始時刻やTrigger成功から業務締め完了 |
| **Output/Publication/Consumer Qualified** | 実Outputの法則適合、独立Expected、実Member bytes、Generation/Reader pin、Effect/Ownerの別証拠 | Transform成功から公開一式性・取込・業務受理 |

Claimに必要なSource/Reader/Relationが未Loaded・Unknown・非開示なら、そのClaimについて全件・不存在を主張しない。一件の存在の観測は限定的な陽性Witnessとなり得るが、不存在や全称には必要なClosureが要る。公開可能なCoverage理由と秘密の件数・パスも分ける（#2514/#2516、Studio Draft #4510、TEA Draft #4487）。ここでは新しいClaim Engineを実装しない。

**比較票（正式Schemaではない）**：Question、Accepted Meaning/Function、Data/Fact identityとbase grain、業務Periodとtimezone、必要Source roots/partition/lookup-rate revisions、期待する集合・complete条件、実Snapshot/Reader attempt/byte digest、Actual Consumed range、Coverage/欠落理由、Operator Lawと単位、独立Expected、Disclosure/Current Grant。Required・Declared・Selected・Consumed・Observedは混ぜない（#2831/#3279）。

### 18.2 業務Dの時点分離と再実行の三形態

同じ業務期Dであっても、(1) valid/effective business period、(2) source snapshot / capture as-of、(3) processing・physical Attempt、(4) publication・Reader observedAt は別の時点。event timeとingestion timeも別。Periodの端点・timezone・業務カレンダーを固定し、実行日から勝手に業務基準日を推測しない。

| 実行形態 | Source/Result条件 | 拒否・停止条件 |
| --- | --- | --- |
| **Same-Run Resume** | pinned Source/Plan/Meaning、Input digest・処理済み範囲、出力世代、current fenceを照合 | old checkpointを新Snapshotへ流用しない。Unknown Effectを自動再実行しない |
| **New Delta Run** | base revisionからtarget revisionへ連続した変更履歴、削除/訂正/再出現・watermark・遅着を確認 | base=targetを要求しない。CDCのgapや欠落が不明ならunsupported/unknown |
| **Restatement / Correction Output** | 同じ業務期でも新Source Snapshotまたは訂正Ruleを明示し、新Run・新Artifact Generationを形成 | 旧Runや旧Outputを黙って上書きしない。業務期のclose/reopen、Ledger訂正/Effectは別Owner判断 |

初期の安全な候補は**有界Snapshotを完全再読込して別Generationで再集計**する方式。CDC完全性や外部Stream受信保証が実証されるまでは自動差分・遅延補正を全件成功としない。ただし全案件に全件再集計を強制する新しい製品規則ではない。予定Slot/Trigger occurrence、配送Attempt、RunKey、実行Attempt、Operation ID、再公開Generationは別の同一性であり、同じ日付から無条件に同一IDを導出しない（#4502）。

### 18.3 MergeableとRetractableを別Traitとして扱う

- 同一単位のSUMがチャンクmerge可能でも、Sourceの全Factが重複なく被覆されたという保証は別。AVGは(sum,count)、COUNT DISTINCTは正確な集合状態等が必要で、Stockの時間方向SUM、JOIN fanout、quantityの単位混在はDecimal型だけでは適格にならない（#2593）。
- **撤回/訂正可能性は結合則からは導けない**。SUMを安全に差分更新するにはFact identityとbefore/afterまたは有資格なsigned delta、変更・削除履歴の全件性が必要。MIN/MAXで過去の極値を削除すれば残る集合または十分な状態を再参照する必要がある。Mergeable / Retractable / Source Complete / Effect Replay Safeは独立の資格。
- Traitの設計意図、静的推論、実Provider/TargetでのConformanceを区別。SQL Pushdownや並列化を数学的法則だけで実行許可せず、NULL/Collation/Decimal精度・Snapshot/Scope/現在Grantを確認する（#2622/#3279/#2514）。

### 18.4 固定反例（既存N/EF/APの独立Expectedへ接続）

新しいScenario正本を作らず、N-02/03/06/08/11/12、#2542 EF-01/04、#2604 AP-01/06の一因子変形として扱う。ここでは全行JPY、quantityは同種の「個」で統一し、rate/丸めを固定する。

| 入力の差分 | 手書きIndependent Expected | 禁止する誤成功 |
| --- | --- | --- |
| R1のp1: o1=A accepted 2個×100円、o3=A pending 100個×1円。p2: o2=A accepted 3個×200円、o4=B accepted 1個×50円 | 完全取得時はA数量5/800円、B数量1/50円 | pending算入、分割された同Groupの最終chunkだけを採用 |
| p1のEOF/digestは正常だがp2のReaderはfailed/未取得 | p1の部分結果は計算可能でも、R1全体はunqualified。Bの不存在は主張不可 | p1の物理読了を業務期D全件Completeとする |
| R1公開後、同じ業務期Dの遅着o5=A accepted 1個×40円を別Snapshot R2で観測 | R2のA数量6/840円、B数量1/50円。旧R1は当時のas-ofとして保持 | 旧Checkpointへo5を継足し、R1の証拠を改ざん |
| o2が3個から2個へ訂正されたR3 | 完全な差分または全件再読込ならA数量5/640円 | after-onlyを新規Factとして加算、旧Contributionを黙って残す |
| 同じ予定Slotを再Deliveryする一方、SourceはR1→R2に変わる | Trigger/Delivery/Run/Source/Generationを別列で観測 | 同じ日付だから同一Operation IDで無条件再開してよい |

R1のFile byteが完全でも、業務対象母集団がProvider/OwnerによりClosedと確認されなければ、R1の結果は**観測Snapshot相対**の意味に限る。遅着後の再集計がAnalysis Artifactの再発行で済むか、Period reopen / Correction / Business Effectを必要とするかは#1129/#1130/#1174のOwner判断。Artifactを新世代で公開できたことは業務Ledger更新済みの証明ではない。

### 18.5 反映順・Owner境界

1. **P0 / #4410 W1/W2**：固定TSVのEOFとClaim-relative Population Completenessを別Assertionにする。N-11の真の空/未取得、N-03のGroup merge、#2593の単位を既存独立Expectedで確認。初回SliceへStream/CDC Engineを足さない。
2. **P1 / #4410 W3/W5、#2604**：Data Output digest、Run Evidence、R1/R2公開Generationと実Consumer readbackを別資格として比較。新Publisher/Manifestを作らない。
3. **P1 / #4411 W6/W7**：Resume/Delta/Restatementの適格性表、Required・Consumed Source Closure、Source改版時の失効、Commit Unknownの照会停止を既存Runlog/Receiptへ接合。新RunKey・Checkpoint正本を追加しない。
4. **P2 / #2542/#2593/#4502/#4194**：実Provider/TargetのSnapshot・CDC・Event遅着・Trigger occurrence・EKI Resultを実証した範囲から段階的に接続。外部ACKやWorker CompletedはBusiness Acceptedではない。
5. **Studio / #1124/#4510**：Dataflow/Temporal Dependency/Publication/Evidenceを型付きGraphの用途相対Projectionとし、未取得/非開示を「図に無いので実際に存在しない」と見せない。Source改版後の旧Preview・旧選択は実編集/Effect資格に昇格させない。

**実証範囲と停止線**：今回の追補はmainの設計・既存Issue/PRの静的照合に基づく設計候補。上表の新規Execution/Go/CLI/実DB・MFT/EKI/Studio/Independent Consumer/Owner採用はNOT RUN。第二のSource/Scenario/Oracle/Temporal Engine/Host/Registry/Authorityを導入しない。

## 19. 最適化の意味保存：宣言Graph・選択Graph・実観測を照合する（2026-11-12、設計候補）

**位置づけ**：§18のInput Closure・Temporal Cutを満たしたうえで、既存Plugin最適化要件 [PRO-03〜06、PRO-09、OPT-04/06/07](PLUGIN_REALIZATION_AND_OPTIMIZATION_REQUIREMENTS_2026-11.md)、[Data Operations #2622](https://github.com/hat47x/tei-platform/issues/2622)、[Measure #2593](https://github.com/hat47x/tei-platform/issues/2593)、[Incremental Reuse #2600](https://github.com/hat47x/tei-platform/issues/2600) の適用条件を**ETL/Batch Consumer**に具体化する。正式Optimizer・新共通Plan Schema・新Verifier・新Runnerではなく、既存のSource/Typed Expression/Relation/Function/RunPlan/Plugin選択/実行Evidenceから導出する**比較投影**の提案である。

### 19.1 入出力の値一致だけでは意味保存ではない

最適化候補を、対象Question/Source Scope/Revision/Temporal Cut/Targetに相対化した**観測可能な契約**の下で比較する。

| 保存すべき観測意味 | 比較するもの | 値だけ一致した場合に見落とすこと |
| --- | --- | --- |
| Value / Grain / Multiplicity | 型、NULL/空/未取得、Fact identity、Group、重複と並び順、Unit/Decimal/Rounding | JOIN fanout、順序依存、誤った集約粒度 |
| Required Input / Coverage | 必要と実消費したSource/Relation/Partition、Snapshot、業務期、Lookup版 | 同じ行数でも対象母集団欠落、旧rateのCache hit |
| Validation / Failure | どの対象行でいつ検証し、Reject/Unknown/Partial/Diagnosticが発生するか | Filter移動で不正行の診断が消える |
| Authority / Disclosure | 選択Providerの実Read Scope、現在Grant、Consumerの開示可能範囲 | 正しい値を権限外Consumerへ返す |
| Effect / Delivery | Operation/Import Unit/Receipt、Commit Unknown、出力Generation・実Reader | 変換結果一致でも再送やPublishで業務Effectが重複 |
| Provenance / Realization | Semantic Plan/選択したBinding、実Toolchain/Artifact/Provider、実Attempt | 別Provider実行、図外処理、未ロード経路を見落とす |

**適格性の判定**： (a)必要なSource/Rule/Provider Law・現在資格が充足、(b)選択した書換えの前提が満たされ、(c)限定された独立Expectedで正常・失敗・権限の差を反証、(d)実Target上の選択・実消費・結果を別観測すること。独立FixtureのPASSは一般の数学的同値証明でも実Provider全範囲のConformanceでもない。保証範囲外はconditional/unknown/unsupportedとして止める。

### 19.2 三つのGraphを「一つの正本の異なる投影」として比較する

| 投影 | 主な要素 | まだ保証しないこと |
| --- | --- | --- |
| **Semantic / Declared Flow** | Accepted SourceからのFunction・Operator・Data/Control Dependency、Required Closure、Effect/Validation/Outputの宣言 | Providerが選ばれた、実処理した |
| **Selected / Lowered Execution** | 選択したProvider/Plugin Instance・Target/Artifact・書換え前後のRelation、実行入口/予定・Capability/Resource | 実Provider呼出・実Input Read・実Effect |
| **Observed Attempt / Consumer** | 実Entry/Loaded bytes、実Read範囲・結果・診断・Attempt/Receipt・公開Member・実Reader readback、未観測の区分 | 観測境界外に工程・Effectが無いこと |

**Edge対応票**：declared relation/operation → selected or rewritten equivalent → actual consumed/produced/receipt → evidence/coverage/unresolved reason。最適化で消えたNodeは「条件付きで正しく省略された」のか「実際には未観測・漏れた」のかを区別。選択Graphに存在しないSourceの実消費はUnexpected Edgeとして診断するが、Traceに無いものを自動的に「実行されなかった」と判定しない。Claim相対のRequired・Observed closureが有資格なときに限り限定的な不存在が言える（#2831、#1124、Studio Draft #4510、TEA Draft #4487）。秘密のSource/Edgeは許可されたViewerだけに開示する（#2516）。

### 19.3 既存Pressure Caseで検証する書換え反例

**Pushdown / Outer Join（PRO-03、OPT-04、DO-01）**：顧客A/Bがあり、Aにのみ status=accepted の注文があるとする。元Queryが LEFT JOIN の後に WHERE 注文.status=accepted を評価するなら結果はAだけ。誤った書換えでその条件を JOINのONに**移し、WHEREから取り去る**と、Bが右側NULLの行として結果に残る。ProviderでのSQL文法が正しくても、出力集合の意味は同値ではない。公式SQL説明：[PostgreSQL 18 Table Expressions](https://www.postgresql.org/docs/current/queries-table-expressions.html)。外部結合のNull-extended rows、三値論理、他のWHERE条件まで含めて書換えを資格化する。

**Filter / ValidateのFusion・順序変更（PRO-03/04、W2）**：未受理レコードに不正な数量/通貨がある入力で、業務契約が「全入力をValidateしてからacceptedだけFilter」と指定する場合、Filterを先行すると失敗行の診断が消える。逆に「acceptedだけ検証する」と受理されたMeaningならこの差は起きない。**同じ成功Output byteでもError/Diagnosticが異なる正負例**を一組として比較。Filter/Validationのどちらを正とするかをOptimizerが決めない。

**Aggregation/Join・Measure（DO-01/07、#2593 MC）**：1件の100円Factに2個のTagをJOINして結果をSUMすると200円になり得る。元Factの粒度で100円とするMeaningから、JOIN後SUMへ勝手に変換しない。単にDISTINCTした結果もFact identityやTag/OccurrenceのMeaningを失い得る。Quantity Kind、Group Key、Multiplicty、Fact identityと対象Populationを前提条件とする。

**Cache/Scope・Temporal ABA（PRO-05、OPT-06、#2600 IB-09/10）**：Source byteとRuleが同一でも、Reader Snapshot/Period・Role・ProviderのCurrent GrantやDisclosureは別資格。Tenant Aが読める結果を、同DigestだけでTenant Bに返すCache Hitは拒否。同byteへ戻ったSource A→B→Aでも、過去の受理と現在権限は自動復活しない。計算キャッシュの再利用可能性と実Consumerの現在の使用・開示資格は**別判定**。

**Effectful Fusion/Retry（PRO-06、OPT-07、#2542/#1174）**：二つのTransform Stageを融合しても、Transfer/Import/Business CommitのReceiptを一つのResultへ縮退しない。ACK-lossやCommit Unknown後に「Fusion成功/同Result digest」を再送可能性として扱わない。純粋な変換だけの成功を外部Effectの実行資格と区別する。

**正例の要件**：同一固定Snapshotの純粋なProjection/Filter等で、NULL・欠損・Order・診断・Scope・Output encodingが保存される条件を事前に宣言した場合だけ候補書換えを許容。実際に書換えた二実装の結果・診断・選択Provider/実入力を照合し、適合しなければ通常経路を維持する。新Case IDや独立Expectedの第二正本は増やさず、既存OPT/DO/MC/Nケースの一因子変形として扱う。

### 19.4 Optimizerを新設せず可視化・検証する最小手順

1. **P0 計画票**：既存のW1/W2固定TSVのFlowを「変更前Semantic Graph」として固定し、候補書換え・変更後Selected Graph・適用条件・Reject reasonをread-onlyで記述。No Rewriteを既定とし、現行実装に自動書換えは追加しない。#2622/#951/#3279の型/Provider適格性を使用。
2. **P0 反証**：上記Outer JoinのA/B、Filter/Validateの不正行、Cacheの別Scopeで、成功行だけでなくReject/Unknown/Disclosureも含む独立Expectedを作成。既存#2514と#2593のFixture Ownerが受理するまで設計候補に留める。
3. **P1 一つの合法な書換え**：W1〜W5の初回純粋Go/CLI縦断が成立した後、最小のPure操作についてBaseline/選択候補を同一pin・同じTarget/Expectedで比較。selected providerとactual invoked/read rangeを確認し、意味PASSと速度改善PASSを独立記録（#2600/#2018）。
4. **P1 Studio/Headless**：Semantic → Selected → ObservedのEdge対応・Coverage/Unknown/Source位置をRead-only投影。図に表示された処理と実ObservabilityのClosureが不足する場合、「図中に無いものは存在しない」を主張しない。Source変更やGrant失効後は旧Plan表示と現在実行資格を分離。
5. **P2 Provider別Target適合**：実DB pushdown、Fusion、Cache、部分集計、差分適用は、それぞれのProvider Contract・Snapshot・Target/Toolchain・実Policyが確認できた範囲だけ増やす。常駐Runtime/第二Graph Storeや共通Cost-based Optimizerを新設しない。

**採用しないもの**：最適化候補の自動生成をもって安全な実行許可とすること、Value digestだけを意味・権限・効果の証明とすること、Trace未発見から業務全域の不存在を主張すること、性能改善により失敗・権限・未受理条件を上書きすること。

**証拠状態**：既存設計とSQL/Changelog文献の比較・反例の静的推論に限る。書換えコード、実DB/Provider、実SQL plan、実Go/CLI/Wails、二Consumer/OS権限、独立Expected実試験、性能測定、Owner意味受理はすべて **NOT RUN / NOT ACCEPTED**。

## 20. 複数Sourceの整合Cut・用途別Graph・実装責務（2026-11-12、設計候補）

**この節の固有論点**：§19がRewriteの意味保存・失敗・権限・実行前後の対応を扱うのに対し、§20は **複数Sourceの観測時点を業務的な整合時点と区別すること** と、同一標準構造の **Data/Control/Effect/Observationという用途別の辺** を整理する。先行のRewrite一覧、Cache/Retryの拒否条件は§19.3および既存PRO/OPT/DO/MCへ委譲し、第二のOptimizer/Source正本を設けない。

### 20.1 追加の順序反例：Filter / Sort / Page / Limit

| 一因子変更 | 独立Expected | 偽成功として拒否すること |
| --- | --- | --- |
| 入力の先頭がstatus=rejected、次行がaccepted。Filter後にLimit 1を取る場合と、Limit 1後にFilterする場合を比較 | 前者はacceptedが1行、後者は0行。両者は同値ではない | FilterとLimitを一般に交換可能と判定。Readerがpartial/順序不定なのに同じ上位1件の取得を宣言 |
| 同点Sort Keyを持つ2行を異なるページで読取り、ページ間にSource変更を挟む | Stable tie-breakとPinned Snapshot/全Partition Coverageが欠ければ全ページを一つの順序付き集合とみなさない | 安定ソートなしのPaginationを全件完了・決定論的出力の保証へ昇格 |

正負例は#1130 N-02、#2622 DO-01/07、#2593の既存Expectedへ束縛する。複数Providerで同じRow集合を得てもOrder/Diagnostic/Scopeが異なればTarget側の等価性が未資格となり得る。実QueryのPushdown・Pagination試験はNOT RUN。

### 20.2 複数SourceのSnapshot Vectorと業務上の整合時点を分離する

§18の`Physical Read Complete`をSource AとBで別々に満たしても、A/Bの行集合に**単一の業務的な整合時点**が存在することは証明されない。例えば注文Aが時刻t2のSnapshotに新しい顧客C2を含み、マスタBがt1のSnapshotでC2作成前なら、両ReaderともEOF/byte digest/件数が正しくても、Joinで得た`C2なし`は「C2が業務上存在しない」の証拠ではない。

- **Vector-pinned data**：`orders@r2 + customers@r1`のように各Sourceの版・Scope・取得時点・実Consumedを明示することはできる。これは「この版の組の下での演算結果」の資格であり、同時点性の証明ではない。新しい`SnapshotVector` Core型や分散Snapshot Engineを先に要求せず、既存Source/Lineageと関係へ投影する。
- **Coherent business cut**：共通Tx/Snapshot、Providerに認められた有効時点契約、業務Ownerが受理した許容時差/As-of規則など、問いに応じた**実効的な横断整合の根拠**が別途必要。Source個別のAs-of表記やWall-clock読取時刻の近さだけで自動昇格しない。
- **欠落の意味**：横断整合が未確定ならJoin不一致を`unresolved/not-observed-at-this-cut`とし、実在する顧客の不存在・削除・欠損0件と決めない。受入対象が「二Sourceの観測版Vectorから計算した分析値」に限定されるなら、その狭いClaimを示せる。
- **時点変更後**：同じ業務Periodでも新Vectorでの再集計は§18.2のRestatement/新Generation。旧結果を改ざんしない。Output値が偶然一致してもSource依存・現Consumer資格の再評価が必要。

### 20.3 図表の完全性：Data/Control/Effect/Observationを別Relationで接続

同じ標準構造のNode/Port/Binding/Relationを使い、次の**用途相対Projection**を重ねる。第二のGraph/Flow/Source正本を作らない。

| Projection | 表示・検査する辺 | 描かれていても保証しないもの |
| --- | --- | --- |
| Data dependency | Dataset→Transform→Output、必要Inputと実Consumed、Grain/Scope/Snapshot | 次Stepの起動、全データ取得、外部Sourceの不存在 |
| Trigger/Control | Entry・Schedule/Event/手動Trigger→Invocation、順序/条件/Blocked | 実Scheduler実行、Current Grant、Commit成功 |
| Effect/Subsystem | Function→意図したCUD/Effect→Owner/Provider・委譲先、Commit/Receipt境界 | 宣言が実Writer/実Commitだったこと、越境CUDが0であること |
| Observation/Lineage | Source/Plan/Build/Run/Attempt/Result/Manifest/Reader証拠、ObservedAt/未被覆/開示 | ある図表の表示完了から、見えていない外部処理・業務受理まで証明すること |

[#3250](https://github.com/hat47x/tei-platform/issues/3250)の論理Subsystem更新凝集は**ETLが横断Readを実行できること**と両立する。しかしETL Jobが別SubsystemのTableへ直接C/U/Dを書き込む場合、Dataflowの一本の矢印で境界を隠さず、対象Ownerの型付きAction/Providerへの委譲か、直接越境CUDとその理由・Transaction/Recovery/Authorityの未解決を明示する。論理Subsystemは物理DB/Processの配置と同義ではない。Export→Transfer→Import等へ切り替えても、Effectが消滅したとは主張しない（[#2542](https://github.com/hat47x/tei-platform/issues/2542)、[#1174](https://github.com/hat47x/tei-platform/issues/1174)）。

**図中になければ無い**と判断できるのは、採用したQuestion/Scope/Accepted Revisionに対して**必要Source・Operator・外部境界・実Consumed・実Effect観測の閉包が証明され、非開示・未読込・不透明なProviderが残らない範囲に限る**。そうでなければ外部/unknownの境界Nodeを残す。Studio[#1124](https://github.com/hat47x/tei-platform/issues/1124)/[Draft #4510](https://github.com/hat47x/tei-platform/pull/4510)のread-only ProjectionとShared Acceptanceを使い、図を直接第二Canonicalにしない。

### 20.4 最小の検証・実装順（既存Expected/Issueを再利用、すべてNOT RUN）

1. **P0/#4410 W1/W2と#2593/#2622**：すでにあるN-01/02/03/12・MC-01/02/05・DO-07から、§19.3の合法/不合法Rewriteを**同一の独立Expected**で比較。最初の固定TSVでは`Filter→Convert→Aggregate`を固定したままとし、新Optimizerや実DB PushdownをW1のHard Gateにしない。
2. **P1/#2622/#3279/#2514**：同じ受理済みMeaningの二実現（直接Go/SQLなど）でSource Vector、選択/実ロードProvider、使用した演算、実Output/NULL/失敗、Target・Lawを別列に取り、実Conformanceと型宣言だけのFitを分離。失われた前提を`unverified/unsupported`に落とす。
3. **P1/#2831/#2600**：Planの描画上の辺と、実行したCommand/Actual Consumed Input/暗黙のrate表・Query/Cache/出力Memberを突合する。実際に実行されなかったNodeや未宣言Pushdown/外部Sourceを負例化。結果digest一致だけでPlan閉包やセルフビルド資格に昇格しない。
4. **P1/#3250/#1132/#1174**：横断集計の**Read依存**と、他Subsystemへの**direct/effective CUD**を別々に投影。直接WriterをAdapter呼出しへ変えた場合も、Effect/Owner/Receiptが消えていないことを観測票で確認する。新分散Tx・Grant・Effect Runnerを作らない。
5. **P2/#2542/#4194/#4502/#4510**：Event/External Worker/Relay/Studioは既存起動・Delivery・Observationの契約へ差し込む。外部Task Success・ACK・Manifest・画面の緑表示のいずれもBusiness Acceptedへ自動昇格させない。

**外部調査との位置づけ**：Apache BeamのWindow/Trigger/Pane/late dataの区別（https://beam.apache.org/documentation/programming-guide/）と、OpenLineageのJob/Run/Input/Output facetの区別（https://openlineage.io/docs/spec/facets/）は、上の責務分離の比較対象である。BeamやOpenLineageのRuntime/SchemaをTEIへ導入する判断ではない。新しいGo/SQL/二Provider・実DB/Stage・Studio・Business Owner受理の試験は**NOT RUN**。

## 21. 部分入力でも保持できるClaim：Witness・Bound・Finalの資格（2026-11-12、設計候補）

**問題設定**：§18のPopulation Completeが不明でも、一部の事実は限定的に正しく言える。ただし「partial resultを業務の最終集計としてPublishしてよい」こととは別。§19の演算子Rewrite資格、§20の複数Source Business Cut・用途別Graphと合わせ、[Conformance #2514](https://github.com/hat47x/tei-platform/issues/2514)、[Measure #2593](https://github.com/hat47x/tei-platform/issues/2593)、[Data Operations #2622](https://github.com/hat47x/tei-platform/issues/2622)、[Trait #3279](https://github.com/hat47x/tei-platform/issues/3279)、[Cross-layer Draft #4522](https://github.com/hat47x/tei-platform/pull/4522)へ受入条件として返す。正式なClaim enum/証明Engine/特別なPartial Publisherを新設する提案ではない。

### 21.1 同じ固定Snapshotに対する部分集合と、別Revisionの変化を混ぜない

対象Fact集合を D、実検証済みの部分集合を S とする。以下の限定推論は **Sが同じ固定・必要なSnapshot Dの真正な部分集合であり、Fact identity・重複・Null/単位/Scope・対象業務Cutが保全**された場合に限る。Sの行がSourceに存在したことと、未取得D＼Sについて何が存在するかは独立である。

**非単調なChangeと区別**：新Revisionで更新・削除・Rekey・Correctionがある場合はS⊆Dという拡張条件を失う。同一内容へ戻るSource A→B→A、Cross-source不整合、Providerのopaque Filter、後着Recordの削除、Current Grant失効でも旧Witness/Boundの現在適格性は自動保持されない。Incremental RunのDelta/Restatementは§18.2、実変更履歴は#2622/#2600へ返す。

| 提示可能なClaim | 固定Snapshot内の証明条件 | 完全性が無いと禁止するClaim |
| --- | --- | --- |
| **Verified Witness（存在する一件）** | そのFactの出典、入力Scope・型・Fact identityと現在の閲覧資格が検証済み。正の存在判定・条件を満たす実例の観測 | 観測していない対象が存在しない、期間全件が成立 |
| **One-sided Bound（下界・上界）** | Measure/演算が追加Factに対して単調であること、値域・重複規約・計算の精度が有資格。例：非負ContributionのSUMは下界、重複なしCOUNTは下界、非空集合のMAXは下界・MINは上界 | 真の総額・最終COUNT/順位・一般的なSUMの下界 |
| **Complete / Final Result** | 必要Source/Partition/Temporal Cutの閉包・全件被覆、選択演算の独立Expected、出力・公開Generationと必要なConsumer readback。業務Effect/Owner受理はさらに別 | EOF・Watermark・Partial PaneのみでBusiness Acceptedへ昇格 |

**数式上の候補**：入力拡張 S⊆D が資格化された場合、肯定的な存在証人の保持、重複を正しく扱ったCOUNTの下界、値が非負のSUMの下界は成り立ち得る。一方、受理されたFactに負の金額・返品・取消が混ざるとSUM(S)≤SUM(D)とは限らない。負値のない集合でも平均、中央値、Top-N、NOT EXISTS/anti-join、全件Validation PASSを部分観測から確定できない。外部JOINの有効時点・参照Sourceが未整合なら、陽性Witnessさえ「同じ業務Cutで真」とは言えず、観測Vectorに相対化する（§20.2）。

### 21.2 「単調」でも業務的な最終性・公開権限は得られない

- **演算単調性**と**事実の時間的安定性**は独立。固定Snapshotの追加観測に対して保持される陽性Witnessであっても、後の訂正RevisionでそのFactが削除されれば新RevisionのTruthとは限らない。Mergeable、Retractable、Monotone-under-extension、Population Completeを一つのTraitに畳まない。
- **局所的な導出は全体集計を証明しない**。例えば顧客Aの100円明細を1件観測したなら「Aにその明細が存在した」というClaimは可能だが、Aの請求額100円・未取得取引なし・会計Period終了は不可。厳密に非負でその他の条件を満たした場合だけ「同じ対象集合・同じSnapshotにおける合計が少なくとも100円」というBound候補を示せる。
- **存在の表示自体が情報漏洩になり得る**。Witness/Bound/unknownの区別、件数・パス・Source identityを公開Consumerへ出す前に#2516のCurrent Disclosureを評価する。拒否・未取得・秘匿はそれぞれ意味が違うが、低権限側には秘匿要素の存在を推測できる情報を返さない。
- **部分公開は明示的なConsumer契約が必要**。Early/Partial/Aggregate candidateを出せる計算能力と、そのConsumerがその不完全結果を受け取る権限・用途は別。W3の最初の固定TSV Batchは従来どおり未完了を通常Final成果物として公開しない。必要なら将来のアナリティクスConsumerだけに、必要範囲・as-of・Qualifierを伴った限定出力候補を検討し、#2604の一式公開/Reader資格や#267 Owner受理を飛び越えない。
- **WatermarkはProviderが定める進捗意味**。Window end通過・Pane emitから実Source/外部マスタ・業務CUDまで完全閉包したとは推測しない。参考：Apache Beamの[Watermarks and late data](https://beam.apache.org/documentation/programming-guide/)では遅延要素とearly/late paneを区別している。TEIにBeamのWindow/Trigger Runtimeを必須導入する趣旨ではない。

### 21.3 既存N/MC/DOへの反証ケース（独立Expectedの候補、未実行）

同じ顧客A・同一JPY単位・Pinned Snapshot R1を固定し、実入力S={o1: accepted 1個×100円}、R1内の未取得候補を分けて比較する。下記は生成プログラムの出力をOracleに使わない手書きExpectedであり、現行製品実装の合格ではない。

| 一因子変更 | 限定的な正しいClaim | 拒否すべき誤Claim |
| --- | --- | --- |
| Sを真正に観測、残りPartitionは未取得 | R1相対のo1存在はWitness。非負金額・重複なしの保証があれば金額100円は下界候補 | R1の総額100円、他顧客が存在しない、Final |
| 未取得Partitionに+40円の同顧客Fact | 観測Sの金額下界100円、全件を読んだ場合の金額140円 | 部分入力で総額100円と確定 |
| 未取得Partitionに返品−200円のFact | R1の全件金額は−100円。Sの100円は下界ではない | SUMというOperator名だけで下界を自動付与 |
| 未取得顧客マスタに対応Cが存在 | ordersにあるCの観測は可能。anti-joinでのC未登録は未資格 | partial masterにC行が無い→Cは存在しない |
| R2でo1を訂正削除する | 旧R1のWitnessは歴史的記録として保持。R2の新Claimは再資格化 | R1の陽性Witness・GrantをR2の現行真実へ流用 |
| Sの実観測と値は同じだが閲覧Grantを失効 | 歴史的Evidenceは残せる。現在の開示は拒否/保留 | Value digest一致からCurrent Disclosureを復活 |
| 1行目acceptedの到着後にEarly Triggerが発火 | 当該Snapshotの部分Paneという説明は可能（Providerが保証する範囲内） | Trigger発火・Watermark通過だけでFinal/Business Accepted |

関係代数の追加入力に対する単調性は、選択・射影・正のJOIN/UNIONと、否定/差集合で異なる（例：[monotone query研究レビュー](https://pmc.ncbi.nlm.nih.gov/articles/PMC10328546/)）。この数学的区別を、Providerの実保証、Current Authority、業務的な時間Cutへそのまま昇格しない。

### 21.4 受入順・既存Ownerと停止線

1. **P0 / #4410 W1/W2**：現行TSVのpartialをFinalとして出さない規則を維持。独立Expectedに、真の空・未取得・真正な陽性Witness・非負Contribution/負Contribution・マスタ欠落の説明を追加する候補。追加Fixture/Case番号はOwnerが既存N-02/03/11/12、MC/DOのいずれに収めるか判断。
2. **P1 / #2593/#2622/#3279/#2514**：Measure×Operation×Axes×Temporal Cut×Providerの条件付き法則と実Conformanceを分離。手書きOracle→限定Go/Provider独立対照の順で検証し、数学的性質のみから全Consumerへ保証を広げない。
3. **P1 / #2604/#2516/#1130**：同じCandidate結果がData Readで可視でも、Final Artifact Publication/Consumer use/業務Effectへ昇格しない。部分結果を公開する要件がない現在のP0では新Partial Publisherを作らない。
4. **P1〜P2 / #1124/#4510/#4487**：Studio/TEAの問い相対Graphで、限定Witness・Bound・Unknown・完全結果を可視化。隠れたWriter、未読Source、Observed cut不整合を偽の不存在・Writerゼロへ縮約しない。新Graph正本/専用Claim Engineなし。
5. **P2 / #2542/#4502/#4194**：実外部Stream/Relay/EKIのWatermark・ACK・Worker Resultと、Provider Dataset/Business OwnerのFinal資格を分離して観測する。締め・補正の実承認は従来Ownerへ戻す。

**停止線と検証範囲**：この§21は既存設計・独立小規模数学反例と公開研究に基づく設計案。Syntheticな非負SUMと返品の対照はローカルPython上で集合追加ケースを列挙して確認したが、TEI Go/CLI/実DB/EKI/MFT/Studio・独立Consumer/実Disclosure/Owner受理・exact-head CIは **NOT RUN / NOT ACCEPTED**。新Schema、演算Engine、Scheduler、Host、Validator/Claim Storeや第二のOracleを追加しない。

## 22. Change Streamと再計算の受入：Snapshot接合・撤回可能性・回復（2026-11-12、設計候補）

**位置づけ**：§21は**固定Snapshotの部分入力**でも保持できるWitness/Bound/Finalを扱う。これに対して本節は**異Revision間の追加・更新・削除イベントを使った再計算**が適格かを扱う。§18.2のSame-Run Resume / New Delta Run / Restatementと§19の意味保存を前提とし、ETL専用CDC Engine、第二Checkpoint/Scenario/Manifest/State Store、Scheduler、TEI必須Hostを導入しない。#1130のW7・[#4411](https://github.com/hat47x/tei-platform/issues/4411)の受入Pressureとして実装Ownerへ渡す。

### 22.1 「変更が届いた」と「変更集合が完全」を区別する

| 変更・配送形式 | 必要な追加保証 | 無条件に主張しないこと |
| --- | --- | --- |
| **Append-only** | Sourceの不変Fact規約、Event/Fact identity、重複、対象Partition、取り込んだ位置と完了範囲 | 後から訂正・取消・削除されないことが宣言だけで成立する |
| **Keyed Upsert（after-only）** | 一意で安定したKey、Keyごとの順序/Revision、**同じScope・版の旧Fact State** | UPDATE_AFTERをINSERTとして加算できる。旧寄与を知らずにSUMを訂正できる |
| **Before/After Retract** | 旧行と新行の対応、Update/Deleteの連続性、失われたEvent/二重配送の判別、演算側のretraction適格性 | beforeとafterの片方が欠けても同じ結果になる |
| **Key-only DELETE** | 過去のFact/寄与を復元できる有資格State、または残存集合の完全再読込 | Keyだけから過去の金額・Groupを導出できる |
| **Snapshot→Tail接合** | 固定Snapshot境界、Change log開始位置、並走範囲の重複排除、Gap/Retention、必要なSource間のCut | Snapshot終了後のTail購読があれば途中の変更を全件捕捉した |
| **Watermark/Deadline/Trigger** | Event timeと観測/処理時点、Sourceによる遅延規則、窓、late eventの扱い、業務期/Ownerの締め規則 | Watermark通過・Scheduler成功・Task Completed = Business Final |

Snapshotのp1/p2読取が成功しても、Tailの途中Gapで訂正イベントを失えばNew Delta Runは`unqualified`である。未確認の欠落を「変更0件」にせず、完全に固定したSnapshotを再読取できる場合のみ**別Run・新GenerationのFull Restatement**へ移る。Full Snapshotすら不完全なら`unknown/unsupported`。ProviderのSnapshot/CDC形式と実保持期限は#2622/#2542が適合確認を担当する。

### 22.2 演算の結合法則と撤回法則は別

- **Mergeable**：固定Cutで重複・欠落なく分割した部分状態を結合できる。SUMの合成、AVGの`(sum,count)`はこの例である。これだけではDelete・Correctionの安全な適用は証明できない（#2593）。
- **Retractable**：更新/削除されるFactの**旧寄与**を有資格に取得し、新寄与との差を演算規則に従って反映できる。SUMでも旧Contribution不明なら差分訂正は不可。DISTINCTは必要なID/出現回数、MIN/MAXは削除後の残存候補を要する。
- **Impact complete**：実読行に加え`NOT EXISTS`等の否定依存、Filter/Top-Nの集合依存、FX rate/Lookup/Calendar/Rule版・Scope/Temporal Cut変更を含む、安全な影響**上界**が確認できる。`Consumed 0 rows`を依存0としない（#2600 IB-07/09/10）。
- **Replay-safe**：値の再計算と、DB Write/Import/Message等のEffect再送は別資格。Operation ID・Current Grant・Commit/Receiptを使い、Commit Unknownを再送可能として推論しない（#1174/#2499）。

### 22.3 固定反例（既存§18.4 R1〜R3を拡張、Expectedは手計算）

業務期D、同一JPY/数量単位、R1はAのo1=2×100円、o2=3×200円、Bのo4=1×50円で全件固定。R2でAのo5=1×40円を追加。R3でo2を3個→2個へ訂正。R4でo1を削除する。

| 対照入力 | Aの独立Expected（数量/金額） | 誤成功として検出すること |
| --- | --- | --- |
| R1 Full Snapshot | **5 / 800円** | p2未取得なのに全件Final |
| R2 Append o5 | **6 / 840円** | 同Event IDの再配送を二重加算 |
| R3 Correct o2（旧3×200→新2×200） | **5 / 640円** | after-only新400円を追加し旧600円を残す |
| R4 Delete o1 | **3 / 440円** | key-only Deleteなのに旧寄与未取得で差引成功 |
| R4以降、o5の追加Eventを同一Event IDで再配送 | **3 / 440円**（そのSourceが同一Eventとして識別できる場合） | 同Byteの別業務Eventまで誤dedup |
| Rate表だけ改版、注文Factは不変 | 新Rate・適用日未指定のため金額Expectedは**未定**。必要Sourceを再評価 | 注文byte不変⇒出力不変 |
| R2→R3の訂正Event欠落 | Deltaは`unknown`。完全な新SnapshotならRestatement候補 | 不完全TailからR3の全件Finalを推定 |

独立Expectedは実装出力から生成しない。R1〜R4の算術が一致しても、Change log連続性、Source/Scope/Business Cut、公開Generation、Consumerの実読取、Owner受理を**別列**で確認する。新しいCase ID正本を作らず、§13 N-02/03/06/08/11/12、#2593 MC-08、#2600 IB-09、#2514へ一因子変形として返す。

### 22.4 CheckpointとPublicationの非原子境界

**記録したoffset＝Effectなし／Published済み**ではない。純粋計算でも「新Generationを公開した直後にRunLogのCheckpoint保存に失敗」と「Checkpoint保存後、公開前にCrash」は別の故障注入である。再起動時にCheckpointの進捗だけを信用せず、Run/Attempt・Source Cut・物理Member/Generation・実Publisher状態を独立Readbackし、古いWriterのFence/実Provider Commitは対応Ownerが確認する§9/#2604/#1174）。同じByteの再生成が安全でも、外部Recipientへの再送は別。明示的な一式公開・独立Reader保証がない場合`unknown`を維持し、第二のPublisher/Transaction Coordinatorは先行導入しない。

### 22.5 既存計画への最小追加

1. **P0 / #4410 W1/W2**：最初のPure TSV実行を優先し、上表の算術Expectedを既存Nに関連付ける。CDC/Stream/追加DSLは初回受入のHard Gateにしない。
2. **P1 / #4411 W7 + #2622/#2593/#3279**：Source Change Form×Operator Law×旧Stateの有資格性をread-onlyで比較。Snapshot→Tail Gap、before欠落、key-only Deleteを`unsupported/unknown`とする。実Target Fulfillmentは別。
3. **P1 / #2604/#2514/#1174**：一つの故障注入でPublished Generation、Checkpoint、Consumer Read、Effect/Receiptを別観測。純粋変換のRecoveryから業務Effect再試行の安全性を一般化しない。
4. **P2 / #2600/#2831/#2582/#4510**：正しい局所再計算と過小・過大な再評価、Negative Dependency、旧Planの失効と現行Disclosureを複合GraphへProjection。図外の実消費・非開示を不存在扱いしない。

**外部比較（採用決定ではない）**：Apache FlinkはAppend/Upsert/Retractと`UPDATE_BEFORE`/`UPDATE_AFTER`/`DELETE`を区別する（https://nightlies.apache.org/flink/flink-docs-stable/docs/dev/table/functions/ptfs/）。Debezium PostgreSQLは初期Snapshotと継続的なChange Streamを接続する（https://debezium.io/documentation/reference/connectors/postgresql.html）。BeamのWatermark/Allowed Lateness/Triggerは遅着データ処理の比較対象（https://beam.apache.org/documentation/programming-guide/）。TEI固有のCDC/Temporal Engineを採用する根拠にはしない。

**証拠と停止線**：本文は設計・独立算術Expected・外部仕様の静的比較。Go/CLI/実DB・CDC・Snapshot→Tail・Crash/復旧・実Consumer・実Provider Receipt・Owner意味受理は **NOT RUN / NOT ACCEPTED**。新Core型、第二DSL/Verifier/Artifact Store、必須Hostを追加しない。

## 23. 再集計Generationの受領とConsumer使用資格：公開・読取・適用を分ける（2026-11-13、設計候補）

**追加するGap**：§18のRestatement、§21のWitness/Bound/Final、§22のChange Stream・Checkpoint/Publicationの非原子境界に対して、**下流Consumerがどの世代を実際に読取り、採用・適用・業務受理したか**は独立に未確認となり得る。これは [#2604](https://github.com/hat47x/tei-platform/issues/2604) の一式公開、[#2516](https://github.com/hat47x/tei-platform/issues/2516) の用途別Disclosure、[#1174](https://github.com/hat47x/tei-platform/issues/1174) のEffect/Receipt、[Cross-layer Draft #4522](https://github.com/hat47x/tei-platform/pull/4522) のClaim再資格化の **ETL Reader/Consumerへの適用**。別のRelease Store、Consumer State Engine、Publisher、Scheduler、Host、Claim Registryを設けない。

### 23.1 Publicationの成功とConsumer適用の間にある独立境界

| 観測できた段階（説明用） | 確認する対象 | そこから主張してはいけないこと |
| --- | --- | --- |
| **Produced / Staged** | Function/Source Cut/Run/Attemptに結び付く出力byte、Member候補とdigest | 一式公開済み、外部到達 |
| **Verified Member Set** | 必要Memberの列挙・内容/digest・役割/版・欠落/旧残存、独立Expected | Current Releaseが切り替わった、任意Readerの受領 |
| **Published Generation** | Publisherが有資格に公開し、必要ならActive Refを望む旧世代から条件付き切替した証拠 | 全Consumerが同Generationを読んだ、業務へ適用した |
| **Consumer Read Qualified** | Readerが選択したGenerationと同版の必要Member/byteを**実際に**読んで検証、Read Scope/ObservedAt/現在Disclosure | Consumerが取込み・Effect Commitまで完了 |
| **Applied / Imported Effect** | 対象Consumerが実行したImport Unit/Operation ID/実Provider Commitまたは結果不明の照会 | その結果を業務Ownerが意味的に受理した |
| **Business Accepted** | 業務主体が正しい入力Cut・意味Revision・期間・結果/訂正の適用を判断した記録 | 外部キャッシュ/CDN全域の失効や過去Byteの物理消去 |

表は新しい永続状態列挙ではなく、既存Artifact/Run/Effect/Consumerの**Evidence比較票**。不十分な階層では、その対象のClaimをunknown/partial/not-observedに止める。現在Grantは各読取・適用の有資格性として再評価し、過去のRead/Commit Evidenceを物理削除したと扱わない。

### 23.2 Active世代の切替と、読み始めたConsumerのPinned世代

例としてR1入力からG1、訂正後R3入力からG2を生成する。**PublisherがG2を公開した時点**と、**各ConsumerがG1/G2を読み終えた時点**は異なる。

- **Producerの安全候補（Providerの実保証が前提）**：G2の必要Member全体をStage→byte/digest/独立Expectedで確認→Generation単位の参照を公開→必要な場合だけexpected active=G1からG2への条件付き切替を試みる。条件付き切替、原子的更新、世代別の不変Body、Read Lease/保持期間がTargetで実現できるかは別。単一File rename・Catalog pointer変更・Manifest Completeを複数Memberや複数Providerでのatomicityの証明としない（#2604）。
- **Consumerの安全候補**：最初に選択したG1/G2を固定し、そのGenerationの必要Memberだけを取得・Digest照合する。途中でActive RefがG1→G2へ動いても、一つの結果にG1 summaryとG2 manifestを混ぜない。Pinned G1がまだ読めればG1相対の**歴史的結果**を取得できるが、「現行最新版」とは言えない。G1のMemberが既に失効・期限切れなら欠落を**明示的に停止**し、現在Refで別Read Attemptを始めるかOwner指定のRecoveryに戻る。
- **切替失敗・競合**：同じexpected active=G1で異なるG2/G3が並走したら、Last Writer Winsで業務意味の新旧順序を推測しない。CAS相当がないProviderでは採用できない保障をunsupported/unknownに落とし、単一物理Object/対象Hostが提供する範囲だけ条件付き運用する。
- **ReadとApplyの間**：ReaderがG2の全Memberを見ても、下流DBへのImportがfail/partial/commit-unknownならConsumed/Appliedを一つのPASSにしない。業務締め・訂正の適用には、受理した業務期・Owner・Operation/Receipt/最新Grantとの対応が必要。ACK-lossだけで再送可と推論しない（#1174/#4411）。
- **Retention/使用資格**：過去G1の証拠保持・GC可能性・Active ConsumerのPin期間は別の判断。G1 Source/ArtifactをGCした後も歴史的Evidence記録は残り得るが、再ReadやReplayを保証しない。Source Change Logの保持期限切れは§22のDeltaに、旧公開物の残存/撤去と低権限Consumerの情報漏洩は#2604/#2516に返す。Current Grantを失ったConsumerには、たとえG1が保存されていても現行Read資格を自動復活させない。

### 23.3 独立Expectedの例（既存N/AP/EFへ一因子で追加、正式Fixtureではない）

| 故障・変化注入 | 判定対象 | 手書きExpected / 禁止する偽成功 |
| --- | --- | --- |
| R1→G1でA=5個/800円、R3→G2でA=5個/640円（§18.4、§22.3） | Generationsと業務Cut | それぞれ当該Source/Rule版への出力。双方の数量5でも金額・Source Cut/Generationが異なる |
| Consumer C1がG1をPinし、summary取得後にActiveをG2へ切替 | 一式Readerの版混在 | C1は保持されたG1 manifest・全Memberを読むか、欠落で停止。G2 memberを混ぜてPASSしない |
| C2がG2全Memberを読んだがImportはタイムアウト、Commit不明 | Read vs Applied/Receipt | Consumer Readは限定PASS、Effectはunknown。Owner Acceptedは未観測 |
| G2 Published後もC1がG1を読んでいる | Consumerごとの進捗/鮮度 | C1のG1歴史的ReadとG2最新版のPublishedは同時に真。G1 ReadからC1がG2を消費済みとは言えない |
| G1の必要MemberがRetentionで消え、C1がG1 Pinのまま取得を続ける | Retention/Release lifetime | Read拒否・停止または明示的新Attempt。G2の同名Memberへ無断フォールバックしない |
| G2出力byteがG1と偶然同じで、Run/Disclosure/Source Cutだけ改版 | Content reuse vs Usage | 計算Byteの再利用候補と、Generation/Reader/Current Grant・Owner受理の再資格化を分離 |
| 公開範囲のGrant失効後、古い外部Reader/CDNでG1を取得 | 秘密・撤去の保証 | Producerの「非公開設定」だけで実外部Readが拒否済みと宣言しない。実Target/Disclosure証拠なしならunknown、公開範囲の差を低権限へ漏らさない |

**小さな独立計算**：2世代×2回のMember参照でActiveが変化し得る4通りを列挙すると、参照を毎Member更新する非Pin方式では2通りで世代混合、最初の世代にPinすれば0通りとなる（**不変Memberが保持され、各Readが成功する場合の合成モデル**）。これは実Publisherの原子性、TargetのRead Consistency、実Consumerの資格を試験した結果ではない。

### 23.4 実装へ渡す最小接続と優先順

1. **P0 / #4410 W1〜W5**：純粋TSV→決定的Output BodyとRun/Attempt Evidenceの区別を優先。初回の正常経路にRelease Switch/外部Consumer・部分結果配信を新規導入しない。G1/G2のExpectedは将来の独立受入票として固定候補に留める。
2. **P1 / #2604 AP-01/06, #2514**：既存Artifact/Manifestとread-only Consumerを使い、G1→G2のActive切替前後で、同一Pinned GenerationのMember集合・byte・missingを比較する。Providerに世代固定/保持保証がなければ未適格として扱い、第二Publisherを実装しない。
3. **P1 / #4411 W6〜W7, #1174**：同じInputCut/Rule/OutputでRun/Attempt別、Commit Unknown、Consumer Read/Applyの独立Eventと照会資格を対照。再試行可否をData digest/Task ACKから決めない。
4. **P1 / #2516/#2600/#4522**：公開Scope/Grantだけ変化した場合、旧byteの計算利用可否と新Reader使用資格/外部撤去の独立判定、正しい部分再資格化/不要な全失効を検証。別のClaim CacheやPolicy発行者を作らない。
5. **P2 / #1124/#4510/#4487**：Studio/TEAへSource Cut→Generation→Reader/Effect/OwnerのEvidence Edgeを問い相対に投影。「現在表示」の意味を、観測した具体的Consumer/世代/時刻に限定。非開示Source/古いView/旧Pinを現在Edit/Run適格性と混同しない。

**外部仕様との比較（採用判断ではない）**：[Apache Iceberg仕様](https://iceberg.apache.org/spec/)はReader Snapshot隔離・Snapshot Reference/Retentionを区別する。[Delta Lake Change Data Feed](https://docs.delta.io/delta-change-data-feed/)はChange Logの保持期限/VACUUMと前提Versionを扱う。[OpenLineage](https://openlineage.io/docs/spec/facets/)はJob/Run/Input/Outputの追跡Facetを分ける。これらは検討対象であり、TEIにIceberg/Delta/Lineage Runtimeや新Schemaを必須採用する理由ではない。

**検証上限**：Issue/PR/設計文書と公開仕様の静的照合、及び上記2世代のローカル合成組合せ計算。実TEI Go/CLI/Publisher/DB/Consumer/Host/CDN/EKI/Studio、現行Grant/Owner Meaning Acceptance、exact-head CIは **NOT RUN / NOT ACCEPTED**。新Runtime/Release Engine/Publisher/Manifest/DB Transaction Coordinatorなし。

## 24. 依存Graph改版によるETL再計算：削除辺・否定依存・局所再資格化（2026-11-13、設計候補）

**今回の追加論点**：§18〜§23はInput/Change/計算/公開の正当性を分離した。しかし**計算で参照するData/Rule/Source/Providerの関係自体が増減・再束縛された場合**、古いGraphから新しいGraphへどのOutput・Evidenceを保持してよいかは別の問いである。#2600のDI-01〜08（依存辺差分・正しいskip/過剰失効）、#2582のRC-01〜09（Data Contextの反応的再評価）、#2622のData Operations、#2514の独立Expectedを、ETL/BatchのConsumerへ**最小限に適用**する。新しいDependency Engine/Graph正本/Optimizer/Cache/Claim Registryを採用しない。

### 24.1 「変化した事実」と「変化した依存構造」を別々に比較

| 判定面（既存Source・Projectionを再利用） | Before/Afterで確認すること | これだけでは不足すること |
| --- | --- | --- |
| **Required / Accepted** | Accepted Meaning、Function Data Context、Predicate/Join/Group/Scope/Temporal Cut、Required Source・否定/集合依存、Role/版 | 宣言しただけではProviderがその範囲を実Readした証拠はない |
| **Selected / Lowered** | 解決したQuery/Operator/Plugin Instance/Target、実際のSQL/外部Binding・Source選択、Toolchain/Artifact | 正しい選択でも実ロード/実Read/実Outputと一致するとは限らない |
| **Consumed / Observed** | 実際のSource/Range/Member/byte、Returned rows/NULL、Read Attempt/Result/拒否、Coverage/未ロードの理由 | 0行から対象Collectionが空とは言えず、未読の負の依存も消えない |
| **Reusable Result** | 正しい計算結果の候補、Semantic版・Output bytes/Generation・現在必要なReader条件 | 計算Byte一致だけでは古い独立Expected・Owner受理・現在Grant/Disclosureは再利用できない |
| **Impact / No-impact** | 上記の前後Edge/PathとClosureで、Claimごとの影響の**安全な上界**・非影響の理由 | 未発見の辺・opaque Providerを「影響なし」へ丸めてよいわけではない |

**判定単位はConsumer/Claim × Meaning revision × Source Scope/Cut**。同じDataを読む二つのFunctionでも、必要なRule/Source・Group/Output・認可範囲は異なる。差分で変更されたEdgeの単なる前後集合だけでは不十分で、削除されたEdgeから到達していたConsumer、新しく到達するConsumer、別経路の有無と既存実Consumed/未観測範囲を区別する（#2600 DI-01/02）。新Graphが閉じていない場合は「このConsumerでは未解決」と表示し、無関係な他Consumerまで一律に失効させない。

### 24.2 一つの正しい再計算省略と、一つの誤省略を同じ入力で識別

**合成の独立Expected**：同じOrders Snapshot O1を読む二Consumer A/Bを想定する。

| 変更 | A: 数量集計SUM(quantity) | B: JPY換算売上SUM(amount×rate) | 診断・再計算のExpected |
| --- | --- | --- | --- |
| Rate版R1→R2、Orders O1は不変。Required GraphでAはRate非依存、BはRate依存と閉包確認済み | **計算結果の再利用候補** | **再計算/再検証必要** | Aのrate非干渉が説明可能。Bで旧rateの結果を流用したらFalse Reuse |
| AのMeaning v1は売上、v2は数量へ変更し、Rateへの依存辺も削除 | **Meaning変更時点に再評価が必要** | Bは従来通り | Aで旧売上Outputを新数量Outputとして流用するのは誤り。v2の成立後の次回Rate変更ではAは非影響候補 |
| 新規RuleがAのFilter条件に追加。現Snapshotの結果byteが偶然同一 | **Required Closure/意味版の再資格化が必要** | Ruleに無関係なBは候補保持 | Result一致をRule/検証資格の永久再利用へ昇格しない |
| BがNOT EXISTS参照先の新しいDeliveryを観測（旧Readは0行） | AがそのSourceに非依存なら候補保持 | Bの該当集合を撤回/再評価 | 旧Consumedが0件だったことから「Bに依存無し」と言わない |
| Query IDは同じでSQL/WHERE/ORDER/Environment/Scopeだけ変化 | 対象Claimの選択/Scopeを再評価 | 同左、必要なものだけ | Query ID一致・前回Rows一致を実Provider意味適合へ昇格しない |
| Source A→B→A、結果byteは最終的に同一。旧Artifact/Grantは別版 | 計算byte再利用は条件付き候補 | 同左 | 現在のSource/Owner/Expected/Reader/Grant受理は再検証。古いReceiptを復活させない |

「影響あり」は必ず全件再実行を意味しない。局所的な再導出、Stage単位の再評価、Artifact bytesの有資格な再利用、独立Expected/Disclosureだけの再資格化などを別選択肢とし、手段を**意味を保存できる実Targetの能力**に応じて選ぶ。逆に「影響なし」には、対象Claimに必要な閉じた依存範囲と、その変更の非干渉根拠が必要である。単に実Readされた行が同じ・選択Graphに辺が見えないだけでは不足する。

### 24.3 実Sourceへ接続する最初の観測点：#4539の資格上限

[統合済み #4539](https://github.com/hat47x/tei-platform/pull/4539) は、Environmentに束縛した実PostgreSQL Query Read→限定Function Data Context→別ProcessのReference CLI受領を実証している。ただしこれは**製品 tei CLI / Studio / 認可された実Hostの一気通貫ではない**。実Queryの0行についてEmptyJustifiedはfalse、RLSで2/3行になってもCoverageはunestablished、マスクViewでNULLが返る場合はNULLの出自が不明、SQL/Source版変更に応じた全件・不存在/再利用判定は未証明である。

初期接続では、次の三種の入力だけを既存Provider/Function Context比較票へ束縛する。

- **Q1 positive**：固定Environment・SQL・scopeで実取得行とObserved Attemptを記録。限定した陽性Witnessは可能だが、全件数・全Contextを証明しない。
- **Q2 empty/RLS**：0件・2/3件・NULL maskingを各々別の部分/未資格Claimとして表示。見かけ上同じ0件/NULLでも実SQL・Environment・Grantを省略しない。
- **Q3 changed selection**：SQL/WHERE/Scope/Role/Source alias変更後、実選択・実Consumedの再比較が必要。Source/Query IDが同名でも旧Contextの現在性は自動継承しない。

#2586/#2622のProvider/実Query入力が完了する前に、#4410のPure TSV最小縦断を実PG/Studioに置き換えない。#4539のReference Entryを製品Batchが採用済みと偽らず、独立Read/Ownerを求める。

### 24.4 Test/計画の優先順位（既存DI/RC/Nを再利用）

1. **P0 / #4410 W1〜W5**：TSV Filter/Convert/Aggregateの固定Expectedと二入口は維持。Rate依存、Rule改版、余分/欠落Partitionについて、事前に独立ExpectedのAffected/Unaffected一覧を手書きし、実装出力から採点表を生成しない。
2. **P1 / #2600 DI-01/02/03/05・#2582 RC-01/03/04/05**：改版前後のRequired/Selected/Consumed EdgeとScopeを同じClaimへ対応させ、False Reuse、Unnecessary Invalidation、Unknown Erasedを区別。**最小の次の一辺**が未観測ならそのEdgeとOwnerを示し、Graph全体の完全性を捏造しない（[横断設計Draft #4527](https://github.com/hat47x/tei-platform/pull/4527)、[#4522](https://github.com/hat47x/tei-platform/pull/4522)）。
3. **P1 / #2622/#2586/#2514**：#4539のReadを参照起点としてQ1〜Q3のSQL/Row/Context/Environment束縛を独立Expectedで判別し、製品 tei CLI/Studio・Closed-world Scopeは別Gateのまま保持。失敗/未取得/NULL/部分RLSを同列の成功Rowsに変換しない。
4. **P2 / #2600/#2831**：Full clean vs Selectiveの意味・Member Set・実入力・Changed Edgeを同一対象Revisionで比較し、実Latency/再実行数を**正しさの後**に計測。Stage1生成Builder/Stage2実ProcessのSelf-build問題をETL単独の成功で代替しない。
5. **P2 / #2604/#2516/#4510**：§23のReader Pin/公開Generation・Current Disclosureと、Studioの問い相対Data/Control/Effect/Evidence投影を接続。計算Outputを再利用できても、現Consumer/業務Ownerの実使用資格を別照合する。

**禁止する設計の混線**：#2600のBuild Skip正本、#2582の反応的Data Context、#2622のProvider能力、#2514の独立Expected、#1180のScenario、#2604の一式公開、#267の意味受理は各Ownerのまま。新Graph Solver、Source Registry、Cache、Evidence/Claim Engine、CDC Runtime、固定Hostは追加しない。§24は**宣言・選択・実観測の既存Graphを比較するread-only設計候補**であり、実Go/CLI/DB差分、Selective性能測定、Studio/Owner受理は **NOT RUN / NOT ACCEPTED**。

## 25. Generation退役とConsumer到達の閉包：Pin・Retention・GCと訂正受理を分ける（2026-11-13、設計候補）

**§24との責務分離**：§24は**依存Graph/Rule/Source/Providerの改版後、どの結果を再計算・再資格化するか**を扱う。対して本節は、§23のG1→G2 Publication以降、**古いG1をどのConsumerが読んでおり、いつ退役・物理回収できるか、G2の業務訂正がどの受取先まで届いたか**を扱う。既存#2604（PublicationとReader）、#2542（外部連携）、#1174（Effect）、#2516（Disclosure）、#2600（Reuse）、#4522（Claim再資格化）がOwner。**第二のLease Registry/Release Coordinator/GC Engine/Publisher/Claim Storeを採用しない**。

### 25.1 「読取終了」「物理回収」「訂正完了」は別Claim

| 問い相対の主張 | 必要な証拠 | その証拠だけで言えないこと |
| --- | --- | --- |
| **Reader完読 / 安全な失敗** | 対象ConsumerのPinned Generation・必要Member・実Readの結果・保持期限/失効時拒否 | 他Consumerが同じVersionを読んだ、G2が業務適用された |
| **G1退役 / Blob回収** | Providerの不変世代・生存参照、Pin/Lease、Retention・実GC操作・削除後Readback | 外部CDN/Recipient/Backupが全て消去済み、全宛先にG2訂正済み |
| **G2配布 / 適用 / Owner受理** | 受理された用途・期間・**有限Consumer cohort**と、各Read/Import/Effect/業務Owner記録 | 未登録・不特定・Offline Consumerまで訂正済み |

**この三者は一般に互いを含意しない**。G2をBillingがApplied済みでも、G1をArchiveが合法的に歴史読取中のことがある。逆にG1をGCできてもOfflineの宛先がG2をAppliedした証拠ではない。G1が保管されていても現在のGrant/Disclosureは別。

### 25.2 退役の二つの正当化方式を混同しない

- **継続Read保証型**：不変Generation/版付きMember、Readerの実Open同一性、対象のactive Pin/Lease、Member保持とGCの競合制御が必要。新Pin取得とRetire判定の間に未保護区間があれば、旧Readerを無停止完了できると主張しない。Providerが保証できなければunsupported/unknown。
- **期限後のfail-safe許容型**：Readerとの明示的な契約でRetention期限後の旧Read拒否・別Attempt開始を認めるなら、全Offline Consumerの完了をGCの絶対要件にしない。ただしG1/G2のMemberを混ぜて成功とはせず、必要な期限・現在Grant・参照の検証を要する。継続Read保証と同じ契約として表示しない。
- **共有Content Blob**：G1のManifestや参照を退役しても、G2が同一SHA256のByteを参照していれば、G2が必要とする実Blobを削除できない。Contentの再利用可能性、世代所属、物理Reachability、現在の開示可否を別資格にする。
- **外部に残る旧Byte**：OriginのGCやRevocationを確認しても、外部Recipient/CDN/Backupからの物理撤去を証明したことにはならない。配布済みコピーや既読Evidenceの扱いは#2516/#2542の実Target境界で判定。
- **対象Consumer集合の閉包**：「全員G2済み」を主張するには、Ownerが受理した用途相対のConsumer集合と各宛先の実Read/Effect受理を必要とする。公開Reader一覧に現れないものを不存在へ丸めず、未観測の宛先はunknownとする。

### 25.3 既存実試験を前提にする反例と、まだ未実証の競合

統合済み[#3256](https://github.com/hat47x/tei-platform/pull/3256)の実CLI/別Process Reader試験には、**R1とR2で5 Memberのbyteが同じでも、同Directory上書き後の旧R1 Pinを拒否**する例がある（[#2604実証記録](https://github.com/hat47x/tei-platform/issues/2604#issuecomment-6740694625)）。統合済み[#4240](https://github.com/hat47x/tei-platform/pull/4240)には**R1→R2→R1の中間ABA**、Member Path→物理Open同一性不足の負例がある。これはReadの保証上限の実試験であり、**不変Generation/並走Reader/GCの実現・完全性を証明したものではない**。

| 一因子の競合・変更 | 不適格な主張 / 条件付きの期待 |
| --- | --- |
| GCがPin=0を確認→ReaderがG1 Pinを新規取得→GCがG1削除→Reader Read | 一瞬のPin=0で安全GCとしない。ProviderがPin登録/Retireの競合を制御しない限りcontinued-readはunknown |
| R1→R2→R1、または同Byteの外部MemberへPath差替え | Manifest前後一致・Digest一致から連続世代/同一実Openを推測しない（既存ABA/TOCTOU負例） |
| G1をRetire、G2も同じ物理Blobを参照 | G2の生存参照があればBlobを保持。G1参照削除を共有Byte削除の根拠にしない |
| BillingはG2 Applied、ArchiveはG1 Reading、別の宛先はOffline | 全Consumerの訂正済みは不適格。Owner受理のcohortと未観測を示す |
| G1のDelta baseをGCした後にG1→G2差分のみ届く | Base/旧State/連続Deltaが未資格ならFull G2を再取得、またはunknownへ（§22） |
| G1 Current Grantが失効したが旧Origin Byte・外部Cacheが残る | 保存Byteから現在Read資格を復活しない。Origin revokeで全外部コピー消去を主張しない |

**独立の合成モデル**：4イベント GC_check、Reader_register、Reader_read、GC_delete で、checkがdeleteより先、registerがreadより先という制約を置くと**6通りの合法順列**がある。そのうち check→register→delete→read の**1通り**が、新Pin登録後にGCが進む危険な競合を示す。Pin取得とRetireの線形化が実Providerで成立するなら、Pin先行ではGCを延期、Retire先行では新Pinを拒否できる。**この列挙はTEIの実OS/FS/Provider試験ではない**。

### 25.4 Owner別の受入順（新Runtimeなし）

1. **P0 / #4410**：固定TSVの純粋Go/CLIを優先。GC/Lease/Reader cohort管理を初回BatchのHard Gateに追加しない。
2. **P1 / #2604 AP-01/06/09/12、#2514**：統合済み#3256/#4240の実Reader/ABA/TOCTOU試験を再利用し、限定Providerで継続Read保証型かfail-safe期限型の**どちらを提供するか**Owner判断。実Writerと並走する二世代Reader、Pin vs GCの競合、共有Blob参照を独立Expectedで故障注入。第二Readerを実装しない。
3. **P1 / #4411/#1174/#2542**：Archive/Billingという有限の受取先を用途付きで固定し、G2 Published、Read、Applied/Commit unknown、Owner Acceptedを独立比較。Transport ACKと実Importを混同しない。
4. **P1 / #2600/#2516/#4522**：同ByteのGeneration再利用、Grant-only変更、旧Source A→B→A、GC後の証拠保持を区別し、false retain / unnecessary invalidation / unknown erasedを計測する。
5. **P2 / #1124/#4510/#4487**：Studio/TEAの用途相対Data/Effect/Observation Edgeへ、Reader/cohort/Retired/unknownを必要な公開範囲だけ投影。秘密Readerの件数・存在は開示しない。

**比較先**：[Apache Iceberg Snapshot Reference/Retention](https://iceberg.apache.org/docs/latest/branching/) と [Delta Lake VACUUM保持期間](https://docs.delta.io/delta-utility/) は実Readerと旧世代回収を区別する参考。TEIへIceberg/Delta Runtimeや新GC Schedulerを採用する決定ではない。

**証拠上限**：GitHubの既存限定実Reader負例・静的Source照合・独立合成4イベントの順列のみ。新しいTEI Go/CLI、並走Provider、GC/OS原子性、外部Consumer/Import、Current Grant/Owner業務受理、exact-head CIは **NOT RUN / NOT ACCEPTED**。

## 26. 変更前後Graphと同一CandidateのFull対Selective：過少・過大再計算の独立検収（2026-11-13、設計候補）

**追加した判断**：§24でAffected/Unaffectedを論じたが、「R0とR1で値が違うことを検出した」だけでは**R1の選択的再計算が正しい**ことを証明できない。比較は(1)R0→R1の依存・意味・Source変更を発見、(2)**同一R1のAccepted Meaning・Scope/Cut・Target契約**でFull CleanとSelectiveを独立に実行/観測、(3)対象Claim・Output Member・診断/Effectを照合、(4)省略の根拠と実時間を別判定、の順とする。新たなETL専用キャッシュ検証器、Runner、Fixture/Scenario SSOTは作らない。正式なOperator最適化規則の採用ではなく、既存受入を利用した最小Pressure。

### 26.1 不変条件：Before/Afterは影響発見、Full/SelectiveはR1正しさの判定

| 比較 | 固定する項目 | 許される結論・禁止する結論 |
| --- | --- | --- |
| **R0→R1 Meaning/Dependency差** | Consumer/Question、R0とR1のSource/Rule/Graph版、Edgeの追加・削除・再束縛・実Scope/Predicate | 影響候補と未知の発見。**R0 ResultとR1 Resultの不一致**自体はSelective失敗ではない |
| **Full(R1) と Selective(R1)** | 同じAccepted Meaning、Required集合・Source Cut（複数Sourceなら適格なVector）、Target/Operator/Decimal・Rule版、Expectedと対象Consumer | 出力値・Row/Group/NULL・必要Member/診断を比較。両者一致だけで「全業務入力が完全」「外部Effect安全」と言わない |
| **省略の資格** | Claim-relative Required閉包、R0/R1の旧消失辺、R1選択/実消費、必要な集合・否定依存、未観測Scope、実変更の非干渉 | 影響なしの**証拠があるClaimだけ**skip候補。他ClaimのUnknownを無条件に伝播させない |
| **再利用の資格** | Result/Artifact byte、Source/Meaning/Plugin版、Independent Expected、Release Generation・Reader pin・Current Disclosure | Output同byteでも旧Evidence/Grant/Owner Meaningを無条件に復活させない（§23/§25） |
| **性能** | Full/Selectiveで同条件・同対象の実Run/Cache hit/miss、再実行Stage数、CPU/I/O、冷暖起動/時間分布 | 省略の正しさが未確認なら高速化成功と評価しない |

既存[統合済み #4277](https://github.com/hat47x/tei-platform/pull/4277) の `tools/reference-cache-full-vs-incremental-verification/` はSource/Model/Dependency/Operator/Plugin/Toolchain/Receipt・Artifactの比較とCE-01〜06の負例を持つ。ただし**read-onlyのRunReport比較であり、実Full/Selective Runnerを自ら駆動しない**。そのレポートにFixture値を入れてPASSした事実を実Go/ETL実行の同等性へ昇格しない。まず既存報告形式と期待判定を再利用し、実Runnerがある段階でだけ同一R1を二方式で実行し独立Observedを投入する。

### 26.2 旧依存辺の削除と問い相対Required：N16/N17をETLへ接続

[統合済み #4591](https://github.com/hat47x/tei-platform/pull/4591) は参照B2C Inspectorに**申告された**R0-only Edgeを加え、削除辺の旧dependentを影響候補に残せる。ただし[未統合 #4592](https://github.com/hat47x/tei-platform/pull/4592) の反例は、旧辺削除からR1の全下流を無条件に辿ると、対象外のConsumer/Unknownまで過剰失効し得ることを示す。採用候補は、旧辺由来の起点は**対象ConsumerのR0∪R1 Required上流の内側**に制限し、別途明示されたRule変更は本来の全下流を評価する、という探索の分離。#4592はまだ実装・Ownerの最終受入が済んだ正本ではなく、main #4591との差を維持する。

| 一因子差分（合成。Edge方向はSource→Dependent） | 対象Consumer PのExpected | 別Consumer S/残存UnknownのExpected |
| --- | --- | --- |
| R0: Rate→Rule→P、R1: Rule→P。Rate→Ruleだけ削除 | **PのMeaning/経路再評価**。R0の旧従属を見落とさない | Sに旧Edge由来のRequired Pathがないなら、削除だけでSを失効させない |
| 同じR1でRule→Sも存在、明示Changedは空 | Pに対する旧辺削除の影響候補を保持 | SをR1 forward siblingだからという理由だけで影響ありとしない |
| 同じGraphで明示Changed=Ruleが追加 | PとSの各Required/選択経路を再評価 | 「旧辺削除に関係しない」からとSのRule変更影響を隠さない |
| R0のRate→Rule→P二段鎖がR1で丸ごと削除、Source Rateは未観測 | Pの旧従属と未観測上流を**unknown**として残す | 未観測Sourceを不存在としない。対象外ConsumerへUnknownを伝播しない |
| R0で別Rule X→別Consumer Tの無関係旧Edgeだけ削除 | Pの閉じたRequired集合が本当に無関係なら非影響候補 | Tは別問いで独立評価。変更があるから全世界再実行とはしない |
| 同一Query IDのままWHERE/ORDER/SQL/Environment、RLS Scopeのみ変更 | Required/Selected/Observedの再照合が必要 | 出力Rows一致・0件から物理/業務全件性を推測しない |

**Sourceの実保証**：R0-only Edgeは入力者が申告した仮定であって、旧Native/SQL/Plugin依存を実Readerが全て列挙した証明ではない。#4592修正が将来mergeされても、実R0/R1 Source・Actual Consumed Closure・Current Grantは追加資格を要する（#2586/#3262/#2514/#2600）。自動的なNative/SQLソーススキャナや正式Dependency Core型は本案に含めない。

### 26.3 「読取要求」と「実Read」を区別する独立検証

[Draft #4608](https://github.com/hat47x/tei-platform/pull/4608) はRuleのValidation Sourceに対するNumber/RelationKeysの**呼出先・順序・回数**をProbeで照合し、同じ値が返る誤Attribute/Relation参照を検出する。これはValue equality（EQ1）と**Interface Read Demandの限定的なEQ2**を区別する実装候補であって、実PGの読込列・Runtimeの短絡・Provider出典・秘匿/ACLを観測したことではない。

[統合済み #4539](https://github.com/hat47x/tei-platform/pull/4539) のEnvironment→実PostgreSQL→一時Function Data Contextは別の限定したObserved Read起点である。Query 0件/RLS 2/3/Mask NULLでもCoverageはunestablished。従って、**要求されたSource（Required）→実呼出要求（Demand）→実Provider Reader（Observed）→独立Consumer（Received）**は一つのPASSに畳まない。Rule DraftをPG実Readerへ勝手に結線せず、#2586/#2514/#988と既存Ownerに接続候補を返す。

### 26.4 最小実証順：まず正しい部分集合、次に速さ

1. **P0 #4410**：最初の副作用なしTSV Filter/Convert/Aggregateと独立Expectedを優先。新Full/Selective Engineを必須化せず、R0/R1のRule・Rate・Source変化に対する手書きAffected/Unaffected・Unknownだけを既存Nへ付帯する。
2. **P1 #2600/#4518/#2582**：#4591が申告旧Edgeまで対応した事実と#4592の未統合修正を区別し、**旧辺削除・無関係旧辺・二段鎖・sibling**の影響票を確認。正しく局所化できないときは対象Claimをunknownにし、他Claimを不用意に失効させない。
3. **P1 #4277/#2514/#2622**：ReferenceのFull/Incremental比較に独立Expectedと同一R1・同一Target条件を束縛。Read-onlyレポート比較の段階はREFERENCEと表示し、実Provider/ETLのFull/Selective両Runは新規受入対象として分離。
4. **P1 #4608/#4539/#2586**：限定Rule Demand、実Environment/PGのRead、Data Context、同一Contextを読む二Consumerの**証拠辺**を個別に検証。正式製品`tei` CLI、Studio、Owner Meaning・現在Grantは未証明のまま。
5. **P2 #2831/#2604/#4510**：セルフビルドの実Builder/Stage2/Artifact消費、OutputのGeneration Pin、Studioの問い相対Graphを照合。skipの正しさを実証した後、Cold/Warm FullとSelectiveの時間・再実行数を計測する。

**現時点の証拠上限**：統合済みの参照比較#4277、申告旧Edgeの#4591、限定実PGの#4539、およびDraft #4592/#4608を静的照合した設計案。今回、新たなGo/CLI/PG、同一R1 Full/Selective二実行、失敗注入、CI、実Consumer/Studio、Owner意味受理は**NOT RUN / NOT ACCEPTED**。新Second Oracle/Graph Solver/Cache/Runtime、固定Host/新Grant Issuerは作らない。

## 27. 下流Effectの旧Generation適用拒否と業務補償：Accepted Supersessionを別契約にする（2026-11-13、設計候補）

**独立Gap**：§23はPublication/Consumer Read/Applied/Owner Acceptanceの証拠境界、§25は旧G1のRetention/GCと有限Consumer群への到達を扱う。§24/26は変更依存とFull/Selective再計算の正しさを扱う。しかし、下流の**現在の業務状態がG2へ進んだ後、遅れて届いたG1や比較不能な別訂正によってG2が上書きされない条件**は未明示である。このGapは[Effect Fence #1174](https://github.com/hat47x/tei-platform/issues/1174)、[Command/Receipt CAS #2564](https://github.com/hat47x/tei-platform/issues/2564)、[Import Unit #2542](https://github.com/hat47x/tei-platform/issues/2542)、[Subsystem Owner #3250](https://github.com/hat47x/tei-platform/issues/3250)に渡す。**新しいETL用Transaction Engine、Saga Coordinator、Command State Store、Host/Grant Issuer、第二Receipt Ledgerは作らない**。

### 27.1 Generation identity／受理済み業務順序／実Applyを混同しない

| 独立のClaim | 確認条件 | 推定してはならないこと |
| --- | --- | --- |
| **Source/Result Generation** | 対象Owner/Subsystem、業務期間、Source Cut/Rule・Meaning版、Output/Manifest/Reader Attempt | 世代文字列の辞書順、作成時刻、配送順だけで業務的な前後関係が決まる |
| **Accepted Supersession** | Ownerが当該対象・期間で受理した現行版、期待旧版→次版の業務意味/訂正根拠 | 複数Source Vector・別業務期・別Branchを単一の巨大な全順序にできる |
| **Authorized & Committed Apply** | 現在Grant/Scope、対象Providerの同一効果境界における期待旧版検査、Operation/Request、Receipt・独立Readback | 一式Read完了・公開済み・ACK受信なら業務効果がCommitした |
| **Cohort Correction Accepted** | 受理された有限Consumer集合と各Consumerの取込・業務Owner判断（§25） | BillingがG2適用済みならArchiveや外部受取先も同じ |

Event時刻／Business Effective時点／Source Cut／Generation公開時刻／配送到着時刻／実Apply・Owner受理時刻も別。**最後に到着したものが正しい、またはGeneration番号が大きければ自動上書き可**とは決めない。異なるRule・並行する業務Proposal・Source Vectorは、Ownerが明示する比較可能性/受理関係が無ければ未比較・保留とする。

### 27.2 下流の現在版と期待旧版を、実更新と同じ境界で照合

同一Owner・業務期間Dの現在適用済み版がG1（顧客A：数量5/金額800円）で、受理済みの訂正G1→G2（数量5/640円）を適用する場合の候補手順：

1. **独立Read資格**：G2の固定Input Cut・Rule・必要Output Memberを§23に従いPinned Generationで実読取。Reader成功はAction Authority/Commitではない。
2. **Ownerが受理した更新Intent**：対象のLogical Owner/Period/Scope、G1→G2という具体的Supersession・訂正内容・副作用形式を確認。Expected currentが不明ならGeneration文字列の比較で代用しない。
3. **対象ProviderのEffect境界**：**適用時のCurrent Grant**と、Targetの現行適用版がG1であることを確認し、合致した場合だけG2更新と版・受領記録を結ぶ。これを別々のread-then-writeに分けて競合窓を作らない。Providerが条件付き書込みや同Tx Receiptを実現できなければunsupported/unknownに止め、TEI Coreで分散原子性を補ったとは言わない。
4. **ConflictとUnknownを分離**：現行版が既にG2/G3なら旧G1からの適用はReject/Conflictとし、G1へ黙って戻さない。ネットワーク切断、Commit ACK欠落、Recorder故障なら実Effectはunknownであり、独立Receipt Lookup・Resource Read・現在Grant/Operation同一性を#1174/#2564で再確認する。明確なConflictとunknownを同じretry対象にしない。
5. **別Subsystemのデータ**：ETLから他SubsystemのTableへ直接CUDするより、Owner側の受理済みAction/Importへ委譲する。委譲できても実Import/Owner受理が自動成功するわけではない（#3250）。

**単一DB参照候補**：PostgreSQLではUPDATE WHEREに現在版の期待値を含め、RETURNINGで実際の更新行を確認できる（https://www.postgresql.org/docs/current/sql-update.html および https://www.postgresql.org/docs/current/dml-returning.html）。Read Committed/Repeatable Read/Serializableによる競合時の結果・Retry条件は別に評価する（https://www.postgresql.org/docs/current/transaction-iso.html）。**一行の条件付きUPDATEは、複数DB・外部Recipient・業務Owner Acceptanceの分散原子性を証明しない**。

### 27.3 Restatement、業務補償、公開ポインタRollbackは別操作

- **結果Restatement**：新Source/Rule Cutに基づく完全な新Generationへ置換。ただし必要なOwner受理済みの前後関係・適用版条件がなければ下流CUDを行わない。
- **業務補償**：G2の訂正をさらに取り消す場合、Ownerが受理する**新しいCorrection/Compensation Effect**を要求する。最終金額がG1と同じ800円でも、G1の再送ではなくG2やG3を経た新G4の履歴/Operationを保全する。外部の送金・通知等が必ず一つの逆操作で戻せるとは言わない。
- **Release pointer rollback**：公開ポインタG2→G1の切替は、既に適用されたDB Effect、受取先への配送、Business Acceptanceを取り消さない。旧データのRead可否も現行Disclosure/保持期限に従う（#2604/#2516）。
- **同Byte・別意味**：G1とG4のOutput金額/byteが偶然一致しても、Source Cut・意味版・受理されたSupersession・新Request/Receipt・現在Grantは別。Result Digestを冪等キーやBusiness Versionの代用にしない。

### 27.4 最小独立Expected：G1→G2→G3→G4（既存N/EF/RCAS/APへ一因子で返す）

合成の受理済み直線経路を、Owner=Billing、期間D、A数量5、G1=800円、G2=640円、G3=700円、G4=800円（新補償）と定義する。これは**設計用の手書きExpectedであり、実業務Ownerの承認記録ではない**。

| 一因子変更・配送順序 | 条件付きExpected |
| --- | --- |
| G1 Applied後に受理済みG1→G2を適用 | G2/640円の実Target状態候補。Receipt/独立Read/Owner Acceptanceを別採点 |
| G2の後で古いG1初回配送やG1前提のWrite | Conflict/Reject。G2を800円へ上書きしない |
| G3がG2に先行到着、現在TargetはG1 | G1→G3を別途Owner受理していない限り拒否/保留。最大世代選択しない |
| G1を期待版に持つG2と別G2bが競合 | 同じOwner/対象を同一条件付きCommitで両方成功させない。敗者は再比較/再受理 |
| G2 Commit後のACK喪失・Log不全 | Effect unknown。照会優先、無条件の再Import不可 |
| G3後に新補償G4へ適用、金額800円 | 新G4/新Operationであり、古いG1のReceiptや実行を復活させない |
| Billing G2済み、Archive G1 Read済み、第三宛先Offline | Billing局所Appliedと業務cohort全体の訂正完了は別 |
| G2 Read時はGrant有効、Apply前に失効 | 適用権限の再確認・拒否。Read時Grantを現在のApply根拠にしない |
| G2→G1へ公開ポインタのみRollback | 実Target Applied StateとOwner AcceptanceはG2のまま残り得る |

**局所参照検算**：Python標準SQLiteの単一in-memory接続で、Owner/Period/期待旧Generationを条件とするUPDATEを実行。G1初期800→G2 640成功、遅延G1再Insert/G1前提Writeは更新0件、G2→G3 700成功、G3→新G4 800成功、最終G4/800円を確認。**同時実行、実PostgreSQL、TEIのProvider/Effect Receipt/Current Grantは検証していない**。

### 27.5 既存Owner・検証優先順

1. **P0 / #4410 W1〜W5**：固定TSVのPure Go/CLI同義・独立Expectedを優先し、DB CUDや新Sagaを初回受入Gateにしない。
2. **P1 / #4411/#1174/#2564/#2514**：既存RCAS-01〜06・RequestHash/OperationIDと一つのOwner/Periodを束縛。Conflict、ACK-loss Unknown、同OperationのReceipt再観測、後続別Operation、現在Grantを同じ実Provider/Targetで別々に試す。既存実PG/Command局所証拠を流用してETL本体Apply成功と断定しない。
3. **P1 / #2542/#3250/#2604**：Owner側Action/Importへの委譲、Pinned Generation、実Effect/業務受理を接続。外部Recipientとの非原子境界を残す。
4. **P1 / #2600/#2516/#4522**：Source A→B→A、G1/G4同Byte、Grant-only変更、G3先行と無関係Consumerを一因子変化させ、false retain/過剰invalidating/unknown erasedを独立採点。
5. **P2 / #1124/#4510/#4487**：Studio/TEAはSource Cut→Generated→Published→Read→Effect→Owner、Supersession/Conflict/unknownを問い相対に表示。非開示Writer・Readerの数を漏らさず、画面上の旧選択でEffectを実行しない。

**停止線**：業務版の比較・補償の意味・適用権限の発行主体・外部Commitの再試行可否は既存Ownerに委ねる。自動的な最大Generation採用/逆演算/Saga、TEI Coreの新全順序型やTransaction Coordinatorを追加しない。**§27は設計・SQLite局所参照検算・既存Issue静的照合であり、新たなTEI Go/CLI・実PG/Provider/外部Import/Owner意味受理・exact-head CIは NOT RUN / NOT ACCEPTED**。

## 28. 実行・省略工程の観測閉包：Planと同一Attemptの実Traceを突き合わせる（2026-11-13、設計候補）

**焦点**：§26では同じR1に対するFull/Selectiveの結果と影響範囲を比較し、§27では後続業務Effectの受理済み順序を分けた。しかし「Plan内で選択したStageが本当に走ったか」「省略したStageは何を根拠に不必要と判断したか」「図にない実Read/外部処理がないと言えるか」は、**値やArtifact一致からは導けない別Claim**である。セルフビルド[#2831](https://github.com/hat47x/tei-platform/issues/2831)のBuildIntent→実Builder→Stage2、Studio[#1124](https://github.com/hat47x/tei-platform/issues/1124)/[Draft #4510](https://github.com/hat47x/tei-platform/pull/4510)の図表、増分[#2600](https://github.com/hat47x/tei-platform/issues/2600)の実Consumed、横断[Draft #4522](https://github.com/hat47x/tei-platform/pull/4522)のEvidence Joinを、**ETL一件のread-only受入票**として合成する。新Runtime/Tracer/Source Registry/Graph正本/Receipt Engineは採用しない。

### 28.1 実行Graphは「宣言・選択・実呼出・読取・結果」の範囲付き突合

| 観測段階（正式状態enumの提案ではない） | 独立に必要な証拠 | その段階のPASSだけでは言えないこと |
| --- | --- | --- |
| **Declared / Required** | 受理済みFunction/Plan・Operator、Required Source/Rule/Scope、想定Stage/Effect・外部境界、Meaning/Graph Revision | 全依存が選択・実ロード・実消費された |
| **Selected / Lowered** | Step/Provider/Plugin/Targetの選択、実SQLやFile/Command・版、R1 Source Cut、Plan→選択Stageの対応 | 実Providerが呼ばれた、OS/外部Pluginで他工程が存在しなかった |
| **Invoked / Completed / Failed** | **同じ計算Attempt**でのStep開始・終端・失敗・順序、実invoker/実Process・Provider呼出の対応、観測器の被覆 | 呼出成功＝期待した入力byte消費、結果確定、全業務Effect発生なし |
| **Skipped / Blocked / Unknown** | Skipは対象StepとClaimに相対化した依存閉包・再利用候補・非影響根拠。BlockedはGuard/Validation/前段失敗。Unknownは実行状態または被覆不足 | 単にStepのTraceがない＝安全Skip。Blocked/未開始＝Cache Hitや全体無作用 |
| **Consumed / Produced** | 実Open/File Member/SQL実行とReturned Rowset/NULL/部分性、Source occurrence・Scope/Cut、実Output/Artifact Member・Digest・Attempt | 「0 rows」＝Collection全件空、宣言Source＝実Consumed、Output digest一致＝同一実行 |
| **Published / Received / Effect Readback** | 出力GenerationのPublish、別ConsumerのPin/Read、下流Operation/Commit・後日の独立Readback・Owner/Current Grantを**別々の証拠**で接続 | 異なるAttemptのReadbackを生産Attemptと同一視、ACK/画面表示＝業務受理 |

**同一Attemptを要求する箇所**：Planの選択→Stepの実Invocation→その実Read/実Outputへ因果帰属させる際には、Run/Attempt/Stepまたは同等の実行識別、Source/Rule revisionと実Targetを関連付ける。**別Attemptを許容する箇所**：独立Reader/DB ReadbackやConsumer検収は別の実行であること自体が有効で、対象State/Generation・時点・Scope・前段Result/Operationの関係を示し、限定的な整合Claimだけを形成する。二つのGreen・同じValue/Digestだけでは因果Joinを成立させない。Source A→B→Aや同Byte別Workspace、並行別Runがある場合は特に旧Attemptの資格を流用しない。

### 28.2 「図中になければ無い」を成立させるための観測範囲

**図の完全性は問いと観測境界に相対化する**。例えば「この限定した純粋TSV参照処理の、既知のFile ReaderとStage Adapter経由のI/O/実行Stageは図と一致する」というClaimなら、列挙した入口・実呼出・計算・出力で閉包を検査できる。一方、任意Pluginの不透明な内部、OSの暗黙I/O、DB Trigger、外部Worker、未保存Editor Buffer/Studio別Workspace、非開示Sourceまで図外の不存在を普遍的に宣言しない。

最低限の三つの突合：

1. **Planned→Executed**：列挙したSelected StageごとにCompleted/Failed/Blocked/Skipped/Unknownを別に集約する。Completedには実CallerのIdentity/Attempt/版、Skippedにはその対象Claimについて成立したRequired・Consumedの閉包と非影響根拠を対応付ける。単純な計算Stageの予定外実行/未申告外部Side Effectがあれば、観測可能な範囲でPlan discrepancyとして扱う。
2. **Required→Consumed**：必要なSource/Range/Predicate/negative dependency/Lookup/rate等と、実Provider/Readerが**そのAttemptで**消費した入力を照合。実Read行が0・部分Page・RLS/Mask・Source Binding未検証・opaque externalはClosure=unknownを保持。読取要求（Draft[#4608](https://github.com/hat47x/tei-platform/pull/4608)の限定Probe）と物理Read（統合済[#4539](https://github.com/hat47x/tei-platform/pull/4539)の実PG）を別証拠とする。
3. **Produced→Used**：Planにある最終Output member/Manifestが、実生成・Stage→Publish→Reader pin・実Consumerの必要Memberを経て到達したかを照合。公開Ref切替・二世代混在・途中停止・Commit Unknownは§23/§25/§27と[#2604](https://github.com/hat47x/tei-platform/issues/2604)/[#1174](https://github.com/hat47x/tei-platform/issues/1174)に委譲する。

Instrumented boundaryが部分的なら、**「観測できた範囲のGraph整合」**までは言えても「他に処理がない」と言ってはならない。この区別はStudioの表示にも残し、Unknown Node/外部境界を図から消さない。Plan Graphを実行器そのものに置換せず、Data/Control/Effect/Observationの別Relationとして同じSource/Revへ投影する（§20）。

### 28.3 独立Expected：同じR1の二方式と未観測工程

すべて設計上の手書きExpected。固定R1でConsumer AはQuantityだけ、Consumer BはRate依存のJPY金額を出すという§24/§26と同じ合成条件とし、実製品の試験ではない。

| 一因子の違い | Qualified claim / 意図的に拒否するClaim |
| --- | --- |
| R1でRateだけ改版。AはそのSourceと非干渉が**閉包証明済み**。SelectiveがAを根拠付きSkip、Bを実Invoke/実Consumed | Aのskipは**局所候補**、Bの再計算を証拠付ける。Full(R1)とSelective(R1)のResult/必要Member/診断は別Attemptで比較。Aの過去Grant/Meaning受理を継承しない |
| 同じResult byteだが、Bが未Invoke/欠落Trace、skip理由も不明 | Bの計算実行・安全skipは**unknown**。Byte比較PASSのみから実行Graph完了へ昇格しない |
| Rateの実Readerは宣言Graphに無く、Runtimeで外部Lookupが行われた | 監視対象内で実Readを観測したなら**宣言Graphと実行Graphの不一致**。実際のI/Oが追跡対象外なら「無い」ではなく観測閉包未証明 |
| Guard NGでTransformが一度も呼ばれない。前段に既にEffect/Temp/ログ出力がある | Transformは**blocked/not-invoked**であり、最適化skipではない。Run全体を無作用へ丸めない |
| Rule Probeは正しいAttributeを要求したが、Providerが別Environment/SQL/Scopeを読んだ | Interface Demandは限定PASS、実Source/Target一致は不適格。Read Demandと実Provider Readを同一証拠にしない |
| PostgreSQL Queryが0行またはRLSで2/3行、Result Digestは期待と同じ | Source Population Complete/全件不存在は**未成立**。見かけの0件を全業務Factが0としない |
| Generation G2を別Consumerが**別Attempt**でPinして必要Memberを読んだ | 別Attempt/別時点のQualified Read Witnessとして記録。ProducerがG2を生成した同じAttemptだったとは言わない |
| BuildIntentに記載のないesbuild/embed/外部Commandや、Stage1自身が起動していないStage2が存在 | 限定的な工程観測は記録できるが、Self-build L2/L3と「図に無ければ無い」の完全Claimは保留（#2831/#4583） |

### 28.4 実装順：新Tracerを作る前に既存の観測を結び付ける

1. **P0 #4410 W1/W2/W3/W5**：固定TSVのInput/Output digest、直接Go・CLI二入口、Independent Expectedを先行。Pure Runについては観測できる入口/Reader/Writerと失敗時の未公開を、**そのRunだけ**の限定Ledger/RunLog/Artifact対照票へ記録する。未実装のHookを製品Runtimeへ必須化せず、外部Effect=0の検収は試験対象のI/O/Effect境界を明示する。
2. **P1 #4277/#2600/#2514**：既存`tools/reference-cache-full-vs-incremental-verification`のread-only Full/Incremental比較、#2600 DI-01/02/07の事前固定Expected、§26の同一R1比較へ「各StageのPlanned・Invoked・Skipped reason・実Consumed」を照合するレビュー列を付ける。参照RunReportに作者が値を埋めた成功を実Runner Observedへ昇格しない。
3. **P1 #2586/#2622/#4608**：まずDraftのRule Demandの実呼出範囲と、merged #4539の実PG Provider Read/Contextを別列で取り、次に**通常`tei` CLI→実Context→異質二Consumer**で実Producer Attempt/Consumer Readを独立結線。Query/Provider版・RLS・Source Coverageを持たないなら欠損を表示。#4539の成功を製品CLI受領へ昇格しない。
4. **P1〜P2 #2831/#4583/#4510**：同じBuild Target/RevisionのBuildIntent→選択Build Step→実Invoked Builder→選択/実消費Input→Artifact Member→実WebView2/Stage2 Processの**不足一辺**を逐次観測。別HEADのGreen、選択されたGo/esbuild Source一覧と実Read時Source Closure、Stage1自身のStage2駆動を混同しない。Studioのread-only図表はScope/unknown/外部Boundaryを残す。
5. **P2 #2604/#1174/#2542/#2516**：Pub→Pin→Read→Applied/Effect→Ownerを別Attempt/時点と現在資格でつなぐ。Commit Unknown/旧Generation/現在Grant失効時に自動再送・無条件公開へ進めない。

**証拠上限**：本§28は既存main/Draft/Issueと横断設計の静的照合に基づく**ETLの限定受入案**で、正式Event schema・Recorder/Graph Engine・新Runner・新Proof Store/Hostを追加しない。TEI Go/CLI/Provider/Studio/Stage2の新実行、全工程網羅性、外部Effect非存在、独立Consumerの同一Result受領、現在Disclosure・Owner Meaning、exact-head CIは **NOT RUN / NOT ACCEPTED**。

## 29. 下流Applyの競合粒度：Full Replace・Scoped Replace・Deltaと不変条件（2026-11-13、設計候補）

**§27を精密化し、§28と区別する**：§27の「受理済み期待旧Generationを検査してから下流EffectをApply」は、同じOwner/期間に閉じた訂正を安全側で説明する。しかし**期間D全体を1つのCAS粒度**にすると、独立した顧客A/Bの修正さえ競合し得る。逆に**明細Key単位のCAS**なら同じ顧客の集計・期間合計・他SubsystemのBusiness Invariantを壊す余地がある。§28は実行/省略Traceの閉包を扱い、本節は**業務的に許されるWrite・Read依存・Invariantの範囲**を扱う。既存[#3250のInvariant-aware検討](https://github.com/hat47x/tei-platform/issues/3250#issuecomment-6853879476)、[#2564](https://github.com/hat47x/tei-platform/issues/2564)、[#1174](https://github.com/hat47x/tei-platform/issues/1174)、[#2593](https://github.com/hat47x/tei-platform/issues/2593)、[#2622](https://github.com/hat47x/tei-platform/issues/2622)、[#2600](https://github.com/hat47x/tei-platform/issues/2600)へ条件を返す。新Write Scope DTO・Saga/Transaction Engine・専門のConflict Resolverを追加しない。

### 29.1 競合単位は最低三つのFootprintから決まる

| 分離する面 | 当該Operationで検証すること | 省くと起こる失敗 |
| --- | --- | --- |
| **Write footprint** | Owner・業務対象/期間・Group/Field・実際に更新/削除/追加する対象集合 | 古いFull Resultが他Groupの更新を上書き。削除対象や新規行を見落とす |
| **Read/Precondition footprint** | 集計に寄与するSource Cut/Rule/FX/Lookup・Predicate/Scope・期待旧版、必要な未取得/否定依存 | 書き先Keyが別でも、古いRateや新規Accepted明細の影響を見落とす |
| **Business Invariant/Effect footprint** | 期間合計・グループ横断制約・受理済み更新Intent・現行Grant/Receipt・Owner側Action | 行が別なら独立と誤認。越境Direct CUD=0だけで整合性PASSとする |

**局所化の十分条件候補は、同じOwner/Source Cut・意味版のもと、相互のWrite/Required Read/Invariant依存に干渉がないことを有資格に証明でき、実TargetのCommit境界がその条件を守る場合だけ**。不足があればConflictを過小検出してはいけないが、無関係と確認済みの別Groupまで一律失効/Rejectしない（#3250/#4522/#2600）。

### 29.2 Full／Scoped／Deltaは適用時の意味が異なる

| 形態（説明用の呼称。正式Enumではない） | 必要な追加資格 | 安易に認めないこと |
| --- | --- | --- |
| **Full Replace（業務期D全体）** | Dの必要Input/Population/Temporal Cutの閉包、対象全集合（削除/欠落も含む）、期待旧版群またはOwner受理の同等Fenceを同じEffect境界で照合 | Aの期待版だけ一致しているからBも古い値で上書き可能。既存行の件数一致からPhantom不在 |
| **Scoped Replace（顧客A等）** | 受理済み対象Predicate/KeyとScope内の完全性、Group移動・未取得/新規Matching行、Scope外Read/Invariant非干渉、現在Applied版 | 既存のA行だけCASすれば「新規A明細なし」も保証されたと宣言 |
| **Corrective Delta（旧寄与→新寄与）** | Fact/Event identity・連続したbase→target・旧寄与/新寄与・単位/丸め/Operator Retractable・重複配送とReceipt | UPDATE_AFTERだけを別INSERTとして加算、key-only DELETEから旧金額を推定、Delta Byte一致で再送安全 |

これらはTEI共通Operation/Function・Providerの型付きContractを比較するための表であって、新DSL/Engineを選定するものではない。**同じArtifact Generationを読めたことと、下流のどのScopeを置換できるかは別**（§23）。条件付きRow UPDATEだけでは複数行のFull Replaceや外部Recipientまで原子的に保全できない。参照：[PostgreSQL UPDATE](https://www.postgresql.org/docs/current/sql-update.html)、[Transaction Isolation](https://www.postgresql.org/docs/current/transaction-iso.html)。

### 29.3 二顧客と共有集計による独立Expected

合成のBilling/期間D、同一JPY単位で、最初にA=800円/rev1、B=50円/rev1が実Targetにあるとする。A→640円、B→70円の二つの訂正を考える。**A/B間にBusiness Invariant・共通Read依存がない**とOwnerが受理した有限Caseに限れば、異なるGroupへの期待旧版付き更新は両方成立できる。この状況で期間全体を一つの版として必ず競合にするのは、安全だが過度に直列化する候補である。

| 一因子条件 | 独立Expected |
| --- | --- |
| A rev1→rev2=640、B rev1→rev2=70、相互に独立が立証済み | 最終A640/B70。両操作を不要に全期間Conflictとしない |
| 古い全期間Snapshot (A800/B50) から作ったResult (A640/B50) を両Patch後にFull Replace | Bのrevが旧値と違うのでFull不適格。B70を50へ巻き戻さない |
| 二つの明細o1/o2は別Keyだが同じA集計に寄与 | 明細のRow CAS成功だけでA集計が一致すると主張しない。A集計側の再評価/CASまたは有資格演算合成が必要 |
| 旧SnapshotでPredicate status=acceptedの行を列挙後、新しいaccepted行を別TransactionがInsert | 既存RowのCASだけではPredicate対象集合の閉包を示せない。Providerの隔離/業務Cutと再資格化が必要 |
| AとBが共通期間総額を満たす、またはAがBのFX/Ruleを参照 | 異なるWrite Groupでも非干渉ではない。Read/Invariant footprintの保全が必要 |
| 同OperationIDで期待旧版または対象Scopeだけ変える | 同じ再送とは扱わない。#2564のRequestHash・同一性規則へ戻す |
| Full ApplyのACK-loss、または途中までのEffect | Unknown/部分成功の実照会。無条件retry・別Scopeへの自動分割をしない |

**局所計算**：Python標準SQLiteのin-memory表でA/B各rev1から、B=70/rev2とA=640/rev2の期待旧版付き更新を順に適用し、旧rev1によるA再更新は0行であることを確認した。A/Bの現行rev2からは、旧Fullの期待版(A rev1/B rev1)が一致せず拒否候補となる。もしBを無条件に50円へ上書きすると20円のLost Updateである。**同時実Tx・Full Replace/Predicate Lock・実TEI/PG/Owner Grantの試験ではない**。

### 29.4 最小実装順と責務の停止線

1. **P0/#4410**：固定Pure TSV→Filter/Convert/Aggregate→決定的Body＋独立Expected・二入口が先。Full/Scoped/Deltaの実DB Applyを初回のHard Gateにしない。
2. **P1/#1174/#2564/#2514/#2542**：Owner/Period/Scope・期待旧版・Operation/Requestと対象ProviderのAtomic条件付きApplyを結び、Full/Scoped/DeltaのConflict/Unknownを一因子ずつ実検証。新DB/Effect Coordinatorなし。
3. **P1/#3250/#2593/#2622/#3279**：業務不変条件を必須同居へ機械変換せず、Owner側Action、条件付Commit、順序・補償、要件の緩和を独立なTrade-offとして可視化。
4. **P1/#2600/#4518/#4522**：Full/Selectiveの同一Revision検証とは別に、Applied Scopeの過小/過大Conflictと現在Disclosureを比較。
5. **P2/#2604/#1124/#4510**：実Artifact Generation/ReaderからOwner側Applied Scope/Conflict/UnknownへEvidence Edgeを結ぶ。秘密Source・Reader・Writerを公開図で漏らさない。

**検証上限**：GitHubのIssue/既存限定参照実装・公開SQL仕様の静的照合と、独立SQLite単一接続の局所更新検算まで。新TEI Go/CLI・実PostgreSQL/複数Process・Full Replace/外部Import・Current Grant・Owner意味受理・exact-head CIは**NOT RUN / NOT ACCEPTED**。新Canonical Write Mode/Proof Engine/Hostを追加しない。

## 30. 要件選択を跨ぐETL/Batchの使用資格：即時不変条件・保留許容・補償の境界（2026-11-14、設計候補）

**他レーンの新たなInput**：[#3250のRequirement Pressure](https://github.com/hat47x/tei-platform/issues/3250#issuecomment-6855362290) と [Minimal Conflict Witness・K/L/S案](https://github.com/hat47x/tei-platform/issues/3250#issuecomment-6855706071) は、受注と在庫の論理分割を、越境Direct CUDの最小化だけでなく**「いつ、誰に、何を保証するか」「失敗をどこまで許容するか」という要件選択**に戻して検討する方針を示した。§27はGeneration/受理済み訂正順序、§29は適用Write/Read/Invariant Footprintに着目している。本節は**Batch/ETLがその不変条件を実際の使用目的で侵害しないか**を追加する。第二Requirement/Scenario/Business Acceptance正本や専用Saga/Tx Engineは作らず、#3250の判断資料を#1130/#2542へ用途相対で受け渡す。

### 30.1 要件を選ぶ前に分割案を合格扱いしない

固定した同一Owner/業務対象/Source revisionで、合成例の即時不変条件を「OrderがConfirmedなら、各Commit後の観測可能な状態において有効なReservationが存在する」とする。注文の受付はConfirmedとは限らない。

| 業務の選択肢（#3250のK/L/S、いずれも未受理の説明用候補） | Batch/越境Functionへ必要な条件 | 残すべきTrade-offと未証明 |
| --- | --- | --- |
| **K: 強い約束を維持** | Confirmedへの確定とReservationを、実Targetで保証された一つの整合性境界に置く。Owner側の統合Command・同一Tx責務を第一候補とする | 同じSubsystem名だけで原子性は成立しない。論理上複数Subsystemでも同じ実Txに閉じる設計は可能性として残すが、実Provider保証が必要。別RunのETLファイル搬送で即時原子性を代用しない |
| **L: 引当不成功を業務的に許容** | Accepted Orderの未引当・失敗・取消等の業務状態、失敗種別、最大保留期限、通知・返金・代替・手作業Ownerと、再試行/重複/不明Effectの処理を受理する | 技術的な結合が減っても顧客体験・運用・資金拘束・照会負担は増え得る。Eventualと書くだけでは期限内解消・補償完遂を保証しない |
| **S: 受付と確定の時点を変更** | Received/Pendingを許し、Reservationが有資格に成立した後だけConfirmedに移る。Pendingから拒否/取消/期限切れの終端を定義する | 非同期Command/Batchがあっても、Confirm直前の引当失効・期限切れ・ACK-loss・並行二重引当への保護が必要。状態追加はAtomicityの実証ではない |

**適用順は「要件衝突の説明→業務Ownerが変更案を受理→受理済みMeaning/Versionに対して分割候補を再評価→実Target/Effectを検証」**。AIや分割OptimizerがKを勝手にL/Sへ緩和しない。KとL/SはBusiness obligationそのものが違い、Cross-Write Countや単一Pareto Scoreを同条件の構造最適化改善として比較しない。強い要件を維持する候補では、分割を見送ることも適格な結論とする。K/L/Sを新Core enumに固定しない。

### 30.2 ETLのOutput作成と業務Commandへの使用を分類する

| ETL/Batchの使用位置 | 成立し得る限定Claim | 追加で要求すべき境界 |
| --- | --- | --- |
| **Read-only集計・過去帳票の生成** | 固定Cutと独立Expectedに相対するPure Result/Artifact | Input Closure、Rule/Time、Partial/Final、Generation一式公開。これだけではOrder/Reservationの業務Commitを起こさない |
| **注文・在庫の非正本Projection/Catalog更新** | 投影/検索対象だけがR1→R2へ進んだ | Projectionの鮮度・Consumerが利用した世代、現在Disclosure。Projectionの最新値を、確定直前の在庫の権威ある引当証拠へ昇格しない |
| **Ownerに委譲するCommand/Import** | Owner側Operationへの送信と受信/実Commit/Readbackが別々に観測された | Authority、Source/Request Hash、Operation ID、Expected prior state、Provider Commit/Receipt、現行Grant。越境Direct CUD=0でも、委譲先の実Effect・不変条件は残る |
| **Restatement・遅着Delta・補償Batch** | 新しいSource Cut/Accepted supersessionに対する再計算結果 | 旧Generation適用拒否、Scope内のRead/Write/Invariant footprint、既受理状態との因果、通知/補償/再送とOwnerの期限。過去G1の配送成功から現行G2の適用を推測しない |
| **External Trigger/Worker/MFT・不透明Plugin** | 宣言/Selected/Transport ACKなど観測した段階まで | 未追跡のWriter/Trigger、外部Recipientの状態、独立Readback、現行Disclosure。図表に見えない処理を不存在と断定せずunknownで残す |

**論理分割は物理配置ではない**。同一DBを用いる複数論理Subsystemと、別DBの一Subsystemも概念上は区別する。TEIの標準Graph/typed Port・Operation/Provider/既存Pluginから比較し、特定の分散Tx、Saga、Host、専用Workflow/CDCエンジンを先行採用しない。

### 30.3 最小の独立Expected（#3250 SP-N01〜N09と#1130 N/EFへ対応付ける候補）

| 一因子の合成Trace | 確認すべきExpected / 偽成功の拒否 |
| --- | --- |
| Confirmed Order Commit → Inventory Reserve不成功。別Subsystemの越境Direct CUDは0 | **Kなら不変条件違反**。構造の越境Writeゼロ・Transportが成功したことから合格にしない |
| Inventory Reservation成功 → Confirmed Order Commit失敗 | 孤立Reservation・解放/TTLとOwnerを未解決で残す。「順序を逆にすれば無条件安全」としない |
| Received/PendingをCommit → Reserve成功 → 実状態照会 → Confirmed（S候補） | 当該合成Traceでは確認前のConfirmedを避けられるが、並行更新・Lease/TTL・Target Commitを実証するまで製品安全性はunknown |
| Pending → Reserve技術Timeout/ACK-loss、Provider Commitは不明 | **unknown Effect**。業務的欠品・確実な予約失敗・二重再送許可に縮約しない。Operation/Receipt/独立Readbackへ |
| L候補でPendingのまま期限超過し、Cancel/通知/Refundが実行されない | 「失敗を許容する」という宣言だけでは合格しない。業務Ownerが受理した期限・手動復旧・未終端を別に検証 |
| G1売上集計の作成/公開成功だが、同値を用いて即時Inventory確保済みと表示 | Artifactの公開から現在在庫引当のEffectを推測しない。利用用途に必要なFreshness/Authoritative Ownerを照合 |
| G2に訂正した後、外部Batchが古いG1でOwner Importを再送 | 期待旧版/Operation/Scopeが不適格なら拒否・照会。配送・計算再実行の成功を業務適用成功にしない（§27/29） |
| OwnerがK→Lへ要件緩和を**提案**、未受理のまま分割/ETLを選択 | 旧Kへの要求不適合を維持。承認前にAccepted Rule/Batch/Grant/Studioへ自動反映しない |
| OwnerがLを**正式受理**し、新たな通知/補償義務が増える | 新Meaning/Scope/Source CutでRequired/Selected/Consumed/Effectを再評価。旧K版と新L版の数値Scoreを同一問題として比較しない |

最小反例は、特定の順序・故障・観測範囲についての**設計上のWitness**であり、全分散Targetでの不可能性証明や正式な業務要件受理ではない。複数Sourceの整合Cut（§18/20）、旧GenerationのPublication/Reader/GC（§23/25）、外部EffectのCommitted/Unknown（§27）、実行Graph（§28）を一つの成功判定へ押し込まない。

### 30.4 優先順・Owner境界・検証資格

1. **P0 / #4410**：Pure TSVの最小Go＋CLI二入口と固定Expectedを完了させる。K/L/Sの実業務Command・別DB・Compensationを初期Gateへ追加しない。
2. **P1 / #3250/#1125/#1127/#267/#2514**：既存SP-N01〜N09のK/L/S合成Trace・業務例外・Source逆引き・未受理をread-only説明として接続。個別の要件衝突に対し最小の問い/不足Evidenceを示し、Ownerが採用した変更だけを新Sourceへ反映。Subsystem候補のAccepted Direct CUDと実Effectは別資格。
3. **P1 / #1130/#4411/#2542/#1174/#2564**：ETL Import/Command/遅着Correctionについて、必要なAccepted Meaning、Source Cut、Business State/Deadline、Expected prior state、Operation/Receipt/独立Readbackを別観測。Timeout・ACK-loss・partial Applyはunknown/partialのまま照会に戻す。
4. **P1 / #2593/#2600/#2622/#3279**：R0→R1で業務契約が変更された場合の依存改版・実Read・Full/Selectiveの比較を「同じR1の意味で」実施。見かけ上の同byte・同数値で現在の業務約束を再利用しない。
5. **P2 / #1124/#4510/#2831**：同じSource由来の問い相対図表にBusiness Promise→Data Invariant→Function/Effect→Batch/Generation→Failure Witness→Owner Decisionを投影。観測範囲外のCommand/秘密情報を欠落ゼロとして可視化せず、実Studio/Builder/Targetで別検証する。

**停止線と現状**：#3250が分割候補/Requirement Pressure/Owner判断先を所有し、#267が正式意味受理、#1174が実Effect、#1130がBatchの縦断受入を所有する。本節は既存Issue/PRの静的照合による**ETL側の要件・停止条件・独立Expectedの引渡し**である。新たなProduct Go/CLI/実PostgreSQL/Command・別Subsystemの実Tx、障害回復、Owner受理、exact-head CIは **NOT RUN / NOT ACCEPTED**。新Core K/L/S型、第二Oracle/Scenario SSOT、分散Tx/Saga Coordinator、必須Hostは追加しない。

## 31. 期間締め・訂正再開とBatch Effect：旧Open資格のABA失効（2026-11-14、設計候補）

**§30との分離**：§30は業務要件としての即時整合K、保留許容L、受付/確定分離Sを選ぶ資格を所有する。本節は要件が受理済みの場合でもなお生じる**Period StateのOpen→Closed→Open、古いRunPlan Precheck、下流のApply/Effect Commit間のTOCTOU**を扱う。§27/29の受理済みSupersession、適用Scope、期待旧版へ**期間観測資格**を加えるだけで、新Period DSL・Generation Core型・Effect Engine/Host/分散Txを作らない。

### 31.1 三つの観測境界（同じPASSへ潰さない）

| 境界 | 分離して確認するもの | それだけでは未証明 |
| --- | --- | --- |
| **RunPlan / Admit** | Required Period/Effect Guards、Callerが実際に供給したGuards、観測Period State・Source Revision | Commit時もPeriod Open/Grant有効、Required Guardがすべて渡された |
| **Period Transition / Durable Readback** | Open→Closing→ClosedまたはClosed→Openの状態・遷移、CloseとCorrectionの宣言Capability、後続の実State Readback | CallerのCapability ID申告＝実Capability実行/業務Receipt、State Fileと下流DBが同Tx |
| **Apply / Business Effect** | 受理済みOwner・期間・訂正Intent、**現在期間状態と現在性**、現在Grant、期待旧Applied Generation、実Provider Commit/Receipt・独立Resource Read | Reader/RunPlan/Period File/CLI Exitだけで業務上の訂正完了、外部Recipient全員へ到達 |

既存[#2882](https://github.com/hat47x/tei-platform/issues/2882)はWorkspaceのPeriod/Effect RuleからRunPlanへ接続する局所試験を持つ（Period State Map、合成Step/Journal）。**Callerが全Required Guardを渡したこと、外部DB Commit時の期間状態の再検証を証明していない**。このGapは#1130のRequired-vs-Suppliedと実Target受入へ戻す。

### 31.2 Close能力X、Correction能力Y、State変更後Errorを別々に扱う

未統合[Draft #4397](https://github.com/hat47x/tei-platform/pull/4397)は、宣言されたCloseRequires XをOpen→ClosingとClosing→Closedに、Correction YをClosed→Openに分ける。XとYが異なるとき、ReopenにXまで要求するのは過剰であり、Yだけで通常Closeを許すのも誤り。ただし**Capability IDが一致しただけでは実Y Invocation/現在Grant/Owner Acceptanceの証拠にならない**。

未統合[Draft #4389](https://github.com/hat47x/tei-platform/pull/4389)は、実Workspaceで状態File保存後のCheckPeriods拒否によりHandlerがErrorを返し得ることのReadbackを担当。**CLI非0・Handler Error ≠ Period State変更なし**。独立State Read、変更済みStage、下流Effect/Recipientの別観測を要する。両DraftのGo/CLI変更は別OwnerのためこのETL設計PRからは編集しない。

### 31.3 Open→Closed→OpenのABAを見逃さない

固定の合成例としてPeriod Dの観測版を (Open,rev1)→(Closed,rev2)→(Open,rev3) とし、旧Run Aはrev1時点でPlan precheckを通過したとする。**rev1/rev2/rev3は試験用の比較値であり、現TEIのperiodstateにepochが実装済みという意味ではない**。

| 一因子変更・障害 | 手書きExpected |
| --- | --- |
| rev1/OpenでRun AをAdmit後、Period Closeがrev2へ | 古いPlan admissibleをCommit資格へ昇格しない。Aの旧G1→G2 Applyは拒否/保留 |
| 正規のCorrection Yを受理してPeriod rev3/Openへ再開 | OpenというStateだけから旧rev1 Attempt、旧Grant、旧Checkpointを復活させない |
| rev3/Openにrev1を期待するAが遅着 | State=OpenでもRev ABAのため**拒否**。必要なOwner/Rule Cutで新Attemptを作る |
| rev3で受理済みの新Attemptが期待G1→G2を適用 | 対象Providerが現在Period/Applied版/Authorityを**同じ更新境界**で保全できる場合だけEffect合格候補 |
| 再Close rev4後、rev3で準備した旧G2→G3が遅着 | 期間の現在資格不一致で拒否/保留。旧訂正を自動Resumeしない |
| Period State File更新後に非0 Exit | StateをReadback。Errorだけで無変更、あるいは他Provider Effect確定とは判定しない |
| 外部Import/DBと期間Fileが別Provider | 一つのSQL UPDATEで二者の原子性を保証できない。実Fence/回復/Ownerが無ければunknown |

**独立合成検算**：Python標準SQLiteの**単一in-memory DB/単一接続**にPeriod(D,state,epoch)と集計Target(A,amount,appliedGeneration)を設け、条件付きUPDATEに現Open/期待epoch/期待AppliedGenerationを含めて実行。初期rev1/G1=800円→Close rev2→旧Apply更新0件→Reopen rev3→旧rev1 Apply更新0件→**新rev3/G1→G2(A640円)更新1件**→再Close rev4→rev3向け旧Apply更新0件を確認した。**これは対象PeriodとTargetが同じDB・Transaction境界に存在するという合成仮定**であり、TEIの現Period File／実PostgreSQL/複数Process／Current Authorityや実Receipt/Owner受理を試験していない。

### 31.4 業務不変条件・実装Ownerへの引継ぎ

- **P0 / #4410**：固定Pure TSVの二入口・独立Expectedを先行。Close/Reopen・Effect Writerを初回のHard Gateにしない。
- **P1 / #1130・#2882・Draft #4397/#4389**：Required Guard集合とSupplied Guard集合、Close X／Correction Yの遷移、CLI Error後State Readbackを既存テスト/Ownerで実証し、Declared/Observed/実Invocationを分離。重複実装・同一Fixtureの第二正本なし。
- **P1 / #1174/#2564/#2514**：実ProviderのClose/Apply競合、現行Period版/Applied版、RequestHash/OperationID、Commit ACK-loss/Recorder失敗とReceipt/独立Readbackを対照。期間とEffectが異Providerなら、同一Txの保証を偽称しない。
- **P1 / #3250/#2542/#2604**：締め済み期間の不変条件と、訂正を拒否する／明示再開する／次期補償とする等の業務Trade-offをOwnerへ提示する。直接越境CUD=0、RunPlan completedは整合性の十分条件ではない。
- **P2 / #1124/#4510/#4522**：Accepted Period/Revision、Plan/Commit Attempt、実Reader/Effectと未証明Edgeを問い・Disclosureに相対化して投影する。図上にないWriter/秘密情報を不存在としない。

**停止線**：本節は既存Source/Issueの静的照合と単一SQLite DBの局所条件付きUPDATEの検算。新TEI Go/CLI・実PostgreSQL/Period実Provider／二Process、MFT/外部Effect、Current Grant、業務Owner受理、exact-head CIは **NOT RUN / NOT ACCEPTED**。新PeriodRevision Core型、Period/Recovery Engine、Saga/Distributed Tx、Hostの採用は提案しない。

# QX-14追補：SQL解析の「入力帰属」と演算・Data Contextへ渡せる証拠

- 状態：**Draft設計・合成Go負例の実証候補**。正式Protocol/API/ADRの採用ではない
- 参照main：`f84ca05936370e431d4ad27922eca29cbcc7d41b`（2026-11-03 JST時点の分岐元）
- Owner：[#2387 Query構造Explain](https://github.com/hat47x/tei-platform/issues/2387)。Function Data Contextは[#2506](https://github.com/hat47x/tei-platform/issues/2506)、反応的な再評価は[#2582](https://github.com/hat47x/tei-platform/issues/2582)。演算数学則は#1063、開示と安全な公開は#2516、実Effect/Host権限は#1174
- 補助：既存[Draft #2388 Query計画](https://github.com/hat47x/tei-platform/pull/2388) §7.4 QX-14・§7.10/§7.22/§7.27。現mainに統合済み[#2454](https://github.com/hat47x/tei-platform/pull/2454)・[#2471](https://github.com/hat47x/tei-platform/pull/2471)・[#2577](https://github.com/hat47x/tei-platform/pull/2577)・[#2634](https://github.com/hat47x/tei-platform/pull/2634)の責務と重複しない

## 1. 今回測る狭い未確認境界

現行`analyzer.Request`には`Query`、`ProtocolVersion`、`Operation`、`Dialect`、`Source`、`Schema`がある。`RunPythonWorker`はProcessへRequest JSONを送る。現行`analyzer.Result`はQuery/Protocolと解析構造を持つが、**結果がどのSQL本文・Schema・Dialect・Worker実体を使用したかをResult内で照合できない**。現在の実装は意図どおり**Query IDとProtocol versionの一致だけ**を検査する。

[Go回帰テスト `client_qx14_origin_test.go`](../../internal/query/analyzer/client_qx14_origin_test.go)は既存`TestSQLAnalyzerWorkerHelper`を実Processとして呼び、同じQuery ID/Protocol versionのもとで以下のRequestを別々に入力する。ワーカー代役のhelperはSource/Schema/Dialectを解析しないため、**同じResultが返る**。これは現行wireの情報不足を可視化する合成負例であり、実SQLGlot workerの不正動作や製品不具合を証明するものではない。

| 条件変更 | Request JSONハッシュ | Resultだけで区別 | 区別すべきもの |
| --- | --- | --- | --- |
| SQLのWHERE `active=1` → `active=0` | 変わる | できない | 原SQL byte、著者mode、展開済みSQL（存在する場合）、Query revision |
| Schemaの列集合を変更 | 変わる | できない | Schema入力の具体的な対象・版・名前解決条件 |
| DialectをPostgreSQL→SQLiteに変更 | 変わる | できない | SQL解析対象方言・演算意味Profile |
| Workerのファイル/実行環境を変更（未実行） | Requestが同じでも変わらない可能性 | できない | 実際に呼んだworker artifact / tool versionと観測 |

**重要な区別**：(a)旧Requestのdigestを付けた旧Resultを、新Requestに再束縛することは、Request/Resultをひとつの実行票として保持すれば拒否できる。(b)しかし**新Requestで呼び出した相手が古いResultをそのまま返した場合**、単なるRequest保存やResultのQuery ID echoではその虚偽を検出できない。Worker identity/digestや自動echoを追加しても、それだけで実行された解析内容・業務意味・Schema正しさを証明しない。

## 2. 導入候補：非正本のAnalysis Invocation Evidence（正式APIではない）

意味モデルに新しい正本を作らず、解析1呼出しの**非正本の観測票**として、既存Source Index/実行記録にぶら下げる方向を試す。最低限、次の観測を独立に保持する。

| 段階 | 固定する対象 | **証拠の限界** |
| --- | --- | --- |
| Authoring | Query canonical ID・著者mode・SQL source revision・原SQL byte digest。2waySQL等の展開がある場合は原文/展開後を別欄 | 内容一致は業務同義性ではない。図/モデル著者方式ではモデルが正本 |
| Request | Query ID、Protocol、Operation、Source exact byte、Dialect、Schema入力/名前解決環境のfingerprint、送信JSON digest | 同一RequestであることはWorkerが同じ意味に解析する保証ではない |
| Worker invocation | 実行したworker artifactと必要な依存、process/実行環境、開始/完了・Attempt/入力と結果の対応 | 呼出し記録はworkerの真正性、解析器の正しさ、相手プロセスによる虚偽応答を自動保証しない |
| Result | 結果のraw byte/digest、Response Query/Protocol/Status、診断、`DerivedIR`への派生が使用した同じResult | Result byte一致は同じsourceを解析した保証ではない |
| Consumption | 現在のQuery/source/schema/dialectと要求時の対象を比較。不一致は古い解析説明を現在の説明として扱わず再評価候補 | 現在の業務Meaning/Read Cohort/Providerの取得完全性/Effectまで有効とはならない |

`analysisRunId`などを設ける場合も、実行の相関子であり暗号学的な証明や新Registryではない。`requestFingerprint`は`Source`/Dialect/Schemaを含むRequestの正規の**バイト列**から算出する。Schemaが外部実DBの状態であれば、Requestに渡したSchemaと実Provider/DBの実効Schema・search_path/RLS/権限はさらに別の観測。QX-18〜20の例外・開示境界を迂回しない。

## 3. 検証と実装順（新Gateなし）

1. **M0**：Goの既存サブプロセスhelperで同Query ID/ProtocolのSQL/Schema/Dialect変更を独立に送信し、Request digestは異なるがResultは同じであることを実測。GitHub Actionsのexact head/実Go toolchainの実行結果を別に確認。失敗は治療するためのGate追加ではなく設計上の証拠。
2. **M1**：呼出し票と`DeriveFromAnalysis`を**同一の呼び出し単位**で保持し、旧Result+旧Requestを新Queryへ消費する場合に拒否する局所Consumer試験。対象Queryの「表示」だけを再利用する場合も現在性/開示資格を再確認。新`Proof Registry`を導入せず既存ツール実行の記録を活用。
3. **M2**：正規のPython SQL解析workerを、具体的に固定したSQL/Dialect/Schemaを入力として実呼び出しし、意図的に違うSchema・方言・同じIDの別SQLで別分析を比較する。worker/version、Source Index、実DB観測、独立SQLite/Postgres oracleを混同しない。
4. **M3**：Go`DerivedIR`のScope/Operation/Output Fieldに、QX-27で確認された**非出力Predicate・Sort・相関条件**の必要な依存を供給できるか確認。#2506へRequired Read Slice、#2582へRelation Path/否定集合/ABAの再評価契約、SHO/TAIへ権限判定後の**read-onlyな説明Projection**を渡す。

## 4. Owner判断・未解決

- RequestとResultを束ねる実装場所（既存の呼出し口／Editor Intelligenceのセッション／CLI読取投影）を比較し、現行Goの`analyzer.Result`を無条件に拡張しない。
- 実効DB名前解決（Schema/search_path/RLS/View/Function権限）と解析器へ渡すSchemaContextは同義ではない。
- 同じ表示値でもQuery Source/Relation Path/Predicate/Read Cohortが変われば、古いClaimを現在の意味へ自動転用しない。
- Raw SQL/Field/Schemaの解析情報は機密を含み得る。SHO、TAI/AIハーネス、CUI/Modelerへの表示は既存開示資格/用途相対Projectionで制限。説明可能性は開示許可を与えない。
- Authored SQLから一般的Data Operator/DSLへ変換しても、局所Fixtureの結果一致は多Target全般の意味同値・丸め/NULL・Java全class閉包の受理にはならない（QX-22〜30）。

**現時点のテスト**は、同一IDを返す合成Go worker helperとの実Subprocess通信。実SQLGlot worker／実Provider/DB／Schema・Dialectの実意味／SHO/Host/Owner acceptanceは未確認。正式なResult wire拡張、Source Index改変、SQL正本の置換は行わない。

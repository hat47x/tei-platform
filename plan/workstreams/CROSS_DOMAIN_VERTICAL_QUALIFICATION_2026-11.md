# 横断機能を一つの実使用経路で検証する計画（2026-11）

- **状態**：既存計画を接続する実証順序の提案。新たな製品Gate、ABI、意味正本、Runtime Host、Resolver、Registry、Schedulerは定義しない。
- **位置づけ**：各領域の詳細設計は既存文書と担当Issueが正本。この文書は重複実装を減らすために、**同一の使用経路を何がどこまで証明したか**を横断して確認する。
- **基準日**：2026-11-08。本文の検証例は未実行の提案であり、既存の限定ReferenceやCI成功を製品全体の完了へ拡張しない。
- **参照**：[製品計画](../MASTER_PLAN.md)、[Core以外Plugin化の実装計画](../design/PLUGIN_FIRST_TOOLING_RUNTIME_DOGFOOD_2026-11.md)、[操作面の導出](../design/OPERATION_SURFACE_DESIGN_2026-11.md)、[性能・AI最適化](../design/AI_PERFORMANCE_TUNING_OBSERVABILITY_2026-11.md)、[API失敗とClient反応](../design/HTTP_RESPONSE_DECISION_GRAPH_DESIGN.md)、[増分再利用の実証](INCREMENTAL_REUSE_EVIDENCE_EXECUTION_PROTOCOL_2026-11.md)。

## 1. 統合判断

TEIが共通化するのは画面やDomain固有の工程列ではなく、**意味・対象・型・権限・副作用・証拠・変更時の検査**である。Coreは小さく保ち、その外側のLanguage／Operator／Data Provider／Generator／CLI・Studio Adapter／標準表現等を契約に基づく選択可能なPluginとして扱う。標準Rendererを利用しないHeadlessなアプリケーションも、標準的な利用形態とする。

一つの操作に対して、少なくとも次を**別の事実**として扱う。

1. **意味・契約が記述された**：Source occurrence、Accepted Meaningと未受理Candidate、型／NULL／単位／粒度、OperationとScopeを追える。
2. **実構成で選択された**：Capability候補一覧とEffective Selection／Lock／Plugin実装版／Providerの適合を区別する。
3. **実Buildに含まれた**：実入力閉包、Toolchain、Artifact Revision、リンク後の物理混入・非干渉を検査する。
4. **実Consumerから呼び出された**：通常CLI、Studio、Web等のどの入口かを示し、実際に選択Providerへ到達した証跡を持つ。
5. **結果・副作用を独立に観測した**：Result、外部Read、Commit/Receipt、拒否、Commit Unknownを分離する。
6. **現在のConsumerに適格である**：Source・Plugin・Grant・Disclosure・Provider状態・必要Artifact集合の鮮度を評価する。
7. **独立Expectedで受け入れた**：正常系だけでなく部分取得、誤選択、失効、失敗・再試行を照合する。Ownerによる業務意味受理とは別に記録する。

これは新しい七段階Gateではなく、既存BOM／Source Evidence／Consumer資格を同じ試行に対応付ける**説明用の観点**である。「宣言された」「選択された」「CIで限定PASS」「実Providerで作用した」を一つのDoneへ潰さない。

## 2. 共通の実行・図面契約

標準／案件カスタム／合成後の実効状態を、**型付きOperation Lifecycle Graph**として比較する。Intent → Prepare → Validate → Authorize → Execute → Observe → Resultは説明用の基本系列であり、固定実行順序や一律の同期処理を強制しない。GraphのSlot／Hook／Guard／分岐／サブフロー／Provider交換は構造化したPatchとして表し、型・権限・副作用・受理境界を検証してから生成に使う。図面のレイアウト変更は意味変更ではない。

**Interaction Flow／Business Flow／Adaptive Flowを分ける。** AIチャットのStreaming・Tool Call・Approval・Branch、音声入力の無音・中断・再確認、独自B2C FrontendのJourney／ExperimentはInteraction側の異なる事例であり、Business Effectを自動的に承認しない。標準表現型はFunction内の表現インスタンスに配置し、Data topologyとPresentation topologyのBinding、読み書きの可逆性、情報の欠落を明示する。編集できない投影をEditableと表示しない。

図面からの生成は説明用ダイアグラムのHTML化ではない。最低限、`Source→Typed Graph→Validated Patch→Effective Graph→Target Artifact→実Consumer結果`を追い、Source・Graph・生成器・Targetの版を束縛する。図面の全容性はRequired／Loaded／Unresolved／Hiddenを区別し、未取得部分を勝手に補わない。

## 3. 重複を避ける最小縦断試験

| 優先 | 単一の試行で確かめること | 必須の負例・独立Expected | 既存担当 |
| --- | --- | --- | --- |
| **P0-A** | 通常配布形態のGo CLIから一OperationをInspect→Resolve→Invokeし、明示選択したPlugin B／Provider Bへ実際に到達する。通常のBuildと実呼出しを同一実装版で照合する。 | Bを選択したのにAを呼ぶ、暗黙fallback、未選択Pluginの実混入、契約だけ合うが実体不在、権限不一致。 | #1251／#951／#1016／#3267 |
| **P0-B** | P0-Aと同じSource・選択・Attemptを使い、CLIのCatalog/Inspect/Trace/Validate、BOM、Artifactを突き合わせる。 | 0件／未読込Source／同名別内容／実消費Source差、選択前の候補と実選択の混同。 | #2595／#2600／#1180／#2514 |
| **P1-A** | 一つのFunctionからQuery→Dialect→実DB Provider→Result→Contextへ進むread-only試行。既存SQLの直接利用も許容し、TEI専用SQLへの強制変換はしない。 | NULL／欠損／0件／partial／permission-denied、JOIN・非存在条件などの否定依存、Provider切替後の実Readの誤認。 | #1126／#2582／#2622 |
| **P1-B** | 既存Capabilityの検証→認可→書込み→外部Observation／Receiptを一Attemptで分離して記録する。 | business reject、technical failure、commit unknown、重複要求、CAS競合、前段成功のみでCommit成功を主張する誤り。 | #1174／#1129／#3274 |
| **P1-C** | 同一Meaningからread-only Model Map、三視点図、CRUD Matrix、標準/カスタム/実効Lifecycleを導出し、変更案のImpact→Accept→再生成を限定一件で照合する。 | 未受理関係の自動採用、同label別identity、多重辺消失、レイアウト変更をMeaning更新扱い、非可逆投影からの無断書戻し。 | #1124／#1125／#1126／#1127／#2497／#3503 |
| **P2-A** | 実Consumerを変えても意味・Operationの契約が同一に扱えるか、完全Headlessの独自UIとStream/Approval型Interactionで圧力試験する。 | 必須TEI DOM・Web Runtime・追加hopの強制、UI操作の成功を業務効果と混同、UI変更に伴う無関係なMeaning再受理。 | #3509／#3503／#1158 |
| **P2-B** | P0/P1で資格が確認できた同じ入力の変更系列をCold／Warm／Incrementalで反復し、提案→Preview→Verify→Accept→Apply→再観測を比較する。 | Plugin候補だけの変化と実Selection変化、A→B→A、誤再利用、不要な全面失効、速度改善したがGrant／Source資格が古い。 | #2600／#3271／#2018 |

**先行順**：P0-Aの「選択B→実B呼出し」が最初の停止条件。P0-Bは同時並行でよい。P1-A/B/Cは独立して進められるが、選択・実入力・実Consumerの証拠が欠けた部分は `unknown` のまま保持する。P2は比較用の異質事例であり、P0を妨げない。

## 4. 同じ試行を結び付ける記録

個別のOwnerが持つ既存ログ／BOM／Trace／Artifactを、次の項目で対応付ける。新たな中央台帳や第二正本は不要。

| 確認の単位 | 記録する最低限の情報 |
| --- | --- |
| 試行・入力 | Attempt/Scenario ID、独立Expected版、World／Source Scope、Source occurrenceとRevision、実Readの完全部分 |
| 解決・Build | Contract/Capability/Plugin/Providerと実選択の版、Build／Toolchain、実入力閉包、Artifact集合のRevision |
| 動作・作用 | 入口Consumer、Operation/Function/Query、実Invoke先、認可Scope、Result、外部Read／Effect／Receiptの観測時点 |
| 現在資格 | Source fresh/stale/unavailable/partial等、Consumer必要member、Grant、Disclosure、適用できる対象範囲と未確認範囲 |
| 判定 | 実測PASS／FAIL／NOT RUN／UNKNOWN、機能・意味・観測・性能の別判定、Ownerの受理状態 |

外部情報が移動して内容が同一の場合にも、計算済みMeaningの再利用可能性とSource位置・引用・現在権限の再資格化を別に扱う。Artifact bytes一致は実Providerの選択やOwnerによる受理を証明しない。現存するResultと、現在そのResultを利用できるかも独立に判定する。

## 5. 既存計画への組込みと採用判断

- **#1119の主線を変更しない**。横断試験は既存の三視点ローコード一往復に重ねるが、Gate A/B/Cの過去のReady判定を変更せず、新たな必須Gateも加えない。
- **Plugin方式**は#951／#1016／#1251／#3267の既存Core／Capability／Composition／BOMへ返す。静的合成・動的ロード・外部Processの選択は、Go単一バイナリと外部Targetの要求に照らして実測後に決める。全部同じロード方式にはしない。
- **図面／操作／Studio**は#1124〜#1127／#3503の既存Design・Target Consumerに返す。DiagramとRuntime Graphは必ず構造上の同一性と変更差分を検査し、画面を新たな業務意味の正本にしない。
- **Query／Cache／最適化**は既存DB/Source/Provider・BuildのOwnerへ返す。Optimizing Pluginは結果の正しさを先に証明し、その後にp50/p95、再Read数、Cold/Warm差、実CPU/待ち時間を比較する。
- **B2C／チャット等の異質Consumer**は標準表現型の強制対象ではない。共通契約を使えることと、専用Frontend・外部Providerを自由に選べることを共に守る。

**改定条件**：P0-Aで実Providerへ到達しない、P1で独立Expectedが失われる、P1-CでGraphから生成できず別の意味定義が必要になる、P2で独自UIに不要なRuntime負荷が生じる、またはReuseで誤資格付与が見つかる場合は、該当Ownerの契約・要求・選択方式へ戻す。証拠が揃わない領域を新Componentの導入で覆い隠さない。

**この更新の限界**：ここで列挙したP0〜P2の同一試行、実DB Effect、図面駆動Target生成、異質Consumer完全受領、定量的な高速化、Ownerの意味受理は**今回未実行**。既存文書の限定試験成功と、今後成立させるべき製品受入を混同しない。

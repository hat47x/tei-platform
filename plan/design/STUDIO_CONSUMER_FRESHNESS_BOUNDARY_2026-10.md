# Studio Consumerの応答世代・選択同一性・Source鮮度の境界（#1124）

- 状態: **未採用のConsumer設計補足**。実装・製品Gate・実行テストの成功を意味しない。
- 対象: [#1124](https://github.com/hat47x/tei-platform/issues/1124)のread-only Consumer、特に第4追補(f)。[#1120](https://github.com/hat47x/tei-platform/issues/1120)の型付き参照、[#267](https://github.com/hat47x/tei-platform/issues/267)のShared Acceptance、[#1121](https://github.com/hat47x/tei-platform/issues/1121)/[#1126](https://github.com/hat47x/tei-platform/issues/1126)/[#1127](https://github.com/hat47x/tei-platform/issues/1127)の実現準備判定を再実装しない。
- 方針: [Draft PR #2226](https://github.com/hat47x/tei-platform/pull/2226)のTEI独自共通Host非必須方針を前提とする。PR/ADRの統合済み状態は主張しない。
- 静的照合基準: main `b36ffdfd609bbc73bb1be93867576280c913f9d9`（2026-10-26）。本書の概念型は公開APIではない。

## 既存の保証（重複実装しない）

| 層 | 既存実装・確認済みの限定保証 | 非保証 |
| --- | --- | --- |
| Workbenchの応答 | `reference/tei-studio-workbench/responses.ts` の `createResponseScope`: contextとlane別sequenceが現行の成功・失敗だけをpublishする。`main.ts::loadWorkspace` は `responses.reset()`、`inspect` は `responses.run("inspection", ...)` を使用 | Sourceの現在版、Scope付き同一性、処理の中止は判定しない |
| Graph描画 | `reference/tei-studio-graph/graph.ts` の描画`revision`と`generation !== revision`は旧DOMイベントを無効化する。`graph.test.mjs` に対応試験がある | 同じnode IDが次の描画でも存在すればselectionは保持される。意味的対象の一致は判定しない |
| Catalog / Inspect | `presentation.ts::readCatalog` は`modelRevision`、`worldFingerprint`、`sourceIndexFingerprint`を受け取り、`readInspection` は後二者をCatalogと比較して`STALE_MODEL_PROJECTION`を返す | CatalogとInspectの双方が古い同一Snapshotでも現在のSourceと等しいとは限らない。`subjectKey([kind,id])`はScopeを含まない |
| Studio Authoring | `reference/studio-authoring/bridge.mjs::projectWorld` は`modelRevision`を返し、`realizationReadiness.status="not-verified"`を維持する | Accepted Source・再投影・Validation表示だけから実Provider/DB/Invocationがreadyとはならない |
| Acceptance / Source | Shared AcceptanceのDecisionは`baseModelRevision`/`modelRevisionBefore`/`modelRevisionAfter`を記録。Source Indexは管理対象のpath＋byte fingerprintからrevisionを算出 | Decision記録のみで現在の受理資格、原子的な複数ファイルSnapshot、現在の実行許可を証明しない |

既存の`internal/sourceacceptance`には特定Readerから現在のAccepted Sourceを照合するReferenceがあるが、Readerの信頼性・権限はその提供側に依存する。これを非Host Consumerの汎用Acceptance Authorityとみなしたり、追加の共通Host導入を要求したりしない。

## 三つの独立した失効条件

1. **Response世代**: 要求時context＋lane sequenceが現在値と違えば旧成功・失敗を表示へ反映しない。Workspace再読込や同laneの後続要求で実装済み。
2. **Graph選択の意味**: 描画のepoch、node ID→元の`modelref.Key{Scope,Subject}`の同一Snapshot内の対応、Graph/Model versionをConsumerが保持・再確認する。旧mapまたはscope未確定なら表示候補と診断は保ち、scopeなしInspectやSource locatorの確定に進まない。単なるkind＋ID、label、namespaceフィルタ、node ID再利用でScopeを推定しない。
3. **Source/Accepted/realization鮮度**: 現在のSource`modelRevision`、参照対象のScope付きidentity、Acceptedの現在性、評価時のSource版とTarget/Provider/DB/preflightなどの根拠を独立に照合する。版未照合または根拠不明なら`ready`ではなく`unknown/not-verified`を表示する。Source改訂の検知時には、後着成功だけでなく**表示済み**旧schema・旧readinessも失効させる。

これらは一つのglobal epochを追加する理由ではない。Graph描画だけの変更でSourceの意味・依存Evidenceが不変なら、すべての実現準備を一律に失効させる必要はない。一方で、Source変更をConsumerが検知するまでの鮮度をepochだけで保証することもできない。

### Graph更新拒否→旧描画維持→選択map失効（#1124 [コメント #6426483819](https://github.com/hat47x/tei-platform/issues/1124#issuecomment-6426483819) の追加差分、未実装）

**正常なG1→G2置換で同じ表示IDを再利用する負例とは別**に、G2が`missing_endpoint`等で`graph.update(G2)`から拒否された場合を扱う。`reference/tei-studio-graph/graph.ts:284-316`は`normalizeGraph`成功後だけSVGと描画revisionを置換するため、失敗時にはG1のSVG・選択が残り、G1の可視nodeイベントは依然として有効である。成功後の旧DOMイベント拒否では、このケースを防げない。現行Workbench `main.ts:50-57`の`onSelect(nodeId)`は現在の`entries`を検索するため、G1の表示node IDをR11/G2側の候補や別scopeへ誤対応させ得る（Consumerでの発生は未実証）。

**Consumerの追加採用条件（提案）**：G2用の`view-node ID → qualified Key`対応表M2とGraph/Catalog snapshotは、描画更新が成功し対応するSnapshotを確定するまで**仮置き**し、G1のDOMへ公開しない。更新が拒否されたらM2をcommitしない。さらに、Source R10→R11等によってG1のSnapshotが現在のCatalog/Source/Graphと一致しなくなったときは、**旧描画が残っても**G1用の確定選択map M1を失効し、表示選択からのread-only InspectとSource locator確定を停止する。可視の旧G1はstale/unresolved診断付きの参照表示にとどめる。G1のSnapshotがなお現在と検証できる場合まで無条件にM1を失効させる必要はないが、拒否されたG2のM2でG1のKを再解釈してはならない。

**既存証拠と限界**：`reference/tei-studio-graph/graph.test.mjs:139-159`は不正更新時の旧SVG・旧選択維持と、**成功更新後**の旧DOMイベント無効化を個別確認する。更新拒否時のG1 DOMとM1/M2の版対応、クリック後の`InspectSubject`呼出し0、Source locator確定0はConsumer横断の未実装・未実行条件。Graph部品の仕様変更や新Core/Host APIを要求しない。

### Accept結果によるSource改訂とDecision記録失敗の合流（#1124 [コメント #6470746209](https://github.com/hat47x/tei-platform/issues/1124#issuecomment-6470746209)、未実装）

これはSource改訂後の旧応答失効とDecision記録失敗を**同じConsumer上で同時に扱う**設計負例である。固定main `b301d39604c79c1128e7031b788cf51a73e3ebc1` の [`TestAcceptReportsSourceChangedWhenDecisionLedgerWriteFails`](https://github.com/hat47x/tei-platform/blob/b301d39604c79c1128e7031b788cf51a73e3ebc1/tools/reference-shared-acceptance/ledger_root_test.go#L95) は、CLIが非ゼロ終了でも `decision=accepted` と `sourceChanged=true` を返し、Decision ledger書込み失敗を `DECISION_RECORD_FAILED` として示し、`recordPath`を空に保つ条件を確認する。これはSource受理が成立した結果の応答であり、Decision ledger永続化の証拠ではない。事実の読み取り根拠は[#267の既存実装記録](https://github.com/hat47x/tei-platform/issues/267#issuecomment-6247518560)と[#1124の結合負例](https://github.com/hat47x/tei-platform/issues/1124#issuecomment-6470746209)を参照する。

**現在の境界**：Workbenchの`responses.ts`は表示応答の世代と順序を制御し、`presentation.ts::readInspection`は応答fingerprintを保持中Catalogと照合する。一方、現行WorkbenchにAccept UIやSource watcherはなく、この非ゼロAccept結果を受けて表示を失効させるConsumer接続も確認されていない。したがって以下は将来のtyped結果Consumerに対する設計候補であり、実装済みの動作ではない。

**受入候補**：
1. R10のschema/readiness表示とR10 Inspect応答が保留中に、同じWorkspaceでAcceptを実行し、SourceがR11へ変わる一方でDecision ledger記録が失敗する。非ゼロ終了コードだけからAccept拒否・Source不変を推定せず、typed結果の`accepted + sourceChanged`をSource変更の観測として扱う。
2. R10の表示済みschema/readinessを直ちに`再照合が必要`へ失効させ、保留中のR10 Inspect成功も表示を上書きしない。Decision記録が失敗していても、Acceptを自動再試行しない。
3. Sourceの現在版とDecision ledgerを独立に再読取する。再読取まで`ready`や永続化済みを表示せず、再読取がR12を示す場合はR12を現行として扱う。結果本文を解釈できない非ゼロ応答など効果不明の場合も、自動再試行せず状態を再観測する。

このケースを既存負例Aと別々の成功例へ分解して代用しない。Source受理、Decision記録、UI freshness、Ownerによる業務意味の受理、Target readinessは別の結果である。Shared Acceptanceの既存契約や新しいHost/Acceptance APIを追加する提案ではない。テスト・GUI・Owner受理は**NOT RUN**。

## 一つの競合例（未実行の受入負例）

| 時点 | 事実 | 誤表示を招く経路 |
| --- | --- | --- |
| t0 | Workspace W、Source R10、Graph G1、UI context=4。R10に関するreadiness要求AとInspect要求Bが保留。以前のR10 `ready`表示が残ると仮定 | 応答のrequest epochは4 |
| t1 | Shared Acceptance後、SourceはR11へ改訂。Graph G2を再投影し、表示node ID `K`の意味的ScopeがAからBへ変わるか曖昧になる | Workspace IDが不変で`responses.reset()`やGraph選択map失効が連動しなければ、UIのcontextは4 |
| t2 | R10のA/B成功が後着 | epoch一致なら応答を採用し得る。古いCatalogとInspectのfingerprint同士が一致してもR11現在性は証明しない |
| t3 | 旧Graph由来の選択`K`、旧schema、旧readinessをG2/R11へ持ち越す | 新しいScopeが未解決なのに「実行可能」と誤表示し得る |

このシーケンスは**条件付きの設計負例**であり、現行Studioが`ready=true`を返しているという主張ではない。現行`projectWorld`は明示的に`not-verified`である。

## 最小のConsumer採用・拒否契約（設計候補）

`ready`を表示してよいのは、(a)応答世代が現行、(b)評価時Source`modelRevision`が現在確認したSource版と一致、(c)Scope付きSubjectと対応Graph/Catalog snapshotが有効かつ非曖昧、(d)Acceptedの現在性が適切な既存の判定元で確認済み、(e)必要なrealization Evidenceが当該版・Target/Provider条件に適用可能、と確認できたときだけ。単なる`ok:true`、Catalog/Inspect指紋一致、Source Accepted、成功した再投影を`ready`へ昇格させない。

Source変更を検知した時点で旧表示のcurrent-readyを取り消し、再評価中は`unknown/not-verified`に戻す。新しい成功が届いても旧版・旧Graph map・曖昧Scope・未照合Evidenceなら採用しない。read-only候補・診断表示は維持する。ユーザー操作の実行許可・effect・current authorityは#1121/#1127等が別途判断し、UI上の`ready`表示から推定しない。

**Source atomicity**: `modelRevision`やSource Index fingerprintの前後一致は、分離したWorld/Index/Graph/Accepted/Provider観測を原子的に取得した証拠ではない。時点間の変更・復元（ABA）、別Sourceの割込み、読み込み中の競合は未解決になり得る。Consumerは提供されたsnapshot整合性の証拠範囲を表示し、不明ならcurrentと断定しない。正式なSource conflict拒否は既存Shared Acceptance、Provider readinessは該当Ownerへ委譲する。

## 最小回帰案と証拠状態

- **負例A（現在未実行）**: W固定でR10→R11変更、R10のready成功とInspect成功を後着。R10の表示済みreadyは失効、両旧成功は現行表示を上書きせず、R11の独立な根拠が無ければ`unknown/not-verified`を維持。
- **負例B（現在未実行）**: G1のnode ID `K`とG2の`K`が別Scopeの対象。同じview-node IDがあっても旧選択mapは失効し、Scopeが一意になるまでscopeなしInspect・Source locator確定をしない。
- **負例C（現在未実行、#6426483819）**: R10/G1で表示node ID `K`→Scope oneのM1が成立。SourceをR11へ更新し、K→Scope twoのM2を準備したG2を`missing_endpoint`で更新拒否。G1のSVGとKが残ること、M2が公開されずM1もR11では失効すること、残ったKのclick後も`InspectSubject` spy呼出し0・Source locator確定0・stale診断保持を同じConsumer試験でassertする。Source/Graphが変わらない検証済みG1ではM1を不必要に無効化しない。
- **負例D（現在未実装・未実行、#6470746209）**: R10の表示済みschema/readinessと保留中Inspectがある状態で、Accept後にSourceがR11へ変わりDecision ledgerだけ記録失敗となる。非ゼロ終了でも`accepted + sourceChanged`結果が得られた場合はR10表示を失効し、遅延R10成功を拒否する。Sourceとledgerの再読取前に`ready`・記録済みを表示せず、Accept自動再試行は0回。
- **既存証拠**: `responses.workspace-switch.test.mjs` は旧Catalog/Validation/XEAD応答（成功・失敗）の拒否、`graph.test.mjs` は旧DOMイベント拒否、`presentation.fingerprint-contract.test.mjs` はCatalog/Inspect fingerprint契約、`bridge.test.mjs`/`e2e.mjs` はSource Acceptance後も`not-verified`表示を維持する条件を、それぞれ**試験コードとして**持つ。今回の新しい負例をPASSと扱わない。
- **未証明**: 非Host ConsumerのSource変更通知、readiness Evidence schema/発行者/失効条件、原子的な複数資源Snapshot、実Provider/DB/Invocationの現在性、実UIの負例E2E。

本書は#1124第3・第4追補および既存設計の境界を整理してConsumerに固有な拒否規則だけを追加する。Core契約、受理機構、実現準備判定、Hostを再実装しない。

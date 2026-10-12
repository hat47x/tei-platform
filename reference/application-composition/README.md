# 案件単位のアプリケーション構成 — 検証資料

TEIの既存資産を案件単位で組み合わせる際、どの情報を再利用でき、どこに未解決の接続判断が残るかを調べます。完成した構成プロファイルや、新しい依存解決器を提供するものではありません。

[設計の整理](../../plan/design/PROJECT_PLUGIN_COMPOSITION_DESIGN.md)と[管理課題 #1016](https://github.com/hat47x/tei-platform/issues/1016)を正本として、実例を次の順に確認します。

- [E1：既存資産の所在](source-inventory.md) — `order_entry` とSUIの意味、実行、UI、保存、環境設定を区別する。
- [E1：既知のソース観測器](e1-observe.mjs) — ソースの存在と文字列の変化だけを検知する。
- [E2：二種類の変更を比較](E2_CHANGE_SCOPE.md) — UI見出しと開発環境のDB接続先の変更を読み取り専用で提案・比較する。
- [E2：変更提案の検査器](e2-change-probe.mjs)／[合成試験](e2-change-probe.test.mjs) — 許されない上書きや解決不能な参照を未解決として返す。
- [E3：追加で必要な選択と再検証範囲](E3_REQUALIFICATION_AND_RESIDUAL_CHOICES.md) — 既存Sourceを二重管理せず、選択・参照の不足と更新影響を絞る。
- [E3：対象Source単位の再検証試作](e3-requalification.mjs)／[合成試験](e3-requalification.test.mjs) — Source指紋を比較し、無関係な利用先を一律失効させず、不明な互換性を未解決として残す。
- [E4：Source Indexと構成の接続境界](E4_SOURCE_INDEX_JOIN.md) — Source Indexが示す所在・内容と、別途必要な意味依存辺を区別する。
- [E4：Subject単位の変更検査](e4-index-scope.mjs)／[合成試験](e4-index-scope.test.mjs) — 既存IndexのJSON形状に沿って、呼び出し側が選択したSubjectのみを再確認する。依存辺を自動導出した証拠ではない。
- [E5：解析済みApplicationの依存とSource Indexの接続](E5_PARSED_WORLD_SOURCE_JOIN.md) — Goの既存Loader／型付きWorldを使い、明示Validation・Database ContractとPatternからData／Relation／Capability／Queryへの直接参照を照合する。
- [E5：読み取り専用Go検証器](e5-world-index.go)／[Go回帰テスト](e5-world-index_test.go) — 実Workspaceでの動作と全体Healthは未実施。合成fixtureのテストは追加済みだが実行結果は未取得。
- [E6：解析済み参照とSource Indexの接続](E6_TYPED_INDEX_BRIDGE.md) — E5の部分的な明示依存をE4の既存変更検査へ結び、Source出典と宣言集合の変化を検出する。
- [E6：限定的な変更検査の接着部分](e6-typed-index-bridge.mjs)／[合成試験](e6-typed-index-bridge.test.mjs) — 依存集合変化、指紋取り違え、出典曖昧化を未解決にする。実Workspaceの統合実行は未実施。
- [E7：同じ読取りで得た診断とSource Indexを比較](E7_PAIRED_OBSERVATION.md) — E5のオプトイン出力をE6へ渡す。Sourceの真正性や全依存閉包を保証するものではない。
- [E7：限定Evidenceの比較器](e7-paired-evidence.mjs)／[合成試験](e7-paired-evidence.test.mjs) — 形式偽装、依存追加、指紋不一致を診断。実Workspaceでの統合実行は未実施。
- [E7a：選択対象だけのSource Index証跡](E7A_MINIMAL_DISCLOSURE.md) — 対象外のWorkspace Source pathを省き、定義出典の欠落・重複を拒否する。
- [E8b/E8c：Capability入力のType・Dataと変更影響の限定追跡](E8_TYPED_DEPENDENCY_EDGE_CONTRACT.md)／[診断・変更比較器](e8-typed-closure.mjs)／[合成テスト](e8-typed-closure.test.mjs) — 実受注明細のCapability入力がSemantic Dataを参照することを反映。E4の既存Index検査でSource証跡を確認し、2時点で入力定義Source変更だけを比較する。全依存ClosureやSemantic Validationではない。
- [E8：SUI Native Canvasの第二Consumer評価](E8_SUI_NATIVE_CONSUMER_EVIDENCE.md) — 既存Native操作と未接続のTEI Runtime Actionを区別する。
- [E9：Contract Query入力属性のSource追跡](E9_CONTRACT_QUERY_INPUT_SOURCE.md) — 実際の顧客参照Patternが参照する意味属性の定義変更を再検証に含める。Goテスト・実Workspaceは未実行。
- [E10：構成Profileを採用すべきかの判定](E10_COMPOSITION_ADOPTION_DECISION.md) — 受注明細・SUIの二Consumerで、業務意味・Provider/Target・UI・Runtime権限の検証を分離し、Profileへ追加する最少情報と採否の実証条件を定める。
- [E11：限定Source変更と型付き影響候補のレビュー接続](E11_TYPED_IMPACT_REVIEW_GATE.md) — E7の部分的なSource検査結果と#1120候補の型付きImpactを重複Graphなしで束ねる。E5の明示的なPattern所属で判定（命名prefix推論を廃止）。E7と#1120 Go Projectionの実JSON形状を結ぶ入口を追加。ただし同一snapshot未証明なら`unresolved`で止める。E11合成テスト30件はV8互換試験で確認、Go/JS実結合は未実施。
- [E12：型付き影響レビューを実際のJSONファイルから確認するCLI](E12_PAIRED_SOURCE_TYPED_REVIEW.md) — E11のGo形式JSON用判定器を単一の正本とし、E12は入力サイズ・終了コード・出力制限のみ担当。Node標準テスト13件。Goの同一snapshot結合とAuthorityは未実証。
- [E13：E5と既存Derived Indexを一度の解析済みWorldから取得](E13_SAME_WORLD_DERIVED_OBSERVATION.md) — E5にオプトインの`-include-direct-derived`を追加し、既存`index.Build(loaded.World)`から必要な直接参照5種類だけを同時出力。Go試験7件を追加（**未実行**）。[厳格なローカル検証](qualify-e13-local.sh)はbash構文のみ確認済み。#1120 Go Impact/同一Source bytes保証は未結合。
- [E14：受注明細の三視点直接参照とSourceの実Workspace確認](E14_CROSS_VIEW_SOURCE_WITNESS.md) — 既存Derived IndexのApplication→Pattern→Relation→Dataと出典の一意性をGoテストに固定。Go4関数・Source負例5サブケース（**未実行**）。E13の未解決時の追加開示抑止についてGoテスト2関数を追加。#1120 Impactへの統合は別Gate。
- [E15：実Goの型付き変更影響とE12 CLIの結合確認](E15_MODELREF_E12_INTEGRATION_GATE.md) — 関連機構はmainに統合済み。Node.js子プロセスを使うためビルドタグ付きの追加試験を維持する。実行・PASSは未確認。
- [E16：統合済みGo機構による実Workspaceの変更影響回帰検査](E16_POSTMERGE_TYPED_IMPACT.md)／[Goテスト](e16_postmerge_modelref_test.go) — 追加のビルドタグやNode.jsなしで、既存の型付き索引・差分・影響解析を受注明細の実Worldへ適用する。**テスト2件登録・実行未確認**。
- [生成AI向けの要件整理](requirements-prompt.md) — 既存の定義を参照し、案件固有の判断を抽出する。

これらは、案件構成をTEIの受理済み意味や実行権限へ昇格させないための小規模検証です。実行確認の範囲と未実施事項は各資料に明記しています。


## 局所検証の範囲（2026-10-10）

E4・E6・E7・E7a・E8・E11の**GitHubに登録された原文12ファイル**（本体6＋テスト6）を取り込み、V8インメモリの互換テストランナーで計**119件成功・0件失敗**を確認した。内訳はE4が12件、E6が25件、E7が11件、E7aが14件、E8が27件、E11が30件。ES Module importと`node:test`、`assert`、`structuredClone`を試験用に代替したため、**Node.js標準の`node --test`で119件を実行したとの主張ではない**。E12の実ファイルCLI試験はこの119件に含まない。

すべて合成入力の限定検証であり、E5のGo実装、実`workspace.Load`・Source Indexバイト一致、#1120のGo型付きImpact、実SUI Native接続、Node CLIの真正なFile I/O、trusted exact-head full Repository Healthは未実証。合成PASSをmain統合資格に流用しない。Blob・ケースと対象関数の詳細は各E4〜E11資料と[PR #1051](https://github.com/hat47x/tei-platform/pull/1051)を参照。

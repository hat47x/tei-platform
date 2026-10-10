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
- [E8：SUI Native Canvasの第二Consumer評価](E8_SUI_NATIVE_CONSUMER_EVIDENCE.md) — 既存Native操作と未接続のTEI Runtime Actionを区別する。
- [E9：Contract Query入力属性のSource追跡](E9_CONTRACT_QUERY_INPUT_SOURCE.md) — 実際の顧客参照Patternが参照する意味属性の定義変更を再検証に含める。Goテスト・実Workspaceは未実行。
- [E10：構成Profileを採用すべきかの判定](E10_COMPOSITION_ADOPTION_DECISION.md) — 受注明細・SUIの二Consumerで、業務意味・Provider/Target・UI・Runtime権限の検証を分離し、Profileへ追加する最少情報と採否の実証条件を定める。
- [E11：限定Source変更と型付き影響候補のレビュー接続](E11_TYPED_IMPACT_REVIEW_GATE.md) — E7の部分的なSource検査結果と#1120候補の型付きImpactを重複Graphなしで束ねる。実Go/JS結合は未実施。E5の明示的なPattern所属で判定（命名prefix推論を廃止）。初回12件はNodeで実行、更新後18件はV8互換試験のみ。
- [E12：E7のSource報告をE11型付きImpactレビューへ結ぶ](E12_PAIRED_SOURCE_TYPED_REVIEW.md) — E7の実比較結果と同じCurrent ReportからApplicationのPattern所有情報を取り出す。15件のNode標準テスト。Goの同一snapshot結合とAuthorityは未実証。
- [生成AI向けの要件整理](requirements-prompt.md) — 既存の定義を参照し、案件固有の判断を抽出する。

これらは、案件構成をTEIの受理済み意味や実行権限へ昇格させないための小規模検証です。実行確認の範囲と未実施事項は各資料に明記しています。

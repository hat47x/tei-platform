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
- [生成AI向けの要件整理](requirements-prompt.md) — 既存の定義を参照し、案件固有の判断を抽出する。

これらは、案件構成をTEIの受理済み意味や実行権限へ昇格させないための小規模検証です。実行確認の範囲と未実施事項は各資料に明記しています。

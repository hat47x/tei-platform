# セルフビルド実現の計画～設計の完了記録

本記録は、TEI セルフビルドを実現するための**計画～設計作業**が完了したことをまとめる。

## 到達した状態

**能力宣言 → catalog → 計画 → 成果物 → L2-bのClaim** が、製品の契約を一つも変えずに、二つのホストで通る状態を設計・実装・観測した。

| 層 | 状態 |
| --- | --- |
| L1 再現 | 同じsourceから同じバイナリが出る（linux/amd64, windows/amd64） |
| L2-a 受け渡し | entryが宣言を引き渡し、builderが読んだbyteとentryの保持byteが一致 |
| L2-b 解釈 | entry自身が宣言を解釈して計画を導く（in-process + childの両経路） |
| L2-b (c) | 同じplanを二回buildして同じartifactが出る（`-rebuild-check`） |
| L2-b Claim | 五条件の同時成立を述語とし、`l2b.claimed`で述べる |
| L3 起動 | stage1の`tei`バイナリがstage2（child builder）を動かす |
| 鎖 | 宣言→catalog→plan→artifact→Claimまでが一本で通り、CIで毎回回る |
| 隔離 | 計画がmodule contextを名指すとき、**隔離ビルドが既定** |
| 双方向一致 | ビルドが読むものが計画に載り、計画が名指すものをビルドが読む |
| 物性値 | 27.68 MiB / 起動7.9 ms / RSS 12.2 MiB（`tei version`） |

## 設計文書の状態

- `plan/design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md`: §4.1〜§4.54まで記載
- §7の終了条件: **47/48チェック済み**、未チェック1項目（`darwin`）
- Evidence: 複数の記録と索引が `plan/evidence/README.md` に登録済み

## 完了した設計判断

| 判断 | 結果 |
| --- | --- |
| catalogの出所 | K-3: 呼び手が名指し + digest必須 |
| catalogの場所 | 製品の外（Workspaceもplugin構文も変えない） |
| 既定経路 | in-process据え置き（名指しcatalogを読まない） |
| `-rebuild-check`既定 | catalogを名指しした試行に限って既定化 |
| L2-b Claim既定 | 立てない（呼び手がin-process + `-rebuild-check`を明示） |
| 能力宣言の版付け | 付けない（配るのはcatalog、catalogは`formatVersion`を持つ） |
| 隔離 | 既定（planがmodule contextを名指すとき） |
| 双方向一致 | 計画⇄ビルドの両方向を検査 |

## 残る項目（完了条件の範囲外）

| 項目 | 理由 |
| --- | --- |
| `darwin`での観測 | このホストにmacOS実行環境が無い（クロスコンパイルは通るが走らせられない） |
| 交換・停止手順の観測 | 別の担当・別の観測が必要 |
| 独立Consumerでの適合 | 第二のconsumerが現れた時点で設計 |
| G4 判定 | `UNQUALIFIED`のまま（材料は揃ったが判定ではない） |

## 主要なマイルストーン

- PR #4254: CIで全鎖の観測を回す (§4.53) → merged
- PR #4263: CIでの実測を記録 (105秒) → merged
- PR #4301: G4物性値を測る (§4.54) → merged
- PR #4339: 実現状況の総括を現在のmainに更新 → open（証拠は既にmainに載っている）

## 検証

- CI (`reference-self-build`) で全鎖が毎回回る
- `go test ./tools/repository-plan-check` がPASS
- `doc-reference-probe` 未解決参照0件
- 文字化け検査 該当なし

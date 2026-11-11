# TEI Studio から SHO 文書編集部品を利用する形態（C-2）

- **状態**: `代行決定済み`（2026-11-12）。実測に基づいて**子プロセス起動を維持**する。
- 正本: [TEI_SHO_DOCUMENT_DRAFT_NEXT_STAGES_2026-11.md](https://github.com/hat47x/tei-platform/blob/main/plan/design/TEI_SHO_DOCUMENT_DRAFT_NEXT_STAGES_2026-11.md) の C-2。ここには複製しない。

## 実測（2026-11-12・Windows・Node 24.19.0）

`scripts/tei-sho-embed-measure.cjs`（SHO repository）で、実 `bin/sho-doc.cjs` を
5回ずつ起動して測った。

| 操作 | 中央値 | 最小 | 最大 |
| --- | --- | --- | --- |
| `open`（draft→文書生成・readback検証込み） | 162 ms | 161 ms | 194 ms |
| `resume`（保存済み文書の読戻し） | 132 ms | 121 ms | 177 ms |

**数値目標は先に置いていない。** この実測は「どれを選ぶかの材料」であり、
目標を決めてから測ったのではない。

## 決定: 子プロセス起動を維持する

**候補**:

| 形態 | 評価 |
| --- | --- |
| **子プロセス起動**（現状） | 実測 130–190 ms/呼出。**採用** |
| Wails 資産への同梱 | `bin/sho-doc.cjs` は CommonJS の `.cjs` で、WebView 内で評価される JS ではない。資産にしても起動は `node` が行うので**spawn コストは同じ**。梱包の複雑さだけが増える |
| 外部プロセス（stdio の長寿命サーバー） | **プロトコルが存在しない**。必要を測る前に作るのは推測で補完になる |

**根拠**:
1. 公開面は `open` と `resume` の**読み取りのみ**（`save` は出さない）。1操作あたり
   130–190 ms は、文書を開く・保存後に読戻すという粒度では許容できる。
2. keystroke 単位の操作は公開していないので、この遅さが問題になる経路がない。
3. Wails 資産化は同じ spawn コストで梱包だけが増える（ `.cjs` は WebView では動かない）。
4. 外部プロセスはプロトコルが無い。作る前に必要を測る。

**撤回条件**: (a) 公開面に保存や編集のような高頻度操作が増え、実測がそれを許さない、
(b) SHO 側に stdio プロトコルが実装され、同じAttemptで比較実測される、
(c) Wails が `.cjs` を WebView 内で直接評価できるようになる。いずれかをADRで通す。

## 失敗分類（6種・安定）

`cmd/tei-studio/sho_document.go` が区別する。測定では「コマンドが無い」場合は
node 自身のメッセージで非0終了する（`sho-nonzero-exit`）ことを確認した——
**空の文書を「空のDraft」と読ませない**ため、失敗時は文書状態を返さない。

`sho-not-configured` / `sho-invalid-argument` / `sho-spawn-failed` /
`sho-timeout` / `sho-nonzero-exit` / `sho-receipt-unreadable`

## 未測定

- Wails 資産としての**ロードコストと失敗形状**（資産パイプラインはあるが未測定）
- 外部プロセス（プロトコルが無い）
- 配布（インストーラ・bundle サイズ）
- Linux / macOS での実測（この実測は Windows のみ）

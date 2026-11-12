# SHO 保存物（`*.tei.md`）の TEI 側読み取り設計（C-5）

- **状態**: `設計のみ`（2026-11-12）。**実装は Parser（[#3346](https://github.com/hat47x/tei-platform/issues/3346)）の完了後**。
- 正本: [TEI_SHO_DOCUMENT_DRAFT_NEXT_STAGES_2026-11.md](https://github.com/hat47x/tei-platform/blob/main/plan/design/TEI_SHO_DOCUMENT_DRAFT_NEXT_STAGES_2026-11.md) の C-5。ここには複製しない。

## 位置づけ

SHO が `bin/sho-doc.cjs save` で書く文書は、occurrence marker で囲まれた
生成領域と、著者が自由に書いた領域が混在する1つの Markdown file である。
TEI 側がそれを読む経路は**まだ存在しない**。`tei-md-authoring-parser-is-unbound-on-current-main`
という authorityLimit が、全ての Draft に常置されている。

この文書は「Parser が完了するまでの間、着手できる範囲」を固定する。実装は含まない。

## 読み取り側が答えるべき4つ（Independent Expected の候補）

| # | 問い | 答えにならないもの |
| --- | --- | --- |
| R-01 | この文書は**どのDraft**から作られたか（`draftId` と model reference） | 「同じ内容なので同じDraft」——内容一致は identity ではない |
| R-02 | 各 occurrence の**現在の状態**は何か（intact／著者編集／marker 不整合） | 空欄や欠落を「未編集」と読む |
| R-03 | 著者が書いた領域はどこか（occurrence marker の外側） | occurrence 内の著者編集を「生成領域」として読む |
| R-04 | モデルはこの Draft の時点から動いたか（reconcile の結果） | Draft の model reference を「現在のモデル」と読む |

## 境界（Parser 完了後も変えない）

1. **読み取りは読取りだけ**。`*.tei.md` を読んでも TEI モデルは変わらない。文書→モデルの経路は作らない（C-3）。
2. **著者文は上書きしない**。再投影は許可リストのみ（C-4 の代行決定）。
3. **未解析部分は unknown として残す**。Parser が読めない構文を「空」や「デフォルト」にしない。
4. **marker の byte 不一致は改変として報告する**。`authorEditedGeneratedBlock` を修復しない。

## Parser（#3346）へ期待する入出力契約

Parser は「SHO が書いた occurrence marker 付き Markdown」を読み、
以下を返す。**判断はしない**（どちらを採るかは Consumer/Owner）。

- `draftRef`: header の `draftId`・`documentKind`・`model-revision`・`occurrences` 数
- `occurrences[]`: `occurrenceId`・`kind`・status・byte 範囲
- `authorRegions[]`: occurrence marker の外側の byte 範囲
- `unknownRegions[]`: Parser が解釈できなかった byte 範囲（**空にしてはいけない**）
- `findings[]`: marker 不整合の診断（未終端・孤立 end・id 不一致）

## 未決（Owner判断）

- SHO 側の `OccurrenceStatus` の5値と Parser の findings をどう対応させるか
- `unknownRegions` を Consumer にどう見せるか（禁止して読みを止めるか、警告して続けるか）
- `*.tei.md` を Workspace の Source root に置くべきか、別の場所に置くか

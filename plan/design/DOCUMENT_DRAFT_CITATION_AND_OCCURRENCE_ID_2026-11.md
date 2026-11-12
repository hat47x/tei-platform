# Draft 引用の解決と Occurrence ID の永続性（代行決定）

- **状態**: `代行決定済み`（2026-11-12）。**既定案をその採用定**する。
- 正本: [TEI_SHO_DOCUMENT_DRAFT_NEXT_STAGES_2026-11.md](https://github.com/hat47x/tei-platform/blob/main/plan/design/TEI_SHO_DOCUMENT_DRAFT_NEXT_STAGES_2026-11.md) の Owner判断一覧。ここには複製しない。

## 決定 1: Draft 引用は `recorded-not-resolved` のまま

**決定**: Draft の引用 Occurrence は Source identity・Revision・range を**記録するだけ**。取得・検証・relink は行わない。解決は Consumer の明示操作にする。

**根拠**:
1. 解決には artifact の実バイト読みが必要で、それは documentdraft の「Workspace を読む」境界を超える。参照解決をここへ入れると、読み取り投影が副作用を持つ。
2. SHO repository には `src/g2-references.cjs` の引用Register（意図したSource・名目解決先・取得byte・宣言/観測fingerprint・Review現在性を別欄で持つ）が既にある。Draft の recorded-not-resolved は**別のモード**であり、Register の代替ではない。2つを統合すると「どちらが正本か」が曖昧になる。
3. 未取得・未検証を「取得済み」と見せないためには、記録と解決を分けるのが最も単純で確認しやすい。

**撤回条件**: (a) Consumer が recorded だけでは作業できず、同じ Attempt で解決できることを示す、(b) Register と Draft 引用の統合を ADR で通す。いずれも現時点で未実施。

## 決定 2: Occurrence ID は snapshot-local のまま

**決定**: `occurrenceId` は**一つの読み取り単位（snapshot）内で一意**な識別子。文書をまたいで安定させない。永続化が必要になったら ADR を先に通す。

**根拠**:
1. ID は `occ-<kind>-<連番>` で、投影の順序から導かれる。順序は入力順で決まるので、**入力を変えれば ID の指す内容が変わる**。これを文書をまたいで安定した identity として使うと、同じ ID が別の内容を指す。
2. 内容の同一性は `contentDigest` で判定できる。ID を永続化しなくても「同じ内容か」は判定できるので、永続化の必然性がない。
3. 永続化には subject identity と rule version と generation conditions を束縛した安定キーが必要で、それは contract の変更。contract 変更は ADR で通す。

**撤回条件**: (a) 文書をまたいで occurrence を追う必要が実測で生じる、(b) subject identity と rule version を束縛した安定キーの設計を ADR で通す。

## どちらも「作らない」決定である

2件とも**実装を足さない**決定。理由は「判断を待たずに着手可能な範囲を進め」る方針ではあるが、この2件は着手すると contract が変わる（引用の解決は documentdraft の副作用境界を壊す、ID の永続化は contract の変更）ため、**既定案を採用して確定させる**方が安全。

# 開いた集合を列挙型として書くときの規約（OPEN-01）

親Issue：[#2533](https://github.com/hat47x/tei-platform/issues/2533)。要求ロール（案）：`architect`。状態：**設計案**（採用判断は行っていない）。
研究文書：[TEIの汎用モデリング能力を業務外領域で確かめる限定実証](../../plan/research/TEI_CROSS_DOMAIN_PROBE_CSW_VS_BEAR_2026-11.md) §9 OPEN-01。

## 1. 問題

TEIの`type ... from std.code values { ... }` は**閉じた集合**を宣言します。ところが、対象によっては上流が**実行時に新しい値を宣言できる**ため、集合が閉じていません。

実例（VS Bear Simulator）：`BearTypedDecision.source` は、同期の規則producer（`none`・`typed_wire`・`jev-shaped-rules`・`failure-aware-rules@0.1.0`）だけでなく、**transportが宣言する`backend_id()`**（既定 `async-wire`）も入ります。transportは差し替え可能で、新しいbackendは新しいIDを宣言できます（`cognition/bear/async_wire_backend.gd` の `backend_id()`）。

このとき、TEI側の書き方には3つの選択肢があります。本ファイルは、それぞれの帰結を実例で示し、**暫定案**を記録します。採用判断は行いません。

## 2. 3つの選択肢と帰結

### 選択肢A：閉じた集合として列挙し、未記録の値をerrorにする

```
type DecisionSource from std.code values {
    "none" "typed_wire" "jev-shaped-rules" "failure-aware-rules@0.1.0" "async-wire"
}
```

未記録の値をerrorとして拒否します。

**帰結**：
- 列挙が**意味を持つ**（「この集合の外は不正」と読める）。
- しかし、**正当な新しいbackendが拒否される**。transportを差し替えただけで、記録の読み取りが不合格になる。
- 上流の実装（`backend_id()` が任意の文字列を返せる）と**矛盾する**。上流は閉じていないのに、TEIは閉じたと宣言する。

**この選択肢は、上流が実際に閉じている場合にだけ正しい。** VSベアの`source`については誤りです。

### 選択肢B：記録済みの値を列挙し、未記録の値をwarningにする

```
type DecisionSource from std.code values {
    "none" "typed_wire" "jev-shaped-rules" "failure-aware-rules@0.1.0" "async-wire"
}
// 未記録の値は warning。列挙は「記録済みの集合」であって「閉じた universe」ではない。
```

未記録の値を**warning**として報告し、判断自体は使えるままにします。

**帰結**：
- 正当な新しいbackendが拒否されない。
- 上流の挙動と整合する（`effective_weight()` は producer 名を見ない。`is_valid_at()` も見ない）。
- **列挙の意味が弱まる**。「この集合の外は不正」とは読めなくなる。列挙は「既知の値の一覧」に格下げされる。
- 読み手が「列挙にない＝おかしい」と誤読する危険は残る。

### 選択肢C：列挙型を使わず、`std.text` として書く

```
attribute source type std.text required
```

**帰結**：
- 上流の実装と完全に整合する（任意の文字列）。
- **列挙の利益（既知の値の一覧・表記の揺れの検出）を失う**。
- 未知の producer 名が**まったく目立たない**。warningすら出ない。

## 3. 比較

| 観点 | A（閉じる・error） | B（列挙＋warning） | C（text） |
| --- | --- | --- | --- |
| 上流が閉じている場合の正しさ | ○ | △（緩い） | ×（緩すぎる） |
| 上流が開いている場合の正しさ | **×（正当な値を拒否）** | **○** | ○ |
| 列挙の意味（「外は不正」） | ○ | **×（弱まる）** | × |
| 既知の値の一覧性 | ○ | ○ | × |
| 表記の揺れ・新しい値の可視性 | ○（拒否するので必ず気づく） | ○（warningで気づく） | **×（見えない）** |
| 実装の追加負荷 | 小 | **小〜中**（warning severity の導入） | 最小 |

**読み方**：AとBは「上流が閉じているか開いているか」で優劣が反転します。**上流の性質を先に確認しないと選べません。**

## 4. 上流が閉じているか開いているかをどう判定するか

判定は、上流のコードを読んで次の問いに答えることで行います。

1. その値は**定数**か、それとも**差し替え可能な部品が宣言する**か。
2. 上流に「この集合の外は不正」という**検証**があるか。あれば閉じている。なければ開いている可能性が高い。
3. 上流の検証が**列挙の外を拒否する**か、**既定値へ落とす**か。落とすなら、その集合は「既知の値の一覧」であって universe ではない。

VSベアの`source` への適用：

| 問い | 答え |
| --- | --- |
| 定数か、差し替え可能な部品が宣言するか | `backend_id()` は transport が宣言する。差し替え可能 |
| 「集合の外は不正」という検証があるか | ない。`is_valid_at()` は `source` を見ない |
| 外を拒否するか、既定へ落とすか | どちらもしない（`source` は判定に使われない） |

→ **開いている**。したがって選択肢B（またはC）。

## 5. 暫定案

**上流が開いている場合は選択肢B**（記録済みの値を列挙し、未記録をwarning）を採る。ただし次の3条件を満たすことを条件とする。

1. **列挙の宣言にコメントで「閉じた集合ではない」と書く。** 列挙だけを見た読み手が「外は不正」と読むのを防ぐ。
2. **warningを診断として報告する。** 黙って通さない（選択肢Cとの差）。
3. **warningを実行時に無視できる経路を用意しない。** 研究物としては、warningを受領側が読む前提。製品化する場合は、warningの扱いを決める別の判断が必要。

VSベアのprobeは、この暫定案を実装しています（`bear/contract.go` の `ValidDecisionSources` と `BEAR_DECISION_SOURCE_UNRECORDED`、`bear/workspace/semantic/bear/decision.tei` のコメント）。試験 `TestUnrecordedProducerIsAWarningThatKeepsTheDecisionUsable` が、warning でも判断が使えること（`Weight` / `EncodeWire` が通る）を固定しています。

## 6. この規約が解かないこと

- **warningの製品上の扱い**。製品で warning を「無視してよい」とするか「reviewを要求する」とするかは、別の判断です。本規約は研究物の範囲で止まります。
- **列挙の外の値が増え続ける場合**。warningが常態化すると、警告としての価値がなくなります。値を記録する更新の頻度と責任を決める必要があります。
- **上流が「閉じている」と「開いている」の混在**。一部の値だけが差し替え可能な場合（例：既定の producer は固定だが transport だけ可変）は、AとBのどちらでも表現しきれません。**この混在は現行DSLでは書けません**（#2533 への具体的な不足として記録）。
- **値の意味の同一性**。`async-wire` と `async-wire@2` が同じ producer かは、文字列からは分かりません。producer ID の版管理は上流の責任です。

## 7. 未実施（NOT RUN）

- ADR の起案と採択。
- 製品コード（`internal/semantic`・`internal/validation`）への warning severity の実装。
- #2533 の結論変更。
- 上流が閉じている別の実例での検証（本規約は1例で組み立てている）。

## 8. #2533 へ返す具体的な不足

1. 列挙型は閉じた集合しか書けない。開いた集合を書く手段がない。
2. 「一部だけ差し替え可能」（既定は固定・transportだけ可変）の混在を書く手段がない。
3. warning severity を意味DSLの診断として扱う構文がない（現行の診断は error 中心）。

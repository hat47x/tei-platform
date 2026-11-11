# 検証語彙の設計案：範囲・有限性・列挙値（VOC-01）

親Issue：[#2533](https://github.com/hat47x/tei-platform/issues/2533)。要求ロール（案）：`architect`。状態：**設計案**（採用判断は行っていない）。
研究文書：[TEIの汎用モデリング能力を業務外領域で確かめる限定実証](../../plan/research/TEI_CROSS_DOMAIN_PROBE_CSW_VS_BEAR_2026-11.md) §9 VOC-01。

本ファイルは、TEIの意味DSLに**範囲（range）・有限性（finiteness）・列挙値（enumerated value）**の検証を追加する設計案です。**製品DSLの変更を提案するものではなく、研究上の選択肢を整理します。** 採用判断は行いません。

## 1. 問題

現在のTEIの`validation`は2種類だけです（`internal/syntax/parser.go` 2370行付近）。

| 種別 | キーワード | 意味 |
| --- | --- | --- |
| `positive` | attribute単体 | 値が正（>0）である |
| `relation-unique` | relation + attribute | relation内でtarget attributeが重複しない |

業務外領域の実例では、この2種類では不足します。

### 1.1 VS Bear Simulator：範囲と有限性

`BearTypedDecision` の数値フィールドは次の制約を持ちます。

| フィールド | 制約 | 上流の実装 |
| --- | --- | --- |
| `heading_bias_rad` | ±0.18 | `clamp(value, -0.18, 0.18)` |
| `persistence_bias` | ±0.25 | `clamp(value, -0.25, 0.25)` |
| `brace_bias` | ±0.25 | `clamp(value, -0.25, 0.25)` |
| `uncertainty` | 0..1 | `clamp(value, 0, 1)` |
| `confidence` | 0..1 | `clamp(value, 0, 1)` |
| `ttl_s` | 0.01..0.50 | `clamp(value, 0.01, 0.50)` |
| `full_age_s` | 0..0.10 | `weight = full if age <= 0.10 else 0` |
| `discard_age_s` | 0..0.25 | `weight = 0 if age > 0.25` |

**範囲**：上流は `clamp` で範囲を表現します。TEIの`positive`は「正」だけで「範囲」を書けません。

**有限性**：`full_age_s <= 0.10` と `discard_age_s <= 0.25` は、**上限が有限**であることを要求します。TEIには「有限」を宣言する手段がありません。

### 1.2 CSW：列挙値と有限性

CSWの`Attribution`は4種類の列挙値です。これは`std.code values`で書けます。しかし：

- **列挙値の数が有限である**ことは、TEIの列挙で暗黙に表現されます（宣言が有限だから）。
- **「この4種類以外は禁止」** という意味は、現行DSLでは読めます（valuesの外は受理しない）。
- しかしCSWは「全候補へのラベルや一覧表は要らない」方針なので、**列挙が増える可能性**があります。列挙を固定した瞬間に、この方針と矛盾します。

→ 列挙値と有限性は、**固定したい場合と動的にしたい場合**で扱いが分かれます。

## 2. 3つの検証次元

本設計案は、次を区別します。

| 次元 | 問い | 実例 |
| --- | --- | --- |
| **範囲（Range）** | 値はどの区間に属するか | `0.0 <= heading_bias_rad <= 0.18` |
| **有限性（Finiteness）** | 上限または下限が有限か | `ttl_s <= 0.50`（上限が有限） |
| **列挙値（Enumerated）** | 値は宣言された集合のどれか | `intent ∈ {maintain, observe, reduce, disengage}` |

**重要**：この3つは独立です。列挙値は有限ですが、範囲ではありません。範囲は有限でも列挙値ではありません（`0.0..1.0`は無限の実数を持つ）。設計ではこの独立性を保つ必要があります。

## 3. 設計選択肢

### 選択肢A：既存の`validation`キーワードを増やす

```
validation decision.heading_bias range -0.18..0.18
validation decision.ttl_s finite upper 0.50
validation decision.intent enumerated {maintain, observe, reduce, disengage}
```

**帰結**：
- 既存の構文に追加するため、パーサの変更範囲が小さい。
- キーワードが増える（`range`・`finite`・`enumerated`）。
- **検証語彙がDSLに固定される**。新しい次元（例：時間窓 `time_window`）を追加するたびにパーサを変える必要がある。
- `range` は開区間・閉区間の区別が必要（上流は `clamp` なので閉区間だが、一般には開区間もありうる）。

### 選択肢B：属性レベルで型と制約を統合する

```
attribute heading_bias type std.decimal range -0.18..0.18
attribute ttl_s type std.decimal finite upper 0.50
attribute intent type std.code values {maintain, observe, reduce, disengage}
```

**帰結**：
- 検証が属性の定義に統合される。DSLが**宣言的**になる（「この属性は何を書くか」が1行で読める）。
- 既存の`validation`ブロックは、関係的な制約（`relation-unique`）だけに残る。
- 型と制約が密に結合する。`std.decimal` に `range` が付くのが自然だが、`std.text` に `range` が付かないことも明示的に書ける。
- **属性の数が増えると、宣言が縦に長くなる**。100属性ある型は100行の属性宣言になる。

### 選択肢C：独立した`constraint`ブロックを設ける

```
type Decision from std.code {
    attribute heading_bias type std.decimal
    attribute ttl_s type std.decimal
    attribute intent type std.code values {maintain, observe, reduce, disengage}
}

constraint decision.heading_bias range -0.18..0.18
constraint decision.ttl_s finite upper 0.50
constraint decision.intent enumerated {maintain, observe, reduce, disengage}
```

**帰結**：
- 型宣言と制約が分離される。型は「何を書くか」、constraintは「どのように書くか」。
- 制約がスキーマとは別の次元で増やせる（時間窓・文字列長・パターンなど）。
- **既存の`validation`との住み分けが曖昧**。`validation`も`constraint`も「制約」であり、2つのキーワードが似た意味を持つ。
- 実装が増える（constraintのパース・評価・エラー報告）。

### 選択肢D：値を閉じた型として宣言する（現状の延長）

```
type HeadingBias from std.code values {-0.18, -0.15, ..., 0.15, 0.18}
type TTLSeconds from std.code values {0.01, 0.02, ..., 0.49, 0.50}
type Intent from std.code values {maintain, observe, reduce, disengage}
```

**帰結**：
- 既存のDSLで実現できる。パーサの変更が**不要**。
- **値が離散化される**。`0.18` は書けるが `0.181` は書けない。上流は連続値を`clamp`しているので、表現力が落ちる。
- 列挙が膨大になる（`0.01`刻みのTTLは50値）。可読性が極端に悪い。
- **「範囲」を「列挙」で近似する**。この近似が必要な時点で、DSLに範囲がないことが露呈する。

## 4. 比較

| 観点 | A（validation増設） | B（属性統合） | C（constraint分離） | D（列挙近似） |
| --- | --- | --- | --- | --- |
| パーサ変更範囲 | 小 | 中 | 大 | なし |
| 宣言の読みやすさ | 中（validationが増える） | **○（属性定義が1行で読める）** | ○（分離して読める） | ×（膨大な列挙） |
| 新規次元の追加の容易さ | ×（パーサ変更が必要） | △（型に新しい修飾子が必要） | **○（constraintだけ追加）** | ×（列挙を手で書く） |
| 範囲の表現 | ○ | ○ | ○ | **×（離散近似）** |
| 有限性の表現 | ○ | ○ | ○ | △（上限値が有限であることは示せる） |
| 列挙値の表現 | △（enumeratedキーワードが必要） | **○（valuesで既に書ける）** | ○ | ○ |
| 型と制約の分離 | × | × | **○** | × |
| 既存DSLとの互換 | ○（追加） | △（属性構文が変わる） | △（新キーワード） | **○（そのまま）** |

## 5. 推奨案（暫定）

**B（属性レベルで型と制約を統合）** を暫定案とします。理由：

1. **VS Bearの実例が属性中心**である。`heading_bias_rad`・`persistence_bias`・`brace_bias`・`uncertainty`・`confidence` は、それぞれが「型（`std.decimal`）＋範囲」を持つ。属性単位で書くのが自然。
2. **列挙値は既に属性で書ける**（`type std.code values { ... }`）。`enumerated` キーワードが不要。
3. **範囲と有限性は、型の修飾子として追加する**。たとえば `std.decimal range -0.18..0.18 finite` のように書く。これは属性宣言に直接付き、「この属性は小数で範囲は-0.18から0.18まで、有限」と1行で読める。
4. **constraint分離（C）より実装が小さい**。属性のパース時に型と同時に制約を読むだけで済む。

ただし、Bには次の制約があります。

- **関係的な制約（relation-unique）は別途`validation`に残す**。属性レベルに収まらない制約は、従来通り`validation`ブロックを使う。
- **時間窓などの新しい次元は、型の修飾子を追加する必要がある**。`std.timestamp time_window 2026-01-01..2026-12-31` のような拡張が、型システムに入る。
- **開区間・閉区間の区別**。上流の`clamp`は閉区間だが、一般には開区間も必要。設計は閉区間を既定とし、開区間は明示的に書く（例：`range (-0.18, 0.18)`）。

## 6. 実装の増分（研究物の範囲）

本設計案は**製品DSLの変更を提案しない**が、研究物としての実装負荷を推定します。

| 段階 | 内容 | 負荷 |
| --- | --- | --- |
| **VOC-01a** | パーサに`range`・`finite`キーワードを追加（属性宣言内） | 小（Aに近い） |
| **VOC-01b** | バリデータに範囲チェック・有限性チェックを追加 | 小〜中（既存の`positive`を拡張） |
| **VOC-01c** | 列挙値の検査を`values`から独立した診断として扱う | 中（OPEN-01のwarning設計と統合） |
| **VOC-01d** | Workspaceの`tei validate -json`に検証結果を出力 | 中（診断構造の拡張） |

**製品化の判断は別**です。研究物としては、VOC-01a〜01dの順に少量ずつ実装し、CSW・VSクマのprobeで「範囲・有限性・列挙値の検証が書ける」ことを確認する段階を想定します。

## 7. 本Issueへ返す具体的な不足

#2533 のDSLは、次の検証次元を書けません。

1. **範囲（range）を書けない。** 最小値と最大値のペアを宣言する手段が無い。`positive`は「正」の特別な場合に過ぎない。
2. **有限性（finiteness）を書けない。** 上限または下限が有限であることを宣言する手段が無い。列挙値の有限性は`values`で表現されるが、数値の有限性は書けない。
3. **列挙値は書けるが、開いた列挙を warning で扱う手段がない。** `values`は閉じた集合として振る舞う。開いた集合を書くには、OPEN-01の設計が必要。
4. **型と検証が分離している。** `validation`は属性・関係の外に置かれ、属性宣言と一緒に読めない。宣言的な読みやすさが低い。

## 8. 未実施（NOT RUN）

- 製品DSLの変更。
- パーサ・バリデータの実装（VOC-01a〜01dは研究物の推定）。
- #2533 の結論変更。
- 上流が閉じている別の実例での検証（本設計案はVS Bearの連続値clampを主な実例としている）。

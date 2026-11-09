# TEI の標準意味型は TEI で宣言されていない（2026-11-09、第25ラウンド）

- 対象: [#2620](https://github.com/hat47x/tei-platform/issues/2620)（Everything is Data: 標準データ構造への意味役割付与・自己記述）、[#2619](https://github.com/hat47x/tei-platform/issues/2619)、[#2831](https://github.com/hat47x/tei-platform/issues/2831)（Self-build）。
- 実行日: 2026-11-09（Asia/Tokyo）。
- 状態: **read-only の実測。欠陥の起票ではなく、設計判断に必要な材料の提供。**

## 1. 目的

本ドッグフーディングの目的は「生成AIがTEIを用いてTEI自身の設計を効果的に行えること」の検証である。ここまでの24ラウンドは CLI の整合性を測ってきたが、このラウンドは**TEI 自身が TEI で書かれているか**を測る。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `49224178312a320ddd9337435d7dba92f3b646eb`（`origin/main`） |
| CLI実行物 | `sha256:9ad0d9f78b37571e594741c9e11b71ff87e354f32c91f4dd6f85e06d27dc0c6` |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.27.3 linux/amd64`（gofmt は 1.26.10） |

## 3. 観測

### 3.1 `std.*` はモデルの要素だが、Source を持たない

`tei model catalog -workspace reference/workspace` は `std.binary`〜`std.time` の12型を返す。

```json
{"subject": {"kind": "semantic-type", "id": "std.binary"},
 "prelude": true, "parent": {"kind": "semantic", "id": "std"},
 "symbol": "binary", "detail": "primitive", "sourceStatus": "not-indexed"}
```

`prelude: true` と `sourceStatus: "not-indexed"` が同時に付く。**モデルの要素だが、どの Source にも由来しない**という表明である。

### 3.2 `inspect` は解決できるが、根拠が空

```bash
tei inspect -workspace reference/workspace -subject-kind semantic-type -subject-id std.text
```

```text
sourceStatus: not-indexed
sources: {"subject": {...std.text}, "artifacts": [], "references": []}
canonical: {"id":"std.text","symbol":"text","kind":"primitive"}
```

対照: ファイル宣言された `sales.order` は `sourceStatus: "indexed"` で、`artifacts` に `semantic/sales/order.tei` とその指紋が入る。

### 3.3 Source Index に `std` は1件も無い

`tei source-index -workspace reference/workspace` は参照106件・artifact 54件を返すが、**`std.*` への参照は0件**、`std` を定義する artifact も0件。

### 3.4 定義は Go にある

`internal/stdlib/semantic.go` の `SemanticNamespace()` が12型を Go のリテラルで作る。コメント自身が過渡期であることを認めている（「A later package/registry implementation can replace this prelude without changing the normalized IR」）。

### 3.5 `std` は**使えるが宣言できない**

`std` 名前空間を `.tei` で宣言しようとすると拒否される。

```tei
namespace std

type text from std.text
```

```text
SEMANTIC_TYPE_ALIAS_CYCLE    semantic type alias cycle reaches std.text -> std.text
DUPLICATE_SEMANTIC_ELEMENT   semantic element std.text is declared more than once as type
DUPLICATE_SEMANTIC_NAMESPACE duplicate semantic namespace std
```

一方、別名前空間からの利用は通る。

```tei
namespace mine

type text_t from std.text
```

→ `ok: true`。

`std` を特別扱いしている箇所は少なくとも2つある。`internal/validation/validation.go:1185`（`std` を除いて「空でない」を判定）と `internal/operations/declaration_counts.go:38`（`std` を宣言数から除外）。**`std` は「モデルに元から在るもの」として扱われている。**

## 4. 合格条件が CLI だけで測れることの確認

`std` を `.tei` にしたときに何が変わるべきかを、宣言可能な名前空間で先に実演した。

```tei
namespace mine

type text_t from std.text
type money_t from std.money
```

同じ Workspace の catalog に両方が並ぶ。

| subject | prelude | sourceStatus |
| --- | --- | --- |
| `mine` | （なし） | indexed |
| `std` | **true** | **not-indexed** |
| `mine.text_t` | （なし） | indexed |
| `std.text` | **true** | **not-indexed** |

`inspect` も `source-index` も `mine.text_t` は根拠付きで返す（`artifacts: ["semantic/mine.tei"]`、参照3件）。**`std` を `.tei` にすれば、この差がそのまま埋まる。** 新しい API も検証機構も要らない。

## 5. 何が問題か

- **「TEI で TEI を設計する」が標準型の層で止まる。** 利用者の Data は `.tei` で宣言され `inspect`・`trace`・`source-index` で根拠を辿れるが、`std.*` は同じ操作で根拠が空になる。同じ「型」という概念が、由来を持つものと持たないものに分かれている。
- **標準型を変える経路が CLI に無い。** `std` の型を増やす・意味を変えるには Go を書き換える。利用者が `std.money` の意味を確かめようとしても、`inspect` は `canonical` の3欄（id・symbol・kind）しか返さない。
- **`prelude` と `sourceStatus: not-indexed` の併存が、その状態を正規のものとして固定している。** 意図的な表明だが、#2620 の自己記述が達成されたなら `prelude` は「in-process の代替」ではなく「同梱の `.tei`」になるはず。

## 6. 提案（判断は #2620 の Owner）

`std` 名前空間を同梱の `.tei` として宣言し、loader がそれを読む。合格条件は CLI の出力2つで測れる。

1. `tei inspect ... -subject-id std.text` の `sourceStatus` が `indexed` になり、`artifacts` に stdlib の `.tei` が入る。
2. `tei source-index -workspace <ws>` の `std.*` への参照が0件でなくなる。

移行は「Go の定義を残したまま `.tei` を併置し、両者の一致を確かめる」段階を挟むのが安全である。一致は `model catalog` の `prelude` 件数と `.tei` の宣言件数の突き合わせで測れる。

**注意**: `std` の特別扱い（上記2箇所）と `internal/stdlib/semantic.go` の二重定義が問題になる。`.tei` を正本にするなら、これら3箇所を同時に扱う必要がある。

## 7. 記録の位置付け

これは欠陥の起票ではない。**欠陥ではないが、目的の達成を妨げている構造**であり、判断は #2620 の Owner に属する。私は材料（現状の測定値・合格条件・移行時の注意点）を提示するに留め、勝手に `std` を `.tei` 化していない。

## 8. 読み取れないこと（NOT RUN）

- `std` を `.tei` にした場合の実際の挙動（loader の探索順・`std` 特別扱い2箇所との干渉・指紋への影響）。
- `reference/plugin-self-build/workspace` は2プラグイン・1環境のみで、data 宣言が0件だった。**TEI が TEI を設計している実例としては薄い**（plugin の供給経路の検証が主目的と見られる）。
- 実DB・実Provider・実Host、HTTP・Desktop、意味の受理・権限・配備。

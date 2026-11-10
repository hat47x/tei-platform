# std への宣言が「prelude」と誤判定され、件数からも消えていた（2026-11-09、第26ラウンド）

- 対象: [#3851](https://github.com/hat47x/tei-platform/issues/3851)、[#3347](https://github.com/hat47x/tei-platform/issues/3347)（`prelude` 欄の導入元）、[#2620](https://github.com/hat47x/tei-platform/issues/2620)、[#3419](https://github.com/hat47x/tei-platform/issues/3419)。
- 実行日: 2026-11-09（Asia/Tokyo）。
- 状態: **3つの矛盾を1つの原因（名前の前方一致）に対して修正**。

## 1. 目的

25ラウンドで「`std` を `.tei` にできるか」の移行コストを見積もる過程で、`std` の特別扱いが**名前の前方一致**に依存していることに気づいた。利用者が `std` に型を足した場合に何が起きるかを測った。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `fe1286421`（`origin/main`） |
| CLI実行物（修正前） | `sha256:68507f658303bb4461f3b99d3949870d493d29638344986563a389569e8bf845` |
| CLI実行物（修正後） | `sha256:9431e8416743f09c0b1dd9dc342be7c1c530b33da8c5713aa5c66091c3a4e007` |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.27.3 linux/amd64`（gofmt は 1.26.10） |

再現する Workspace:

```tei
namespace std

type email from std.text
```

## 3. 観測（修正前）— 3つの矛盾

### 3.1 名前空間の重複エラーが誤り

```json
{"code":"DUPLICATE_SEMANTIC_NAMESPACE","severity":"error","message":"duplicate semantic namespace std"}
```

catalog も `std` を2件、**両方 `ambiguous: true`・`sourceStatus: "indexed"`** で返した。prelude 自身が `indexed` になるのは矛盾で（他の12型は `not-indexed`）、`std` 名前空間の項目だけが利用者のファイルに索引付けられていた。

### 3.2 利用者が宣言した型が `prelude: true`

```text
std.binary     prelude=True  sourceStatus=not-indexed
std.email      prelude=True  sourceStatus=indexed      ← 利用者が宣言した型
std.text       prelude=True  sourceStatus=not-indexed
```

契約文書 §59 は `prelude:true` を「プロセス内の標準意味型であり、**Workspaceが宣言したものではない**」と定義している。`docs/getting-started.md:160` も「`prelude`が無い要素が、Workspace自身の宣言です」と書く。その規則では `std.email` の由来を読み手が判別できない。

### 3.3 件数が0のまま、警告も出る

```json
{"namespaces":0,"types":0,...,"total":0}
{"code":"WORKSPACE_DECLARES_NO_MODEL_ELEMENTS","severity":"warning",
 "message":"Workspace declares no model elements; only the standard semantic prelude is loaded"}
```

一方 `inspect -subject-id std.email` は `sourceStatus: "indexed"`・`artifacts: ["semantic/extra.tei"]` を返す。**同じ Workspace について CLI が2つの矛盾した答えを返していた。**

## 4. 原因

`std` の特別扱いが「名前の前方一致」に依存していた。

| 箇所 | 判定 |
| --- | --- |
| `internal/operations/model_catalog.go:73` `preludeSubject` | `s.ID == "std" \|\| strings.HasPrefix(s.ID, "std.")` |
| `internal/operations/declaration_counts.go:38` | `ns.ID == "std"` で名前空間ごと除外 |
| `internal/validation/validation.go:1185` | `ns.ID != "std"` で空判定から除外 |
| `internal/workspace/duplicate_semantic_sources.go` `standardSemanticSubject` | **prelude が実際に持つ型だけを照合（正しい）** |

さらに、loader が prelude の `std` を先に append したあと、利用者の `std` を**別の名前空間として追加**していた（`byNamespace` が空から始まっていた）。これが `DUPLICATE_SEMANTIC_NAMESPACE` と `ambiguous: true` の直接の原因。

## 5. 修正

**「prelude である」ことを名前ではなく同一性で判定する。**

- `internal/operations/model_catalog.go`: `isPreludeType(id)` を追加し、`preludeSubject` がそれを使う。`std` 名前空間そのものだけは常に prelude。
- `internal/operations/declaration_counts.go`: `std` 名前空間を丸ごと飛ばすのをやめ、**prelude が持つ型だけを差し引く**。何も足していない prelude 名前空間は数えない。
- `internal/validation/validation.go`: `hasNonPreludeDeclaration(ns)` を追加し、「空でない」判定が prelude 外の宣言を数える。
- `internal/workspace/loader.go`: `byNamespace` を既存の `World.Semantics` から初期化し、利用者の `std` 宣言が prelude の名前空間に**合流**するようにした（重複名前空間を作らない）。

## 6. 修正後の実測

```text
semantic       std            prelude=True  src=not-indexed     ← 1件だけ、not-indexed
semantic-type  std.email      prelude=None  src=indexed         ← 利用者の宣言
semantic-type  std.text       prelude=True  src=not-indexed     ← prelude
declarationCounts: {"namespaces":1,"types":1,...,"total":2}
validate: rc=0 ok=True （診断なし）
```

対照:

- prelude だけの Workspace は `total:0` のまま（退行なし）。
- `reference/workspace` は `total:71`・`types:9`・`namespaces:4` で不変。

## 7. 試験

`internal/operations/prelude_identity_test.go`。

| 試験 | 内容 |
| --- | --- |
| `TestStandardNamespaceIsNotAPreludeMark` | `std` と `std.text` は prelude、`std.email` は違う |
| `TestDeclarationCountsIncludeTypesAddedToStd` | prelude のみは0件、`std.email` 追加で `types:1`・`total:2` |
| `TestIsPreludeTypeUsesIdentity` | `std.text` は真、`std.email` は偽 |

わざと壊した2種:

| # | 変えた箇所 | 結果 |
| --- | --- | --- |
| M1 | `preludeSubject` が `std.*` をすべて prelude とする | FAIL: `std.email was declared by the Workspace` |
| M2 | `declaration_counts` が `std` 名前空間を丸ごと飛ばす | FAIL: `types=0, want 1` |

`go test ./internal/operations ./internal/validation ./internal/workspace ./cmd/tei -count=1` はすべて ok（2.6s / 0.1s / 2.9s / 105.1s）。gofmt・vet の指摘なし。

## 8. 互換性

**意図的な挙動変更を含む。**

- `std` に型を足した Workspace の `declarationCounts` が 0 から実際の件数に変わる（正しい方向）。
- 同じ Workspace の `validate` が `error` から成功に変わる（正しい方向）。
- prelude の12型と `std` 名前空間の `prelude: true` は不変。`reference/workspace` の出力も不変。

## 9. 読み取れないこと（NOT RUN）

- `std` を `.tei` にした場合の loader の探索順・指紋への影響（25ラウンドの提案そのもの）。今回は「利用者が `std` に足す」経路だけを直した。
- `std` の型を**上書き**しようとした場合（`type text from std.text` のような自己参照）の診断。今回は新規追加のみ。
- 実DB・実Provider・実Host、HTTP・Desktop、意味の受理・権限・配備。

## 10. 位置付け

25ラウンドで見つけた「`std` は Go にある」という構造の、**利用者側から見た症状**を直した。`std` に型を足すという自然な拡張が、矛盾した診断（error＋「何も宣言していない」＋`prelude:true`）を伴わずに通るようになった。

`std` 自体を `.tei` にする判断（#2620）は未着手のまま。ただし今回の修正で「prelude と利用者の宣言を同一性で区別する」規則が3箇所に入ったので、`std` を `.tei` にしても同じ規則で扱える。

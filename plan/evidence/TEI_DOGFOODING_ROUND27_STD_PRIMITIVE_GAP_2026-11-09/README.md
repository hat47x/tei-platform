# std を .tei にできない理由: DSL に primitive を書く構文が無い（2026-11-09、第27ラウンド）

- 対象: [#2620](https://github.com/hat47x/tei-platform/issues/2620)（Everything is Data）、[#2619](https://github.com/hat47x/tei-platform/issues/2619)、[#3851](https://github.com/hat47x/tei-platform/issues/3851)（第26ラウンドで解消）。
- 前段: [第25ラウンド](../TEI_DOGFOODING_ROUND25_STDLIB_SELF_DESCRIPTION_2026-11-09/README.md)で「`std` は Go にある」ことを実測し、「`std` を `.tei` にすれば合格条件が満たせる」と提案した。
- 実行日: 2026-11-09（Asia/Tokyo）。
- 状態: **read-only の実測。前ラウンドの提案に、実装上の障害があることを特定した。**

## 1. 目的

25ラウンドの提案「`std` を同梱の `.tei` として宣言し、loader がそれを読む」が、**いまの実装で可能か**を確かめる。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `91b380d483a88e5f3c3ed522361b6f3e79bb0f0c`（`origin/main`） |
| CLI実行物 | `sha256:4a0abb8cd993b697832a9ad4e13e92867e27059be194a9530a9ad627b020f9c6` |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.27.3 linux/amd64`（gofmt は 1.26.10） |

## 3. 観測

### 3.1 `worldFingerprint` は prelude を含む

loader が prelude を注入する行を外して（実験のみ、workspace は変更せず）同じ Workspace を測った。

| | worldFingerprint |
| --- | --- |
| prelude あり（現状） | `sha256:c92de09f9595b1f46…` |
| prelude なし（実験） | `sha256:8d7e160bee26c2884…` |
| `sourceIndexFingerprint` | **同一** |

`worldFingerprint` は正規化 World の JSON 全体のダイジェスト（`internal/operations/snapshot.go` の `worldFingerprint`）なので、prelude の12型が含まれる。**prelude の定義を変えると、既存の全 Workspace の `worldFingerprint` が変わる。** 継続トークン（`result.snapshotFingerprint`）も World を含むため、保存済みトークンは失効する。

### 3.2 `.tei` は primitive を書けない

`type` 宣言の文法は必ず `from` を要求する（`internal/syntax/parser.go:1902`）。

```go
if _, err := p.expectText("from"); err != nil {
	return semantic.Type{}, err
}
```

直接確かめた。

```tei
namespace mine

type text
```

```text
PARSE_ERROR | expected "from"
```

primitive を書く構文は無い（`internal/syntax/` に `primitive` を扱う箇所が無い）。**`.tei` の `type` はすべて別名**であり、`reference/` 配下の `.tei` が宣言する型20件もすべて `from std.*` の別名である。

### 3.3 したがって `std` を `.tei` にすると循環する

`std.text` 自身を `.tei` で書こうとすると、`from` の先が要る。書けるのは `type text from std.text` だけだが、これは自分自身を指す。

```text
SEMANTIC_TYPE_ALIAS_CYCLE     semantic type alias cycle reaches std.text -> std.text
DUPLICATE_SEMANTIC_ELEMENT    semantic element std.text is declared more than once as type
```

**`std` の12型は「根」であり、根を `from` で書くことはできない。**

## 4. 結論

前ラウンドの提案は、**そのままでは実装できない**。理由は2つで、どちらも独立に効く。

1. **文法**: `.tei` に primitive を宣言する構文が無い。根となる型は DSL の外（Go）でしか定義できない。
2. **指紋**: `worldFingerprint` が World 全体のダイジェストなので、prelude を `.tei` に移すと**全 Workspace の指紋と保存済み継続トークンが変わる**。

したがって `std` を `.tei` にするには、先に次が必要である。

- **primitive を宣言する構文**（例: `type text primitive` のような根の宣言）。これが無い限り、`.tei` に移せるのは「別名」だけで、根は Go に残る。
- **移行時の指紋の扱い**の決定。全 Workspace の `worldFingerprint` が変わることを許容するか、prelude を指紋の対象外にする（=「prelude は World の一部ではない」と決める）か。後者なら `worldFingerprint` の意味が変わるため、契約文書と継続トークンの説明も更新が要る。

## 5. 代替案（私の推奨）

「`std` を `.tei` にする」の目的は #2620 の**自己記述**、すなわち「標準型も CLI で根拠を辿れる」ことである。その目的は、次のどちらでも達成できる。

| 案 | 内容 | 指紋への影響 |
| --- | --- | --- |
| A. `std.def` を同梱し、prelude の**由来**として示す | 12型の意味（`symbol`・`kind`・説明）を `.tei` ではなくデータファイルで持ち、`inspect` がそのパスを `artifacts` に返す | 小（World の構造を変えない） |
| B. primitive 構文を追加し、prelude を `.tei` へ | 根を DSL で書けるようにする | 大（全指紋が変わる） |

案 A は「根拠を辿れる」という合格条件を満たしつつ、`worldFingerprint` を変えない。案 B は本質的だが、primitive 構文の設計・全指紋の移行・契約文書の更新を伴う。

**どちらを選ぶかは #2620 の Owner の判断**であり、私は材料を提示するに留める。

## 6. 読み取れないこと（NOT RUN）

- `worldFingerprint` から prelude を除外した場合の影響（継続トークン・`STALE_SNAPSHOT` の挙動・既存 Evidence との整合）。
- primitive 構文を追加した場合の正規化 IR・検証・全ターゲットへの波及。
- 実DB・実Provider・実Host、HTTP・Desktop、意味の受理・権限・配備。

## 7. 位置付け

25ラウンドの提案を、実装可能性の面から検証した。**「`std` を `.tei` にすればよい」は正しい方向だが、そのままでは不可能**であり、必要な前提（primitive 構文・指紋の扱い）を特定した。これは #2620 の判断に必要な材料であり、判断そのものは Owner に属する。

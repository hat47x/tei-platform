# catalogはbuilderのdigestを持っているが、entryは`-child`と比べていない

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-C0 §4.37 は、条件5の**builder同一性**を「**`-child`に名指ししたファイルのbyte**」と定義しています。前の記録（[builder同一性の範囲](TEI_GO_SELF_BUILD_CHILD_IDENTITY_SCOPE_2026-11-10.md)）は**プロキシ**を使ってこの範囲の限界を示しました。

**本記録はプロキシを使わず、実物の別バイナリで同じ限界を示します。** そして**catalogがbuilderのdigestを実際に持っている**ことを確認しました。

## 実測

| 項目 | 値 |
| --- | --- |
| catalog の `providers[0].artifactDigest` | **`sha256:7b90fa44…`**（実在するbuilderのsha256と**一致**） |
| `-child` に渡したファイル | **`sha256:8b0efd46…`**（同じsourceを`-ldflags "-s -w"`でbuildした**別の実バイナリ**） |
| 二つの関係 | **別ファイル。プロキシではない。どちらもchild protocolを自分で話す** |
| 実行結果 | **exit 0、`ok:true`、`artifactReproducibility.state:reproduced`** |
| `firstChildDigest` / `secondChildDigest` | どちらも **`sha256:8b0efd46…`**（渡したファイル） |
| **`l2b.claimed`** | **`true`** |

**catalog が「このbuilderだ」と宣言したdigestと、実際に使ったbuilderのdigestが違うのに、Claimが立ちました。**

## 二つの記録の違い

| 記録 | 差し替え方 | 示したこと |
| --- | --- | --- |
| 前の記録（プロキシ） | **同じファイルのbyte**のまま、背後で別の実行物を起動 | ファイルのbyteを見る検査はプロキシに無力 |
| **本記録（別バイナリ）** | **ファイルのbyte自体が違う** | **catalogの宣言と実物が違っても、entryは比べていない** |

前の記録は「同じファイルを差し替えられた」話で、本記録は「**そもそも別のファイルを渡せた**」話です。**後者の方が素直な攻め方**（プロキシという細工が要らない）です。

## 比べる材料はすでに両方ある

| 材料 | どこにあるか |
| --- | --- |
| builder の正しい digest | **catalog の `providers[].artifactDigest`**（実測で実物と一致） |
| 実際に使う builder | **`-child` のファイル**（entry が読める） |

**entry は両方を持っていますが、比べていません。** `cmd/tei/cli_self_build_outer.go` に `artifactDigest` を builder と突き合わせる処理はありません（`builderDigest`・`artifactDigest` の比較箇所なし）。

## していないこと

- **検査は追加していません。** catalogの宣言と`-child`を比べて拒否するのは**受理範囲の変更**で、§4.37 の持ち主の判断です。本記録は材料が揃っていることの実測までです
- **`-child` を catalog と一致させても、プロキシは防げません。** 前の記録の限界はそのまま残ります
- `linux/amd64` の一回の観測です
- **`qualified` は `false`** です

## 検証

catalog の `artifactDigest` が実builderのsha256と一致することを確認し、別バイナリを`-child`に渡して `claimed=true` を観測しました。

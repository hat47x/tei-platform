# 「state分離」は、childの申告と文字列が一致するかである

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-C0 §4.37 は、L2-b の条件5が「stateの分離」を確かめるとしています。`docs/status.md` も **「「state分離」はchildの申告との一致で確かめます」** と書いています。

**本記録は、その検査が「実際に別のstateを使ったか」ではなく「childが渡された値をそのまま返したか」であることを実測で示します。**

## なぜ差が出るか

エントリの判定は `reproducibilityStateVerdict` の**文字列比較**だけです。

```go
if filepath.Clean(passed) != filepath.Clean(reported) {
    return "the child used a different state directory than the one this entry passed"
}
```

- `passed` は entry が `-state` で渡した値
- `reported` は **child が stdout で申告した `stateDir`**

**child が別のディレクトリで構築して、渡された値をそのまま申告すれば、比較は一致します。**

## 実測

child を**ラッパー**にしました。ラッパーは:

1. 受け取った `-state` の値を**控える**
2. child へ渡す引数の `-state` を**共通の1ディレクトリ**へ**差し替える**（二回の build が同じ state を使う）
3. 本物の child を走らせ、出力 JSON の `stateDir` を**控えた値へ書き戻す**

| 項目 | 結果 |
| --- | --- |
| 実際に child が使った state | **二回とも `/tmp/.../shared`（同じ）** |
| `firstStateDir`（entry が渡した値） | `/tmp/tmp.fRLOrowzQO/state` |
| `secondStateDir`（entry が二回目に渡した値） | `/tmp/tei-self-build-rebuild-2781618408/state` |
| `artifactReproducibility.state` | `reproduced` |
| **`l2b.claimed`** | **`true`** |
| `l2b.unmet` | 空 |

**二回の build が同じ state を共有したのに、Claimが立ちました。**

## これは何を意味するか

| 言えること | 言えないこと |
| --- | --- |
| 「state分離」は **child の自己申告との文字列一致**である | **実際に別のstateが使われたこと** |
| 申告を正しく返す child なら、裏で同じ state を使っても Claim は立つ | その Claim が「state を共有しなかった」ことを意味する |
| `firstStateDir`・`secondStateDir` は**entry が渡した値**であり、child が使った値ではない | 記録された state が実際の state であること |

**エントリは child に `-state` を渡し、値を差し替えて渡している**（二回目は新しいディレクトリ）。**しかし child がその値を使ったかは見ていません。**

## 条件5の他の要素との違い

- **成果物のバイト一致**は entry がディスクから読み直して比べます（本物の検査）
- **builder同一性**は `-child` のパスの byte です（[別記録](TEI_GO_SELF_BUILD_CHILD_IDENTITY_SCOPE_2026-11-10.md)）
- **state分離**は child の申告との文字列一致です（本記録）

**三つのうち、entry が自分で確かめるのは成果物のバイトだけです。**

## 何を言っていないか

- **`darwin` は扱っていません。** `linux/amd64` のみです
- **悪意ある呼び手の一般論ではありません。** 示したのは「申告と実態が一致しない構成があり得る」ことだけです
- **本記録は対処を決めません。** `-state` の使用を entry が確かめる手段（ビルド後のディレクトリ検査など）は、Claim の意味の変更であり §4.37 の持ち主の判断です
- **`qualified` は `false`** です

## 限界

- `linux/amd64` の一回の観測です
- ラッパーが意図的に申告を書き換える構成であり、通常の利用ではありません
- **本物の child はこの構成をしません。** 示したのは検査の性質であって、同梱 child の欠陥ではありません
- 値はコミットに依存します

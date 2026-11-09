# `parameters` も照合する（§4.17の残作業）

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-C0 §4.17が「照合の意味が未決」として残した**`parameters`**を、**実測して前提を確かめてから**実装した記録です。

**L2-bは依然として成立しません。**

## §4.17の前提は誤りだった

§4.17は`parameters`を照合しない理由として「resolverが正規化・順序替えをする可能性があり、不一致が『誤った宣言』と『異なる正規化』を区別しない」と書きました。

**実測すると、どちらの側も名前でソートするだけで、値を変えていませんでした。**

| 側 | 正規化 | 何をするか |
| --- | --- | --- |
| 宣言 | `buildintent.NormalizeIntent` | `Parameters`を**名前でソートするだけ** |
| plan | `plugin.NormalizeBuildPlan` | `Parameters`を**名前でソートするだけ** |

`resolve.go:200` は `Parameters: n.Parameters` と代入するだけで、値の変換も正規化もしません。

**したがって「値の違い」は本物の違いである。** 順序だけが正規化の対象なので、**名前で突き合わせれば順序に依存しない**。

## 実装

`compareIntentFields`に`parameters`の照合を足しました。

| 検証 | 内容 |
| --- | --- |
| 名前の存在 | 宣言の各パラメータがplanにもあるか |
| 値 | 同じ名前の値が一致するか |
| 件数 | 両側の件数が一致するか |

名前で突き合わせるので、**宣言の順序とplanの順序が違っても一致**します。不一致は**パラメータ名と両方の値**を名指しして拒否します。

## 実測（新規6件）

| 検証 | 結果 |
| --- | --- |
| **正規化の性質**（宣言側・plan側とも名前順ソートのみで値は不変） | PASS。名前順であることと値が変わらないことを直接確かめる |
| 同じパラメータを**逆順**で並べる | **通る**（順序に依存しない） |
| 値を変える（`size` → `speed`） | 拒否、**名前と両方の値**を名指し |
| 宣言にだけあるパラメータ | 拒否 |
| planにだけあるパラメータ | 拒否（件数不一致） |
| 両側とも0件 | **通る** |

1行目が要点です。**照合が「名前順ソートのみ」という性質に依存している**ので、その性質自体を試験で固定しました。将来`NormalizeIntent`が値を正規化するようになったら、この試験が落ちて気づけます。

## 何が増えて、何が増えないか

| 増える | 増えない |
| --- | --- |
| 宣言とplanが同じパラメータ集合（名前と値）を指すことの確認 | **L2-b**（`l2bClaimed`は`false`） |
| 順序に依存しない照合 | 「その宣言がその計画を導く」こと |
| 正規化の性質そのものの固定 | (c)（artifactがその計画の出力であること） |

## §4.17の記述を訂正した点

§4.17は「`parameters`は正規化の意味を決める必要がある」と書きましたが、**測ると正規化はソートだけでした**。この記述を実測に基づいて訂正しました。**推測で「未決」と書いていた**わけです。

## 残る未決

| 欄 | 状態 |
| --- | --- |
| `builder.contract`・`compatibleMajor` | planが持つのは**解決済みの`BuilderIdentity`**で、宣言の要求と形が違う。要求と解決結果の対応規則を決めれば照合できる |
| `secretRequirements` | planは名前を`SecretRequirementNames`として持つが`plugin.BuildPlan`には無い。宣言の`Authority`に対応する欄が無い |

**この2つは「測れば決まる」種類ではなく、対応規則そのものを決める必要があります。**

## 検証

- `gofmt -l tools/reference-plugin-package-build-child`：差分なし
- `go vet ./tools/reference-plugin-package-build-child`：PASS
- `go test ./tools/reference-plugin-package-build-child -count=1`：PASS（既存16件＋新規6件）
- `go build ./...`：PASS
- `go test ./cmd/tei -count=1 -run "TestSelfBuild|TestPlan|TestIntent|TestFingerprint"`：PASS（72秒）
- `go test ./tools/repository-plan-check -count=1`：PASS

**NOT RUN**：Windows、別Go版、実プロセスでの全欄一致観測。

## 限界

- パラメータは**名前と値だけ**を比べます。`BuildParameter`は`Name`と`Value`しか持たないので、それ以上はありません。
- 残る2欄（`builder`・`secretRequirements`）は未着手です。
- G4は`UNQUALIFIED`のまま。

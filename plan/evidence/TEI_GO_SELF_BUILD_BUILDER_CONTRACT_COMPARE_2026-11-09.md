# `builder` の契約も照合する（§4.18の残作業）

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-C0 §4.18が「規則そのものを決める必要がある」として残した**`builder`**のうち、**contract と major の対応規則が実装上すでに決まっている**ことを確認し、照合を実装した記録です。

**L2-bは依然として成立しません。**

## 前提の確認

§4.18は`builder`について「planが持つのは解決済みの`BuilderIdentity`で、宣言の要求と形が違う。対応規則を決める必要がある」と書きました。

**規則は既に実装されていました。**

```go
// tools/reference-build-authoring-intent/fixture.go:276
ContractVersion: fmt.Sprintf("%s/v%d", p.Contract, p.ContractMajor),
```

宣言の`builder.contract`と`builder.compatibleMajor`は、解決すると`<contract>/v<major>`という`ContractVersion`になります。**決めるべき規則は既にあり、コードに書かれている**わけです。

## 実装

`compareIntentFields`の先頭で、この対応規則を**明示的に**適用して比べます。

| 宣言 | plan | 比較 |
| --- | --- | --- |
| `builder.contract` + `compatibleMajor` | `builder.ContractVersion` | `<contract>/v<major>` を組み立てて一致を確認 |

不一致は**組み立てた値とplanの値を両方**名指しします。

**この規則をchildの側にも書いたのは意図的です。** resolverが綴りを変えたら、この照合が壊れる場所になる。その場所を試験で固定しました（後述）。

## 比べないもの

| 欄 | 理由 |
| --- | --- |
| `builder.Provider` | 選択方針であり、解決結果ではない |
| `builder.Pin` | 同上 |
| planの`builder.ID`・`Version`・`ArtifactDigest` | 宣言は**方針**を書き、planは**解決済みの実体**を持つ。方針から実体は決まらない |
| `secretRequirements` | 下記 |

## `secretRequirements` は照合できない

| 事実 | 帰結 |
| --- | --- |
| `plugin.BuildPlan`に`SecretRequirements`欄が無い | plan側に比べる相手が無い |
| 名前は`ResolvedBuildReceipt.SecretRequirementNames`に載る | **receiptはchildに渡っていない**（childが受け取るのは`-plan`と`-intent`） |
| 宣言の`Authority`（取得元）に対応する欄がplanにもreceiptにも無い | 比べる対象が存在しない |

**「測れば決まる」問題ではなく、「比べる相手が無い」問題である。** 照合するには、receiptをchildへ渡すか、planへ欄を足すか、いずれかの設計変更が要る。

## 実測（新規6件）

| 検証 | 結果 |
| --- | --- |
| **対応規則の綴り**（`contract/vN`）がplanと一致すること | PASS |
| 一致する宣言 | **通る** |
| `compatibleMajor`を変える | 拒否、`major 2`・`contract/v2`・`contract/v1`を名指し |
| `contract`名を変える | 拒否 |
| planの`Builder.ArtifactDigest`を変える（**解決済み実体**） | **通る**（宣言の不一致と読まない） |
| `builder.Provider`を足す（**方針**） | **通る** |

## 何が増えて、何が増えないか

| 増える | 増えない |
| --- | --- |
| 宣言の契約要求とplanの解決済み契約の一致確認 | **L2-b**（`l2bClaimed`は`false`） |
| 対応規則の綴りをchild側でも固定 | 「その宣言がその計画を導く」こと |
| 方針と実体を区別して比べないことの明示 | `secretRequirements`（比べる相手が無い） |

## 検証

- `gofmt -l tools/reference-plugin-package-build-child`：差分なし
- `go vet ./tools/reference-plugin-package-build-child`：PASS
- `go test ./tools/reference-plugin-package-build-child -count=1`：PASS（既存22件＋新規6件）
- `go build ./...`：PASS
- `go test ./cmd/tei -count=1 -run "TestSelfBuild"`：PASS
- `go test ./tools/repository-plan-check -count=1`：PASS

**NOT RUN**：Windows、別Go版、実プロセスでの全欄一致観測。

## 限界

- **`secretRequirements` は未照合。** 比べる相手がplanにもchildの入力にも無い。
- `builder.Pin` は照合していません。pinは「意図した exact pin」であり、解決結果との比較は`BuilderResolution.Pinned`が担う領域です。
- G4は`UNQUALIFIED`のまま。

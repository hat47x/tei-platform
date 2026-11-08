# entryが宣言を自分で読む（§4.13 I-1a）

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-C0 §4.13が次の一歩に決めた**I-1a（解析だけを`internal/`へ移す）**を実装した記録です。

**L2-bは依然として成立しません。** 本記録が増やすのは「entryが保持した宣言について、**entry自身の照合値**を持てる」ことです。

## 何を移したか

`tools/reference-build-authoring-intent`のうち、**宣言の型と厳密な読み取りだけ**を`internal/buildintent`へ移しました。`Resolve`（計画の導出）は`tools/`に残しています。

| 移した | 残した |
| --- | --- |
| `BuildIntent`以下の型8つ | `Resolve`・`MarshalReceipt`・`VerifyDerivedReceipt` |
| `ParseIntent`（厳密JSON読み） | `ParseFixtureCatalog`・`NewFixtureContext`（fixture catalogは参照実装の固定入力） |
| `ValidateIntent`相当の`Validate`・`ValidateSourceRef` | `ResolutionContext`の設計 |
| `NormalizeIntent`・`IntentFingerprint` | `DirSourceResolver` |
| `ValidateInputJSON`・`RejectKeyCaseVariants` | CLI（`main.go`） |

**`ParseFixtureCatalog`を移さなかった**のは§4.13の判断どおりです。fixtureは名前のとおり参照実装の固定入力であり、`internal/`へ置くと「fixture catalogが製品の一部」に見えます。

## 二つの読みを区別して記録する

`intentDeclaration`に欄を2つ足しました。

| 欄 | 意味 |
| --- | --- |
| `digest` | entryが保持した**バイト**のdigest（従来から） |
| `intentFingerprint` | そのバイトが**宣言として何を意味するか**をentry自身が導いた値（新） |
| `strictlyRead` | 共有readerが通ったか |
| `strictReadFailure` | 通らなかった理由 |

**厳密読みが失敗しても記録は残します。** `strictlyRead=false`と理由を書き、`intentFingerprint`は空にします。**欄が無いことを一致と読ませない**ためです。`formatVersion`が無い場合は従来どおり停止します（entryが何を読んだか記述できないため）。

## 実測

| 検証 | 結果 |
| --- | --- |
| 参照fixture（`testdata/intent.json`）をentryが読む | `strictlyRead=true`、`intentFingerprint`非空 |
| 同じバイトを参照resolverに解かせる | `receipt.intentFingerprint`が**entryの値と一致** |
| 導出値を手書きした宣言 | `strictlyRead=false`、`intentFingerprint`空、理由を記録、`digest`は記録 |
| `formatVersion`が無い宣言 | 従来どおり停止 |

**entryの導出値と参照resolverの導出値が一致しました。** これまではentryが宣言の意味を読めず、`intentFingerprint`はchildの申告だけでした。

## 何が増えて、何が増えないか

| 増える | 増えない |
| --- | --- |
| entryが**保持した宣言**について、自分で導いた照合値を持てる | L2-b。計画の導出は依然`Resolve`（child）にある |
| `intentBinding`の基準を「childの申告」から「entry自身の導出値」へ移せる土台 | `planCrossCheck`の限界（二つの自己申告の一致） |
| 厳密読みの可否と理由が記録に残る | (c)（artifactがその計画の出力であること） |

**計画の導出をentryが行う経路（I-1b）は未着手です。** `ResolutionContext`の設計判断を伴うため、別の設計判断として切り出す（§4.13）。

## 検証

- `gofmt`：差分なし（`internal/buildintent`・`tools/reference-build-authoring-intent`・`cmd/tei`）
- `go vet ./internal/buildintent ./tools/reference-build-authoring-intent ./cmd/tei`：PASS
- `go build ./...`：PASS
- `go test ./internal/buildintent -count=1`：PASS（9件。厳密読み・派生値拒否・大小文字・未知key・末尾データ・source ref・digest形状・定義元・secret名）
- `go test ./tools/reference-build-authoring-intent -count=1`：PASS（既存試験が新しい配置で通る）
- `go test ./cmd/tei -count=1 -timeout 900s`：PASS（305秒）
- `go test ./cmd/tei -run TestIntentDeclaration`：PASS（3件。参照resolverとの一致を含む）

**NOT RUN**：Windows、CI、`repository-plan-check`以外のリポジトリ全体チェック。

## 限界

- **`IntentFingerprint`は宣言の意味の照合値であり、計画の照合値ではない。** `planReceipt.planFingerprint`とは別物として扱う。
- **`intentBinding`はまだchildの申告に依存している。** entryが導出値を持てるようになったので、次の周で「entryの導出値とchildの申告を比べる」形へ移せる。
- fixture catalogを移していないため、entryは依然として計画を導けない。
- G4は`UNQUALIFIED`のまま。

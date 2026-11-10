# entryの導出したintent fingerprintが参照resolverと一致することの検証

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-C0 §4.13（I-1a）は[#3647](https://github.com/hat47x/tei-platform/pull/3647)で実装済みです。この記録は、実装そのものではなく**実装が成り立っているかの検証**を足すものです。

**L2-bは依然として成立しません。**

## なぜ検証を足すか

#3647 は`internal/buildintent`を新設し、entryが`intentDeclaration.intentFingerprint`を自分で導くようにしました。`internal/buildintent/intent_test.go`は**reader自体**を7件で固定しています。

しかし、**entryの導出値が参照resolverの導出値と一致するか**は、どこでも固定されていませんでした。両者が同じパッケージを使うため一致するはずですが、「同じパッケージを使っている」は実装の性質であり、観測ではありません。片方だけが別経路に変わったとき、テストが無ければ気づけません。

## 足した検証

`cmd/tei/cli_self_build_intent_reader_test.go` に3件を追加しました。

| 検証 | 何を固定するか |
| --- | --- |
| `TestIntentDeclarationMatchesTheReferenceDerivation` | entryの`readIntentDeclaration`が返す`intentFingerprint`が、**同じバイトを参照resolverに解かせたreceiptの`intentFingerprint`と一致する**。あわせて`digest`（バイト）と`intentFingerprint`（宣言の意味）が**別の値**であること |
| `TestIntentDeclarationRefusesWhatTheSharedReaderRefuses` | 導出値を手書きした宣言は、entryが**拒否する**。fingerprintを導かない（読めない宣言の照合値を持たない） |
| `TestIntentDeclarationStillRequiresFormatVersion` | `formatVersion`が無い宣言は従来どおり停止する |

1件目は**参照resolverのバイナリを実際にビルドして起動**し、その出力と比べます。同じ関数を2回呼ぶだけでは、呼び出し側が同じ値を別経路で加工する変化を捕れないためです。

## 実測

| 検証 | 結果 |
| --- | --- |
| 参照fixture（`testdata/intent.json`）でentryの導出値 | 非空、`digest`と異なる |
| 同じバイトを参照resolverに解かせる | `receipt.intentFingerprint`が**entryの導出値と一致** |
| 導出値の手書き | entryが拒否、fingerprintは空 |
| `formatVersion`なし | 従来どおり停止 |

## 何が増えて、何が増えないか

| 増える | 増えない |
| --- | --- |
| entryの導出値と参照resolverの導出値が**実測で一致する**ことの固定 | L2-b。計画の導出は依然`Resolve`（child/link側）にある |
| 同じパッケージ共有が**観測**として確かめられていること | `planCrossCheck`の限界、(c) |
| 読めない宣言からfingerprintを導かないことの固定 | 計画の導出をentryが行う経路（I-1b） |

## 検証

- `gofmt -l cmd/tei`：差分なし
- `go vet ./cmd/tei`：PASS
- `go test ./cmd/tei -count=1 -run TestIntentDeclaration -v`：3件PASS
- `go test ./tools/repository-plan-check -count=1`：PASS

**NOT RUN**：`go test ./cmd/tei -count=1`の全体（CIで確認）、Windows、別Go版、I-1b。

## 限界

- **`intentFingerprint`は宣言の意味の照合値であり、計画の照合値ではない。** `planFingerprint`とは別物として扱う。
- この記録は**実装を含まない**。#3647 の実装が対象である。
- `intentBinding`はまだchildの申告に依存している。entryが導出値を持てるようになったので、次は「entryの導出値とchildの申告を比べる」形へ移せる。
- G4は`UNQUALIFIED`のまま。

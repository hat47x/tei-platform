# 宣言したbuilderと、起動したファイルを結ぶ

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-C0 §4.47 の実測記録です。[child同一性の範囲の記録](TEI_GO_SELF_BUILD_CHILD_IDENTITY_SCOPE_2026-11-10.md)が測った「プロキシの背後の差し替えは止まらない」に対し、**何を塞ぎ、何が残るか**を示します。

## 評価：欠陥か、保証範囲の限定か

両方が混ざっていたので分けました。

| 事象 | 判定 | 理由 |
| --- | --- | --- |
| 計画が宣言するbuilderのdigestと、起動したファイルのdigestを比べていない | **実装の欠落** | 両方ともentryが持つ値で、childの申告なしに比べられた。計画がbuilder Dを名指しするのに、byte Eのファイルを起動してもClaimが立った |
| 起動したファイルが、背後で別の実行物を起動する | **保証範囲の限定** | entryはexecを監視しない。宣言されたbuilderがコンパイラを起動するのと区別できない |
| ファイルを読んでからパスで起動する間に、ファイルが変わり得る | **保証範囲の限定** | 保持したイメージからの起動はRuntime側の仕組みで、この入口には無い |

一行目を塞ぎ、二行目と三行目はClaimの文言を限定しました。

## 変更

- `cmd/tei/cli_self_build_builder_binding.go`：`builderBinding`の記録。計画の`builder.artifactDigest`と`invoked.executableDigest`を比べる
- `assessL2B`：`builderBinding`が`bound`であることを要求する（六つ目の条件）
- `l2b`と`artifactReproducibility`の境界文：「同じchild実行物」を「`-child`に名指ししたファイルの同じbyte」に直した

新しい型・実行器・Evidenceの形式は足していません。

## 実測（実プロセス、`windows/amd64`、Go 1.27.0）

`cmd/tei/cli_self_build_builder_binding_test.go`の試験です。構築器は同じsourceから作った、**byteが違い振る舞いが同じ**二つです。

| 試験 | 構成 | `builderBinding` | 再現 | `l2b.claimed` | 試行 |
| --- | --- | --- | --- | --- | --- |
| `…HoldsWhenTheStartedFileIsTheDeclaredBuilder` | catalogがAを宣言、`-child`もA | `bound` | `reproduced` | **`true`** | `ok` |
| `…WithholdsTheClaimWhenTheCatalogNamesOtherBytes` | catalogが別のbyteを宣言 | `mismatched` | `reproduced` | **`false`** | `ok` |
| `…WithholdsTheClaimBehindAProxy` | catalogがAを宣言、`-child`はA→Bと起動するプロキシ | `mismatched` | `reproduced` | **`false`** | `ok` |
| `…TreatsADeclaredProxyAsTheDeclaredBuilder` | **catalogがプロキシを宣言**、プロキシはA→Bと起動 | `bound` | `reproduced` | **`true`** | `ok` |
| `…AndReproductionStopADirectReplacement` | `-child`が初回に自分のファイルを別のbuilderへ書き換える | `bound`（1回目） | `mismatched` | **`false`** | 失敗 |

プロキシの試験は、起動記録（1回目A、2回目B）、`firstChildDigest == secondChildDigest ==` プロキシのdigest、を確かめたうえで、`builderBinding`だけがClaimを止めていることを見ています。直接の差し替えは`mismatchReason`が`the two runs used different builder executables`であることを見ています。

## 四行目が示す限界

**プロキシ自体をcatalogが宣言すれば、Claimは立ちます。** これは見落としではなく、この入口が言える範囲です。

- 同一性は**起動したファイルのbyte**である
- そのファイルが何を起動したかは、**宣言されたbuilderの振る舞い**であり、entryは観測していない
- プロキシをbuilderとして宣言するcatalogを受け入れるかは、供給のTrustの判断である

したがって現行のClaimは「同じ構築器で二回buildした」ではなく、**「計画が宣言するbuilderと同じbyteのファイルを、二回起動した」**です。

## 検証

- `go test ./cmd/tei -run 'BuilderBinding|BindBuilderToPlan|AssessL2B|TestBuildCatalogAlone|TestExplicitRebuild'`：PASS
- 既存の正例（`TestSelfBuildEstablishesL2BEndToEnd`、`TestSelfBuildEstablishesL2BFromARealGoBuild`）は、catalogが実際に起動するbuilderのbyteを宣言しているので、変更なしで通る
- `TestBuildCatalogAloneRunsTheSecondBuild`は、catalogのbuilder digestが仮の値のままClaimを期待していた。**この試験自体が塞いだ欠落に依存していた**ので、catalogが起動するファイルを宣言するように直した

**NOT RUN**：`linux`・`darwin`・`arm64`、CI。[再導出の一式](TEI_GO_SELF_BUILD_CHILD_IDENTITY_SCOPE_2026-11-10/README.md)の`run.sh proxy`（シェルのプロキシ）は未実行です。現行では`l2b.claimed=false`になるはずで、その旨を同READMEに注記しました。`raw/`の取り直しは残っています。

## 限界

- TOCTOU（読んでから起動するまで）は閉じていません
- `mismatched`は試行を失敗にしません。Claimだけを落とします（理由と覆し方は§4.47）
- catalogを渡さない試行には計画が無いので、`builderBinding`は記録されません。その試行はもともとClaimが立ちません
- `qualified`は`false`、G4は`UNQUALIFIED`のままです

# 起動主体と被起動主体を分けたBuild駆動の記録

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-C0/C1の前提として、**Buildを別プロセスの起動として駆動し、invokerとinvokedの同一性を別々に記録する**参照実装を走らせた記録です。[SB-C0の入口契約](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)の案Bに対応します。

**L3は主張しません。** L3（stage1が生成したTEI自身がstage2を駆動した）は、Stage1の生成物を実際に駆動主体として使ったときに初めて成立します。

## なぜ要るか

「二回Buildして同じbyteになった」（[SB-N8](TEI_GO_BUILD_REPRODUCIBILITY_2026-11-07.md)）だけでは、**どちらのプロセスがBuildを主導したか**が分かりません。案BではStage1のTEIが子プロセスを起動してStage2を走らせるため、次を別々に固定する必要があります。

1. 駆動主体（invoker）の実行ファイルdigest
2. 被駆動主体（invoked）の実行ファイルdigest・argv・PID・exit
3. 子が報告したartifact digestと、**ディスク上に実際にできたartifactのdigest**の一致

## 観測

2026-11-07、Windows 11・Go 1.27.0。`reference-go-build-with-supply`を子プロセスとして起動し、`cmd/tei`を実Buildしました。

| 欄 | 値 |
| --- | --- |
| `driverSelfDigest` | `sha256:4971c24be2ad5fa34b9ee5aea48f321bce2892486e8d8ef41c6f9e660fa597dd` |
| `invoked.executableDigest` | `sha256:1b4343c23ff182eac3505791f33f2aadcf4123272eec684919af9498c9d82913` |
| `invoked.exitCode` | `0` |
| `childArtifactDigest` | `sha256:ac18ed8ed6626392c894de08ea222dc98de2d5d29399080124d22ac3d947ae6e` |
| `artifactCrossCheckedDigest` | 同上（一致） |
| `artifactCrossCheckMatched` | `true` |
| `l3Claimed` | `false` |

**駆動主体と被駆動主体のdigestが異なる**ことが要点です。同じプロセス内Build（案A）では区別できない二つを、別々の値として残せています。

## 解釈

**わかったこと**：案Bの形（子プロセス起動）で、invoker/invokedの実行ファイルdigest・argv・exit・PIDと、子の主張とディスク実物の一致を、機械可読に記録できる。no-clobber、PATH非解決、失敗も証拠として残ることも確認した。

**主張しないこと**：

- L3（`l3Claimed=false`）。Stage1の生成物をinvokerにした実験は未実施。
- 受理・署名・権限・配備は扱っていません。
- 1回のWindows観測です。Linuxでは同じテストが通りますが、digestのOS間一致は主張しません。

## 実装上の判断と、実際に踏んだ問題

1. **`-child-arg-file`を追加**。シェル経由でJSON配列を渡すと引用符処理で壊れ（`invalid character 'r' in numeric literal`）、引数一式をファイルから読む経路が必要になった。
2. **PATH非解決**。`-child`は実在する通常ファイルを要求し、digestを起動前に取る。
3. **no-clobber**。artifact先が存在すれば起動前に拒否。
4. **子のJSONを信じない**。ディスクから読み直したdigestと別欄で比較する。
5. **失敗も証拠**。子の非0終了は`exitCode`として残し、driverの終了コードは「観測できたか」だけを示す。

## 検証

- `gofmt -l tools/reference-go-build-stage-driver`：差分なし
- `go vet ./tools/reference-go-build-stage-driver`：PASS
- `go test ./tools/reference-go-build-stage-driver -count=1`：PASS（7件。WindowsとWSLの両方）
- `go test ./tools/repository-plan-check -count=1`：PASS

**NOT RUN**：Stage1の生成物をinvokerにするL3実験、二回の駆動試行の比較、CI、trusted full checkout。

## 次の一歩

`reference-go-build-stage-driver`の`-child`に、**Stage1で作った`cmd/tei`のバイナリ**を渡すのがL3の最小実験になります。ただし現時点の`cmd/tei`はBuildを提供しない（[SB-C0の入口契約](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)§2）ため、先に入口をどうするかを決める必要があります。

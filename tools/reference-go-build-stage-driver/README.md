# 起動主体と被起動主体を分けて記録するBuild駆動

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-C0/C1の前提として、**Buildを別プロセスの起動として駆動し、起動した側と起動された側の同一性を別々に記録する**参照実装です。[SB-C0の入口契約](../../plan/design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)の案B（既存toolのsubprocess起動）に対応します。

**この実装はL3を主張しません。** L3（stage1で生成したTEI自身のバイナリがstage2を駆動した）は、Stage1の生成物を実際にStage2の駆動主体として使ったときに初めて成立します。ここで示すのは「誰が、どの実行ファイルを、どのargvで起動したか」を記録できることだけです。

## 解く問題

「二回Buildして同じbyteになった」だけでは、**どちらのプロセスがBuildを主導したか**が分かりません。案Bでは、Stage1のTEIが子プロセスを起動してStage2を走らせる形になるため、次を別々に固定する必要があります。

1. **駆動主体**（invoker）の実行ファイルdigest
2. **被駆動主体**（invoked）の実行ファイルdigest・argv・PID・exit
3. 子が自分で報告したartifact digestと、**ディスク上に実際にできたartifactのdigest**の一致

三つ目を別に取るのは、子のJSONを信じるだけでは「子が別のファイルを作った」ケースを見逃すためです。

## 出力

```text
tei.reference.go-build-stage-driver/v0
  driverSelfDigest              駆動主体の実行ファイルdigest
  invoked.executableDigest      被駆動主体の実行ファイルdigest
  invoked.argv                  exact argv（executableをargv[0]に含む）
  invoked.pid / exitCode
  invoked.stdoutDigest          子の標準出力のdigest
  childSchema                   子が返したJSONのschema
  childArtifactDigest           子が主張したartifact digest
  artifactCrossCheckedDigest    ディスクから読み直したartifact digest
  artifactCrossCheckMatched     上二つの一致
  invokerIdentityRecorded / invokedIdentityRecorded
  claimedInvokerDigest          駆動主体が自称したdigest（-claimed-self指定時のみ）
  invokerClaimChecked           自称と実imageを比較したか
  invokerClaimMatched           比較結果（ClaimChecked=true のときだけ意味を持つ）
  l3Claimed                     **常にfalse**
  claimBoundary                 何を主張しないかの一文
```

## SB-N13 の照合（2026-11-08 追補）

SB-N13は「Stage1のexecutable digestをmanifestに書くだけで、実際に起動されたのは別SHAのバイナリ」を不合格とする。**自称digestを記録するだけの実装では、この不合格を検出できない。**

そこで`-claimed-self`を追加しました。駆動主体が自称するdigestを渡すと、driverは**実際に走っているprocess imageのdigest**と比較し、`invokerClaimChecked`と`invokerClaimMatched`を別欄で返します。不一致は`failureCause`に残ります。

**自称を単独で受理しない**ことが要点です。`invokerClaimMatched=true`でも`l3Claimed`は`false`のままです。自称は同一性の主張であって、どのstageが成果物を著したかの証明ではありません。

独立した失敗は`;`で連結して保持します。自称の不一致とartifact照合の不一致が同時に起きても、片方が他方を隠しません。

## 実測（2026-11-07 / Windows 11・Go 1.27.0）

`reference-go-build-with-supply`を子プロセスとして起動し、`cmd/tei`を実Buildしました。

| 欄 | 値 |
| --- | --- |
| `driverSelfDigest` | `sha256:4971c24be2ad5fa34b9ee5aea48f321bce2892486e8d8ef41c6f9e660fa597dd` |
| `invoked.executableDigest` | `sha256:1b4343c23ff182eac3505791f33f2aadcf4123272eec684919af9498c9d82913` |
| `invoked.exitCode` | `0` |
| `childArtifactDigest` | `sha256:ac18ed8ed6626392c894de08ea222dc98de2d5d29399080124d22ac3d947ae6e` |
| `artifactCrossCheckedDigest` | 同上（一致） |
| `artifactCrossCheckMatched` | `true` |
| `l3Claimed` | `false` |

**駆動主体と被駆動主体のdigestが異なる**ことが、この記録の要点です。同じプロセス内Build（案A）では区別できない二つを、別々の値として残せています。

## 実装上の判断

1. **PATH解決をしない**：`-child`は実在する通常ファイルのパスを要求し、digestを起動前に取ります。`go`のような名前だけの指定は受け付けません。
2. **no-clobber**：artifact先が既に存在すれば起動前に拒否します。上書きfallbackを持ちません。
3. **子のJSONを信じない**：子が報告したdigestと、ディスクから読み直したdigestを別欄で比べます。不一致は`failureCause`に残ります。
4. **失敗も証拠**：子が非0で終了しても、driverは観測を完了し`exitCode`を記録します。driverの終了コードは「観測できたか」だけを示します。
5. **`-child-arg-file`を追加**：シェルの引用符処理でJSON配列が壊れる問題を避けるため、引数一式をファイルから読む経路を用意しました。実際にシェル経由の指定でJSONが壊れ、この経路が必要になりました。
6. **自称digestは実imageと比べる**：`-claimed-self`は保存するだけにせず、実行中のprocess imageのdigestと比較します。比較しなかった自称は、SB-N13の不合格を検出できないためです。

## 限界

- L3は主張しません。`l3Claimed`は常に`false`です。Stage1の生成物を駆動主体にする実験は別途必要です。
- `invokerClaimMatched=true`は「自称digestが実行中のimageと一致した」ことだけを言います。**そのimageがStage1の生成物であること**は別のClaimです（Stage1のBuild記録との照合が要る）。
- 受理・署名・権限・配備は一切扱いません。
- 子の標準出力全体のdigestを残しますが、子のJSONスキーマは既知の1種類しか解釈しません。
- 1回のWindows観測です。Linuxでは同じテストが通りますが、digestのOS間一致は主張しません。

## 実行した検証

- `gofmt -l tools/reference-go-build-stage-driver`：差分なし
- `go vet ./tools/reference-go-build-stage-driver`：PASS
- `go test ./tools/reference-go-build-stage-driver -count=1`：PASS（10件。WindowsとWSLの両方）

**NOT RUN**：Stage1の生成物を駆動主体にするL3実験、二回の駆動試行の比較、CI、trusted full checkout。

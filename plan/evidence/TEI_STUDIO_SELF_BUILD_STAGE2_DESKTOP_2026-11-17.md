# Stage1のTEIがStage2でStudio Desktopを構築し、新しいプロセスで生成Readerが使われた

[#4583](https://github.com/hat47x/tei-platform/issues/4583)・[#2831](https://github.com/hat47x/tei-platform/issues/2831)・[#1824](https://github.com/hat47x/tei-platform/issues/1824)。手順は[`tools/reference-studio-part-generator`](../../tools/reference-studio-part-generator/README.md)の「Stage2でStudio Desktopそのものを構築し、新しいプロセスとして使う」です。2026-11-17の一回の連続実行の実測値だけを置きます。受理・Runtime Authority・Seed Trust・`qualified`は主張しません。

[前回の記録](TEI_STUDIO_SELF_BUILD_S1_INSPECTION_READER_2026-11-13.md)では、Stage2が再構築したのは`cmd/tei`・生成器・部品までで、Studio Desktopは外部の`go build`でした。今回はStudio Desktop実行ファイルも、Stage1のTEIが起動したchildが構築し、その実行ファイルを新しいプロセスとして起動して使いました。

## 条件

| 項目 | 値 |
| --- | --- |
| Source | main `91372a17111975834da89ff8611f3bdd1cdb07a1`（`stage_reentry.py`とREADMEは本変更の未コミット版。どちらもGoのBuild入力ではない） |
| OS | Windows 11（10.0.26100）、1台 |
| Stage0 seed | `go version go1.27.0 windows/amd64`（`sha256:7d828191…`）、外部seedで駆動されたプロセスではない |
| 供給 | `GOMODCACHE\cache\download`（childが`GOPROXY=file://`として隔離キャッシュへ読む） |
| Go build child | `reference-go-build-with-supply`、両段ともStage0のseed製（`sha256:ce29e1ba…`） |
| WebView2 | 157.0.4341.46（同版のmsedgedriver） |
| 所要 | 1,767秒（同じ機械で他の作業の`go test`が多数並行していた） |

## 各段の成果物

| 欄 | Stage1（起動主体：Stage0のTEI `sha256:83340f06…`） | Stage2（起動主体：**Stage1のTEI生成物**） |
| --- | --- | --- |
| 起動主体のdigest（ディスクから） | `sha256:83340f060f4ed6ea948e581078c409bb9bc7624501b1b4dfaa84af84777137c8` | `sha256:1f65b2db9bb142fb6a3562a36408a4604861810961f6447e47b5b77397670f06` |
| `cmd/tei` | `sha256:1f65b2db9bb142fb6a3562a36408a4604861810961f6447e47b5b77397670f06` | 同左 |
| 部品生成器 | `sha256:b289a06efdeaf504921bc9e39530fb32e9aee776711704a8af123422e0424ba9` | 同左 |
| 生成部品 | `sha256:52a9ffde5f6b46bdfd3bc02c06fa99017467222717d91ca4de9432c6dd4c316c`（コミット済み部品と一致） | 同左 |
| **Studio Wails Desktop** | `sha256:c3b44f7a3cc1cf83ec2784755389f2bb80caff00c1c023e43a5be137a3ff5b38` | 同左 |

Stage2の全4工程を起動したTEIのdigestは、Stage1が書いた`cmd/tei`のdigestと一致しました（`tei self-build`の`invokerDigest`と、スクリプトがディスクから読んだ値の両方）。

## Stage2のDesktopに入っているもの

| 欄 | 値 |
| --- | --- |
| コミット済みbundle `cmd/tei-studio/assets/workbench/main.js` | `sha256:a77782ebf115d50a5229012d1d335660aa032d55e9396a0b04bcf3d830243cbe` |
| Stage2 Desktopのbyte列にbundleが含まれる回数 | 1 |
| Stage2で再生成した部品が名指す契約 | `sha256:0a65d341992bee8ab44f9e767321536a7c8e0d13bc4c60a2ddba34a18bf1359b` |
| bundleが名指す契約 | 同上 |

## 新しいプロセスでの使用

既存の`desktop_webview_smoke.py`に**Stage2のDesktop実行ファイルのパス**を渡し、新しいWindowsプロセス（pid 29876）として起動しました。結果は`PASS_NATIVE_WEBVIEW2_EDITOR_KEYBOARD_AND_OBSERVEXEAD`、プロセスは終了コード0で閉じました。

| 観測 | 値 |
| --- | --- |
| smokeが起動した実行ファイル | `sha256:c3b44f7a…`（Stage2のDesktopそのもの） |
| ページに残った生成Readerの契約digest | `sha256:0a65d341…`（再生成部品と一致） |
| 初回Inspect後の受理回数 | 1 |
| Graphからの二回目のInspect後 | 3 |

## 判定

`stage_reentry.py --desktop-smoke`の13判定はすべて`true`で、`reentryObserved=true`でした。`--self-test`の13件（Stage0のTEIが二回作った場合、bundleを含まないDesktop、別契約のbundle・reader、CIの通常buildなど別の実行ファイルでのsmoke、1回だけの使用、smoke失敗、Stage0のTEIが作ったDesktopなど）もPASSです。

同じ日、seedのGo build childを**TEIを介さず**直接起動して同じDesktopを構築したところ、digestは同じ`sha256:c3b44f7a…`でした。byteの一致だけでは誰が構築したかを区別できず、起動主体の照合が別に必要であることを示します。

最初の試行では、smokeの記録形式がmainで変わっていたため`stage2DesktopUsedGeneratedReaderInNewProcess`が`false`になりました（smoke自体はPASS）。読み取り箇所を`page_state.graphSelection`へ直し、保存済みの記録で再判定したうえで、上記の全工程を最初から再実行しています。

## 主張しないこと

- **Node/esbuildのbundlingはTEIが行っていません。** Desktopが埋め込んだのはコミット済みbundleで、判定はそれがbundleそのものであり、再生成部品と同じ契約を名指すことまでです。
- **計画は使っていません。** 各Buildは`tei self-build`のprimitive形式（起動と記録）で、Build Intent・catalogによるBuilder選択、`l2b`、閉包の資格は主張しません。
- Go build childは両段ともStage0のseed製で、Stage0のGo toolchainも外部seedです。
- 1台・1 toolchain・1回の観測です。CI、別のWindows、別Go版では未実施です。
- 定義だけで必須項目を一つ追加し、Generator本体を変えずに反映する一巡（#4583のD0〜D5）は、まだ行っていません。

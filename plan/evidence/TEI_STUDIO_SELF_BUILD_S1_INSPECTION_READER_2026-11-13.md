# Studioの部品一つをTEIのSourceから生成し、実Desktopで使われたことを観測した

[#1824](https://github.com/hat47x/tei-platform/issues/1824)・[#2831](https://github.com/hat47x/tei-platform/issues/2831)。対象と手順は[`tools/reference-studio-part-generator`](../../tools/reference-studio-part-generator/README.md)にあります。本記録は2026-11-13の実測値だけを置きます。受理・Runtime Authority・Seed Trustは主張しません。

## 対象の同一性

| 項目 | 値 |
| --- | --- |
| 基準main | `cd109e5823a1e689440a6b642dfd35eef1d49c20` |
| 実測したbranch head | `1bcd0352b4015810f2fe688df1e6ba011bc46959` |
| 契約Source（byte） | `sha256:fa319a68d551efd2d6ee22fcffd4e825848a963509c3a3c02e21ab34343de4f2` |
| 契約Source（正規形。部品が持つ来歴） | `sha256:7156cba6cee13d0e4564e3c1d0b8e8c43e18702a3dffe4032d7773ccee3dac3c` |
| Generator Source（byte） | `sha256:47cc8dcccc9753f3a18c3fa9065e97855fb886aa31849d668fc277eabbff758c` |
| Generator Source（fingerprint） | `17ae220a93ceea9de2c456968c29d2c8c96231623f40fcb7434fec6983ae3b78` |
| 生成部品 `inspection-reader.generated.ts` | `sha256:6d3dfe806ef758dc4746ccc34b5ae646533eb056b78cbbc098630993d384586b` |
| 埋込みbundle `assets/workbench/main.js` | `sha256:f6e4dee84ce01f9937a9c7517dd18e2e25d723238fc6bb88125423c0a94bad04`（Windows・Node 24.19.0・esbuild 0.25.12で生成） |

## Stage1：実Wails Desktopでの収録と使用

Windows 11（10.0.26100）、Go 1.27.0 windows/amd64、Wails v3.0.0-beta.29、WebView2 157.0.4341.46。`cmd/tei-studio`で`go build -mod=readonly`した通常Desktop実行ファイルを、`desktop_webview_smoke.py`で操作しました。

| 実行ファイル | 埋め込んだbundle | 結果 |
| --- | --- | --- |
| `sha256:7c110b9cae65d70f9d3d71357386996ba44665fbeb78f45a35729985e2da116a` | 生成部品を含む（上表） | **PASS**。初回Inspect後に`data-tei-inspection-reader`が契約の正規形digestと一致し受理回数1、Graphからの二回目のInspect後に2 |
| `sha256:a3f40bf7a95b9a18b5c702312d5409841a1c5858a07d7943ee8e1f190e24300a` | 基準mainのbundle（生成前の手書き実装） | **FAIL**。Catalog・Inspect表示など従来の確認はすべて通り、`Inspect was not rendered by the Inspector reader generated from the current contract ... observed ''`だけで停止 |

二行目は、Inspectorが正しく表示されることと、生成部品が使われたことが別の観測であることを示します。

Go試験（WSL2 Ubuntu 22.04・Go 1.27.3）では、埋込みbundleの来歴digestを書き換えた場合と、部品を欠いた場合に`TestEmbeddedWorkbenchCarriesTheCurrentGeneratedInspectionReader`が失敗することも確認しました。

## Stage2：Stage1のTEI実プロセスによる再生成

WSL2 Ubuntu 22.04、`go1.27.3 linux/amd64`、供給は`GOMODCACHE/cache/download`。`stage_reentry.py`を実行（所要1分58秒、終了コード0）。

| 欄 | Stage1（起動主体：Stage0のTEI） | Stage2（起動主体：**Stage1のTEI生成物**） |
| --- | --- | --- |
| 起動主体のdigest | `sha256:eabfe5b2fb9ed…` | `sha256:36345b681ceb5…` |
| `cmd/tei`の再構築 | `sha256:36345b681ceb5…` | `sha256:36345b681ceb5…` |
| 生成器の再構築 | `sha256:bcd8adbd7af13…` | `sha256:bcd8adbd7af13…` |
| 生成した部品 | `sha256:6d3dfe806ef75…` | `sha256:6d3dfe806ef75…` |

判定10項目（`stage1TeiArtifactIsStage2Invoker`・`stage2PartMadeByStage2RebuiltGenerator`・`partsEqual`・`stage2PartIsTheCommittedPart`ほか）はすべて`true`で、`reentryObserved=true`でした。`--self-test`の5件（Stage0のTEIが二回作っただけの場合などを不成立とする）もPASSです。

この試行は上記headに、未コミットの本記録・README・`stage_reentry.py`が加わった作業木で行いました（いずれもBuild入力のGo sourceではありません）。

## 実行した検証

- `go test ./tools/reference-studio-part-generator`：PASS
- `cmd/tei-studio`で`go test -mod=readonly -tags server .`：PASS
- `reference/tei-studio-workbench`で`npm test`（39件）・`npm run test:p0-presenter`（7件）・`npm run typecheck`・`npm run build`：PASS（Windows・Node 24.19.0）
- 実Windows Desktop WebView2 smoke：上表

## 未実施・主張しないこと

- **NOT RUN**：GitHub Actionsでの実行（embed CIのLinux・Node 24.21.0でのbundle byte一致を含む）、`go test ./...`全体、Server版のブラウザーsmoke、macOS・arm64、人による操作。
- Node/esbuildのbundlingとWails BuildはTEIのBuild graphの外です。`tei self-build`のBuild Intent・catalogに生成工程は載っておらず、`l2b`は主張しません。次に必要なのは、生成とbundlingをrecipeとして計画に載せ、Studioのnested moduleをBuild対象にすることです。
- Stage2のGo build childは両段ともStage0のseed製です。Windows Desktopの実行ファイル自体をStage1のTEIが構築したのではありません。
- 生成したのは純関数一つで、Studio全体の生成ではありません。

# Studioの部品一つを、TEIのSourceから生成する

[#1824](https://github.com/hat47x/tei-platform/issues/1824)と[#2831](https://github.com/hat47x/tei-platform/issues/2831)の接続点です。TEI Studio Workbenchの**読取り専用Inspector reader**（`readInspection`）一つを、契約Sourceから既存の決定論transform実行機構で生成し、実Wails Desktopに収録して、使われたことを独立に観測します。

Studio全体の生成、新しいGenerator基盤、Plugin解決、Hostの追加ではありません。transformの演算も増やしていません。

## 何が正本で、何が派生か

| 役割 | ファイル | 性質 |
| --- | --- | --- |
| 契約Source | [`inspection-reader.contract.json`](../../reference/tei-studio-workbench/inspection-reader.contract.json) | **正本**。`InspectSubject`応答のどの項目を読み、何を拒否し、何を投影するか |
| Generator Source | [`builder-sources/inspection-reader-v1.transform.json`](../../reference/tei-studio-workbench/builder-sources/inspection-reader-v1.transform.json) | **正本**。`deterministic-transform-v1`のTEI管理下IR |
| 実行機構 | `internal/plugin/transform` | 既存。このtoolは呼ぶだけ |
| 生成部品 | [`inspection-reader.generated.ts`](../../reference/tei-studio-workbench/inspection-reader.generated.ts) | **派生**。手で編集しない |
| 埋込みbundle | `cmd/tei-studio/assets/workbench/main.js` | **派生**。esbuildの出力 |

手書きの`readInspection`は`presentation.ts`から削除しました。`presentation.ts`は生成部品を再exportするだけで、`main.ts`は変更していません（固定pinの対象のまま）。変更は契約Sourceに対して行い、再生成します。

生成前の手書き実装は[`testdata/inspection-reader.seed-oracle.ts`](../../reference/tei-studio-workbench/testdata/inspection-reader.seed-oracle.ts)に**試験専用の比較対象**として残しています。Workbenchからimportされず、bundleにも入りません（埋込みbundleの試験が確認します）。

契約Sourceは`InspectSubject`の二つ目の定義ではありません。`TestContractNamesWhatTheRealOperationReturns`が、契約の名指す項目を実際の`operations.InspectSubject`・`operations.CatalogWorkspace`の応答（`reference/workspace`）と照合します。

## 再生成

```bash
rm reference/tei-studio-workbench/inspection-reader.generated.ts
go run ./tools/reference-studio-part-generator \
  -transform reference/tei-studio-workbench/builder-sources/inspection-reader-v1.transform.json \
  -contract reference/tei-studio-workbench/inspection-reader.contract.json \
  -artifact-out reference/tei-studio-workbench/inspection-reader.generated.ts
```

出力先が既に存在すれば書きません（古い部品が今回の出力として残るのを避けるため）。その後、`reference/tei-studio-workbench`で`npm ci && npm test && npm run typecheck && npm run build`を実行し、`dist/`の4ファイルを`cmd/tei-studio/assets/workbench/`へ写します。この写しは[embed CI](../../.github/workflows/tei-studio-workbench-embed-ci.yml)がbyte比較します。

stdoutは1件のJSONです。`tei self-build`のchild文法（`-artifact-out`・`-state`・`artifactDigest`・`stateDir`）を満たします。

| 欄 | 意味 |
| --- | --- |
| `artifactDigest` | このプロセスが書いたbyteのdigest |
| `transformBytesDigest` / `transformSourceFingerprint` | 読んだGenerator Sourceのbyteと、正規化した同一性 |
| `contractBytesDigest` | 読んだ契約Sourceのbyte |
| `contractCanonicalDigest` | 契約の正規形のdigest。**生成部品が自分の来歴として持つ値**で、transformとは独立に計算する |
| `engine` | 実行機構のpackage・profile・Go版・target |

## 入力の受理

transform IRには文字列をエスケープする演算がありません。そのためこのtoolは、契約Sourceの文字列と項目名を`[A-Za-z0-9_.:-]+`に限って受理します（引用符の外へ出られない値だけ）。これはこの生成器の入力受理であり、汎用のSource言語ではありません。対象ID（`part.id`）と`formatVersion`はGenerator Source側のassertionが照合します。

## 区別して確かめること

| 問い | どこが確かめるか | 何に対して |
| --- | --- | --- |
| 生成器が正しいSourceを消費したか | `TestCommittedPartIsTheCurrentGeneration` | コミット済み部品 ＝ 今の契約とGenerator Sourceからの生成結果（byte） |
| Source Revisionの変更 | `TestSourceRevisionChangesPartAndIdentity`、Nodeの`the part names the contract Source that is on disk now` | 契約を変えると部品も来歴digestも変わる。整形だけなら変わらない |
| 対象IDの不一致・不正な入力 | `TestRejectedSources`（10件） | 別の`part.id`、欠けた項目、引用符を破る値、重複キーなどを部品なしで拒否 |
| 生成前後で振る舞いが同じか | `inspection-reader.generated.test.mjs`（16行） | 手書きのStage0実装と生成部品を同じ入力で比較し、期待値は別に手で書く |
| 実Buildへの収録 | `cmd/tei-studio`の`TestEmbeddedWorkbenchCarriesTheCurrentGeneratedInspectionReader` | `//go:embed`された**実byte**を読む。未収録・古い契約からの部品・二重実装で失敗 |
| 実際に使われたか | `cmd/tei-studio/test/desktop_webview_smoke.py` | 実WebView2で、部品が投影を受理したときだけ残す記録（`data-tei-inspection-reader`）を読み、契約から再計算したdigestと照合。二回目のInspectで回数が増えることも確認 |

**収録と使用は別の観測です。** 前者はGo試験、後者は実WebViewでしか確かめられません。生成部品は投影を**受理したときだけ**`document.documentElement.dataset`へ契約digestと回数を書きます。画面表示は変えていません。

## Stage0・Stage1・Stage2

| 段 | 到達条件 | この変更での状態 |
| --- | --- | --- |
| Stage0 | 既存の手書きGo/JS/WailsでStudioをビルド・起動 | 既存（変更なし。Bridge・Host・`main.ts`・Graph・Command・Editorは手書きのまま） |
| Stage1 | TEIのSourceから生成した部品が実Desktop成果物に入り、使われる | 成立（上表） |
| Stage2 | **Stage1で構築されたTEIの実プロセス**が次段のBuildへ再入場し、同じGenerator Sourceと固定入力から同じ部品を再生成する | [`stage_reentry.py`](stage_reentry.py)で観測 |

```bash
python3 tools/reference-studio-part-generator/stage_reentry.py --self-test
python3 tools/reference-studio-part-generator/stage_reentry.py \
  --repo . --supply "$(go env GOMODCACHE)/cache/download" --work /tmp/studio-part-reentry
```

各段のGo buildと部品生成は、すべて`tei self-build -child ... -artifact ...`で**TEIのバイナリが起動**します。スクリプトはStage0のseedを作り、TEIを起動し、TEIが記録した起動主体・被起動主体・成果物を突き合わせるだけです。判定は次を別々に出し、全部が真のときだけ`reentryObserved`になります。

| 判定 | 意味 |
| --- | --- |
| `stage1TeiArtifactIsStage2Invoker` | Stage2の全工程を起動したのがStage1の生成物そのもの |
| `stage2PartMadeByStage2RebuiltGenerator` | Stage2の部品は、Stage2が再構築した生成器が作った |
| `partsEqual` / `stage2PartIsTheCommittedPart` | 二段の部品が一致し、Workbenchが使っているコミット済み部品とも一致 |
| `generatorsEqual` / `teiArtifactsEqual` | 生成器とTEIの再構築も一致 |
| `everyStepCompletedAndCrossChecked` | childの申告とディスクの実byteが一致 |

**同じbyteが二回できただけでは再入場としません。** `--self-test`が、Stage0のTEIが二回作った場合・Stage2が再構築していない生成器を使った場合・別の部品を再生成した場合・申告とディスクが違う場合を、いずれも不成立と判定することを確かめます。

実測は[証拠記録](../../plan/evidence/TEI_STUDIO_SELF_BUILD_S1_INSPECTION_READER_2026-11-13.md)にあります。

## 主張しないこと

- **Studio全体の生成**ではありません。生成したのは純関数一つです。
- **Node/esbuildのbundlingとWailsのBuildは、TEIのBuild graphの外**です。`tei self-build`の計画（Build Intent・catalog）に生成工程は載っておらず、`l2b`の五条件は主張しません。使ったのは起動と記録だけを行うprimitive形式です。
- Stage2で使うGo build child（`reference-go-build-with-supply`）は両段ともStage0のseedが作ったものです。Seed Trust、受理、Runtime Authorityは主張しません。
- `studio_generated`タグの生成Goサービス（PoC B）とは別の経路です。そちらは`main`に生成物が無く、このtoolは触れていません。
- G1（生成AIによるGenerator生成規約）は扱いません。Generator Sourceは人が書いた既存IRです。

# Inspect応答の必須項目を、契約の定義だけで一つ足した

[#4583](https://github.com/hat47x/tei-platform/issues/4583)のD0〜D1に当たる観測です。手順は[Workbench README](../../reference/tei-studio-workbench/README.md)の「Inspect応答の必須項目を契約だけで足す」にあります。2026-11-17、Windows 11・Go 1.27.0・Node 24.19.0での実測です。

## 問題

Inspector ReaderのGenerator Sourceは、契約の名前付きスロットを埋める形でした。必須の応答項目を一つ足すには、Generator Source（約690行）を書き換える必要がありました。transform IRの`join-strings`は固定個の式しか連結できず、任意の長さの一覧を文字列にできなかったためです。[#5365](https://github.com/hat47x/tei-platform/pull/5365)は、この状態で契約だけを変えると新しいdigestだけが付き、検査が入らないことを防ぐため、未実装の節を拒否しました。

## したこと

| コミット | 変えたもの | 変えていないもの |
| --- | --- | --- |
| `5a6ce942`（A） | transform実行機構に有界の`join-array`を追加。Generator Sourceに`requiredResponseFields`を読む汎用規則を追加。生成toolが、この一節だけを#5365の許可表に加えて受理。契約は空の一覧 | Readerの振る舞い |
| `1d442ce8`（B） | 契約に`{"name": "schemaVersion", "type": "string"}`を追加。部品・bundleを再生成。独立Expected 3件を追加。`schemaVersion`を欠いていた手書きfixture 4ファイルに実Operationの値を追加 | **Generator Source・生成tool・transform実行機構（差分0）** |
| `02277d4f`（C） | 生成Workbenchの固定版を追加（`main.ts`は同じで、生成部品のblobだけが変わる） | 既存の固定版 |

AとBの間でGenerator Sourceのblobは`086ea8390da5ef3d949e904b17e2476a90c13246`のまま同一です。

## 観測

| 観測 | 結果 |
| --- | --- |
| 新しい契約の正規形digest | `sha256:20d5ed64da11151a03e8e045cb5d78c3ec3411598042a39224b5e37a0cb6e947` |
| 実`InspectSubject`の`schemaVersion` | `"tei.operations.inspect/v0"`（文字列）。Go試験が実Operationの応答と照合 |
| 新しいReader：`schemaVersion`欠落・数値・null | 3件とも`INVALID_INSPECTION_RESPONSE` |
| **Aの部品のまま**（契約だけ新しい） | 上の3件と、契約digestの照合が失敗 |
| **Aの部品に新しいdigestを付け替えた場合** | digest照合は通るが、上の3件が失敗し、Goの「コミット済み部品＝現在のSourceからの生成結果」も失敗 |
| 埋込みbundle | 新しい契約digestの部品がちょうど一つ（Go試験PASS） |
| 実Windows Desktop（WebView2 157、exe `sha256:9392b2c7…`） | smoke PASS。ページに残った契約digestが`20d5ed64…`、受理回数は初回1、Graph後3 |
| 既存の固定版 | 新しい生成部品を`workbench_closure_revision_mismatch`で拒否。新しい固定版は受理 |

## 試験

- `go test ./internal/plugin/... ./tools/reference-transform-runtime-engine ./tools/reference-studio-part-generator ./tools/reference-studio-static-composition`：PASS
- `cmd/tei-studio`で`go test -tags server .`：PASS（Aの時点）、`-run EmbeddedWorkbench`：PASS（B・Cの時点）
- `reference/tei-studio-workbench`で`npm test`（66件）・`npm run test:p0-presenter`（7件）・`node --test inspection-reader.generated.test.mjs`（28件）・`npm run typecheck`：PASS
- `node --test tools/reference-studio-static-composition/*.test.mjs`（20件）：PASS

**NOT RUN・未確認**：`tools/reference-plugin-self-build`はWindowsのパス長制限でcheckoutが失敗し、この機械では実行できなかった（transform実行機構を共有するためCIで確認する）。`pin-inspection.test.mjs`の4件はmainの時点で失敗しており、`npm test`にもCIにも含まれていない（本変更とは無関係）。

## 主張しないこと

- 汎用規則は**必須応答項目の有無と型**だけです。`effect`、Subject、fingerprint、`authorityLimits`の規則は、まだGenerator Sourceの名前付きスロットです。
- `tei.authoring/v1`のData項目・Ruleから契約を導いてはいません。契約は人が書いたJSONです。
- 必須項目を足しても、画面の表示は変わりません。検査が増えるだけです。
- この変更を、Stage1のTEIがBuild Intentで選んだBuilderから構築したStage2で確かめてはいません（#4583のD2以降）。Stage2がStudio Desktopを構築する観測は[別の記録](https://github.com/hat47x/tei-platform/pull/5373)で、この変更の前の契約に対するものです。

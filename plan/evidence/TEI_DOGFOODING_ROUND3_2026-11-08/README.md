# TEI自身の設計をCLIで扱う実測 第3ラウンド（2026-11-08、Windows）

- 対象課題: [#3149](https://github.com/hat47x/tei-platform/issues/3149)（Closed）。
- 前ラウンド: [第2ラウンド](../TEI_DOGFOODING_ROUND2_2026-11-08/README.md)（WSL2、`4fd88c2b1`）。
- 実行日: 2026-11-08（Asia/Tokyo）。
- 状態: **read-only の再実測**。意味の受理・権限付与・本番配備・書込みは行っていない。

## 1. このラウンドの目的

第1・第2ラウンドで起票・追跡した D-1〜D-10 の対応PRが `main` へ統合された後の版で、**同じ実行物を固定して**次を確かめる。

1. D-1・D-2・D-4〜D-8 が実際の応答で解消しているか。
2. D-3・D-9・D-10 の到達点（版の欄、Markdown設計文書、応答の形）がどこまで進んだか。
3. 第2ラウンドと**同じCLI操作**が、Windows と go1.27.0 でも同じ結論になるか（OS・toolchain を変えた再現）。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `hat47x/tei-platform` commit `9a8b9e214f5eeb6b9e91a9adec2fcc17d3fac211`（`origin/main`） |
| 実行物 | この commit の独立worktreeで `go build ./cmd/tei` した単一ファイル |
| SHA-256 | `7abce6ff8d7e4d74ed9c1384de95a0e388058a35507f659a95eab578fa0d408c`（28,307,456 バイト） |
| 実行物の自己申告 | `version -json` は `goVersion: go1.27.0`・`goos: windows`・`goarch: amd64`・`source: unavailable`（`go run` ではなくビルド済み実行物のため VCS 情報は埋まらない） |
| 環境 | Windows 11、Go `1.27.0 windows/amd64` |
| 対象Workspace | 同じ commit の `reference/workspace`・`examples/xtea-lowcode`・`examples/first-workspace` |

`version -json` の `source: unavailable` は、実行物に VCS 情報が埋まっていないことを示す。commit の同一性は 00-binary-identity.txt の値と、この実行物をビルドした worktree の HEAD で示す。第1ラウンドの実行物（`bf6930a0…`）・第2ラウンドの実行物（`16c9090d…`）とは**別の実行物**であり、数値を直接比較しない。

## 3. 生データ

同じディレクトリの `*.txt` が、実行時の標準出力のコピーである。標準エラーは全件空だったため `.err` は残していない。`.out` はリポジトリの `.gitignore`（`*.out`）に当たるため `.txt` で保存した。

| ファイル | 呼出し | 終了コード |
| --- | --- | --- |
| `00-binary-identity.txt` | 実行物の commit・SHA-256・環境 | - |
| `01-commands.txt` | `commands` | 0 |
| `02-trace-candidate-key.txt` | `trace … -subject-kind semantic-candidate-key` | 2 |
| `03-inspect-reference.txt` | `inspect -workspace reference/workspace -subject-kind semantic-data -subject-id sales.order` | 0 |
| `04-inspect-xtea.txt` | 同じ Canonical ID を `examples/xtea-lowcode` で `inspect` | 0 |
| `05-validate-reference.txt` | `validate -workspace reference/workspace -json` | 0 |
| `06-model-catalog.txt` | `model catalog -workspace reference/workspace` | 0 |
| `07-requirements.txt` | `requirements -workspace reference/workspace` | 0 |
| `08-workspace-files.txt` | `workspace-files -workspace examples/first-workspace` | 0 |
| `09-help-inspect.txt` | `inspect -h` | 0 |
| `10-trace-relation.txt` | `trace … -subject-kind semantic-relation -mode full` | 0 |
| `11-view-json.txt` | `view … -format json` | 0 |
| `13-plugin-lock.txt` | `plugin-lock -workspace reference/workspace` | 0 |
| `14-inspect-not-found.txt` | 存在しない subject を `inspect` | 1 |
| `15-inspect-unknown-kind.txt` | 未知の `-subject-kind` を `inspect` | 2 |

`12-requirements-shape-check` は `07` の JSON を読む検査で、独立のCLI呼出しではないためファイルは無い。

## 4. 観測

### 4.1 第1・第2ラウンドの課題

| 課題 | 現行main（`9a8b9e214`）での観測 | 状態 |
| --- | --- | --- |
| D-1 `trace` の未対応kind | `semantic-candidate-key` は `UNSUPPORTED_SUBJECT_KIND`・終了コード2。`SUBJECT_NOT_FOUND`（1）と区別できる（`02`） | **解消** |
| D-2 Workspace同一性 | `validate -json` の最上位と `model catalog`・`inspect`・`trace` の `result` に `workspace.name`・`formatVersion`・`manifest.sha256` がある（`05`・`03`・`04`。`trace` は `10`） | **解消** |
| D-3 版の欄の不統一 | `view -format json` は `result.semanticFingerprint` に加えて**最上位**に `worldFingerprint`・`sourceIndexFingerprint` を持つ（`11`）。`model catalog`・`inspect`・`trace`・`validate -json` と同じ計算で、同じWorkspaceなら値が一致する | **解消**（`view` の版の欄は #3366） |
| D-4 `view` の作用と広告 | `view` は `effect: read-only` のまま、`writeFlags` に `-operation-log`・`-lifecycle-state` を載せる（`01`） | **解消**（`effect` 語彙の統一は未決） |
| D-5 `plugin-lock` のround-trip | `-plugin-lock` が envelope をそのまま受理する（第2ラウンドの`06`で確認、本ラウンドは再実行していない） | **解消**（#3214） |
| D-6 不正lockの終了コード | 第1ラウンドで観測した2コマンドの差は main の修正で終了コード2に揃った（本ラウンドの直接再実行はしていない） | 解消（間接確認） |
| D-7 `-h` | `inspect -h` は終了コード0・標準出力にusage（`09`） | **解消** |
| D-8 要件JSONの形 | 読み訳ガイドに形の表と実例ファイルへの参照があり、最小形は `{"expectedKind":"semantic-data"}` で `applicable` を通る（第2ラウンドの`08`） | **解消** |
| D-9 Markdown設計文書 | `docs/`・`plan/` をWorkspaceにしても管理ファイルとして扱う（第2ラウンドの`32`・`33`）。設計判断（#3346）として残る | **残る**（設計判断） |
| D-10 応答の形 | `requirements` は `resultShape: "array"` を広告し、実応答の `result` は配列（`01`・`07`）。`model catalog` の `entries` は `prelude: true` で標準定義を印付け（`06`。`"subject":{"kind":"semantic","id":"std"},"prelude":true`）。`validate` の `diagnostics[]` は `diagnostic` を持つ入れ子のまま（契約文書§4） | **一部解消**（形の統一判断は残る） |

`01-commands.txt` には `writeFlags`・`resultShape`・`argumentValues` が載っている。`prelude` は entry の欄ではなく `model catalog` の応答側の欄なので、`commands` には出ない。

### 4.2 同じWorkspaceで4つの読取りの照合値が一致する

`03`・`05`・`06`・`11` の `worldFingerprint` は `sha256:c92de09f9595b1f46…`、`sourceIndexFingerprint` は `sha256:1827f4b46169710cbe…` で一致した。`10` の `trace` も同じ2値を持つ。**同じWorkspace・同じ実行物での4コマンドは同一版を指す**。

異なるWorkspaceで同じ Canonical ID（`sales.order`）を `inspect` した `03` と `04` は、`worldFingerprint` が異なり（`c92de09f…` と `8930145c…`）、`workspace.name` も `tei_reference` と `xtea_lowcode` で異なる。**同じIDが別の意味であることを応答だけで区別できる**。

### 4.3 失敗の区別

`14` は `SUBJECT_NOT_FOUND`・終了コード1、`15` は `UNSUPPORTED_SUBJECT_KIND`・終了コード2。`02` の未対応kindも終了コード2で、未知kindと同じ符号になる（#3180 の期待どおり）。

## 5. 第2ラウンドとの差（OS・toolchain を変えた再現）

第2ラウンドは WSL2 Ubuntu 22.04.5・Go 1.27.3、本ラウンドは Windows 11・Go 1.27.0 である。**同じ main ではない**（`4fd88c2b1` と `9a8b9e214`）ため、差は「版の差」と「OS/toolchain の差」が混ざる。本ラウンドで言えるのは次の2点だけである。

- 第2ラウンドで解消済みだった D-1・D-2・D-4・D-5・D-7・D-8 は、Windows でも同じ結論（同符号・同終了コード・同欄の存在）になった。
- 第2ラウンドで残っていた D-3 は `view` の最上位 `worldFingerprint` で解消、D-10 は `resultShape` と `prelude` で一部解消した。

**照合値の数値を第2ラウンドと直接比較しない**。同じ commit の別実行物・別OSで作った値であり、改行やtoolchainの差が指紋に影響し得る（#2399の扱いと同じ）。

## 6. 未確認（NOT RUN）

- 実DB・実Provider・実Host・実Target。
- MCP入口（#3400）。
- 意味の受理・承認・権限付与・本番配備。
- `view -execute` の実ファイル書込み（第2ラウンドで観測済み、本ラウンドは再実行していない）。
- D-5・D-6 の本ラウンドでの直接再実行。
- Repository Health 全体（PRごとのCIで確認し、この記録では実行していない）。
- `semanticsFingerprint` と `worldFingerprint` の差が、名前だけの問題か範囲の問題かの完全な整理（#3345の範囲）。

## 7. 再実行手順

```powershell
git checkout 9a8b9e214f5eeb6b9e91a9adec2fcc17d3fac211
go build -o "$env:TEMP\tei-r3-windows.exe" ./cmd/tei
& "$env:TEMP\tei-r3-windows.exe" commands
& "$env:TEMP\tei-r3-windows.exe" trace -workspace examples/xtea-lowcode -scope r3 `
  -subject-kind semantic-candidate-key -subject-id sales.order.order_identity `
  -direction both -mode summary
```

`version -json` の SHA-256 と `00-binary-identity.txt` の値が一致すれば、同じ実行物である。一致しない場合は、記録の数値を同じ実行物の観測として扱わない。

# `-state-boundary` を実装し、書込境界が実際に効くことを確かめた（S2）

[#4486](https://github.com/hat47x/tei-platform/issues/4486) の S2 を実装しました。**`tei self-build -state-boundary` を付けると、2回目のbuildが「root tree が読み取り専用で、渡したstateと成果物ディレクトリだけ書ける」mount namespace の中で走ります。**

## 実装

| 部分 | 内容 |
| --- | --- |
| `stateBoundaryLauncher` | `unshare -rm sh -c <script> sh` を組む。root を `remount,bind,ro` し、`ALLOW_STATE`・`ALLOW_ART` だけ `remount,bind,rw` で戻し、`ALLOW_FIRST_STATE` を空のtmpfsで覆い、`exec "$@"` |
| 起動 | 既存の `driveSelfBuildIn`（記録するbuilderは**`-child`のまま**。launcher は記録しない） |
| 入口 | `-state-boundary`（opt-in）。**`-rebuild-check` を暗黙に有効にする** |
| 報告 | `stateSeparation.basis` が **`host-boundary`**、`scope` は「したこと」だけ |

**allow path は namespace に入る前に作ります。** script は最初にrootをroにするので、その後では作れません。

## 実測1：境界が書込を実際に止める

同じ script 形を、allow path を正しく渡して駆動しました。

| 試したこと | 結果 |
| --- | --- |
| 許可外のパスへの書込 | **`Read-only file system` で失敗**（エラー出力へのリダイレクト自体が失敗した） |
| `ALLOW_STATE` への書込 | **成功** |
| `ALLOW_ART` への書込 | **成功** |
| 1回目のstate の中身 | **`FIRST_STATE_ENTRIES=0`**（元は1件。tmpfsで覆われ、**読取も見えない**） |
| namespace の後、許可外ファイル | **存在しない** |

**書込は閉じ、許可した2つは開き、1回目のstateは読めなくなっています。**

## 実測2：product の flag 経由で動く

| 実行 | 終了 | `claimed` | `repro` | `basis` |
| --- | --- | --- | --- | --- |
| 既定（境界なし） | 0 | `true` | `reproduced` | `entry-post-hoc` |
| **`-state-boundary`** | **0** | **`true`** | **`reproduced`** | **`host-boundary`** |
| **`-state-boundary` 単独** | **0** | **`true`** | **`reproduced`** | **`host-boundary`** |

**3行目が重要です。** `-rebuild-check` を書かなくても再現検査が走り、境界が使われています（flag のヘルプと実装が一致）。

## 独立レビューの指摘と対応（前段の実装から継続）

| 指摘 | 判定 | 対応 |
| --- | --- | --- |
| `-state-boundary` が `-rebuild-check` を有効にせず、単独で黙って無視される | **実在** | 修正。上の3行目で確認 |
| `scope` が機構より強く、`/proc`・`/sys`・`/dev` は書き込み可能なまま | **実在** | 文言を「rootと別マウントは `not covered`」へ変更。**試験で固定** |
| 引数名の綴り誤り | **誤り** | 実物は正しい。**直していない** |

## 実装担当の分担（実測の記録）

`stateBoundaryLauncher` の初稿は Step Code（`step-5-preview`）に `implementer` として書かせました。**その出力には `exec.LookPath(unshare)` という未定義変数のバグがありました**（Go では識別子 `unshare` は存在しない）。**そのままではコンパイルが通らないため、修正して採用しています。** 生成物をそのまま信じない手順を踏みました。

## 限界（明示）

- **読取の共有は閉じていません。** 境界が閉じるのは**書込による共有**だけです
- **root と別にマウントされたファイルシステム（`/proc`・`/sys`・`/dev`）は対象外**。`scope` に `not covered` と書きました
- ネットワーク・常駐Process・環境変数は境界外
- **opt-in** で、既定の挙動は変えていません
- `linux/amd64`（WSL2）での観測。**`darwin`・Windows は対象外**（mount namespace は Linux 限定）
- **`qualified` は `false`**、G4 は `UNQUALIFIED` のまま

## 検証

`gofmt`（変更3ファイルは整形済み）、`go vet ./cmd/tei` 通過、`TestStateBoundaryLauncherShape` **PASS（skipではない）**、`TestStateBoundaryScopeNamesWhatIsNotCovered` PASS、既存の state 分離・再現性試験 PASS。

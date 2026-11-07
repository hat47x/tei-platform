# 構成Kernel移行Gateの最初の実証パケット — 実行記録（2026-11-07）

- 対象: [構成Kernelの移行判定](../../gate/COMPOSITION_KERNEL_MIGRATION_GATE_2026-10.md)の「4.1 統合済み部品から始める最初の実証パケット」
- 位置づけ: 同節のA・B・C三経路を、**同じ固定checkoutで別々に**実行した記録。Gateの合否判定ではない。
- 実行環境: Windows 10.0.26100、Go 1.27.0（windows/amd64）。詳細は`environment.txt`。
- 実行HEAD: `18cf8ae0f5cef085a441444364353bcadd72e9aa`（当時の`origin/main`）
- 実行前の作業領域: `git status --porcelain=v1 --untracked-files=all` が0行（本ディレクトリの出力を書く前）

この記録は上のHEADで取得したものです。PRはその後に進んだ`main`へ載せたため、**記録が指す版とPRが載る版は同じではありません**。下の観測値は`18cf8ae0`時点の値であり、後の`main`で同じ値になることを示しません。

このディレクトリのファイルが生データです。SHA-256は第5節にまとめました。

## 1. 何をしたか

[Gate 4.1](../../gate/COMPOSITION_KERNEL_MIGRATION_GATE_2026-10.md)が指摘する二重Load（`cmd/tei/main.go`の分岐前`loadWorld`と、`operations.SurfaceWorkspace`／`artifact.RenderWorkspace`の内部Load）を避ける修正はせず、**現行のまま**実行しました。三経路は別プロセスで、出力は`cmd /c`のリダイレクトで受けています（PowerShellの`Out-File`を経由するとBOM付きUTF-16になるため）。

| 経路 | 実行 | 出力 |
| --- | --- | --- |
| A（Source→Operation→Artifact） | `go run ./cmd/tei validate -workspace reference/standard-crud/workspace`、`source-index`、`operations -application ledger`、`artifacts -application ledger`、`artifacts … -check` | `validate.txt`、`source-index.json`、`operations.json`、`artifacts.json`、`artifacts-check.json` |
| A（既存試験との照合） | `go test ./internal/operations -run '^TestSurfaceWorkspaceStandardCRDFixture$' -count=1 -v` | `test-operations.txt` |
| B（保護Web Action→Service Carrier） | `go test ./reference/web-composition/go-host -run '^TestCanonicalQuoteActionUsesHostOwnedGenerationServiceRoute$' -count=1 -v` | `test-web-composition.txt` |
| C（Provider Read→Consumer Result） | `go test ./internal/driver -run '^TestD2DriverRechecksAuthorityAfterCompleteSourceRead$' -count=1 -v` | `test-driver.txt` |

すべての標準エラー出力（`*.err.txt`）は0バイトで、各コマンドは終了コード0でした。

## 2. 観測した値

`validate`: `ok`。

`source-index`（`tei.cli.source-index/v0`）: Source 9ファイル、参照17件。参照のSubject種別は`application`・`binding`・`database`・`persistence`・`semantic`・`semantic-attribute`・`semantic-capability`・`semantic-data`・`semantic-type`の9種。

`operations`（`tei.operations.operation-surface/v0`、`effect: read-only`）:

- `worldFingerprint` = `sha256:bd53dbac…1817f`
- `modelRevision` = `sha256:c60ed68c…b04d`
- 規則 = `tei.operation-surface/standard-crud` / `v1` / `sha256:92adf0fa…9ae0`
- 操作5件（すべて`origin: derived`）: `list.sales.note`・`list.sales.order`・`read.sales.order`は`unattended`、`create.sales.order`・`update.sales.order`は`attended`
- 欠落・衝突4件: `sales.note`の`create`と`read`は`missing-semantics`、`sales.order`の`list`は`conflict`（datasource `archive`がwrite-only）、`retire`は`missing-input`

`artifacts`（`-check`付きも出力は同内容で終了コード0）:

- artifact 5件: `function-definition`×1、`operation-surface`×1、`table-definition`×3
- `inputRevision`は`operations`の`modelRevision`と同一値（`sha256:c60ed68c…b04d`）
- `manifest.complete` = `false`、`manifest.digest` = `sha256:b4fa1ee8…584e`、`findings` 0件、未解決の合計6（function-definition 1、operation-surface 4、table-definition 1）
- `-check`は「整合しない場合に失敗する」旗であり、`complete=false`を失敗にしない。ここでは終了コード0

試験: `TestSurfaceWorkspaceStandardCRDFixture`（`--- PASS`）、`TestCanonicalQuoteActionUsesHostOwnedGenerationServiceRoute`（`--- PASS`）、`TestD2DriverRechecksAuthorityAfterCompleteSourceRead`（親1件とサブケース2件が`--- PASS`）。3パッケージとも`ok`。

### 依存の静的な分離（`go list -deps`）

「AとBは同じ業務Requestを通っていない」を、実行結果ではなく**パッケージ依存の形**でも確かめました。`dependency-matrix.txt`は三つの実行単位について、主要な内部パッケージが依存閉包に入るか（true/false）を示します。生の閉包は`deps-*.txt`です。**この観測は上のCLI実行とは別のcheckout（作業コミット`d7a7ad767`、親`b8aee3320`）で取った**ため、版が違います。`main`が`18cf8ae0`から`b8aee3320`へ進んだ差分（`cmd/tei-studio/README.md`と`tools/reference-first-web-frozen-world-generate/`）は、下の三つの実行単位のどれにも含まれません。該当パッケージのimportがこの間に変わった場合、表の値も変わり得ます。

| 実行単位 | `internal/composition` | `internal/operations` | `internal/plugin/runtime` |
| --- | --- | --- | --- |
| `cmd/tei`（AのCLI） | false | true | true |
| `reference/web-composition/go-host`（B） | true | false | false |
| `tools/reference-first-web-target`（Bの起動側） | true | false | false |

（`false`は「その実行単位の依存閉包に含まれない」の意で、パッケージが存在しないという意味ではありません。）

- Aの実行単位`cmd/tei`は`composition`を含まず、Bの実行単位（`go-host`と`reference-first-web-target`）はどちらも`operations`を含みません。**この三つの実行単位のうち、AとBの両方の閉包を同時に持つものはありません**。リポジトリ全mainパッケージについては、この三つ以外を同じ方法で調べていません。
- `internal/servicehost`（`SelectedServiceCarrierInvoker`）は、非試験のファイルからは**どこからもimportされていません**。importしているのは`internal/driver`・`reference/web-composition/go-host`の試験3ファイルだけです。したがってCの`TestD2DriverRechecksAuthorityAfterCompleteSourceRead`が通ることと、Service Carrierが実Hostから呼ばれることは別です。
- `pruntime.ServiceCarrierBinding`を非試験で設定しているのは`internal/plugin/runtime`自身だけで、Host選択の実装は試験側にあります。

これは**静的なimport関係の観測**であり、実行時に同じRequestが流れたか、Provider候補が共有されたかを示しません。依存の形が変われば結論も変わります。

## 3. この記録が示すこと

- A経路で、`operations`の`modelRevision`と`artifacts`の`inputRevision`が**同じ値**になったこと。両コマンドは`reference/standard-crud/workspace`の同じパスを読み、Source Indexの9ファイルも同一Workspaceから得たものです。
- `validate`・`source-index`・`operations`・`artifacts`が、**同じ固定HEADで成功した**こと。
- CLIが出した操作面（操作5件・欠落/衝突4件）が、`TestSurfaceWorkspaceStandardCRDFixture`の期待と矛盾しないこと。
- BとCが、Aとは**別の試験として**成功したこと。

## 4. この記録が示さないこと

- **A・B・Cが同じ業務Requestを通ったこと**。Bの`order_entry.preview_order_quote`とAの`ledger.update.sales.order`を、同一Contract・同一業務Actionとして接続していません。[Gate 4.1](../../gate/COMPOSITION_KERNEL_MIGRATION_GATE_2026-10.md)の「製品としての次の一辺」は未解決のままです。
- **正式なOwner受理・Host current Route・実DB Effect**。`ledger.update.sales.order`は候補であり、書込み・Receipt・実DBの効果はE3（#1174）の担当です。
- **実Provider・実SUI・実Browser・実HTTP送出**。Bは合成TLS受信先と合成Scopeまで、Cは合成D2試験までの範囲です。
- **二重Loadが同じSource bytesを読んだことの証明**。`operations`と`artifacts`は別プロセス・別Loadであり、`modelRevision`の一致は同じパスを読んだ結果の一致を示すだけです。二重Loadの解消も未実施です。
- **Gate G1〜G4の合否**。本記録は4.1の実行記録であり、[移行判定](../../gate/COMPOSITION_KERNEL_MIGRATION_GATE_2026-10.md)の各Gateを合格にするものではありません。
- **`tools/repository-health-check.sh`の全項目**。この実行はその代替ではありません。

## 5. 生データ

| ファイル | SHA-256 | バイト数 |
| --- | --- | --- |
| `environment.txt` | `e2c05cd131a7ff25e5b0b0da539256ab9bab7eb4316b7ef8f789ddb1e13acd12` | 422 |
| `validate.txt` | `dc51b8c96c2d745df3bd5590d990230a482fd247123599548e0632fdbf97fc22` | 3 |
| `source-index.json` | `aee8c1ce4082383c75ce62c600ec90ba2d075cc5cff14916534fdbb3291779cc` | 9604 |
| `operations.json` | `e6a8443c13d8715f2015de58d0abf25691065d8d1c8a42a0474723692622d171` | 3268 |
| `artifacts.json` | `5ca9d4b49bb3e456578e41a535f5870e9a611993047e5f3c17a772d7ecb67c3f` | 9672 |
| `artifacts-check.json` | `5ca9d4b49bb3e456578e41a535f5870e9a611993047e5f3c17a772d7ecb67c3f` | 9672 |
| `test-operations.txt` | `d69e97d8aca6f517ca8f41f66cd5798749a9d75bf665f92705985952ba92d636` | 147 |
| `test-web-composition.txt` | `e5a2b97b6716ae53df08117591580163b1a57ada316561d87a25f9054ce0772a` | 203 |
| `test-driver.txt` | `183d62528abb7d75f56aee46f6bad1a17efbc76b424cbf0fb82cea59211fef40` | 551 |
| `dependency-matrix.txt` | `f190ba0dff9f47a666eca50d1086dfc8303d2fc47d2381f54faed35e836279ac` | 721 |
| `deps-cmd-tei.txt` | `02aa9bfdc1df61a5a2cec60efb17ef834107aeae45b3f43e5b6f5b006746c9b2` | 6678 |
| `deps-web-composition-go-host.txt` | `71e510a75f99378de4af70ad01acb735b7f7f9335f6bdcae92e0b5497b54db28` | 4949 |
| `deps-reference-first-web-target.txt` | `03098e8feb5d9cd82499f0bcf1a3bd2df5f6b9e248def2758903abf48bdb70b4` | 6768 |
| `validate.err.txt`・`source-index.err.txt`・`operations.err.txt`・`artifacts.err.txt`・`artifacts-check.err.txt` | `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`（空） | 0 |

`artifacts.json`と`artifacts-check.json`のSHA-256が同じなのは、`-check`が整合時には出力へ何も足さないためです。**旗が効いていることを、この同一ハッシュだけからは示せません**（不整合時の失敗は別途の試験が担当します）。

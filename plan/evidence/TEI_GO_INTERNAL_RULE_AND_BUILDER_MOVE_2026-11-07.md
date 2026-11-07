# Goのinternal規則と、`gobuilder`移動の影響範囲

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-C0で、`cmd/tei`から`tools/internal/gobuilder`を呼ぶ案（A）を検討した。本記録は、その前提が**Goのinternal規則に抵触する**ことを実測で確かめ、移動する場合の影響範囲を数えたものである。

**設計判断ではありません。** A-1（移動）とB（subprocess）のどちらを採るかは[#1090](https://github.com/hat47x/tei-platform/issues/1090)と#2831の所有者が決めます。

## 観測1：`cmd/`から`tools/internal/`はimportできない

`tei/tools/internal/gobuilder`は`tei/tools/`配下の`internal`である。Goのinternal規則では、`.../internal/...`は**その親ディレクトリの配下から**しかimportできない。`tei/cmd/tei`は`tei/tools/`の外なので届かない。

一時的なprobe（`tei/.tmp_probe/cmd`）を置いて実際に確認した。

~~~text
環境: WSL2 Ubuntu 22.04、Go 1.27.3 linux/amd64
対象: main 96d276b74 のworktree

$ go vet ./.tmp_probe/cmd
package tei/.tmp_probe/cmd
	.tmp_probe/cmd/probe.go:4:2: use of internal package tei/tools/internal/gobuilder not allowed
~~~

probeは確認後に削除し、worktreeへ残していない。

**読み取れること**：`cmd/tei`から`tools/internal/gobuilder`をimportする案Aは、そのままではコンパイルできない。
**読み取れないこと**：`tools/`配下から同じpackageをimportできるか（これは後述のとおり可能である）。

### 既存の禁止規則との関係

`internal/operations/dependency_test.go`は`tei/tools/`を禁止している。これは「Operationがtransport/toolingへ依存しない」ための**方針上の**禁止である。

本観測の拒否はそれとは**別の層**にある。Goのinternal規則はコンパイラのパス判定であり、`cmd/tei`が`tools/`を参照することを禁じるものではない。したがって「`internal/operations`の規則を緩めれば`cmd/tei`から呼べる」という読み方は成立しない。

## 観測2：移動した場合に書き換える範囲

`tools/internal/gobuilder`をモジュール直下の`internal/`配下へ移す案（A-1）について、旧pathを参照しているファイルを数えた（main `96d276b74`、`.git`を除く）。

| 種別 | 件数 | 内訳 |
| --- | --- | --- |
| Goソース（import文） | **16** | `tools/reference-*`のmain.go・environment.go、および既存の`tools/internal/gobuilder`内のテスト |
| Markdown（説明・Evidence） | **12** | `plan/design/` 5、`plan/evidence/TEI_GO_*` 4、`plan/workstreams/` 1、`tools/reference-*/README.md` 2 |
| ワークフロー定義 | **1** | `.github/workflows/tei-go-builder-windows-qualification.yml` の対象package指定 |

合計**29ファイル**である。

Goのimport文16件は機械的に置換できる。問題は残り13件で、次の性質を持つ。

- **Markdownの12件**は、コマンド例（`go run ./tools/reference-go-module-closure-preflight ...` や `go test ./tools/internal/gobuilder`）で旧pathを書いている。Evidence文書は「その時点で実行した手順」を記録しているため、**単純に書き換えると記録の意味が変わる**。移動したことを注記する形が必要になる
- **ワークフロー1件**はCIの実行対象packageを指す。移動と同時に直さないと、Windows qualificationが空振りする

**読み取れること**：A-1は16ファイルのimportと13ファイルの記述に触れる。「一度きりで機械的に追える」とは言えるが、Evidenceの記述は機械的に置換できない。
**読み取れないこと**：移動後にCI・Evidenceの実行手順がそのまま通るか（未実行）。

## 観測3：`tools/`配下からは移動後も届く

internal規則は「親ディレクトリ配下」を許す。移動先がモジュール直下の`internal/`であれば、`tei/tools/...`は`tei/internal/...`をimportできる（`tei/`の配下だから）。したがってA-1では、既存の`tools/reference-*`は新pathをそのまま使える。

この帰結は**規則からの推論**であり、移動を実施して確かめたものではない。

## まとめ：A-1とBの比較に足す材料

| 論点 | A-1（移動） | B（subprocess） |
| --- | --- | --- |
| `cmd/tei`からの到達 | 移動すれば可能 | 不要（child executableを起動） |
| 変更ファイル数 | 29（Go 16、Markdown 12、CI 1） | `cmd/tei`と契約文書のみ（実装は未着手） |
| Evidenceへの影響 | 実行手順の記述を注記する必要がある | なし |
| 駆動主体の証明 | 同一プロセス内でBuildするため、駆動と実行の区別が弱い | Stage1のTEIが起動したchildのdigestで示せる |
| 未確認の前提 | 移動後のCI・Evidenceの再実行 | `StdioProcessLauncher`をBuild用child起動へ流用できるか |

## 主張しないこと

- A-1を実施していない。移動後のビルド・CI・Evidenceは未実行である。
- Bで使う`StdioProcessLauncher`がBuild用child起動に使えるかを確かめていない。Plugin用のwire protocolを前提にしており、そのまま流用できる保証はない。
- 本記録は`cmd/tei`から`tools/`を呼ぶことの是非を決めない。判断は設計案と所有者のもの。

## 実行した検証

- `go vet ./.tmp_probe/cmd`（probe、確認後に削除）：上記の拒否を観測
- `grep -rl 'tools/internal/gobuilder' . --exclude-dir=.git`：29ファイルを列挙
- `go test ./tools/repository-plan-check -count=1`：PASS

**NOT RUN**：`gobuilder`の移動、移動後の全体ビルド・CI、Bの試作、SB-C0の実装。

## 参照

- 設計案：[`SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md`](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md) §3.1・§3.2
- 外部Module方針の判断材料：[`EXTERNAL_GO_MODULE_ADMISSION_CONTRACT_2026-11.md`](../design/EXTERNAL_GO_MODULE_ADMISSION_CONTRACT_2026-11.md)
- 判定の正本：[`docs/status.md`](../../docs/status.md)

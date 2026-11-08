# cmd/teiからの薄いBuild入口（SB-C0 §3.4 / §4.5 D-3）

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-C0で確定した**案B（`cmd/tei`からchild builderをsubprocess起動する薄い入口）**を実装した記録です。

**この入口はL3を主張しません。** 出力の`qualified`は常に`false`です。artifactができたことは、受理でも、閉包資格でも、Runtime Authorityでもありません。

## 決めたこと（設計文書の決定を実装に落とす）

| 判断 | 実装 |
| --- | --- |
| 案Bを採る（A-1の`gobuilder`移動は採らない） | CLIは`tools/internal/gobuilder`をimportせず、**明示されたchild実行ファイルを起動**する |
| D-3（catalogへ載せない） | `acceptedCommands`へ追加し、`cliCommandCatalog`へは**追加しない**。`tei self-build -h`が自分のflagを説明する |
| PATH解決をしない（SC0-09） | `-builder`は実在する通常ファイルのパスを要求。裸の名前は拒否 |
| no-clobber（SC0-04） | `-output`が既存なら、childを起動する前に拒否 |
| stateの分離（SC0-03） | `-build-state`を必須にし、既定値を持たない |
| 子の自称を信じない（SC0-07/10） | 子が`artifactDigest`を返しても、**ディスクから読み直したdigestと別欄で比較**する |

## 実装

`cmd/tei/cli_self_build.go`に`tei self-build`を追加しました。

```text
tei self-build
  -workspace   <path>   Build Intentの宣言元（必須。validateでerrorが無いことを要求）
  -intent      <id>     宣言ID（この入口では解決しない。記録のみ）
  -supply      <dir>    子へ渡すGOPROXY=file供給
  -build-state <dir>    試行ごとの隔離状態（必須・既定値なし）
  -output      <path>   artifact出力（既存なら拒否）
  -builder     <path>   exact child executable（必須・PATH解決しない）
  -child-args  <json>   子へ渡す追加引数（JSON配列）
```

子へは`-artifact-out <output> -state <build-state>`を渡します。これは既存`tools/reference-go-build-with-supply`が受け取る契約と同じです。

### 拒否条件（Buildを開始しない）

| 条件 | コード | 終了コード |
| --- | --- | --- |
| `-workspace`が無い | `INVALID_ARGUMENT` | 2 |
| `-build-state`が無い | `SELF_BUILD_PRECONDITION_FAILED` | 2 |
| `-output`が無い／既存 | 同上 | 2 |
| `-builder`が無い／裸の名前／実行不能 | 同上 | 2 |
| Workspaceが読み込めない、または`validate`がerrorを返す | 同上 | 2 |
| 子が非0で終了 | `SELF_BUILD_CHILD_FAILED` | 1 |
| 子は成功と言ったがartifactが読めない | 同上 | 1 |

**検査の順序**：`-builder`の妥当性はWorkspaceの読込みより**先**に確かめます。最初はWorkspace読込みを先にしたため、裸の名前の拒否を試す試験が「workspace load failed」で落ちました。flagの誤りがWorkspaceの問題に隠されないよう順序を入れ替えています。

## 実測（2026-11-07 / WSL2 Ubuntu 22.04・Go 1.27.3 linux/amd64）

`reference/workspace`を渡し、childに`tools/reference-go-build-with-supply`（`./tools/reference-build-authoring-intent`を対象）を起動しました。

| 欄 | 値 |
| --- | --- |
| `ok` | `true` |
| `child.executableDigest` | `sha256:7692c4772974646d87975e488eba4acb99ff9430ba1b63ae0c573149ff8e1715` |
| `child.exitCode` | `0` |
| `artifact.digest` | `sha256:5bb9f33bfb33465fc0b89e3853104d98d57372d2923f75e63523cc92025ebd70` |
| `artifact.bytes` | 5,091,915 |
| `artifact.childReportedDigest` | 上と同じ |
| `artifact.childReportedDigestMatches` | `true` |
| `qualified` | **`false`** |

CLIは`cmd/tei`自身であり、childは別バイナリです。**駆動主体と被駆動主体が別digest**であることを、入口の出力として残せています。

## 退け方を確かめた入力

| 入力 | 実測 |
| --- | --- |
| `-workspace`なし | `INVALID_ARGUMENT`、child未起動 |
| `-output`が既存 | `SELF_BUILD_PRECONDITION_FAILED`、child未起動 |
| `-build-state`なし | 同上（共有stateを「二回」と数えない） |
| `-builder go`（裸の名前） | 同上（digestを記録できない起動を許さない） |
| Workspaceが読めない | 同上 |
| 子が誤った`artifactDigest`を返す | `childReportedDigestMatches=false`（ディスク実物と比較している） |

## 検証

- `gofmt -l cmd/tei`：差分なし
- `go vet ./cmd/tei`：PASS
- `go test ./cmd/tei -count=1`：PASS（既存のdiscovery・catalog・dispatch試験を含む）
- 実プロセスでのend-to-end：上表のとおり成功

**NOT RUN**：`-intent`の解決（この入口はIDを記録するだけで解決しない）、2回の独立試行、`cmd/tei`を対象にしたL3、Seed Trust、CI。

## 限界

- **`-intent`を解決していません。** SB-C0 §4.2の拒否条件2（Build Intentが解決できない場合）は、この薄い入口では未実装です。Intentの解決方式は設計文書の未決事項のままです。
- **L2（TEI-defined build）は成立しません。** Build Intent/Graphから実Buildへの結線はまだ無く、この入口はchildへ引数を渡すだけです。
- catalogへ載せていないため、`tei commands`からは見えません（D-3）。呼び手は`-h`か文書で知ります。
- 供給のTrust・署名は未検証です。G4は`UNQUALIFIED`のままです。

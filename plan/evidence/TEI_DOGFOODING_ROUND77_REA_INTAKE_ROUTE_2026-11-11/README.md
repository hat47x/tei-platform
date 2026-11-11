# 実行器はあるのに振り分けが無い（2026-11-11、第77ラウンド）

- 対象: 第76ラウンドで「未接続」と記録した `rea-intake`。
- 前段: [第76ラウンド](../TEI_DOGFOODING_ROUND76_MAIN_GREEN_2026-11-11/README.md)。
- 実行日: 2026-11-11〜12（Asia/Tokyo）。
- 状態: **`rea-intake` が動くようになった。14件の FAIL が 0 になった。**

## 1. 目的

第76ラウンドで `cmd/tei` の `rea-intake` 6試験（サブテスト込み14件）が FAIL すると記録した。**原因を特定して直す。**

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `d9a14cddf`（`origin/main`） |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.26.11 linux/amd64` |

## 3. 原因: 実行器は存在するが、振り分けが無い

```go
func runThreeElementCommand(args []string, out io.Writer) (int, error) {
	switch sub {
	case "validate": ...
	case "crud": ...
	case "cross-reference": ...
	case "diff": ...
	default:
		return failThreeElement("INVALID_ARGUMENT", fmt.Sprintf("unknown three-element subcommand %q", sub), 2, out)
	}
}
```

`runThreeElementReaIntake` は `cli_three_element_rea.go` に**実装済み**で、テストも12箇所から呼んでいる。**しかし `switch` に `rea-intake` の case が無い。**

```text
{"schemaVersion":"tei.cli.three-element/v0","ok":false,
 "error":{"code":"INVALID_ARGUMENT","message":"unknown three-element subcommand \"rea-intake\""}}
```

## 4. なぜ見落とされたか

**テストが実行器を直接呼んでいたからである。** `runThreeElementReaIntake` を直接叩けば振り分けを通らないので、case が無くても気づかない。**今回 FAIL したのは、テストが `runThreeElementCommand` 経由で呼ぶよう書かれていたから**である。

つまり**テストの書き方のほうが正しく、実装が遅れていた**。

## 5. 修正

```go
case "rea-intake":
	// The executor existed but was never routed here, so every rea-intake
	// invocation answered "unknown subcommand" while the tests expected it
	// to run. The missing case is why that gap went unnoticed: a test that
	// calls the executor directly never touches the dispatcher.
	return runThreeElementReaIntake(args[1:], out)
```

usage 行にも `rea-intake` を足した。

### 5.1 実測

```text
修正前: rea-intake の FAIL 14件
修正後: 0件
```

## 6. 変異試験

| # | 変えた箇所 | 結果 |
| --- | --- | --- |
| M1 | case を削る（修正前の挙動） | FAIL **14件**（修正前と一致） |
| M2 | 別の実行器（`validate`）へ振る | FAIL `flag provided but not defined: -evidence` |

**M2 が示すのは、case を足すだけでは足りない**ということ。**正しい実行器へ振る**必要がある。

## 7. 併せて見つけた: `rea-evidence` は実装自体が無い

同じく FAIL していた2件（`TestDiscoveryAdvertisesEveryModeAResponseCanCarry`・`TestDiscoveryWriteFlagsMatchObservedFileEffects`）は、**`rea-evidence` という未実装コマンド**を参照していた。

```text
grep -rn "runReaEvidence" --include="*.go" cmd/tei/ | grep -v _test   -> 空
```

**こちらは `rea-intake` と違い、実装が存在しない。** テストだけが先にある。**振り分けを足すだけでは直せない**ので、本ラウンドでは直していない（範囲外）。

## 8. gofmt は直さず記録した

`gofmt -l .` が main で **6ファイル以上**を指摘した（`cmd/tei/cli_relation_query.go`・`internal/database/sqlgen/*` ほか）。1ファイルだけ直すのは**恣意的**なので、**記録だけした**。`repository-health` は gofmt を実行していないので CI には影響しない。

## 9. 回帰確認

| 対象 | 結果 |
| --- | --- |
| `./cmd/tei -run TestThreeElement` | ok |
| `go vet ./cmd/tei` | 0 errors |
| `gofmt -l cmd/tei` | 変更ファイルなし |

## 10. 読み取れないこと（NOT RUN）

- `rea-evidence` コマンドの実装（テストは12箇所が期待）。
- gofmt 違反6ファイルの整形（`repository-health` が見ていないため優先度は低い）。
- 実DB・実Provider・実Host、HTTP・Desktop。

## 11. 位置付け

**「実装はあるが配線が無い」を直した回**である。実行器は完成していたが、誰も呼べなかった。

**テストが実行器を直接呼んでいたために見落とされた**、というのが本質である。**入口から呼ぶテストが無い機能は、存在しても存在しないのと同じ**になる。

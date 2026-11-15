# CLI が解決の拒否理由を、prose の外へ出せるようになった（2026-11-11）

- 対象: [`cmd/tei/cli_result.go`](../../../cmd/tei/cli_result.go)、[`internal/plugin/diagnostics.go`](../../../internal/plugin/diagnostics.go)
- 位置づけ: **製品コードの変更と実行記録**。
- 計測した版: `raw/head.txt`（`5fbff63a` の作業ツリー＋このPRの変更）。

前のPR（#4245）で resolver が自分の拒否理由を `code` として返すようになりました。しかし **CLI 側の呼出し元は0件**でした。このPRは**その1点**を繋ぎます。

## 1. 何を変えたか

`cliFailure` に `resolveReason` を足し、**両方の解決失敗地点**が同じ helper を通るようにしました。

```go
type cliFailure struct {
    Code          string `json:"code"`
    Message       string `json:"message"`
    ResolveReason string `json:"resolveReason,omitempty"`
}
```

**`Code` も `Message` も変えていません。** 足したのは**より狭い分類**だけです。

## 2. 両方の地点で同じ形にした

| 地点 | 変更後 |
| --- | --- |
| `plugin-resolve` | `failResolve(...)` が `pluginResolveFailure` を通す |
| `plugin-runtime-plan` | 同じ helper を通す |

**片方だけ直すと、同じ失敗が2つのコマンドで違う形になります。** 揃えました。

## 3. `INVALID_PLUGIN_LOCK` には理由を付けていません

**lock が使えないのは入力の問題で、resolver の判断ではありません**（#3184 の分類）。したがって `resolveReason` は付かず、**既存の分類は変わりません**。

## 4. 観測

```text
--- PASS: TestPluginResolveFailureCarriesTheResolverReason
--- PASS: TestAnOrdinaryErrorCarriesNoResolveReason
--- PASS: TestTheReasonIsOmittedFromJSONWhenAbsent
```

3つ目が要点です。**理由が無いときは JSON に `resolveReason` が出ません**（`omitempty`）。つまり **`code` と `message` だけを期待する既存の読み手は、同じオブジェクトを見ます。**

## 5. この記録が示すこと

- **CLI の呼出し元が、resolver の分類を使えるようになりました。** 機械可読な `resolveReason` で、cycle と missing-plugin を **prose を読まずに**区別できます。
- **既存の契約は壊していません。** `code`・`message`・`INVALID_PLUGIN_LOCK` の扱いはそのままです。
- **足した欄は省略可能**なので、既存の読み手の見る形は変わりません。

## 6. この記録が示さないこと

- **`tei commands` の広告へ `resolveReason` を載せたこと。** 載せていません。
- **`ResolveDiagnostics` の連鎖を CLI が出すこと。** CLI は**最も外側の code** だけを出します。
- **`INVALID_PLUGIN_LOCK` に理由を付けるべきか。** lock の分類は別の型（`InvalidLockError`）で、resolver の code を持ちません。
- **`cmd/tei` の全試験**。解決に関わる範囲だけを実行しました（下記）。全試験は600秒で打ち切られ、これは clean main でも同じです。

## 7. 検証の範囲

```text
gofmt -l .                                                     0件
go vet ./cmd/tei                                               ok
go test ./cmd/tei -run 'Plugin|Lock|Resolve' -count=1           ok (407.8s)
```

**全試験は実行していません。** 実行時間の上限（約600秒）を超えるためで、**この制約は clean main でも同じ**です。

## 8. 作業環境の注記

作業中、**私の worktree が複数回、外部から削除されました**（`tei-wt-clr-20261111`・`tei-wt-clr2-20261111`・`tei-wt-clr3-20261111`）。**パッチの内容は同じものを再適用し**、最終的には**独立クローン**（`tei-clr-20261111`）で検証と証拠取得を行いました。**他の作業者の worktree には触れていません。**

## 9. 生データ

| ファイル | 内容 |
| --- | --- |
| `run.sh` | 再現手順（欄と helper・CLI code の不変・試験） |
| `raw/head.txt` | 計測した版とToolchain |
| `raw/pre-status.txt` | 計測前の作業領域 |
| `raw/anchors.txt` | 欄・helper・code の出力 |
| `raw/tests.txt`・`raw/tests-summary.txt` | 追加した試験の出力 |

## 10. 生データのSHA-256

| ファイル | SHA-256 | バイト数 |
| --- | --- | --- |
| `run.sh` | `5707b40144585b6c85f36a1f4ac75e45e86b602408aa0bdd4507b30badd0a942` | 1481 |
| `raw/head.txt` | `245dc449c8a091a97b3eea6510b777eaae18be2501f95925957d7a9cfcafb2ac` | 73 |
| `raw/pre-status.txt` | `e1c403052d83cd76042d49ca60fd8dcbb6c350450765c147f8a54c012168b380` | 111 |
| `raw/anchors.txt` | `e69a6b5e9e3ed3a1ef6166ab9264d190284b5c0749cfdc39bab915b2dcfe3e72` | 910 |
| `raw/tests.txt` | `987096127e34c8a918c836aad48e9a5b9d3e5904788acdfd1e715dee1851785b` | 377 |
| `raw/tests-summary.txt` | `22b59a7ece3e552caeb3fa85c08bc017c753b25c8266f2dc737e018384b46d5a` | 372 |

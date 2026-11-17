# `reads`の実データ観測（2026-11-17）

## 対象

- 実装: `internal/command/reference_store_attributes.go`（`AttributeValues`・`Keys`）、`internal/hook/observe.go`（`ObserveFromStore`・`PatternFor`）、`cmd/tei/cli_view_hooks.go`（`runDeclaredHooksHalfFrom`・`storeReads`）
- 試験: `internal/hook/observe_test.go`
- 対象版: `a44b0ed25`（`main`）
- 環境: Windows、Go `1.26.10`

## これまで何が起きていたか

`hook.ReadsFor`は、渡された値を`reads`宣言と**照合するだけ**でした。値をどこから得るかはホストの責務で、CLIは`-input`の値だけを渡していました。

そのため`after-*`の位相は、**適用後のレコードを読めませんでした**。`after-*`の意味は「変更後のレコードを見る」ことなのに、見ていたのは変更を起こしたリクエストそのものでした。

実測（同一fixture、`-reference-store`あり）:

| 版 | `hooks` |
| --- | --- |
| `main`（未改変） | `skipped:["after-create"]`、理由「no value was supplied for the declared read…: sales.order.status」 |
| 本ブランチ | `executed:["after-create"]` |

`-input`に`status=SHIPPED`（パラメータ名）だけを与え、属性名`sales.order.status`は与えていません。**ストアが属性として保持している値を読んだ**ため、位相が実行されました。

## 変更

1. `MemoryStore.AttributeValues(data, key)` — レコードの**属性ビュー**を返す。ストアは入力ビュー（パラメータ名）と属性ビュー（属性名）の2つを持ち、`reads`が名指すのは属性なので、引けるのは後者だけです。レコードが無い場合は`false`を返し、「属性が無いレコード」と「レコードが無い」を区別します。
2. `MemoryStore.Keys(data)` — キーを**昇順**で返す。キーを指定しない読み取りが、同じストアに対して常に同じレコードを見るようにするためです。
3. `hook.ObserveFromStore(store, pattern, declared)` — 宣言された属性だけをストアから引く。**保持していない属性は結果に入れません**。空文字を入れると`ReadsFor`が「観測された」と誤認するためです。
4. `hook.PatternFor` — ホストが同じパターン解決を使えるように公開。
5. `cmd/tei/cli_view_hooks.go` — `after-*`の半巡にだけストアを渡す。`before-*`は従来どおり入力の値を使います（まだ何も適用されていないため）。

`storeReads`は、宣言された属性について**ストアの値で上書き**し、ストアが持たない属性は入力の値を残します。これにより、レコードが書かれなかった実行では、従来と同じ「未観測」の報告になります。

## 実行

```powershell
go test ./internal/hook -run 'TestObserveFromStore' -count=1
go test ./internal/hook ./internal/command -count=1
go test ./cmd/tei -count=1
gofmt -l internal/hook internal/command cmd/tei
```

## 観測

- `TestObserveFromStore...` 4件 成功
- `internal/hook`・`internal/command` 成功
- 実プロセスでの比較（上表）で、`main`と本ブランチの差を確認

## 読み取れること

- `after-*`の位相が、適用後のレコードから`reads`を観測できる。**CLIとStudioの両方**で（Studioは`runHooksHalfFrom`がストアを渡す）。
- 宣言外の属性は、ストアが保持していても観測されない。
- ストアにレコードが無ければ、従来どおり未観測として報告される。

## この実装が正しいと言える範囲

**ストアにレコードが1件のときだけ**です。`ObserveFromStore`はキーを持たないため、複数レコードがあるストアでは**昇順先頭のレコード**を見ます（再現性はありますが、呼出し元が意図した行とは限りません）。

フック宣言は「どの行についてのフックか」を持っていません。`scope row`のフックは`RunRowCalls`が行ごとに値を渡しますが、レコード単位のフックにはその機構がありません。**どの行を見るかを決めるのは設計判断**であり、本記録はそれを先取りしていません。

したがって、複数レコードを持つパターンの実行結果を「フックが目的の行を読んだ」と読んではいけません。コードのコメントにも同じ制約を書いています。

## 読み取れないこと

- **実Provider（PostgreSQL等）からの読み取り**。観測するのは参照用のメモリStoreだけです。
- **複数レコードがあるときの、行の選択規則**（上記）。
- `produces`の実データ観測。提案は実装が返す値であり、データからの観測ではありません。
- キーを指定しない読み取りの**意味**。現在は昇順先頭のレコードを見ます。複数レコードがある一覧・明細のパターンで、どのレコードを見るべきかは設計判断です。
- Studioの**実画面**での操作確認（配線は`ExecuteCommandWithReferenceStore`→`runAfterHalfRevertingOnRefusal`→`runHooksHalfFrom`まで通し、`TestStudioAfterHalfReadsFromTheStore`で固定しました）。

## 位置付け

参照実装の確認記録であり、製品の受入や本番適格性を示しません。「`reads`がリクエストではなくレコードを読む」ことを、**参照ストアの範囲で**成立させた記録です。実Providerからの観測は未着手です。

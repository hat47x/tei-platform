# CLIの`-reference-store`配線の実プロセス確認と、引数検査の位置の修正（2026-11-11）

## 対象

- 実装: `cmd/tei/cli_view.go`（`-reference-store`の引数検査をモード分岐の前に移した）
- 試験: `cmd/tei/cli_view_reference_store_e2e_test.go`
- 対象版: `4b82484ea`（`main`）を基点とした作業ブランチ
- 環境: Windows、Go `1.26.10`（`go.mod`の版）

## 見つけた欠陥

`-reference-store`の引数検査が、**`-command`のモード分岐の内側**にありました。そのため次の呼出しは、フラグを黙って無視して成功します。

```text
tei view -workspace ... -application order_entry -pattern order_entry.order_record -reference-store -format json
```

実行結果（修正前）: `ok:true` で、`result.kind:"record"` の単票ビューが返る。`-reference-store`は実行の中身を変えるフラグなので、実行しない呼出しで受け取って捨てるのは誤りです。前段のPRでは単体の試験でしか確認しておらず、**flag解析から実際の経路まで通した試験がありませんでした**。

## 変更

1. 引数検査を`-command`分岐の**前**へ移した。フラグはビューのオプションではないので、どのモードも受け取って捨てられません。
2. 実プロセスを通す試験を3件追加した（ビルドした`tei`を起動する）。

## 試験（3件）

- `TestViewExecuteReferenceStoreReachesTheAfterHook` — `-execute -reference-store`で`before-create`と`after-create`が両方`executed`に入る。
- `TestViewExecuteWithoutReferenceStoreSkipsTheAfterHook` — `-execute`だけでは`after-create`が`executed`に入らず`skipped`に入る。
- `TestViewReferenceStoreWithoutExecuteIsAnArgumentError` — `-reference-store`のみで`INVALID_ARGUMENT`、終了コード2。**修正前は`ok:true`で落ちる試験**。

## 実行

```powershell
go test ./cmd/tei -run 'TestViewExecuteReferenceStoreReachesTheAfterHook|TestViewExecuteWithoutReferenceStoreSkipsTheAfterHook|TestViewReferenceStoreWithoutExecuteIsAnArgumentError' -count=1
go test ./cmd/tei -count=1 -timeout 40m
go vet ./cmd/tei
gofmt -l .
```

結果: 3件とも成功。`cmd/tei`の全試験は911秒で成功。`go vet`と`gofmt -l .`は無出力。

## 読み取れること

- CLIの実際の起動経路で、`-reference-store`が`after-*`まで到達する。
- 既定の`-execute`は変わらず`after-*`を理由つきでskipする。
- `-execute`を伴わない`-reference-store`は、どのモードでも`INVALID_ARGUMENT`（終了2）で拒否される。

## 読み取れないこと

- StudioのUIからの呼出し（未配線）。
- `after-*`拒否時の取り消しの**実運用**到達（規則Pluginの供給待ち）。
- 実Provider・実DBでの実行。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。単体試験だけでは届かなかった配線の穴を、実プロセスの試験で見つけて塞いだ記録です。

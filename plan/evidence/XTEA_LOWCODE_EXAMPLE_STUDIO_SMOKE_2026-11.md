# X-TEA例のStudioサーバー起動確認（2026-11-06）

## 対象

- 作業領域: `examples/xtea-lowcode`
- 対象版: `c32d9ed42`（`main`）
- 環境: Windows、Go `1.27.0`

## 実行

```powershell
cd cmd/tei-studio
$env:CGO_ENABLED = "0"
go build -mod=readonly -tags server -o "$env:TEMP\tei-studio-server.exe" .
```

```powershell
$p = Start-Process -FilePath "$env:TEMP\tei-studio-server.exe" `
  -ArgumentList "-workspace", "<repo>\examples\xtea-lowcode", "-port", "43123" `
  -PassThru
Invoke-WebRequest -Uri "http://127.0.0.1:43123/health" -UseBasicParsing
```

## 観測

- server buildは成功した（出力なし、終了コード0）。
- `/health`は`200`で、本文は`{"status":"ok"}`。
- 起動したプロセスは確認後に停止した。

## 画面資産

- `GET /`は`200`で、タイトルは`TEI Studio — read-only PoC A`。
- 配信HTMLに`view-pattern`・`view-package`・`command-periods`・`execute-command`の入力・ボタンが含まれることを確認した。
- `/workbench/`は別の作業画面で、これらのX-TEAビュー・Command入力を持たない。

## 未確認

- ブラウザー上での画面操作・表示結果。
- Wails RPC（`ListWorkspaces`、`ViewPattern`など）の実HTTP経路。
- Desktop WebView。
- 接続者認証、業務操作の認可、利用者間の分離。

## 位置付け

参照実装の起動確認であり、製品の受入、実Provider接続、実データベース、本番配備を示しません。Studioはloopback限定の読取り専用Hostです。

# UI Adapter境界の型付きハーネス（#1158 条件4・5）

- 担当：[Issue #1158](https://github.com/hat47x/tei-platform/issues/1158)
- 状態：**合成ハーネス**。実Web・実Wails/WebView2の実行ではない。

## 目的

同じ型付きCommand操作を、in-process・HTTP・local IPCの3経路で実行し、次を確認する。

1. 同じWorkspace・同じ検証済み主体・同じ要求で、業務結果が一致する
2. 無認証・権限なし・CSRFなしでは操作が実行されない
3. 古いrevisionは業務結果として拒否される
4. IPCでは主体をメッセージから受け取らず、Hostが固定する（メッセージで権限昇格できない）

## 仮決定（Owner仮承認・2026-11-06）

| 対象 | 仮決定 | このハーネスでの検証 |
| --- | --- | --- |
| Web | **HTTP（POST + JSON）**。認証済みセッション・CSRF・Action認可はHostが所有 | 実 `net/http`（httptest）で検証 |
| Desktop | **Wails v3 をHostの候補**とし、Go本体は Wails に依存しない。Desktop固有層は**型付きlocal IPC**で接続 | `net.Pipe`上の型付きIPCで検証。**Wails/WebView2は未実行** |

仮決定は実Web・実Wails・実OSで見直す。ここでのHTTPはTLS・Origin・Cookie・本番セッションを持たない。

## 実行

```sh
go test ./reference/interface-adapter
go vet ./reference/interface-adapter
```

## 確認できること

- in-process / HTTP / IPC の `Result` が一致する（`TestAdaptersAgreeOnTheSameOperation`）
- HTTP: Bearer欠落→401、viewer→403、CSRF不一致→403。いずれも操作は呼ばれない（`Invocations()==0`）
- HTTP: 古いrevision→409（Host通過後、操作内で拒否）
- IPC: Hostの主体が空→`UNAUTHENTICATED`、古いrevision→`STALE_REVISION`
- in-process: 主体なし→`UNAUTHENTICATED`
- in-processは既存`command.Invoker`＋`ReferenceExecutor`（CLIの参照実行経路）と同じ結末を返す

## 確認できないこと（条件4の区別）

- 実Webの認証・セッション・TLS・Origin・CSRF（本ハーネスのトークンとCSRFは固定fixture）
- 実Wails v3・WebView2・Windows/macOS/LinuxのDesktop実行、配備・署名・自動更新
- CLIバイナリ実行とHTTP/IPCの縦断（CLI↔in-processは`cmd/tei/cli_inprocess_parity_test.go`が担当）
- 実ProviderのEffect、受理済み意味、監査・保持

## 非目標

- 第二のSource Authority・Plugin Resolver・Application Driver
- 本番HTTPサーバ・認証基盤の実装
- Wails依存のGo本体への持ち込み

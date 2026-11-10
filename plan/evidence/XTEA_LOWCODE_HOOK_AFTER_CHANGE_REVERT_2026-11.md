# `after-*`拒否時の効果の補償の確認（2026-11-10）

## 対象

- 実装: `cmd/tei/cli_view.go`（`effectReverted`、スナップショットと復元）、`cmd/tei/cli_view_hooks.go`（`runAfterHalfRevertingOnRefusal`、`runDeclaredHooksHalf`の規則の型を`hook.Implementation`へ広げた）
- 試験: `cmd/tei/cli_view_after_revert_test.go`
- 対象版: `fe6aa3034`（`main`）を基点とした作業ブランチ
- 環境: Windows、Go `1.26.10`（`go.mod`の版）

## 背景

前段（[CLIから`after-*`フックへ到達する配線](XTEA_LOWCODE_HOOK_AFTER_CHANGE_CLI_2026-11.md)）で、`-reference-store`により効果をメモリ上のStoreへ適用し、変更後のフックを実行できるようにしました。ただし**`after-*`が拒否しても適用を戻さない**点を未実装として残していました。これを処理します。

## 実装したこと

`-reference-store`の実行で:

1. コマンド成功後、Storeのスナップショットを取る（適用後の状態）。
2. `after-*`を実行する。
3. 拒否されたら**スナップショットから復元**し、応答に`effectReverted: true`を載せ、テキストには「effect: reverted（変更後の位相が拒否したため）」と出す。

応答の`effectReverted`は、「適用した後に戻した」唯一の状態を示します。これを出さないと、報告された状態が「変更が残った」のか「戻した」のか読めません。

## 到達性について（正直な記録）

**CLIの既定の参照規則（`hook.ReferenceRule{}`）は決して不成立を返しません。** そのため、CLIの実運用では`after-*`の拒否が起きず、この復元経路は既定では到達しません。

それでも実装したのは、`after-*`を拒否できるのは**規則Pluginが供給する規則**であり、その配線（`AC-2`の「規則Pluginの契約と束縛」）が入ればこの経路がそのまま効くからです。到達しないまま「実装した」と書かないよう、次の2つを行いました。

- `runDeclaredHooksHalf`の規則の型を`hook.ReferenceRule`（具象）から`hook.Implementation`（インターフェース）へ広げ、**拒否する規則を差し込める**ようにした。
- 拒否する規則（`refusingRule`）を使う試験で、復元が実際に起きることを固定した。

## 試験（2件）

- `TestAfterChangeRefusalPutsTheStoreBack` — Storeへ効果を適用し、空のスナップショットを「実行前の状態」として渡し、拒否する規則で`after-*`を走らせる。結果: エラーあり・`reverted=true`・**Storeから変更が消えている**・報告の`Refused`が`after-create`。
- `TestAfterChangeWithoutRefusalKeepsTheChange` — 参照規則（拒否しない）では`reverted=false`で、適用した変更が残る。

## 実行

```powershell
go test ./cmd/tei -run 'TestAfterChangeRefusalPutsTheStoreBack|TestAfterChangeWithoutRefusalKeepsTheChange' -count=1
go test ./cmd/tei -count=1 -timeout 30m
go test ./internal/hook ./internal/command ./internal/studiohost ./internal/validation -count=1
go vet ./cmd/tei
gofmt -l .
```

## 読み取れること

- `after-*`が拒否したとき、適用した効果がStoreから取り消され、応答がその事実を述べる。
- 拒否しない実行では、適用した変更が残る。

## 読み取れないこと

- **CLIの既定経路でこの復元が起きる状況**。参照規則が拒否しないため、規則Pluginの供給（`AC-2`の「規則Pluginの契約と束縛」）が入るまで到達しません。
- **手順の補償**。効果そのものは戻しますが、`after-*`が呼んだ能力の効果（`uses`経由）は戻しません。外部効果の補償は別の設計が要ります。
- Studio側の同じ扱い。
- 実Provider・実DBでの取り消し（Storeの`Restore`はメモリ上だけ）。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。`after-*`拒否時の効果の取り消しを、**拒否する規則を差し込める形にして試験で固定した**記録です。既定経路での到達は規則Pluginの供給待ちであることを明記しています。

# Studio経路での`after-*`拒否時の取り消しの確認（2026-11-11）

## 対象

- 実装: `internal/studiohost/command.go`（`CommandRunValue.EffectReverted`、スナップショットと復元）、`internal/studiohost/hooks.go`（`runAfterHalfRevertingOnRefusal`、`runHooksHalf`の規則の型を`hook.Implementation`へ広げた）
- 試験: `internal/studiohost/command_after_revert_test.go`
- 対象版: `b54bd8e16`（`main`）を基点とした作業ブランチ
- 環境: Windows、Go `1.26.10`（`go.mod`の版）

## 前段で残っていたこと

CLIは`-reference-store`で`after-*`へ到達し、拒否時は効果を戻して`effectReverted`で示します（[確認記録](XTEA_LOWCODE_HOOK_AFTER_CHANGE_REVERT_2026-11.md)）。Studioは`ExecuteCommandWithReferenceStore`で到達できるようになりましたが（[確認記録](XTEA_LOWCODE_HOOK_AFTER_CHANGE_STUDIO_2026-11.md)）、**拒否時に取り消しの表明を出さず、Storeを捨てるだけ**でした。これを揃えます。

## 実装したこと

- `runAfterHalfRevertingOnRefusal`を`internal/studiohost`に追加。`after-*`が拒否したらStoreをスナップショットから復元し、`reverted`を返す。
- `CommandRunValue.EffectReverted`（`effectReverted`）を追加し、応答で表明する。
- `runHooksHalf`の規則の型を`hook.ReferenceRule`（具象）から`hook.Implementation`（インターフェース）へ広げ、**拒否する規則を差し込める**ようにした（CLIと同じ形）。

## 到達性について（正直な記録）

**Studioの既定の参照規則も決して不成立を返しません。** そのため実運用ではこの復元経路に到達しません。到達しないまま「実装した」と書かないよう、拒否する規則（`studioRefusingRule`）を使う試験で復元が実際に起きることを固定しました。`after-*`を拒否できるのは規則Pluginが供給する規則で、その配線が入ればこの経路がそのまま効きます。

## 試験で踏んだ誤り

最初の試験は「拒否する規則を使ったのにエラーが返らない」で落ちました。原因は**フックが宣言した`reads`に値が無く、位相が規則へ届く前に「未評価」で止まっていた**ことです。規則が判定するのは読み取りが観測された後なので、読み取りに値を与えて直しました。この順序（未評価 → 判定）は設計どおりで、試験側の前提が誤っていました。

## 試験（2件）

- `TestAfterChangeRefusalPutsTheStudioStoreBack` — 効果をStoreへ適用し、空のスナップショットを「実行前の状態」として渡し、拒否する規則で`after-*`を走らせる。結果: エラーあり・`reverted=true`・**Storeから変更が消えている**・報告の`Refused`が`after-create`。
- `TestAfterChangeWithoutRefusalKeepsTheStudioStore` — 参照規則（拒否しない）では`reverted=false`で、適用した変更が残る。

## 実行

```powershell
go test ./internal/studiohost -run 'TestAfterChangeRefusalPutsTheStudioStoreBack|TestAfterChangeWithoutRefusalKeepsTheStudioStore' -count=1
go test ./internal/studiohost ./internal/hook ./internal/command ./internal/validation -count=1
go build ./...
gofmt -l .
```

## 読み取れること

- Studio経路でも、`after-*`が拒否したときに適用した効果がStoreから取り消され、応答がその事実を述べる。
- 拒否しない実行では、適用した変更が残る。
- CLIとStudioで、拒否する規則を差し込める形と表明が揃った。

## 読み取れないこと

- **Studioの既定経路でこの復元が起きる状況**（規則Pluginの供給待ち）。
- デスクトップUIからの呼出し（`ExecuteCommandWithReferenceStore`をUIはまだ呼びません）。
- `uses`経由で呼んだ能力の効果（呼出しはデータを適用しないため戻す対象が無い。[確認記録](XTEA_LOWCODE_HOOK_USES_COMPENSATION_2026-11.md)）。
- 実Provider・実DBでの取り消し（`Restore`はメモリ上だけ）。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。Studio経路の取り消しの表明を、拒否する規則を差し込める形にして試験で固定した記録です。既定経路での到達は規則Pluginの供給待ちであることを明記しています。

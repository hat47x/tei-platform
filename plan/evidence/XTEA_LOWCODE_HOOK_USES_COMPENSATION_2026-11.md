# `uses`経由で呼んだ能力の効果に補償が要らないことの確認（2026-11-10）

## 対象

- 読取り対象: `cmd/tei/cli_view_hooks.go`・`internal/studiohost/hooks.go`（フックの呼出し元の構成）、`internal/hook/command_caller.go`、`internal/command/reference.go`・`reference_store.go`
- 対象版: `b45cd40df`（`main`）
- 環境: Windows、Go `1.26.10`

## 背景

`AC-2`の残る未確認に「`uses`経由で呼んだ能力の効果の補償」がありました。`after-*`が拒否したとき、フックが`uses`で呼んだ能力の効果も戻す必要があるかを調べました。

## 実測した事実

### 1. フックの呼出し元は、データを適用しないExecutorを使う

CLIとStudioの両方が、フックの`uses`呼出しのために組む`command.Invoker`に`command.ReferenceExecutor{World: world}`を渡しています。

```text
cmd/tei/cli_view_hooks.go:        Executor:   command.ReferenceExecutor{World: world},
internal/studiohost/hooks.go:     Executor:   command.ReferenceExecutor{World: world},
```

`ReferenceExecutor`は能力と効果を**返す**だけで、記憶域へ適用しません（適用するのは`ReferenceDataExecutor`で、こちらは両ホストのフック経路で使われていません）。

### 2. `-reference-store`を付けても、フックの呼出しは同じ経路

CLIの`-reference-store`は**コマンド本体**のExecutorだけを`ReferenceDataExecutor`へ差し替えます。フックの呼出し元は別のInvokerを組み立てており、そちらは`ReferenceExecutor`のままです。

したがって、フックが`uses`で呼んだ能力の効果は、参照Storeにもどこにも適用されません。

### 3. 適用していないものは、戻す必要がない

`after-*`が拒否したときに取り消すのは、適用した効果です。フックの`uses`呼出しは何も適用していないので、**補償すべき効果がありません**。

## 判断

補償を実装しません。実装しても戻す対象が無く、「宣言は在るが実行では何もしない」種類の処理になります。

これは手抜きではなく**保守的な選択**です。フックの呼出しがデータを適用しないため、フックが呼んだ能力の効果が参照Storeへ混ざることはなく、`after-*`拒否時に「どこまで戻したか」が曖昧になりません。

将来フックの`uses`呼出しも参照Storeへ適用するなら、そのときは補償の単位（コマンド本体の効果とフックの呼出しの効果をどうまとめて戻すか）を決める必要があります。この記録はその判断を先取りしません。

## 読み取れること

- フックの`uses`呼出しは、コマンド本体の効果とは別の経路を通り、データを適用しない。
- `after-*`拒否時の取り消しは、コマンド本体の効果だけで足りる。

## 読み取れないこと

- フックの`uses`呼出しを参照Storeへ適用すべきか（意味の設計判断）。
- 実Pluginの`uses`呼出しが外部効果を持つ場合の補償（`CommandCaller`の外側の話）。
- 実Provider・実DBでの取り消し。

## 位置付け

参照実装の確認記録であり、製品の受入や本番適格性を示しません。補償が不要である理由を、呼出し経路の実測で示した記録です。

# 同じコマンドの効果を2プロセスが同時に適用した場合の整合（2026-11-11）

## 対象

- 読取り対象: `cmd/tei/cli_view.go`（`-reference-store`のStore生成範囲）、`internal/studiohost/command.go`（`NewMemoryStore`の生成範囲）、`internal/command/reference_store.go`（`MemoryStore`の排他とキー）、`internal/command/reference_failure.go`（`record-exists`）
- 対象版: `97591d578`（`main`）
- 環境: Windows、Go `1.26.10`（`go.mod`の版）

## 背景

`AC-5`の残る未確認に「同じコマンドの効果そのものを2プロセスが同時に適用した場合の整合」がありました。段階の排他（`lifecyclesqlite`）は確認済みですが、**効果の側**がどうなるかを実測しました。

## 実測した事実

### 1. 参照Storeは実行ごとに作られる

CLIもStudioも、**1回の実行の中で**`command.NewMemoryStore()`を作ります。

```text
cmd/tei/cli_view.go:   referenceStoreInstance := command.NewMemoryStore()
internal/studiohost/command.go:   referenceStore := command.NewMemoryStore()
```

どちらも関数のローカル変数で、実行を跨いで共有されません。応答もこの点を述べています（`persisted:false`・`persistence:"not-applicable"`・`duration:"none"`）。

### 2. 2つのプロセスが同じコマンドを適用しても、互いに干渉しない

同じ`create`を2回、それぞれ別プロセスで実行し、`-operation-log`で適用内容を記録しました（`-reference-store`あり）。

```text
run1 exit= 0 ok= True
run2 exit= 0 ok= True
se2-op1.json applied= ['order=ORD-1\x00sales.order.order_id=ORD-1']
se2-op2.json applied= ['order=ORD-1\x00sales.order.order_id=ORD-1']
```

**両方とも成功し、同じキーへ適用しました。** それぞれが自分のStoreへ書いたので、2つ目の`create`は「既に在る」と判定されませんでした。

### 3. 同じStoreの中では、重複createは拒否される

`TestMemoryStoreRejectsDuplicateCreate`が成功します。`MemoryStore`は`sync.RWMutex`で排他し、`record-exists`（`reference_failure.go`）で拒否します。

### 4. これは欠陥ではなく、参照実装の表明どおり

`persisted:false`・`persistence:"not-applicable"`は「この実行は永続へ書かない」という表明です。永続が無い以上、**プロセス間の整合を参照実装が保証する必要はありません**。整合を要するのは耐久Storeを持つ実装であり、その実装はまだ無い（`TEI_EFFECT_POSTGRES_DSN`を設定した場合だけ試験する経路があるが、今回の経路ではない）。

## 判断

「同じコマンドの効果を2プロセスが同時に適用した場合の整合」は、**参照実装では観測対象にならない**（永続が無いため）。新しい符号や試験を足していません。実装を持つ段階で、耐久Storeの一意性制約として扱うべき論点です。

この判断を「確認済み」と書かず、**「参照実装では対象外」**として残します。

## 読み取れること

- 参照実行のStoreは実行ごとに閉じ、プロセス間で共有されない。
- 同じコマンドを2プロセスが実行しても、互いのStoreへ影響しない（両方成功）。
- 単一Store内では重複`create`が`record-exists`で拒否される。

## 読み取れないこと

- **耐久Storeでの一意性**（同時に同じキーを作ろうとしたときにどちらが勝つか）。実装が無いので測れません。
- 複数プロセスが**同じ耐久Store**を共有した場合の結末と段階の整合。
- 楽観的排他・悲観的排他の選択（意味の設計判断）。

## 位置付け

参照実装の確認記録であり、製品の受入や本番適格性を示しません。「同時適用の整合」が参照実装では観測対象にならない理由を実測で示し、`AC-5`の残る未確認から本項を**実装待ち**として切り分ける根拠にします。

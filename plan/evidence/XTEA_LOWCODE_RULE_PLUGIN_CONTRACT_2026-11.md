# 規則Pluginの契約と束縛の現状（2026-11-11）

## 対象

- 読取り対象: `internal/hook/run.go`（`Invocation`・`Implementation`）、`internal/hook/calling.go`（`CallingImplementation`）、`internal/hook/guarded.go`（`Guarded`）、`internal/hook/reference.go`（`ReferenceRule`）、`internal/hook/reads.go`
- 設計文書: `plan/design/XTEA_LOWCODE_RECEPTION_ARCHITECTURE_2026-11.md`（D2・D2.1）
- 対象版: `94d131463`（`main`）／確認は`113f61aa8`で再実行
- 環境: Windows、Go `1.26.10`

## 背景

`AC-2`の残る未確認に「規則Pluginの契約と束縛」がありました。何が決まっていて、何が決まっていないかを実測しました。

## 実測した事実

### 1. 実装側の契約は存在し、境界に置かれている

```go
type Implementation interface {
	Run(ctx context.Context, invocation Invocation) error
}
```

`Invocation`は`Application`・`Pattern`・`Phase`・`Reads`（属性→値）だけを持ちます。**宣言された読み取りだけが渡ります**（`hook.ReadsFor`が保証）。実装はL4（規則Plugin・Provider）に置く想定で、`internal/hook`は境界だけを持ちます。

### 2. 能力を呼ぶ実装には、別の契約がある

```go
type CallingImplementation interface {
	Implementation
	RunWithCaller(ctx context.Context, invocation Invocation, caller Caller) error
}
```

`Runner`は実装がこの型を満たすときだけ、**宣言に制限した`Caller`**を渡します（`run.go`・`rows.go`・`sequence.go`の3経路）。宣言外の呼出しは実装の判断ではなく、`Declared`の層で拒否されます。

### 3. 参照実装が2つある

- `ReferenceRule` — 宣言した`Required`・`Positive`の属性だけを評価し、同じ入力に同じ結果を返します。能力は呼びません。
- `Guarded` — 判定が通ったときだけ宣言済みの能力を呼びます（`var _ CallingImplementation = Guarded{}`）。

**どちらも「参照実装」であり、外部から差し込む経路はありません。** CLI・Studioは`hook.ReferenceRule{}`を直接渡しています。

### 4. Pluginの契約へ載せる判断は、設計文書が先送りしている

設計文書（D2.1）は次のように書いています。

> アプリケーションの対話フックを表す型ではないため、フックをこの契約へ直接載せません。フックはL2の宣言とし、**本体の規則Pluginを別の契約で束縛するかを、D2の採用時に決めます**。

つまり「規則Pluginをどう発見し、どう版を束縛し、どう差し込むか」は**未決定**です。決まっているのは「フックの本体は`Implementation`を実装したもの」という境界だけです。

## 判断

**規則Pluginの契約（発見・束縛・版）は実装しません。** 設計文書が明示的に「D2の採用時に決める」と先送りしており、ここで決めると**未受理の設計判断を固定**することになります。

一方で、**境界は既にあり、差し込み口も存在します**（`Runner.Implementation`・`Runner.ByPhase`に任意の`Implementation`を渡せる）。CLI・Studioの`after-*`拒否時の取り消しは、この口へ拒否する規則を差し込むことで試験されています。

したがって本項は「未実装」ではなく、**「境界は実装済み、発見・束縛・版の契約は設計判断待ち」**として整理します。

## 読み取れること

- フックの本体は`Implementation`（能力を呼ぶ場合は`CallingImplementation`）として境界が定まっている。
- 宣言された読み取りだけが実装へ渡り、宣言外の呼出しは境界が拒否する。
- 任意の実装を`Runner`へ渡せるため、規則Pluginを差し込む口はある。

## 読み取れないこと

- **規則Pluginの発見方法**（どこから読み込むか、どう識別するか）。
- **宣言とPluginの版の束縛**（どの版の規則がこのフックを実行したかをどう記録するか）。
- **実Pluginの供給路**（`ReferenceRule`・`Guarded`以外の実装）。
- D2自体の採用（設計文書は「設計案」であり、採用済みではありません）。

## 位置付け

参照実装の確認記録であり、製品の受入や本番適格性を示しません。**境界は既にあり、契約の発見・束縛・版は未決定であること**を切り分けた記録です。設計判断を先取りしていません。

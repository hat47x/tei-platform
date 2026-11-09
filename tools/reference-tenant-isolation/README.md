# reference-tenant-isolation（#2020）

SaaS向けの提供形態で使う、テナント分離の基盤機構の最小Referenceです。ADR-0036の方針に従い、SaaS専用のアプリケーション実装を作らず、アプリケーションが選択して利用できるpluginとして提供します。

```text
go run ./tools/reference-tenant-isolation/cmd/tei-tenant-isolation check \
  -tenant tenant-a -root /tmp/tenant-a -request tenant-b
```

`check`は、現在のテナントと要求テナントが一致するかを確認し、一致しなければ`CROSS_TENANT_DENIED`をJSONで返して終了コード1になります。

## アプリケーションからの利用

```go
scope, err := tenantisolation.Open(tenantID, tenantRoot)
if err != nil {
    return err
}
defer scope.Close()

if err := scope.RequireTenant(requestTenant); err != nil {
    return err // *tenantisolation.Denial
}
content, err := scope.Read("orders/today.json")
```

- `Open`はテナントrootを`os.OpenRoot`で固定する。
- `RequireTenant`は、要求テナントが空または別テナントなら拒否する。
- `Read`・`Write`は、テナントroot内の相対pathだけを受け付ける。
- 絶対path、`..`、root外へのシンボリックリンクを拒否する。
- 越境の試行は`Denial`として記録し、`Scope.Denials()`で取得できる。
- 存在しないfileは`os.ErrNotExist`として返し、越境拒否と混同しない。

## plugin記述

`plugin.json`は、この機構が提供する能力と保証、提供しない保証を宣言します。

- 能力: `tenant-isolation`、`audit`
- 保証: テナントrootへの閉じ込め、越境拒否、シンボリックリンク拒否
- 保証しない: DBの行レベル認可、外部APIの分離、プロセス・ネットワークのサンドボックス

## 守っている境界

- テナントrootの外を読んだり書いたりしない。`os.Root`の相対操作を実際の境界とし、path検査は拒否理由を明示するために行う。
- 越境を「データなし」や空の成功へ変換しない。`CROSS_TENANT_DENIED`、`TENANT_REQUIRED`、`OUTSIDE_TENANT_ROOT`を分ける。
- テナントIDは英小文字で始まる識別子に限る。
- このReferenceは、アプリケーションの意味・権限・本番配備を代替しない。

## 未実装

- DBの行レベル認可、外部API・キャッシュ・検索indexのテナント分離。
- テナントごとの暗号鍵、データ所在、監査証拠の保存期間。
- Delivery Profileのplugin選択・検証との接続。
- 実SaaS環境、複数プロセス、分散配置での検証。

# 機能の業務結果と実行エラーの準拠確認

## 目的

この準拠確認では、意味上の機能を正常に実行して得られた業務結果と、実行自体を完了できなかった状態が混同されないことを確かめます。

対象はIssue #77とADR-0020で定めた、第1版の最小限の境界です。

## 正本

業務上の結末は、意味上の機能に定義した`output`または名前付きの`outcome`を正本とします。

```text
Capability
├─ output
└─ outcome <name> <payload-type>
```

処理を正常に完了した場合は、`BusinessResult`として業務結果を返します。一方、データベースへの接続失敗、対応付けの解決失敗、接続部の障害、契約にない結果は業務結果へ変換せず、実行エラーとしてGoの`error`で返します。

```text
Invoke Capability
       │
       ├─ 正常に実行できた
       │      ↓
       │   BusinessResult
       │      ├─ output
       │      └─ named outcome
       │
       └─ 実行を完了できなかった
              ↓
            error
```

## 認証の実参照例

参照用の作業領域では、`security.authenticate`を定義しています。

```tei
capability security.authenticate {
    input login_id std.identifier
    input credential security.credential
    outcome authenticated security.authenticated_subject
    outcome denied security.authentication_denial
}
```

この機能では、認証情報の不一致やアカウントのロックを`denied`として扱います。いずれも認証の判断を正常に完了した結果であり、実行エラーではありません。

一方、データベースや接続部の障害を`denied`に置き換えることはしません。

## 実行時の準拠確認

`internal/semantic/capabilityinvoke`のテストでは、参照用の作業領域から正本の機能定義を直接読み取り、次の動作を確かめます。

| 条件 | 期待結果 |
| --- | --- |
| 認証情報が正しい | `authenticated` Business Result |
| 認証情報が一致しない | `denied` Business Result |
| アカウントがロックされている | `denied` Business Result |
| データベースや接続部の障害 | 実行エラー |
| 定義されていない結果を返す | 実行エラー |

正常に判断した結果としての拒否をGoの`error`だけで表すことも、実行障害を`denied`として扱うこともしません。

## 既存の結果を返す機能との互換性

認証に固有の仕組みになっていないことを確かめるため、既存の`sales.order.submit`も同じ呼び出しの境界を通します。

この機能は、従来どおり一つの出力を持ちます。

```tei
capability sales.order.submit {
    input order sales.order
    output sales.order
}
```

同じ`Invoke`を通して、`output`で表した業務結果を受け取れることを確認します。

また、定義者が指定していない`submitted`などの結果を、暗黙に追加してはいけません。一つの出力だけを定義した機能が名前付きの結果を返した場合は、実行エラーとして扱います。

## データ本体の境界

第1版の呼び出しの境界では、受け渡すデータの具体的な業務オブジェクトの構造を定義しません。

正本の機能定義には結果の名称とデータ型の意味を保持します。一方、Goでの値の表現や既存システムの値・外部APIの応答からの変換は、接続部が担当します。

これは、ドライバーより先に汎用的な値のモデルを固定しないための意図的な制約です。型を持つデータの共通表現が複数の実際の参照例で必要になった場合に、別途設計します。

## 非目標

この準拠確認では、次の事項は扱いません。

- HTTPの状態コードやWeb画面への表示
- 汎用的な例外階層
- 汎用的な直和型
- 既存の認証用データ構造への接続
- 認可
- 秘密情報の解決
- 観測による外部状態の確認

既存の認証方式との接続は、この業務結果の境界を前提に、Issue #68で実証します。

関連: #68 / #69 / #77 / #124 / ADR-0020

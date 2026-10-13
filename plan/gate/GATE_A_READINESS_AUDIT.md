# Gate A 到達性監査

## 目的

この文書は、第1版の参照アプリケーション`order_entry`をアプリケーションドライバーへ引き渡す前に、判定段階Aの「アプリケーション上の意味をドライバーへ渡せる」という条件を横断的に確認した監査記録です。

判定段階Aの合否は、関連するIssueの完了件数では決めません。ドライバーがウェブ画面や専用のGoコードで不足する意味を補わずとも、必要な情報をリポジトリの参照事例と準拠確認から再現できることを基準とします。

## 評価区分

| 区分 | 意味 |
| --- | --- |
| **Ready** | Gate Aに必要な範囲をReference / Conformanceから再現できる |
| **Partial** | 重要な境界は成立しているが、Driverへ渡す情報が不足している |
| **Blocked** | Driverへ進む前に独立した実証が必要である |
| **Deferred** | 必要な作業だがGate A以後でよい |

## 結論

**判定段階AはReadyです。アプリケーションドライバー#69の主な実装を開始できる状態です。**

最後まで残っていたIssue #68「既存認証の互換性」について、PR #148・#149・#151で判定段階Aに必要な検証根拠が揃いました。

```text
Application Pattern / Datasource      Ready
Capability Binding                    Ready
Business Validation                   Ready
Business Result / Execution Error     Ready
Authentication                        Ready
Authorization                         Ready
Legacy authentication compatibility  Ready
                      ↓
                 Gate A Ready
                      ↓
                  Driver #69
```

この判定の根拠は、#68をクローズしたこと自体ではありません。認証と認可を含め、ドライバーに必要な意味をリポジトリから再現できると確認した結果です。

## Gate A項目別評価

| 項目 | 判定 | Evidence | Driverへ渡す境界 |
| --- | --- | --- | --- |
| browse対象 | Ready | `order_entry.recent_orders` | Patternを正本として利用する |
| record対象 | Ready | `order_entry.order_record` | Patternを正本として利用する |
| reference対象 | Ready | `sales.order.customer` + datasource `master` | Relationとlogical datasourceを利用する |
| header-detail関係 | Ready | `sales.order.lines -> sales.order_line many` | Semantic Relationを利用する |
| command対象 | Ready | `order_entry.submit_order -> sales.order.submit` | Command PatternからCapabilityを解決する |
| Pattern identity / Source | Ready | Source Index / rename invariance / Golden | file pathをidentityにしない |
| Datasource read / write | Ready | `sales` read-write / `master` read | `DatabaseRequirement`を正本として利用する |
| Business Resultの単一output | Ready | `sales.order.submit -> sales.order` | 既存output契約を維持する |
| named Outcome / Execution Error | Ready | #77 / PR #132 / #141 | `capabilityinvoke`境界を利用する |
| Authentication | Ready | #68 / PR #148 / #151 | `security.authenticate`と明示Bindingを利用する |
| Authorization | Ready | #68 / PR #149 | `security.authorize`とApplication Pattern identityを利用する |
| legacy認証Compatibility | Ready | PR #148 / #149 / #151 | legacy schema / functionをadapterの内側へ保つ |
| 業務Validation | Ready | #125 / PR #136 / #139 / #140 | Canonical ruleとEvaluatorを利用する |
| Capability Binding | Ready | #126 / PR #135、認証はPR #148 / #149 | Application + Capabilityから明示的に解決する |
| Command / Observation分離 | Ready | 既存Runtime / Deployment Evidence | command応答と外部状態を別の問いとして扱う |
| Target固有Presentation | Deferred | Gate Aでは不要 | 第一Web Target #70で扱う |

## Application Pattern — Ready

Issue #67では、`order_entry`の次の5種類のアプリケーションパターンが、実際の参照事例として成立しています。

- `browse`
- `record`
- `reference`
- `command`
- `header-detail`

パターンは既存資産の意味を書き換える場所ではありません。アプリケーションが正本上のどの識別情報を、どの目的で使うかを保持する層です。

パターンの識別情報はファイルのパスに依存せず、ソース索引から記述元のソースへたどれます。この5種類を恒久的な標準カタログとする判断は、ドライバーと複数の実行先で検証根拠が得られるまで保留します。

## Datasource境界 — Ready

`order_entry`は、次の二つの論理的なデータ接続先を利用します。

```text
sales  -> sales_core  -> read-write
master -> master_data -> read
```

データ接続先へのアクセス条件はパターンへ複製せず、アプリケーションの定義を正本とします。ドライバーも同じ`DatabaseRequirement`を使い、独自の設定には書き写しません。

認証情報の保存先を、この二つのデータ接続先に無理に組み込むことはしません。既存の認証機構がデータベース・LDAP・IdP・既存関数・外部APIなどのどこにあっても、ドライバーは認証・認可の能力を通して利用します。

## Capability Binding — Ready

Issue #126・PR #135により、アプリケーションと能力の組合せから`sales.order.submit`を明示的に解決できる参照事例が成立しました。

```text
order_entry / sales.order.submit
```

PR #148と#149では、同じ原則を認証・認可にも適用しています。

```text
order_entry / security.authenticate
order_entry / security.authorize
```

ドライバーは、ファイルパス・Goのフック名・列挙順から実装を推測しません。候補が0件または複数件で一意に特定できなければ、業務上の結果ではなく、実行準備が完了しないエラーとして扱います。

## 業務Validation — Ready

Issue #125・PR #136・#139・#140では、業務上の検証規則の正本、記述元のソース、最小限の評価器が成立しています。

```tei
validation sales.order_line.quantity_positive positive {
    attribute sales.order_line.quantity
}

validation sales.order.lines.line_number_unique relation-unique {
    relation sales.order.lines
    attribute sales.order_line.line_number
}
```

評価結果には`Satisfied / NotSatisfied`を用います。必要な入力がないなど、評価そのものを完了できない場合は、別のエラーとして区別します。

ドライバーは`quantity > 0`などの規則の本文を専用のGoコードに複製せず、同じ正本上の検証規則と評価器の境界を利用します。

## Capability結果境界 — Ready

ADR-0020・Issue #77に従い、能力を正常に呼び出せた場合の業務上の結果と、呼出しそのものを完了できなかった状態を区別します。

```text
Business Result
  ├─ existing output
  └─ named Outcome

Execution Error
  └─ Invocation自体を完了できない状態
```

PR #141の参照事例では、少なくとも次の結果を区別できます。

- 正しい認証情報 → `authenticated`
- 認証情報の不一致 → `denied`
- アカウントのロック → `denied`
- データベースや変換器の障害 → 実行時エラー
- 定義されていない結果 → 実行時エラー

既存の`sales.order.submit`は、従来の`output sales.order`を維持し、同じ呼出しの入口から利用できます。認証のために、既存の能力へ暗黙の結果を追加したわけではありません。

## Authentication — Ready

PR #148では、`security.authenticate`を第1版の参照アプリケーション`order_entry`から既存の認証機構へ明示的に接続しました。

参照事例では、既存の利用者情報に含まれる次の意味を保持しています。

- user ID
- authenticated subject identity
- credential reference
- disabled
- validity period
- login failure count
- lock state
- organization
- role

これらをTEI標準の利用者スキーマに作り直さなければ利用できない構造にはしていません。

`order_entry/security.authenticate`を一意に解決でき、ソース索引から対応付けの記述元へたどれます。実行時の認証情報は検証の境界にだけ渡し、正本のモデル・ソース索引・業務上の結果・通常の検証根拠へコピーしません。

## Authorization — Ready

PR #149では、`security.authorize`について、次の最小限の契約を実証しました。

```text
subject + action + resource
          ↓
 security.authorize
     ├─ allowed
     └─ denied
```

認可の対象となる資源には、ウェブ画面名やURLではなく、既存のアプリケーションパターンの識別情報を利用します。

```text
browse  / order_entry.recent_orders
edit    / order_entry.order_record
execute / order_entry.submit_order
```

既存システムに固有の組織・役割・規則は変換器の側で利用します。アプリケーションやドライバーに公開して同じ認可条件を再実装させることはしません。

`allowed / denied`は、認可の判断を正常に完了した際の業務上の結果です。規則の保存先などに障害があった場合は、実行時エラーとして区別します。

## 既存認証の実現方式交換 — Ready

PR #151では、`order_entry/security.authenticate`というアプリケーション側の対応付けを維持したまま、背後の実現方式だけを交換できることを確認しました。

比較したのは次の二つの経路です。

1. 既存の利用者情報を読み、認証情報の照合と既存の規則を組み合わせる経路
2. 既存の関数・ストアドプロシージャ・外部APIなどに相当する認証関数を呼び出す経路

後者では、既存システム固有の戻り値を変換器で`authenticated / denied`へ対応付けます。対応していない戻り値や既存関数の実行障害は、実行時エラーとして扱います。

どちらの経路も、正本上の同じ`security.authenticate`契約・同じ対応付けの識別情報・同じフックを保ち、同じ認証済み主体を返します。

PR #151では、`reference/workspace`と`reference/world.json`も変更していません。CI #374の参照用モデルの再現性検査も、期待出力を更新せずに成功しました。実現方式の違いを正本のモデルやアプリケーション上の意味へ持ち込まずに交換できたことを示しています。

## Secret境界 — Ready

参照事例の認証用試験データや関数との対応付けには、パスワードの平文・パスワードのハッシュ・認証情報のハッシュ・トークン・秘密情報を保存しません。

正本上の`security.credential`は入力の意味を表す型であり、実際の認証情報の値を保存する場所ではありません。

判定段階Aでは、次の境界を確認しました。

- Gitで管理する参照事例に実際の秘密情報を置かない
- 正本のモデルに実際の秘密情報を置かない
- ソース索引に実際の秘密情報を置かない
- 業務上の結果へ実行時の認証情報をコピーしない
- ドライバーのシナリオの再現性を理由に、秘密情報を通常の検証根拠へ記録しない

秘密情報を実際に取得・解決する方式は、本番用IdPなどの具体化と合わせて、判定段階Aの後で扱うことができます。

## Driverがlegacy schemaへ直接依存しないための条件 — Ready

判定段階Aで必要なのは、ドライバーの実装そのものではありません。ドライバーが旧システムのスキーマを知らなくても認証と認可を実行できる入口が揃っていることです。

この監査時点では、次の構造をリポジトリから解決できます。

```text
security.authenticate
        ↓
Application-scoped Binding
        ↓
legacy adapter / function adapter
        ↓
authenticated subject
        ↓
security.authorize
        ↓
Application Pattern identityを使ったaction / resource
```

組織・役割・旧システム固有の戻り値・物理スキーマなどを、ドライバーの正本上の入力へ追加する必要はありません。

ドライバー#69ではこの境界を実際に利用し、旧システムのスキーマへ直接依存していないことを、判定段階Bの準拠確認として改めて検証します。

## Driver開始判定

**ドライバー#69の実装を開始できる状態です。**

判定段階Aの必須項目には、独立した阻害条件は残っていません。

ただし、判定段階Aを通過したことは、ドライバーの完成を意味しません。次は、同じアプリケーション上の意味を画面なしで一連の処理として実行し、判定段階Bへ進めるかを確認します。

最小限のシナリオは次のとおりです。

1. 認証を実行して主体の識別情報を得る
2. `browse`で受注の一覧を扱う
3. `record`で一件の受注を扱う
4. `reference`で`master_data`から顧客を解決する
5. `header-detail`で受注明細を扱う
6. 正本上の業務検証規則を実行する
7. 認可を評価する
8. `command`で`sales.order.submit`を呼び出す
9. 業務上の結果と実行時エラーを区別する
10. 必要に応じて観測を通じて外部の状態を確認する

## Gate A以後へDeferredするもの

次の事項は重要ですが、判定段階Aの通過を妨げる条件にはしません。

- 名前付きの結果を最初のウェブ実行先で表現すること
- HTTPの状態コードや画面のメッセージへの派生表示
- 本番運用に耐えられるIdP
- 本番環境の高可用性
- 実行先に依存しない表示の最終的な抽象化
- 五つのパターンの共有カタログ化
- `OrderLine`の物理的な永続化と外部キーの一般化
- 本番環境のトランザクション調整機構
- 汎用の検証式言語と規則の実行機構
- データベースの制約やウェブフォームの検証器への検証規則の変換
- 能力の入出力を表す汎用オブジェクトモデル
- 実行時に複数の実装を動的に選ぶ汎用機構
- `std.auth.*`の標準化

## 関連

- `plan/V1_REFERENCE_APPLICATION_PLAN.md`
- `plan/design/LEGACY_AUTH_V1_DESIGN.md`
- `plan/design/CAPABILITY_OUTCOME_V1_DESIGN.md`
- ISSUE-000003-legacy-authentication-compatibility-reference.md
- ISSUE-000006-application-driver-vertical-slice.md
- ISSUE-000009-capability-outcome-and-execution-error.md
- ISSUE-000013-gate-a-readiness-audit.md
- ISSUE-000014-business-validation-reference.md
- ISSUE-000015-submit-capability-binding-reference.md
- GitHub Issue #67 / #68 / #69 / #77 / #124 / #125 / #126
- GitHub PR #132 / #135 / #136 / #139 / #140 / #141 / #143 / #148 / #149 / #151
- ADR-0019 / ADR-0020

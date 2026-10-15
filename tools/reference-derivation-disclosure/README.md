# Data Derivation D5：導出結果の開示境界 Reference（#1693）

本ディレクトリは **既存 `security.authorize` へ判断を委譲する読取り専用の合成Reference** であり、TEIの正式な認可実装、Query実行器、SQL書換え、Privacy Engine、Business Factの保存機構ではありません。実装主体は `guard.go`、否定Fixtureは `guard_test.go` です。

## 既存責務との接続

- **D1 #1637**：`internal/semantic/derivationref` の純粋規則・結果状態を流用し、式計算やRule正本を作りません。
- **D2 #1640**：`QueryPlan.Surfaces` と `Rows` は、既存Query/Hostが**必要な操作と対象行を完全に決めた後**に注入するものです。このReferenceは SQL のFilter→Sort→Count→Limit、JOIN多重度や物理的行絞り込みを実装しません。Query解析不完全時にこのGateを使って「安全」を主張できません。
- **D4 #1692**：対象・規則の受理と型付きBinding解決はD4に属します。D5は、これらを受け取ったHostから渡された規則と結果対象に対する開示のみ検査します。未統合PR #1851の所有コードへ書き込みません。
- **#1118/#1043/#1656**：Host認証済みのSubject、Tenant、Purpose、Provider identity、評価文脈を要求し、`internal/security/legacyauthorization.Adapter` の `security.authorize` に引き渡します。Client/Pluginから受けた `TrustedContext` を信頼する機能は**ありません**。正規のHost identity/linkと現在のAuthority/Policy Epochの照合が前段に必要です。
- **#951**：内部集約Providerの実Artifact、Selection、Capability、Plugin Lock、実行許可は既存Resolver/Hostの責務です。`ProviderSubjectID` はHost固定値という前提であり、それを選択・検証するResolverを新設しません。

## 認可操作の判定マトリクス

| 操作 | 権限の受益主体 | `security.authorize`へ渡すaction（例） | 条件 |
| --- | --- | --- | --- |
| 行の使用 | caller | `derive.row.include` | 選択された行すべて、計算・件数・集約前 |
| 規則の呼出 | caller | `derive.rule.execute` | 規則ごと |
| 明細入力の読取 | caller | `derive.input.read` | 入力属性ごと |
| 非公開集約入力 | 選択されたProvider自身 | `derive.input.internal` | callerへraw入力を渡さない |
| 明細/集約結果の公開 | caller | `derive.detail.disclose` / `derive.aggregate.disclose` | 結果Targetごと |
| 集約実行 | caller | `derive.aggregate.run` | Providerの内部権限とは別 |
| 任意の開示利用 | caller | `derive.surface.{select/filter/sort/count/group/page/metadata/trace}` | Queryが使う面ごとに個別判定 |

これらのaction文字列は合成Referenceで既存Authorityへ送る*テスト用の具体的な入力*です。既存認可ルールを上書きせず、製品全体の権限語彙として無条件採用しません。ResourceはHost指定のTenant/Purposeと論理Rule/Target/Input/Row IDから構築し、物理列名やSymbolで結び付けません。Auth `allowed` は **Subject/Action/Resourceが完全一致するGrant** のみ受け入れ、拒否・Backend故障・不整合をすべてfail-closedにします。

## 合成Fixtureと負例

1. 管理者：数量2×非公開原価300→金額600、全段階を個別に許可した場合のみ公開。
2. 一般利用者：結果`amount`の表示権限があるだけでも、`cost`の読取権限がなければ明細600を非開示。単なる表示列除外では読取境界を迂回しません。
3. 一般利用者の部門集約：一般利用者には原価読取を与えず、Hostに選択された内部Provider自身だけが `derive.input.internal` を持ち、行・規則実行・集約結果・`group`の公開権限が揃ったときだけ集約900を返す。集約許可は明細600の公開許可へ転用できません。
4. `select/filter/sort/count/group/page/metadata/trace` のいずれの権限も欠ければEvaluatorを呼ばない。
5. Tenant T1のQueryにT2行が含まれた場合、権限テーブルを見る前に不完全なPlanとして拒否。さらに選択行単位の許可も別に必要。
6. 実行後のGrant失効、保存済み結果の再読取時の失効、認可Adapterエラー、Evaluatorエラー、別主体の偽装Grantを拒否し、secretをError文字列に載せない。

## 境界と未証明事項

- D5の`Evaluate`は **実行の前後**にAuthorityを再照会しますが、二回目の判定と公開の間は原子的ではありません。全体の現在権限、真正なHost ContextやProvider selection、テナントと行の完全な取得、実DB/Queryへの接続、キャッシュの保存・失効（#1641）、trace/metadataの安全な投影（#1696/#1178）は未成立です。
- 特に公開を許した小さな集約集合からの再識別、複数リクエスト間の差分攻撃、タイミング漏洩の防止は証明していません。Host/D2が対象行の集合を正しく限定しなければ本Gate単独では防げません。
- テストでは非公開入力をProviderの閉包内で算出する合成例を使用します。物理DB列読込・`Wasm`・`Go`・実HTTPの同等性、Production Authorityや実利用者の権限を証明したものではありません。
- `context.Context`のcancel、Provider側ログ・panicなどの実装固有秘匿保証は一般に証明しません。

## 検証

```bash
# リポジトリルートにて、go.mod指定のGoで実行
gofmt -l tools/reference-derivation-disclosure/*.go
go test -count=1 ./tools/reference-derivation-disclosure
go vet ./tools/reference-derivation-disclosure
```

成功・失敗の確定記録は対象PRの**exact head**とIssue #1693へ別途残します。テストコードを追加しただけではPASSとみなしません。

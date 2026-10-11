# Business-system design Reference

このdirectoryは、`reference/workspace`のCanonical meaningを変更せずに、Data / Function-Interaction / Businessのcross-view設計境界を再現するためのReference fixtureを保持します。

現時点ではBusiness ProcedureをCanonical Workspaceのsource rootへ追加しません。

## `order-entry-business-procedure.json`

`order_entry`の既存Application / Semantic identityを参照しつつ、Business側でしか持てないpurpose / trigger / actor候補 / manual actionを保持するReference authoring fixtureです。

このfixture自身はCanonical World authorityではありません。一方、Business Procedure側の固有意味まで「derived」とみなすこともしません。

```json
{
  "artifactRole": "reference-authoring-fixture",
  "canonical": false,
  "procedure": {
    "identityStatus": "reference-local-stable"
  }
}
```

`procedure.id`はReference corpus内でreview target、handoff endpoint、cross-referenceを安定して指すためのauthoring keyです。top-level Canonical `Procedure`を採用したことを意味しません。

主な圧力は次です。

- Business ActionとApplication Functionを同一視しない
- manual / external / unresolved stepにfake Capabilityを要求しない
- Application Patternをexact identityで参照する
- Patternがすでに所有するData / Relation / Query / Capability / DatasourceをBusiness Procedureへ重複記述しない
- Pattern-owned meaningはnormalized WorldからBusiness Action × Pattern meaning viewへderivedする
- Procedure側で独自に必要なValidation等のexplicit referenceだけを別に保持する
- 名称類似からmissing relationやCapabilityを推測しない

### Authoring inputとderived Projectionを分ける

Business Procedure fixtureには、Application / Semantic Worldからは導出できない意味があります。

```text
purpose
trigger / event candidate
actor / job context candidate
manual / external / unresolved Business Action
judgment / continuation
```

これらはReference実験のauthoring inputです。

一方、Business ProcedureからApplication Patternを参照した場合、次の意味はPattern側を正とします。

```text
Application Pattern
  ├─ Data
  ├─ Relation
  ├─ Query
  ├─ Capability
  └─ Datasource
```

したがって、たとえばBP-02は次だけをauthoring inputとして保持します。

```text
Business Action:
  confirm the customer associated with the order

Application Pattern:
  order_entry.customer_reference
```

`customer_reference`が`relation=sales.order.customer`、`datasource=master`であることはfixtureへ再記述せず、`workspace.Load()`後のnormalized Application Patternから導出します。

同様に、BP-07が`order_entry.submit_order`を参照するとき、`capability=sales.order.submit`はPatternから導出します。

三視点を分離することと、同じ意味を三箇所へコピーすることを混同しません。

### direct semantic reference

Patternから導けない意味をBusiness Procedure側が明示的に要求する場合だけ、Data / Relation / Validation / Capabilityのdirect referenceを持てます。

現在のfixtureではBP-05のValidationがこれに該当します。

checkerはdirect referenceもexact identityで解決し、似た名称から補完しません。

baseline確認:

```bash
go run ./tools/reference-business-procedure-check
```

成功時はJSON reportを出力し、blocking diagnosticなしで終了します。

出力には少なくとも次が含まれます。

- `businessActionFunctionRows`
  - 各Business ActionのApplication Pattern参照
  - Patternを持たない場合は`noApplicationFunction: true`
- `businessActionPatternMeaningRows`
  - PatternからderivedしたData / Relation / Query / Capability / Datasource
- `explicitBusinessMeaningRows`
  - Business Procedure fixture自身が明示したdirect semantic reference

これらのMatrix / diagnostic出力はderived Projectionです。Reference authoring fixtureへ逆書きしません。


## Applicationと業務手順のValidation参照を比較する（評価中）

Application全体のValidation宣言と、Business Procedureの各validation-supportステップが明示した参照は、目的も適用範囲も異なります。集合差は**確認すべき候補**であり、業務仕様の欠落や不整合を自動的に意味するものではありません。

既存checkerに変更を加えず、次のフラグを指定したときだけ読み取り専用の比較結果をJSONへ追加します。

    go run ./tools/reference-business-procedure-check -review-validation-scope

### 結果の読み方

| 項目 | 意味 |
| --- | --- |
| applicationContext / step | 明示された比較対象ApplicationとBusiness ProcedureのStep ID |
| applicationOnly / procedureOnly | 片側にだけ明示されたValidation ID |
| sharedValidations | 両側に明示されたValidation ID。一致しても業務上の同値性は保証しない |
| review-needed | 参照集合に差がある。適用範囲を人が確認する |
| no-set-difference | 集合差はない。業務上の適用範囲や実行結果まで等しいとは限らない |
| unresolved-context | Step IDが空・重複する、またはApplicationを一意に指定・解決できない |
| unresolved-reference | Validation定義の欠落・曖昧性・宣言の重複がある。集合差を確定しない |
| referenceIssues | 未解決参照のside（application / procedure）、id、reason（missing-definition / ambiguous-definition / duplicate-declaration） |
| procedureSource | 業務手順の入力fixtureのパス。Canonicalな意味IDではない |
| sourceEvidence / unresolvedSource | 出典が特定できた定義の相対パス・行・指紋／特定できなかった意味ID |

Step IDは種類を問わず同一手順内で一意でなければなりません。比較するApplicationは、fixtureの指定を使い、名称の類似から所属を推測しません。Validation参照が解決できない行は、集合差を返す代わりに原因を示します。

比較できる行がある場合に限り、既存Source Indexを検証します。Application定義と共通・片側固有のValidation定義を追跡し、定義やファイルの重複、出典の欠落を推測で補いません。不正な出典パスや指紋は根拠に採用せず、再解析でも古い根拠を残しません。Source Indexの構築そのものが失敗した場合は、成功したものとして表示せずエラーで停止します。

現在の受注Referenceでは、BP-05の二つのValidationが共通し、Application側にsales.order_line.line_number_positiveが一つ残る想定です。これは**人による適用範囲の確認を促す結果**であり、Business Procedureへ規則を書き戻す指示ではありません。Workspace、Acceptance、Authorization、既存のblocking diagnosticは変更しません。

### 手動検証（CI停止中も同じ条件）

信頼できる実Repositoryのクリーンなチェックアウトで、対象commitと比較先mainを固定し、以下を実行します。

    git rev-parse HEAD
    git status --porcelain --untracked-files=all
    go version
    go test ./tools/reference-business-procedure-check -count=1
    go vet ./tools/reference-business-procedure-check
    go run ./tools/reference-business-procedure-check
    go run ./tools/reference-business-procedure-check -review-validation-scope
    go run ./tools/reference-business-procedure-check -review-validation-scope -probe reference/business-system-design/order-entry-customer-edit-conflict.json -expect-diagnostic BUSINESS_FUNCTION_DATASOURCE_ACCESS_CONFLICT
    bash tools/repository-health-check.sh
    git status --porcelain --untracked-files=all
    git rev-parse HEAD

上記に加え、ignoredファイルとindexの状態も前後で確認します。記録対象は、commit SHA、main SHA、Go環境、コマンドごとの終了コード、既存診断とレビュー表示の比較、BP-05の共通・差分IDと正本出典、前後のclean状態です。**この記述は手順であり、コマンドを実行した記録ではありません。** 過去の隔離テストを最新headの資格に転用せず、full Repository healthが未実施・不合格ならmainへ統合しません。Draft PRは検証に合格しても自動的にマージしません。

### 人のレビュー負担を測る

[技術リサーチ統合方針](../../plan/TECH_RESEARCH_INTEGRATION_2026-10.md)のREF-SPEC-01に従い、通常のSource探索（A）と比較JSONを併用する方法（B）を、同じ業務上の確認課題で比べます。所要時間・開いたファイル数・再検索回数に加え、参照IDの取り違え、未解決事項の記載漏れ、根拠のない断定を記録します。Bが速くても誤判断が増えれば成功としません。評価順序や対象課題を入れ替え、同じ人の学習効果を独立な試行として数えないようにします。

**実験は計画段階で、所要時間や効果の実測値はありません。** 出典を限定して確認した事実、過去の隔離テスト結果、残る検証条件は[REF-SPEC-01監査記録](../../docs/audits/ref-spec-01-2026-10-09-source-contrast.md)を参照してください。

## `order-entry-follow-up-procedure.json`

同じ`order_entry.order_record`を別のBusiness Procedure文脈から利用するReference authoring fixtureです。

primary Procedureと同じPatternを再利用してもBusiness Actionを一つへ畳まず、multiple Procedure -> same Patternのtraceabilityを圧力試験します。

このfixtureも`identityStatus=reference-local-stable`であり、Reference内のlogical authoring keyを持ちます。

## `order-entry-procedure-reference-set.json`

二つのProcedure authoring fixtureとProcedure間handoffを同じReference setとして束ねます。

Procedure keyとartifact locatorを分離します。

```text
order-entry.order-handling
  -> order-entry-business-procedure.json

order-entry.follow-up
  -> order-entry-follow-up-procedure.json
```

handoffはfile pathではなくProcedure key + Step keyを参照します。

```text
order-entry.order-handling / BP-09
  -> order-entry.follow-up / FU-00
```

そのため、pure file moveではlocator mappingだけを更新し、handoff semantic endpointは変えません。

一方、Procedure key自体をlogical renameしたのにhandoff linkを更新しなければfail closedにします。類似名称から接続し直しません。

このReferenceはhandoffをruntime Workflowへ変換しません。retry / timer / queue / scheduler等を含意しません。

## `second-corpus/`

`order_entry`とは独立した、二つ目のBusiness Procedure Referenceです（受注生産の製造と、完成品の入庫）。Applicationを参照しない二つのProcedureと、その間の責任の受け渡し（Handoff）一件、出典・fact・設計判断・未解決事項を分けて記録するseed ledgerを持ちます。

同じloader、正規化Business model、D4導出、standalone Handoff source形式、baseline checkerへそのまま通ります。詳細、review状態、主張しないことは[`second-corpus/README.md`](second-corpus/README.md)を参照します。

## Review / approval continuity

Reference-local Procedure keyはreview historyのtargetを束ねるためにも使います。

ただしapproval Evidenceは次のsource snapshotへ束縛します。

```text
Procedure key
Repository revision
Artifact fingerprint
Source locator
Decision
```

同じProcedure keyを保ってBusiness Action文言を変更した場合、review targetの連続性は保ちますがfingerprint / revisionが変わるため旧approvalを自動carry-forwardしません。

pure file moveでbytesが同じ場合も、Procedure key / fingerprintは保てますがsource-dependent Evidenceを別revisionへ黙って継承しません。

Procedure key自体のrenameはlogical identity changeとして扱います。

詳細は`plan/BUSINESS_PROCEDURE_REVIEW_HANDOFF_PRESSURE_2026-09.md`を参照します。

## `order-entry-customer-edit-conflict.json`

最初の意図的cross-view defectです。

`order_entry.customer_reference`は`master` datasourceを使いますが、現行Application contractは`master`を`read`として要求しています。

probeはBP-02へexperiment-onlyの`read-write`要求を重ねます。

```text
Business / Function candidate
  edit customer through order_entry.customer_reference
        ↓
Pattern-derived datasource
  master
        ↓
Application contract
  datasource master = read
        ↓
BUSINESS_FUNCTION_DATASOURCE_ACCESS_CONFLICT
```

再現は次のとおりです。

```bash
go run ./tools/reference-business-procedure-check \
  -probe reference/business-system-design/order-entry-customer-edit-conflict.json \
  -expect-diagnostic BUSINESS_FUNCTION_DATASOURCE_ACCESS_CONFLICT
```

`-expect-diagnostic`を指定した場合、期待したblocking diagnosticだけが出たときに成功します。期待外のsemantic / architecture-policy diagnosticが同時に出た場合は失敗します。

このprobeは「Customer編集を永久に禁止する」という要求ではありません。

現在の`order_entry` contractではそのinteractionを実現できないことを示します。実際にCustomer更新が必要なら、Projectionだけでwriteを追加せず、別Application / accepted Capabilityへ接続するか、authoritative Application / Database Contractへ戻る必要があります。

## Unit test

```bash
go test ./tools/reference-business-procedure-check
```

少なくとも次を確認します。

1. baseline fixtureのexplicit referenceがnormalized Worldへresolveする
2. BP-04 / BP-09がApplication Functionなしで正常に残る
3. Pattern-owned Data / Relation / Capabilityをfixtureが二重管理せず、normalized Patternから導出する
4. baselineの`master=read`はconflictにならない
5. edit probeの`read-write`要求はaccess conflictになる
6. Procedure keyだけの変更ではPattern-derived cross-view meaning自体は変わらない
7. multiple Procedure -> same Patternを区別してtraceできる
8. 類似名称からRelationを推測しない
9. Source Index Artifact fingerprintはpure file moveでは保たれるがsemantic identityとしては使わない
10. manual / external Business ActionはPattern-derived Projectionから再生成できない
11. Business-only wording changeをApplication semantic changeと混同しない
12. Reference-local stable keyでProcedure間handoffを解決できる
13. artifact moveでhandoff semantic endpointを変えない
14. logical Procedure rename後のstale handoffをfail closedにする
15. review target continuityとapproval snapshot validityを分離する
16. fixtureの未知field、未対応schemaVersion、別artifact roleを黙って受理しない

## Authority boundary

このReferenceで証明するのはBusiness Procedure viewpoint、authoring residual、cross-view diagnostic、Reference-local identity pressureです。

このartifactの存在だけを根拠に、次を採用しません。

- Canonical `Procedure`
- Canonical `Step`
- Workflow Engine
- retry / timer / queue / scheduler
- Business actorからAuthorization Roleへの自動mapping

review continuityとhandoffを加えた結果、安定した**authoring-level Procedure key**には具体的価値があることが確認できました。

ただし現時点で必要なのはReference-local stable keyであり、Canonical World上のProcedure identityではありません。

次に残る圧力は、Business Procedureをfile-native accepted authoring sourceへ進めた場合のreload / Source Index / Shared Acceptance、複数Subject Area / Subsystemを跨ぐhandoff、外部contractや長期reuseでCanonical Procedure subjectが必要になる具体例です。

# TEIの業務システム設計に関する要求の解釈

状態：現行の要求解釈

## 1. この文書の位置づけ

この文書は、TEIの業務システム設計に関する製品要求と設計文書の対応を示します。

正式な製品要求の正本は[`TEI_PRODUCT_REQUIREMENTS.md`](TEI_PRODUCT_REQUIREMENTS.md)です。この文書を第二の正本にはしません。

現在、業務システム設計について特に直接的な要求として次が採用済みです。

- `RQ-025` — 業務上の行為・手順をアプリケーションの機能とは独立して設計できること
- `RQ-026` — データ、機能・操作、業務手順の異なる視点から相互検証できること


## 2. 設計原点へ立ち返る順序

判断に迷う場合は、次の順で設計原点へ立ち返ります。

1. 三要素分析法など、業務システム設計について出典を特定できる一次資料
2. X-TEA / XEAD Modeler / Driverの公開マニュアル・モデル構造
3. XeadModeler / XeadDriver公開ソース
4. 参照モデルでの実際の使われ方
5. 現行TEIの製品要求、構成設計、参照事例、検証根拠
6. 小規模なTEIの参照実験


この確認順序は、過去のモデル構造をそのままTEIへ移植するためのものではありません。過去の概念がどのような設計上の失敗を防いでいたかを確認し、TEIの意味、権限、提供実装、実行先の境界に照らして考えます。


## 3. RQ-025の解釈

RQ-025では少なくとも次を分離します。

```text
Business Action
    != Application Function / Application Pattern
    != Capability
    != Realization
```

業務上の行為は、アプリケーションの機能を伴わなくても成立します。人間、AI、外部組織、SaaS、現物を扱う作業、既存システムなどによる仕事を、実行可能に見せるためだけに架空の機能へ置き換えてはいけません。

業務手順では、必要に応じて目的、開始条件や出来事、行為主体、業務上の行為、人や外部による作業、アプリケーションの機能の利用、引継ぎ、完了や継続について説明します。

ただし、設計の品質を「すべての項目が埋まっていること」とは定義しません。情報源から確定できない行為主体、開始条件、順序、実現方法、現実の状態、責任の境界などは推測で埋めず、未解決のまま保持できる必要があります。

```text
explicit unresolved
  = knowledge boundaryがreview可能
  != design failure

unsupported completion
  = false completeness
  = design quality defect
```

業務手順を一つのアプリケーションの内部に固定することも要求しません。関係するアプリケーションは、明示された構成パターンへの参照などを、受理・正規化した情報に基づいて正確に解決し、確認します。参照用の試験データで中心となるアプリケーションや、報告上の文脈を、業務手順の意味を管理する主体や権限として暗黙に扱いません。

また、参照・報告のためにアプリケーション名を記録していても、それは業務上の意味が成立するための必須条件ではありません。明示された参照を解決できる限り、文脈の欠落や未知の名称だけを理由に業務手順を無効とはしません。

ただしRQ-025は、`Procedure`を必ず正本の最上位の対象とすることも、業務手順の実行エンジン、分岐、タイマー、再試行、スケジューラ、処理の統括機構を中核に追加することも要求しません。

また、

```text
Business Actor / Job
    != Authentication Principal
    != Authorization Role / Policy
```

を維持します。

責任の引継ぎについても、行為主体の変更や手順の前後関係だけを根拠にはしません。行為主体の違いは判断の補助になりますが、業務上の責任が継続するかどうかは、明示して受理した業務の意味から判断します。

## 4. RQ-026の解釈

RQ-026では、データ、機能・操作、業務手順を、同じ業務システムを確認するための異なる視点として扱います。

```text
Data / Relation / Validation
          ↕
View / Pattern / Query / Action / Capability
          ↕
Business Procedure / actor / manual / trigger
```

三つの視点に対応する同型の正本を作ることが目的ではありません。

一つの視点で見つかった矛盾を、他の視点と正本となる情報源に戻って修正できることが製品の価値です。

視点をまたぐ検証では、結果を単なる合格・不合格にまとめず、少なくとも次を区別します。

```text
supported
  exact accepted authoritiesがcorroborateする

contradiction
  accepted authorities間に具体的な不整合がある

unresolved
  exact identity / authority / actuality等が不足し、肯定も否定もできない
```

これは新しい永続的な列挙値や診断の重大度を要求するものではありません。特に、一つの視点で追跡情報が見つからなくても、他の視点に対象が存在しないと決めつけてはいけません。

視点をまたぐ検証で見つけるべき問題には、次のようなものがあります。

- business identity / candidate key / functional dependencyがPersistence上の一律なsurrogate IDへ隠れ、異なるRelation semanticsを区別できない
- Data上のuniqueness / determinant / cardinalityと、FunctionやBusiness Procedureが前提にする識別・選択単位が矛盾する
- Business Procedureが必要とする情報がDataにない、または必要性自体がまだunresolved
- UI / Functionがread-only datasourceへwriteを要求する
- Business ActionがApplication Functionへ過剰に還元される
- actor labelがAuthorization Roleへ暗黙変換される
- Data / Relation変更がProcedure / Viewへ与える影響を追えない
- Application Business Resultを外部世界の業務完了と混同する
- physical factを観測しただけでresponsibility Handoffを成立させる
- Handoff conditionと、そのcondition成立を支えるEvidence / Observationを同一視する

対応表、CRUD表、流れ図、関係図などは、できる限り明示された参照から導出し、第二の正本として二重管理しません。

また、業務上の行為、アプリケーションの処理、外部で起きた事実を次のように同一視しません。

```text
Business instruction
  != system result
  != physical / external fact
  != Business acceptance / continuation
```

分野固有の契約によって一致が証明された範囲に限り、同じ意味として扱います。

## 5. 関連する既存Requirement

RQ-025とRQ-026は独立して追加されましたが、次の既存要求と組み合わせて成立します。

- RQ-001 — 業務意味と実装技術の分離
- RQ-003 — 暗黙の複数正本を作らない
- RQ-006 — Desired / Actual / Diff / Unresolved
- RQ-010 — 高密度Editor
- RQ-011 — 複数Projection
- RQ-012 — Driverによる設計検証
- RQ-014 — Application Pattern再利用
- RQ-017 — 二重管理最小化
- RQ-018 — Command / Observation分離
- RQ-019 — authority / Resource境界
- RQ-021 — Reference / Conformance
- RQ-022 — 設計検証段階と完成製品の区別

RQ-006の`Unresolved`は主に外部状態の比較に関する要求です。ただし、「情報不足を成功や差分なしと扱わない」という考え方は、業務を複数の視点から確認する場合にも適用します。これだけを根拠に、RQ-006の永続的な表現を業務手順へ流用しません。

## 6. Evidence待ちとして残す論点

次の事項は製品要求として採用済みであっても、表現方式については参照事例の検証根拠に基づいて判断します。

1. Business Procedureにdurable Canonical identityが必要か
2. actor / event / handoff / decision / manual instructionの構造化範囲
3. DFD型Business Flowの最小表現
4. `Subsystem` / `SubjectArea`のdurable semantic boundary
5. Target Adapter前のTarget-neutral View / Interaction構造
6. realistic Reference Modelのscale / packaging / licensing、および実装・実行によって方法論 / platform claimを反証可能にするEvidence条件
7. temporal semanticsのdomain meaning境界
8. cross-view diagnosticのseverityをどこまでdeterministic Conformanceへ昇格するか
9. optionalなReference / reporting Application contextをfixtureへ残すか完全除去するか、そのcompatibility
10. source-backed fact / accepted Business meaning / derived / proposal / unresolvedのnatureを、どこまでdurable metadataとして保持する必要があるか
11. Handoff condition成立を支えるEvidence traceを、既存Provenance / Evidence mechanismだけで十分にreviewできるか
12. business identity / candidate key / functional dependency / domain constraintを、既存primitiveでlosslessに扱えるか。専用semantic contractが必要か
13. business occurrence identityとhistory / version identity、validity period、identifier reuse、current-state selectionを既存primitiveでlosslessに扱えるか。時間軸を必要としないDataへ履歴構造を一律に導入せずに済むか

これらを解決するために、外部の資料に同名の概念があるという理由だけで、中核の概念種別やスキーマ項目を増やしません。

## 7. Acceptance Trace

RQ-025とRQ-026の主要な製品受入シナリオであるPAS-07は、[`plan/UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md`](../plan/UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md)へ統合済みです。

詳細なReference / Evidence mappingは [`plan/evidence/RQ025_RQ026_ACCEPTANCE_TRACE.md`](../plan/evidence/RQ025_RQ026_ACCEPTANCE_TRACE.md) をsupporting Evidence contractとして参照します。この索引自体をRequirementまたはPASの第二正本として拡張しません。

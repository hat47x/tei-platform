# RQ-025 / RQ-026 PAS-07 Evidence Contract

状態：**作業領域Bの検証根拠を補う契約です。PASの基準は`UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md`、作業領域Cの検証根拠の詳細はISSUE-000042を正本とします。**

## 1. 正本関係

本書は、採用済みのRQ-025・RQ-026をPAS-07の検証根拠へつなぐ補足契約です。

```text
PAS-07 user value / Requirement mapping / Gate relation
  -> plan/UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md

Lane C detailed Reference Evidence inventory
  -> ISSUE-000042 A-set

Business Procedure design-quality review

本書
  -> A-set -> E7 / S7 mapping
  -> A-set detailを逐語複製しない
```

参照事例の試験データ・検査器・正本に対する検証は作業領域Cが管理します。判定段階B・Cの必須検証根拠は変更しません。

## 2. PAS-07で壊してはいけない境界

```text
Business Action != Application Function
Business Flow != Procedure != Runtime Workflow Orchestration
Business Handoff != full Business Flow / DFD
Business Actor / Job != Authentication Principal != Authorization Role / Policy
Business Procedure != mandatory single Application child / owner
reference target != Application realization
ref kind separation != ref role separation
Validation PASS != whole-business approval
Application Business Result != external-world Observation
Command accepted != physical-world fact observed
physical-world fact observed != accepted responsibility Handoff
Handoff condition != Observation / Evidence identity
Data trace projection != complete Data dependency authority
no current Data trace != no Data dependency
Projection != second source of truth
explicit unresolved != design failure
unsupported completion = false completeness
```

利用者価値・代表的なシナリオ・要求との対応関係は、主となる追跡文書を正本とします。

## 3. Representation pressure

PAS-07では、正本の要素として`BusinessProcedure`という種類が存在することを前提にしません。

```text
V0  derived Projection
V1  Business固有meaningを持つauthoring source
V2  durable Canonical Business meaning / identity
```

Lane C current candidate stack:

```text
D3    broader thin Business Step body
D4    smaller evidence-backed Step body
S4-A  Procedure-scoped Step local address
H1/R3 thin normalized Handoff / handoff-only relation
C1    optional reporting Application context
O1    Application Pattern protected-target overlay
```

D4 current minimum:

```text
businessAction
applicationPatternRefs[]   // direct realization only
validationRefs[]
capabilityRefs[]
```

データや関係への直接参照が最小限の契約に必要かどうかは、まだ立証されていません。将来、受理済みのソースに不可欠と判明した場合は、安全側に停止して再検討します。いずれも本番向けの採用を意味しません。

事実・受理済みの意味・派生情報・提案・未解決事項というレビュー上の区別は品質のために維持します。ただし、この区別だけを根拠に、新たなV2のメタデータ列挙型を要求することはしません。

## 4. RQ-025 acceptance pressure

- 対応する機能を持たない手作業・外部作業・人の判断・物理的な行為も、業務上の意味として保持する。
- 業務の担当者・職務から、認証主体・認可の役割・権限を自動生成しない。
- `direct realization != protected target / concern`という区別を保持する。
- 業務上の引継ぎを実行基盤のワークフローへ変換しない。
- 目的・起動条件・担当者の文脈・手作業・判断・後続処理を、パターンから派生した情報とは扱わない。
- 業務手順を単一のアプリケーションの配下に閉じ込めない。
- ソースから確定できない担当者・起動条件・順序・実現方式・実際の状態・責任の境界を、根拠のない完成形で埋めない。
- 業務上の指示・システムの結果・外部で実際に起きたこと・業務上の受理と後続処理を、自動的に同一視しない。

## 5. RQ-026 acceptance pressure

- 受理済みの識別情報を正確に参照し、名前の類似や接頭辞から補わない。
- 受理済みのデータ接続先や方針との矛盾は、診断結果として明示する。
- 業務固有の事項・引継ぎ・対象・実際の状態の変更を、アプリケーションが所有する意味の変更として誤って広げない。
- 論理的な位置と引継ぎの端点を、ファイルパスとは区別する。
- レビューと承認を対象の版・照合値に結び付ける。
- 検証根拠がないまま、業務手順・引継ぎ・汎用の参照役割・グラフ・全体共通のステップID・ワークフロー・業務フローを本番向けの定義として固定しない。
- パターンの管理者と方針は正しい権限元から解決し、旧システムの方針から業務上の対象を推測して補わない。
- 正規化した引継ぎの整合性を、ソース索引や参照先の位置だけで代用しない。
- 物理的な状態の観測だけから、データの所有関係・アプリケーションパターン・責任の引継ぎを生成しない。
- データを複数の視点から照合する表には根拠のある追跡情報だけを表示し、追跡できないことを依存関係が存在しないこととはみなさない。
- 派生表示の内容を、業務のソース・アプリケーション・意味の管理権限へ逆流させない。
- 複数の視点を照合するレビューでは、根拠がある状態・矛盾・未解決を区別し、検証根拠の不足を合格や不存在とはみなさない。
- 引継ぎの条件と、その成立を裏付ける観測・文書などを同じ管理権限で扱わない。

## 6. PAS-07 Evidence mapping

作業領域Cの詳細な状態は、ISSUE-000042のA-setを正本とします。この文書が参照する項目は**A1〜A124**です。

次の表はA1〜A83で確立したE7の対応関係の基準です。A84以降の詳細を逐語的に追記して、第二の正本を作ることはしません。A84〜A124は、CRUD操作の権限・複数ソースの記述・鮮度・移行と切替え・ソースの一意性・準備状態の境界について、既存E7の根拠を補強します。

| Evidence | Foundation supporting A-set | Claim |
| --- | --- | --- |
| E7-01 | A2 / A8 / A21 / A25 / A33 / A37 / A67 / A71 / A72 | manual / external / physical meaningをFunctionへ還元せず、targetもrealizationへ誤変換しない |
| E7-02 | A1 / A3 / A4 / A20 / A38 / A39 / A55 / A57〜A59 / A61 / A63 / A66 / A68〜A71 / A75 / A80 / A81 / A83 | exact authority trace、非重複、kind / role / authority direction、synthetic ownership禁止、derived Data trace |
| E7-03 | A5 / A6 / A39 | accepted datasource policy contradictionをbaselineと区別して検出 |
| E7-04 | A9 / A19 / A23 / A28 / A31 / A32 / A45 / A46 / A48 / A49 / A51 / A54 / A65〜A67 / A69 / A72〜A74 / A76 / A78 / A80 / A82 | Business / Handoff / target / actuality / responsibility / Data-trace meaningを分離 |
| E7-05 | A10 / A11 / A25 / A26 / A37 / A38 / A40 / A44 | reuse、multi-Application、single-owner非依存、optional reporting context |
| E7-06 | A12 / A13 / A22 / A23 / A34〜A36 / A41〜A43 / A45 / A46 / A49 / A64 / A65 / A76〜A79 | source move、Step address、explicit Handoff authority、referential integrity |
| E7-07 | A14 / A15 / A17 / A35 | review continuity、snapshot-bound approval、Acceptance target |
| E7-08 | A7 / A27〜A30 / A35 / A44 / A62 / A68 / A70 / A74〜A79 / A82 / A83 | fail closed、unknown保持、policy / Observation / adjacency / partial traceから未証明meaningをbackfillしない |
| E7-09 | A18 / A20 / A24 / A26 / A30〜A34 / A36 / A40 / A41 / A43〜A64 / A69〜A83 | identity / runtime / enum / ownership / graph / Business Flow / generic ref-role / Data projectionをEvidenceなしでproductionへ固定しない |
| E7-10 | A16 / A17 + PAS-01 | authoritative file-native sourceをstale baseから守りfresh Accept後にreload / re-deriveする |

一つのA項目が複数の境界を裏付ける場合があります。この対応関係は網羅率の点数ではありません。

A84〜A124が主に補強する事項は次のとおりです。

```text
A84〜A90
  CRUD projectionをexact operation authorityへ限定
  Data trace / Pattern kind / DB write / LifecycleからC/U/Dを推測しない
  -> E7-02 / E7-04 / E7-08 / E7-09

A91〜A110
  multi-source Business revision
  source freshness != semantic revision != Conformance base
  coordinated rename / migration preview / candidate-only reload
  stale envelope / source-set membership
  -> E7-06 / E7-07 / E7-08 / E7-10

A111〜A120
  existing implementation reuse boundary
  Contract ADR vs Production adoption
  fresh-process feasibility
  C/U/D / ir.World.Business blocker縮約
  -> E7-08 / E7-09 / E7-10

A121〜A124
  duplicate logical Procedure / standalone Handoff fail closed
  endpoint-only Handoff identityを発明しない
  external reference Evidence責務をquestion-specific auditへ限定
  -> E7-02 / E7-06 / E7-08 / E7-09
```

新たなE7項目は追加しません。

## 7. Semantic pressureの現在地

### A54〜A83 — meaning / authority / physical-world boundary

```text
A54〜A59
  realization / Authorization / Observation / explicit-ref authority

A60〜A65
  D4 minimum refs + S4-A + H1/R3 composition

A66〜A69
  ref kind != ref role / O1 / D4+O1

A70〜A71
  Authorization policy authority direction / external-reference negative evidence

A72〜A79
  manufacturing / inventory outline
  physical Observation
  synthetic Data authority deferral
  Handoff non-inference / explicit-source authority

A80〜A83
  exact refsからBusiness Action × Data/Relation V0 traceをderive
  no current Data trace != no Data dependency
  O1 protected-target Dataをhelper / adjacencyから補完しない
```

A80〜A83で確認する変更の影響の向きは次のとおりです。

```text
Business wording-only change + refs unchanged
  -> Data projection unchanged

referenced Pattern / Validation / Capability authority change
  -> Business source unchangedでもData projectionは追随

missing / ambiguous referenced identity
  -> fail closed
```

### A84〜A124 — operation / authoring / freshness / readiness boundary

A84〜A90では、CRUD操作を意味上のデータの追跡情報と区別します。A91〜A124は、現在の意味の候補を増やすための項目ではありません。記述形式・ソースの発見・鮮度・移行時の切替え・実装の再利用・契約に関する設計判断と本番採用の境界を、より厳密に検証します。

詳細な状態については、ISSUE-000042のA-setを正本とします。

## 8. O1 / B-F27

```text
role separation is representable
  != protected target is authoritative Business source meaning
  != reload / RQ-002 trace / Shared Acceptance preserves it
```

B-F27は、意図的かつ条件付きで未解決のままにしています。

A83により、将来、保護対象との関係を受理済みの意味として保持する場合、データの派生表示と変更影響分析の対象範囲に具体的な影響があると分かりました。ただし、現在の一例だけを根拠に、O1の永続化や汎用的な役割の分類体系を本番向けに採用することはしません。

## 9. Secondary semantic boundaries

### S7-01 — Validation / business judgment

人による業務判断と決定論的な検証規則の評価を、一つの業務行為としてまとめません。実際の実行と承認の意味は、PAS-02の検証結果と照合します。

### S7-02 — actor / Authorization

```text
Business actor / job != Principal / Role / Permission
BP-06 supporting authority -> security.authorize
BP-06 protected target     -> O1 pressure
Application Function       -> none
```

旧システムの認可方針は対象を裏付ける材料にはなりますが、その方針から業務上の対象を新たに記述したり推測して補ったりはしません。実際の認可の強制はPAS-02が扱います。

### S7-03 — Business Result / Observation / continuation

```text
Business instruction
  != Command accepted
  != Observation read succeeded
  != physical fact observed
  != Business acceptance / responsibility Handoff
```

実行環境の`ExecuteObservationRead`と`ReadIdentity`を、業務や意味の識別情報へ昇格させません。A73・A74は機械的な検証項目ですが、PAS-02全体の実行環境の検証根拠には代わりません。

引継ぎ条件は業務の後続処理を判断する条件であり、観測や検証根拠の識別情報そのものではありません。専用の検証根拠の項目が必要かどうかは、現実的な参照事例で意味が失われる問題が見つかるまで判断を保留します。

### S7-04 — Data trace / dependency completeness / change direction

```text
evidence-backed Data trace exists
  != complete Business Data dependency known

NoDataTraceFromCurrentRefs = true
  != no Data dependency

Business source + referenced accepted authorities
  -> V0 Data projection

V0 Data projection
  -X-> author Business source
  -X-> author Application / Semantic authority
```

業務の文言だけを変更し、参照が変わらなければ、データの派生表示には影響を及ぼしません。一方、参照先の権威ある定義が変わった場合は、派生表示にも反映します。識別情報が欠ける場合は安全側に停止します。

A80〜A83による派生表示は、データを定義する権限・所有関係・CRUD操作・網羅性の判断を担いません。

### S7-05 — Evidence nature / false completeness

```text
source-backed fact
accepted TEI Business meaning
derived cross-view fact
proposal / inference
unresolved
```

これらはレビュー上で区別します。現時点では正本の列挙型として固定しません。

特に、公開情報や外部の概略情報から、TEIの担当者・手順のステップ・引継ぎを確定した定義として自動生成しません。図を完成させるためだけに未解決事項を補うこともしません。

## 10. Gate 15 / RRB

```text
15a synthetic multi-Application structural pressure
15b current order_entry social-system realism
15c broader realistic pressure
    = concrete minimum-contract insufficiencyが出た場合だけopen
```

当時のソースを使った検証では、契約に不足する具体的な事例が見つからなかったため、15cは開始していません。

```text
Gate 15c not activated
  != realistic Reference maturity established
```

B-F19b status:

```text
RRB-P1 provenance-safe TEI-native second Business source exists
RRB-P2 P1 sourceをsource/test cross-viewへ統合
RRB-P3 actual execution-qualified breadth Evidence
```

ISSUE-000050では、出所を確認できるソース・検証根拠の性質・業務用の初期データを受理する契約まで整理しましたが、当初は受理済みの業務用初期データをまだ記述しておらず、P1・P2・P3は未達でした。（2026年9月21日の追記：初期データを記述し、P1・P2〔ソース・試験の段階〕・P3の検証根拠を取得しました。レビューは作成者本人のみです。ISSUE-000050の実施記録を参照。）

## 11. Current findings / status

```text
Lane C A-set
  A1〜A124

B-F27 authoritative protected target
  intentionally / conditionally unresolved
  projection coverage consequenceは具体化

H1/R3 authority direction
  explicit source only
  source/test-level strengthened

Business design quality
  fact / accepted / derived / proposal / unresolvedをreview上分離
  false completenessをreject
  production metadataは未採用

second realistic corpus
  manufacturing / inventory pressure selected
  Business Seed Acceptance Contract prepared
  TEI-native seedをauthor済み（2026-09-21、作成者self-review。独立review未実施）

RRB-P1 / P2 / P3
  Evidence取得済み（2026-09-21、R = 82e4301。P2の項目4は未確認。ISSUE-000050実施記録）

actual gofmt / go vet / go test / checker green
  unresolved

production Canonical contract
  not adopted

Pre-ADR
  Not Ready
```

## 12. Existing PASとの関係

- PAS-01：修正・レビュー・共通の受理を扱います。E7-10の一連の処理が成立したことを示すには、実際の検証根拠が必要です。
- PAS-02：検証規則・認可・業務上の結果・観測について、実行時の意味を扱います。
- PAS-04：実行先が受理済みの業務・機能・データの意味を損なわないことを確認します。ただし、PAS-07を判定段階Cの必須検証根拠には加えません。
- PAS-06：参照事例やパターンの知識を探索する機能と接続できますが、それだけで業務手順が成立したとは扱いません。

## 13. Requirement -> PAS

| Requirement | Primary PAS | Secondary / cross-cutting |
| --- | --- | --- |
| RQ-025 | PAS-07 | PAS-02 / PAS-04 |
| RQ-026 | PAS-07 | PAS-01 / PAS-02 / PAS-04 |

RQ-018・RQ-021・RQ-022にも横断的に関係します。要求とPASの対応関係は、主となる追跡文書を正本とします。

## 14. Gate / Stage

PAS-07を理由に、現在の判定段階を変更することはしません。

```text
Gate A / B / C
  existing definitions unchanged

PAS-07 Reference Evidence
  Gate B Required Evidenceへ追加しない
  Gate C First Web Target Required Evidenceへ追加しない
```

将来、「業務システムの設計として成熟した」と主張する場合は、PAS-07を実際に実行した検証根拠と、RRB-P3に相当する現実的な事例の広がりを別途求める可能性があります。ただし、現時点で新たな判定段階は設けません。

## 15. 更新規則

1. 作業領域CのA-setを詳細の正本とします。この文書はE7・S7との対応関係と、範囲ごとの要約だけを更新します。
2. 新たな検証項目はE7を増やす前に、既存のE7へ位置付けられるかを確認します。
3. A-setの項目を追加するたびに表を逐語的に拡張することはしません。意味の分類が変わった場合だけ、対応関係を更新します。
4. B-F27は、実際の検証根拠から権威ある保存が必要と判明した場合に限り、再検討します。
5. S7-01〜05はPAS-02と受理済みの意味を管理する権限に照らして確認し、表現形式・実行環境・網羅性・検証根拠の性質を混同しません。
6. Pre-ADRの準備状況は作業領域Cの正本から導出し、作業領域Bに独立した判定段階の一覧を作りません。
7. RRB-P1・P2・P3とGate 15cの状態を一つにまとめません。
8. 観測・隣接関係・機能の実行順序・担当者の違いから、データを管理する権限や引継ぎを推論しません。
9. データへの追跡情報がないことを、依存関係がないこととは扱いません。追跡情報の派生表示を第二の正本にすることもありません。
10. 派生表示を充実させることだけを目的に、V1の記述元へデータの直接参照やO1の保護対象を追加しません。
11. CRUD操作の意味は、操作を定義する権限について検証根拠に基づく導出ができるまで追加しません。
12. PAS-07の利用者価値・要求との対応関係・判定段階への影響が変わる場合は、主となる追跡文書を先に更新します。

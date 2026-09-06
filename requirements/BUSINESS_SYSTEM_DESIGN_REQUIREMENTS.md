# TEI 業務システム設計 要求解釈索引

Status: current interpretation index

## 1. この文書の位置づけ

この文書は、TEIの業務システム設計に関するProduct Requirementと設計文書を結ぶ索引です。

正式なProduct Requirementの正本は [`TEI_PRODUCT_REQUIREMENTS.md`](TEI_PRODUCT_REQUIREMENTS.md) です。この文書を第二のRequirement source of truthにはしません。

現在、業務システム設計について特に直接的な要求として次が採用済みです。

- `RQ-025` — 業務上の行為・手順をApplication Functionから独立して設計できること
- `RQ-026` — Data / Function-Interaction / Business Procedureの異なる視点から相互検証できること

詳細な概念・Architecture境界は [`docs/business-system-design.md`](../docs/business-system-design.md) を参照します。

## 2. 設計原点へ立ち返る順序

判断に迷う場合は、次の順で設計原点へ立ち返ります。

1. 渡辺幸三氏の業務システム設計・三要素分析法に関する明示的な言説
2. X-TEA / XEAD Modeler / Driverの公開マニュアル・モデル構造
3. XeadModeler / XeadDriver公開ソース
4. Reference Modelでの実際の使われ方
5. 現行TEI Product Requirements / Architecture / Reference / Evidence
6. 小さいTEI Reference実験

作業規則は [`plan/UPSTREAM_XTEA_DESIGN_FALLBACK_PROTOCOL.md`](../plan/UPSTREAM_XTEA_DESIGN_FALLBACK_PROTOCOL.md)、調査Evidenceは [`plan/XTEA_DESIGN_EVIDENCE_REGISTER_2026-09.md`](../plan/XTEA_DESIGN_EVIDENCE_REGISTER_2026-09.md) に置きます。

このfallbackは歴史的shapeをそのままTEIへ移植するための規則ではありません。歴史的概念が防いでいた設計上の失敗を確認し、現行TEIのmeaning / authority / Provider / Target境界へ翻訳します。

## 3. RQ-025の解釈

RQ-025では少なくとも次を分離します。

```text
Business Action
    != Application Function / Application Pattern
    != Capability
    != Realization
```

Business ActionはApplication Functionを持たなくても正当です。人間、AI、外部組織、SaaS、物理作業、既存システム等によるmanual / external workを、実行可能に見せるためだけに架空Capabilityへ変換してはいけません。

Business Procedureでは、必要に応じてpurpose、trigger / event、actor、business action、manual / external work、Application Function利用、handoff、completion / continuation等を説明します。

ただしRQ-025は、`Procedure`を必ずtop-level Canonical objectにすることや、Workflow Engine、branch、timer、retry、scheduler、orchestration semanticsをCoreへ追加することを要求しません。

また、

```text
Business Actor / Job
    != Authentication Principal
    != Authorization Role / Policy
```

を維持します。

## 4. RQ-026の解釈

RQ-026では、Data、Function / Interaction、Business Procedureを同じ業務システムに対する異なるreview viewとして扱います。

```text
Data / Relation / Validation
          ↕
View / Pattern / Query / Action / Capability
          ↕
Business Procedure / actor / manual / trigger
```

三つの同型Canonical objectを作ることが目的ではありません。

一つの視点で見つかった矛盾を、他の視点とauthoritative sourceへ戻して修正できることがProduct Valueです。

代表的なcross-view pressureは次です。

- Business Procedureが必要とする情報がDataにない
- UI / Functionがread-only datasourceへwriteを要求する
- Business ActionがApplication Functionへ過剰に還元される
- actor labelがAuthorization Roleへ暗黙変換される
- Data / Relation変更がProcedure / Viewへ与える影響を追えない
- Application Business Resultを外部世界の業務完了と混同する

Matrix、CRUD表、flow、graph等は可能な限りexplicit referenceからderivedし、第二正本として二重管理しません。

## 5. 関連する既存Requirement

RQ-025 / RQ-026は独立して追加されましたが、次の既存Requirementと組み合わせて成立します。

- RQ-001 — 業務意味と実装技術の分離
- RQ-003 — 暗黙の複数正本を作らない
- RQ-010 — 高密度Editor
- RQ-011 — 複数Projection
- RQ-012 — Driverによる設計検証
- RQ-014 — Application Pattern再利用
- RQ-017 — 二重管理最小化
- RQ-018 — Command / Observation分離
- RQ-019 — authority / Resource境界
- RQ-021 — Reference / Conformance
- RQ-022 — 設計検証段階と完成製品の区別

## 6. Evidence待ちとして残す論点

次はProduct Requirement採用済みであっても、表現方式はReference Evidenceから決めます。

1. Business Procedureにdurable Canonical identityが必要か
2. actor / event / handoff / decision / manual instructionの構造化範囲
3. DFD型Business Flowの最小表現
4. `Subsystem` / `SubjectArea`のdurable semantic boundary
5. Target Adapter前のTarget-neutral View / Interaction構造
6. realistic Reference Modelのscale / packaging / licensing
7. temporal semanticsのdomain meaning境界
8. cross-view diagnosticのseverityをどこまでdeterministic Conformanceへ昇格するか

これらを埋めるために、X-TEAに同名概念があるという理由だけでCore kindを追加しません。

## 7. Acceptance Trace

RQ-025 / RQ-026のProduct Acceptance Scenario、Reference、Evidence位置は [`plan/RQ025_RQ026_ACCEPTANCE_TRACE.md`](../plan/RQ025_RQ026_ACCEPTANCE_TRACE.md) にまとめます。

最終的には既存の `UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md` へ統合し、この索引自体をRequirement正本として拡張しません。

# Business Handoff Second Realistic External Evidence Pressure — 2026-09

Status: **Lane C / external design evidence pressure / no second-corpus fixture / Gate 15c not activated / no production schema adoption**

## 1. 目的

Pre-ADR Gate Cには:

```text
second realistic Handoffで
from / to / condition sufficiencyを確認
```

が残っている。

Actual Repository runner green前にsecond corpusをTEI fixtureへ昇格しない方針は維持する。

本書では、公開されている三要素分析法 / 上流工程資料から、H1/R3を壊し得る**realistic external design pressure**だけを抽出する。

Exact Procedure / Step / actor / Data / event / Handoffを外部資料から捏造しない。

## 2. Source

### W-01 — 三要素分析法集中講座（2）1/3

渡辺幸三「データ、UI、業務手順の『3要素』をとらえる：分析設計技法『三要素分析法』集中講座（2）」ITmedia / ＠IT, 2006-05-09。

Public URL:

`https://www.itmedia.co.jp/im/articles/0605/09/news085.html`

Relevant source meaning:

- Data model / Function model / Business modelは企業システムを異なる方向から見る独立viewである。
- 三viewは独立しつつ相互に牽制する。
- FunctionはBusinessとDataの間を接続する位置にある。
- 一viewだけを独立に完成させるより、相互照合で不足を発見する。

Lane C interpretation:

```text
Data / Function-Interaction / Business
  -> one modelへ潰さない
  -> independent authority / projection boundaryを維持
  -> cross-view contradictionで不足を発見
```

Current Lane C architectureを強くcorroborateするが、TEI field / typeを直接authorするsourceではない。

### W-02 — 三要素分析法集中講座（2）3/3

同記事3/3:

`https://www.itmedia.co.jp/im/articles/0605/09/news085_3.html`

Relevant source meaning:

- Business modelは業務手順 / manualとして、職務・業務・手順を扱う。
- Business systemをsocial systemとして捉え、computer外の主体の動きも基本設計対象とする。
- 業務フローは三要素を広く俯瞰する別viewである。
- 業務フローは拡張DFDで表し、processがどのeventで起動されるかも示す。
- 現場の仕事は単純な業務間logicではなくevent-drivenに配置される、という説明がある。

Lane C interpretation:

```text
Business Procedure
  != Application Function sequence

Business Handoff
  != Step adjacency

Business Handoff
  != full Business Flow / DFD

Business work continuation
  may depend on event / social responsibility context
```

をcorroborateする。

### W-03 — 『業務システムのための上流工程入門』目次情報

Public bibliographic / summary sourcesでは、業務フローのpatternとして少なくとも次の分類が確認できる。

```text
要求の認識と報告
作業指示の立案と発行
作業の実施と報告
実績の集計と分析
```

Source example:

`https://doc.dev1x.org/book/72/`

これは目次レベルの情報であり、exact process definition / actor / event / data / handoff conditionのauthorityには使わない。

## 3. Current H1/R3へのpressure

Current H1/R3:

```text
Handoff
  from { procedure, step }
  to   { procedure, step }
  condition
```

W-02 / W-03から、second realistic pressureとして次を考えられる。

```text
responsibility A
  recognizes / reports a need

responsibility B
  plans / issues an instruction

responsibility C
  performs / reports work

responsibility D
  aggregates / analyzes actuals
```

ただしこれは**pattern sequence candidate**であり、W-03の見出しからexact Handoffをauthorしてはいけない。

例えば:

```text
instruction issued
  -> execution may become eligible to begin
```

はrealistic hypothesisにはなるが、sourceからaccepted factとして確定しない。

## 4. H1/R3がまだ壊れていない理由

当該外部資料のEvidenceはBusiness Flow / event semanticsがProcedure bodyより広いことを示すが、Handoff relationそのものに次が必須だとはまだ証明しない。

```text
eventId
businessDocument
payload/data flow
participant role taxonomy
state machine guard
executable predicate
```

Current H1/R3は:

```text
explicit responsibility continuation
  from exact local Business address
  to exact local Business address
  + accepted Business condition wording
```

だけを所有する。

W-02のevent-driven Business FlowをHandoff fieldへ複製せず、必要ならBusiness Flow側のseparate pressureとして扱う。

したがって現時点では:

```text
H1/R3 insufficient
  -> not proven

full Business Flow required for Handoff
  -> not proven
```

である。

## 5. Handoff conditionへのimplication

W-02からは、Business continuationが単なる前Step完了ではなくevent / surrounding business situationに依存し得ることが強く示唆される。

これはcurrent:

```text
Step adjacency
  -X-> Handoff

explicit accepted condition
  -> Handoff meaning
```

を補強する。

一方conditionを今すぐ:

- executable Boolean expression
- runtime event guard
- Workflow transition condition

へ変換しない。

Natural-language accepted Business conditionがsecond realistic caseでlossyになるEvidenceが出た場合だけreopenする。

## 6. Procedure triggerとの関係

Current Procedureには:

```text
trigger { epistemicStatus, meaning }
```

がある。

W-02のevent-driven explanationは、Business Procedure triggerを保持する方向をcorroborateする。

ただし:

```text
Handoff condition
  == target Procedure trigger
```

とはしない。

理由:

- ProcedureはHandoff以外のeventから始まる可能性がある。
- Handoffはresponsibility continuationのrelation meaningである。
- triggerはProcedure開始条件 / contextの意味である。
- 一つを他方から自動deriveするとauthority directionを失う。

将来second realistic sourceで両者のsame/different caseをPressureする価値がある。

## 7. SubjectArea pressure

W-02ではSubject Areaが業務フローを俯瞰し、modeling work unitとして使われる説明がある。

これはcurrent Lane Cの:

```text
Namespace != SubjectArea
ownership/access lens != Subsystem
```

を否定しない。

むしろSubjectAreaにはtechnical Namespaceとは別のBusiness modeling roleがあり得ることを補強する。

ただしexternal conceptual evidenceだけでCanonical `SubjectArea`を導入しない。

Reopen条件は引き続き:

```text
current derived lensでは
realistic TEI Referenceをlosslessにreview / partitionできない
```

具体例が出た場合とする。

## 8. Second realistic Handoff acceptance seed

Actual green後にGate Cのsecond realistic Handoffを作る場合、外部資料のPatternから直接fixtureをコピーせず、TEI-original small caseをauthorする。

Candidate pressure shape:

```text
Procedure A
  manual / social-system responsibility
  reports / makes an accepted business fact available

Procedure B
  separate responsibility
  may continue only when an explicit business condition holds
```

Minimum questions:

1. `from {procedure, step}`でsource responsibilityをlosslessに指せるか。
2. `to {procedure, step}`でnext responsibilityをlosslessに指せるか。
3. natural-language `condition`でaccepted continuation meaningを保持できるか。
4. target Procedure triggerとHandoff conditionを別authorityとして維持できるか。
5. Business document / event identityがHandoffに必須になるか。
6. actor / job changeがHandoff endpointだけで十分に理解できるか。
7. Business Flow / DFDを導入しなくてもHandoff reviewを成立させられるか。

このうち5〜7で具体的lossが出た場合だけH1/R3またはBusiness Flow pressureをreopenする。

## 9. Gate 15cとの関係

本external evidence reviewだけではGate 15cを開かない。

```text
external conceptual pressure
  != TEI-original realistic fixture
  != current minimum insufficiency
```

当該外部資料のEvidenceはcurrent architecture boundaryのcorroborationとして使い、production schemaを外部手法へ合わせる目的では使わない。

## 10. Current conclusion

```text
three-view mutual constraint
  -> strong external corroboration

Business model as social / procedural meaning
  -> strong external corroboration

Business Flow broader than Procedure / Handoff
  -> strong external corroboration

Business work event-driven rather than adjacency-driven
  -> Handoff non-inference boundaryを補強

H1/R3 from/to/condition insufficiency
  -> not yet proven

second realistic Handoff fixture
  -> actual Repository green後のTEI-original authoringへdefer

SubjectArea Canonical adoption
  -> not proven

Gate 15c
  -> not activated
```

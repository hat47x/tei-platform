# Business Procedure Step Identity Pressure — 2026-09

Status: **Lane C / test-only design pressure / no Core adoption**

## 1. 問い

Business Procedure Stepの`BP-04`等のkeyは、production Canonical body meaningそのものに必要なdurable identityか。

current ReferenceではStep keyが二つの役割を持つ。

1. Procedure内部のauthoring / review上のlocal key
2. handoff endpointを外部relationから指すaddress

この二つを同一視しない。

## 2. body meaningからのidentity subtraction

Step IDを除いたbody candidate:

```text
Procedure
  id
  purpose
  trigger
  actor
  ordered steps

Step
  businessAction
  explicit semantic refs
```

ordered sequenceは保持する。

このcandidateではStep local key renameだけではProcedure body meaningを変えない。

## 3. addressability pressure

一方、current handoffは

```text
Procedure/Step -> Procedure/Step
```

をexact endpointとして使う。

handoff元`BP-09`だけをrenameしReference setを更新しなければstale linkとしてfail closedする必要がある。

位置や名称類似から補完しない。

したがって、**body meaningにStep IDが不要であること**と、**cross-referenceのためにstable Step addressが必要であること**は両立する。

## 4. test-only Pressure

`tools/reference-business-procedure-check/step_identity_pressure_test.go`で次を確認する。

1. Procedure bodyをStep IDなしで投影できる。
2. Step local key renameだけではID-free body meaningを変えない。
3. Step order入れ替えはbody meaning changeとして残る。
4. current handoff endpointのStep key renameはstale linkとしてfail closedする。
5. endpoint identityへruntime execution / instance / state identityを混ぜない。

actual Repository runnerで実行されるまではgreen Evidenceではない。

## 5. production候補の選択肢

### Option S1 — Step IDをCanonical body identityにする

長所:
- handoff / review / trace targetが直接安定する
- cross-revision linkが単純

リスク:
- authoring key renameをBusiness identity changeへ過剰昇格する
- body meaningとaddressing concernを結合する

### Option S2 — bodyはID-free、別のaddress identityを持つ

長所:
- body semantic meaningとcross-reference identityを分離できる

リスク:
- identity object / mappingが増える
- Source / Canonicalの二重管理を作る危険

### Option S3 — Reference-local Step keyのまま

長所:
- Core拡張を避けられる

リスク:
- accepted handoff relation / review continuity / change graphがSource artifactへ依存し続ける可能性

## 6. 現時点の判断

Step durable Canonical identityはまだ採用しない。

current Evidenceが示したのは次までである。

```text
ordered Business Action body meaning
  does not inherently require Step ID

external handoff / review addressability
  does require a stable exact endpoint
```

次に必要なのは、accepted cross-procedure handoffやrevision-aware reviewがSource-local keyだけで安全に維持できるかのEvidenceである。

Source-local keyで十分ならCoreへStep identityを増やさない。十分でなければ、body semantic identityとaddress identityを分離する案も含めて再評価する。

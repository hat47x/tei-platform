# Business Procedure Minimal Identity Shell Pressure — 2026-09

Status: Test-only Evidence in progress / useful identity primitive / insufficient as standalone Canonical meaning

## 目的

Business Procedureのpath-independent identityを、source pathやruntime semanticsから分離できるか確認する。

このPressureはCanonical `Procedure`採用ではない。

## 1. shellの上限

probe typeは意図的に次だけを持つ。

```text
CanonicalProcedureIdentityShellProbe
  ID
```

次は持たせない。

- purpose
- trigger
- actor
- step
- Business Action
- instruction
- handoff
- branch
- lifecycle
- state
- timer
- retry
- authorization mapping
- source path
- fingerprint
- Source range
- runtime orchestration

## 2. authoritative sourceから導出する

identity shell用の第二ID fileは作らない。

```text
file-native Business Procedure source
  ↓ strict decode
procedure.id
  ↓ normalized
identity shell
```

同一IDを複数authoritative sourceが宣言した場合はfail closedにする。

## 3. 成立したidentity性質

### pure file move

source bytes / Procedure IDが同じならidentity shellは不変。

### Business wording change

purpose / Business Action等が変わってもProcedure IDが同じならidentity shellは不変。

### logical rename

`procedure.id`変更はidentity change。

名称類似からrenameを推測しない。

### Source Index target

```text
kind = business-procedure
id   = Procedure ID
```

というpath-independent Subject候補を作れ、Artifact path / fingerprint / rangeはSource Index側へ残せる。

## 4. 重要な追加反証

前段では、このidentity shellだけでRQ-002のCanonical trace targetを作れれば最小十分かもしれないと考えた。

その後の`BUSINESS_PROCEDURE_CANONICAL_MEANING_PRESSURE_2026-09.md`で、次の不足が明確になった。

```text
same Procedure ID
+
changed accepted Business meaning
```

ではidentity shellは変わらない。

したがってIDだけでは、

- Business-only Semantic Diff
- Business change -> Data / Function impact
- accepted Business meaningのCanonical exploration
- reload後Business Conformance
- 000039 change graph上のcontent change

を表現できない。

## 5. 更新後の位置づけ

identity shellは不要ではない。

むしろ、thin Canonical Business Procedure meaningの

```text
ID部分
```

として有用である可能性が高い。

ただし、**ID-onlyをproduction `ir.World`へ先行導入する判断は停止する。**

比較対象は次とする。

```text
A: identity only
vs
D: identity + cross-viewに必要な薄いBusiness residual meaning
```

現時点のEvidenceはD側を支持し始めている。

## 6. Option Dとの境界

thin Business meaning候補は次を保持する。

```text
Procedure
  id
  purpose
  trigger
  actor
  ordered steps

Step
  id
  businessAction
  realization
  explicit semantic refs
```

一方、identity shellで排除したruntime semanticsはOption Dでも排除する。

- retry
- timer
- queue
- saga
- scheduler
- runtime state
- Authorization Role

したがって「identity-onlyでは足りない」ことを理由にWorkflow Engineへ飛ばない。

## 7. handoffとの関係

handoffはProcedure bodyへ埋め込むのではなく、

```text
Procedure/Step -> Procedure/Step
```

のthin Business semantic relationとして別Pressureを行っている。

詳細:

- `BUSINESS_PROCEDURE_HANDOFF_2026-09.md`（Part 1、旧`HANDOFF_CANONICAL_PRESSURE`）
- `canonical_handoff_pressure_test.go`

## 8. Verification

`tools/reference-business-procedure-check/identity_shell_pressure_test.go`

主なtest:

- ID-only field guard
- authoritative sourceからidentity導出
- duplicate ID reject
- pure file move不変
- Business wording changeでidentity不変
- logical renameでidentity change
- Source Index Subject候補

actual Repository runnerがcheckout / Go testへ到達するまではgreen Evidenceとは数えない。

## 9. production判断

identity shell単独ではCore integrationへ進めない。

次の全体Pressureを先に満たす必要がある。

1. thin Procedure body meaning
2. thin handoff relation
3. Pattern-owned meaningの非重複
4. manual / external work保持
5. Business-only semantic diff / impact
6. file move / logical rename
7. actual Repository test green

その結果、full Business contentをCanonical layerへ置かずとも同等のdiff / impact / explorationが成立すると実証できた場合だけ、identity-only案を再評価する。

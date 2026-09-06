# Business Procedure Realization Subtraction Pressure — 2026-09

Status: **Lane C / test-only design pressure / no Core adoption**

## 1. 問い

current Business Procedure fixtureの`step.realization`は、production Canonical Business meaningに本当に必要か。

現在のReferenceでは次の値が使われる。

```text
application-use
manual-unresolved
validation-support
security-boundary
observation-boundary
manual-external-unresolved
```

これらはレビュー時には有用だが、異なる軸を一つの文字列へ畳んでいる可能性がある。

## 2. 反証対象

`realization`を削ると、次まで失うならCore field候補として残す必要がある。

- manual / external Business Action
- Business-only semantic diff
- explicit Application Pattern relation
- cross-view review

逆に、これらが`businessAction + explicit refs`で保持できるなら、`realization`をproduction enumへ固定する必要は弱い。

## 3. test-only subtraction model

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
  explicit semantic refs
```

`realization`を持たない。

## 4. Pressure

`tools/reference-business-procedure-check/realization_subtraction_pressure_test.go`で次を確認する。

1. canonical candidate step shapeに`realization`がない。
2. `realization` labelだけを変えてもsubtracted meaningは変わらない。
3. BP-04 manual Business ActionはBusiness Action wordingとして残る。
4. BP-04へfake Application Pattern / Capabilityを生成しない。
5. Business Action wordingを変えればmeaning changeになる。
6. explicit Application Pattern refを変えればmeaning changeになる。

actual Repository runnerで実行されるまではtest-only specificationでありgreen Evidenceではない。

## 5. なぜ現時点でCore enumへしないか

### `manual-unresolved`

manual / systemというrealization axisと、candidate / unresolvedというepistemic axisを一語へ混ぜている。

### `security-boundary`

Business review上は重要だが、Authorization Role / Principal / PermissionをProcedureが所有することを意味しない。

### `observation-boundary`

Command / Observation分離をBusiness上説明するが、特定runtime implementation kindではない。

### `application-use`

Application Pattern refが既に明示されている場合、system interactionの存在を別fieldで重複Authority化する可能性がある。

## 6. 現時点の判断

`realization`はReference authoring / review labelとして残してよい。

ただしproduction Canonical contract候補からはいったんsubtractし、Business meaningに欠落が出るかを実Evidenceで確認する。

欠落が見つかった場合もcurrent複合文字列をそのままenum化せず、必要なBusiness distinctionだけを抽出する。

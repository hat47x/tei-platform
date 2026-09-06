# Business Procedure Minimal Identity Shell Pressure — 2026-09

Status: Test-only Evidence in progress / no Core adoption

## 目的

`BUSINESS_PROCEDURE_IDENTITY_OPTION_AUDIT_2026-09.md`で第一候補となったOption Aについて、production `ir.World`を変更する前に、identity-only contractがReference圧力を満たせるかtest-onlyで確認する。

このPressureはCanonical `Procedure`採用ではない。

## 1. shellの上限

probe typeは意図的に次だけを持つ。

```text
CanonicalProcedureIdentityShellProbe
  ID
```

reflect testでfield数を1に固定する。

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

Business meaning本体とSource metadataをidentity shellへ侵入させない。

## 2. authoritative sourceから導出する

identity shell用に別のID fileを作らない。

現在のBusiness Procedure authoring sourceから

```text
procedure.id
```

だけを導出する。

```text
file-native Business Procedure source
  ↓ strict decode
reference-local stable ID
  ↓ derived
identity shell
```

したがってidentity shellは第二authoritative sourceではない。

## 3. duplicate identity

同じBusiness Procedure IDを複数sourceが宣言した場合はfail closedにする。

```text
source A -> order-entry.order-handling
source B -> order-entry.order-handling
```

をheuristic mergeしない。

これはRQ-003の単一正本境界と整合する。

## 4. pure file move

同じsource bytesを別pathへ移した場合、identity shellは不変である。

```text
path
  changed

Business Procedure ID
  same

identity shell
  same
```

source locatorをCanonical identityへ混ぜない。

## 5. Business wording change

BP-04等のBusiness Action文言だけを変更してもidentity shellは変わらない。

Business source fingerprint / Business Projectionは変わるが、path-independent identityは同じである。

```text
same Procedure identity
changed Business source snapshot
```

を表現できる。

## 6. logical rename

`procedure.id`自体を変更した場合はidentity shell changeになる。

```text
order-entry.order-handling
  -> order-entry.order-handling-renamed
```

はfile moveではなくlogical identity changeとして扱う。

名称類似からrenameを推測しない。

## 7. Source Index target

probe shellから次のSubjectを構成する。

```text
kind = business-procedure
id   = identity shell ID
```

Business Procedure source Artifactからwhole-file Rangeを作り、`RoleDefinition`のReferenceとしてSource Indexへ登録するpressure testを追加した。

確認する性質:

- Subject IDはpathではない
- Subject IDはfingerprintではない
- Artifact path / fingerprintはSource Index側に残る
- identity shellとsource locatorを分離できる

現時点ではJSON identifier用exact Range parserを作らずwhole-file Rangeでpressureを確認する。

production採用時にexact anchorが必要なら別に実証する。

## 8. RQ-002に対する意味

このshellがproduction Worldへ統合されれば、概念上次が成立する。

```text
Business Procedure authoring source
   ↓
Canonical Business Procedure identity
   ↓
Source Index Subject
```

これによりRQ-002のCanonical trace targetを、full Procedure modelなしで作れる可能性がある。

重要なのは、Canonical化するのがBusiness Procedureの全内容ではなく**identityだけ**という点である。

## 9. まだ実証していないこと

本Pressureはtest-only typeであり、まだ次を変更していない。

- `internal/ir.World`
- production workspace loader
- production `procedures` source root
- Normalize
- World validation
- golden World
- Source Index production subject kind
- Shared Acceptance production API
- 000039 impact graph

したがって「Option Aのbehavioral contract候補」を検査しているだけである。

## 10. Core integration前の停止条件

次のいずれかが成立したらOption AをCoreへ入れず比較へ戻る。

- test-only shellでもBusiness detailが必要になる
- identityをsourceから一意に導出できない
- file moveでidentity維持ができない
- duplicate authorityを許容しないと成立しない
- Source Indexへ結ぶためsource metadataをshellへ入れる必要が出る
- Shared Acceptance targetとしてReference-local keyとの差が説明できない

## 11. Core integrationへ進める条件

actual Repository testがgreenになり、かつ次が成立する場合だけ、次段でproduction integration pressureを検討する。

1. shellはID-onlyを保つ
2. authoritative Business sourceからderivedできる
3. duplicate IDをfail closedにできる
4. file move / wording change / logical renameを期待どおり分離できる
5. Source Index subjectとしてpath-independentに利用できる
6. RQ-002のCanonical trace targetとして明確な追加価値がある

その場合も最初のproduction experimentでは`purpose` / `step`等をWorldへ追加しない。

## Verification

`tools/reference-business-procedure-check/identity_shell_pressure_test.go`

主なtest:

- `TestCanonicalProcedureIdentityShellProbeContainsOnlyID`
- `TestCanonicalProcedureIdentityShellProbeDerivesFromAuthoritativeSource`
- `TestCanonicalProcedureIdentityShellProbeRejectsDuplicateSourceIdentity`
- `TestCanonicalProcedureIdentityShellProbeSurvivesPureFileMove`
- `TestCanonicalProcedureIdentityShellProbeIgnoresBusinessWordingChange`
- `TestCanonicalProcedureIdentityShellProbeTreatsLogicalRenameAsIdentityChange`
- `TestCanonicalProcedureIdentityShellProbeCanBackWorldStyleSourceSubjectWithoutSourceMetadata`

actual Repository runnerがcheckout / Go testへ到達するまではEvidence in progressとする。

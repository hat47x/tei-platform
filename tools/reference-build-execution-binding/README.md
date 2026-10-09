# Build Execution / Subject Binding Reference

GitHub #1080 / #1097 / #1102 のread-only Referenceです。

現行の`internal/plugin.BuildPlan`は、一つのPlugin buildを正確に表す契約として維持します。このReferenceはproduction型を変更せず、そこから「build execution」と「logical Plugin subject」を分けて表した場合に何が残るかを検証します。

## 分けるidentity

```text
BuildExecutionPlan
  exact Sources
  exact Builder
  Recipe
  Output Contract
  build-only Dependencies
  Target / Profile / Parameters

ArtifactSet
  physical output members
  role / format / digest
  locator is not content identity

BuildExecutionResult
  ExecutionPlan fingerprint
  ArtifactSet fingerprint

PluginSubjectBinding
  Plugin identity
  Plugin definition fingerprint
  ExecutionPlan fingerprint
  ArtifactSet fingerprint
  consumed output roles

RuntimeEngineSubjectBinding
  Runtime Engine identity
  ExecutionPlan fingerprint
  ArtifactSet fingerprint
  consumed output role
```

Sourceの`Locator`もcontent identityとは分けます。exact digestが同じであれば、Repository pathやstore locatorが変わっただけではBuildExecutionPlan fingerprintを変えません。Locatorは再取得・観測のために保持しますが、artifact-affecting input identityではありません。

重要なのは、同じartifact bytesと同じbuild executionを同一視しないことです。

```text
same bytes + different build inputs
  -> same ArtifactSet content identity
  -> different BuildExecutionResult identity
```

逆に、一つのbuild execution / physical artifactへ複数logical Pluginを結ぶ場合は、executionとArtifactSetを共有し、subject bindingだけを分けます。

## current BuildPlanとの互換

`projectPluginBuild`は、current `BuildPlan + LockedBuild`を次へ射影します。

```text
current Plugin BuildPlan
  -> BuildExecutionPlan
  -> one-member ArtifactSet
  -> BuildExecutionResult
  -> PluginSubjectBinding
```

current BuildPlanの`OutputKind`は消しません。Referenceではdomain-neutral側の`OutputContract`へ保持し、output semanticsが違うbuildを同じexecution identityへ潰さないようにします。

これはcurrent BuildPlanを新しい型へ移行したことを意味しません。production modelの変更可否は、Referenceのpressureが十分に揃ってから判断します。

## Artifact Set

#1102の最小pressureとして、複数physical outputを一つのcontent setとして扱います。

最初のprofileではroleを一意にします。filesystem pathやURLは`Locator`であり、content identityには含めません。

そのため、同じrole / format / digestを別の場所へ配置してもArtifactSet fingerprintは変わりません。一方、member bytes、role、formatが変わればidentityは変わります。

Target固有のrole enumをCoreへ追加する提案ではありません。

## Authority

このReferenceのfingerprintやbindingは、次のAuthorityを与えません。

- Plugin Acceptance
- Runtime selection
- Capability Provider Authority
- deployment permission
- artifact signature trust

Build executionをlogical subjectから分けて追跡できることだけを確認します。

## 非目標

- current `BuildPlan` / `LockedBuild`の置換
- production Build scheduler
- packaging optimizer
- Artifact Store
- Attestation / signing
- multi-output deployment layout
- filesystem pathをartifact member identityにすること


## Reproducibility Evidence

Artifact Setの二回build比較では、aggregate fingerprintの一致/不一致だけを残しません。

Referenceは次を別々に保持します。

- second buildから欠けたrole
- second buildに増えたrole
- 同じroleでformat / digestが変わったmember

Locatorだけの変更はcontent driftに数えません。

これにより`reproducible=false`でも差分Evidenceを失わず、どのphysical outputが変わったかを追跡できます。

## 非Plugin artifact

#1080のpressureとして、既存`internal/plugin/runtime.RuntimeEngineIdentity`をfake Pluginへ変換せず、`RuntimeEngineSubjectBinding`で同じBuildExecutionPlan / ArtifactSetへ結べることも確認します。

Runtime Engine bindingはselected output roleのdigestと`ImplementationDigest`、Build TargetとEngine Targetをcross-checkします。

このReferenceにより、共通化の単位は「Plugin Build」ではなくbuild execution / physical outputであり、PluginやRuntime Engineは後段のtyped subject bindingとして分けられる候補が得られます。

## read-only Evidence runner

Referenceはfilesystemやnetworkへ書き込まず、固定fixtureから決定論的なJSON Evidenceを出力します。

```bash
go run ./tools/reference-build-execution-binding
```

Evidenceには次を含めます。

- one execution / one Artifact Setを共有する二つのPlugin binding
- multi-output Artifact Set
- missing / extra / changed memberを保持した非再現比較
- fake Pluginを使わないRuntime Engine binding
- authorityを付与しないことを示すlimit

runnerの出力が成立してもproduction Build model採用やAcceptance成立とは扱いません。
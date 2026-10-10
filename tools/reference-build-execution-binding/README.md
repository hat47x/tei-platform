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

ただし、path自体がbuild意味へ影響する場合まで無視してよいわけではありません。Goのpackage/module path、relative embed、生成器のlogical entrypoint等、位置がartifactへ影響し得る情報は、builder adapterがSource closure digest、Recipe、Parameter等の**build input identity**へ含める必要があります。Locatorだけにbuild意味を隠したままidentityから落とすことは許容しません。

SHA-256の16進表記は、現行Build / Runtime contractが大文字・小文字のどちらも受理するため、identity計算時に小文字へ正規化します。同じdigestを文字表記の大小だけで別artifactや別executionとして扱いません。`sha256:` prefix自体は現行contractどおり固定です。

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

## Build-only DAG

#1103のpressureも同じReferenceへ接続します。Runtime Plugin dependencyとは別に、resolved Build execution間のproducer / consumer因果だけを`BuildGraph`で表します。

```text
BuildGraphNode
  node-local ID
  exact BuildExecutionPlan fingerprint

BuildGraphEdge
  producer node
  output role
  consumer node
  exact consumer input binding
    - builder
    - or BuildDependency(kind / id / version)

BuildGraphSeed
  exact external artifact
  external boundary
  consumer node
  exact consumer input binding
    - builder
    - or BuildDependency(kind / id / version)
```

Graph identityはnode宣言順やschedulerの実行順に依存しません。一方、node execution fingerprint、edgeのoutput selector、stage0 seed artifact、trust boundaryを変えればidentityが変わります。

通常のdependency cycleはfail-closedで拒否し、診断にはcycle pathを残します。

```text
a -> b -> c -> a
```

self-hostingはcycle例外にしません。

```text
stage0 exact trusted seed
  -> stage1 node
stage1 builder-artifact
  -> stage2 node
```

stage0はBuild nodeへ偽装せず、`BuildGraphSeed`として明示します。Reference runnerもこのstage0 / stage1 / stage2境界をEvidenceへ出力します。

既存`Definition.Requires`のRuntime Plugin dependencyからBuild edgeを暗黙生成しません。既存Plugin resolverのversion resolution / Runtime semanticsとも統合していません。

### stage0 boundaryとTrust

`BuildGraphSeed.TrustBoundary`は「ここから先が外部seedに依存する」という境界を記録するためのfieldです。Referenceの値は`external-stage0-seed`とし、stage0 seedが信頼済みであること自体は主張しません。

seed issuer、attestation、現在のTrust受理・失効は#1104の責務です。Build graphの存在やstage convergenceだけでstage0 trustを証明しません。

Build graphのedgeは順序だけを表すものではありません。`verifyBuildGraphBindings`はproducerのArtifact Setから指定Roleを選び、そのdigestがconsumerのexact `Builder.ArtifactDigest`またはexact `BuildDependency(kind/id/version).ArtifactDigest`と一致することを確認します。

これにより「graph上は依存しているが、実際のconsumerが別artifactを使った」という偽のprovenanceを拒否します。stage0 seedも同じinput bindingを通し、stage1が実際にそのexact seed builderを使うことをcross-checkします。

## Subject bindingはBuildExecutionResultへ束縛する

`ExecutionPlanFingerprint`と`ArtifactSetFingerprint`を別々に保持するだけでは、別々のbuild runから得たplanとartifact setを後から組み合わせる余地があります。

そのためPlugin / Runtime Engineのsubject bindingは、両fingerprintから構成される`BuildExecutionResultFingerprint`も必須で持ちます。

```text
BuildExecutionPlan
  +
ArtifactSet
  ↓
BuildExecutionResult
  ↓
typed Subject Binding
```

binding verificationでは、指定されたplan / setからResult fingerprintを再計算し、bindingのResult fingerprintと一致することを要求します。

これによりsubjectは「このplan」と「このartifact set」を独立に参照するのではなく、その組合せとして観測されたbuild resultへ束縛されます。

ただし、このReferenceはbuild executionの実行時刻やHost observationまで署名済みEvidenceにしたものではありません。Attestation / Trustは#1104の別責務です。

## Attestation / Trust

#1104の最初のReference pressureも同じread-only toolへ接続します。

```text
BuildExecutionResult
  -> BuildAttestation
       exact result fingerprint
       Artifact Set / member subject
       Issuer
       Signature Profile
  -> BuildAttestationVerification
  -> current Trust Policy
  -> AttestationTrustDecision
```

Referenceの署名profileは`tei.reference.fake-signature/v1`です。これはbinding semanticsを検証するための決定論的fake署名であり、暗号的安全性を一切証明しません。

重要な分離:

- reproducible build != trusted build
- signed attestation != trusted issuer
- immutable attestation != immutable trust
- policy revocationでTrust decisionは変わる
- policy revocationでAttestation fingerprint自体は変わらない

Trust評価はraw Attestationだけでは行いません。先にBuildExecutionResult / Artifact Set / fake signatureを検証して`BuildAttestationVerification`を得て、そのverification Evidenceと一致するAttestationだけをcurrent policyへ渡します。

Artifact Set全体へのattestationと、特定member role/digestへのattestationも区別します。

## Build graphはambient inputを許さない

Build graphを単なる順序図ではなくprovenanceとして扱うため、各Build nodeのexact inputはgraph内で一意に説明できることを要求します。

対象:

- exact `Builder`
- すべての`BuildDependency`

それぞれが、次のどちらか一つへ戻れなければなりません。

```text
producer node output
or
explicit external seed
```

graphに現れないambient builder / SDK / generator artifactは受理しません。

逆に、同じconsumer inputを複数edgeやseed+edgeで二重供給することも拒否します。

## Producer outputもBuildExecutionResultへ束縛する

producer nodeの`BuildExecutionPlan`と`ArtifactSet`を別々に正しいものとして持つだけでは、別runのoutput setを後から組み合わせる余地があります。

そのためgraph内のproducer outputは:

```text
BuildGraphNodeOutput
  ArtifactSet
  BuildExecutionResultFingerprint
```

として保持し、producer nodeのexact planとそのArtifactSetからResult fingerprintを再計算して一致を要求します。

これにより:

```text
producer exact plan
  +
producer exact ArtifactSet
  ->
exact BuildExecutionResult
```

の組合せまでgraph provenanceに含めます。

## self-hosted Builderのsubject binding

self-hostingでは、stage1が出したbytesとstage2のBuilder artifact digestが一致するだけでは十分ではありません。同じbytesを別のlogical Builder identityとして扱う余地が残るためです。

Referenceでは`BuilderSubjectBinding`を追加し:

```text
stage1 BuildExecutionResult
  -> exact BuilderIdentity
  -> stage2 BuildExecutionPlan.Builder
```

を検証します。

一致条件:

- Builder ID
- Version
- ContractVersion
- ArtifactDigest

ArtifactDigestは現行contractどおりhex caseを正規化して比較します。

このbindingもAcceptance / Trust / Runtime Authorityを付与しません。stage0 seedのTrustは引き続き別責務です。

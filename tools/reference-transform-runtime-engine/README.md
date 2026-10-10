# Deterministic Transform Runtime Engine Reference

このディレクトリは、TEI の deterministic transform Source を実行する Runtime Engine の最初の独立 Reference です。

実装言語は Go ですが、Go を Runtime Engine の必須言語とはしません。Runtime Engine の論理的な契約と、個々の実装 artifact の identity は分けて扱います。

## 役割

この Reference は次の境界を実証します。

```text
transform Source
  + typed explicit inputs
  + exact Runtime Engine artifact
  -> deterministic typed output
```

Runtime Engine は Plugin の logical identity ではありません。同じ Plugin / builder Source を異なる Runtime Engine で実行できる可能性を残します。

一方、interpreted execution の再現には Source artifact だけでは足りません。実行時には、どの Runtime Engine artifact を使ったかも固定する必要があります。

## 入出力

Source は `-source` で明示します。

stdin は次の schema を持つ JSON です。

```text
tei.reference.transform-engine-request/v1
```

stdout は次の schema を持つ JSON です。

```text
tei.reference.transform-engine-response/v1
```

Value は次の kind を持ちます。

- `null`
- `bool`
- `number`
- `string`
- `array`
- `object`
- `bytes`

`object` は key-value map ではなく field の順序を保持する列として表現します。これは、既存 Plugin package の byte-level contract と ordered Value semantics を維持するためです。

`bytes` は base64 として明示的に表現します。JSON string との暗黙変換は行いません。

## fail-closed

次は受理しません。

- unknown request field
- unknown nested wire field
- duplicate JSON object key
- duplicate input name
- duplicate ordered-object field
- invalid UTF-8
- invalid base64
- kind と別 kind 用 field の混在
- Source が宣言していない input
- Source の resource limit を超える input
- unsupported transform operation

wire decode の途中から `maxInputNodes`、`maxInputBytes`、`maxCollection`、`maxNestingDepth` を適用します。すべてを Value 化してから初めて超過を判定する構造にはしません。

stdin 全体にも Host 側の有限な envelope 上限を設けます。この上限は wire metadata と base64 の膨張を許容するため semantic input limit より大きく取りますが、無制限にはしません。

## authority

この executable は build、Acceptance、Runtime Authority を自動的には取得しません。

将来の bootstrap runner では、少なくとも次を別々に固定します。

```text
builder Source artifact
Runtime Engine artifact
Runtime Engine contract
Target
Host acceptance
Build / Runtime locks
```

engine 自身が「自分はこの digest である」と自己申告した値を Authority にはしません。trusted build / verification 側で actual executable digest を取得し、expected identity として渡す必要があります。

## 言語中立性

Go は最初の Reference implementation です。

将来、Rust、Wasm、JavaScript、その他の実装を追加しても、logical transform contract を Go の package identity へ結び付けません。

別実装を同等と扱うには、少なくとも次が必要です。

- 同じ Source / typed input に対する conformance
- deterministic output の一致
- resource-bound semantics の一致
- exact implementation artifact identity
- Target / contract version の明示

実装言語が違うこと自体は拒否理由になりません。一方、同じ contract 名を名乗るだけで同等性を仮定もしません。



## build Evidence

exact executable のbuild provenanceは、Plugin用の`BuildPlan`へ無理に載せません。Runtime EngineはPluginではないためです。

Referenceでは次のtoolを使います。

```text
tools/reference-transform-runtime-engine-build
```

このtoolはclean Repositoryから次をEvidenceとして出力します。

- Repository revision
- Runtime Engine logical ID / version / contract
- Target
- exact Go package entrypoint
- actual module-local build Source closure
- Source closure各fileのdigest
- Source closure aggregate digest
- Go version
- Go executable digest
- Go toolchain closure digest
- fixed build settings
- build recipe
- 独立した二回のbuild artifact digest
- reproducibility result
- exact engine artifact digest
- Evidence fingerprint

Source closureは`go list -deps -json`で、そのTarget・build設定に対して実際に選択されたmodule-local compiler inputから作ります。standard libraryはGo toolchain closureへ含めます。外部module dependency、Repository外のmodule、Gitで追跡されていないbuild inputはfail-closedです。

build subprocessはHost環境を広く継承しません。Go executableの選択もPATHへ依存せず、既定ではEvidence tool自身の`runtime.GOROOT()/bin/go`を使い、必要なら`-go`でexact executableを明示します。

buildは別々のGo cache / module cache / GOPATHに加え、HOME / USERPROFILE / TMPDIR / TMP / TEMPもisolated stateへ閉じます。少なくとも次を固定します。

```text
CGO_ENABLED=0
GOENV=off
GOEXPERIMENT=
GOFLAGS=
GOPROXY=off
GOSUMDB=off
GOTOOLCHAIN=local
GOVCS=*:off
GOWORK=off
GOOS=<target OS>
GOARCH=<target arch>
```

Unix系ReferenceではHost environmentを原則継承しません。Windowsではprocess起動上必要となり得る`COMSPEC`、`PATHEXT`、`SYSTEMROOT`、`WINDIR`だけをcase-insensitive allowlistとし、実際に継承した値はBuild Evidenceの`hostEnvironment`へ残します。allowlist外のPATH、locale、proxy、credential等はbuild subprocessへ渡しません。

`-artifact-out`を指定すると、二回のbuildでdigest一致を確認したfirst artifactをRepository外へコピーします。既存fileは上書きしません。これにより、Evidenceでdigestを固定した同じexecutableをprocess smokeや後続のRuntime Engine bindingで利用できます。

二回のbuild digestが一致しなかった場合もEvidence自体は失いません。`firstArtifactDigest`と`secondArtifactDigest`、`reproducible=false`を出力し、canonicalな`artifactDigest`は空のままにします。この場合はartifactをexportせず、CLIもnon-zeroで停止します。

再現できなかった場合もEvidence自体は失いません。一方、そのEvidenceを成功扱いしたり、artifactをAccepted状態へ進めたりはしません。

Repository revisionはRuntime Engineの論理版には使いません。Referenceではlogical Engine Versionを明示的な版として固定し、Repository revisionはbuild provenanceへ残します。再現成功時だけ、logical ID / version / contract / Targetとactual executable digestからexact `RuntimeEngineIdentity`を構成します。docs-only commit等でSource artifactが変わらない場合に、Repository revisionだけを理由としてEngine Versionを変えないためです。

このEvidenceは、将来のdomain-neutral build contractを検討するためのReference pressureです。Plugin identityを捏造せず、現行Plugin `BuildPlan`の意味も変更しません。build成功だけでAcceptance / Runtime Authorityを付与することもありません。

## 現在の非目標

この Reference だけでは次を達成したとは扱いません。

- production Host の loaded machine-code attestation
- Runtime Engine 自身の self-hosting
- Web / Wasm portability
- semantic-equivalence convergence
- stage0 trust の除去
- transform JSON serialization の最終 DSL 化

bootstrap runner への exact executable 接続は、bootstrap stage execution Evidence の整理後に行います。

# Plugin self-build Reference

このReferenceは、ISSUE-000079でPlugin self-buildを検証するための独立Workspaceです。通常の`reference/workspace`へprocess Pluginを追加せず、self-build固有のSource → Build → Artifact → Plugin Lockを分離して検証します。

Plugin descriptorの正本は`workspace/plugins/tei-process-reference/1.0.0/plugin.tei`です。実装SourceはRepository既存の`tools/reference-process-plugin/main.go`を使います。実装SourceをこのReferenceへ複製しません。

Source descriptorでは`artifact`を宣言しません。artifact digestはBuildの出力であり、生成前のauthoritative sourceへ書き戻す前提にはしません。Build後はderived Runtime Definitionへdigestを束縛し、既存Plugin Lockへ渡します。

このReferenceが現時点で主張するのは、artifact未生成のSource Definitionを通常のWorkspace loaderで読み込めることまでです。Go toolchain identity、actual build execution、Build Evidence、bit-for-bit reproducibility、stage0 / stage1 / stage2 convergenceは後続Evidenceです。


## E2 runner候補

候補runnerはRepository rootから次のように実行します。

```bash
go run ./tools/reference-plugin-self-build -repo .
```

runnerは開始時と終了時にGit worktreeがcleanであることを要求します。Plugin artifactはRepository外の一時directoryへ出力し、Repositoryへ生成物を残しません。

最初のE2ではSource closureをtracked Repository全体として扱います。また、Go toolchainの実行ファイルだけでなく、compiler / linker /標準ライブラリSourceを含むtoolchain closureをfingerprintします。対象packageに外部Go module依存が見つかった場合は、module cacheを暗黙の入力にせず停止します。

同じSource / builder条件で、別々のbuild cacheを使って2回buildし、SHA-256 artifact digestが一致した場合だけ`reproducible=true`を出力します。その後、Build Plan、BuildArtifactLock、derived Runtime Definition、既存Plugin Lockを相互照合します。

この説明はrunnerの候補実装を示すものであり、actual Evidenceの成立を示すものではありません。exact-headのtrusted checkoutでrunnerとRepository Health Checkが成功するまでSB-2完了とは扱いません。


## SB-1 package self-build

同じrunnerで、通常のReference Workspaceにある`tei.runtime.reference@1.0.0`から決定論的なPlugin packageも生成します。

このpackageはdescriptorだけを対象とする最初のSB-1 profileです。Source DefinitionにRuntime artifact digestが含まれている場合は受け付けません。生成物の種類は`plugin-package`として記録し、`runtime-artifact`とは別のBuildArtifactLock entryにします。

Build Planにはbuilderだけでなく、`tei-plugin-package`というrecipe kindと対象descriptorのentrypointも記録します。そのため、同じSource closureとbuilderを使って別のPluginをpackageした場合でも、Build Plan fingerprintから区別できます。

SB-1 packageの生成に成功しても、Pluginの実行実装をTEI自身で生成できたことにはなりません。process Pluginのruntime artifactを2回buildしてdigest一致を確認するE2とは、別の到達段階として扱います。

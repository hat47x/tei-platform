# Plugin self-build Reference

このReferenceは、ISSUE-000079でPlugin self-buildを検証するための独立Workspaceです。通常の`reference/workspace`へprocess Pluginを追加せず、self-build固有のSource → Build → Artifact → Plugin Lockを分離して検証します。

Plugin descriptorの正本は`workspace/plugins/tei-process-reference/1.0.0/plugin.tei`です。実装SourceはRepository既存の`tools/reference-process-plugin/main.go`を使います。実装SourceをこのReferenceへ複製しません。

Source descriptorでは`artifact`を宣言しません。artifact digestはBuildの出力であり、生成前のauthoritative sourceへ書き戻す前提にはしません。Build後はderived Runtime Definitionへdigestを束縛し、既存Plugin Lockへ渡します。

このReferenceが現時点で主張するのは、artifact未生成のSource Definitionを通常のWorkspace loaderで読み込めることまでです。Go toolchain identity、actual build execution、Build Evidence、bit-for-bit reproducibility、stage0 / stage1 / stage2 convergenceは後続Evidenceです。

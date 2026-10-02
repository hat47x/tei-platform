# TEIの命名とWorkspace identity

この文書は、TEIで使う名称と、Workspace・Semantic ID・物理ファイルを混同しないための規則を整理します。名称そのものの変更ではなく、現在採用している命名方針の説明です。

## Product name

現在のプラットフォーム名は TEI（鼎／テイ） です。

`TEI`を一つの英語展開へ固定しません。次の3つは競合する正式名称ではなく、同じ設計を異なる面から説明する補助的な読みです。

- **Traceable Execution of Intent** — 意図がSemantic Definition、Realization、Execution、Observation、Evidence、Recoveryまで追跡できること。
- **Target-neutral Executable Infrastructure** — Semantic / Application Definitionが、特定の実装言語、Database、UI技術、Deployment target、Executorを意味上の特権として持たないこと。
- **Three-Element Integration** — 異なる層を一つのsource of truthへ潰さずに接続すること。たとえばsemantic / realization / execution、logical / physical / observed、application / database / environmentなどです。

通常のUIや文書で、TEIを毎回これらの英語へ展開して表記する必要はありません。

## Canonical technical names

技術上の標準名称は次のとおりです。

- monorepo: `tei`
- CLI: `tei`
- visible workspace declaration: `tei.workspace.yaml`
- TEI-native DSL extension: `.tei`
- standard semantic namespace: `std.*`
- generated artifacts: `.tei-generated/`
- derived metadata / index / cache / tool state: `.tei/`
- Go module used by this architecture probe: `tei`

業務上のCanonical IDには`tei.` prefixを付けません。たとえば`sales.order`は`sales.order`のままであり、`tei.sales.order`にはしません。

## Workspace identityは拡張子だけで決めない

TEI Workspaceは、`tei.workspace.yaml`と、そこで管理対象として宣言されたsource root / directoryによって識別します。

`.tei`はTEI-native DSLの拡張子ですが、「そのdirectoryがTEI Workspaceである」ことを示す唯一の印ではありません。

物理ファイルは、その用途に自然な形式を保ちます。`.sql`、`.go`、`.json`、`.yaml`、`.md`等をTEI管理下へ置いても、それらを無理にTEI DSLへ変換しません。Editorは管理対象resourceとして列挙できますが、すべてのファイルを同じparserへ押し込めません。

この方針では、次の3つを明確に分けます。

- **workspace / unit identity** — directory + manifest / context
- **semantic identity** — `sales.order`等のCanonical ID
- **physical representation** — `.tei`、`.sql`、`.go`、`.json`、`.yaml`、`.md`等

現在のcompilerは、TEI-native DSL semanticsが必要な`.tei`をparseします。他のmanaged fileはnative fileとして保持・列挙し、必要なProviderやEditor Capabilityが扱います。

## 現在使っているProduct boundary名

次の名称は、場当たり的に`TEI` prefixを付け直さず、アーキテクチャ上の役割を表す名称として維持します。

- **Editor Intelligence** — 切り離し可能なanalysis / intelligence境界。SQL analysisでは現在Python + SQLGlotを使います。
- **Database Workbench** — schema / query / lineage / planを扱うderived editor projection。Canonical source of truthにはしません。
- **DatabaseProvider / CatalogReader** — driver-neutralなDatabase observation境界。
- **Migration Provider / HistoryReader** — 外部migration toolとの統合境界。
- Execution Plan IR, Physical Schema Snapshot, Derived Query IR, Persistence Mapping — 明示的なmodel / artifact名称。
- **Deployment Executor SPI** — provider-neutralなcommand境界。
- **Observation Provider** — provider-neutralなlive-state observation境界。
- **Evidence Provider** — rollout / decisionに必要なevidenceの境界。
- **Resource Broker** — logical Resource IDをinvocation-scoped secret handle、network endpoint、mount等へ解決するHost境界。
- **Sandbox Profile** — 1回のprocess invocationに必要なleast-authority enforcement request。Profileが存在するだけで実際のenforcement済みとはみなしません。
- **Runtime Policy fingerprint** — Plugin Lock identityとは別のexecution-policy identity。Durable Runが利用するCapability PermissionとResource Bindingを固定します。

## Historical references

X-TEA / XEAD は現在の製品・プラットフォーム名ではありません。

TEIはX-TEA / XEADの公式な後継・派生プロジェクトを名乗りません。公開資料や参照モデルから得た設計上の示唆を説明するとき、またはprovenanceを保つ必要がある既存artifactを指すときに、独立した参照対象として歴史的名称を用います。

新しい実装symbol、Repository path、CLI command、generated artifact、Canonical DefinitionではTEIの名称を使います。

## Naming rule

概念に明確なdomain / architecture上の役割名がある場合は、その名称を優先します。`Database Workbench`、`Execution Plan IR`、`Migration Provider`のような語へ、機械的にすべて`TEI`を付ける必要はありません。

`TEI` prefixは、プラットフォーム、CLI / module / repository boundaryを示す場合や、他システムとの区別が必要な場合に使います。

## PluginとBundleの命名

TEIが配布するPlugin package IDにはTEI package namespaceを使います。例として`tei.migration.atlas`、`tei.database.postgres`があります。

Capability IDは`migration.atlas`、`database.postgres`、`runtime.reference`のように役割を表す名称のままにします。Bindingが指すのはpackageそのものではなくCapabilityだからです。

Bundle IDは、`tei.reference.providers`のように、まとめて配布するpackage setを識別します。

業務上のCanonical IDには、ここでも`tei.` prefixを付けません。

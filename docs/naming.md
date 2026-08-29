# TEI naming and workspace identity

## Product name

The current platform name is **TEI（鼎／テイ）**.

`TEI` is deliberately not constrained to one mandatory English expansion. Three complementary readings are retained because they describe different architectural aspects of the same platform:

- **Traceable Execution of Intent** — intent remains traceable through semantic definition, realization, execution, observation, evidence, and recovery.
- **Target-neutral Executable Infrastructure** — semantic/application definitions are not semantically privileged toward one implementation language, database, UI technology, deployment target, or executor.
- **Three-Element Integration** — TEI repeatedly connects distinct layers without collapsing them into one source of truth; examples include semantic / realization / execution, logical / physical / observed, and application / database / environment.

These are explanatory readings, not competing product names and not a requirement to spell out TEI in normal UI copy.

## Canonical technical names

- monorepo: `tei`
- CLI: `tei`
- visible workspace declaration: `tei.workspace.yaml`
- TEI-native DSL extension: `.tei`
- standard semantic namespace: `std.*`
- generated artifacts: `.tei-generated/`
- derived metadata/index/cache/tool state: `.tei/`
- Go module used by this architecture probe: `tei`

Business canonical IDs do not receive a `tei.` prefix. For example `sales.order` remains `sales.order`, not `tei.sales.order`.

## Workspace identity is not a file-extension rule

A TEI workspace is identified by `tei.workspace.yaml` and its managed source roots/directories. `.tei` remains the native DSL extension, but it is not the marker that makes a directory or project a TEI workspace.

Files keep the format that best matches their physical purpose. Native `.sql`, `.go`, `.json`, `.yaml`, `.md`, and other files may live inside TEI-managed directories and remain native files. The Editor can enumerate them as managed workspace resources without pretending that every file is TEI DSL.

This deliberately separates:

- **workspace/unit identity** — directory + manifest/context,
- **semantic identity** — Canonical IDs such as `sales.order`,
- **physical representation** — `.tei`, `.sql`, `.go`, `.json`, `.yaml`, `.md`, etc.

The current compiler parses `.tei` where TEI-native DSL semantics are required. Other managed files are preserved/enumerated and may be consumed by the relevant provider or editor capability rather than by the TEI DSL parser.

## Product boundaries already in use

The following names are retained as architectural boundary names rather than being replaced by ad-hoc TEI-prefixed variants:

- **Editor Intelligence** — detachable analysis/intelligence boundary; SQL analysis currently uses Python + SQLGlot.
- **Database Workbench** — derived editor projection for schema/query/lineage/plan work; never a canonical source of truth.
- **DatabaseProvider / CatalogReader** — driver-neutral database observation boundary.
- **Migration Provider / HistoryReader** — external migration-tool integration boundary.
- **Execution Plan IR**, **Physical Schema Snapshot**, **Derived Query IR**, **Persistence Mapping** — explicit model/artifact names.
- **Deployment Executor SPI** — provider-neutral command boundary.
- **Observation Provider** — provider-neutral live-state observation boundary.
- **Evidence Provider** — provider-neutral rollout/decision evidence boundary.
- **Resource Broker** — host boundary that resolves logical Resource IDs into invocation-scoped secret handles, network endpoints, and mounts.
- **Sandbox Profile** — derived least-authority enforcement request for one process invocation; not a claim that enforcement already occurred.
- **Runtime Policy fingerprint** — execution-policy identity distinct from Plugin Lock identity; pins capability permissions and Resource Bindings for a Durable Run.

## Historical name

**X-TEA** is not the current product/platform name. It may appear only when referring to the historical/reference model, design lineage, or old artifacts where preserving provenance is useful. New implementation symbols, repository paths, CLI commands, generated artifacts, and canonical definitions use TEI naming.

## Naming rule

Prefer a domain/architectural name when the concept already has a clear role (`Database Workbench`, `Execution Plan IR`, `Migration Provider`). Prefixing every concept with `TEI` is unnecessary. Use `TEI` when naming the platform, CLI/module/repository boundary, or when disambiguation from another system is useful.

## Plugin and Bundle naming

Plugin package IDs use the TEI package namespace when they are TEI-distributed packages (for example `tei.migration.atlas` or `tei.database.postgres`). Capability IDs remain role-oriented (`migration.atlas`, `database.postgres`, `runtime.reference`) because bindings target a capability, not a package. Bundle IDs identify curated package sets such as `tei.reference.providers`. Business canonical IDs continue to avoid the `tei.` prefix.

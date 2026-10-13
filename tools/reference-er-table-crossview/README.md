# Logical ER × Physical Table Crossview（#1126）

This reference projection reads one TEI Workspace's structured World and emits two **read-only, cross-referenced** views in one artifact. It is not a modeling editor, shared-acceptance decision, live database catalog, or migration planner.

From repository root:

```sh
go run ./tools/reference-er-table-crossview/cmd/tei-er-crossview \
  -workspace reference/workspace \
  -out /tmp/tei-er-crossview.json
```

The file must not already exist. It is created with permissions `0600`. No TEI source, database, or acceptance state is modified.

## What the projection includes

- `data`: Semantic Data / Attribute IDs, declared types, explicit candidate keys and functional dependencies, and the IDs of explicitly mapped physical tables. This is the source for a logical ER table/field view; it does not infer requiredness or physical primary keys.
- `relations`: Source and target Semantic Data, declared cardinality, and **author-specified** compound RelationKey pairs. This is the source for ER relationship edges; it never manufactures FK constraints.
- `tables`: Database-owned persistence Mapping ID, explicit physical table and column names, nullability/storage types, physical PK, UNIQUE and FK declarations. Multiple physical Mappings may point to one Semantic Data; an unmapped Data remains visible.
- `keyComparisons`: Each **declared** logical candidate key versus each **declared** PK/UNIQUE in the same explicitly associated Mapping. `same-attribute-set` means only that the sets of Semantic Attribute IDs match. It does not assert business uniqueness, physical constraint compatibility, PK choice, null treatment, provider behavior, or an accepted mapping equivalence. `distinct-attribute-set` deliberately keeps a business key separate from a surrogate physical PK.
- `unresolved`: Missing physical Mappings, out-of-snapshot Relation target, Mapping to unknown Semantic Data, column attributes outside the logical Data, and PK attributes without a declared column. These findings are not automatically corrected.

The `structureRevision` is a SHA-256 hash of serialized Semantic / Database / Persistence structures in this World. It does **not** authenticate the physical schema of a running database, the creator or approver of the business rules, or the bytes of Workspace source files. Neither `candidateKeys` nor `functionalDependencies` imply confirmed business truth merely because they were authored in a canonical Semantic source.

## Synthetic regression case

The tests include an OrderLine entity whose **logical business candidate key is** `(order_number, line_number)`, while its **physical primary key is** `row_uuid`; a separately authored UNIQUE constraint happens to use the business attribute pair. The read-only comparison shows distinct PK attributes and matching UNIQUE attributes independently. No DDL or new rule is generated. Tests also cover missing Mappings, multiple mappings for one Data, a Semantic Relation without a physical FK, an invalid mapping reference, duplicate identifiers and input immutability.

This is a downstream projection over the existing World model; no new Core schema or permanent UI component type is added. UI layout, bidirectional editing, rule update/removal, reviewed Logical→Physical mapping choices, database introspection, and runtime verification remain outside this reference. The type distinction and cross-reference are designed to be usable by a standard ER/table UI plugin later.

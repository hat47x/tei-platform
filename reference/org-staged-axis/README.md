# org-staged-axis

WU-02 以降の型付き複数軸・語彙分離・役職区別・二時点・as-of Query・遡及訂正の受入fixture群。WU-01 は `reference/org-staged-basic` を参照。

## WU-09 承認候補の動的解決と実行可能性の分離

このfixtureは「承認候補」と「承認可能性」を分離する境界を固定するためのものである。

- `organization.role_assignment`・`organization.responsibility` は構造Relationにすぎない。
- これらから `Capability`/`EffectRule`/`RunPlan` の存在は導出されない。
- 候補解決は read-only の導出であり、通知・割当て・認証・現行権限・職務分掌・対象revisionは別資格として扱う。

## WU-14 保護対象の区別と例外管理

通常の組織Dataと保護対象の特定情報を区別し、一律の制御を課さない。

- 部門・職位・所属・組織図・配置表・人数は特別な機密扱いにならない。
- 保護対象が混在する場合のみ項目・Provider・運用境界で個別管理する。
- 組織図や人数を標準で秘匿しない。
- 外部Host/別Tenantの実行認可を通常Data閲覧と混同しない。

新ACLレイヤ・OS隔離・匿名化は作成しない。

## as-of Query の受入境界（2026-11-13）

`org_axes_view.as_of_membership` は、既存DSLが受け付けるPostgreSQLの**候補SQL**であり、製品の二時点Queryを実行して正しさを確認した証拠ではない。

- 物理参照先をPersistence定義の `public.org_memberships` に統一し、引数を `valid_at`（業務有効時点）と `known_at`（把握時点）に区別した。
- 各時点に独立した半開区間 `[from, to)` を仮定し、`from <= at AND (to IS NULL OR at < to)` で絞る。旧例の `known_from >= :known_from` は比較方向が逆であり撤回した。NULLはこの参照例では終了期限なしの意味とするが、全業務ドメインの標準規則にはしない。
- `valid_to`・`known_to` のPersistence Mappingを追加した。ただし**物理テーブルの実DDL・履歴行の実在は未確認**。この例の主キーは `membership_id` 単独で、同一所属occurrenceの複数把握版をそのまま同一テーブルに共存させる設計にはなっていない。時点付き複数版の永続化を完了したとは扱わない。
- `org-staged-axis` のGoテストはLoader、入力名、SQL文字列形状を点検する**構造検査**である。実DB接続、Snapshot/Cohort完全性、遡及訂正差分、NULL/partial/permission-denied、Query Provider、HostのGrant/Effectは検証しない。実Query経路の検証は #2586/#2506/#2622、組織振る舞いの独立Expectedは #4490 に帰属する。
- 二時点を必要としない簡易的な組織モデルへ、履歴保持テーブルを強制しない。履歴保持が要る場合は、行版識別子/複合キーまたは履歴ストアの方式、上流情報の観測時点と上流が主張する既知時点との差、Source/Providerの取得完全性を別途設計・受理する。

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

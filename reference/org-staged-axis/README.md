# org-staged-axis

WU-02 以降の型付き複数軸・語彙分離・役職区別・二時点・as-of Query・遡及訂正の受入fixture群。WU-01 は `reference/org-staged-basic` を参照。

## WU-09 承認候補の動的解決と実行可能性の分離

このfixtureは「承認候補」と「承認可能性」を分離する境界を固定するためのものである。

- `organization.role_assignment`・`organization.responsibility` は構造Relationにすぎない。
- これらから `Capability`/`EffectRule`/`RunPlan` の存在は導出されない。
- 候補解決は read-only の導出であり、通知・割当て・認証・現行権限・職務分掌・対象revisionは別資格として扱う。

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

## 二時点Query：実装可能な範囲と未成立の履歴モデル（2026-11-13）

`applications/org-axes-view/queries/as_of_membership.tei` のSourceと
`databases/org-store/persistence/membership.tei` を照合し、Queryの参照テーブルを実Mapping先 `public.org_memberships` に揃えた。
`valid_from <= validAt < valid_to`、`known_from <= knownAt < known_to`（上端NULLの場合はこのfixtureでのみ無期限と仮定）という**半開区間の試験条件**を表現し、従来の `known_from >= knownAt` と終了時刻条件なしを修正した。将来の全TEIドメインへ半開区間を強制する仕様ではない。

- `go test ./internal/operations -run TestOrgAxisAsOfQueryPhysicalBinding -count=1`：実TEI Loader → Native Query → PostgreSQL用named parameter束縛 → 検査用SQL connection → Semantic Result Mappingの経路を確認。これは実DBのWHERE評価ではない。
- `TEI_ORG_AXIS_PG_DSN` と `TEI_ORG_AXIS_PG_CONFIRM_ISOLATED=tei_org_axis_e2e` を明示した場合に限り、`go test ./internal/operations -run TestOrgAxisAsOfActualPostgresOptIn -count=1 -v`：**指定DB名 `tei_org_axis_e2e` を再検証し、既存 `public.org_memberships` が存在すれば拒否**。独立に手書きした結果集合を、実PostgreSQL＋Native Queryで比較する。通常のCIでは環境変数未設定ならSKIP。
- `allocation_min` による抽出、valid開始・終了境界、known開始・終了境界を別の物理行で検証する。NULL/部分取得からの「不存在」「完全性」の認定は行わない。

**重要な未解決事項**：現行Persistenceの物理PKは `membership_id` 単独。既存 `org-synthetic-r7` のA2のように、**同一Membership occurrenceの訂正前後がそれぞれknown期間を持つ履歴**を単一の実テーブルへ共存させる契約にはなっていない。今回の実DB試験は別々のMembership IDの有効期間を照合する限定試験であり、遡及訂正の保存・再読込や多版同一性は未受入。物理版キー／History保存／Sourceへの訂正適用と、Sourceのknown時点・TEIの観測時点は、#2386/#2510/#4490とDB/Provider Ownerの判断対象。無断の複合PK変更や新しいTemporal Engineは設けない。

さらに、受取範囲の完全性、Provider/Scope/Record Policy、Function Data Context、Caseの移管、Grant/Effect/ACKはこのQueryテストでは検証できない。SQLが0件を返しても「権限のある完全範囲に対象が存在しない」とは限らない。

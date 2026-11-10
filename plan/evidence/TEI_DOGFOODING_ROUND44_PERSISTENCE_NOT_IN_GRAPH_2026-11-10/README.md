# 永続化マッピングの列対応が参照グラフに現れない（2026-11-10、第44ラウンド）

- 対象: [#4001](https://github.com/hat47x/tei-platform/issues/4001)、[#3983](https://github.com/hat47x/tei-platform/issues/3983)（同じ型。実装束縛）、[ADR-0048](../../adr/0048-shared-application-api-and-transport-adapters.md)。
- 実行日: 2026-11-10（Asia/Tokyo）。
- 状態: **read-only の実測。新しい欠陥を検出して起票。コードは変更しない。**

## 1. 目的

「この属性はどの列に格納されるか」を CLI だけで答えられるかを試した。永続化は ADR-0048 の Binding/Realization 層に属し、設計者が参照する情報である。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `0d56e44f5`（`origin/main`） |
| CLI実行物 | `sha256:647dc2aaa875a92258769fd8fe827ea072197eb59f3555d249fc9a7c5229985e` |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.27.3 linux/amd64`（gofmt は 1.26.10） |

## 3. 先に確認した正常な挙動

`commands` と `-h` の一致（第38ラウンドで私が指摘した点）は、他者の修正で**解消済み**だった。

| コマンド | `-h` | `commands` | 差分 |
| --- | --- | --- | --- |
| `validate` | 3 | 3 | なし |
| `world` | 2 | 2 | なし |
| `inspect` | 3 | 3 | なし |
| `trace` | 9 | 9 | なし |
| `plugin-resolve` | 5 | 5 | なし |
| `view` | 15 | 15 | なし |

第42ラウンドの `realizes` 辺、第28ラウンドの `outcome` 辺も健在だった（`security.authenticate` の total=5、`authorize` の total=6、incoming に `order_entry - realizes`）。

## 4. 観測: 列対応がグラフに無い

### 4.1 宣言は実在する（21件）

```tei
// databases/sales-core/persistence/order.tei:10
column sales.order.total -> total_amount storage decimal required
```

```tei
// applications/order-entry/queries/recent_orders.tei:9
column total_amount -> sales.order.total
```

`reference/workspace` 全体で `databases/` 配下に16マッピング・21列対応、`applications/` 配下に5件。

### 4.2 trace の incoming は 0 件

```bash
tei trace -scope s -subject-kind semantic-attribute \
  -subject-id sales.order.total -direction incoming -mode full
# total: 0
```

`sales.order.total` は上記2箇所から参照されているのに、**incoming は0件**。

### 4.3 属性20件のうち incoming を持つのは3件だけ

```text
attributes: 20
with incoming edges: 3
```

その3件は `semantic-validation -attribute->` の辺を持つものだけ（`internal/index/index.go:303` が作る唯一の属性向き辺）。**残り17件は参照されていても incoming を持たない。**

### 4.4 `persistence` は subject-kind に無い

```text
trace -subject-kind: semantic, semantic-type, semantic-data, semantic-attribute,
                     semantic-relation, semantic-validation, semantic-capability,
                     application, application-pattern, query
```

```bash
tei trace -subject-kind persistence -subject-id sales_core.order
# UNSUPPORTED_SUBJECT_KIND
```

`inspect` の `InspectionSubjectKinds` にも無い。**subject としても辿れない。**

## 5. 原因（実測）

World には列対応がある。

```go
// internal/database/persistence/model.go:40
type Mapping struct {
	ID           core.CanonicalID
	Database     core.CanonicalID
	SemanticData core.CanonicalID
	Table        string
	Columns      []ColumnMapping   // ← 列対応
	...
}

type ColumnMapping struct {
	Attribute core.CanonicalID   // どの属性か
	Column    string             // どの列か
	Storage   StorageType
	Nullability Nullability
}
```

索引は列対応を**捨てている**。

```go
// internal/index/index.go:405
for _, m := range w.Persistence {
	d.PersistenceMappings = append(d.PersistenceMappings, PersistenceIndexEntry{
		Mapping: m.ID, Database: m.Database, SemanticData: m.SemanticData, Table: m.Table})
	// m.Columns を写していない
}

// internal/index/index.go:153
type PersistenceIndexEntry struct {
	Mapping      core.CanonicalID
	Database     core.CanonicalID
	SemanticData core.CanonicalID
	Table        string
	// 列対応の欄が無い
}
```

さらに、**いずれの依存リストにも1本も写していない**（`DeclaredDependencies`・`SemanticDependencies` とも）。

## 6. 影響

- **「この属性はどの列か」「この列はどの属性か」を CLI で答えられない。** 私は `grep` で見つけるまで、`sales.order.total` が永続化されていることに気づけなかった。
- `trace` で属性の incoming を見て「参照されていない」と誤読する。実際には17件が参照されている。**前ラウンドで起票した「未結合」の誤読と同じ型。**
- `#3983`（実装束縛）と同じ「索引またはモデルにあるが参照グラフに無い」が、層をまたいで残っている。

## 7. 起票した案

| 案 | 内容 | 判断が要る点 |
| --- | --- | --- |
| **A** | `persistence` を subject-kind に加え、`persistence -stores-> semantic-data`・`persistence -column-> semantic-attribute` の辺を作る | 索引に列対応を通す変更が必要 |
| B | `semantic-data -persisted-as-> database` の辺だけ作る | 小さく済むが「この属性はどの列か」には答えられない |
| C | 現状を `AuthorityLimits` に明記する | 実装変更なし。設計質問には答えられない |

**A が本筋**だが、`ir.World` → 索引 → 投影の3段に列対応を通す変更が要る。

## 8. 除外の意図は見つからなかった

ADR-0048 の層分離では、永続化は Binding/Realization 層に属する。`#3983` で実装束縛を辺にしたのは、それが「どの application がどの capability を実装するか」という設計上の問いに直結するからだった。永続化も同様に「どの属性がどの列か」という設計上の問いに直結する。

**層分離を理由に意図的に除外されている可能性はあるが、その根拠を文書で見つけられなかった。** 判断は Owner に委ねる。

## 9. 読み取れないこと（NOT RUN）

- 列対応を辺にすることの設計判断（案 A/B/C の採否）。
- `RelationStorageMapping`・`PrimaryKey`・`UniqueConstraints`・`Indexes`・`ForeignKeys`（`Mapping` の他の欄）も同様に辺にすべきか。
- 実DB・実Provider・実Host、HTTP・Desktop、意味の受理・権限・配備。

## 10. 位置付け

**同じ型の4例目**である。

| ラウンド | 見えなかったもの |
| --- | --- |
| 28 | `capability.outcome` |
| 41→42 | `realize` の実装束縛 |
| 43 | 他の3種の束縛 |
| 44 | 永続化の列対応 |

「モデルにあるが参照グラフに無い」は、層ごとに繰り返し現れる。**今回は `ir.World` の欄ではなく、索引が捨てている中間段（`Mapping.Columns`）にあった**点が新しい。前回は「索引が読んでいない」だったが、今回は「索引は読んでいるが写していない」。

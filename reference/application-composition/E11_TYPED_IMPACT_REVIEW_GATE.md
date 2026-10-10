# E11：Source出典変更と型付き参照Impactを別責務のまま束ねる

- **状態**：限定Reference。正式なApplication Composition Profile、型付き依存グラフ、実行可否判定、Provider/Business Acceptance/Runtime Authorityではない。
- **管理**：[Issue #1016](https://github.com/hat47x/tei-platform/issues/1016)／[Draft PR #1051](https://github.com/hat47x/tei-platform/pull/1051)。採用判断は[E10](E10_COMPOSITION_ADOPTION_DECISION.md)。
- **接続の相手（別Draft）**：[Issue #1120 / PR #1139](https://github.com/hat47x/tei-platform/pull/1139) の`modelref`と`derivedbridge`。これらのAPIをTEI製品へ移植したという意味ではない。

## 過去の検査では足りない理由

E7の`limited-unchanged`は、E5で列挙したApplicationの直接定義Sourceだけについての変更検査である。**Relation、Capability、業務意味の変更がPatternを経由してApplicationへ与える影響**を、現行E7が完全に見つけられるわけではない。

一方、#1120の`modelref.Impact`は、既存Derived Index等から投影した型付き参照Graphを探索し、変更の影響を受けるConsumerの**候補**を返す。しかし、それ単独ではSourceの出典指紋、採用済み意味、Runtime Permissionを証明しない。

**E11ではSource変更診断と型付きImpact結果を入力として受け取り、限定的な再レビュー対象をまとめる。** 新しい意味Graph、Provider選択、Source Index、Plugin Lock、別Resolverは一切作らない。

## 入出力契約（ReferenceのAdapter形状）

[読み取り専用の接着関数](e11-review-gate.mjs)は、下記の既存の観測結果を、呼出し元が指定したApplicationとscopeについて再確認する。

- `sourceDrift`：E6/E7の`assessParsedApplicationSourceDrift`または`assessPairedEvidence`の結果。限定Source集合の`limited-unchanged` / `unresolved`しか許可しない。`authority:none` / `fullClosure:false` と既知Qualifierを必須とする。
- `typedProjection`：#1120の`derivedbridge.Projection.Graph`の`schema`、`scope`、`nodes`、`Projection.Unresolved`、`modelref.Validate`の`diagnostics`、`modelref.Impact`の`affected`を**外部からまとめて渡す暫定的なAdapter形状**。このReference自身はGo APIを実行しない。
- `application`・`scope`：選択したConsumerを明示する。GraphのApplicationがない、別scopeが混入、Nodeが重複、Impact先がGraphに無い場合は保留。選択した`application`または現行命名で`application.*`に属するPatternだけを局所Review対象へ含める。

出力は`limited-unchanged`、`review-candidate`、`unresolved`のどれかで、**どの状態でも`fullClosure:false`／`authority:none`／`adoption:not-decided`を保つ**。したがって、`review-candidate`は変更影響の**確認対象**を列挙するもので、変更の適用・実行・デプロイの許可ではない。E7が未解決なら、型付きGraphが一見正常でも`unresolved`を維持する。

### 実装の責務境界

- Source Indexの指紋・RoleとSource変更検査：E4 / E6 / E7に委譲。
- Applicationが何を直接参照しているか：E5の型付きWorld、既存解析器が所有。
- 型付きGraphの依存辺・scope・Version・差分・Impact：**#1120 `modelref`が所有**。本Referenceでは`Graph.Validate`や`Impact`を複製しない。
- SUI Native Canvasの最終Intent／保存・CAS／Tenant/Permission：#993およびSUI正本が所有。
- 業務仕様の採用、実接続Provider・Grant：Shared Acceptance、Capability/Runtime/Environmentが所有。

## 負例と局所試験

```sh
node --test reference/application-composition/e11-review-gate.test.mjs
```

2026-10-10、Node.js **22.16.0の実標準テストランナー**で12件成功／失敗0。テストした二つのローカルファイルのGit blob SHAと、GitHubブランチ上に再取得したGit blob SHAが一致：

- `e11-review-gate.mjs` = `06f7263bde425a4dff3b1464ae5fa0c0d03377c0`
- `e11-review-gate.test.mjs` = `b1211e338bf1e77428620cb28cf0172b5ff0690f`

ケース：Relation→PatternとRelation→Applicationの局所レビュー、別Applicationへの影響の排除、Source出典変更による未解決維持、型付き未解決／Validate診断、Application欠落、scope不一致、Node重複、Graphに無いImpact、重複Impact、Authority偽装、未指定入力。

**試験は合成の入力**であり、実WorldのE5→E7出力や#1120のGo `derivedbridge.FromIndex`／`Impact`出力を使った結合ではない。Graphの`unresolved:[]`や`diagnostics:[]`は、入力形状が正しいだけではその結果の真正性を保証しない。実際の`modelref`の結果と、E7のSource出典が**同じ解析済みWorkspace snapshot**であることも、本関数では証明できない。

### 次の受入条件

1. #1120側のGo `Projection`・`Validate`・`Impact`と#1016側のE7の実出力を同一解析snapshotで取得し、Adapterを生成する正式なCallerを検討。Go/JSの第二意味解決は作らない。
2. `order_entry.customer_reference → sales.order.customer → customer.customer`で、Relationだけが変更されたときにPattern／ApplicationへのReview候補が現れることを**実Sourceで確認**する。
3. 複数Application、複数scope、別aliasの同一Database Contract、未知Edge、親Kitの更新、SUIの外部Repository出典を負例にする。現行のPattern名prefixによる局所分類は**暫定の命名Convention**であり、恒久の所属関係の正本ではない。型付きApplication所有参照が実装側から供給されるまでは完全な対象抽出に昇格しない。
4. [#1017 / PR #1111](https://github.com/hat47x/tei-platform/pull/1111)のLoader解析時bytes統一、#1120 Source BridgeのScope不一致の扱いを確認する。
5. exact-headのtrusted clean full checkoutでGoテスト、Nodeテスト、実WorkspaceのE5→E7→typed Impact、最後に`tools/repository-health-check.sh`全項目PASSを得る。それまではDraftを維持し、mainへ統合しない。

**結論**：今回必要だったのは「構成用Graphの新設」ではなく、**すでに担当を持つ二つの診断結果を、権限を増やさずに一つのレビュー導線へ渡す最小の接着部**である。正式Composition Profile自体の要否はE10の二Consumerの実測後に判断する。

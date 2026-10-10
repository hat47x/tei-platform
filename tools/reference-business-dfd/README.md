# Business DFD read-only Reference (#1125)

このツールは、既存の Business Procedure / Handoff Reference と正規化済みの Application Pattern を参照し、**一階層の業務図を描くための派生情報**を JSON / Graphviz DOT で返します。Canonical World や手順の正本は書き換えません。実装形態は独立した薄い Projection adapter の実験であり、TEI の標準 Plugin として登録されたことを意味しません。

## 実行

リポジトリのルートから実行します。

```sh
go run ./tools/reference-business-dfd -format json
go run ./tools/reference-business-dfd -format dot > /tmp/order-entry-review.dot
go test ./tools/reference-business-dfd
```

DOT の描画には任意で Graphviz を使えます。ツール自体は Graphviz に依存しません。DOT の node 配置・装飾は一時的な表示情報で、Procedure の意味や step の同一性を構成しません。

## Source / authority の対応

| 入力 | 意味 | Projection 側の扱い |
| --- | --- | --- |
| `order-entry-business-procedure.json` | 非 Canonical な Reference authoring fixture | `procedure.id` と `steps[].id` を exact identity として参照。actor / trigger は候補のまま保持 |
| `order-entry-procedure-reference-set.json` | 独立した Handoff Reference | `from` / `to` / `condition` のみ投影。手順の自動実行は含意しない |
| `order-entry-dfd-review-proposal.json` | **未受理の図レビュー案** | human / event / schedule の契機候補、順序・分岐・例外の矢印。必ず `authority=proposal` |
| `reference/workspace` | 現行 Workspace の正規化 Application Pattern | Step の明示 Pattern ID だけを exact resolution し、Data / Relation / Capability を派生表示 |

**「手順ファイル内の Step の並び」と「業務の実際の遷移」は別物です。** 遷移の Source が無いとき、配列順から矢印を自動生成しません。提案経路は DOT では破線として表示します。3種類の契機も runtime trigger / schedule に束縛しておらず、いずれも `runtimeBinding=unbound` です。

`manual-unresolved`、`manual-external-unresolved` の Step は対応 Application Function がなくても valid な業務上の作業候補として残します。`ActorCandidate` は担当業務上の主体であり、Security / Authorization Role へ自動変換しません。`RoleAuthorizationBinding=not-inferred` は認可成功を意味しません。

JSON の `source.file` と `source.pointer` は当該 Source の位置を示します。存在しない Pattern、複数解決、欠けた Step、無条件の分岐・例外等は source-aware diagnostics として報告します。`unreferencedPatterns` は**この一つの Procedure に限る未使用一覧**であり、Application 全体の死んだ機能という断定ではありません。

## 実証と残る作業

最初の縦切りで確かめるのは #1119 の **V02 / N07**（Function のない manual / external Step が消えないこと）と、図レビュー候補を受理済み意味へ昇格しないことです。unit / integration test はそれらに加え、3種の trigger 候補、例外、Handoff、Source locator、無効参照を検査します。

現時点で以下は**未実証**です。

- Authoritative な Business Procedure の編集と Acceptance / 再読込（#1124）
- Actor / 業務 Role の複数責任割当て、具体的な手作業・電話確認、外部依存の設計編集
- 分岐・契機候補の業務上の確定、runtime trigger / schedule への Binding（#1121）
- Function / Data / Procedure を往復できる Studio UI と source-aware cross-view review（#1120 / #1132）
- 別資料の原文互換、旧製品挙動の再現、actual Target / Provider Evidence

この Adapter によって新しい Canonical Procedure / Workflow / Authorization Role を追加しません。今後「提案された図の修正 → Source の変更提案 → Shared Acceptance → 再投影」を進めるときは、既存契約の owner（#1120 / #1124）と調整します。

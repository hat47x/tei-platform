# TEI × SEI: ローカルCPU認知による実装再利用・標準部品化候補の研究

Status: **研究仮説／未実装／製品採用未決定**（2026-10-13）

関連Issue: TEI [#843](https://github.com/hat47x/tei-platform/issues/843), [#876](https://github.com/hat47x/tei-platform/issues/876), [#997](https://github.com/hat47x/tei-platform/issues/997), [#998](https://github.com/hat47x/tei-platform/issues/998), [#1121](https://github.com/hat47x/tei-platform/issues/1121), [#267](https://github.com/hat47x/tei-platform/issues/267); SEI [#543](https://github.com/hat47x/sei-cognition/issues/543), [#546](https://github.com/hat47x/sei-cognition/issues/546)。

## 背景と設計仮説

TEIの標準Web/Desktop業務UIでは、中央管理する標準コンポーネント／画面パターンを宣言的な定義から実行する方式を優先候補とする。一方、業務ロジック、専用UI、特殊実行要件は、引き続き汎用のコードジェネレーター・既存Native資産・案件Extensionで実現できる。コード生成機構を標準UIの例外に限定しない。

複数アプリケーションに現れる似た専用UI／専用ロジックから、**共通化に値する契約・設定可能な差異・移行不能な差異を発見する**のが本研究の目的である。発見の一次経路をローカルCPUで成立させ、曖昧な事例にのみJev／小型モデル／外部LLM／人手を用いる。ただしローカルCPUで必要な品質・費用を満たせるかは実測して判断する。

## 分けて保持する主体と責務

| Owner | 所有する内容 | 認知候補からの制約 |
| --- | --- | --- |
| #998 業務仕様テンプレート | Data・Function・Business Procedureの意味、案件具体化 | 見かけの類似で意味の同一性を確定しない |
| #997 UI構成キット | Renderer、Component、Host Bridge、標準UIパターン、案件Layer | UI操作・拡張点を扱い、業務仕様の第二正本を作らない |
| #1121 Lowering／#951契約 | Accepted meaningから実装への実現計画、Provider／Target適合 | 類似性スコアで暗黙Provider選択・自動fallbackしない |
| #843/#876 認知研究 | 候補集合の生成・ランキング・保留／棄却・評価データ | 学習器は提案専用。#843のissue-only／qualification停止条件を維持 |
| #267 Shared Acceptance | 採否、正本更新、reload／Validation／Conformance | 認知モデル・teacherの判断をAcceptと同一視しない |
| SEI #543/#546 | ローカル推論・蒸留の汎用技法／横断タスク分類 | TEIの正本や業務上の決定権を所有しない |

## 初期の入力・特徴・出力契約（案）

**入力**はアクセス許可済みの既存Source／Accepted Canonical／Action契約／UI Component contract／生成物への参照から作る*read-only* projectionに限定する。候補単位で `application/context identity`、`source locator + digest`、`baseModelRevision`、`contract identity/version/fingerprint`、Target／Framework、provenance、利用を許可された範囲を残す。案件をまたぐ探索には明示的なアクセス境界が必要であり、未許可の顧客コード・仕様・Telemetryを横断学習へ取り込まない。

**特徴**は少なくとも三系統に分ける。

1. 構造: 入出力の型、データ・関数従属性、Actionのeffects、状態遷移、依存グラフ、Extension Point、エラー契約、Target制約。
2. 振る舞い: Validation／Conformance／Scenarioの結果、契約テストの差分、入力・画面遷移（許可された集計のみ）、既知の負例。
3. 表層: 名前、コメント、ラベル、実装コード類似度。**順位づけの手掛かりであって業務意味の同一性の証拠ではない**。

**出力**は `reuse_candidate`（標準UI、標準ロジック、業務仕様テンプレート、生成テンプレート、案件固有維持）、`evidence_refs`、`common_contract`、`configurable_variants`、`incompatible_differences`、`unknown`、`baseModelRevision`、`confidence/ranking`、`recommended_tests` の*提案*のみ。固定した契約schemaを新Core kindとして先行導入しない。

## CPU-first候補探索

```text
許可済み複数アプリケーション／既存モデル／証拠
  -> read-only structural projection と候補集合の決定的生成
  -> hard incompatibility filter（契約・権限・Target等）
  -> lexical/hash・構造近傍のCPUベースライン
  -> 必要に応じて SEIのLightGBM等の軽量ranker / novelty / abstention
  -> 不確実・新規・高リスク時のみ Jev / local SLM / LLM / 人間
  -> explicit-reference Proposal + 差分・テスト候補
  -> #267 Shared Acceptance（採用を決めた場合のみ）
  -> 正本変更後のreload / Validation / Conformance
```

プロセス間・プラグイン境界には#951のService Contract／Authority scopeを利用する。抽出器、ranker、teacher、UI KitのいずれもCanonical write権限を暗黙には取得しない。導出インデックスや推論モデルは破棄・再構築できるCache／Research Artifactであり第二正本ではない。

## 共通部品への昇格条件

- 業務意味・公開Action契約が共通であるか、許容する差異をパラメータとして明示できる。
- UI操作の差異（フォーカス・IME・明細編集・a11y等）を正当なTarget差として扱える。Web標準＋LitとDesktop WebView／Go Hostの実証は#997のUI実装責務に従う。既存React Referenceを無条件に否定しない。
- API／権限／effects／失敗・Unknown／トランザクション／Revisionの適合性が機械検証される。
- 元の専用実装を保持した比較試験、複数案件での受入、移行／ロールバック計画が揃う。
- 共通化コストが案件差分の増大、複雑な設定、性能低下を招く場合は昇格させない。

## 最初の小規模Referenceと評価

**R0 — 学習なしの観測**: 受注明細UIや入力確定Actionなど、許可済み2アプリケーションの狭いスコープで類似候補を収集。単なるリネーム、表面的に類似しても業務意味が異なる負例、意味は同じでも実装が違う正例、実装のないunknownを用意する。適合判定は人間の確認と決定的な契約検証を分離する。

**R1 — ベースライン**: 正規化した契約比較、ルールフィルタ、lexical/hash類似、構造近傍検索を測る。候補抽出と判定に要した作業時間を記録する。

**R2 — 軽量学生モデル（解除条件成立後のみ）**: #843/#876の候補・履歴データ契約を再利用してlogistic/LightGBM等と比較。Jev等のteacherは正解の保証ではなくラベル候補。**#843の既存issue-only停止、#876のexact-head qualification、#267の実Acceptance履歴要件を変更しない。** 新しいteacher／model-training branchを本提案から立ち上げない。

**R3 — UI/ロジックの昇格試行**: 候補一件だけを#997 UI部品または#998意味Templateへ切り出し、元の案件実装と共通契約テストを比較する。標準化不適と判定した負例もEvidenceとして保存する。

指標を**品質と資源の独立軸**で測る: 候補recall@k、誤共通化提案、Unknown/OOD時の保留、revision drift、必要レビュー時間、案件別差分行数、共通部品化により減った保守コスト、CPU inference p50/p95、cold-start、peak RSS、model/index size、外部teacher呼出削減。閾値は計測前に恣意的に「達成済み」としない。

## 明確な非目標

- 自動的なCanonical／共通部品への昇格、無審査の業務意味統合。
- ソースコードの類似だけによる標準化、全業務で一つの万能モデルを使うこと。
- 全アプリケーションのコードを常時横断取得すること。
- 現行のTEI-LD研究停止条件の解除、Production機能完成の主張。
- 新たな認知用Canonical、第二Resolver、TEI専用のAcceptance経路の導入。

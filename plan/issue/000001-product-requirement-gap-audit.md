# ISSUE-000001: 現行実装をProduct Requirementsへ逆マッピングする

- 状態: Done

## 背景

TEIはM0.32までに多くの設計境界を実行可能なコードで検証してきましたが、その進捗はこれまでM0.xx、ADR、GitHub Issueを中心に整理されてきました。

新しくProduct Requirementsを上位基線として置いたため、現行Repositoryの実装・Conformance・Reference・文書を各Requirementへ逆マッピングし、どこが十分に検証され、どこがまだ概念だけなのかを明示する必要がありました。

## 関連するProduct Requirement

RQ-001〜RQ-024を対象としました。

## 実施内容

- 各Requirementについて、実装、Reference、Conformance / CIのEvidenceを確認
- `Strong Evidence`、`Partial Evidence`、`Design Only`、`Not Started`へ区分
- 実装量ではなく、Requirementに対する再現可能なEvidenceの有無で評価
- 未実証項目をそのまま24本の作業へ分解せず、KJ法で意味の近いGapへ再統合

## 成果物

監査結果は [`../PRODUCT_REQUIREMENT_GAP_AUDIT.md`](../PRODUCT_REQUIREMENT_GAP_AUDIT.md) に整理しました。

主要な発見は、M0.32までのTEIがSource / Meaning、Database、Execution / Observation、Plugin / Runtimeの下部構造に強いEvidenceを持つ一方、次の領域には実証がほとんどないことです。

- 既存認証・認可を保持した移行
- standard Application Pattern
- Driverによる設計シミュレーション
- target-neutral Presentation / Interaction
- 開発者向けEditor
- AIによる知識探索、確認提案、判断履歴

これらはProduct Vision上の大きな空白であり、今後のv1方向ではDatabase側の局所的な深掘りと比較して優先的に不確実性を減らす対象とします。

## 後続作業

- `ISSUE-000002`: v1 Application vertical slice
- `ISSUE-000003`: 既存認証Compatibility Reference
- standard Application Patternの実証
- Driverの最小vertical slice
- 第一TargetへのPresentation Projection
- Editor / AI支援UX

後半の作業は、Application vertical sliceの具体化結果を見て、必要以上に先回りした抽象設計にならない粒度でIssue化します。

## 完了確認

RQ-001〜RQ-024について「何がEvidenceで、何がまだ未検証か」をRepositoryだけから説明でき、未実証領域をProduct Valueに沿った次の作業へ接続できる状態になったため、このIssueは完了とします。

# ISSUE-000001: 現行実装をProduct Requirementsへ逆マッピングする

- Status: Ready

## 背景

TEIはM0.32までに多くの設計境界を実行可能なコードで検証してきましたが、その進捗はこれまでM0.xx、ADR、GitHub Issueを中心に整理されてきました。

新しくProduct Requirementsを上位基線として置いたため、現行Repositoryの実装・Conformance・Reference・文書を各Requirementへ逆マッピングし、どこが十分に検証され、どこがまだ概念だけなのかを明示する必要があります。

## 関連するProduct Requirement

全RQを対象とします。特に、Database / Runtime側と比べて実証が薄いRQ-010〜RQ-017を重点的に確認します。

## 作業内容

- RQ-001〜RQ-024ごとに、関連実装、ADR、Conformance、Reference、Issueを列挙する
- 状態を少なくとも次に分ける
  - 実装・Conformanceあり
  - 部分実証あり
  - 設計のみ
  - 未着手
  - 現行実装と要求の間にGapあり
- 同じ機能を複数Requirementへ重複計上する場合は、どの側面を検証しているかを明記する
- 実装量ではなく、要求に対するEvidenceの有無で評価する
- Gapを見つけても、このIssue内で無条件に実装へ進めず、必要なら後続Issueへ分離する

## 特に確認する偏り

M0.26以降、Database / Providerの意味境界が深く進んでいるため、次を確認します。

- Application PatternがProduct Visionに対して十分か
- 既存認証・認可互換のReferenceがあるか
- Driverを設計検証環境として扱う経路があるか
- target-neutral Presentationの実証があるか
- Editorの高密度Projectionが未実装のまま、Core側の設計だけが進みすぎていないか
- AI支援UXを受け止めるDecision / History / Suggestion境界があるか

## 成果物

`plan/`配下にGap Audit文書を追加し、必要に応じて後続Issueを起票します。

## 完了条件

別の開発者や生成AIがRepositoryだけを見て、各Product Requirementについて「何がEvidenceで、何がまだ未検証か」を説明できること。

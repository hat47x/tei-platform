# TEI Planning

このディレクトリには、TEI（鼎／テイ）の上位価値とProduct Requirementsを、実際の着手順、並走関係、設計判断、調査課題へつなぐ文書を置きます。

`plan/`はProduct Valueを定義する場所ではありません。何を守るかは`product/`、何を満たすかは`requirements/`、何から進めるかは`plan/`で扱います。

## 文書

- [`MASTER_PLAN.md`](MASTER_PLAN.md): 開発全体の作業軸と依存関係
- [`MILESTONES.md`](MILESTONES.md): 期限を置かないCapability MilestoneとVersion Horizon
- [`PRODUCT_REQUIREMENT_GAP_AUDIT.md`](PRODUCT_REQUIREMENT_GAP_AUDIT.md): Product Requirementsと現行実装のEvidence / Gap
- [`V1_REFERENCE_APPLICATION_PLAN.md`](V1_REFERENCE_APPLICATION_PLAN.md): `order_entry`を使ったv1 Application実証の主線とEvidence Gate
- [`adr/`](adr/README.md): 長く残す設計判断の正本
- [`issue/`](issue/): 調査、実装、文書整備などの作業単位

## 現在のv1主線

現在は、既存の`order_entry`をReference Applicationとして育てます。

```text
Application Pattern ────────┐
      ↓                     │
Application Driver     既存認証Compatibility
      ↓                     │
第一Web Target ─────────────┘
      ↓
Editor / AI支援
      ↓
第二TargetでTarget-neutral性を検証
```

個々の機能が実装されたことだけでは、次の段階へ進みません。

[`V1_REFERENCE_APPLICATION_PLAN.md`](V1_REFERENCE_APPLICATION_PLAN.md)では、Application上の意味をDriverへ渡せること、UIなしで主要シナリオを再現できること、第一Web Targetから共通意味とWeb固有差を分離できることなどをEvidence Gateとして定めています。

このGateによって、WebやEditorの実装都合から上位の意味モデルを先回りして固定することを避けます。

## ADR

ADRには、実装方法が変わっても残る設計判断を記録します。現在の正本は[`plan/adr/`](adr/README.md)です。

ファイル名は4桁の連番を使います。

```text
plan/adr/0018-example.md
```

ADR-0001〜0015は以前`docs/adr/`にありましたが、番号と本文を変えず`plan/adr/`へ移しました。ADR-0016以降も同じ連番で続いています。

旧URLや過去文書からの参照を壊さないため、`docs/adr/`にはADR-0001〜0015の互換案内だけを残しています。設計判断の本文を新旧二か所へ複製せず、`plan/adr/`だけを正本とします。

新しいADRは`plan/adr/`にのみ追加します。同じ番号を別の意味へ再利用しません。

## Issue文書

Repository内に長く残したい作業定義は、6桁の連番で`plan/issue/`へ置きます。

```text
plan/issue/000001-example.md
```

GitHub Issueの`#123`とRepository内のIssue IDは別物です。

- Repository内Issue: 長く残す作業定義、背景、制約、完了条件
- GitHub Issue: 実際の進行、会話、PRとの接続

必要な場合はRepository内IssueからGitHub Issue番号を参照します。GitHub Issueだけで十分な小さな作業まで、機械的に文書化する必要はありません。

## 計画の更新方法

計画は日付を消化するためではなく、Product Valueに対する不確実性を減らすために更新します。

各作業では、少なくとも次を確認します。

1. どのProduct Requirementに関係するか
2. 何がまだ分かっていないか
3. 実験・設計・実装のどれで確かめるか
4. 他のworkstreamと並走できるか
5. 先に固定すると後戻りしにくい判断があるか
6. 完了時に何をEvidenceとして残すか

実装しやすい領域だけが先行し続けないよう、MilestoneではSemantic、Application、Database、Editor、AI、Target、Runtimeの進み方を横断して確認します。

## 自然な日本語の確認

計画文書、ADR、Issueは、内容を整理した後に必ず全文を読み直し、意味を変えず自然な日本語へ整えます。

機械的な箇条書きだけで背景や判断理由が失われている場合は文章として補います。逆に、長い文章の中で条件や完了基準が埋もれる場合は、箇条書きへ分けます。

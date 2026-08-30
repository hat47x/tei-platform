# TEIの考え方

この文書は、[はじめてTEIを知る方へ](for-readers.md) より一段だけ技術寄りに、TEIの設計上の考え方を説明します。

## 一言でいうと

TEIは、業務システムの「意味」をコード・DB・画面・クラウド構成より一段上に置き、その意味がどの実装へ落ち、実際にどう実行されたかまで追跡できるローコード基盤を目指しています。

従来のローコード製品では、画面・テーブル・ワークフローを一つの専用環境に閉じ込めると短期開発は速くなっても、既存システム移行、複雑なSQL、外部API、固有認証、運用ツール、長期保守で専用世界の外へ出たくなることがあります。

TEIはその逆方向から設計しています。

- 意味はできるだけ明示する
- 実装方式は交換可能にする
- nativeなSQL / Go等はnativeなまま使える
- 実行時の推測は減らし、Bindingを明示する
- derived artifactはcanonical sourceへ逆流させない
- 実行結果はObservation / Evidenceで確認する

ここで **Canonical** は「特定の画面やDB製品などに依存しない、基準となる意味表現」、**Binding** は「その意味や要求を、環境上の具体的な実装へ結び付けること」を指します。

## 生成AIとの関係

TEIは「生成AIがあるからコードは全部毎回作り直せばよい」という前提には立ちません。

AIが最初に探す対象を、高水準の意味部品へ上げることを想定しています。

```text
1. Existing Domain Component
2. Existing Semantic Pattern / Facet
3. Existing Policy / Variation
4. Composition
5. Declarative expression / rule
6. Capability call
7. Custom implementation
```

つまり、まず既存の業務概念・意味パターン・規則・Capabilityを探索し、組み合わせ、それでも不足するところだけを個別実装へ落とします。

これによりAIの役割を、単なる大量コード生成から、既存知識の探索・組み合わせ・不足部分の実装へ寄せられます。

実行時には、AIのその場の判断を規則そのものとして扱うのではなく、採用したルール・Binding・権限・実現方法を明示的に固定し、後から追跡できることを重視します。

## CanonicalとProjection

TEIではCanonical Modelを一つの画面表現やファイル表現と同一視しません。

**Projection** は、同じ意味を用途に応じて見せたり編集したりするための表現です。

同じCanonicalな意味を、たとえば次のように投影できます。

- DSL
- Matrix / table
- Graph
- Flow
- UI
- Natural language

Projectionは見やすさや作業性のために存在し、Projection側の都合だけでCanonicalな意味を勝手に書き換えません。

## ApplicationとDatabase

業務システムでは、一つのApplicationが複数Databaseを利用するのは珍しくありません。TEIはこれを例外扱いしません。

Application側はlogical datasource aliasとDatabase Contractを要求し、Environmentが具体instance / providerへBindingします。

Database Contractは「そのDatabase ReleaseがApplicationへ何を提供できるか」を表す利用契約です。

DatabaseはDatabase自身のRelease / Migration / Contract / Data Readinessを持ち、Application Releaseとは独立します。

## Realizationとは何か

TEIでは、業務上の意味そのものと、その意味をどの技術で実現するかを分けます。

**Realization** は、Canonicalな意味や要求を、SQL、Go、特定Database Provider、外部Capability等の具体的な実現方式へ落とす段階です。

「PostgreSQLで実装した」という事実は重要ですが、それだけを「受注とは何か」という業務上の意味そのものにはしません。

## 実行と観測

TEIがDeployment commandを発行しても、command acknowledgementを最終事実にはしません。

外部世界をObservationし、必要ならEvidenceを得て、そこで初めてRunを進めます。この考え方はDB migration、canary、health gate、drain、retire、recoveryに共通です。

## なぜ現在Security / Deploymentのコードが多いのか

Repositoryを見ると、初期段階にもかかわらずPlugin permission、Resource Lease、fencing、takeover等のコードが多くあります。

これはTEIの中心がDeployment製品だからではありません。

「意味上はきれいでも、実行境界で自動推測・権限拡大・二重実行が起きる」設計を後から直すのは難しいため、Canonical Modelと同時に実行境界の性質を早期検証しています。

次に実装の全体像を追う場合は [Architecture](architecture.md)、現在どこまで実装済みかを確認する場合は [現在の実装状況](status.md) を参照してください。

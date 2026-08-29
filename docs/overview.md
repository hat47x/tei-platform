# TEIの考え方

## 一言でいうと

TEIは、業務システムの「意味」をコード・DB・画面・クラウド構成より一段上に置き、その意味がどの実装へ落ち、実際にどう実行されたかまで追跡できるローコード基盤を目指しています。

従来のローコード製品では、画面・テーブル・ワークフローを一つの専用環境に閉じ込めると短期開発は速くなっても、既存システム移行、複雑なSQL、外部API、固有認証、運用ツール、長期保守で専用世界の外へ出たくなることがあります。

TEIはその逆方向から設計しています。

- 意味はできるだけ明示する
- 実装方式は交換可能にする
- nativeなSQL/Go等はnativeなまま使える
- 実行時の推測は減らし、Bindingを明示する
- derived artifactはcanonical sourceへ逆流させない
- 実行結果はObservation/Evidenceで確認する

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

これによりAIの役割を、単なる大量コード生成から、既存知識の探索・組み合わせ・不足部分の実装へ寄せられます。

## CanonicalとProjection

TEIではCanonical Modelを一つの画面表現やファイル表現と同一視しません。

同じ意味を、用途に応じて次のように投影できます。

- DSL
- Matrix / table
- Graph
- Flow
- UI
- Natural language

Projectionは見やすさや作業性のために存在し、Canonicalな意味へ勝手に逆流しません。

## ApplicationとDatabase

業務システムでは、一つのApplicationが複数Databaseを利用するのは珍しくありません。TEIはこれを例外扱いしません。

Application側はlogical datasource aliasとDatabase Contractを要求し、Environmentが具体instance/providerへBindingします。

DatabaseはDatabase自身のRelease / Migration / Contract / Data Readinessを持ち、Application Releaseとは独立します。

## 実行と観測

TEIがDeployment commandを発行しても、command acknowledgementを最終事実にはしません。

外部世界をObservationし、必要ならEvidenceを得て、そこで初めてRunを進めます。この考え方はDB migration、canary、health gate、drain、retire、recoveryに共通です。

## なぜ現在Security/Deploymentのコードが多いのか

Repositoryを見ると、初期段階にもかかわらずPlugin permission、Resource Lease、fencing、takeover等のコードが多くあります。

これはTEIの中心がDeployment製品だからではありません。

「意味上はきれいでも、実行境界で自動推測・権限拡大・二重実行が起きる」設計を後から直すのは難しいため、Canonical Modelと同時に実行境界の性質を早期検証しています。

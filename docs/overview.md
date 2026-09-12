# TEIの考え方

この文書は、[はじめてTEIを知る方へ](for-readers.md) より一段だけ技術寄りに、TEIの設計上の考え方を説明します。

個別の型やAPIを追う前に、「なぜTEIが意味・実装・権限・観測を分けるのか」をつかむための入口です。技術境界をさらに詳しく読む場合は、[Architecture 読み方ガイド](architecture-reading-guide.md)を経由すると全体像を追いやすくなります。

## 一言でいうと

TEIは、業務システムの「意味」をコード・DB・画面・クラウド構成より一段上に置き、その意味がどの実装へ落ち、実際にどう実行されたかまで追跡できるローコード基盤を目指しています。

従来のローコード製品では、画面・テーブル・ワークフローを一つの専用環境に閉じ込めると短期開発は速くなっても、既存システム移行、複雑なSQL、外部API、固有認証、運用ツール、長期保守で専用世界の外へ出たくなることがあります。

TEIはその逆方向から設計しています。

- 意味はできるだけ明示する
- 実装方式は交換可能にする
- nativeなSQL / Go等はnativeなまま使える
- 実行時の推測は減らし、Bindingを明示する
- derived artifactはCanonical sourceへ逆流させない
- 実行結果はObservation / Evidenceで確認する

ここで **Canonical** は「特定の画面やDB製品などに依存しない、基準となる意味表現」、**Binding** は「その意味や要求を、環境上の具体的な実装へ結び付けること」を指します。

## なぜ「意味」を先に置くのか

たとえば「受注は顧客を参照し、複数の明細を持つ」という業務上の意味を考えます。

Web画面では顧客検索dialogと明細tableになるかもしれません。Desktopでは別のUIになり、APIではCustomer IDと明細配列になるかもしれません。

もし画面やframeworkの構造を業務意味そのものにすると、Targetや技術を変えるたびに設計の中心まで作り直すことになります。

TEIでは、まず変わりにくい意味を保持し、その後で用途別の表現や実装へ落とします。

```text
業務上の意味
  OrderはCustomerを参照する
  OrderはOrderLineを複数持つ
        ↓
Application Pattern / Interaction
        ↓
Web / Desktop / API等のProjection
        ↓
具体的な実装
```

この方向を守ることで、技術を入れ替えても「何を実現するシステムなのか」を失いにくくします。

## 生成AI・ローカル認知との関係

TEIは「生成AIがあるから、設計書やコードを毎回大量に作り直せばよい」という前提には立ちません。

AIが自由生成する前に、TEI自身が持っている既存の意味、関係、制約、Reference、実行能力を使い、問題をできるだけ小さくします。

まず、再利用できるものを次の順で探します。

```text
1. Existing Domain Component
2. Existing Semantic Pattern / Facet
3. Existing Policy / Variation
4. Composition
5. Declarative expression / rule
6. Capability call
7. Custom implementation
```

たとえば「受注明細を持つ受注登録を設計したい」という要求に対して、AIへいきなり自由回答を求めるのではなく、先に次を確認できます。

- 既存のOrder / Customerはあるか
- header-detailのApplication Patternは使えるか
- 顧客masterはread-onlyか
- quantityに既存Validationがあるか
- Targetが必要なinteractionを提供できるか
- 類似するReferenceや過去の設計はあるか

そのうえで、複数の設計候補を作り、既知のconstraintや他の設計viewから不適合案を除外します。

```text
業務意図
  ↓
既存資産・制約・Referenceをローカルで探索
  ↓
複数の設計候補
  ↓
constraint / cross-viewで候補を縮約
  ↓
残った少数案をlocal LLMで比較・統合
  ↓
人が意味上の選択を確認して受理
  ↓
実装・テストを可能な範囲で再生成
```

ここで重要なのは、**LLMを最初の探索器にも、最終authorityにも固定しないこと**です。

exact lookup、graph traversal、constraint check、Pattern recall、構造的な類似探索等は、できる限り軽量なローカル処理で行います。その結果を小さく構造化したうえでlocal LLMへ渡せれば、大きなmodelへRepository全体を毎回読ませなくても実用的な設計支援へ近づけます。

local LLMで十分なReference Scenarioを持つことを目標にしつつ、必要ならremote LLMをoptionalに利用できる構造を想定しています。特定のmodel名やparameter数をCanonical Architectureへ固定しません。

AIが作った内容は、採用されるまではProposalです。AIの提案能力が高くても、Canonical meaningの変更権限、本番操作権限、secret / network / filesystem等のResource authorityを自動的に持つわけではありません。

より詳しい設計方向は、[Definition-Centered Feedback Loop](definition-centered-feedback-loop.md) と [`Design Synthesis Loop Requirement Draft`](../plan/DESIGN_SYNTHESIS_LOOP_REQUIREMENT_DRAFT_2026-09.md) を参照してください。

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

たとえばMatrix上で空欄に見えるからといって、それだけで「関係が存在しない」とCanonicalへ書き戻すことはしません。単にtraceが不足している可能性や、まだ判断できない可能性があります。

## ApplicationとDatabase

業務システムでは、一つのApplicationが複数Databaseを利用するのは珍しくありません。TEIはこれを例外扱いしません。

Application側はlogical datasource aliasとDatabase Contractを要求し、Environmentが具体instance / providerへBindingします。

Database Contractは「そのDatabase ReleaseがApplicationへ何を提供できるか」を表す利用契約です。

DatabaseはDatabase自身のRelease / Migration / Contract / Data Readinessを持ち、Application Releaseとは独立します。

たとえば受注Applicationが販売DBへ書き込み、顧客masterを読み取り専用で参照する場合、二つのDatabaseを同じrelease cycleへ固定する必要はありません。

## Realizationとは何か

TEIでは、業務上の意味そのものと、その意味をどの技術で実現するかを分けます。

**Realization** は、Canonicalな意味や要求を、SQL、Go、特定Database Provider、外部Capability等の具体的な実現方式へ落とす段階です。

「PostgreSQLで実装した」という事実は重要ですが、それだけを「受注とは何か」という業務上の意味そのものにはしません。

この分離により、後からProviderや実装方式を変更しても、同じ業務意味を維持できる余地を残します。

## 実行と観測

TEIがDeployment commandを発行しても、command acknowledgementを最終事実にはしません。

外部世界をObservationし、必要ならEvidenceを得て、そこで初めてRunを進めます。この考え方はDB migration、canary、health gate、drain、retire、recoveryに共通です。

たとえばmigration commandが成功しても、接続先が想定と違っていたり、外部状態が期待どおりになっていない可能性があります。

```text
「実行を要求した」
    !=
「外部で意図した状態になった」
```

この区別は、AIやRuntimeが「成功」と返しただけで業務上の事実を確定しないためにも重要です。

## なぜ現在Security / Deploymentのコードが多いのか

Repositoryを見ると、初期段階にもかかわらずPlugin permission、Resource Lease、fencing、takeover等のコードが多くあります。

これはTEIの中心がDeployment製品だからではありません。

「意味上はきれいでも、実行境界で自動推測・権限拡大・二重実行が起きる」設計を後から直すのは難しいため、Canonical Modelと同時に実行境界の性質を早期検証しています。

たとえばHost交代時に古いHostが同じResourceを操作し続ければ、上流の設計が正しくても二重実行や破壊的操作が起こり得ます。こうした失敗を防ぐ境界を先に検証しています。

Security / Runtimeの具体例は [Security / Trust Boundary](security.md) を参照してください。

## 次に読むもの

- Architectureを根幹から読みたい: [Architecture 読み方ガイド](architecture-reading-guide.md)
- 正式なProduct Requirementを理解したい: [Product Requirements 読み方ガイド](../requirements/README.md)
- 業務システム設計を知りたい: [Business-System Design](business-system-design.md)
- 実装の全体像を追いたい: [Architecture](architecture.md)
- 現在どこまで実装済みか確認したい: [現在の実装状況](status.md)

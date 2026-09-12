# TEI Product Requirements 読み方ガイド

この文書は、[`TEI_PRODUCT_REQUIREMENTS.md`](TEI_PRODUCT_REQUIREMENTS.md) にある正式なProduct Requirementを、**根幹の目的から個別要求へ降りて読めるようにする案内**です。

正式な要求の正本は`TEI_PRODUCT_REQUIREMENTS.md`です。この文書はRequirementを追加・変更するものではありません。

TEIには、業務モデリング、Database、Target、Runtime、Security、AI、業務手順など異なる領域の要求があります。RQ番号を順番に読むだけでは、それらがなぜ一つのプロダクトに必要なのか見えにくいため、ここでは「何を守るための要求か」でまとめ直します。

## 1. まず一つの流れとして見る

TEIの要求は、おおむね次の一連の流れを守るためにあります。

```text
業務上の意味を定義する
        ↓
既存の知識・実装へ結び付ける
        ↓
Application / Database / Targetへ具体化する
        ↓
明示された権限で実行する
        ↓
外部で実際に起きたことを観測する
        ↓
Evidenceをもとに設計や実装を改善する
```

この流れの途中で、意味、実装、権限、観測結果を同じものとして扱わないことがTEIの基本です。

たとえば、次の四つは似ていますが別です。

```text
「受注を登録する」という業務上の意味
        !=
Web画面で登録するという実現方法
        !=
その処理を実行できる権限
        !=
実際に外部Databaseへ登録されたという観測結果
```

この区別を維持するために、複数のRequirementが互いに支え合っています。

## 2. 要求群を六つのテーマで読む

### A. 意味を実装から守る — RQ-001〜004

ここでは、業務上の意味をfile path、SQL、Go、Database製品、画面等へ固定しないことを要求します。

主な失敗例は、次のようなものです。

- fileを移動しただけで別の業務概念になってしまう
- generated artifactがいつの間にか第二の正本になる
- SQLや既存Go実装をTEI利用のためだけに書き直す
- 同じ意味を複数sourceからheuristicにmergeし、どれが正しいかわからなくなる

対応する主なRequirement:

- **RQ-001** — 業務上の意味を実装技術から独立して保持する
- **RQ-002** — Authoring sourceとCanonical meaningを追跡する
- **RQ-003** — 暗黙の複数正本を作らない
- **RQ-004** — Native assetをnativeなまま利用する

ここでいう**Canonical**は、特定の画面やDatabase製品ではなく、TEIが基準として扱う意味です。

### B. Applicationと実装を安全に組み立てる — RQ-005〜009

業務システムでは、一つのApplicationが複数Databaseや外部機能を利用し、既存システムと段階的に共存することがあります。

この領域では、Application、Database、Provider、Capabilityを最初から一枚岩にしないことを要求します。

主な失敗例:

- Application ReleaseとDatabase Releaseを常に一対一に固定する
- actual Databaseの情報不足を「差分なし」と判断する
- PostgreSQLや特定Cloudの仕様がCoreへ広がり続ける
- 標準認証へ移行できない既存システムをTEI対象外にしてしまう

対応する主なRequirement:

- **RQ-005** — ApplicationとDatabaseをN:Mで扱う
- **RQ-006** — Desired / Actual / Diff / Unresolvedを分ける
- **RQ-007** — Provider固有知識をCoreへ無制限に持ち込まない
- **RQ-008** — Capability単位で実装を置き換える
- **RQ-009** — 既存認証・認可仕様を保持した移行を可能にする

### C. 設計を多面的に見て、実装前に検証する — RQ-010〜014

TEIでは、設計を一つのフォームや一つのDiagramへ閉じ込めません。

同じ意味を、表、Graph、Flow、UI、自然言語等から確認し、Driverで実行可能な部分を早期に試します。

主な失敗例:

- Editorで表現できるものだけがCore conceptになる
- Web画面の都合がCanonical meaningを決める
- 実装が終わるまで設計矛盾に気づけない
- browse / record等の頻出構造を毎回ゼロから作る

対応する主なRequirement:

- **RQ-010** — 高密度なEditor
- **RQ-011** — 同じCanonical meaningの複数Projection
- **RQ-012** — Driverによる設計検証
- **RQ-013** — Target-neutralなPresentation / Interaction
- **RQ-014** — Application Patternの再利用

**Projection**は、同じ意味を用途別に見せる表現です。Projectionは読みやすさや編集しやすさを提供しますが、それ自体を別の正本にはしません。

### D. AIを「自由生成器」ではなく設計支援へ使う — RQ-015〜017

TEIでは、AIに毎回Repository全体を読ませ、コードや設計を一から自由生成させることを標準経路にしません。

まず既存の意味資産、Pattern、Rule、Capabilityを探し、利用可能な候補を絞り、それでも不足する部分だけを新規提案へ進めます。

主な失敗例:

- 同じ業務意味をAIが毎回違うコードとして再生成する
- AI提案がReviewなしにCanonicalへ書き込まれる
- 過去に回答済みの確認を繰り返す
- AIの自律度が高いことを理由に、write / secret / network権限まで強くする
- 同じ情報をSemantic Model、generated file、AI memoryへ人が二重・三重管理する

対応する主なRequirement:

- **RQ-015** — 既存知識を探索してから新規実装へ進む
- **RQ-016** — AIの確認提案を控えめかつ履歴付きにする
- **RQ-017** — 人の二重管理を最小化する

現在のDesign Synthesis構想では、この要求群をさらに、

```text
複数の設計候補を作る
        ↓
既存定義・制約・他ビューで候補を絞る
        ↓
必要な部分だけlocal LLMで比較・統合する
        ↓
人が意味上の選択を受理する
        ↓
実装・テストを可能な限り再生成する
```

という方向へ発展させる検討をしています。ただし、正式RQの追加・変更はReference Evidenceを得てから判断します。

### E. 実行・権限・観測を混同しない — RQ-018〜020

設計が正しくても、実行時に権限が広がったり、副作用を二重実行したりすれば安全な業務基盤にはなりません。

主な失敗例:

- commandが受理されたことを、外部状態が変わった証拠にする
- PluginがCapabilityを知っているだけで実行権限も得る
- Secretやnetwork endpointをCanonicalへ保存する
- Host takeover後も古いHostが操作を続ける
- 外部processの状態が不明だから同じ処理を自動再実行する

対応する主なRequirement:

- **RQ-018** — CommandとObservationを分ける
- **RQ-019** — 実行時の権限とResourceを明示的に制約する
- **RQ-020** — 再実行・takeover・unknownを安全に扱う

Security / Runtimeの具体的な説明は[`../docs/security.md`](../docs/security.md)を参照してください。

### F. 「動いた」をProduct claimへ飛躍させない — RQ-021〜026

最後の要求群は、TEI自身の設計や評価方法を守ります。

Referenceや実験が成功しても、そのままproduction-readyとは言えません。また、Business Procedure、Data、Function等の一視点だけで業務全体を説明したことにもなりません。

主な失敗例:

- toy fixtureで動いた結果をproduction保証へ広げる
- experimental frameworkの成功率をTEIのKPIへそのまま転記する
- 一つのTargetで速かったからすべてのTargetへ同じ性能条件を課す
- Application Functionに現れない手作業や外部作業を業務モデルから消す
- Data、Function、Businessの矛盾を別々の文書で見逃す

対応する主なRequirement:

- **RQ-021** — Reference / Conformanceで再現可能性を示す
- **RQ-022** — 完成製品と設計検証段階を区別する
- **RQ-023** — 一人でも維持可能な構造を保つ
- **RQ-024** — 性能要求をTargetごとに検証可能にする
- **RQ-025** — Business Action / ProcedureをApplication Functionから独立して扱う
- **RQ-026** — Data / Function-Interaction / Business Procedureを相互検証する

RQ-025 / RQ-026は、[`BUSINESS_SYSTEM_DESIGN_REQUIREMENTS.md`](BUSINESS_SYSTEM_DESIGN_REQUIREMENTS.md)で、もう一段具体的に解釈しています。

## 3. 一つの受注例で要求のつながりを見る

たとえば「顧客を選び、複数の明細を持つ受注を登録する」という業務を考えます。

```text
業務意味
  OrderはOrderLineを複数持つ
  Customerは既存masterから参照する
  quantityは正でなければならない
        ↓
Application
  record + header-detail + reference
        ↓
Database
  受注DBはread-write
  顧客masterはread-only
        ↓
Target
  Webでは受注明細を編集できる
        ↓
Runtime
  認可された利用者だけ登録Capabilityを実行
        ↓
Observation
  登録後のactual stateを必要に応じて確認
```

この一例でも、複数Requirementが関係します。

- 顧客masterのfile位置が変わってもCustomerの意味を変えない — RQ-001 / 002
- 顧客masterへwriteしない — RQ-005 / 019
- read-onlyなのに編集UIを提案したらcross-viewで検出する — RQ-013 / 026
- header-detailを既存Patternから再利用する — RQ-014
- AIは既存Order / Customer構造を探してから案を作る — RQ-015
- `quantity > 0`からnegative testを導出する余地を持つ — RQ-015 / 017 / 021に接続する設計課題
- command successだけで登録完了としない — RQ-018

RQを単独のチェック項目として見るより、同じ業務シナリオの中でどの失敗を防ぐ要求かを見る方が、設計判断へつなげやすくなります。

## 4. Requirementを読むときの注意

### Requirementは実装方式を固定しない

たとえば「staleなAI Proposalを黙って適用しない」という要求があっても、特定のversion storeやconflict algorithmを要求しているとは限りません。

Product Requirementはまず**何を守るか**を定めます。具体的な方式はArchitecture、ADR、Referenceで比較します。

### Requirementがあることと、実装済みであることは別

```text
Product Requirement exists
  != implementation complete
  != Reference Evidence acquired
  != production-ready
```

現在の実装範囲は[`../docs/status.md`](../docs/status.md)を参照してください。

### 「未解決」は要求違反とは限らない

上流設計では、情報が不足している場合に`unknown`や`unresolved`を保持することがあります。

根拠のない値で空欄を埋めるより、何がわかっていないかを明示する方が安全です。何を満たせば判断できるかはReferenceやPlan Issueで具体化します。

## 5. 次に読むもの

- 正式な要求本文: [`TEI_PRODUCT_REQUIREMENTS.md`](TEI_PRODUCT_REQUIREMENTS.md)
- 業務システム設計の要求解釈: [`BUSINESS_SYSTEM_DESIGN_REQUIREMENTS.md`](BUSINESS_SYSTEM_DESIGN_REQUIREMENTS.md)
- Architectureを根幹から読む: [`../docs/architecture-reading-guide.md`](../docs/architecture-reading-guide.md)
- RequirementからAcceptance / Evidenceまで追う: [`../plan/UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md`](../plan/UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md)
- 現在の実装状況: [`../docs/status.md`](../docs/status.md)

# TEI Product Requirements 読み方ガイド

この文書は、[`TEI_PRODUCT_REQUIREMENTS.md`](TEI_PRODUCT_REQUIREMENTS.md) にある正式なProduct Requirementを、根幹の目的から個別要求へ降りて読めるようにする案内です。

正式な要求の正本は`TEI_PRODUCT_REQUIREMENTS.md`です。この文書はRequirementを追加・変更するものではありません。

TEIには、業務モデリング、Database、Target、Runtime、Security、AI、業務手順など異なる領域の要求があります。RQ番号を順番に読むだけでは、それらがなぜ一つのプロダクトに必要なのか見えにくいため、ここでは「何を守るための要求か」でまとめ直します。

## 1. まず一つの流れとして見る

TEIの要求は、おおむね次の一連の流れを守るためにあります。

```text
業務上の意味を定義・拡張する
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

これらの要求は、Product Valueが定める個人・組織・社会の価値を、プロダクトの能力として支えるためにあります。どの価値をどのRequirementが支えるかは、[`TEI_PRODUCT_VALUE.md`](../product/value/TEI_PRODUCT_VALUE.md)の「意味のSSOTが生む価値」で各価値の項目ごとに示しています。

## 2. 要求群を九つのテーマで読む

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

ここでいうCanonicalは、特定の画面やDatabase製品ではなく、TEIが基準として扱う意味です。

### B. Applicationと実装を安全に組み立てる — RQ-005〜009

業務システムでは、一つのApplicationが複数Databaseや外部機能を利用し、既存システムと段階的に共存することがあります。

この領域では、Application、Database、Provider、Capabilityを最初から一枚岩にしないことを要求します。

主な失敗例は次のとおりです。

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

主な失敗例は次のとおりです。

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

Projectionは、同じ意味を用途別に見せる表現です。Projectionは読みやすさや編集しやすさを提供しますが、それ自体を別の正本にはしません。

### D. AIを「自由生成器」ではなく設計支援へ使う — RQ-015〜017

TEIでは、AIに毎回Repository全体を読ませ、コードや設計を一から自由生成させることを標準経路にしません。

まず既存の意味資産、Pattern、Rule、Capabilityを探し、利用可能な候補を絞り、それでも不足する部分だけを新規提案へ進めます。

主な失敗例は次のとおりです。

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

主な失敗例は次のとおりです。

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

主な失敗例は次のとおりです。

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

### G. 人とAIが長く一緒に開発し、業務を続ける — RQ-027〜030

生成AIが実装や業務の一部を担うようになると、問題は「一度作れるか」から「数か月後に、別のAIや別の担当者が安全に続きを担えるか」へ移ります。

この領域では、Applicationの知識を会話履歴や特定の人の記憶へ閉じ込めず、人とAIが同じ正本から理解・承認・再利用・業務遂行を続けられることを要求します。

主な失敗例は次のとおりです。

- 前回のAIとの会話履歴が無いと、Applicationの意味や過去の決定を誰も再構成できない
- 人が生成コードの全量を読めないまま承認し、承認が形だけになる
- 別Applicationへ意味をcopyした結果、元と先が黙って分岐する
- AIエージェントが業務手順を読めることを理由に、外部へ副作用を起こす権限まで持つ
- AIが命令を出したことを、業務が完了したことと読み替える

対応する主なRequirement:

- **RQ-027** — AIが長期間にわたりApplicationへ再入場できる開発contextを提供する
- **RQ-028** — 人間が実装詳細を全面的に読まずに変更を理解・承認できる
- **RQ-029** — Domain・Infrastructure知識を適用条件付きで再利用・移植する
- **RQ-030** — AIエージェントがTEI成果物を業務遂行のOperational Knowledgeとして利用する

この群は、Product Valueでいう個人の価値（引継ぎ、中身のある承認）、組織の価値（知識資産の再利用）、社会の価値（人による監督を名目にしないこと、権限の暗黙の拡大を防ぐこと）に直接つながります。RQ-030の業務遂行も、RQ-018〜RQ-020のCommand / Observation / 権限 / unknown境界を迂回しません。ただし、この群には要求として定めた段階の項目が含まれ、価値の実証ではありません。RQ別の実証状況は[`PRODUCT_REQUIREMENT_GAP_AUDIT.md`](../plan/PRODUCT_REQUIREMENT_GAP_AUDIT.md)が正本です。

### H. Coreを固定語彙へ閉じない — RQ-031〜033

TEIでは、データモデルも含め、現在あるモデル体系を将来のすべてのApplicationへ強制しません。Coreが持つべきなのは業務概念の網羅ではなく、新しい意味を既存の意味と関係付け、衝突を検出し、複数の表現や実装へ接続できる共通機構です。

主な失敗例は次のとおりです。

- データモデルを唯一の上位モデルにし、行為・Interaction・手順等を無理に従属させる
- Pluginがコードは追加できても意味を宣言できず、EditorやAIから不透明な固有実装になる
- 拡張のversionや依存・互換範囲が分からず、同じnamespaceの意味が知らないうちに変わる
- 拡張同士の競合を読み込み順で黙って解決する
- AIが既存の意味拡張を発見できず、同じ概念や規則をApplicationごとに再実装する
- 情報不足や複数候補をAIが暗黙に一つへ決める

対応する主なRequirement:

- **RQ-031** — 特定のモデル体系をCoreで特権化しない
- **RQ-032** — Plugin / DSLで意味論を宣言・合成する
- **RQ-033** — AIとToolが意味拡張を実装のレールとして利用する

この要求群により、TEIの拡張性を「実装を差し替えられること」から「意味体系そのものを後から追加し、既存の意味と矛盾なく扱えること」まで広げます。生成AIは、その拡張を探索・合成・検証することで、毎回ゼロから設計判断をやり直す範囲を減らします。


### I. 開発環境を囲い込まない — RQ-034

生成AIを開発基盤に使う場合も、TEI専用のAgent Runtimeへ作業を閉じ込めません。ChatGPT等のチャット実行環境、Codex、Claude Code、通常のIDEやCLI、CIは、それぞれ独立した開発環境としてRepositoryを扱えます。

重要なのは、どのToolが編集したかをTEIへ細かく記録することではありません。各環境がSourceを理解し、編集し、必要なBuildやTestまで行い、その結果をGit等の標準的な経路で受け渡せることです。統合後は、元のChatやAgentを必要とせずauthoritative sourceからCanonical meaningを再構築できなければなりません。

主な失敗例は次のとおりです。

- TEIを編集するために特定のAgent AdapterやChat UIが必須になる
- AIがSourceを編集する前に、すべての変更をTEI独自Proposal形式へ変換しなければならない
- GitやCIが持つ作業状態をAgent Session、Checkpoint、検証状態等としてTEIへ二重管理する
- Gitで十分な変更にもTEI独自Change Setを必須にする
- MCPから呼び出した操作だけが特別なSemantic authorityを持つ
- Chat環境でMCPを使うという理由だけで、TEI自身のdeveloper MCP serverを必須にする
- 開発環境のShell / Filesystem権限とApplication RuntimeのAuthorityを混同する

対応するRequirement:

- **RQ-034** — 標準的な開発環境からTEIを改修・実行・検証できる

RQ-034はRQ-015やRQ-027を置き換えません。RQ-015はAIが何を参照して実装するか、RQ-027は後から再入場するための開発contextを扱います。RQ-034は、それらを利用する開発環境を特定のAgent製品やTEI専用workflowへ固定しないための境界です。Chat側がMCPを利用する場合も、まずHostが持つShell / Filesystem / Git MCP等からTEI CLIを呼べればよく、TEI固有MCPは追加価値が確認された場合の任意Projectionです。

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

Product Requirementはまず何を守るかを定めます。具体的な方式はArchitecture、ADR、Referenceで比較します。

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

- 要求が守る価値（個人・組織・社会）: [`../product/value/TEI_PRODUCT_VALUE.md`](../product/value/TEI_PRODUCT_VALUE.md)
- 正式な要求本文: [`TEI_PRODUCT_REQUIREMENTS.md`](TEI_PRODUCT_REQUIREMENTS.md)
- 業務システム設計の要求解釈: [`BUSINESS_SYSTEM_DESIGN_REQUIREMENTS.md`](BUSINESS_SYSTEM_DESIGN_REQUIREMENTS.md)
- Architectureを根幹から読む: [`../docs/architecture-reading-guide.md`](../docs/architecture-reading-guide.md)
- RequirementからAcceptance / Evidenceまで追う: [`../plan/UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md`](../plan/UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md)
- 現在の実装状況: [`../docs/status.md`](../docs/status.md)

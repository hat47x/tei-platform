# Wagby 11の機能体系からTEIへ取り込む設計課題（2026-11-15）

> **位置づけ**：外部製品の公開機能紹介・マニュアルを起点とした比較調査と、既存TEI設計・実装Ownerへの引渡し案。TEIの正式なGrammar、Canonical Schema、製品受入、Plugin方式をこの文書だけで決定しない。外部製品の記述をTEIの実装実績と混同しない。
>
> **優先順位**：正式`*.tei.md` Reader／選択Language Provider、通常`tei` CLIからの実Provider呼出し、TEI Studio Stage1→Stage2の実WindowsセルフビルドをP0として維持する。Wagbyとの差分は、その縦断実証の**題材と反例**に接続し、別の新P0や第二のHost／Registry／意味正本を立てない。

## 1. 対象と読み方

公開資料を2026-11-15に確認した。

- [Wagby 11 機能紹介](https://wagby.com/feature/index.html)：16分類の機能一覧とMCPサーバの説明への入口。
- [Wagby 11 マニュアル](https://wagby.com/manual11/index.html)：31グループ。各ページの「機能」「AIへの指示例」「AppSchemaの例」「同梱guide/referenceへの案内」を参照。
- [AppSchemaの改版履歴](https://wagby.com/manual11/overview-revision-history.html)、[参照値の保存方式](https://wagby.com/manual11/calculations-lookup.html)、[親子同時更新](https://wagby.com/manual11/parent-child-bulk-edit.html)、[入力不可項目の送信](https://wagby.com/manual11/input-control-submit.html)、[計算式](https://wagby.com/manual11/calculations-expressions.html)。
- [名前付き検索ビュー](https://wagby.com/manual11/search-views.html)、[権限の範囲](https://wagby.com/manual11/security-scope.html)、[条件付き権限](https://wagby.com/manual11/security-conditions.html)、[承認人数](https://wagby.com/manual11/workflow-approval-count.html)、[代理承認](https://wagby.com/manual11/workflow-delegation.html)。
- [一括取込の失敗時処理](https://wagby.com/manual11/data-transfer-errors.html)、[PDF帳票](https://wagby.com/manual11/reports-pdf.html)、[メール送信管理](https://wagby.com/manual11/mail-status.html)、[業務処理の追加](https://wagby.com/manual11/customization-business-rules.html)、[標準機能とカスタマイズ](https://wagby.com/manual11/customization-standard-boundary.html)、[定義が異なる環境への移行](https://wagby.com/manual11/operations-data-migration.html)、[設計エラーの追跡](https://wagby.com/manual11/operations-schema-errors.html)、[MCPサーバ](https://wagby.com/feature/mcp-server.html)。

公開ページは製品の能力と設定意図を知る資料であり、記載の全機能を製品上で実地検証したものではない。公開ページには、帳票用JRXML、業務SPI用Java、メール配送やMCP接続の別途設定など、宣言だけで成立しない境界も明記されている。機能数・画面数を競うより、業務モデルから実際に動くアプリをどの程度**簡潔に・安全に・検証可能に**作れるかを比較する。

## 2. TEIが採るべき基本方針

**Wagbyからは実務上必要な振る舞いと設定の粒度を学び、WagbyのAppSchemaや実行方式をそのままTEIに移さない。**

TEIでは、業務担当者の認知順で読み書きできる `*.tei.md` と、標準データ構造上の型付き意味を中心に置く。Markdown、表、Graph、関数チェーンはその用途に適した表現であり、競合する第二の意味正本にしない。Goの`struct`／`func`／型引数や、WagbyのYAMLキー群を業務DSLへ露出させない。通常のコード生成でGo単一バイナリと外部Host向け成果物を作り、Target依存の描画・配送・帳票は選択されたPlugin／Generatorへ分ける。生成アプリにTEI独自Hostを常駐させる前提を置かない。

要求の分割は以下を基準とする。

| 意味上の責任 | 標準の振る舞い | 実装・検証上の境界 |
| --- | --- | --- |
| Data／Relation | 識別、複合キー、参照・多重性、親子孫、導出、時点と履歴 | 候補／受理済みMeaning／物理PK／業務キーを区別 |
| Query／Projection | 条件・並べ替え・名前付き検索・集計・全文検索・表示用参照 | QueryフィルタはAuthZの代替ではない |
| Operation／Lifecycle | 登録・更新・削除、バッチ、確認、Undo候補、事前検証、commit | UI仮値、検証、DB確定、外部Effectを分離 |
| Policy／Organization | CRUD、所属、所有、項目公開、操作時点の許可、委任 | Actor、責任者、委任元、現在Grantと監査記録を分離 |
| Presentation／Interaction | 入力制御、画面、ナビゲーション、帳票、メール、Mobile/VUI/MCP | 共通業務Operationを使うが媒体ごとの受領は別 |
| Build／Evidence | 定義変更、生成、試験、移行、配布と復元 | Source／Meaning／Artifact／実Provider／実Effectの資格を混ぜない |

## 3. 第一優先：TEIの意味モデルを鍛える六つの実務課題

### 3.1 参照と導出の「いつの値か」

Wagbyのルックアップでは、取引時点の商品名を**値として保存**、参照先のキーを保存して**現在値を表示**、または**保存せず表示時に取得**する方式を区別する。最初のキーが不変でも中間の参照先が変わると結果が異なる。この差は単なるUIプロパティではなく、業務記録の意味である。

TEIは「参照先Identity」「採用時点のSnapshot」「現在のProjection」「再取得契機」「欠損時の扱い」を分離する。価格・名称を変更後も保存した受注明細の単価が変わらないこと、一方で現在商品名の参照画面は更新されることを受入に使う。REAの経済事象・資源・主体、`validAt`／`knownAt`、Source/Providerの取得完全性と接続する。保存値をキーに切り替える変更はDB Migrationであり、単なる画面設定変更としない。

### 3.1.1 複数のデータモデルが同じ物理表を使う場合

Wagbyには、[同じ物理表を複数のモデルから利用する仕組み](https://wagby.com/manual11/relations-shared-table.html)があり、詳細データと集計・参照用モデルを分けられる。さらに[複合キーの中間表](https://wagby.com/manual11/relations-multi-composite.html)、[多段階のルックアップ](https://wagby.com/manual11/relations-lookup-via.html)が実務上の前提となる。TEIでは「業務上の実体とRelation」「読取用Projection」「物理表と索引」「画面上の表示モデル」を別の役割として関連付け、一つの物理表を参照するViewを**第二の業務正本**にしない。複合キーの一部を省いて誤結合しないこと、同じ物理行を表示する複数Projectionの参照権限を個別に評価すること、導出モデルの更新が元モデルへ自動的に書き戻されないことを検証する。

ただし、共有物理表を製品の標準DSL構文として最初から固定する必要はない。最初は既存のData／Relation／Query／Providerで表現できる範囲と不足を示す。既存DBを使う場合には、[DB構造と生成物の不整合の診断](https://wagby.com/manual11/operations-database-troubleshooting.html)も参照し、型・列・キー・参照制約・Migration資格を実Providerで確認する。

### 3.2 親子孫と複合キーを含む一回の保存

親と複数子を画面上で追加・変更・削除し、一回の保存で全体を確定する。親が複合主キーなら各子は**親の全キー**で所属を判定する。親合計は削除予定行を除く全明細（未変更行も含む）から再計算し、子が0行なら明示した規則に従う。採番、入力途中の計算、保存時の再計算を混同しない。

TEIでは汎用多階層SmartDB保存を主線とし、`prepare → authorize → validate → derive → commit → receipt`の順序と、原子的に扱う範囲を受入契約に記す。外部EffectはDB transactionと独立し、commit成功時のみ配送要求を発行する。局所的な画面プレビューが保存の証拠にならないようにする。親子一括保存が失敗した場合、正常な親だけ残す振る舞いを既定としない。

### 3.3 「未入力」「送らない」「nullで消す」は別

Wagbyの入力不可項目は、フォームに入力途中の値が残っていても更新対象から除き、保存済みの値を保持する。TEIでは次を区別する。

- 値が存在しない（未取得・初期値未設定）、明示的な`null`、空文字、未変更。
- 画面に表示されない、編集できない、送信しない、更新許可がない。
- 保存前のDraft値、Providerへ送るPatch、DB確定値、閲覧用Projection。

**表示状態から更新可能性を推論しない**。無効フィールドの値を攻撃者が直接APIへ送ってもHost側が拒否または安全に無視する。UIとCLI/MCP/Importで同じ基本検証を通し、媒体固有の入力補助だけを分ける。文字数・バイト数・表示幅、型付きDecimal、日付、必須、項目間比較、警告とエラーは独立Expectedを持つ。

### 3.4 検索ビューと認可を分離する

名前付き検索ビューの固定条件は利用者の追加条件で置換できない。ただし検索条件を固定しても、閲覧可能範囲を限定したことにはならない。TEIのLINQ風Queryと階層／グラフ／RDBの同型操作では、Query、表示用Projection、AuthZ適用を区別し、API/CLI/GUI/MCPで漏洩がないことを確認する。

部門・配下部門と自己所有、CRUD操作別の範囲、保存済み状態による項目の非表示／マスク／読取専用を試験対象とする。非表示はレスポンスへ含めない、マスクは必要な範囲で置換する、といった区別はPresentationでなく開示境界の契約に属する。文書・ファイル全文検索の索引、集計結果、ダウンロード、エラーメッセージでも開示制御を迂回しない。AuthZEN相当の外部AuthZ問い合わせ候補を残すが、現在の資格を確認しないまま過去Grantを継承しない。

### 3.5 ワークフローの人数条件・委任・時点

承認には順次／並列、誰か一人／全員／N人中M人、条件による段階省略、差し戻し、却下、再申請、取下げがある。**人数の充足**と**誰がその承認を行う資格を持つか**は別。候補者集合を確定した時点、組織改編・離職などの有効時点、現在の実行許可を分ける。代理承認は元の責任者と実Actorを別々に履歴へ残し、委任で閲覧・編集権限を勝手に拡張しない。委任の循環や自己承認、同時操作競合を負例にする。

TEIは業務行項目を第一候補の状態保持場所とし、履歴・承認待ち・操作記録を必要な範囲で別管理する。汎用状態表や独立Rule Runtimeの導入は前提にしない。Owner受理、現在Grant、業務状態変更、Effect Receiptの四者を別に確かめる。

### 3.6 一括取込と復元の失敗モデル

ファイル取込は検証のみ（VALIDATE_ONLY）、全件原子反映（ALL_OR_NOTHING）、正常行だけの行単位反映（VALID_ROWS）を区別する。部分反映後の再実行では、前回どのキーに何が確定したかをreceiptとoperation identityで追う。文字コード・BOM、元の主キー維持、採番同期、参照整合、上限、エラー行の再実行条件をテストする。

設計情報のリストア、同一Schemaへの完全復元、異なるSchemaへの業務データ移行を別操作にする。現行の権限・利用者・秘密情報を古いバックアップで上書きしない。World Fingerprint、Model Revision、Artifact Revision、DB Schema Revision、Migration Planと実DB適用結果は別の資格である。逆移行の可否や部分失敗時の復旧計画を明示する。

## 4. 第二優先：業務アプリの使い勝手を標準能力にする

| Wagbyで確かめる粒度 | TEIでの展開 | 製品受入前の前提 |
| --- | --- | --- |
| CRUD、確認画面、コピー登録、論理削除、複数行一覧更新 | DataとOperationからUI/CLI/APIを共通生成。確認はUI、保存は業務操作の責務 | 生成画面だけでなく実DBで成功・拒否を確認 |
| 初期値、条件付入力、サジェスト、住所補完、トレースリンク | 標準入力・検索ComponentとRelation参照。初期値は適用時点・対象経路を限定 | UI初期値と全経路の保存時計算は別 |
| 集計、名前付き検索、カレンダー、タイムライン、地図 | Query＋Projectionから複数表示を派生。時間帯・設備・住所などの意味を残す | 集計の計算、表示、アクセス権は別 |
| CSV／TSV／XLSX／PDF、親子帳票 | 同じ業務QueryとResultから出力。様式・Target特性はGenerator/Templateへ | 帳票のレイアウト・日本語・ファイル内容を実Consumerで受領 |
| ロケール別・運用変更可のメール、送信管理 | TemplateとOutbox Operation、試行・再送・取消・監査を分離 | DB保存成功＝送信成功と見なさない |
| メニュー、権限別初期画面、ガイド、ツアー | Application Graph／Role／Operationから誘導と操作文書を生成 | ガイドに書いた操作が実在することを検査 |
| Web・Mobile・Desktop・MCP | 共通Application API/Operation。Medium専用Adapterで実現 | UI用権限表示やMCP説明を実行Grantの代用にしない |

帳票でWagbyがJRXMLの登録を別途必要とすることは、TEIに「PDF・Excel出力の型付き契約は共通、実レイアウトはTarget Generator」とする境界を与える。メールのOutboxは配送の実効果を監査し、業務Transactionや再試行から独立して進捗確認できるようにする。MCPは標準Operationへ到達する一つのインターフェースであり、MCP専用の業務意味や独立Hostを設けない。

## 5. 第三優先：AI共同開発とStudioの完成度

Wagbyの公開マニュアルは、**実現したい機能→AIへの短い指示／詳細指示→AppSchemaの例→参考guide/reference→生成・起動・確認**をページ単位で対応させている。TEIにとって重要なのはプロンプト例の数ではなく、指示の出典と変更対象を現在のModel/Source/Providerへ結び付けること。

- 業務設計の説明、Source編集位置、意味受理、実生成、Target差分、実操作、検収結果を同じSubject/Revisionで追跡する。変更候補はOwner受理に昇格しない。
- DSL/Graph/Markdown表を往復編集するとき、表示Labelや描画Edge IDではなく安定したRelation occurrence IDと元Byte範囲を使う。自動整形は糖衣表現を不必要に書き換えない。
- マニュアル・操作ガイド・AI Skillは、選択Pluginの公開文書・Contract・上位依存・適切なDisclosureに参照で接続する。文書閲覧の可否とProviderを起動する資格は別。
- 画面の診断でファイル・位置・業務ID・制約ID・参照元／先を示す。ソース側修正を基本とし、生成Goを直接修正して追跡不能にしない。
- 編集直後診断→プレビュー→生成→外部Hostコンパイル→実画面・CLIの順を測定し、遅延の原因と独立Expectedを区別する。

WagbyのAppSchema履歴は設計のまとまりごとに記録し、製品版と配置版を分ける。TEIでは既存のSource/World/Model/Artifact/Bundle指紋、Build AttemptとGit履歴を使う。新しい履歴正本を設けず、構造の意味変更・表示だけの変更・Plugin更新・Source A→B→Aを別の失効原因として評価する。

## 6. 実装に落とす代表シナリオ（受入の土台）

題材は「受注→明細→承認→帳票・通知・移行」を一つの小さな `*.tei.md` から段階的に構成する。以下は**仕様・テスト候補**であって、新しいTEI構文や実装済み機能の主張ではない。

| ID | 操作・反例 | 独立期待 |
| --- | --- | --- |
| W11-01 | 複合キーの親＋子2行を新規登録 | 親の全キーが子に結び、全体が一回で確定 |
| W11-02 | 子の入力失敗／競合、別の子は正常 | 原子保存モードでは親子すべて未確定 |
| W11-03 | 明細の追加・削除予定と未変更行 | 金額・税・親合計を保存直前に再計算、削除予定行を除く |
| W11-04 | 商品価格変更後に既存受注を表示 | 取引単価Snapshotは不変、現在値Projectionだけ更新 |
| W11-05 | 参照元キー不変・中間参照先が変更 | 固定キー・保存値・非保存の現在参照を混同しない |
| W11-06 | UIで入力不可へ切替、古いDraft値を保持 | 送信対象から除外し、既存DB値を保持。null消去しない |
| W11-07 | APIで読取専用値／認可外の列を改変 | UIと無関係にHostの現在権限・制約で拒否 |
| W11-08 | 「自分の未完了」ビューへ矛盾条件を追加 | 固定条件を上書きせず0件。権限外は他経路でも漏れない |
| W11-09 | 部門兼務、配下組織、異動前後、集計と全文索引 | 有効時点の読取範囲を維持、権限外件数・添付も漏らさない |
| W11-10 | 3候補者のうち2人承認、途中で資格喪失 | 候補・人数・現行資格を分け、未達なら進行しない |
| W11-11 | 期限付き代理、同時操作、自己承認・循環 | Actor／委任元の履歴を別記録、不正な経路は拒否 |
| W11-12 | 行エラーを含む取込を3処理モードで実施 | 全件中止・正常行のみ・検証のみの効果が区別される |
| W11-13 | 部分反映後、同じファイルを再送 | operation identityとreceiptで重複副作用を検出・制御 |
| W11-14 | 受注commit後のメール送信失敗 | DB commitは維持、Outboxで再送/取消/失敗履歴を追う |
| W11-15 | 受注と明細のPDF/Excel出力 | 値・権限・日本語・様式・実ファイルを独立Consumerで検証 |
| W11-16 | 定義変更後の業務データ移行 | Schema差分と権限保護を検査し、古いGrantを復活させない |
| W11-17 | CLI、Web、MCPから同じ業務操作 | 同じMeaning/Validation/AuthZを通し、各Adapterの実受領を分ける |
| W11-18 | Sourceの説明のみ変更／関係意味変更／A→B→A | 影響粒度に応じて再検証し、古い意味受理や実行資格を自動復活させない |

最小の第一縦断はW11-01〜04に絞る。W11-05〜18は同じテストデータ・独立Expectedを伸ばして検証する。全面的なWagby互換を宣言しない。

## 7. 既存Ownerへの引渡しと順序

| 優先 | 成果物・判断 | 引渡し先 | 完了の主張に必要な証拠 |
| --- | --- | --- | --- |
| P0-A | 正式`*.tei.md`一件を現在Workspaceから選択・読取し、Data/Relation/Ruleの同一Source byteを通す | #2621、#1018、#988 | 正式Readerの選択とSource位置、拒否、現行Revision |
| P0-B | 受注親子・取引時点単価・合計の一部を標準Data/Function/Operatorで実Go生成し、通常CLIからProviderを利用 | #4495、#1129、#1637、#2586、#1251 | 実Generator、actual Build、実DB結果と独立Expected。Mockだけでは未達 |
| P0-C | 同じ製品基盤の一部をTEIで生成し、Studio Stage1→Stage2 Windows WebView2で実受領 | #2831、#4583、#1824 | 同一AttemptのStage1実行・Build閉包・Stage2実exe・画面 |
| P1-D | 複合キー・親子同時保存、Query View、入力Patch、項目制約の負例 | SmartDB／Data/Query／#1180／#2514 | API/CLI/UI/DBを跨ぐ結果、原子性と拒否 |
| P1-E | 組織スコープ・委任・人数条件・現在Grant・項目Mask | #2386、#4490、認可/Workflow Owner | 時点別Identity、Role、Actor、公開の独立検査 |
| P1-F | 取込3モード、Outbox、帳票、移行と運用診断 | 関連Target Generator／Effect／Migration Owner | 実ファイル、実DB、実Outbox、移行結果とreceipt |
| P2-G | Mobile/Calendar/Map/全文索引/運用ツアーなどの広範な標準化 | Presentation/Query/Tooling Owner | 対象Targetの実Consumerと権限、機能別測定 |

既存Issueを引渡し先に使用し、未調査の所有範囲を新しい専用Engine・Gateway・認可Storeとして増やさない。各Ownerは仕様候補の採否と既存テストへの取り込みを判断する。新機能の検討でP0の実Entry、生成物の実消費、セルフビルドの合格基準を引き上げない。

## 8. 比較を継続するための物差し

- **標準記述率**：一つの業務シナリオの処理・画面・制約のうち、TEI標準構造と型付きOperatorで表せた範囲。標準化のために不自然なDSL記法を増やしていないかも評価する。
- **手書き実装の削減**：外部ライブラリ呼出しや薄いTarget Adapterと、業務固有ロジックの手書きを分けて測る。生成Goの行数が多いだけでは悪化と判定しない。
- **意味と効果の一致**：Source→受理済みMeaning→実Generator→実Provider→DB／UI／帳票／送信結果の差を独立Expectedで確かめる。
- **編集・確認の費用**：業務変更1件に対するDSL編集量、Studioの再診断・再生成の時間、Target別回帰・移行の範囲を実測する。
- **負例による信頼性**：権限、時点、複合キー、部分失敗、二重送信、参照更新、A→B→Aで期待どおり止まるか。
- **セルフビルド比率**：StudioとPlugin自身の実装にTEI生成部分が占める割合、および定義の簡素化。単なるGo delegateでは成功としない。

## 9. 現時点の結論と非採用事項

Wagby 11の強みは「現場の業務アプリを成立させる機能が細部まで体系化され、AIへの依頼と設定・確認が対応している」ことにある。TEIは、その完成度を目安にしながら、意味の再利用、複数Targetへの生成、Sourceと実行の証拠、セルフビルドという異なる価値を測る。

この文書では、Wagby 11のAppSchema/YAMLをTEI DSLへ移植しない。Wagby互換ランタイム、専用Generatorの全面再実装、汎用ワークフロー状態表、第二のBRMS、独立のMCP業務Host、別のSource Registryは採用しない。必要な能力は既存TEIの意味モデル、Operator、Plugin、Target Generator、SmartDB／Operation、Evidenceへ統合する。公開マニュアルの読解、設計統合、製品実証、Ownerによる最終受理は別段階である。

## 10. 重複実装を避けるアウフヘーベン：意味の責務を一つに、実現を複数に（2026-11-15）

### 10.1 機能を移植するのでなく、既存の責務へ還元する

Wagbyの機能一覧は利用者が目的の機能を探すための分類であり、TEIの内部構成を定義する分類ではない。例えば「入力チェック」「計算」「ルックアップ」「保存前処理」「承認条件」は異なる製品メニューに掲載できても、内部では同じ型付きRelation、Operator、DerivationまたはPredicateを利用し得る。一方、**条件式が同じというだけで**、入力値の妥当性・遷移条件・実行許可・DB確定を一つの責務へ統合してはならない。

この検討は新しい「Wagby互換機能群」や第二のMeaning Catalogを設置する提案ではない。既存の標準データ構造／Meaning Role／Facet、Data／Relation、Function／Operator、Query、Application Operation／Lifecycle、Host Authority、Presentation／Target、Provider／Build／Evidenceの**役割分担の確認とGapの引渡し**である。

- **一つの業務意味には、受理された定義元を一つだけ持つ。** 別のMarkdown表、Graph、フォーム、SQL、Go、Plugin説明はSource-backedな参照／投影／実現とし、独立した第二正本にしない。既存Sourceの選択・復元・Owner受理は#2621/#2497/#267に従う。
- **複数の意味の用途を、無理に一つの責務へ押し込まない。** 同じ「金額が正である」をData Constraint、受注受付Guard、Policyと結び付ける場合は、式と必要な型付き入力を再利用するが、適用時点・母集団・失敗結果・Authority・副作用を用途ごとに区別する。
- **標準構造に収め、専門機能は既存要素の組合せを第一候補にする。** 既存要素で表せないことを反例で示したときだけ、該当OwnerのTrait／Operator／Provider／Target Adapterを最小拡張する。機能名に合わせてParser、Runtime、Store、独自Kindを増やさない。
- **一つの業務操作に複数の入口を接続できるが、実効果の入口は一つの契約から生成する。** CLI／Web／Studio／MCP／バッチ、画面保存・ワークフロー承認・Nodeフローは共通Application Operationを使う。UIやNodeに業務書込みコードを重複保持しない。外部Hostの実GrantとReceiptは呼出しごとに検査する。
- **共通化の評価には、表現可能性だけでなく意味保存・実Consumerを含める。** DSL文字数の削減だけで決めず、Source→型付きMeaning→Targetの結果、変更影響、失敗・未知・未取得・NULL、現在権限、実Provider／DB Effectを比較する。

**説明用の判定単位**は「Subject、受理された述語・導出／関係、適用先、粒度、時点、入力の完全性、失敗時の扱い、必要なAuthority、Effect境界」とする。これは既存契約を読み取り専用で照合する項目であり、汎用の新しいCanonical IDや製品Gateを導入しない。同一目的らしく見える二つの定義について、これらが等しいなら**二重定義の疑い**、異なるなら**必要な責務の分離**と説明する。自動判定だけで片方を削除しない。

### 10.2 Wagby機能の「意味上の受け皿」を固定する

| Wagbyでの機能・見え方 | TEIの既存責務（定義／意味のOwner） | 実現・投影する側 | 新設・重複させないもの |
| --- | --- | --- | --- |
| モデル、項目、親子、主キー | Data／Relation／Identity／FD・多重性 #1126/#2495 | SQL Schema、フォーム、ER、Graph表示 | 親子CRUD専用データ型、画面用の別Identity |
| 同じ物理表を複数モデルで利用 | 一つのDataと複数Query／Projection、物理Mapping | Provider／Target Adapter | 同一物理表に対応する複数の業務正本 |
| 複合キーの中間表・多段参照 | Relationの端点・役割・向き・全キー・多重度 | Query Join／参照Picker | 専用の別Relation体系 |
| ルックアップ、保存時点の値 | Relation／Derivationの入力と採用時点、Dataの保存属性 | Query／フォーム／Data Operation | Lookup式Engine、表示値の自動DB書込み |
| 初期値、保存時計算、表示時計算 | 適用先と時点を分けたDerivation／Operator #1637/#1640/#1641 | Form初期化、Save Operation、Query Projection | UIごとの計算正本、初期値専用式言語 |
| 必須・範囲・文字列・項目間チェック | Data Constraint／Validationと既存型付きOperator #988 | 入力中診断、Operation検証、DB制約 | フォーム専用の規則正本 |
| 画面入力不可・非送信 | Operation入力Patchの有無／更新契約とUI制御 | Presentation／Client Adapter | 「非表示なら書込禁止」という擬似認可 |
| 名前付き検索・集計・全文検索 | Query／Projectionと結果粒度・取得Scope | SQL／NoSQL／索引Provider、一覧・Chart | 検索画面専用Query意味体系 |
| 部門・所有者・項目権限 | Organization／PolicyとHostの現在AuthZ | Provider／API／UIの開示と実行境界 | Query FilterまたはCSSによるAuthZの代用 |
| CRUD、親子保存、複数行更新 | Data Operation／Function、Constraint、Transaction・Effect | SmartDB処理とDB Provider | CRUD Engineと独立「保存前後Engine」の併存 |
| 申請・承認・代理・人数要件 | 業務Lifecycle状態・Operation・組織時点・Policy | 標準Process Flow表示、通知Trigger／Provider | 独立Approval Rule Runtime・万能状態表 |
| UIイベント・HTTP・タイマー・MCP | Trigger→型付きInvocation→Application Operation #1158/#4495 | Target別Transport／Host | 入口媒体ごとの業務Operationの複製 |
| 一括取込、部分成功、再実行 | Data Operationの集合処理・Effect・Receipt、Schema Mapping | Import Adapter、Data Provider | 別のImport業務意味／Receipt Store |
| メール送信、帳票生成 | Operation結果と通知／出力Effect、Query Result | Template、Outbox Provider、PDF/Excel Generator | メール・帳票側の業務判定／書込み正本 |
| Nodeフロー、ダッシュボード、ガイド | Operation／Function／Graph／Queryの利用と表現 | Node Editor、UI Kit、説明文生成 | Node専用業務DSL、独立ワークフローDB |
| AI指示、マニュアル、MCP公開 | Source-backed Model／Plugin Contract／Disclosure | Studio／SHO／Harness／MCP Adapter | Prompt文を新しい業務正本や実行Grantにする仕組み |

この表のOwnerは**新しい組織上の担当者名ではなく既存設計の責務位置**を示す。複数の行に同じ要素が現れるのは、その要素の再利用であり、同一意味の複製ではない。Queryの表示用結合と保存時Snapshotのように意味が異なる場合は別の利用先を持つ。

### 10.3 衝突しやすい六つの境界で何を統合し、何を残すか

**A. ルックアップ／初期値／導出**：商品マスターの現在単価を表示するQuery、受注明細へ転記する契約単価、当該単価×数量の金額導出は同一ではない。参照元Relationと型付き演算子は共有し、取引時点の単価はDataの保存属性に受理された意味として残す。表示用の値を保存時計算へ自動転用しない。反例は「マスター価格変更後も受注保存単価が不変」。

**B. 入力チェック／業務Guard／AuthZ**：「数量 > 0」を一つの型付きPredicateとして参照できる。しかし「画面で警告する」「保存を拒否する」「承認状態を遷移させる」「担当者が更新できる」は異なる適用契約。ValidationとRuleを新しい二つの式文法にせず、既存Operatorと適用先の責務を分ける。現在AuthZは業務Guardだけで許可済みとしない。反例は「CLIで画面Disabled項目を送っても現在Grantに反する更新は拒否」。

**C. 画面入力制御／Patch／保存**：read-only UIの入力値が送られないのは、PatchのPresenceとUIからの送信構成の問題。明示NULLで消す、未送信なので旧値を保つ、取得できていない、非開示のため表示しない、という状態を区別する。保存操作は現在の型・制約・権限で再検査し、UIのdisabled情報を信頼しない。反例は「未送信がNULL更新に変換されない」。

**D. 検索条件／開示範囲**：検索ビューが組織条件を持つ場合でも、Queryの母集団指定とHostのAuthZは別。設定した見た目の条件を変えずに別入口で検索した場合も権限外情報が漏れないようにする。必要ならQueryへPolicy由来の制限を実装上Pushdownするが、Filterが存在する事実を許可の証拠としない。反例は「全文検索のヒット件数・集計・添付でも権限外を漏らさない」。

**E. ワークフロー／Process Flow／Nodeフロー**：承認・差戻しなどの**業務状態遷移**は業務Lifecycleの意味であり、HTTP待受け／ジョブ起動の**Process Lifecycle**や、Node配線の**処理グラフ**とは異なる。Process/Nodeは同じApplication Operationを呼び、承認ルールは既存Rule/Guardから導く。承認人数や代理は組織・時点・Actor/責任者の適用条件であり、Nodeの配線順を業務承認の正本にしない。反例は「同じ承認OperationをUIとNodeから呼んでも人数判定・AuthZ・監査が一致」。

**F. データ取込／SmartDB保存／通知**：全件原子、行別反映、検証のみの違いはData OperationのTransaction/Effectの実行方針であり、データ取込専用の業務エンジンを意味しない。取込Adapterはファイル解釈と行ごとのSource位置を受け持つ。保存後のメールや外部連携は別EffectのReceiptを持ち、DB Commit成功と配送成功を同一にしない。反例は「部分反映後の再実行で重複業務Effectが生じない」。

### 10.4 既存の責務への収容方法を審査する（別の審査Engineは作らない）

新機能候補ごとに、既存の設計Review／Issue／独立Expectedを使い、次の順で判断する。

1. **利用者が期待する業務上の結果を記す**：「条件付きルックアップ」ではなく「保存した受注金額は後から商品マスターが変わっても変化しない」など、観測可能な要件を先に書く。
2. **既存の意味参照に還元する**：Data／Relation／Query／Derivation／Constraint／Function／Operation／Policy／Lifecycle／Projectionのどれの意味か、型・粒度・時点・Effectから判定する。表示カテゴリ名だけで決めない。
3. **同じ目的の既存定義を列挙する**：Source参照と安定IDでたどり、重複候補、用途を分ける必要がある候補、単なる表示差を分ける。検索結果0件を「既存機能なし」の証明にしない。
4. **既存契約の再利用を第一案にする**：既存Operator/Function/OperationをTargetやUIから参照する。欠けている能力があれば、最小の適用先・Trait・OperatorまたはProvider適合へ差分を戻す。
5. **重複定義の新設を棄却する**：新しい形式が不可欠なら既存Meaningへの糖衣構文・読取Adapter・Target Generatorとして実装し、別の実行規則・保存場所・認可根拠を増やさない。変換できない意味はLoss/Unsupportedで明示する。
6. **二方向の影響を確認する**：同じSourceをUI／CLI／MCP／Batchで利用し、また同じMeaningをGraph／表／説明文／Target Goへ投影したとき、意味が増殖しないことを確かめる。
7. **既存Ownerの受入と実Provider観測を分離する**：Source候補・型付き意味・業務受理・Builderの実消費・実DB/Consumerは段階別の資格。新しい横断Gate／Owner／Receipt Storeを追加しない。

判断の返却先は既存の#2495/#2497/#2620（標準データ構造と意味）、#988/#1637（Operatorと導出）、#2622（Data Operation）、#4198（Lifecycle Graph）、#1158/#4495（Application Operationと実起動）、#2386/#4490（組織）、#951/#3278（Plugin/Trait）、#1180/#2514（Scenarioと独立Expected）を優先する。正式なOwner変更は各Issue側で判断する。

### 10.5 アウフヘーベンが成立したかを確かめる反例

| ID | 故意に作る衝突・不整合 | 期待する判定と回復先 |
| --- | --- | --- |
| W11-AH-01 | 同じ受注明細金額規則をフォーム計算と保存前Ruleへ別IDで複写 | 重複候補を提示。既存Derivation/Operatorへ共通化し、適用時点だけを別契約へ |
| W11-AH-02 | 保存単価と現在参照単価を「同じLookup」として併合 | 時点・正本属性の違いを維持。Transaction Snapshotを破壊しない |
| W11-AH-03 | 入力必須チェックと承認Guardを条件文字列だけで同一視 | 共通Predicateは利用可能だが、失敗結果・適用先・Authorityを別扱い |
| W11-AH-04 | 画面のread-onlyを解除してAPIから権限外列を更新 | Hostの実AuthZとOperation契約で拒否。画面設定はGrantにならない |
| W11-AH-05 | 名前付き検索条件を外し、同じQueryをMCPから呼ぶ | HostのPolicy/Disclosureを維持。表示結果の件数からも秘匿情報が漏れない |
| W11-AH-06 | 同じ業務承認をフォームボタン・Nodeフロー・CLIで呼ぶ | Operation/Guard/現在Grant/Receiptを共用、媒体別業務実装を作らない |
| W11-AH-07 | 複合キーの一列だけで参照、または表示用Graph IDを関係Identityへ代用 | Relationの全キーと安定参照・多重性を要求。誤結合を拒否 |
| W11-AH-08 | 同じ物理表に対応する参照用Projectionを編集してDB更新 | 明示的な可逆更新契約がなければread-only。第二のData正本を作らない |
| W11-AH-09 | 取込AdapterとCRUD保存でそれぞれ異なるValidationとDB処理を使う | 共通Data Operation/Constraintを使い、Import固有部分はParse/Source位置に限定 |
| W11-AH-10 | 初回成功後メールだけ失敗、同じ操作を再実行 | DB Commitと通知Effectの各Receiptを分け、二重更新・二重配送を防止 |
| W11-AH-11 | 設計説明のみ変更してコード生成まで失効、または意味変更を失効させない | Source／Meaning／Artifact／Provider依存に応じて失効範囲を区別 |
| W11-AH-12 | Pluginが既存Operatorと同名・異なるNULL/精度規則を黙って選択 | Contract/Version/法則の不一致として診断。Plugin順による意味変更を禁止 |

これらは**独立Expectedの追加候補であり、試験済みではない**。#1180/#2514の既存Scenarioに参照して取り込み、W11-AH専用の検証Engine・第二Test SSOTを新設しない。「複数の実装が使える」のが正常な場合（例えば同じQueryをSQLとNoSQL Providerで実現）は、契約が同じであること、Targetの制約、結果とエラーの差を明示した適合試験で認める。

### 10.6 優先する縦断検証：既存のW11-01〜04を深化させる

最初から12件を新しいP0にせず、既存の受注・明細・商品で次の**1件の業務縦断**に絞る。

1. **Meaning**：商品と受注・明細Relationを確定し、取引時点の単価Snapshotと現在の参照単価を異なる意味として表す。数量×採用単価の一つのDerivationを定義する。Source内の見た目上の同名を同一性へ昇格させない。
2. **Source/Reader**：正式選択された一つのSourceから型付き候補とSource位置を取得する（#2621/#1018）。新しいWagby特化Authoring Readerは作らない。
3. **Build/Provider**：選択済みOperator→実Go生成→Builder実消費→通常tei実Providerを一つのFunction/Operationで通す（#4495）。先にParent-child DB commitが必須とならない最小Read/Derivationから始める。
4. **同一規則の別媒体投影**：可能な範囲でUI表示・CLI結果・Query結果から同じDerivation/Relation参照を比較し、別の式実装を増やしていないことを確認する。実Consumerに未到達ならNOT RUN。
5. **SmartDB保存・失敗・拡張**：複合キー／親子原子保存、未送信≠NULL、二重実行の反例を既存Data Operation/Effectへ段階接続する。DB commit／通知／AuthZを未観測のまま成功扱いにしない。
6. **Studioセルフビルド**：#2831/#4583の既存Stage1→Stage2 Windows製品P0を並行継続する。受注保存の全面完成をStudio受入の前提にしない。

成功の指標は、(a)同一Meaningを複製せず複数媒体／Targetへ渡せた割合、(b)新たなSemantic Kind・Parser・Runtime・Store・DSL構文・個別保存ロジックが**不要だった数と根拠**、(c)役割の異なるConstraint/Guard/Authority/Effectを誤統合しなかった反例、(d)定義変更一件に対する実Generator・Consumer再検証の範囲である。単なる機能数、短いDSL、モックのPASSだけを合格にしない。

### 10.7 当面採用しない統合方式

- Wagbyの画面分類に合わせたWagby専用Core機能／「Wagby Plugin標準セット」の導入。
- 全条件式を一つの万能Rule Engine、全状態を一つのWorkflow Engine／状態表へ格納する方式。
- Data／Query／UI／Policy／Nodeごとに同名の業務式言語を増やす方式。
- 表示用GraphやMarkdown表を、同じ意味を持つ別の編集正本として常時同期する方式。
- Nodeフロー／MCP／AIがその場で実Providerや実行権限を新しく確定する方式。
- 共有テーブルを意味的に異なるDataの乱立で表し、DB物理テーブル一致をIdentity同一視する方式。

必要な業務の違いは削除せず、異なるFacet／適用先／時点／Effectと明示して残す。**アウフヘーベンは単なる統合や簡略化ではなく、既存の実務機能を保ったまま、意味の重複と責務の混乱を減らすこと**として評価する。


## 11. X-TEAのFunction独立性を軸にした再配置：Data・Function・BusinessとTable Script（2026-11-15）

> **位置づけ**：利用者の追加指摘を受け、前章の「意味上の受け皿」のうち、データに機能を従属させるように読める箇所を精密化した設計・試験候補。以下の名称は既存のTEI Function Data Context／Data Operations／Data Derivation／Process・State／Pluginの責務を説明するもので、新しいCore Objectや文法を採用する決定ではない。X-TEA原典の実際の実装挙動は#940/#1125で照合する。旧来の資料や対話由来の理解だけで原典仕様を確定しない。

### 11.1 まず存在論を固定する：DataにFunctionを従属させない

Data Modelは「存在するもの、識別、属性、関係、制約」、Function Modelは「実現すること、入出力、対象Context、操作・表示」、Business Modelは「なぜ、誰が、いつ行い、どの状態・責任へ至るか」を別視点から記述する。三要素の関係はTyped Referenceで追跡し、**Data→Functionの単方向包含・自動所有ではない**。Functionは複数Dataを統合しても、一つも業務DBに触れなくても成立する。Business ProcedureはFunctionや手作業・外部作用を参照するが、Function成功を業務完了へ自動昇格させない。

- X-TEAを参考にした既存のFunction分類（Action／View／他Function呼出し等）は維持する。Functionを抽象的な単一万能Objectに押し込めず、既存#2506/#2507/#1158が保持する用途と型付き参照で扱う。機能の同一性はUI画面ID、DBテーブル、Queryファイル名とは異なる。
- Data-firstの即時CRUD／検索は、Dataと利用可能なOperationから生成する**初期利用向けの導出機能**と位置付ける。明示Functionが存在する環境で、それと競合する別の更新規則・権限・意味の正本を自動生成しない（#1127）。Dataしかない段階と、Functionが明示された段階の共存・置換は検証対象である。
- Function ModelのViewは「機能が必要とする情報を利用者へ提示すること」、Query／SQLのViewは「取得・変換されるデータ集合」、PresentationのViewは「描画・操作形式」を指す。**同じ『View』という言葉でも異なる責務**であり、SQL VIEWや表示部品を新しいData Modelと自動同一視しない。
- 既存のData/Function/Businessの参照は関係として保持し、Relationが存在するだけでFunctionの入出力集合を暗黙拡大しない。Functionが要求するScopeと、Providerが実際に返したScope／Completenessを区別する。

### 11.2 Wagby 11で見える機能をTEIの各モデルと実現機構へ割り付ける

| Wagbyでの設定・機能 | 第一の意味上の責務 | Functionから使う要素／実現方式 | 排除すべき重複 |
| --- | --- | --- | --- |
| データ型、主キー、項目、親子・多対多 | **Data**の型・Identity・Relation・FD | Table Scriptの局所Rule、DB Providerの物理Mapping | Tableごとに機能View/業務Operationを所有させる |
| 項目拡張定義、仮想項目、書式 | **Data**の値域・存在・保存性・型とFacet | Derived RuleへのID参照、Presentationへ表示型を委譲 | 項目に埋めたDSLとTable Scriptの二重実行 |
| 取得時・保存時の計算、入力チェック | **Data**のConstraint／Derivationの適用先と時点 | 型付きOperator、Table lifecycle Rule (#1129/#1637) | 画面・SQL・Tableごとの別式意味 |
| 検索条件、名前付き検索、複数一覧 | **Function**のView／入力・結果Context | Query/SQL、Relation Path、Function Data Context (#2506/#2586) | 画面の数だけData Modelを複製 |
| 横断検索、複数表の集計、ダッシュボード | **Function**の集合・集約・粒度・表示用途 | Query/SQL/2waySQL・型付きOperator、Chart等のProjection | Query/GroupByをデータモデルの属性へ押し込む |
| 一覧・詳細・カレンダー・マップ | **Function View**が求める情報・選択・操作 | 共通ContextのPresentation／UI Kit／Target | 表示器へ業務意味・行Identity・書込み資格を複写 |
| 登録・更新・承認・取消 | **Function Action**とBusiness側の状態・責任 | Data Operations／State Transition／実Provider | Data CRUDへの全機能の従属、承認専用DB意味モデル |
| イベント、承認経路、代理、業務手順 | **Business**のProcedure／Event／State／Actor | Trigger→Function Entry／Operation、標準Process Flow | Process図のNodeとFunction実体の無条件同一視 |
| メール・帳票・MCP・Batch | 既存Function／Operationの結果・Effect | 選択Adapter／Template／Target Generator | 媒体ごとの第二のFunction/Rule/Authority |

Wagby 11の[共有テーブル](https://wagby.com/manual11/relations-shared-table.html)は、TEIでは主として**一つのDataと複数Function Context／Projection**で吸収できる。一方、物理テーブルと複数の本当に異なる業務実体が対応する場合は、それぞれのIdentity/RoleをDataとして明示する。物理Tableの一致だけから同じ業務実体とも別実体とも断定しない。

### 11.3 Function Contextを中心に検索・ビュー・SQLを構成する

Functionは、利用者から見た入出力契約と必要なデータ集合を宣言する。既存のFunction Data Context（Definition／Instance／Result Slice）の想定に沿って、Query／SQLから得る列・Relation・複数Dataset、粒度、Identity、時点、取得条件、必要範囲、完全性、参照元、現在Authorityを明示する。

1. **検索機能**：Function Viewが必要な結果集合、検索入力、固定条件と追加条件、sort/pageを定める。Queryはその要求を評価し、Providerで実行する。名前付き検索は別Data Modelではなく、同じFunctionへの特定の入口・Context Patternである。
2. **複数検索・集計・グラフ**：Dataを増やさず、一つ／複数のFunctionで複数Contextを利用する。QueryのJOINで明細行が増幅されてもEntity個数と同一視しない。Aggregation後のResult Grainと編集元Entityは別。
3. **SQLとDSLの分担**：通常のRelation Traverse／Filter／Aggregate／Sortは既存型付きQuery/Operatorを第一候補とし、複雑なSQLやProvider固有のクエリはNative SQL Sourceを保持する。SQL→標準表現への変換はQuery解析・診断・Target再生成の範囲を明示し、Native SQLとDSLの双方を同時に編集する第二正本にしない。2waySQLは別の新Query EngineでなくNative SQL Bindingと適用部分の検証案として扱う。
4. **Function内の表示・編集**：Table/Form/Chart/Map等はContextの異なるProjection。ユーザーの変更から実データへの更新Intentを導けるかは#2500で評価する。非可逆な集計棒の値を、特定の明細へ勝手に逆写像しない。
5. **認可**：固定検索条件はAuthorityの代替でない。ProviderとHostで現在のRead/Write/Disclosureを判定し、Query OptimizerにPolicyのPushdownが可能でも「Filterがあるから許可」と推定しない。

**重要な性能・正しさの制約**：Table Scriptが導出する仮想項目をFunctionのFilter／Sort／Group／Limitで使う場合、取得した一部の行に対して後からスクリプトを実行するだけでは正しい結果にならない。#1640の既存Query計画がDerivationの依存と順序を認識し、SQL Pushdownが意味保存できれば利用する。できない場合は必要範囲を完全取得してから評価するか、\`unsupported\`にする。Wagbyの「表示時計算項目は検索条件・ソート項目にできない」という制限は、TEIの機能境界としてそのまま採用するのではなく、**導出値を検索可能にするための計画・性能・完全性の検証課題**として扱う。

### 11.4 項目拡張定義には意味と参照を、Table Scriptには処理を集約する

推奨する初期方針は、利用者が挙げた**後者**を基本とする。ただし「すべてをTable Scriptへ入れる」ではなく、**Dataに局所的な処理はTable Script／Data Rule、Function固有のデータ取得・変換はFunction Context／Query、複数箇所で再利用する純粋計算は型付きOperator／共通Function、業務状態変更はFunction Action／Business側**と役割を切る。

| 置き場 | 記述すること | 記述しないこと |
| --- | --- | --- |
| Data／項目拡張定義 | 型、単位、値域、NULL、保存／仮想、表示向けFacet、業務上の属性ID、導出・検証Ruleへの**安定参照** | 実行可能な汎用DSL本体、イベントの順序、任意の外部Effect |
| Table Script／Data Rule | 行／Table局所の導出、検証、入力正規化、old/draft/new、read／change／before-writeなどの適用Phase、依存・読み書き集合 | 機能全体の画面・Query集合、組織承認手順、外部APIのEntry契約 |
| 共通Operator／Function | 汎用計算、共通判定、型・単位・丸め・NULLの法則を持つ純粋ロジック | 同じ式の画面・Tableへのコピー |
| Function Model／Context | 機能の利用者・入出力・Query／SQL・複数集合・粒度・Presentation・操作とその適用条件 | 物理Tableの正本、Dataの普遍的制約の重複定義 |
| Business／State／Operation | Task／Procedure、Event、状態遷移、Actor・責任・権限、永続化と外部Effectの意図 | 項目の表示書式や行計算の複製 |

**項目拡張定義からはRule IDを辿れるようにするが、同じ式を二か所へ保存しない。** 例えば\`orderLine.amount\`を「非保存のDecimal／金額」と宣言し、Table Scriptの導出Rule \`orderLine.calculateAmount\`を参照可能にする。このIDの所有・解決は既存Subject/Rule/Source Indexを使い、新たなFormula Registryを作らない。項目画面で式を編集する体験を将来提供する場合も、更新先は**そのRuleを記述したScriptの原文**であり、項目定義に別の式をコピーしない。原文への逆編集が安全でなければプレビュー止まりとする。

ただし、単純な必須・値域・キー等の**宣言的Constraint**はData Model側に残す。型とDB一意性などの契約をスクリプトでしか読めないものにはしない。Constraintを実行するときは、Table Scriptと同じ型付きOperatorを使い、複数のValidation Engineを作らない。

#### 同一の属性に対する計算の帰属

- **Dataに普遍的な値**：\`lineAmount = quantity × agreedUnitPrice\`のような行局所の導出はData RuleとしてTable Scriptにまとまって記述。複数のFunctionで同じ計算を使う。
- **Functionに固有の値**：\`customerOutstandingAmount\`のように複数表、ユーザー、期間、取得範囲で変わる集計はFunction Context／Queryで計算。Data Modelの属性を無理に増やさない。
- **業務上確定した値**：受注確定時の合計金額・単価など、後のマスター変更でも変えない値はAction／Transactionが保存。純粋な導出やキャッシュと区別する（#1641）。
- **外部Effect**：書込み後のメール・在庫予約・連携はTable Scriptの読取Phaseで実行しない。Operation／Host／ProviderとReceiptを通す（#1129/#1174）。

この境界により、**スクリプトが集約される利点**と、**機能モデルがデータから独立する利点**を両立する。Function固有の集計までTable Scriptへ入れるとDataがFunctionへ逆従属するため、意図的に避ける。

### 11.5 Table Scriptのイベントは業務Eventと同一ではない

Table ScriptのPhaseは「行を取得した／Draftを変更した／保存前を検証した／DB確定後の結果を受けた」などの**Data lifecycle接点**である。Business Modelの「申請された／承認された」、Functionの「検索を開始した」、Hostの「HTTP Requestを受け付けた」とは別種類の契機として既存Trigger／OperationへBindingする。

読み取りで行う純粋導出と保存前の書込み検証は、副作用の有無も必要なContextも異なる。同じScript原文内に複数のRuleを置けても、Phase、Read-set／Write-set、型、適用先、再入、評価順、粒度、トランザクション境界を識別可能にする。イベント名の順だけで評価順を自動保証しない。依存が循環する場合は検出し、変更された行／属性に必要なRuleのみ評価できるかを検証する。

初期設計で要求する区別は以下のとおり。

| Phase上の目的 | 例 | 許される効果・注意 |
| --- | --- | --- |
| Context取得／読取側の導出 | 入力行から仮想金額を求める | Pure。全件判定にはQuery計画との接続が必要 |
| Draft変更時 | 数量変更で仮計算・画面再表示 | Draftの一時値のみ。DB確定ではない |
| Before-write | 入力正規化、計算、制約、旧行Version確認 | 宣言したTable／Transaction内の処理だけ。実AuthZとHost Commitは別 |
| After-commit結果の扱い | 保存Receiptから通知を要求 | 実配送は別Effect／Outbox。再試行・二重配送・unknownを区別 |
| Function／Business Event | 申請・承認で状態遷移を依頼 | 現在Actor/委任・State Guard・Operation Grantを再検査 |

これは厳密な共通Hook列挙型を新設する提案ではない。実Phaseと名称・順序・利用可能Contextは既存#1129/#4198/#4495および選択Providerの実契約に照らして決める。

### 11.6 受注・明細の例で一つの正本を保持する

下記は**意味の配置を説明する表であり、採用済みのTEI DSLではない**。

| 定義対象 | 一次の記述位置 | Functionからの利用 |
| --- | --- | --- |
| Product／Order／OrderLine、Relation、複合キー | Data Model | 複数Functionで参照 |
| OrderLine.amount：仮想か保存か、Decimal／通貨 | Dataの項目拡張 | フォーム・一覧・帳票へ項目情報を投影 |
| \`calculateLineAmount\`：数量×取引時点単価、NULL／丸め | OrderLineのTable Script（局所Rule） | 作成・編集・Queryで**同じ導出意味**を参照 |
| 商品の現在価格の取得 | 商品Relation＋Query | 受注入力FunctionのContextで参照 |
| 受注一覧（担当者別・期間別） | Function View＋Context Query | 同じOrder Dataから複数結果を取得 |
| 受注確定時の単価Snapshotと合計保存 | Function Action＋Data Operation | DB Commit・Receiptを明示 |
| 受注承認の段階・承認者・代理 | Business／State Transition／Organization | Function Actionを入口として呼ぶ |
| UI、API、MCP、帳票 | Presentation／Transport／Target | 同じFunction／Context／RuleへBinding |

**例外を無理に隠さない**：受注一覧の検索時に非保存の行金額が必要なら、計算順の正しさを#1640で確認する。保存済み金額だけを使う場合でも元データ・時点・単位を照合する。複数Provider横断で原子的な保存が保証できない場合は、保存成功に見せず部分確定・unknownをそのまま返す。

### 11.7 独立Expected候補：Functionをデータ従属へ戻さない

| ID | 反例・検証内容 | 期待・帰属 |
| --- | --- | --- |
| W11-FM-01 | 同じOrderデータに受注検索・明細編集・売上集計・帳票の各Function | Dataを複製せず、各Functionが別Context／操作契約を持つ |
| W11-FM-02 | 外部APIと計算だけで成立するFunction、DB対象0件 | Data Modelが先にないとFunctionが作れない設計を拒否 |
| W11-FM-03 | Dataから暫定CRUD生成後、明示Functionへ移行 | 同じ業務Operationが二重実行・二重正本にならない |
| W11-FM-04 | 項目拡張に式A、Table Scriptに異なる式B | 単一Rule IDへの帰属衝突として診断。暗黙の実行順で解決しない |
| W11-FM-05 | 画面入力中の計算と通常CLI保存時の計算 | 両者で同じRuleを利用、Draft値とCommit値を別に観測 |
| W11-FM-06 | 仮想導出値で検索／並べ替え／ページ分割 | 不正な早期Limitを拒否。SQL Pushdown・完全取得・unsupportedを条件付き選択 |
| W11-FM-07 | 2子行と別集合3行をJOINし集計 | JOIN後6行を業務明細6件と誤認しない、元行Identity・粒度維持 |
| W11-FM-08 | 一つのFunction ViewをTable／Map／Chartへ投影 | 表示を変えてもFunction Meaningが増えず、集計値への不可逆更新は拒否 |
| W11-FM-09 | 一つの承認ActionをWeb／CLI／Nodeから起動 | 状態遷移、Actor、現在Grant、実Receiptが一致する契約を使う |
| W11-FM-10 | Table Scriptの読取Phaseで外部配送を実行しようとする | Pure/Effect境界で拒否、Operation/Providerへ正しく移管 |
| W11-FM-11 | 商品単価の変更後に受注を再読込 | 取引時Snapshotを保存事実として保持し、現在参照Projectionだけ変化 |
| W11-FM-12 | Native SQLとDSLが同じQueryを別正本として編集 | 正本Sourceと派生表示を識別し、競合・Lossを検出 |

これらは新しい独立Test Suiteを立てるものではない。W11-AH-01〜12と一部重なるものは**別のテストデータやRuleを複製せず同じ既存Scenarioへ参照で束ねる**。検証時の合格・未実行・未対応の分類は#1180/#2514に従う。

### 11.8 既存担当への分配と実装順

| 論点 | 既存Owner・成果を利用する場所 | 追加する最小差分 |
| --- | --- | --- |
| Data／Function独立、三要素の関係 | #1125、#1031、#1126、#2506、#2507 | FunctionなしData／DataなしFunctionを含む対応・不整合診断 |
| Function Contextの検索／集計／SQL | #2387、#2506、#2586、#1640、#2500 | Function別QueryとView・集計・更新逆写像の同一意味検証 |
| Table Script／Derived Field／時点 | #1129、#1637、#1640、#1641、#2622 | Field metadata=意味／参照、Script=Phase別局所Ruleの競合診断 |
| 項目拡張の型・Facet・表示 | #3816、#2495、#2497、#3278 | 仮想・保存・型・表示・Rule参照の分離 |
| 業務Event／State／Process Flow | #4198、#2386、#4490、#1158、#4495 | Table event／Function Trigger／Business Transitionの区別 |
| データ操作・多Provider・実Effect | #2622、#1127、#1174、#2597/#2598 | 同一Function／Operationの実Provider・Effect資格 |
| Authoring・実Generator・セルフビルド | #2621、#1018、#988、#2831、#4583 | 既存Reader/Builderの選択と実消費。新Grammar・新P0なし |

**段階順は変更しない**：先に一つの正式Source／Query／Function Data Contextからの実Read・限定Derived Rule→Go生成・通常CLI実Providerを通す。その後にTable Scriptのイベント付き仮想値、検索用導出最適化、親子保存／業務状態遷移へ広げる。Studio Stage1→Stage2のP0は並行する。

### 11.9 未決の設計判断を狭く残す

1. **ScriptのAuthoring粒度**：Table単位のScriptを一つのファイルにまとめるか、Ruleごとに別ファイルでTable配下に参照するか。いずれも式の一次定義元は一つ。StudioでTableのScriptを一覧・追跡できれば物理ファイル数は意味上の問題ではない。
2. **項目拡張からの参照記法**：Fieldが\`derived-by\`相当の参照を明記するか、Table Scriptの書込先から逆引きするか。両方向を独立編集させず、Source spanと一意性・診断を比較して選ぶ。
3. **Function固有RuleとData Ruleの合成順**：共有計算とFunction専用補正の競合を暗黙の上書き順にしない。適用Phase／依存／優先・失敗／読み書き集合を反例で決める。
4. **SQL Pushdown対象の拡大**：同じRuleをSQL／Go／外部Providerで実現できる場合の型・NULL・丸め・権限・完全性一致を限定試験する。利用できない場合に見かけの全件検索を返さない。

この4点は実装済みの意味契約・受入条件が揃うまで候補として保持する。**方向性としては「FunctionはDataに従属させない」「Dataに普遍的な導出はTable Script／Data Ruleへ集約する」「Field拡張へ別の実行DSLを複製しない」の3点を基準にする。**


## 12. DataとFunctionの独立性を支える境界を検収可能にする（2026-11-15）

この節は§10〜11で定めたアウフヘーベンの**実装時の判定基準**を明確にする。規則をどこへ配置するか、同じデータに複数のFunctionが関わるときに何を共有するかは、[データ導出規則の最小意味契約：配置基準の追補](DATA_DERIVATION_REFERENCE_CONTRACT.md)を一次の設計Ownerとし、ここではWagbyの機能比較から見えた利点・欠点と検証順を整理する。

### 12.1 失ってはならないX-TEAの利点と、その代償

**Data、Function、Businessはいずれも独立して存在できる。** 一つのFunctionは複数Dataを扱え、Dataを永続化しない計算やAPI集約でも成立する。逆に、一つのDataを複数Functionから利用できる。BusinessはFunctionだけでなく手作業や外部主体を含む。参照可能性は包含・所有関係を意味しない。

この方式には意図的に引き受けるコストがある。データモデルだけから機能の全入力・出力・検索範囲を推定できないため、Function Data Contextと型付きRelation Bindingが必要になる。単に抽象化の名称を増やすのではなく、**Functionが何を必要とするかを宣言し、Providerが何を取得したかを別に観測する**既存#2506/#2586へ寄せる。

| 独立性によって生じる課題 | TEIで吸収する既存の仕組み | 機械的に確定してはいけないこと |
| --- | --- | --- |
| Functionが利用するData／Relationがわかりにくい | Function Data Contextの型付きMapping、Relation Path、ScopeとLineage | Relationがあるだけで入力集合へ自動採用 |
| 一つのDataを複数Functionで使うため、規則の置き場が揺れる | Dataに普遍的なRule＝Table Script、機能固有＝Function Context、共通純粋演算＝Operator | UI／CRUD／Functionそれぞれで同じ式を再定義 |
| 検索と編集の対象が一致しない | Query Result Grain、Identity、#2500のViewChange→Operation Intent | JOIN行や集計結果を実体や書込み対象に直接変換 |
| FunctionがDataなしでも成立し得る | 型付きI/O、Function／Application Operation／Provider契約 | 物理TableのないFunctionを不正とする |
| Data中心の即時CRUDが業務Functionと競合し得る | #1127の標準CRUD候補と明示Function Actionの区別 | 自動CRUDの存在から業務更新の許可・意味を確定 |
| 複数Source/Providerを横断すると取得・保存が不完全になる | #2622のData Operations、ContextのCompleteness、#1174のEffect/Receipt | 非原子的API Batchを原子Commitとみなす |
| 仮想計算がSQL検索・ソートの前に必要 | #1640のDerivation依存とQuery Plan、適格なPushdown | 取得後の一部行だけを計算して正しい上位N件と宣言 |

### 12.2 規則配置の判断を「不変である範囲」から始める

- **Dataに帰属するRule**：そのData／Relation、必要な時点・完全性、Rule版が同じなら、利用するFunction・画面・担当者によらず、値や制約の意味が変わらないもの。少なくとも共有Attributeの導出・制約はTable Script／Data Ruleへ集約する。ただし関係集合の全件が必要ならその依存を明示し、行だけで済むと偽装しない。
- **Functionに帰属するRule**：「今月の」「担当者が見られる」「検索条件Aの中で」「この処理の入力だけ」など、Function Contextの入力・期間・取得集合・選択・適用目的によって変わるもの。Dataが持つ汎用属性へ押し込まない。
- **Businessに帰属するRule**：組織上の担当、申請・承認・差戻し、責任の移動、業務状態とその時点を対象にするもの。Functionから参照できても、Functionの画面やData Tableだけで業務プロセスの意味を確定しない。
- **共通Operatorに帰属するRule**：対象Data／Functionに依存しない型付き演算や判断の法則。実装や言語別の関数を業務定義へ露出せず、複数の適用先で再利用する。

これは一つのRuleを必ずどれか一つにしか置けないという分類ではない。**Ruleの一次定義元**と、DataやFunctionやBusinessへの**適用先のBinding**を分ける。前者は一つ、後者は複数あり得る。同じ条件を使うData Constraintと承認Guardは、適用先・拒否の意味・現在Grantを別にする。

### 12.3 Table Scriptを集約点にしても、Functionを下位に置かない

Table Scriptは「Dataのライフサイクルに属する振る舞いを一覧できるAuthoring面」であり、データモデルが所有する汎用業務Processや別Engineではない。項目拡張定義は型、値域、単位、保存性、仮想属性、表示Facet、Ruleへの参照を持つ。計算式は原則としてTable Scriptの一次Sourceに一つだけ存在させる。項目編集画面でその式を見せても、二箇所にコピーして保存しない。

ただし**すべてのRuleをTable Scriptへ集めることはしない**。Function固有の期間別・利用者別集計をテーブルに記述すると、FunctionがDataの内側へ再び押し込められる。Table Script内で得られる値とFunction Contextの結果に、同じ名前の「仮想項目」を無条件に付けない。後者には結果集合の粒度・取得Scope・編集可否を持たせる。

Table Scriptのread/draft/before-writeに適用するRuleと、Business Eventの申請／承認、および外部通知Effectは別の責務である。必要なイベントから同じFunction Action／OperationへBindingする。RuleをどのPhaseで評価するかはRead/Write依存や実Provider能力にも左右され、Source内の並び順・画面での表示順で決めない。

### 12.4 一つの受注データで三つの異なるFunctionを使う縦断

第一の検証題材は、既存W11-01〜04の受注・明細をそのまま使う。新しいモデル一式を増やさない。

| 縦断の対象 | Data／Function／Businessの役割 | 受入の焦点 |
| --- | --- | --- |
| 受注入力Function | Order/LineをData Contextで利用。保存前に共通の行金額Ruleを適用 | 同じData RuleをUI／通常CLIで使う。DraftとDB Commitを区別 |
| 売上集計Function | 同じOrder/Lineを期間／担当／部門で検索・集約する別Context | Table Scriptに売上画面専用式を増やさず、Query grain・Scopeを保持 |
| 受注承認Function | 同じOrderを参照するが、承認ActionとBusiness State／Actorを別に保持 | 承認の資格をField型・画面read-only・Data Ruleだけで決めない |
| Dataを持たない価格試算Function | 型付き入力と共通Operator／外部価格Providerだけで結果を返す | 物理Tableの存在をFunctionの登録・生成条件にしない |

最初は**実際の選択済みSource→一つの純粋Rule→実Generator／Builder消費→通常CLI Provider／独立Consumer**という既存製品P0に接続できる範囲から着手する。Function集計・Table ScriptのEffect・複数Provider・Windows Studioセルフビルドをこの一回の実行の必須条件へ追加しない。後続のP1受入ではW11-FM-01〜12／W11-AH-01〜12を重複なしに束ね、Dataのみ／Functionのみ／双方あり／取得不完全／旧Source／競合Ruleを確認する。独立Expectedは既存#1180/#2514へ渡す。

### 12.5 この段階で確定できることと、残すこと

**設計上の基準として維持するもの**は、(1)Functionの存在はDataに依存しない、(2)Fieldの拡張定義は処理DSLの第二正本にならない、(3)Data共通RuleはTable Script／Data Ruleに集約する、(4)Function固有QueryとContextはData Modelの別コピーにしない、(5)Business Event／Data lifecycle／Host Triggerは別、の五つ。

**実証してから決めるもの**は、Scriptファイルの物理的な分割、Rule IDのField側明示参照か安全な逆引きか、Relation全件を前提とするData RuleのProvider適合、複数Ruleの合成／衝突診断、SQL PushdownのTarget別同等性である。設計文書の一致を、Ownerによる意味受理や実行済みの製品機能と混同しない。

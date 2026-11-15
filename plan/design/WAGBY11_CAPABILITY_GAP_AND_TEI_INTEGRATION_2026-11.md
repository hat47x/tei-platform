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

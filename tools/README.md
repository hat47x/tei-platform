# 検証・参照用ツール

`tools/`には、構成検査、受入根拠の取得、参照実装、再現可能な実験に使う道具を置きます。製品の再利用可能な実装は`internal/`、入力と期待結果は`reference/`、利用例は`examples/`にあります。

表には、この階層の全ディレクトリと直下の実行用ファイルを載せています。各ツールの固有の使い方や判定条件は、そのディレクトリ内のREADME・実装・Issueを正とします。

独立した参照プラグインの検査入口は [`check-reference-plugin-isolation.sh`](check-reference-plugin-isolation.sh) です。`plugins/`配下の参照実装を一時Goモジュールへ複製してオフラインで検証します。

## リポジトリ検査と共通部品

| 場所 | 役割 |
| --- | --- |
| [`internal/`](internal/) | 複数のツールから使う共通処理 |
| [`repository-architecture-check/`](repository-architecture-check/) | パッケージ構成と依存境界の検査 |
| [`repository-dependency-report/`](repository-dependency-report/) | Goの直接依存関係の一覧作成 |
| [`repository-plan-check/`](repository-plan-check/) | ADR・Issueの番号や要求と検証表の対応確認、ADR索引とADRファイルの一致・状態の確認（#2343） |
| [`doc-reference-probe/`](doc-reference-probe/) | 文書中の明示パス参照と暗黙候補を調べる任意実行の限定試作（#2309） |
| [`repository-node-tests/`](repository-node-tests/) | どのworkflowからも実行されなかったNode.js試験5ファイルをgo testから実行する（#2346） |
| [`check-public-boundary/`](check-public-boundary/) | 追跡ファイルに、非公開資料の名前・所在のマーカーが入っていないかを検査する（#2342 / #2436） |
| [`repositoryqualification/`](repositoryqualification/) | リポジトリ検証条件の共通処理 |
| [`reference-cli-trace-benchmark/`](reference-cli-trace-benchmark/) | 合成Workspaceによるtraceの処理段別の時間・メモリ計測と出力不変の確認 |
| [`reference-change-packet/`](reference-change-packet/) | 変更範囲・危険度・検証・停止条件をJSONで宣言し、範囲確認と検証実行・証拠記録を行う参照（#2017） |
| [`reference-generator-selection/`](reference-generator-selection/) | 受理済み契約に対する生成器候補を固定fixtureで比較し、source/binary digest、実行言語・runtime、Go/Java/WASM/WASIのsource path非依存な反復build一致、case実行のexit/stderr digestと結果をreceiptへ残す参照。Go・Python・JavaScript・Java・WASM・WASI、long-running候補、registryの版・置換・失効も確認する（#2016 / #2077） |
| [`reference-generator-attestation/`](reference-generator-attestation/) | 選抜receiptから候補のcanonical statementを作り、ssh-keygenの署名で改変を検出する参照。署名者のAuthority・trust rootは検証しない（#2016） |
| [`reference-generator-artifact-store/`](reference-generator-artifact-store/) | 選抜receiptのGo/Java/WASM/WASI候補source/binaryとPython/JavaScript候補sourceをsource digestでcontent-addressedに保存・取得し、再計算digestで改変を拒否し、receipt/listで言語を返す参照。明示keep setによるlist/prune、成功pruneのhash-chain audit log、ssh-keygenで署名できるaudit checkpointと、署名・store一致を併せて検証するcheckpoint再検証も持つ（#2016） |
| [`reference-generator-context/`](reference-generator-context/) | 受理済み契約・fixture・Workspace revisionから、生成モデル向けの上限付き文脈を作る参照（#2016 / #2021） |
| [`reference-generator-revocation-review/`](reference-generator-revocation-review/) | 生成器registryの失効と、registry編集とは別の承認recordの対象一致を検査する参照。承認者のAuthority・trustは検証しない（#2076） |
| [`reference-generator-extension-points/`](reference-generator-extension-points/) | 生成器が宣言した拡張点に案件側のコードを生成物の外から結び付け、契約変更・再生成での保持・拡張点外の直接編集・業務結果の差を検査する参照（#2090） |
| [`reference-derived-name-check/`](reference-derived-name-check/) | 要素の名前と、生成器・Providerが作る派生名の衝突、予約語、最大長、使用できない文字を、層ごとの宣言された規則で定義の受理前に検査する参照。規則は参照用の宣言で、実在のProviderや生成器の規則とは未接続（#2094） |
| [`reference-semantic-domain-coverage/`](reference-semantic-domain-coverage/) | 意味の定義と要素の対応を、受理済み定義だけを根拠に検査する参照。複数一致（一本道の破れ）、未記入の要素、What/Whyの欠落を一覧する（#2095） |
| [`reference-target-profile/`](reference-target-profile/) | 同じ契約・fixture・候補を複数のTarget Profileで実行し、宣言と実行結果、case別出力の意味一致、oneshot/long-running/batch process model、stdout/stderr/exit-codeの観測、filesystem artifactのreplace/in-place/rolling配備、sqlite artifactの保存・rollback、slot/history指定のrollback、historyの検証・保持件数によるprune、署名済みapprovalによる削除承認、そのhash-chain auditとaudit checkpointを記録する参照（#2019） |
| [`reference-temporal-derivation/`](reference-temporal-derivation/) | 業務有効時点・記録時点・規則版・評価時点・確定保存値を区別し、過去再計算を historical-replay / what-if / current-computation / confirmed-fact-read としてラベルする参照。過去価格や当時の規則版が無ければ未確認とし、確定保存値を書き換えない（#1727） |
| [`reference-derived-field-projection/`](reference-derived-field-projection/) | 導出項目の意味状態（宣言・未保存Preview・実行時導出・確定保存）を表示先へ渡し、導出値への直接書込をAction／保存境界でも拒否する参照。古い表示値を確定値として返さない（#1707） |
| [`reference-productivity-benchmark/`](reference-productivity-benchmark/) | 4タスクの計画とrunのcoverageを検証し、3方法×3回の中央値と範囲を集計する参照（#2018） |
| [`reference-tenant-isolation/`](reference-tenant-isolation/) | SaaS向けに、テナントrootの閉じ込めと越境拒否を提供する基盤pluginの参照（#2020） |
| [`reference-audit-log/`](reference-audit-log/) | 追記専用のhash chain、改ざん検出、pinned headによる末尾削除検出を提供する監査ログ基盤pluginの参照（#2020） |
| [`reference-ai-write-window/`](reference-ai-write-window/) | Pluginが宣言した書込み窓にAIの編集が収まっているかを検査し、窓の外の変更を受理前に拒否する参照。未解決と確認済み違反、Target別の可用性、全体生成の境界を分ける（#2092） |
| [`reference-authentication/`](reference-authentication/) | credentialを発行せず、principalとtenantの認証境界を検証する基盤pluginの参照（#2020） |
| [`reference-offline-storage/`](reference-offline-storage/) | 個人向けに、root内へ閉じたローカル保存を提供する基盤pluginの参照（#2020） |
| [`reference-export-package/`](reference-export-package/) | SIer向けに、manifestとdigest付きの配布zipを作る基盤pluginの参照（#2020） |
| [`reference-delivery-profile/`](reference-delivery-profile/) | 提供形態ごとにpluginを選択し、必須能力が揃っているかを検証する参照（#2020） |
| [`reference-delivery-bundle/`](reference-delivery-bundle/) | 解決済みprofileとplugin descriptorをzip化し、digest・detached signature・merge/fresh install・activate/rollback・health/start/graceful stop/restart/reconcile/superviseを提供する参照（#2020） |
| [`reference-change-assurance/`](reference-change-assurance/) | 変更に応じて、過去の検証結果が現在も同じ主張を裏付け得る範囲を、合成データで示す読取り専用の参照（#1445） |
| [`reference-subproject-impact/`](reference-subproject-impact/) | 確定した二つのコミットの実差分から、プラグイン追加時に既存コアを何か所変更したかを測る参照（#1251） |
| [`reference-studio-host-boundary/`](reference-studio-host-boundary/) | TEI StudioのWeb接続・失効・要求スコープと、DesktopのDraft・private-resource境界を確認する独立回帰（#1824） |
| [`reference-studio-host-qualification/`](reference-studio-host-qualification/) | TEI StudioでCLI・Go共通操作・Host結果の意味を比較する検査（#1824） |
| [`reference-studio-static-composition/`](reference-studio-static-composition/) | TEI Studioの静的合成の最小参照実装（#1824） |
| [`reference-studio-plugin-inventory/`](reference-studio-plugin-inventory/) | Studioのプラグインについて、導入・有効・権限・版対応・提供元・読取り可否を別の値として整理し、未完了の走査や過去の件数を現在の0件・確認済みにせず、無効プラグインを起動せずに診断する参照（#2045） |
| [`reference-studio-plugin-disable/`](reference-studio-plugin-disable/) | Studioのプラグインの無効化・削除・再有効化について、必須依存の追加候補、明示確定、設定版・宣言・ソース指紋・権限の再検査、設定更新と拡張起動の分離、設計ソースのバイト保持を判定する参照（#2046） |
| [`reference-studio-plugin-data-lifecycle/`](reference-studio-plugin-data-lifecycle/) | Studioの生ソース保存と明示的なデータ削除について、プラグイン不在でも保存できること、受理・実行との分離、外部編集との競合、プラグイン削除とデータ削除の区別、共有・不明領域の扱い、保持不能時の全体拒否、変換規則の適用を判定する参照（#2047） |
| [`reference-derivation-disclosure/`](reference-derivation-disclosure/) | 導出結果の開示境界を、既存`security.authorize`へ判断を委譲して判定する読取り専用の合成参照（#1693） |
| [`reference-er-table-crossview/`](reference-er-table-crossview/) | 論理ERと物理表を相互参照する二つの読取り専用の表示を、一つの成果物へ出力する参照（#1126） |
| [`reference-project-advice-lit/`](reference-project-advice-lit/) | 案件専用のGo補助プラグインとWeb標準/Litを組み合わせた、最小の縦断参照（#951 / #997） |

## 設計、業務知識、受入れ

| 場所 | 役割 |
| --- | --- |
| [`first-value-ux-audit/`](first-value-ux-audit/) | 初回利用時の操作上のつまずきの確認 |
| [`reference-authoring-auxiliary/`](reference-authoring-auxiliary/) | 設計の補助情報（用語・改訂履歴・注記・区分）の置き場所と意味の版の境界の確認 |
| [`reference-authoring-witness-openssh/`](reference-authoring-witness-openssh/) | OpenSSHを使った記述作業の参照確認 |
| [`reference-authority-evidence-trust/`](reference-authority-evidence-trust/) | 情報源の権威と検証根拠の信頼境界の確認 |
| [`reference-batch-completeness/`](reference-batch-completeness/) | 一括導出の行別失敗と集合演算の結果完全性を、NULL・未取得・評価失敗を区別して判定する読取り専用の参照（#1724） |
| [`reference-business-dfd/`](reference-business-dfd/) | 業務データフロー図の読取り専用参照 |
| [`reference-business-procedure-check/`](reference-business-procedure-check/) | 業務手順と担当・機能・データの整合確認 |
| [`reference-capability-invocation-context/`](reference-capability-invocation-context/) | Capability実行時の文脈境界の確認 |
| [`reference-capability-mutation-protocol/`](reference-capability-mutation-protocol/) | 変更を伴うCapabilityの手順確認 |
| [`reference-capability-transport/`](reference-capability-transport/) | Capabilityの伝送形式と境界の確認 |
| [`reference-capability-degradation/`](reference-capability-degradation/) | 既存DB・Providerが観測された能力と業務操作が要求する保証を照合し、不足時に操作を生成しないか明示の弱い保証を宣言するかを記録する参照。縮退を同等と認証しない（#2096） |
| [`reference-chat-cli-host/`](reference-chat-cli-host/) | 信頼された開発用Hostの事前診断 |
| [`reference-cognition-port/`](reference-cognition-port/) | 軽量認知を提案ポートとして扱い、文脈・出所・能力限界・失敗状態を記録する参照。受理・権限・実行を決めない（#2021） |
| [`reference-cognition-handoff/`](reference-cognition-handoff/) | 軽量認知のpositive validation提案をShared Acceptanceの変更candidateへ変換する参照（#2021） |
| [`reference-critical-lowering/`](reference-critical-lowering/) | 意味定義から実行形式への変換条件の確認 |
| [`reference-derivation-cache/`](reference-derivation-cache/) | 導出値の鮮度・キャッシュ再利用条件・業務上の確定値の境界を合成入力で判定する読取り専用の参照（#1641） |
| [`reference-derivation-evidence/`](reference-derivation-evidence/) | 導出値の設計由来・評価根拠・診断と、要求者ごとの開示境界を合成入力で判定する読取り専用の参照（#1696） |
| [`reference-derivation-plan/`](reference-derivation-plan/) | 導出値を含む問い合わせ計画の依存順・整列・ページ分割の成立条件を合成fixtureで判定する読取り専用の参照（#1640） |
| [`reference-derivation-projection/`](reference-derivation-projection/) | 導出項目の表示・一時プレビュー・編集不可・確定保存の意味境界を合成入力で判定する読取り専用の参照（#1707） |
| [`reference-derived-validation/`](reference-derived-validation/) | 導出値を用いる業務Validationの評価順と、未取得・実行失敗・正常な不成立の境界を合成入力で判定する読取り専用の参照（#1729） |
| [`reference-epistemic-knowledge/`](reference-epistemic-knowledge/) | 文脈ごとの知識状態と根拠の確認 |
| [`reference-evaluation-context/`](reference-evaluation-context/) | 評価文脈の明示入力・時計固定・決定的再評価を合成Fixtureで検査する読取り専用の参照（#1720） |
| [`reference-expected-state-update/`](reference-expected-state-update/) | 期待状態の比較で同時更新を検出し、競合を業務結果として拒否する参照。冪等キー、権限・検証の区別、digest比較、観測件数を含む（#2483） |
| [`reference-exploratory-prototyping/`](reference-exploratory-prototyping/) | 探索的試作の反復記録と部分採用を、仮説の昇格なしに確認する参照 |
| [`reference-external-coordination/`](reference-external-coordination/) | 外部担当との引継ぎ・協調境界の確認 |
| [`reference-format-version-migration/`](reference-format-version-migration/) | 定義形式の版と変換規則を一か所に置き、読取り手ごとの旧形式対応をなくす参照 |
| [`reference-generated-code-connection/`](reference-generated-code-connection/) | 生成コードをTEIの契約境界へ接続できるかの判定と、部分的なモデル化と保持の参照 |
| [`reference-generation-graph/`](reference-generation-graph/) | phase付き変換グラフのEffect漏れ・型契約・循環・未確定伝播を合成入力で判定する読取り専用の参照（#1063） |
| [`reference-shared-acceptance/`](reference-shared-acceptance/) | 人・画面・AIに共通する変更受入れ処理の参照実装 |
| [`reference-standard-function-candidates/`](reference-standard-function-candidates/) | データ定義から実行前の標準機能候補と不足する契約を提示 |
| [`reference-spec-review/`](reference-spec-review/) | 仕様書の不足・曖昧さ・矛盾の検討支援 |
| [`reference-procedure-expansion/`](reference-procedure-expansion/) | 受理済み手順を非画面・画面・外部・AIの実現先へ展開し、実行可能範囲を分ける参照 |
| [`reference-procedure-formalization/`](reference-procedure-formalization/) | 業務手順の文章から、形式化を妨げる不足と矛盾を根拠付きの質問として示す読取り専用の参照 |
| [`reference-purpose-preflight/`](reference-purpose-preflight/) | 開発目的に対する業務規則の未確定・実行時値待ちを合成データで区別 |
| [`reference-provider-override-update/`](reference-provider-override-update/) | 提供元定義の更新で案件側の上書き・追加を失わないよう、旧版・新版・案件を正本識別子ごとに比較し、`unchanged`/`applicable`/`conflict`/`unresolved`に分類する参照。名前の衝突と更新前への復帰を確認する（#2091） |
| [`reference-query-input-nullability/`](reference-query-input-nullability/) | Query入力のmissingと明示nullを区別し、NULL可否を契約で明示してSQL実行前に拒否する参照。一時PostgreSQLで許可NULL・省略・空文字・拒否を観測する（#2056） |
| [`reference-quantity-arith/`](reference-quantity-arith/) | 数量・単価・金額の単位・通貨・丸めの意味注釈を合成入力で検査する読取り専用の参照（#1705） |
| [`reference-read-consistency/`](reference-read-consistency/) | 一回の導出評価の入力読取り整合・Snapshot保証・競合時の失敗境界を合成入力で判定する読取り専用の参照（#1708） |
| [`reference-realization-compliance/`](reference-realization-compliance/) | 規則が宣言する業務意味と実現の型・NULL方針・丸め・等価性の適合を合成fixtureで検査する読取り専用の参照（#1694） |
| [`reference-relation-aggregate/`](reference-relation-aggregate/) | Relation集合の重複計上・空集合・NULL要素・不完全取得の意味境界を合成入力で判定する読取り専用の参照（#1706） |
| [`reference-rule-binding/`](reference-rule-binding/) | 導出規則の適用先解決・競合・案件上書きを合成入力で判定する読取り専用の参照（#1692） |
| [`reference-rule-candidate-discovery/`](reference-rule-candidate-discovery/) | 入出力例から複数の規則仮説を提示し、境界例で割れる点を示す読取り専用の参照 |
| [`reference-runtime-change-instruction/`](reference-runtime-change-instruction/) | 動作中の画面からの変更指示を設計対象へ結び付け、見た目・業務規則・認可の変更を分ける参照 |
| [`reference-scenario-state-diff/`](reference-scenario-state-diff/) | 型付きDataset/Snapshotの検証と、Scenario期待値・観測の差の読取り専用参照 |
| [`reference-scoped-query-preflight/`](reference-scoped-query-preflight/) | 認可済み行スコープを物理SQLへ適用できるかを実行前に判定する参照 |
| [`reference-tool-neutral-development/`](reference-tool-neutral-development/) | 特定の開発道具に依存しない作業経路の確認 |

## ビルド、配布、実行

| 場所 | 役割 |
| --- | --- |
| [`reference-build-artifact-store/`](reference-build-artifact-store/) | ビルド成果物の保管と識別 |
| [`reference-build-attempt-evidence/`](reference-build-attempt-evidence/) | ビルド失敗を含む実行記録の保管 |
| [`reference-build-authoring-intent/`](reference-build-authoring-intent/) | 人・生成AIが書くビルド宣言から、確定したビルド計画を導出する境界の確認 |
| [`reference-build-execution-binding/`](reference-build-execution-binding/) | ビルド実行と主体・対象の結び付け |
| [`reference-builder-contract-resolution/`](reference-builder-contract-resolution/) | Builder契約の解決と適用 |
| [`reference-studio-composition-evidence-7/`](reference-studio-composition-evidence-7/) | TEI Studioの静的合成を実装外から検査する限定的な黒箱検査器（#1824 / #7） |
| [`reference-host-acceptance/`](reference-host-acceptance/) | 暫定のHost Acceptance Storeへ、issuerが署名したDecisionを鍵生成・署名・取込み・失効する参照ツール。効果許可は与えない（#267 / #1121） |
| [`reference-host-draft-recovery/`](reference-host-draft-recovery/) | Hostの外部編集競合・Draft復旧・副作用再送しない境界を合成入力で判定する読取り専用の参照（#1529） |
| [`reference-deployment-attempt/`](reference-deployment-attempt/) | 配備試行の参照用起動点 |
| [`reference-durable-run-golden/`](reference-durable-run-golden/) | 永続実行記録の期待出力の再現 |
| [`reference-native-plugin-builder/`](reference-native-plugin-builder/) | ネイティブPluginの構築例 |
| [`reference-plugin-capability-contract/`](reference-plugin-capability-contract/) | Pluginが提供する機能の契約確認 |
| [`reference-plugin-package-builder/`](reference-plugin-package-builder/) | Plugin配布物の組立て例 |
| [`reference-plugin-self-build/`](reference-plugin-self-build/) | Plugin自身が構築手順を提供する場合の確認 |
| [`reference-plugin-build-supply/`](reference-plugin-build-supply/) | 技術検証用サンプル`plugin-build-supply`の工程1〜4の通しの検証。構築意図と実ファイル・観測したGoから`tei.process.reference`の計画を確定し、その照合値を、ホストが発行する発生識別子つきの二回の実構築と内容アドレスの保管庫、既存の自己構築の実行器の検証根拠(二回構築、段階の収束、ホスト登録)、保管庫から再取得したバイト列のプロセス通信の版1での呼出しまで引き継ぐ。提供元の曖昧さ・追加・信頼の失効、部分出力を残す失敗・二回構築の不一致・一回の実行の二重計上・同時の試行の強制終了、汚れた作業ツリー・外部モジュール・ソースとツールチェーンの食い違い・ホストのGo設定、工程間の識別の差替えと工程の欠落を注入する(同じ機械の上の実際のGoの構築。提供元のカタログと信頼の失効は合成) |
| [`reference-order-multilingual-operation/`](reference-order-multilingual-operation/) | 技術検証用サンプル`order-multilingual-operation`の工程1〜6の通しの検証。複製した作業領域に受注の所有プラグイン、`std.message`、言語プラグインを加え、所有者・環境・原文言語・項目名を一組にして既存の多言語資産を変更せずに呼ぶ。日本語原文を一回改訂した前後の同じ対象一覧の突合表で、従属訳だけの失効と配布候補の再生成を確かめる。許可・受理・固定情報の失効と言語プラグインの撤去、配布候補の訳文の書換えと固定情報の差替え、改訂直後の旧配布候補、翻訳器の失敗・取消し・結果不明と機密の外部送信、翻訳中の同時編集、PostgreSQLとSQLiteの並びの差を注入する(合成のホスト、模擬の翻訳器、PostgreSQLの一つのインスタンス、SQLiteはPythonの標準モジュールでのファイル、Node) |
| [`reference-external-adoption-authority/`](reference-external-adoption-authority/) | 技術検証用サンプル`external-adoption-authority`の工程1〜4の通しの検証。SUIの島階層の契約を固定版のまま写した影の検査の報告から、拡張の後付けと名前空間所有者の判定、意味の固定記録、署名つきのデータのみの配布物、合成ホストでの受理まで、固定版・写像報告・損失の記録の照合値と`lossless=false`を一つの記録で引き継ぐ。依存blobの変更、同じ名前空間の二つの提供元、所有者と署名者の信頼付与の失効・鍵交換・期限、損失なしへの読替え、提供元の意味の変更とファイルの移動、工程間の差替えを注入する(既存資産を変更せずに固定入力の複製で呼ぶ。外部側は固定表の複製による模擬、信頼付与・鍵・ホストは合成、時刻は論理時刻)
| [`reference-order-rule-design/`](reference-order-rule-design/) | 技術検証用サンプル`order-rule-design`の工程1〜6の通しの検証。合成の旧資料と共有の合成契約の読取りと出所状態(結び付いた・表だけ・対象なし)、受理済み作業領域の複製との突合、変更候補の作成、通常のGitによる受渡しと再構築、影響範囲と過去の検証根拠の再利用可否、審査と署名つき決定までを一つの記録でつなぐ。候補・出所・署名つきの決定は共通受理へpreviewとreviewだけで提出して止め、受理しない(`order-workflow`への提出は未接続)。既存資産は変更せずに呼び、サンプル固有の処理は`internal/`へ入れない |
| [`reference-process-plugin/`](reference-process-plugin/) | 別プロセスで動くPluginの参照用起動点 |
| [`reference-composition-process-bridge/`](reference-composition-process-bridge/) | 合成基盤と従来・独立プロセスプラグインの接続試験 |
| [`reference-runtime-capability-identity/`](reference-runtime-capability-identity/) | 実行時Capabilityの識別方法の確認 |
| [`reference-runtime-snapshot/`](reference-runtime-snapshot/) | 生成時と実行時の投影境界の確認 |
| [`reference-plugin-bundle/`](reference-plugin-bundle/) | 二つの独立プラグインを一つの成果物へ結合し、論理識別子と物理成果物を同一視しないことを確かめる参照試験（#1251） |
| [`reference-plugin-wasi-smoke/`](reference-plugin-wasi-smoke/) | 独立プラグインを同一ソースからLinux/amd64とWASIへ構築し、両方の実行結果を比較する試験（#1251） |

## ウェブ実行先と判定根拠

| 場所 | 役割 |
| --- | --- |
| [`reference-first-web-target/`](reference-first-web-target/) | 最初のウェブ実行先の参照実装 |
| [`reference-first-web-target-architecture-evidence/`](reference-first-web-target-architecture-evidence/) | ウェブ実行先の構成根拠の確認 |
| [`reference-first-web-target-evidence-audit/`](reference-first-web-target-evidence-audit/) | ウェブ実行先の根拠一式の監査 |
| [`reference-first-web-target-gate-c-readiness-audit/`](reference-first-web-target-gate-c-readiness-audit/) | Gate C判定資料の検査 |
| [`reference-first-web-target-manifest-capture/`](reference-first-web-target-manifest-capture/) | 固定した投影結果の取得 |
| [`reference-first-web-target-r7a-handoff/`](reference-first-web-target-r7a-handoff/) | R7A引継ぎ根拠の検査 |
| [`reference-first-web-target-r7b-intake/`](reference-first-web-target-r7b-intake/) | R7B入力根拠の確認 |
| [`reference-first-web-target-r7b-qualification-receipt-audit/`](reference-first-web-target-r7b-qualification-receipt-audit/) | R7B検証済み記録の検査 |
| [`reference-first-web-target-r7b-replay-plan/`](reference-first-web-target-r7b-replay-plan/) | R7B再生手順の検証 |
| [`reference-first-web-target-visual-evidence-audit/`](reference-first-web-target-visual-evidence-audit/) | 表示根拠の監査 |
| [`reference-first-web-target-visual-replay/`](reference-first-web-target-visual-replay/) | 表示検証の再生処理 |
| [`reference-frozen-manifest-target/`](reference-frozen-manifest-target/) | 固定した投影結果を使う最小起動例 |
| [`reference-gate-b-final-evidence/`](reference-gate-b-final-evidence/) | Gate Bの最終根拠の収集・照合 |
| [`reference-gate-b-readiness-artifact-audit/`](reference-gate-b-readiness-artifact-audit/) | Gate B判定用成果物の監査 |
| [`reference-gate-b-repository-evidence/`](reference-gate-b-repository-evidence/) | Gate Bのリポジトリ根拠の確認 |
| [`reference-generated-artifact-evidence/`](reference-generated-artifact-evidence/) | 生成成果物の再現性根拠の確認 |
| [`reference-golden/`](reference-golden/) | 参照用の期待出力の生成 |
| [`reference-target-package-evidence/`](reference-target-package-evidence/) | 実行先パッケージの構成根拠の確認 |
| [`reference-critical-path-evidence-audit/`](reference-critical-path-evidence-audit/) | クリティカルパスの根拠一式の事前検査 |
| [`reference-first-web-frozen-world-generate/`](reference-first-web-frozen-world-generate/) | 正規化したWorldを、既存のSource loaderから書き出す構築専用の診断出力器 |
| [`reference-first-web-frozen-world-verify/`](reference-first-web-frozen-world-verify/) | 同じ型付きIRとencoding/json規約でWorldを再直列化する読取り専用の診断 |
| [`reference-first-web-host-admission/`](reference-first-web-host-admission/) | E1のApplication単位の選択を、Hostが保持する現在の受理snapshotと照合する検査 |
| [`reference-first-web-two-unit-e1/`](reference-first-web-two-unit-e1/) | 第一Web TargetをBrowserとLinux/APIの二成果物へ生成し、局所的に確かめるE1実証（#1182） |

## データベースと外部接続

| 場所 | 役割 |
| --- | --- |
| [`reference-data-first-persistence/`](reference-data-first-persistence/) | Dataに手書きの宣言を重ね、実PostgreSQLで実行前の確認・書込み・別接続の再読込を一巡する確認（生成・認可・製品E2は対象外） |
| [`reference-external-query-evidence/`](reference-external-query-evidence/) | 外部問合せの実行根拠を確認 |
| [`reference-month-end-close-correction/`](reference-month-end-close-correction/) | 技術検証用サンプル`month-end-close-correction`の工程1〜6。版つきの営業日カレンダーから締め日・ジョブの予定・支払期日を確定して改訂の影響を記録し、CP932の固定長ファイルを宣言した仕様で取り込んで模擬のメッセージと一つの事象に照合し、対象月を業務キーとする締めジョブをDBの時計による貸与とDB側のフェンシングトークン検査で実行する。締めの後に順不同で届く事象を版と方針で受け付け、有効時点と記録時点の二つの軸の単価で赤黒の伝票を計上し、請求書単位・税率別の税額をPostgreSQL、Go、SQLiteで比べて締め時点の帳票をバイト単位で再出力する(PostgreSQLの一つのインスタンス。合成のファイル、模擬のメッセージ、合成の事象。SQLiteはPythonの標準モジュールでのメモリ上の計算) |
| [`reference-order-entry-persistence/`](reference-order-entry-persistence/) | `order_entry`の注文を、Persistence Mappingから導いたDDLの使い捨てPostgreSQLへ書き込み、別プロセスが読み戻す確認（#1127）。実行前に物理schemaとSource revisionを別々に確かめ、書込み直前の許可確認を経て更新する。Source・Host許可は合成で、通常のWeb経路・生成CRUD・人的受理・製品E2は対象外 |
| [`reference-order-assist-provider-offline/`](reference-order-assist-provider-offline/) | 技術検証用サンプル`order-assist-provider`の拡張工程7・8と複合シナリオC5。見積の規則の第2世代を、端末(JavaScript)、サーバ(実行体)、一括の見直し(WASI)へ配布する途中で旧版の実行先を判定から外し、オフラインの模擬端末(`reference-offline-storage`に保存)から操作単位で同期して項目単位の競合と旧単価を再検証で止め、旧世代の端末の未知の項目を保つ。値引きの承認を見積の照合値と規程の版に固定し、期間つきの代理、差戻し、規程の移行を扱い、別プロセスの模擬の外部与信の結果不明を照会で確定し、補償の途中で止めた作業者を収束させる(PostgreSQLの一つのインスタンス、同じ機械の上の模擬の端末と外部与信) |
| [`reference-order-data-realization/`](reference-order-data-realization/) | 技術検証用サンプル`order-data-realization`の工程1〜10の通しの検証。複製した作業領域の受注へ属性を一つ加え(直接編集)、外部番号を共通受理で業務キーとして受理し、一意制約を宣言した最終版の照合値を、標準機能候補、旧構造のDBからの移行の計画と適用、一覧と親子明細、登録と別接続の再読込、SQLiteでの往復まで引き継ぐ。計画後の構造変更での停止、旧版の書き手と共存する移行と縮約の拒否、同じ外部番号の同時登録、登録後の再読込の喪失、欠番と採番元の停止、方言の差による値の変化を注入する(PostgreSQLの一つのインスタンス。SQLiteはPythonの標準モジュールでのファイル。手書きの登録アダプター、同じ機械の上の模擬の採番元) |
| [`reference-order-migration-dual-db/`](reference-order-migration-dual-db/) | 技術検証用サンプル`order-migration-dual-db`の工程1〜6。旧DBの観測と照合値付きの移行計画、全件の写し、区切りごとの移行と効果記録を経由した参照系DBへの引当、論理デコーディングによる並走中の変更の反映、得意先マスタの項目ごとの正本と名前の分割、請求システムへの冪等な送信、切替の関門、逆同期と巻戻しの訓練を、PostgreSQLの三つのインスタンスで実行する(異種DBの要件は対象外) |
| [`reference-order-rollout-host-continuity/`](reference-order-rollout-host-continuity/) | 技術検証用サンプル`order-rollout-host-continuity`の工程1〜5の通しの検証。開始記録の最初の命令`sales_expand/apply`を、製品の`Registry`と隔離の実行部で、別プロセスのホストAから子プロセスの移行ツールへ貸与つきで一回呼び、ホストAを止めて期限の後にホストBが排他トークンで引き継ぎ、製品の再接続で実行状態へ戻り、命令の後に取り直した観測でだけ完了とする。所有記録・貸与の権限・耐久呼出の記録はPostgreSQLに置く(サンプル固有)。貸与の更新・心拍の失敗、時計のずれ、環境計画の更新、実行状態の不明と喪失、拡張の間の旧版の書込み、秘密と具体の参照先の漏れを注入する(PostgreSQLの一つのインスタンス、同じ機械の上のホストと移行ツールの別プロセス) |
| [`reference-order-web-delivery-member/`](reference-order-web-delivery-member/) | 技術検証用サンプル`order-web-delivery`の工程6。工程1の読込計画(`reference/web-api-contract`)から会員と社員の射影を導き、会員向けと社内向けを別プロセスの配備単位としてPostgreSQLで動かす。他人の注文と社内向けの項目を返さないこと、会員の照会の急増を429と`Retry-After`で抑えて社内の確定の応答時間を守ること、表示用キャッシュの古さを宣言した上限以内に保ち判断は正本で行うこと、一方の配備単位での剥奪が他方で上限以内に効き縮小後の行が見えないことを確かめる(PostgreSQLの一つのインスタンス、同じ機械の上の合成の負荷) |
| [`reference-oracle-local-fixture/`](reference-oracle-local-fixture/) | 別用途の既存Oracleに一時的な検証用データを作り、ドライバー・名前付きバインド・読取り専用の効き方・検証根拠の受入れを確認 |
| [`reference-postgres-header-detail-runtime/`](reference-postgres-header-detail-runtime/) | PostgreSQL上の親子明細処理の参照実装 |
| [`reference-postgres-integration/`](reference-postgres-integration/) | PostgreSQL接続の統合確認 |
| [`reference-postgres-migration-rendering/`](reference-postgres-migration-rendering/) | PostgreSQL向け移行SQLの生成確認 |

## 形式・移行・分野別の参照

| 場所 | 役割 |
| --- | --- |
| [`reference-example-slide/`](reference-example-slide/) | 例から関係図を組み立てる参照資料 |
| [`reference-data-import-check/`](reference-data-import-check/) | 業務データ取込みの段階D0。タブ区切りの行の形をData定義に照らして検査する参照（#1807） |
| [`reference-function-definition-document/`](reference-function-definition-document/) | 受理済みの意味から機能定義書を導出する参照（#1803） |
| [`reference-ddl-entry/`](reference-ddl-entry/) | 物理対応から提供元が出力できるDDLを表ごとに取り出す入口（#1803） |
| [`reference-developer-context/`](reference-developer-context/) | TEIのGoソース1件に対し、実在する関連テストと明示参照する設計資料を根拠付きの調査候補として返す読み取り専用の参照PoC。ローカルのSystem One互換判断モデルは候補の順位付けにだけ使い、失敗時は決定的な基準順を保つ。受理・権限・実行を決めない（#2196 / #2021） |
| [`reference-design-output/`](reference-design-output/) | 受理済みの意味から表・機能・対応一覧・DDL・静的HTML索引を一つのrevisionへ束縛して導出する参照。書式（markdown/text）を差し替えても意味の行を保ち、編集できる正本の位置と、宣言された利用／観測された副作用を別々に示す（#1803） |
| [`reference-access-matrix/`](reference-access-matrix/) | Application × Dataの宣言された利用の対応表を導出する参照（#1803） |
| [`reference-accounting-period/`](reference-accounting-period/) | 業務日付とシステム時刻を分け、期間の状態・締め済み期間の直接更新拒否・赤黒訂正・締め後の到着方針・順序鍵と版による再生・締め時点の照合値を判定する参照（#2144） |
| [`reference-partial-import-plan/`](reference-partial-import-plan/) | 別Workspaceから選んだData定義の取込み結果を分類する読取り専用の参照（#1805） |
| [`reference-table-definition-document/`](reference-table-definition-document/) | 受理済みの意味から表定義書を導出する参照（#1803） |
| [`reference-list-roundtrip/`](reference-list-roundtrip/) | 意味の要素の一覧を出力し、編集後の一覧を変更案へ戻す読取り専用の参照（#1807） |
| [`reference-type-reorg-impact/`](reference-type-reorg-impact/) | 意味の型の統合案が及ぼす影響を一覧する読取り専用の参照（#1806） |
| [`reference-model-search/`](reference-model-search/) | 意味の名称を横断して検索し、置換を変更案として出す読取り専用の参照（#1804） |
| [`reference-semantic-mapping-check/`](reference-semantic-mapping-check/) | 意味の定義と物理項目の対応について、定義へ遡る道が一つか、未記入の要素、複数の定義に一致する要素、定義の追加で変わる要素の範囲を検査する参照。合成データのみで、実在のデータ辞書とは未接続（#2095） |
| [`reference-semantic-extension-catalog/`](reference-semantic-extension-catalog/) | 意味拡張の登録・解決・検証 |
| [`reference-subsystem-cud/`](reference-subsystem-cud/) | 部分システムの登録・参照・更新・削除の境界確認 |
| [`reference-sui-action-boundary/`](reference-sui-action-boundary/) | SUIの操作とTEI Actionの接続例 |
| [`reference-sui-adoption/`](reference-sui-adoption/) | SUI移行の参照データと検査 |
| [`reference-sui-interaction/`](reference-sui-interaction/) | 画面操作とApplication Actionの接続例 |
| [`reference-transform-runtime-engine/`](reference-transform-runtime-engine/) | 決定的な変換処理の実行例 |
| [`reference-transform-runtime-engine-build/`](reference-transform-runtime-engine-build/) | 変換実行基盤の構築確認 |
| [`reference-transform-seed-builder/`](reference-transform-seed-builder/) | 変換用の初期データを組み立てる処理 |
| [`reference-xtea-control/`](reference-xtea-control/) | X-TEAのSwingアプリとXeadServerを、AIが再現可能に操作する参照用テストインターフェース |
| [`reference-xead-cross-view/`](reference-xead-cross-view/) | Xeadの複数表示間の意味照合 |
| [`reference-xead-source-adapter/`](reference-xead-source-adapter/) | Xead資産を読むSource接続例 |
| [`reference-xead-source-inspect/`](reference-xead-source-inspect/) | Xead資産の読取り・調査補助 |
| [`sei-resume-reference/`](sei-resume-reference/) | SEI参照の再開条件と根拠の確認 |

## 直下の実行用ファイル

| ファイル | 役割 |
| --- | --- |
| [`ci-workflows-test.py`](ci-workflows-test.py) | 手動CIの版ずれ・作業領域検査と、自動健全性CIの公開PR境界・mainへのpush経路（コミットごとの同時実行集合、検査手順のシェル）を、隔離したGit環境で再現する回帰試験（`python3 tools/ci-workflows-test.py`）。本物のCIの実行、実際のpushイベントの挙動、TEI全体検証は扱わない |
| [`verify-critical-path-e1-archive.py`](verify-critical-path-e1-archive.py) | E1合成証拠の8件原本と固定SHA目録、R0/R1/R2・未資格状態を照合する（製品Gateの検査ではない） |
| [`test-critical-path-e1-archive.py`](test-critical-path-e1-archive.py) | E1保全検査の改ざん・欠落・偽資格の回帰試験 |
| [`verify-month-end-close-correction.py`](verify-month-end-close-correction.py) | `reference-month-end-close-correction`の全試験(工程1〜6)を、Dockerで起動した使い捨てのPostgreSQLで実行し、skipが0件であることと観測値、PythonとSQLiteの版をreceiptへ残す。作ったコンテナーだけを終了時に削除する |
| [`verify-order-assist-provider-offline.py`](verify-order-assist-provider-offline.py) | `reference-order-assist-provider-offline`の全試験(工程7・8、C5)を、Dockerで起動した使い捨てのPostgreSQLで実行し、skipが0件であることと観測値、GoとNode.jsの版、C5の通しの記録の要約をreceiptへ残す。作ったコンテナーだけを終了時に削除する |
| [`verify-order-data-realization.py`](verify-order-data-realization.py) | `reference-order-data-realization`の全試験(工程1〜10)を、Dockerで起動した使い捨てのPostgreSQLで実行し、skipが0件であることと、通しの記録(失敗注入と完了条件の合否)の要約、GoとPythonとSQLiteの版をreceiptへ残す。作ったコンテナーだけを終了時に削除する |
| [`verify-order-migration-dual-db.py`](verify-order-migration-dual-db.py) | `reference-order-migration-dual-db`の全試験を、Dockerで起動した三つの使い捨てPostgreSQLで実行し、skipが0件であることと観測値をreceiptへ残す。作ったコンテナーだけを終了時に削除する |
| [`verify-order-rollout-host-continuity.py`](verify-order-rollout-host-continuity.py) | `reference-order-rollout-host-continuity`の全試験(工程1〜5)を、Dockerで起動した使い捨てのPostgreSQL(乱数のパスワード)で実行し、skipが0件であることと観測値、通しの記録の要約、試験の出力にパスワードと接続先がないことをreceiptへ残す。`--race`で競合検出つき。作ったコンテナーだけを終了時に削除する |
| [`verify-order-web-delivery-member.py`](verify-order-web-delivery-member.py) | `reference-order-web-delivery-member`の全試験(工程6)を、Dockerで起動した使い捨てのPostgreSQLで実行し、skipが0件であることと観測値、通しの記録の要約をreceiptへ残す。作ったコンテナーだけを終了時に削除する |
| [`verify-plugin-build-supply.py`](verify-plugin-build-supply.py) | `reference-plugin-build-supply`の全試験(工程1〜4)を、清浄な複製(作業ブランチのコミットの浅いcloneなど)の上で、ネットワークからモジュールとツールチェーンを取得せずに実行し、skipが0件であることと観測値、通しの記録の要約、Goの版、試験の後も作業ツリーが清浄であることをreceiptへ残す。Dockerは使わない |
| [`verify-order-multilingual-operation.py`](verify-order-multilingual-operation.py) | `reference-order-multilingual-operation`の全試験(工程1〜6)を、清浄な複製の上で、Dockerで起動した使い捨てのPostgreSQLで実行し、skipが0件であることと、通しの記録(失敗注入、完了条件、条件付きの条件の合否)の要約、Go・Python・SQLite・Nodeの版、試験の後も作業ツリーが清浄であることをreceiptへ残す。作ったコンテナーだけを終了時に削除する |
| [`verify-external-adoption-authority.py`](verify-external-adoption-authority.py) | `reference-external-adoption-authority`の全試験(工程1〜4)を、清浄な複製の上で、ネットワークからモジュールとツールチェーンを取得せずに実行し、skipが0件であることと、通しの記録(失敗注入と完了条件の合否、信頼の時点)の要約、Goの版、試験の後も作業ツリーが清浄であることをreceiptへ残す。Dockerは使わない |
| [`verify-order-rule-design.py`](verify-order-rule-design.py) | `reference-order-rule-design`の全試験(工程1〜6)を、清浄な複製の上で、ネットワークからモジュールとツールチェーンを取得せずに実行し、skipが0件であることと、通しの記録(失敗注入と完了条件の合否、未検証の一覧)を確かめて受領記録を出す。Node.jsとOpenSSHが要り、Dockerは使わない。`--repeat N`で判定と診断の安定を比べる |
| [`self-build-ci-contract-test.py`](self-build-ci-contract-test.py) | セルフビルド専用CIの起動漏れと証拠欠落を、ワークフロー定義から検査する局所回帰試験 |
| [`verify-critical-path-e1-e2-join.py`](verify-critical-path-e1-e2-join.py) | 保存済みのE1とE2a-1の二つの記録を、証拠の差し替えや水増しなしに突き合わせる。同一の実行系列であることは示さない |
| [`test-critical-path-e1-e2-join.py`](test-critical-path-e1-e2-join.py) | 上記の突き合わせに対する、証拠の差し替え・水増しの回帰試験 |
| [`verify-e2a1-postgres-archive.py`](verify-e2a1-postgres-archive.py) | 保存済みのE2a-1実PostgreSQL試験記録1件について、保存内容の不変と試験限りの範囲を照合する。PostgreSQLの再実行はしない |
| [`test-e2a1-postgres-archive.py`](test-e2a1-postgres-archive.py) | E2a-1保存記録の改ざんを想定した回帰試験 |
| [`check-core-isolation.sh`](check-core-isolation.sh) | 合成基盤の隔離試験と別モジュールからの公開API試験 |
| [`verify-composition-reference.sh`](verify-composition-reference.sh) | 固定したクリーンな版で合成基盤・項目対応・実Process・中核独立性を検査し、結果を標準出力へ返す。全体健全性と本番移行の判定は別に行い、申し送りは[#1223](https://github.com/hat47x/tei-platform/issues/1223)へ記載する |
| [`repository-health-check.sh`](repository-health-check.sh) | 形式、静的検査、全Goテスト、参照Workspaceの検証 |
| [`run-business-procedure-gate-a.sh`](run-business-procedure-gate-a.sh) | 業務手順Gate Aの検証 |
| [`run-first-web-target-gate-c-evidence.sh`](run-first-web-target-gate-c-evidence.sh) | Gate C用のウェブ実行根拠の取得 |
| [`run-first-web-target-r7a-handoff.sh`](run-first-web-target-r7a-handoff.sh) | R7A引継ぎ根拠の取得 |
| [`run-first-web-target-r7b-intake.sh`](run-first-web-target-r7b-intake.sh) | R7B入力根拠の取得 |
| [`run-first-web-target-r7b-local-qualification.sh`](run-first-web-target-r7b-local-qualification.sh) | R7Bのローカル検証 |
| [`run-first-web-target-r7b-replay-plan.sh`](run-first-web-target-r7b-replay-plan.sh) | R7Bの再生計画検証 |
| [`run-knowledge-portability-reference-evidence.sh`](run-knowledge-portability-reference-evidence.sh) | 知識移植の参照根拠の取得 |
| [`run-repository-local-qualification.sh`](run-repository-local-qualification.sh) | リポジトリのローカル検証 |
| [`verify-cli-c1-local.sh`](verify-cli-c1-local.sh) | CLIのC1検証 |
| [`verify-first-web-target-r7b-integration-receipt.sh`](verify-first-web-target-r7b-integration-receipt.sh) | R7B統合記録の照合 |
| [`verify-first-web-target-r7b-post-merge.sh`](verify-first-web-target-r7b-post-merge.sh) | R7B統合後の状態検査 |
| [`business-procedure-gate-a-critical-tests.txt`](business-procedure-gate-a-critical-tests.txt) | Gate Aの確認対象となる重要テストの一覧 |
| [`check-core-identity-isolation.sh`](check-core-identity-isolation.sh) | 合成基盤の識別子まわりの隔離試験 |
| [`check-core-subject-isolation.sh`](check-core-subject-isolation.sh) | 合成基盤の主体まわりの隔離試験 |
| [`check-reference-plugin-bundle-isolation.sh`](check-reference-plugin-bundle-isolation.sh) | `reference-plugin-bundle`の隔離試験 |
| [`check-reference-plugin-wasi-smoke.sh`](check-reference-plugin-wasi-smoke.sh) | `reference-plugin-wasi-smoke`の実行 |
| [`check-reference-project-plugin-isolation.sh`](check-reference-project-plugin-isolation.sh) | 案件専用参照プラグインの隔離試験 |
| [`check-transform-evidence-mutation.sh`](check-transform-evidence-mutation.sh) | Transform Engineのソース変更が、再現可能な構築・Build Plan・Evidence v2へ伝わるかの確認 |
| [`check-transform-wasm-approved-invoke.sh`](check-transform-wasm-approved-invoke.sh) | 署名済みReference Hostの承認を要求して、測定済みのWasm CLIを起動する確認 |
| [`check-transform-wasm-conformance.sh`](check-transform-wasm-conformance.sh) | js/wasmの実行がネイティブのTransform wire契約を保つことの確認 |

R7Bの一回限りの準備・公開・後処理を行うスクリプトは、統合と対象ブランチの終了後に削除しました。現在の根拠確認や再現に使う検査・取得ツールは上の一覧に残しています。

# E21：二つの利用先を構成するときの残余判断

- **管理**：[Issue #1016](https://github.com/hat47x/tei-platform/issues/1016)。操作と認証の実装境界は[Issue #993](https://github.com/hat47x/tei-platform/issues/993)に委ねる。
- **性格**：現存する二つのリポジトリを読んだ構成監査。新たな構成定義、依存解決器、実行権限、適合証明は追加しない。
- **確認した版（2026-10-12）**：TEI `696db81de7528a307889c050c2a97865acd40b38`、SUI `4bd493f9942890d52117461886207d628f3554e2`。個々の資料を読んだ時点の内容を対象とし、両リポジトリで同時に検証済みの配備版を意味しない。

## 今回、境界を引き直す理由

[E20](E20_DUAL_CONSUMER_ACTION_WITNESS.md)は、受注明細の実Commandと、試験の中だけに置いたSUIの候補Commandを同じAction Hostへ接続し、別々の保存先へ書き込めることを確認した。これは共通Action境界の再利用を支持するが、二つの実アプリケーションが同じ構成手順で稼働した証拠ではない。

その後、TEI側では[PR #1329](https://github.com/hat47x/tei-platform/pull/1329)から、独立した保存先へ要求を転送する`HTTPSApplicationStore`を統合した。SUI側でも[PR #3226](https://github.com/hat47x/sui-sensemaking/pull/3226)と[PR #3227](https://github.com/hat47x/sui-sensemaking/pull/3227)から、操作確定アダプター、条件付き保存、専用の試験が統合されている。

したがって、「SUIへの操作口を新規実装する」は既存機構の見落としとなる。残る問いは**どの既存資産を、誰の判断と権限で、どの配備先へ接続するか**である。

## 構成資産と正本の照合

| 関心事 | 受注明細（TEI） | SUI | 判定 |
| --- | --- | --- | --- |
| アプリケーションの意味・操作 | [Application定義](../workspace/applications/order-entry/application.tei)と`order_entry.submit_order`のPatternが存在する | [SUIの確定アダプター](https://github.com/hat47x/sui-sensemaking/blob/4bd493f9942890d52117461886207d628f3554e2/03_Implement/frontend/src/api/tei_card_move_action.ts)は`sui.move`を構築するが、TEI側のSUI Stage 0には正式なApplication定義がない | **異なる**。SUIの操作定義を勝手にTEIの受理済み意味へ昇格しない |
| 画面と局所状態 | [Web/Reactの参照実装](../ui-customization/web-react/)は独立した構成資産 | SUIのCanvas、Undo、プレビュー、[確定後の状態計画](https://github.com/hat47x/sui-sensemaking/blob/4bd493f9942890d52117461886207d628f3554e2/03_Implement/frontend/src/domain/confirmed_card_move_state.ts)をそのまま保持する | **既存資産を選択**。Canvasを標準Formへ移植しない |
| 共通Actionと資源版 | [共通Action v1](../../tools/reference-sui-action-boundary/PROTOCOL.md)、[E20](E20_DUAL_CONSUMER_ACTION_WITNESS.md)のHost二接続 | SUIの操作送信も同じ六つの外側の項目を使用する | **契約再利用候補**。Payloadの意味と資源の所有者は別 |
| 永続化と競合 | E20では試験限定の記録用Store | [SUIのAction受信口](https://github.com/hat47x/sui-sensemaking/blob/4bd493f9942890d52117461886207d628f3554e2/03_Implement/backend/src/sui_sensemaking_api/routes/docs.py)とSQLiteの条件付き保存。入口は既定無効 | **SUIが正本**。TEIで第二のDocument保存機構を作らない |
| 別サービスへの委譲 | [HTTPSApplicationStore](../../tools/reference-sui-action-boundary/actionboundary/upstream_http.go)を既存Hostから選択可能。自動登録しない | 本物のSUI受信口は存在する | **未接続**。TLS模擬受信先への成功をSUI実接続としない |
| 本人とテナントの検証 | 実行時のSubject・Tenant・Audienceの一致を検査する参照処理がある | SUI側の認証・テナント・現在の権限はSUIが確認する | **不足**。実セッションからの真正な委譲資格情報と受信側の検証が未実証 |
| 実行先・構成の採用 | Web/React Referenceは正式な案件UI採用記録ではない | Native Canvasを保持する方針。SUI Stage 0は意味ソースの一部だけ | **未判定**。Target、Framework、UI Kit、実Providerの採否が揃っていない |
| 出典と版 | TEIのWorkspace Source IndexとGit改訂の試験は[E19](E19_GIT_REVISION_SOURCE_IMPACT.md)で確認 | SUIのNativeソースは別リポジトリ。上記の固定Git改訂が出典 | **別管理**。TEI Workspace IndexだけでSUIの全ソースを網羅したことにしない |

この表はSourceと契約の所在を示す。掲載した二つのGit改訂は個別の読取り基点であり、契約相互の適合、ビルドした成果物の同一性、稼働中の権限を認証するものではない。

## 同じ名称でも異なる二つの実行経路

**SUIが現在用意している経路**：

`Native Canvasの確定 → SUI所有の操作アダプター → SUIのAction受信口 → SUI所有のDocument保存`

**TEI Hostを介する構成候補**：

`Native Canvasの確定 → TEIのAction Host → 信頼済み本人のHTTPS委譲 → SUIのAction受信口 → SUI所有のDocument保存`

後者を選択しても、ブラウザーの認証情報をそのまま転送してはならない。TEI HostのサービスIDを利用者本人と偽ってはならない。実セッションに紐づく資格情報と受信側の署名・発行者・対象先・期限・現在の権限の検証が揃わなければ、委譲するBindingを登録しない。結果不明時の自動再送も許さない。この検証と実装は[#993](https://github.com/hat47x/tei-platform/issues/993)の担当であり、#1016が第二の認証機構を作らない。

受注明細側にも、E20の模擬保存を実データベースProviderの認証・保存完了と扱わないという対称的な制約がある。

## 構成プロファイルを決める前に残る判断

| 案件が選択する事項 | 既存の正本に戻せる部分 | 今なお明示判断が必要な部分 | 主な担当 |
| --- | --- | --- | --- |
| 利用する操作口 | 共通Action v1、SUIの現存する操作入口 | 直接SUIへ送るか、TEI Hostを介するか。双方を無条件の代替経路にしない | #993／案件の配備 |
| 誰の資格情報で操作するか | Hostの認可境界、SUIの現在権限と資源版 | 対象利用者・テナント・受信先を結ぶ信頼済み委譲元、失効時の扱い | #993 |
| どの画面と実行先を採用するか | SUI Canvas、TEI Web/React参照資産 | 案件固有のTarget・Framework・UI Kitの選択、採用した版と案件差分 | #1016／#997 |
| どの版を連動させるか | 各Git改訂、Source Index、既存の契約・成果物識別 | 別リポジトリのNative AdapterとHostの版の対応、変更時の再検証範囲 | #1016／#951／#943 |
| 構成のどこまで受理したか | 各案件のAccepted Source、現在の実行時認可 | 構成の採否を記録する出典と判断者。実行Grantは記録しない | #1016／案件 |

上表からは、独立した`Application Composition Profile`用の第二の構成定義を設ける必要性は**まだ導けない**。既存のApplication・Environment・Plugin Lock・UI Kitの参照で足りるか、二Consumerの実接続と更新作業から評価する。

## 次に必要な実証と計測

1. **接続の実証（#993）**：試験用の安全な構成で、実SUI Backendの受信口へTEI Hostから到達し、確定前の本人・テナント・対象先・権限と資源版を検証する。無資格情報、失効、別テナント、版競合、結果不明、取消しに対する無書込み・無自動再送を確認する。試験以外でSUIの既定無効フラグを有効化しない。
2. **構成の比較（#1016）**：受注明細WebとSUI Nativeの各々について、同じ構成手順を適用し、既存定義を再利用できた箇所／案件が新たに選んだ箇所／手作業で接続した箇所を計数する。SUIのDomain型と受注明細の業務意味を共通定義に畳まない。
3. **更新の検証（#1016）**：UI Kit更新、SUI Native Adapter更新、実Provider交換を異なる変更として扱い、対象外の依存まで一律失効させないことと、不明な互換性を成功扱いしないことを確認する。[E19](E19_GIT_REVISION_SOURCE_IMPACT.md)のGit改訂試験はTEI側だけの試験内改訂であり、SUIの更新判定を代替しない。

計測値の記入欄を次のように固定する。「―」は未計測であり、零を意味しない。

| 指標 | 受注明細Web | SUI Native | この段階の判定 |
| --- | ---: | ---: | --- |
| 新規の構成記述行数 | ― | ― | 未計測 |
| 変更を要する元ファイル数 | ― | ― | 未計測 |
| 手動の接続工程数 | ― | ― | 未計測 |
| 既存の正本との重複記述件数 | ― | ― | 未計測 |
| 人間への追加質問数 | ― | ― | 未計測 |
| 更新時の誤失効・見落とし件数 | ― | ― | 未計測 |
| 実Targetから独立読取りまでの成功 | ― | ― | 未実証 |

**判定**：既存の共通Action機構とNative実装を再利用する方向は確認できた。一方、二つの完成したApplicationの採用と接続コストは未実証である。新しい構成DSL・依存解決器・Lock・Core固定列挙の導入は保留し、未解決の型付き参照を含めた完全な閉包や本番認可も完了扱いしない。

# Application Composition — 案件構成を整理するためのAI用プロンプト（候補）

状態: 設計用の指示文。TEIの実装機能・自動構成器ではない。管理: [#1016](https://github.com/hat47x/tei-platform/issues/1016)／[設計候補](../../plan/design/PROJECT_PLUGIN_COMPOSITION_DESIGN.md)。

## 目的

短い依頼から既存のPlugin・UI構成キット・Architecture Profile候補・仕様テンプレートを再利用できるか調べ、**人間が選択すべき未確定部分と、実装・検証すべき差分**を小さくする。案件の業務意味や実行権限を、選択したProfileから勝手に決めない。

## 作業指示

与えられたProject / Applicationについて、次の順序で処理する。

1. **既知情報を取得する**。ProjectのSource / Accepted meaning / Target / Framework / Environment / 既存Plugin Lock / UI Kit / 現行Extensionを読む。既知条件は再質問しない。情報がないときは「未確認」として扱う。
2. **要求を分ける**。業務・Domainの意味、操作体験、アプリケーション構造（通常の画面処理、オフライン同期、Document Editor、リアルタイム等）、Target、Framework、外部接続、非機能要件に区分する。WebであるというだけでReactやCRUDを確定しない。
3. **候補を探索・検査する**。既存Plugin / Provider / Contract、UI構成キット（#997）、業務仕様テンプレート（#998）、既存Application Action / Semanticを優先して参照する。Target/Frameworkの整合、契約fingerprint、Authority scope、Provider Conformance、依存・Lockを確認する。探索順位・類似度を適合性の証明にしない。
4. **案件の変更を分類する**。`Configure`（設定変更）、`Compose`（資産の接続）、`Extend`（公開点への追加）、`Replace`（明示交換）、`Source fork`（元実装を案件が所有）、`Meaning change`（Accepted meaningの変更候補）を区別する。UI Kitのslot operationを他の契約へ転用しない。
5. **差分と影響を提示する**。どの資産・契約・Project-owned source・Environment bindingに変更が必要か、変えてはならない正本は何か、元Profile / Kitの更新で何が`unchanged / applicable / conflict / unresolved`となるか示す。
6. **不足を絞って尋ねる**。必要な契約・業務判断・権限が不足していれば質問する。分からないProviderやFrameworkを暗黙選択しない。未確認の業務規則をAIだけで採用しない。
7. **動作とEvidenceを区別する**。検証方法と失敗例を示す。実際に実行していないTest、構築していないTarget、採用されていないSourceについてPASS・完了と書かない。

## 出力形式

長い部品一覧よりも、以下の順序で簡潔に出力する。

- **案件の前提**: 確認できた事実／未確認条件。
- **採用できる既存構成**: Contract、Plugin、Kit、Template、Profile候補ごとに、適用根拠と採否。
- **案件で所有する差分**: Configure / Compose / Extend / Replace / Source fork / Meaning changeの分類、変更場所、元資産の更新との関係。
- **接続・Authority**: Application Action、Accepted meaning、UI state、Environment、Runtime Authorityを分けた接続図。
- **未決定事項**: 人による判断が必要な最少の質問と、判断前に進められる作業。
- **検証**: 正常系、契約や権限の不適合、版本・fingerprint不一致、更新衝突、実Targetでの検証条件と未実施事項。

## 留意事項

- 既存の「UI構成キット要件整理」は[こちら](../ui-customization/requirements-prompt.md)。本指示文はUIだけでなくProject全体の構成を扱い、UI側の詳細設計を複製しない。
- Plugin、構成Profile、初期Project Template、業務仕様Templateは異なる資産である。配布方式が同じでも正本やAuthorityを混同しない。
- ひとつの「標準Web構成」を唯一の正解としない。SUIのNative Canvas等、既存独自実装を保持した方がよい場合は正規の選択肢として提示する。

## UI以外の構成を欠かさないための確認観点

案件構成はUI Kitの採用だけでは閉じない。**必要になったものを明示的に選ぶ**ため、次の観点を既存のSourceと契約から確認する。すべての案件に同じ機構が必要だと仮定しない。

| 確認すること | 既存構成から探すもの | 設計上の注意 |
| --- | --- | --- |
| データを何に保存し誰が所有するか | Database / Storage Contract、logical datasource、Environment Binding | 画面のモデルや物理主キーを業務識別の正本にしない |
| ApplicationのActionをどう実現するか | Function / Capability契約、実装Provider、同期/非同期、transaction / failure boundary | Pluginの順序によって業務Actionを変えない |
| 認証・認可・秘密情報はどこで判定するか | Authentication/Authorization、Grant、Resource、Session・Tenant境界 | Profileを選んだだけではPermissionを付与しない |
| 外部サービスとどの契約で接続するか | Provider/Adapter、契約Version、失敗・再試行・冪等性 | APIが接続できるだけで意味の整合を証明しない |
| Offline/共同作業などの構造要件があるか | Application Architecture候補、同期、競合、Undo、状態所有 | UI Frameworkの選択だけで解決済みとしない |
| どこに配布して稼働を確かめるか | Target、Build/Release、Migration、部署環境、Health・Observation | Build成功と稼働観測を混同しない |
| 変更をどう検証し更新するか | Test/Conformance、元のSource、Plugin Lock、案件差分 | Versionが合うだけでImplementationの適合は確定しない |

画面の右パネルを変える依頼なら、その影響をUIに限定しても構わない。ただしDatabase切替やオフライン化、認証方式の変更なら、その責務のSourceと契約へ戻る。UIキットの`replace`を永続化・Security・業務の無制限上書き規則へ一般化しない。

出力は必要に応じて**「構成対象／既存の正本・Source／採用するContract・Provider・Target／案件固有の変更／Authority・Scope／検査する不整合／未実施の検証」**を一行ずつ整理する。重複する設定項目を新Profileへコピーせず、元のEnvironment、Application、Plugin Lock、Template採用へ参照で戻れるようにする。

既存Sourceの読取り例と、まだ構成で説明できない箇所の一覧は[受注明細とNative Canvasの構成要素調査](source-inventory.md)を参照する。これは設計用の確認材料であり、実行可能な構成Definitionではない。

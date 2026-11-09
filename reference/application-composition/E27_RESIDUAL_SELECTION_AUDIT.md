# E27：構成選択の残余判断を、既存ソースから再現可能に監査する

- **管理**：[Issue #1016](https://github.com/hat47x/tei-platform/issues/1016)。既存のApplication／Environment／Plugin Lock等に既にある選択と、案件ごとの追加判断・実行時の検証を混同しない。
- **基点**：2026-10-12、TEIの作業開始時点のmain `ab2eef5b37cfc83ce811ee2c60ced664336cbe8c`。SUI側の実装の出典・改訂比較については[E25](E25_CROSS_REPOSITORY_REQUALIFICATION.md)を参照し、この監査では**TEIリポジトリ外のSUIソースを読みに行かない**。
- **読取り監査**：[E27監査器](e27_residual_selection_audit.mjs)／[E27のNode.js回帰試験](e27_residual_selection_audit.test.mjs)。[E1](e1-observe.mjs)の既存14個の既知ソース観測をそのまま用いる。別のSource Index、依存解決、Profile Registryや選択保存先を追加しない。
- **E1の参照先追従**：受注明細の`/scenario/submit`登録は、現在の`tools/reference-first-web-target/order_scenario_http.go`にある。以前のE1観測器は旧`server.go`を見ていたため、実経路の移動に合わせて観測先を修正し、[E1回帰試験](e1-observe.test.mjs)で検査する。旧参照先のまま観測結果を成功と扱わない。

## まず既存ソースが何を語るか

E1は、TEIに存在する既知のソース14観測点（受注明細10、SUI Stage 0が4）について、固定した文字列とファイル内容を読み取る。**一致は文字列が存在したことの観測**であり、完全な構文検証・正本の受理・権限確認ではない。

E27はこのE1結果を10個の*監査上の確認項目*に分類する。これは実案件の手順書そのものや、必要な全判断の網羅一覧ではない。

| 利用先 | 既存の正本から観測すること | Sourceだけでは閉じない判断 |
| --- | --- | --- |
| 受注明細Web | Applicationの存在・Commandの参照 | 現在のAction実行権限 |
| 受注明細Web | ApplicationのDatabase要求と開発EnvironmentのBinding | 実際の接続権限・Provider適合・Migration観測 |
| 受注明細Web | EnvironmentのBundle参照と既存Plugin Lockの登録 | Kit選択・実配備の適合 |
| 受注明細Web | 既存UIアダプターと案件所有の拡張ソース | UI Kitと実Targetの正式採用・ブラウザ検証 |
| SUI Native | Stage 0のSource rootとIsland意味の一部分 | 正式なApplication／移動Actionの採用、入力契約の一致 |
| SUI Native | *Stage 0を観測できた*ということ | Native Framework／Targetの選択、認証済み委譲とDB CAS |

受注明細の参照先を、新しいProfileへ再び書くことを**改善とは数えない**。SUI Stage 0が読めたことも、SUIの実行可能なTEI Applicationが完成したことにはならない。

## 監査の機械可読な出力

```sh
node reference/application-composition/e27_residual_selection_audit.mjs .
node --test reference/application-composition/e27_residual_selection_audit.test.mjs
```

実ソースに変更がない基準点では、**10個の限定した監査項目**は次のように分類される。

| 監査上の判定 | 項目数 | 解釈 |
| --- | ---: | --- |
| `source-observed` | 4 | Web側の既存Application／DB Binding／Plugin参照／案件UIをソースから確認 |
| `stage0-observed-only` | 1 | SUIは部分的な意味参照を持つが、正式受理ではない |
| `selection-unresolved` | 3 | Web Kit/Target、SUI Action契約、SUI Native Kit/Targetの案件判断が別途必要 |
| `runtime-proof-unresolved` | 2 | Webの現在権限とSUIの認証済みCommit／CASは実行時証拠が必要 |
| `source-unresolved` | 0（基準点） | E1が対象とした既知文字列がすべて観測できた場合 |

`source-unresolved`はソース欠落や観測文字列の変化で増える。その場合、欠けたE1観測IDを添えて対象の監査項目だけを未解決とする。無関係なUI・SUI項目を一律に失効させない。E1の`observed`やSHA-256の形式が正しくても、本番の採用・認可の保証は行わない。

出力の`actualManualActions`、`actualAIQuestions`、`acceptedProfiles`は**すべてnull（未測定）**に固定する。3件の`selection-unresolved`は**監査で洗い出した意思決定項目数**であり、実際に人へ3回質問したという意味ではない。

## 比較可能な方法を先に固定する

独立Profileの採否には、二つの方式を*同じ案件・同じ実行先・同じ出発改訂*で測る必要がある。

| 同じ条件で測る項目 | 既存Sourceを直接選択する方式 | 追加Profile候補を経由する方式 |
| --- | --- | --- |
| 既存の正本の再利用 | E1のSource観測と既存Application/Environment/Lockを参照。重ね書きしない | 同じ正本を参照。複写フィールドがある場合は重複として計上 |
| 未解決判断の抽出 | E27の10監査項目・E10残余選択を読み取り提示 | 同じ監査項目・同じ所有者・同じ受入条件を使用 |
| 追加・変更ファイル | E26の実測基点から、実作業で記録 | 同じ基点から、Profile候補を含む実差分で記録 |
| 人間による判断／AIへの追加質問 | 実施した往復だけ数える。候補質問数と混同しない | 同じ定義・同じ停止条件で測定する |
| 更新・失効 | E25の限定変更の検証を使用。誤失効・見落としを記録 | 同じ変更を入力し、誤失効・見落としを記録 |
| 実Targetでの受入 | Web/SUIそれぞれの実動作・現在の権限・保存を確認 | 同一の実TargetとAuthorityを確認。選択済みと実行可能を混同しない |

このPRでは**追加Profileの実装も、本物の二方式の導入実験も行っていない**。従って行数・操作回数・時間の*改善率*を算出しない。手順を記録し、次回の実導入比較における採否を反証可能にした。

## 誰が何を決定するのか

- **案件開発者／採用審査者**：Architecture、Target、Framework／Kit、案件差分、採用するNative Actionの契約・版を選ぶ。独立したProfileを作る前に、既存の選択場所を優先する。
- **SUIとTEI Action担当（#993）**：真の本人確認、Tenant/Resource権限、Payloadの解釈と書込み、CAS・結果不明時の扱いを実装・検査する。このE27は所有を引き取らない。
- **実行時の認可・Release担当**：実Provider、現在のPermission／Secret／Network、移行・稼働観測を判断する。構成候補へ秘密値や承認済みフラグをコピーしない。
- **Source・Plugin担当**：実改訂とSource指紋の出典、既存Plugin Lock、依存関係の正規追跡を提供する。別のLockやGraphを作らない。

## 判定

**現状：構成Profileの製品採用は保留。** E27により、既存ソースで所在が分かっている箇所を、案件ごとの未決定事項から分離できる。しかし、正式なWeb UI Kit/Target、SUI Nativeの正式接続・実認証／保存、比較可能な実際の人手工数とAIへの質問数がそろっていない。新しいCore種別、独立したPlugin Lock、第二Resolver、構成済みの擬似フラグは導入しない。

[E10の採否条件](E10_COMPOSITION_ADOPTION_DECISION.md)と[E26の記述量基準](E26_COMPOSITION_COST_BASELINE.md)を維持し、実導入手順と検証の経過を蓄積してから、追加Profileが本当に重複や往復を減らすか判断する。

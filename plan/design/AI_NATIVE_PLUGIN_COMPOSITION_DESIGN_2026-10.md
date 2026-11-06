# AIが扱いやすい最小プラグイン規約と可変な製品構成（設計候補）

> **後続判断（2026-10-24）**：[ADR-0043](../adr/0043-target-native-artifacts-without-tei-host.md)に従い、以下の`Host`に関する記述は既存の対象環境または個別の実行・検証方式として読みます。TEI独自の共通Hostの設計採用や、全ターゲットでのRuntime管理機構を要求するものではありません。

- **状態**：提案・検証前（2026-10-12）。新しいManifest構文、Production API、実装済み機能を意味しない。
- **目的**：標準機能を含めて製品構成を交換可能にし、同じ仕組みを生成AIの設計・実装・検証の足場とする。
- **関連する判断**：[ADR-0032](../adr/0032-composition-kernel-replacement.md)、[Issue #1223](https://github.com/hat47x/tei-platform/issues/1223)、[#951](https://github.com/hat47x/tei-platform/issues/951)、[#943](https://github.com/hat47x/tei-platform/issues/943)、[#1016](https://github.com/hat47x/tei-platform/issues/1016)。
- **既存の担当文書**：[案件構成](PROJECT_PLUGIN_COMPOSITION_DESIGN.md)、[プラグインのセルフビルド](PLUGIN_SELF_BUILD_PRODUCTION_CONTRACT_2026-10.md)、[実行Target別の構成](RUNTIME_TARGET_BINARY_COMPOSITION_DESIGN.md)。本書はこれらを置き換えず、AIによる実装のための横断規則を整理する。
- **非対象**：Cordis/DSHのAPI移植、Plugin Packageの一律細分化、AIによる権限・業務意味の自動承認、実行時の完全な復元・無停止更新の保証。

## 1. なぜ最小の規約が重要か

TEIがプラグイン機構を導入する目的は、拡張機能を後付けするためだけではない。標準機能を含む製品の構成を利用目的ごとに交換できるようにするためである。

この可変性は、生成AIが設計・実装する際にも有効に働く。必要なサービス契約、実装の候補、適用範囲と検証条件が見えるなら、AIは既存コードを丸ごと模倣するより先に、再利用・接続・差分実装・共通部分の抽出を選べる。

制約を設けない自由なコード生成には、実装様式の分散、既存の安全境界の見落とし、重複する基盤機構の発生という欠点がある。一方、規約を過大にすると、個別機能へ不要な定型コードを強制する。**TEIの規約は実装方法を固定するものではなく、合成と検証に必要な意味だけを表す。**

DeepSeek Harness（DSH）は、製品機能をCordisプラグインとして組み合わせ、プロファイルと構成差分で実装を交換している。また、能力の接続点をService Definition／Provider／Consumerの三者として扱う。TEIではこの価値観を参考にしつつ、論理構成から静的Go、軽量Wasm、デスクトップ、分散Hostへ最適化する部分を独自に設計する。

## 2. 必要な最小情報

一つのプラグインを、機能の論理的な合成単位として考える。配布パッケージ、ソースディレクトリ、プロセス、生成物、実行インスタンスと常に一対一にはならない。

| 要素 | 最小限求める意味 | 省略・導出を認める条件 |
| --- | --- | --- |
| Identity | 何の機能定義か。参照先を一意に指定できること | 既存のCanonical IDから一意に導けるなら別記しない |
| Provides | 外部へ提供するService Contractと適合関係 | Sourceの型・定義から曖昧さなく導けるなら記述を生成する |
| Requires | 必要なContractと選択・適用範囲 | 依存がない純粋計算では空。単なるソースimportを業務契約の要求と混同しない |
| Effect / Lifetime | 所有する一時的作用、解放責任、永続作用との境界 | 純粋計算では「なし」を導出。実際に所有する作用がある場合だけ追加規約を要求する |
| Realization conditions | Target、必要な資源、外部権限、実行可能な形態 | 制約が存在しない場合はTarget共通として扱える。ただし未調査を「制約なし」と推定しない |

Contractが扱う入出力・失敗・副作用の意味を、Providerが独自に再定義しない。Contract definitionと実装の適合性を照合するConformanceは、Plugin生成物とは独立した基準に置く。

Manifest、言語型、Canonical Source、生成されたLockが同じ情報を重複保持する場合は、どれが正本かを定め、派生物を検証可能に生成する。**最小規約は最小の手書き項目数と同義ではなく、正本の重複を最小にする規約**とする。

### 表現例：純粋な文字列正規化サービス

以下は、規約に必要な情報を説明するための**概念例**であり、現行TEIで読み込めるDSLではない。

```yaml
id: example.text-normalizer
provides:
  - contract: example.text.normalize@1
    conformance: reference:text-normalizer-v1
requires: []
effects: none
```

実装言語、ディレクトリ構造、Loader、Logger、権限プラグインはここでは指定していない。言語ごとの実装を追加しても、Consumerが必要とする契約は変わらない。

外部辞書を読む別Providerへ交換するなら、必要な読み取り契約と失敗条件が新たに生じる可能性がある。「同じ文字列が返る」だけで、以前の純粋関数と同等とは扱わない。契約の更新か、依存先を取り込んだ適合証拠が必要になる。

## 3. AIに許す変更の種類を区別する

AIはまず既存資産を調べる。必要なContractが見つからないからといって、最初に新しい共通Frameworkを実装するべきではない。

| 操作 | 例 | 追加の受入条件 |
| --- | --- | --- |
| 再利用 | 既存の認証Providerを案件へ接続する | 契約、Scope、現在のAuthorityとTargetが適合する |
| 組み合わせ | データ読取と表示サービスを接続する | 依存が一意に解決され、作用や権限が増幅しない |
| 交換 | DBの実現をローカルから遠隔へ変える | Consumerの契約を維持し、変更後の失敗・遅延・整合条件を満たす |
| 差分実装 | 既存の読取Providerへ必要な変換を追加する | 何を継承し、何を変更したか追跡できる |
| 共通化 | 二案件の類似処理から共有機能を抽出する | 二つのConsumerの契約差を隠さず、両方の回帰試験を通す |
| 撤去 | 未使用のServiceを構成から外す | 依存側、Effect、Artifactの残余と業務状態を確認する |

**参考実装は契約の証拠ではない。** AIが既存実装を参考にするときは、対応するContract、採用済みの慣例、暫定的な回避策、旧仕様の互換コードを区別する。複数の実装例から同じパターンが見つかっても、その一般化を暗黙の新標準としない。

### AIが参照する情報の最小セット

AIの探索入口には、既存のSource・Contract・Provider・Consumer・適合試験・構成Profileから導いた読み取り専用の索引を用意する。別個の手書きPluginカタログを第二の正本として増やさない。

AIが機能を選ぶ際は、対象Contractと使用例、必要な権限、各Targetでの実現可能性、検証済み範囲、実装の鮮度、取得した正本への参照が分かればよい。情報が欠ける項目は「未確認」として扱う。

## 4. 一貫した生成・検証手順

次の段階を独立させる。生成AIをすべての段階の正本または承認者にしない。

1. **要求の取得**：何が必要か、どこが案件固有か、変更してはいけない業務意味は何かを明示する。
2. **既存資産の探索**：対象Contractに適合する既存Provider、用途に近いConsumer、共通実装候補を探す。
3. **構成案の比較**：新規実装、既存の交換、Adapterの追加などを比較し、変更範囲と推定負荷を示す。Providerが曖昧なら勝手に選ばない。
4. **最小差分の実装**：未充足の機能だけを生成する。既存Kernel、Resolver、権限管理を重複実装しない。
5. **独立した契約検証**：生成コードが提供する機能を、生成物とは別のConformance・否定試験・既存回帰試験で確かめる。
6. **受入と実現**：検証済み候補を正式な受理境界で採否判断し、Targetごとにビルド・配備する。Sourceの受入とRuntime Authorityは区別する。
7. **結果の観測と再利用**：実際の問題、Target差、生成の重複を次の参考例へ反映する。実行結果を黙ってContractへ書き戻さない。

変更の根拠を再構築できるよう、AIの提案には、参照した正本・Contract・既存例の出典、変更点、検証範囲、未確認の前提を残す。これらは新たなビルド署名を生成するものではなく、採用理由を追跡するための証拠である。

## 5. 製品構成の可変性を実証する

TEIの価値を、プラグインの登録数で測らない。同じ機能をどれだけ少ない変更で利用できるかを測る。

**参照例A：受注データを表示する業務Webアプリ**

既存の業務モデル、読取契約、UI構成を使う。ローカルDBを遠隔Providerへ変更する際、Consumerのソースを変更せずに済むこと、書込権限や業務上の意味が拡大しないことを検証する。

**参照例B：SUI型の知識整理アプリ**

Native Canvasなど独自UIを利用しながら、共通の保存・検索・同期契約を再利用する。SUI固有のDocumentやCardをTEIの業務モデルへ強制的に変換しない。業務Webアプリ用の暗黙前提がなければ利用できない共通機構は、汎用化の候補から外す。

この二例で共通のContractを導入する場合、同名のサービスがあることではなく、両者のConsumerが同じ失敗・状態・権限の意味を受け入れられることを確認する。

### 評価指標

| 指標 | 観測したい価値 |
| --- | --- |
| 既存実装の再利用率 | 新規コードを生成せず、要求を満たした割合 |
| 変更範囲 | Provider交換時のConsumer、Source、Testの変更箇所 |
| 人間による追加確認 | 仕様の不明確さによって発生した質問と修正 |
| 契約逸脱の検出 | 異なる失敗・Effect・権限を誤って互換扱いしないこと |
| 重複定義・定型コード | 手書きの重複と、生成器へ移せる部分 |
| 配備コスト | Target別のバイナリサイズ、メモリ、起動時間、実行遅延 |
| 変更後の回帰 | 既存Consumerが満たすべき条件を維持できたか |

数値目標はまだ設定しない。まず既存の手動作業と比較し、差が出た理由を特定する。

## 6. 最小規約が失敗する事例

| 事例 | 拒否・診断すべき結果 |
| --- | --- |
| 異なる契約なのに同じメソッド名だから互換と判断 | 契約と失敗・Effectの意味が不一致 |
| 同一契約に二つのProviderが存在する | 明示選択または十分に定義された選択規則を要求 |
| AIが既存の認証実装を参考に新しいGuardを自作 | 信頼境界の逸脱として受入拒否 |
| テストとProviderを同時生成し、双方が同じ誤解を含む | 独立Conformanceと既存Consumer回帰に照合 |
| 構成Profileが受理済みの業務意味を変更する | Profileの責務を超える変更として拒否 |
| Pluginを撤去したら永続的な業務履歴も削除する | Effect Scopeと永続的な事実の混同として拒否 |
| 純粋な静的Targetに動的Loaderを必須とする | 不要な実行時コストとして失敗 |
| 別案件の類似コードを根拠なく共有化 | 共通Contractの受入根拠不足として保留 |
| Sourceだけ更新し、構成・Build Artifact・Actual Hostが旧版のまま | 設計・ビルド・配置・実行の証拠を区別して不一致を表示 |

## 7. 実証の順序と判断保留事項

1. **読み取り専用の索引**：#951のContract情報、#1016の構成Profile、#943のビルド証拠、既存ソースを対応付ける。新しい二重正本を作らない。
2. **単純な関数Provider**：外部依存・Effectのない例で、手書き必須規約を測定する。
3. **外部資源を持つProvider**：DB読取またはファイルアクセスで、権限・失敗・Scope・Target適合を評価する。
4. **二つのConsumerと交換**：受注Webアプリと非業務アプリを使い、再利用と契約不適合を計測する。
5. **共通部分の抽出**：AIによる共通Plugin候補と、手動で選択した構成の差を比較し、両方の回帰試験を実行する。
6. **静的Lowering**：対象Targetで不要なLoader/Watcherを生成しないことを実測する。

初期試作を通じて、Identity/Provides/Requiresの情報をどこから安全に導出できるかを決める。特定の形式のManifestを先に標準採用しない。現時点の`internal/composition`は実行時の独立Referenceであり、本書のAI探索・生成・静的最適化機構を実装済みとはみなさない。

### 見直すべき条件

- 標準Pluginを一件追加するだけで多量の繰返しコードが必要になる。
- 「共通化」により二つのConsumerの業務意味や失敗契約が曖昧になる。
- 固定の層や`CapabilityKind`列挙を増やすたびにCore変更が必要になる。
- 各Targetに必須ではない実行時機構が残る。
- AIの提案を独立検証するために、毎回手作業の試験器を作り直す。

この場合は、個別Pluginの実装負荷を増やすのではなく、共通規約・導出器・Conformance・構成方式を再検討する。

## 8. 参照したDSHの設計とTEIへの解釈

- [DeepSeek Harness公式アーキテクチャ](https://github.com/deepseek-ai/deepseek-harness/blob/master/docs/architecture.md)：製品機能をPluginとして構成し、Profile/Bundle/構成Patchで交換する。
- [DSHのServiceと依存](https://deepseek-harness.github.io/deepseek-harness/en/develop/framework/service)：サービス要求の宣言、Provider消失時の依存解消と復帰。
- [DSHのAGENTS.md](https://github.com/deepseek-ai/deepseek-harness/blob/master/AGENTS.md)：Service Definition・Provider・Consumerで一つの能力の接続点を捉える。実装境界では暗黙の初期値を避ける。
- [DSHの生成された構成一覧](https://deepseek-harness.github.io/deepseek-harness/en/reference/config-catalog)：実装定義から設定可能な型や依存を抽出し、生成済み文書の鮮度を検査する。

特に最後は、TEIでAI向けの機能索引を独立した手書き正本として増やさず、実際のSource・Contract・構成から生成する理由になる。DSHのNode/TypeScript固有の実装形式を標準化する根拠にはしない。


## 9. AIが実装を始める前の「充足判定」

### 9.1 目的

「必要な機能が存在する」と「現在の構成で利用してよい」は異なる。名前が似たProviderを見つけただけで実装不要と判断したり、Contractの候補が見つからないだけで新実装が必要と決めたりすると、構成の可変性はかえって品質を下げる。

そこで、生成AIによる実装前の作業を**要求の充足判定**として扱う。これはAI専用のResolverを追加する提案ではない。AIは既存のCanonical Source、Contract、適合性検査、構成Profile、現在の実装状況を読み取り、候補と不足条件を説明する。最終的なProvider選択・意味の受理・実行権限は既存の担当境界に委ねる。

判定には、少なくとも次の四つの区別が必要となる。

| 判定 | 意味 | AIの次の行為 |
| --- | --- | --- |
| **充足候補** | 契約と依存が整合し、利用できる実装候補と検証根拠がそろう | 構成案へ組み込み、正式な受理・配備検証に渡す |
| **不足** | 確認した範囲に、要求する契約を満たす提供手段がない | 既存の近接例を探し、不足部分の実装を提案する |
| **競合** | 相互に整合しない候補・選択規則がある | 候補と違いを示し、明示的な選択・修正を求める |
| **未判定** | 正本、適合試験、Target条件などが不足し判断できない | 不足した根拠を列挙して調査・検証する |

「不足」を確定できるのは、探索対象の範囲が明確な場合だけである。索引が不完全なら「見つからない」を「存在しない」に変換しない。また、充足候補であっても、現在のHost Authority、実Artifactの信頼性、稼働状態、利用時点の権限が確認されたことにはならない。

### 9.2 充足判定の順序

一つの利用側要求について、次の条件を順番に確認する。

1. **意味の照合**：契約ID・版・入出力・失敗・未確定値・作用の意味が合うか。形だけが一致する実装を互換としない。
2. **利用範囲の照合**：組織・案件・実行主体・データ境界などの適用範囲に収まるか。
3. **実装の適合性**：Providerが契約に適合したという独立の試験・根拠があるか。提供者自身の宣言だけで確定しない。
4. **依存の充足**：要求された別のサービスまで辿り、欠落・循環・候補の競合を見つける。Optionalや代替条件は、契約に根拠がある場合のみ評価する。
5. **実現可能性**：Targetの制限、必要資源、ビルド手段、配布形式、相互接続方式と矛盾しないか。
6. **受理前提**：どの業務意味、構成、適合根拠、実装が検証済みで、どこから未確認かを整理する。

この判定は**設計時の可否判断**であり、実行時の許可判定ではない。実際の起動・呼出しでは、固定された構成・Artifact・現在のAuthority・稼働状態をHostが改めて検証する。

### 9.3 判定の説明責任

判定結果には「なぜ選べるか」「なぜ選べないか」を残す。ただし巨大な証明書や二重のLock形式は追加しない。既存の正本へ戻れる**導出された根拠一覧**として表す。

| 根拠の種類 | 参照元と意味 |
| --- | --- |
| 要求 | 案件のSource、受理された業務モデル、明示されたTarget |
| 契約 | 正本となるService Contract、適用範囲、必要な作用 |
| 候補 | 対象Provider、必要に応じて固定版・実Artifactへの参照 |
| 適合 | 別途管理された正負の試験、対象版、適用可能な条件 |
| 構成 | 既存の合成定義と、追加・交換・撤去する差分 |
| 未確認 | 探索範囲、欠けた資料、必要な実環境確認 |

AIの生成した説明文や類似度は証拠の探索補助に使えるが、契約の適合性や安全性を自動的に確定する根拠にしない。生成した報告には参照先の識別子と取得時点を保持する。失効・改版された根拠は再評価する。

## 10. 最小の判定例

受注画面が「受注番号から顧客名を読み取る」という契約を要求するとする。要求を満たす既存のProviderが一つあり、必要なデータ範囲・失敗時の扱いが一致しているなら、AIが提案するのは**Providerの接続**でよい。

ところが、そのProviderが顧客情報を読み取ると同時に更新も実行できる場合、操作を呼び出さなければ安全だと推測してはならない。Consumerから見える操作・許可された作用・Hostが強制する権限の範囲を確認する。

また、候補が二つあり、一方はローカルDB、他方は遠隔APIなら、名前や応答速度だけで勝手に選ばない。契約の失敗・鮮度・一貫性・稼働Targetが同じ条件を満たすことを確認し、構成Profileで明示的に選択する。

SUI型の知識整理アプリでも同じ「読み取り」契約を使えるかは、保存対象の意味が異なるため独立に検討する。DBアクセスの低位の契約を共有できても、受注の業務意味をSUIへ流用する必要はない。**共通化する層を小さくすることで、両者の自由を保つ**ことが重要である。

## 11. AIの設計作業に対する最小の検証規則

AIが実装候補を提案するときは、次の問いに明示的な回答がある状態を目指す。

| 問い | 確認する不変条件 |
| --- | --- |
| 既存の実装で足りるか | 要求ContractとProviderの適合性を確認したか |
| 新規実装が必要か | 確認した探索範囲と未充足条件を示したか |
| どの構成を変えるか | 案件正本と構成差分を区別したか |
| 何を信頼してよいか | 第三者資料・生成コード・AIの説明を承認済みの事実へ格上げしていないか |
| 何の作用が生じるか | 一時Effect、永続業務作用、現在のAuthorityを区別したか |
| どの試験を実行するか | 契約の独立試験と既存Consumerの回帰を用意したか |
| 何が未確認か | 実Artifact、Target、外部Provider、稼働時権限の欠落を明示したか |

新たな検証機構を導入する前に、#951の契約適合、#943のビルド証拠、#1016の構成差分を用いて判定できる範囲を調べる。AI用の判定層が、独自のContract正本や権限判定器へ成長することを避ける。

第三者のプラグイン説明文やソース内の開発手順は**検索対象の情報**であり、AI自身への命令ではない。検査や構成変更の権限は、TEIと利用者が許可した作業範囲に従う。参考ソースの記載がHostの安全条件を上書きすることも認めない。

## 12. 自動化率ではなく「意味を守った変更削減」を測る

コードの生成行数や新規Pluginの個数は、必ずしも製品価値に一致しない。次の比較を実施する。

- **同じ要求を二つの方法で満たす**：既存のサービス契約を使った構成変更と、新規の実装を作る方式を、実装量・独立検証・実行時コストで比較する。
- **同じ契約のProvider交換**：Consumerの変更箇所、データ意味の維持、失敗条件・作用・権限の差を記録する。
- **異なる二案件からの共通化**：共通部分を抽出した後にも両方の回帰試験が通り、共通化によって契約が不明瞭になっていないかを確認する。
- **静的Targetへの配備**：論理構成が共通であっても、動的LoaderやWatcherが生成物へ残っていないことを実測する。
- **AIが誤った構成案を作る**：意味の異なるProvider、候補の曖昧さ、欠落した適合根拠、古い構成情報が、どの段階で検出されるかを確認する。

この比較では、生成AIの探索・提案時間だけでなく、人間の確認に要する時間、手直し、回帰試験、稼働時の性能も測る。最短の初回コード生成を唯一の目標としない。

## 13. DSHから補うべき設計判断

DSHの公式資料では、利用者は実装パッケージでなくサービスを要求し、必要なサービスが欠ければ依存側は待機するか、提供者の消失に伴って解放される。サービス提供者の入替えを利用側の変更から独立させることが、実際の実装方針となっている。

- [サービスと依存の公式説明](https://github.com/deepseek-ai/deepseek-harness/blob/master/docs/user/develop/framework/service.md)
- [能力の接続点に関する実装判断](https://github.com/deepseek-ai/deepseek-harness/blob/master/.agents/notes/implemented/architecture/2026-06-13-capability-seams.md)
- [DSHの実装規約](https://github.com/deepseek-ai/deepseek-harness/blob/master/packages/AGENTS.md)

とくにDSHでは、利用側の事情に引きずられた過剰な共通化を避け、**現在必要な契約と所有者があること**を抽象化の根拠とする。TEIでも同様に、AIが抽象化を提案したという理由だけで新しい共通Pluginを採用せず、実際の利用側と独立検証が得られた時点で昇格させる。


## 14. 最初の実装接続：既存登録簿から構成案を診断する

`internal/composition/proposal.go`を小さな参照実装として追加した。生成AIが参照した契約・適用範囲・任意の実装指定について、**既存のRegistryに登録された内容だけ**から合成可能性の候補を示す。

判定は`candidate`（構成上の候補）、`conflict`（候補競合または明示的な契約不一致）、`unknown`（登録簿の不足、依存条件の不成立、入力不明）の三種類とする。本書第9節の「不足」は、信頼できる検索範囲の完全性が示された場合に初めて確定できるため、**現段階では出力しない**。

これはAIに新たな権限を与える機構ではない。最初の機械的な検査として、次を示すことに限定する。

- 契約のID・版・形状とScopeを照合する。
- 候補が複数あるときは、勝手に実装を選ばない。
- 対象Providerに依存不足があれば、利用可能とは報告しない。
- 別TenantのProviderの存在は、候補探索結果から区別できないようにする。
- 選択された依存とProvider Epochを含む結果をコピーして返すが、実行時にはHostで再検査する。

独立したGo試験用土台での確認は完了した一方、公開GitHubからの通常の`git clone`はこの実行環境のDNS制約により未成立であり、**TEI本体の全体試験は未実施**。証拠の範囲と詳細は[Composition KernelのE5](../../reference/composition-kernel/README.md)に記録する。

次に必要となるのは、#951の適合証拠と#943の成果物証拠を、AIの提案根拠から正本へ辿れる読み取り専用の参照関係として接続すること。その後に、複数の要求を束ねた構成差分の検査へ進む。ここでも第二のResolverやAuthorityは導入しない。


## 15. 依存先の変更を踏まえた充足判定

構成案の充足は、直接利用するProviderが変わらなければ維持されるとは限らない。例えば、画面が参照する読取サービス自身は変更されなくても、その内部で利用するDB Providerの構成が変われば、古い構成案の再検証が必要になる。

このため、`Selection.DependencyEpochs`（推移的な依存先の宣言世代）を参照実装へ追加した。構成案に挙げたProviderについて、直接・間接の依存先が変わった場合には、以前の判断を失効させる。一方、依存しない別の機能や別Tenantの更新によって、不要な失効を起こさない。

この方式は**宣言の鮮度**を検査するにとどまり、実装の意味上の適合、ビルド成果物の正当性、現在の権限、実際の依存先インスタンスの生存は証明しない。AIの構成案が「構造上の候補」であることは変わらない。

詳細な試験条件と性能上の未解決点は、[Composition Kernel E6](../../reference/composition-kernel/README.md)にまとめる。今後は#951の契約適合証拠、#943の実成果物、Hostの現在権限を、構成案の**独立した確認項目**として接続する。


## 16. 接続箇所の同一性と、複数要求の整合性

一つのアプリケーションが同じサービス契約を複数箇所で利用する場合がある。例えば、比較画面が同じ読取契約に従いながら、主データと比較用データに別々の提供者を選ぶことは自然である。一方、**同じ画面項目の同じ接続箇所へ、異なる提供者を同時に指定する構成案**は不整合である。

この二つを区別するため、試作の`WiringRequirement`に`SlotID`を追加した。異なる接続箇所には異なる識別子を付ける。同じ接続箇所へ異なる契約や提供者を指定した構成案は`inconsistent-slot`として拒否する。同じ接続箇所に一つの明示的な提供者指定があり、他の要求が未指定なら、指定された提供者をその接続箇所で共有する。接続箇所が異なれば指定の継承は行わない。

`SlotID`の省略時は、**同じ完全な契約を要求する行を一つの接続箇所として扱う**暫定規則とした。複数の独立したConsumerが同じ契約を利用する場合には、接続箇所を明示する。

この規則は、将来の業務モデル、機能モデル、UI構成から接続箇所を導出するための最初の参照形である。識別子を毎回手書きすることを恒久的な規約としない。既存の画面項目、操作、ConsumerのSource identityから一意に導けるなら、別の定義を増やさない。

### 契約適合証拠との接続順序

構造上の判定を終えても、そのまま自動的に「実装済み」「安全」「実行可能」へ昇格させない。今後の接続は次の境界を順に扱う。

1. **契約上の意味**：#951の契約正本に対して、提供者がどの版・入出力・失敗・作用の意味を満たすかを判定する。文字列上の契約形状一致は十分な証拠ではない。
2. **適合性の証拠**：独立した正負の適合試験とその対象版・発行者・検証範囲を参照し、欠けた証拠を「未判定」とする。提供者が記載した自己評価のみで受理しない。
3. **ビルド成果物**：#943の実成果物と正確な入力・生成物・証明の結び付きを検証する。契約に適合したソースの存在は、現在の配備物の正当性を示さない。
4. **現在の実行権限**：Hostの現在の権限、失効、実行先の制約、実際の準備完了を別に検証する。過去に一度合格した証拠を恒久的な実行許可にしない。

この確認列は新しい承認基盤を増やすためのものではない。既存の正本へ辿れる不足条件の一覧を作り、各責務を持つ検証機構へ引き渡すためのものである。第14・15節と同様に、`internal/composition`の実行Kernelへ業務固有の意味検証や証拠署名機構を追加しない。

### 反証すべき事例

- 一括診断の各行が個別に成立しても、同じ接続箇所に異なるProvider指定がある場合は拒否する。
- 同じ契約を別々の接続箇所で使い、異なるProviderを指定できることを保つ。
- Provider ID、契約ID、適用範囲、宣言世代が同じでも、適合証拠が無い場合には実行許可を出さない。
- 旧版の成果物へ適合証拠があっても、新版の実成果物を無条件に承認しない。
- コンパイラが静的に接続箇所を確定しても、動的更新に必要な権限検査を無条件に省かない。

試験と未保証事項は[Composition Kernel E7](../../reference/composition-kernel/README.md)に記す。


## 17. 構成案の充足と、独立した検証に必要な根拠の接続

AIの構成案が「構造上の候補」であることを確認できても、参照するProviderが契約の意味を満たし、現在の配備物として利用可能であるとは限らない。

E8では、`WiringFinding.VerificationTargets`を追加し、構造上の候補と、その直接・間接の依存先を**独立した検証へ渡す対象**として列挙できるようにした。各対象はProvider ID、Scope、Service Contractの正確な識別情報、Registry上の宣言世代を持つ。AIが「既存機能で実装不要」と結論を出す前に、どのProviderを確認する必要があるかが分かる。

ただし、対象の一覧だけでは発行者の権限も、Conformance Evidenceも、Build Artifactも確かめられない。#951のCapability Contract Referenceには、実装のPlugin ID・版・Implementation ID・Artifact Digest・Contract Fingerprint・Conformance Profile・Evidence Fingerprintへの結び付けがあるため、それを**別の正本として複写せず、参照関係として接続する**。

さらに、Providerの契約適合が確認されても、それを現在のHostで利用してよいかは独立した判断とする。実際のPermission、Resource、Lease、FencingはHostのAuthorityを通す。したがって判定の段階は次のように区別する。

| 段階 | 確認すること | 現在の状態 |
| --- | --- | --- |
| 1. 構成の成立 | 契約、接続箇所、依存関係、範囲、宣言世代 | Registry Referenceで局所検証済み |
| 2. 検証対象の網羅 | 候補と推移的な依存先を漏れなく列挙 | E8で局所検証済み |
| 3. 契約の意味と適合 | #951の定義・実装・適合証拠と現在の信頼状態 | 未接続 |
| 4. 実成果物の確認 | #943のBuild Artifactと実配備対象を正確に対応付ける | 未接続 |
| 5. 実行時の受理 | Hostの現在の権限、Lease、Resource、Ready | 未接続 |

この分離によって、AIの提案を**根拠不足のまま承認しない**ことと、必要な根拠を収集する作業を自動化することの両立を目指す。

実装と局所試験の詳細、未成立の保証は[Composition Kernel E8](../../reference/composition-kernel/README.md)に記載する。


## 18. 契約適合証拠への参照を、実行機構と分けて照合する

AIによる構成案の生成では、E8で列挙した検証対象と既存の#951の解決・適合性証拠を対応付ける必要がある。しかし、構成案の検査Kernelが証拠の信頼性や実行権限を自前で判定すると、既存の正式なAuthorityと二重化する。

これを避けるため、[E9の独立した参照実装](../../reference/composition-evidence-handoff/README.md)を追加した。対象はProvider ID、Scope、契約ID・版・形状、宣言世代で照合する。結果には、参照不足、宣言世代の古さ、対象の不一致、参照項目の不足、複数候補の曖昧さを区別して残す。

一方、#951のConformance Evidenceに含まれるPlugin ID・Plugin版・Implementation ID・Artifact Digest・Contract Fingerprint・Conformance Profile・Evidence Fingerprintは、あくまで**独立検証へ戻るための参照項目**として受け取る。構成案の`Contract.Shape`だけで、#951の契約全体の正確な指紋を推定できない。

したがって、すべての参照が一致しても、最終結果は**参照先を発見した（未検証）**で止める。契約定義の発行権限、適合試験の真正性・失効状態、実Artifactと配備対象、現在のHost Permission／Leaseは別々に検証する。AIの自己評価では確定しない。

本書の一連のE5～E9は、要求から構成案を作り、外部検証で必要となる根拠の不足を列挙できるまでのReferenceである。Productionの完全な契約適合、配備、AIによる自律的な実装承認を示すものではない。正式な#951のResolverを通す縦断Consumerと、PR exact-headの全体テストを次の採用条件とする。


## 19. 既存の契約解決器を使う縦断接続（E10）

E9の照合機構が受け取る証拠参照を、実際に#951の`resolveCapability`から得る経路を、同Referenceの`composition_handoff.go`として追加した。AI専用の契約解決器を作らず、#951が採用する契約・適合証拠・提供者の選択規則を利用する。

構成案で選ばれたProviderには、#951側の具体的なPlugin ID・版・実装ID・成果物ダイジェストを**明示的に結び付ける**。契約IDの似た実装を自動で対応付けたり、意味の異なる適用範囲を暗黙に統合したりしない。解決後には、正確な契約版、契約指紋、提供者の識別情報、成果物を再照合してからE9の参照形式へ変換する。

これにより、AIが構成案を作る際に必要な確認を次の三段階として分けられる。

1. **構造上の合成**：どのConsumerがどの契約・Provider・依存先を必要とするか。
2. **契約解決の参照**：既存の#951が、明示した実装に対応する解決結果を生成できるか。
3. **独立した受理**：契約定義と適合証拠の発行権限・失効・成果物の正当性・現在のHost権限を、担当する検証機構で確認する。

今回は第2段階の接続コードと試験をGitHubへ追加した。第3段階は未実装であり、**第2段階の成功を第3段階の承認へ昇格させない**。また、Compositionの`Scope`と#951の`AuthorityScope`、Compositionの`Contract.Shape`と#951の`ContractFingerprint`は、それぞれ別の概念のままである。

具体的な否定試験と検証の未完了事項は[証拠照合ReferenceのE10](../../reference/composition-evidence-handoff/README.md)にまとめた。E9の局所試験は成功したが、新しいE10の#951実Packageを含むテスト、完全なリポジトリの健全性検査はまだ実行していない。


## 20. 構成案の鮮度と証拠照合の時点を区別する

AIがプラグイン構成案を作成した後に、下位のProviderが変更・撤去される場合がある。このとき、過去に合格した構成案と証拠参照の組合せが一致していても、**現在のRegistryに照らして有効とは限らない**。

E11では、構成案の選択と直接・間接の依存先の宣言世代を、現在のRegistryと再照合する`ProposalCurrent`を追加した。E9の照合結果を現在の候補として扱う際には`ReconcileCurrent`を利用する。古い構成案や改変された構成案は遮断する。無関係なProviderの更新までは失効させない。

ただし、鮮度を確認した直後にもRegistryは変更され得る。したがって、これはあくまで設計時の観測であり、**実行時のHostによる選択の再検証、権限、Lease、Fencingを代替しない**。証拠の発行者の信頼性や現在性も別の判断として残す。

E11の試験ソースはPRへ追加済みだが、実Compositionと#951を含む結合試験および信頼できる全Repository Healthは未実施である。変更の目的と否定試験は[照合Reference E11](../../reference/composition-evidence-handoff/README.md)へ記録する。


## 21. DSH比較から導く構成・実使用・証拠の非干渉契約（2026-11-06、設計・未採用）

**管理・適用**：[横断研究 #2899](https://github.com/hat47x/tei-platform/issues/2899)。本節はDeepSeek Harness（DSH）の構成方式を参照して、既存E5〜E11の構成案・証拠照合を**実際の消費・Target・現在資格**へ受け渡すための反証を定義する。#1016のProfile、#951のResolver、#943/#1089のBuild、#1121のLowering、#1174のHost Effect、#2044のStudio、#2589の公開Registryを再実装しない。旧Referenceを本番合格へ昇格しない。

### 21.1 外部実装から確認した事実と、採用しない類推

- [DSH Architecture](https://github.com/deepseek-ai/deepseek-harness/blob/master/docs/architecture.md)：Cordisサービス・イベント・可逆登録効果、順序付きBundle→Profile→home→CLI patch。行単位のpatchは同じidの**全体置換**であり、部分フィールドの意味保存を自動保証しない。sdk-minimalは共通baseを必須としない。
- [DSH Plugin Manager](https://github.com/deepseek-ai/deepseek-harness/blob/master/packages/boot/plugin-manager/README.md)：管理操作の対象は現在Profileで、構成変更は当該Profileのセッション全体へ影響する。HMR有無で適用時点が変わる。Host側Pluginのin-process実行はWorkspace隔離と同義ではない。
- [DSH lifecycle](https://deepseek-harness.github.io/deepseek-harness/en/develop/framework/)：required service消失で依存PluginをUnloadし、復帰後に再ロード可能。これは「DB CommitをUndo」「未完了Effectを安全に再送」「複数Targetで無停止」には当たらない。
- TEIで保持する判断：[ADR-0032](../adr/0032-composition-kernel-replacement.md)の契約中心合成と[ADR-0043](../adr/0043-target-native-artifacts-without-tei-host.md)のTarget-native実現。Cordis ABI、Loader、Fiber/HMRの逐語移植、TEI専用共通Hostの導入は採用しない。

### 21.2 何の変化かを分ける：五つの根拠グラフ（新しい正本ではない）

同じPlugin集合を材料にしても、次のグラフは異なる対象を表す。既存Source/Lock/Build/Receiptから読み取る**診断用Projection**であり、五つの独立したRegistryや固定Core列挙は設けない。

| 観測面 | 頂点・辺の最小意味 | 正本・担当 | その面だけでは分からないこと |
| --- | --- | --- | --- |
| S: Source／Meaning | Source occurrence、Role、Accepted Meaning、依存・Revision | #2619/#2620/#267 | 実Provider選択・権限 |
| C: Composition | 要求Service Contract、Provider候補・exact wiring、Scope、Profile Overlay | #1016/#951 | 実Buildへの物理混入、実行許可 |
| B: Build／Artifact | 実際に消費したSource/Generator/Toolchain/Flags、Artifact member、Digest | #943/#1089/#1097/#1167/#2600 | 現行Hostが何をLoad/Invokeしたか |
| R: Runtime／Authority | 実際のReader/Instance、Invocation Context、現在Grant/Lease、Resource | #1118/#1174/#1370 | 業務意味の受理、Commit結果の確定 |
| E: Effect／Consumer | Operation identity、試行、Receipt、独立Read、開示されたConsumer結果 | #1129/#1174/#2516 | 任意の外部副作用の完全な発見 |

**横断不変条件**：Sの意味が同一でもC/B/R/Eの保証を流用しない。Cが変わってもSを自動受理し直さない。BのDigest一致はIssuer/現在Grant/実起動物同一性の保証ではない。Eの過去ReceiptとRの現在資格を分ける。「見つかった」「検証対象に入った」「結果が返った」を「現在実行可能」「業務効果が確定した」にしない。

### 21.3 Plugin更新の影響判定は三種類の集合を混ぜない

「未使用」を単にProfileの非選択と定義しない。

- **Resolved Selection**：#951が正確に選択したContract/Provider集合。登録順・最高Versionだけ・暗黙fallbackによる選択は拒否する。
- **Actual Consumption Closure**：Compiler、Generator、Go import/init/embed、直接・推移的依存で**実際に消費された**入力の集合。Plugin Lockで未選択でもここに含まれれば無関係とは言えない。
- **Qualified Evidence Scope**：どのSource/Contract/Provider/Artifact/Target/Grant/Consumerと、どの時点のClaimを検証したか。過去にPASSしたClaimを新しい対象へ自動転記しない。

対象変更の影響を unchanged / requires-requalification / conflict / unknown の説明用四分類として表し、未知の依存は「変化なし」でなくunknown/fail-closedへ戻す。これは内部の固定enum採用ではない。

| 変更 | S（意味） | C（選択） | B（実Build） | R/E（現行実行と効果） |
| --- | --- | --- | --- | --- |
| 未選択・実非消費のProvider追加 | 不変候補 | 選択Lock不変候補 | 同条件のArtifact不変候補 | 無関係なGrant/Effectを失効させない候補 |
| 未選択でもimport/init/embedが実消費 | 不変候補 | 選択だけなら不変 | **再評価必須** | 実際のReader/副作用の検証を要求 |
| 選択ProviderのImplementation/Contract変更 | 自動受理なし | exact再解決 | Closure/Artifact再評価 | 旧資格で新実装を実行しない |
| Source内のRole/Operator意味変更 | Owner再受理が必要になり得る | 依存箇所のみ再評価 | 関係Buildのみ再評価 | 旧Effectは歴史的事実として保持 |
| Publisher失効・Index停止 | Accepted意味を自動変更しない | 新規選択に信頼・鮮度問題 | Digest一致だけでは足りない | 現行利用許可はHost/ConsumerのPolicyに従う |
| Host Grant失効／DB Commit応答喪失 | 意味とは独立 | 宣言上の選択は不変 | Artifact自体は変わらない可能性 | 新規Effect拒否／Outcome unknown、operation IDで照会し自動再送を止める |

### 21.4 配置・更新Protocolと、Scopeをまたぐ変更

設計時の候補→予想Diff→正確なContract解決→Build必要集合→Artifact検証→Target実Reader→Host現行Grant→Effect Observationの順に、各Ownerの結果を紐付ける。ただし純粋関数の静的Buildでは動的Load/Watcherを要件化せず、通常のGo単一Binaryは再ビルドと再起動が必要なことを示す。

動的・長寿命Targetに限り、事前検査、旧世代を維持した新Provider準備、readiness、Atomic cutover、in-flight drain、登録解放を評価する。**新ProviderのReady ≠旧状態移行の成功≠可用性SLO達成**。部分的外部Effect、ACK loss、失効したGrantに対しては自動再起動や自動再送を許可しない条件を、#1223/#1174/#1129へ渡す。

Profileの層差分では、同じidでも変更対象のexpected revision/fingerprint、変更のOperation（add/replace/disable/remove）、適用Scope、影響するContract/Provider、Build closure、現在のConsumerを表示する。旧revisionを前提とする変更は黙ってwhole replacementしない。Tenant/Project/Sessionの作用域を分け、あるProfileの有効化を別Host/全Tenantへ自動伝播させない（#1016/#2044）。

### 21.5 第三者PluginとCommunityに必要な検証境界

**Discoverability → Publisher/Namespace Authority → Artifact provenance → Conformance → Consumer/Host admission** は別々の判定。第三者Indexが付けた verified / safe、Star、ダウンロード数、署名の存在だけから安全性を断言しない。#2589の分散Registryは**発見索引**であり第二Resolver/Authorityではない。公開名・Contract定義権・実ArtifactのPublisher・検証Issuerを区別する。

Community向け初期配布物の候補は (1) Contractと正負Fixture（Provider無しでも使用可能）、(2) 高凝集Bundle＋対象別Build Recipe、(3) 独立Conformance Procedure、(4) Consumer Integration例、(5) 安全な公開・失効・保守引継ぎ手順。すべてを「公式Plugin」として一律認証するのではなく、公開主体・保守主体・実行環境・検証範囲を示す。Plugin数を成熟度とせず、Conformant/maintained/portable/composable/reproducibleを観測できる範囲で別指標とする。

### 21.6 横断の最小反証・受入票（DSH-11〜18：独立Expected、現状NOT RUN）

| ID | Fixture / 変化 | 独立に期待する観測 | 既存Owner |
| --- | --- | --- | --- |
| DSH-11 | Profile Aの設定行を上位Patchで一部だけ指定 | 旧フィールド欠落と同ID衝突をDiffで示し、expected revision不一致なら拒否 | #1016 |
| DSH-12 | Bundleに未選択Provider追加、Go import/initを有無で対照 | Selectionだけで不変とせず、actual Build inputs・Linked bytesを比較 | #1251/#1167/#2600 |
| DSH-13 | 同Shape同Version、異なるNULL/decimal/failure/Effectの二Provider | Structural適合でSemantic同等を宣言しない | #951/#2620 |
| DSH-14 | Runtimeが依存Providerを失い復帰するが先のCommitは成功 | Plugin再ロードと歴史的DB Effectを分離し、自動再送なし | #1223/#1129/#1174 |
| DSH-15 | Index上Verified、Publisher権限失効・実Artifact置換 | Index主張だけで採用せずDigest/Authority/現在Trustを個別照合 | #2589/#1104/#1370 |
| DSH-16 | Plugin無効化/削除、未対応Sourceに未知の属性 | raw bytes維持、編集は可能、意味受理/実行許可を勝手に維持しない | #2044/#2553 |
| DSH-17 | TEI外のJava/.NET/Process Consumerへ同契約を渡す | TEI Studio/常駐Host非依存、同じ独立Oracleを実際に実行 | #2390/#1145 |
| DSH-18 | DSH SDK minimal/headlessからTEI CLIをread-onlyで呼ぶ | 実Tool/送出/Process結果とTEI Meaning/Effectの証拠を別列にする | #2509/#2515 |

**受入票の最小列**：Case、対象Source+Revision、Contract+Provider exact、Profile+Scope、実Build input/Artifact、現在Reader/Grant/Lease、Scenario/Expected、実観測と独立Oracle、未確認、Owner/再資格化条件。欠損はunknown／NOT RUNとし、0件確認済みやPASSに置換しない。詳細なSchemaや独立試験Frameworkの新設は先取りしない。

### 21.7 既存計画への返却と撤回条件

- #2899 はDSHのSource/Revision対応、比較表、反例の作成・調整のみ。新しい実ResolverやHostでの実装を所有しない。
- #951/#1016/#1251/#1223/#2589/#2620/#2044/#2390/#2509が各検証を所有。#1119のCritical PathのGateはそのまま。#2017のA0〜A4には作業票/検証Evidenceを渡すが、人間のOwner意味・権限・配備判断は置換しない。
- 実Consumerとactual source/Artifact/Targetを観測して、既存Contractと診断Projectionだけで要件を満たせる場合、新しいRegistry、CoreのPlugin型、Composition正本、Agent Runtimeは**追加しない**。
- **現在の状態**：本節は複数チャットの方針・既存ADR/IssueおよびDSH公式公開資料を照合した設計候補。DSH実コード/TEI実Host/Go Link/DB/外部Consumerの新たな実行試験、Ownerによる正式採用はここでは**NOT RUN／未採用**。

## 22. DSHと組織・Tenant・Communityの交差：配備ScopeとTrustの再資格化（2026-11-06、設計候補）

**対象・参照**：DSH横断研究[#2899](https://github.com/hat47x/tei-platform/issues/2899)、本書§21、[ADR-0032](../adr/0032-composition-kernel-replacement.md)／[ADR-0043](../adr/0043-target-native-artifacts-without-tei-host.md)、[#1016](https://github.com/hat47x/tei-platform/issues/1016) Application Composition、[#2386](https://github.com/hat47x/tei-platform/issues/2386) 組織関係、[#2589](https://github.com/hat47x/tei-platform/issues/2589) Plugin公開、[#2390](https://github.com/hat47x/tei-platform/issues/2390) 外部Consumer。拡張方式を決める三段階の正本は[Draft #2885](https://github.com/hat47x/tei-platform/pull/2885)と既存Runtime/Target設計であり、本節は**方式選定後の適用範囲・実証資格**のみ扱う。閉じた重複Draft #2888の判断を第二正本として復活させない。

### 22.1 DSHから確認した事実と類推の限界

- DSH [Architecture](https://github.com/deepseek-ai/deepseek-harness/blob/master/docs/architecture.md)では、PluginがService/Typed Event/Reversible Registrationを提供し、ProfileにBundle・Patchを積む。Session durable event、Agent live event、Capability extension eventが異なる。Scoped tool registryは提示するTool Schemaを限定し、guarded execution pipelineが実際の実行を扱う。
- これは**何が構成され／モデルへ提示され／呼ばれ／記録されるか**が異なることを示す。ただしSession event記録はTEIのDB Effect独立確認ではなく、Tool schema公開が実Host Grantを発行するわけでもない。
- Profileの定義域と、Tenant・Project・Workspace・Application・Process・Session・Operationの作用域は同一ではない。特にプロジェクトに選んだProviderから、当該Tenantの実データ閲覧権を導出してはならない。
- [#2386](https://github.com/hat47x/tei-platform/issues/2386)で扱うLegal Entity／OrgUnit／Membership／Business Role／Delegationと、配備上のTenant／Workspace／Host principal／resource grantを区別する。組織意味の変化と現在のHost認可の更新は別資格。

### 22.2 Scopeは単一の親子階層でなく、異なる所有者を持つ直交軸

| Scopeの観点 | 主な意味・担当 | 禁止する暗黙の伝播 |
| --- | --- | --- |
| Distribution / Publisher | 名前・Contract定義権、Package/Release、署名・検証（#2589/#951） | Publisherの信頼 → Runtime権限 |
| Authoring / Model | Project Source、Role、Accepted Meaning、改訂（#2619/#267） | Owner意味受理 → 実Tenantデータ権 |
| Composition / Application | 選択されたPlugin、Profile、環境要求と制約（#1016） | 一つのProfile選択 → 全Tenant/全Processへの自動適用 |
| Domain / Organization | Legal Entity、OrgUnit、Membership、Case/責任関係、validAt（#2386） | 組織所属 → Tenant認可、組織再編 → 既存実Grant即時更新 |
| Runtime / Invocation | Verified issuer+subject、Tenant/resource、Current Grant/Lease、実Reader（#1118/#1174） | Clientが述べたTenant → 信頼された実効Tenant |
| Evidence / Disclosure | Context/Consumer許可、実取得範囲、部分性、観測鮮度（#2516/#1178） | read-only diagnostic → 秘密情報の全開示 |

一つのProfileが共通標準Pluginを選ぶ場合も、実行単位・Tenant・Principal・ResourceのBindingはHost側の現在Authorityへ戻す。すべての組織モデルにテナント境界を埋め込まない。「Local Studio no-auth」は「無制限Resource権限」でも「SaaS側no-auth fallback許可」でもない。

**変化の因果**：PluginのImplementation改訂はBuild/Conformance資格を再評価するが、OrgのAccepted Meaningを自動変更しない。Org関係の改訂は関係するBusiness Rule/Query/ContextやAction候補を再評価するが、別TenantのPluginを自動有効化しない。Host Grantの失効は現行の作用を拒否しても、旧Receipt/歴史的業務関係を削除しない。過剰失効と過少失効の両方を負例にする。

### 22.3 「組み立てた・見せた・実行した」を別に証明する

| 段階 | 表示・判断できること | 次段に要求する独立確認 |
| --- | --- | --- |
| Discoverable | Plugin/Contractが検索・Indexに見つかる | Publisher/Contract AuthorityとRelease |
| Selected/Configured | ProfileでProviderを固定・構成した | #951 ResolverとTargetの実制約 |
| Built/Included | 実消費入力とArtifactが対象に含まれる | exact bytes/Load・Conformance |
| Advertised/Exposed | UI・Agent・SDKが操作候補を表示した | Consumerの実Targetと現在Permission |
| Invoked | 実ProviderへRequestを到達させた | Effect有無・結果/失敗/不明 |
| Effect Confirmed | 独立Read等により対象効果を確定した | 監査・Consumer開示の適合 |

同じ見た目の「有効」状態へ統合しない。AI/外部HarnessのGoal/MCP/Tool/Skill公開はHost Authorityでない（#2509/#2515）。Pluginが構成上存在しても、物理Buildに未収録、Runtimeで未ロード、現在Grant失効、送出経路未観測なら別のunknown/拒否を表示する。Toolの表示が読取り専用でも、Hostに残るOS/Network権限やログ送出が安全とは断定しない。

### 22.4 Community公開：Discoveryの先にある採用・退出

第三者Pluginが使いやすいことと、安全かつ継続可能に採用できることを別に扱う。DSH [CONTRIBUTING](https://github.com/deepseek-ai/deepseek-harness/blob/master/CONTRIBUTING.md)の独立Plugin参加と、[SAFETY](https://github.com/deepseek-ai/deepseek-harness/blob/master/SAFETY.md)の開発者プレビュー警告をともに参照する。

TEIでは既存#2589の分散公開・第三者Conformanceに、**採用後の変化と退出**という圧力を追加する。対象は初回Publisher権限／署名／Digestだけでなく、廃止・yank（既存利用の扱いを別にする）、所有権移転、侵害後の発行者失効、offline時の最新Trust不明、脆弱性告知、互換性を保つ代替Providerへの移行。Releaseの既知Digestが一致しても、現在Trusted/許可済みとは断言しない。

公開の最小導線は **Contract-only + 独立Fixture → Provider Implementation + Build recipe → 第三者の範囲付きConformance → Target別Consumer利用例** の四つを区別する。Providerがない段階でも他者がContractを実装可能であり、TEI専用Host/Studioを必須としない（#1975/#2390）。同一Packageに契約と実装を同梱してよく、分割パッケージを義務化しない（#1251）。

Registryは**発見の索引**、Issuerは証拠の発行主体、Host/Ownerは採用・実行判断の主体。第三者Indexに「認証済み」と書いても本人確認／意味／安全性／Runtime資格を自動付与しない。必要なメタデータをもう一つの手書きSSOTとして保存しない。

### 22.5 Scope／Communityの反証を既存Ownerへ追加する（DSH-19〜26、NOT RUN）

| ケース | 入力変化／故障注入 | 独立Expected | 担当 |
| --- | --- | --- | --- |
| DSH-19 | 同一BundleをTenant A/Bで採用しAのみ実Grant | AとBのPlugin選択が同じでもBの実Effect拒否。Orgの兼務からB権限を生成しない | #1016/#2386/#1174 |
| DSH-20 | Profile内でPluginを有効化、別Project/Processが旧版維持 | 必要なScopeにのみ構成変更。旧Reader/Artifactへ暗黙伝播しない | #1016/#1223/#2044 |
| DSH-21 | Agent Tool schemaに「read only」を表示、実OS/Network送出が過剰 | 表示と実権限・送出経路を分け、未観測なら安全と宣言しない | #2509/#2515/#1178 |
| DSH-22 | Org異動で承認候補が変わり、Host Grantは旧版 | Candidate/Role/現Grantの資格を別票。実副作用前に現在Grantと対象を再照合 | #2386/#1174 |
| DSH-23 | ReleaseのPublisher失効、同Digestがキャッシュ中 | 過去のbyte一致を現在Trustへ昇格しない。offlineではunknown条件を記録 | #2589/#1104 |
| DSH-24 | 第三者Index停止、旧Pluginの保守終了・代替候補複数 | Source/既存Contractと未対応データは保持、無根拠に別Providerを自動選択しない | #2589/#951/#2044 |
| DSH-25 | Contract-onlyをTEI非依存Consumerが入手し別言語実装 | PublisherのContract定義と実装Conformanceを分け、外部Hostで独立Expectedを満たす | #1975/#2390 |
| DSH-26 | 永続Session Eventと一時イベント/ログの矛盾、実DB Commit unknown | 観測の寿命・Scope・Effectを別にし、イベント完了だけでDB確定としない | #1223/#1129/#1174 |

各Caseは Source/Revision、Profile Scope、Contract+Provider+Artifact、具体的Host/Principal/Tenant/Resource、Expected、実Observation、未観測/失効の列を持つ**既存のEvidence台帳への要求**。Schema名、新Core enum、新試験基盤、必須の中央Registryを新設しない。実験は公開・合成Fixtureから始め、秘密や未公開Sourceを外部Harnessへ渡さない。

### 22.6 Agent並列作業と計画への受渡し

#2899（外部研究と対応表）→ #2017の既存A0〜A4作業票（対象、変更権限、独立Expected、未実施）→ 各Owner #1016/#951/#2386/#2589/#2390/#2509/#1174で実試験 → #1119の既存横断Gateへ結果参照、という方向とする。DSH研究を進めるためにTEI側にDSH専用Agent Loop/Orchestratorを作らない。

**撤回条件**：Scope/Trustの違いが既存Contract＋Profile＋Runtime Invocation＋Evidenceの読み取りProjectionで表現できれば、新しいScope Hierarchy/Policy Engine/Registry/Grant storeを追加しない。新規Plugin/Target用Core機能が不可欠と主張する場合は、二つ以上の異質Consumerと既存仕組みでの反証を提出する。

**状態**：資料・既存Issue・別PRの設計照合に基づく追加受入候補。DSHの全ソース監査、実Multi-Tenant Host/IdP、Plugin配布失効・Go Link/実DB・Agent OS送出の新しい実測、Ownerの正式意味受理は**NOT RUN／未採用**。本節を含むPRのDraftとmain未統合を維持する。

## 23. 六Scopeの圧力試験と方式棄却：権限の時点・効果の分割を固定する（2026-11-06、設計候補・NOT RUN）

**差分の所有範囲**：§22の六Scope分類とDSH-19〜26は保持。本節は既存#2386（組織意味）、#1016（構成）、#951（Contract/Provider）、#1118（Context slot）、#1174（current Grant/Commit）、#1129（部分Effect）、#2516（開示）の**同一Operationに対する必要確認点**をつなぐ圧力試験である。方式選定の原則とD0/D1/D2は[Draft #2885](https://github.com/hat47x/tei-platform/pull/2885) §14〜21に委譲し、本節で再採用・変更しない。現mainへ採用済みのADR-0043（TEI共通Host非必須）に反しない。

### 23.1 権限に関する非推論契約

| Scope | 確認可能な事実 | そこから推論してはいけないこと |
| --- | --- | --- |
| Distribution | Plugin/ReleaseのPublisherとContract由来、Artifact digest | Publisherが信頼できるから、実行主体にBusiness/Resourceアクセス権がある |
| Authoring | Source revision/MeaningとOwner受理、対象Function/Rule | 設計の受理から、個々のHostの現在のGrantが作られる |
| Composition | Application/ProfileがProviderを選択、Tenant別Binding候補を宣言 | 同じPlugin/同じBinary/同じProfileから、Tenantの資格やデータ共有が自動成立する |
| Organization | Person/Employment/OrgUnit/Membership/Position/Delegation（validAt/knownAt） | 兼務・組織Role・同名/メール/Directory Group一致が別issuerのprincipalやTenant Grantを決定する |
| Runtime | Hostが確認したissuer+subject、Target/Resource/Tenant、current Grant epoch、Effect fence | PreflightのAllowや旧LeaseだけでCommit時にも有効だと断定する。また撤回により既Commitが消えると断定する |
| Disclosure | 特定ConsumerへのReader結果、母集団・complete/partial/denied、開示許可 | SQL result=0やread-only、UI非表示から、別Tenantの不存在・秘密非送出・業務Effectなしを断定する |

**共有の条件**：共有可否はAssetの共有（Package/Bundle/Contract）、Source/Modelの共同Authoring、RuntimeのResource/Recordアクセス、Consumerへの開示についてそれぞれ明示的なBinding/Policy/Authorityを要する。共通Libraryの同梱は共有Storage・共有Principal・暗黙のCredential再利用を意味しない。Tenant内の完全な0件から別Tenantの0件を推論しない。異なるIssuerの同一人物対応は認証された根拠なしに結合しない。元のSourceと未知構造はPlugin解除後も保存する（#2044）。

### 23.2 同じ受注番号を持つ二法人・二Tenantの固定縦断（新Fixture正本ではない）

既存[#2386 S2/S5/S6](https://github.com/hat47x/tei-platform/issues/2386)、[#3016](https://github.com/hat47x/tei-platform/pull/3016)（同じOrderID=9のTenant A customer12とB customer29、実PG opt-in test候補）、[#1174](https://github.com/hat47x/tei-platform/issues/1174)のEffect受入を**同一独立Expectedに戻して**使う。以下は追加の比較手順でありテストは未実行。

| 観測ケース | 固定する条件・故障注入 | 独立Expected、拒否・不明の条件 |
| --- | --- | --- |
| SCOPE-1：二法人兼務 | Person Pは法人A本務、法人B兼務。A/Bは別のissuer+subjectかもしれない。Bで業務上承認候補となるが、B HostのGrantは無い | Bの業務候補は意味として表示可能。ただしBでの実Writeは拒否／未資格。Personの同定・共有は別出典とOwner判断。AのGrantはBの代替にならない |
| SCOPE-2：同Plugin二Tenant | 同じContract/Implementation/Artifact、別BindingでA/BのOrderID=9を読む。Aの実GrantはA-only、Bは無し／別のB限定Grant | Aの許可されたReaderは(A,9,12)のみ、Bが無許可なら**denied（0件とは別）**。B許可時は(B,9,29)のみ。Cache/Pool/Request Contextの流用による混在を拒否。実Consumerの公開列は別Disclosure |
| SCOPE-3：Grant撤回の競合 | T0 Grant epoch eが有効→T1 Preview/Dispatch→T2 Hostがeを撤回→T3 ProviderのCommit境界。対照はT3 Commit済み→T4撤回 | **撤回の効力点と実Provider fenceの契約を先に特定**。T2が強制境界前ならその境界の新規Writeを拒否し、Effect 0を別Readerで確認できる実現だけqualified。Host/DB間の原子的失効保証が無ければ競合区間をunknown/unqualifiedとし、「必ず0件」を自称しない。T3確定後の撤回は旧Commitを消さない |
| SCOPE-4：部分効果とACK喪失 | 同一Business OperationでDB更新AをCommit、外部通知Bが失敗／ACK unknown。旧ProviderをUnload、B側Grantはその間に失効 | Aを実Receipt+独立Readで既確定として保持し、Bをfailed/unknownと明示。Run/Plugin cleanupを全体rollbackやBusiness Successにしない。再送は同一Operation/Effect段階IdentityとProviderの実重複防止根拠が無ければ停止。補償は業務Ownerが許容してから明示別Effectとして扱う |

**時間の意味を混ぜない**：OrgのvalidAt/knownAt、Source/Accepted Revision、Profile/Artifact Revision、Grant/Lease epoch、Operation/Effect attempt時点、独立Readの観測時点を同じTimestampで代用しない。撤回要求をいつ受領したかと、実ProviderのどのCommitが拒否可能だったかは別。OS/IdPとDBに跨る完全な原子性が無い場合は**保証不能条件を先に申告**し、その方式で厳密なrevocation-at-commitが業務要件なら棄却する。

**Disclosure**：Aで読める情報をBへ書込んだりAgentに送出する場合、AでのRead権限はBへのWrite/Disclosure権限でない。元Sourceの機密と開示先を別に評価。独立Readを行う観測器も正当な権限が必要であり、「検証だから」アクセス制約を迂回しない。

### 23.3 方式選定の棄却基準（選定時のStop条件、未採用）

まずD0（標準Data/Rule/直接Library/設定）で必要な振舞いを表現できるかを確認する。**論理Plugin Contractが必要か**と**別Processが必要か**は独立に問う。次の三方式は万能な順位ではない。

| 方式 | 採用を検討できる前提 | 棄却／再設計へ戻す条件 |
| --- | --- | --- |
| D1 静的合成（Go単一Binary・Target native） | 独立更新をBuild/Deployで扱える、実依存・未選択の物理除外を測れる、Host側に必要な権限Fenceがある | Hard要件が無停止の個別Hot swapで再起動不可／不信Pluginコードの強制分離が必要なのに同一プロセスで実行／実選択とactual import/init/embed/Linkの一致を証明できず余計なEffectが残る／必須Grant fenceが組込めない。**その選定・実装の不合格であって静的方式一般の禁止ではない** |
| 動的ロード／同一ProcessのGeneration | Target固有Loader・ABIと宣言された権限、prepare→ready→cutover→drain/dispose、実In-flight/Effect境界が観測できる | Go静的TargetにLoaderを強制／untrusted native pluginをin-processで隔離できると主張／世代切替で旧Resource/Secret/Tokenの残存・権限再検査漏れ・未完了Effectがある／安全なdrain/rollbackを要求されるが保証できない。Registration disposerはDB取消ではない |
| D2 外部Process／独立Host | 契約とTransport/Caller/Target Bindingが明示され、OS/Network/Secretの強制境界を検証でき、Crash/ACK-loss/再送・Receipt/独立ReadがConsumer要求に合う | プロセス分離だけでTrust/Filesystem/Network隔離成立を自称／同一OS user・共有Credentialの越境が防げない／RPCでTenant/Subject/Operation/Grantの受渡しが偽装可能／業務上必須のCommit fence/重複防止/独立観測ができない／実Latency・Memory・配備の上限に違反。**Process＝安全は不採用** |

**三方式共通の棄却**：Owner意味未受理を実行可能とする、要求されるNULL/precision/Unknown/partial/orderingを契約差に押し込む、現在GrantをProfileから推定、明示されないCredential/Networkを利用、実Target・独立Expected・Actual Source/Artifactの対応が取れない、または完全な原子性/Exactly-onceを実証なしに保証すること。必要のない「全Plugin動的化」も、適用・保守・安全の費用を増やすだけなら棄却候補。

### 23.4 実証を取り込む最小の受渡し票と責務

検証の独立列：`(Source/Meaning Revision, Application/Profile+Tenant binding, Contract+Provider+Artifact, Org validAt+knownAt/actor mapping, Host issuer+subject/current Grant epoch+Resource, Operation/Attempt/Effect-stage ID, required/evaluated guards+Provider fence point, effect result/receipt, independently scoped read/time/completeness, Consumer disclosure, Actual/Expected/NOT RUN)`。ここでの列は概念であり、新Manifest・全Host共通Schemaを採用しない。

- **Meaning/Org**：#2386/#267。**Wiring/Scope**：#1016/#951/#1118。**Build/Way selection**：Draft #2885/#1251/#1121/#2390。**Grant/Effect/Recovery**：#1174/#1129/#2564。**Disclosure**：#2516/#1178。**Test Oracle**：#1180/#2514。**Product Gate**：#1119既存G1/G2/G4。
- 既存[#3016](https://github.com/hat47x/tei-platform/pull/3016)を「同Key異Tenantの実PG read」局所回帰として使い、SCOPE-3のGrant撤回・SCOPE-4のDB+外部通知部分Effectや正式IdPまで成功したとは言わない。#2885の[D0/D1/D2](https://github.com/hat47x/tei-platform/pull/2885)は既存23 Vectorと独立実行証拠へ戻し、Scope横断負例を純粋D0の必須テストにしない。
- **状態**：本文は既存Source/PR/Issueの照合に基づく**方式の停止条件・独立Expected案**。このSCOPE-1〜4を一連の実Host/IdP/実DBで実行していない。Static-vs-Dynamic-vs-Processの実コスト・権限強制・失敗時のProduct E2E、Owner意味受理・正式方式採用は**NOT RUN／未採用**。本節とそのDraftを本番保証としない。

## 24. Scope相対のEvidence接合：完全被覆・現在Grant・実消費を横断する最小条件（2026-11-06、設計／Product実証はNOT RUN）

**差分の位置**：§21のSource/Composition/Build/Runtime/Effect、§22の六Scope、§23のSCOPE-1〜4を再掲しない。必要なのは**独立に成立したEvidenceを、そのClaimへ接合できるかの条件**である。mainへ統合された[#2926](https://github.com/hat47x/tei-platform/pull/2926)のCompletion評価、[Draft #3026](https://github.com/hat47x/tei-platform/pull/3026) §9.5のM/G直交と結合対象不一致、[#3016](https://github.com/hat47x/tei-platform/pull/3016)の同Key二Tenant、[#2885](https://github.com/hat47x/tei-platform/pull/2885)のD0/D1/D2差を既存Ownerで使う。追加のCompletion Evaluator、Policy/Grant Engine、証拠台帳、Oracleを設けない。

### 24.1 Claim別の証拠適用条件（新しい共通Schemaではない）

| Claimの接合 | 必須の関係・対象・時点 | 禁止する推論 |
| --- | --- | --- |
| 選択Contractから実Provider | Contract/Provider Pin、Profile/Target、実Build member、Load/Invoke | Catalog/ResolvedSetやD0テストからD1実Link成功を認定 |
| 意味受理からAction | Accepted Rule/Business Action/Subject/Source Revision・受理範囲 | 見積承認を同形の受注確定のOwner受理へ転用 |
| Grantから今回Effect | 実issuer+subject、Tenant+Resource+Action、Grant epochとProvider強制点 | A本務/B兼務、別Tenant読取、Preview許可からBのCommit権限を生成 |
| Provider Readから集合Coverage | 実Query/Reader、Row Scope/Policy、対象母集団と時点、complete/partial、否定依存 | Tenant A内complete zeroをTenant B/全Tenantの不存在にする |
| Receiptから効果 | Operation/Attempt/Effect stage、Target/Expected Version、実Commit、独立ReaderのScopeと時点 | 別対象のCount>0/古いReceiptから今回の効果・全Step完了を推定 |
| Readから開示 | Consumer/Destination、許可されたBody/診断/件数、Readの由来と時点 | 読取許可を別TenantへのWriteや外部Harness送出許可へ流用 |

**重要な既存実装の上限**：mainの internal/semantic/completion.go は、呼出側が与えたTargetResult・CoverageComplete/Partial/Unestablished・EmptyJustifiedを判定する。実ProviderからOwner指定の正しい集合Lを漏れなく取得したこと、Tenant/時点/目的との一致まではEvaluatorが実証しない。したがって不適格なCoverageCompleteを注入して得たholdsをProduct資格としない。0件、hidden、denied、NULL、未取得、partialは別。開示先に秘密の母集団・件数・Grant情報を出さない。

### 24.2 既存独立Expectedへの一因子対照（JOIN-01〜06、すべて新規実行NOT RUN）

| Case | 差し替える一条件 | Expected / 非検出の上限 |
| --- | --- | --- |
| JOIN-01 | Meaning MはOrder O1確定、Grant GはO2閲覧、又は別Tenant O1 | MとGが各自有効でもO1のCommitは認可しない。拒否診断だけで実Provider Effect 0は証明しない。#3026の別試験を作らず参照 |
| JOIN-02 | A Tenantのcomplete zero又はBのhidden/partialをBのTargetSet空へ転用 | Bの不存在・Completion holdsを不認定。#3016の実Read比較だけで全母集団の完全性は証明しない |
| JOIN-03 | T1 M/G有効→T2 Grant revoke→T3 Provider Commit、T4旧Receipt参照 | 現Grantと実Fenceの資格を再判定。原子性不明ならunknown。旧Effectは歴史的に残るが現在の実行/開示資格ではない |
| JOIN-04 | 実Readがpartial/unobservedなのに呼出側がCoverageCompleteを供給 | Evaluatorがholdsを返しても入力Coverageの真正性を実Reader/母集団の独立Witnessで再照合。判定器の誤りと不適格入力を分離 |
| JOIN-05 | 実D0直接計算＋D2独立Processの限定CI合格をD1静的ProviderのEvidenceへ接続 | D0/D2限定PASSを残し、D1には実Resolved→Build/Link bytes→Consumer Invokeが必要。D1は未資格 |
| JOIN-06 | 同OperationでDB A確定後、外部通知B ACK-loss。BのGrantを撤回 | A Confirmed／B Unknownを保持。blind retry、全体成功、旧Commit取消、Receipt現在開示許可を自動推論しない |

各Caseは**同じAccepted Source/Independent Expected/対象Operation**から始め、違いを一因子だけに固定する。CoverageとGrantのClaimを結ぶ際は完全なフィールド一致ではなく、各証拠が対象Action/Scope/時点を**包含すること**を検査する。実ProviderやHostへの強制が未接続ならNOT RUN/unknownで停止。Mutant検出のみで「DB効果なし」「OS送出なし」を証明したとしない。

### 24.3 現物証拠と次の実証・Owner

- **コード統合済み**：[PR #2926](https://github.com/hat47x/tei-platform/pull/2926) Merged。semantic.EvaluateCompletionは与えられた被覆情報を評価する限定実装であり、実Reader Coverageの作成者ではない。
- **限定CI成功**：[Draft #3021](https://github.com/hat47x/tei-platform/pull/3021)の修正HEAD dfc133e2ce1877e3593f1e507cbc3c2b7a82bfc5 で、[Isolated](https://github.com/hat47x/tei-platform/actions/runs/42203769225)・[Repository Health](https://github.com/hat47x/tei-platform/actions/runs/42203769269)・[Dependency](https://github.com/hat47x/tei-platform/actions/runs/42203769210)がSuccessと実PRに記録。D0直接関数／D2既存Processの限定CIであってD1・Host/Ownerの資格ではない。
- **実PostgreSQLの未確認**：[Draft #3016](https://github.com/hat47x/tei-platform/pull/3016)はA/B同じOrderID=9、異なるCustomerIDの実PG opt-inテストコードを追加したが、新試験実PG/SKIPなしPASSはPR上未確認。正式IdP/Grant、Org Case、外部Disclosureは別。
- **実証の最短順**：#3016の実PG Scope Readをexact-headで実行（SKIPなし）→#2586/#2622の正しいFunction/Provider ReadとOwner母集団のCoverageを#2926の入力へ接続→#1174の実Host Grant/Effect/Receipt/独立Readerを**同一Operation・Target・時点**で照合。D1は#2885/#1251で別に試験。SourceのUNBOUND_DATASOURCE等を0件や「プロバイダー選択済み」とみなさない。
- **Owner**：Meaning #267/#1125、Composition #1016/#951、Scope/Data #2386/#2586/#2622、Effect #1174/#1129、Disclosure #2516、Scenario/Oracle #1180/#2514、方式 #2885/#1251。Product gate #1119を増設・変更しない。

**結果と撤回条件**：本節は既存結果から導いたEvidence接合の受入候補であり、JOIN-01〜06、Product実Host/PG/Consumer/Owner受理はNOT RUN。既存のEvidence参照でScope相対の必要Witnessを説明できる限り、新しいRegistry/Proof Store/Status Enum/Hostや共通Completion Engineを導入しない。旧Receiptや一つのCI PASSを別Claimへ無断継承しない。

### 24.4 Source移行のShadowingをPlugin資格と誤認しない（2026-11-06、横断追補）

**他レーンで確認した変更**：[Draft #3030](https://github.com/hat47x/tei-platform/pull/3030)はTEI Authoringの標準を `*.tei.md` へ移し、旧 `*.tei` を段階廃止する明示方針（移行計画M0〜M6、TM-01〜14）。[Draft #2885 §18.12](https://github.com/hat47x/tei-platform/pull/2885)には、現行 `internal/workspace/loader.go` が `plugin.tei` 等の固定名/`*.tei` glob と `filepath.Ext` に依存すること、およびPlugin/BuildでのRequired→Discovered→Loaded→Parsed→Build-consumedの検証要求が既にある。**Parser/Loader対応・旧Source移行は未実装・未検証**であり、本§では新しいMarkdown構文・Parserや重複TM試験を作らない。

**新しい横断上の不一致例（TM-03/12/14＋JOIN-05への追加対照、独立Expectedを再利用）**：Application Profileは新 `plugin.tei.md` のContract/Provider revision **B** を明示している。しかしWorkspaceには同じSubjectの旧 `plugin.tei` revision **A** も残り、実LoaderがAだけを読んでいる。別案として旧Aを削除すると現Loaderは新Bを読まない。いずれもディレクトリにファイルがある、Catalogが非空、旧AのBuild/CIが成功、という表示からBの利用資格を導いてはならない。

| 独立した観測 | Expectedと棄却・保留条件 |
| --- | --- |
| **Authoring intent と実Source入力** | 要求BのSource Location/Revisionと、Workspace実Loaderの読み取ったpath/raw byte/span/Root/Parser区分を突合。Bが未読なら `unsupported/unavailable`（文脈別）で停止。AのみをBの代わりに成功として処理しない |
| **旧Aと新Bの両方が同じSubject** | 同内容でも同一編集正本と推定せず、TM-03に従い競合を明示。新規UI/AIの後勝ち・ローダ順・明示されない旧A fallbackを認めない |
| **解析からContract/Provider選択** | Bの明示TEI領域を実Parser/Roleで解釈し、Meaning/Contract/Implementation Pinが既存#951/#1016で適合して初めて選択。通常 `.md`・frontmatter tag・Git上の存在はOwnerの意味受理でもPluginの実選択でもない |
| **Build・Runtime・Effect** | Bが実Build-consumed/Target-linked/actual Loaded/Invokedである証拠を別に要求。AのCI/Artifact/Receiptは歴史的事実として保存するが、Bの現在Grant、Tenant間共有、今回のEffect・開示許可に昇格しない |
| **正しいBの読み込みが確認できた場合** | 原byteやSource Locationの変更だけで全Contract/Owner Meaningを無条件に再受理させない。変更影響の範囲を#2600に委譲し、影響があり未確認のClaimだけ再資格化する |

**必須の表示と安全な回復**：診断に「意図B／実読取Aまたは未読／選択済Provider／実Binary/Reader／現在Host Grant／開示先」のうち証拠のある範囲を示す。権限のないConsumerにはA/B秘密の存在やSource pathを漏らさない（#2516）。Source保存・Plugin無効化は元ファイルを破壊しない（#2044）。Aを黙って採用したり外部AIへBの未読Sourceを送ったりしない。旧Artifact/Receipt/Decisionの履歴は保持し、Source移行と実Effect取消を混同しない。

**責務と進捗**：Source Reader/Index・移行の正本は#3030/#2621/#2823/#988、Contract/Compositionは#951/#1016、実Buildは#1251/#2600、Scope/Effectは#1174、Disclosureは#2516、Expectedは#2514/#1180へ返す。唯一の横断追加は「**意図したSource Bと実消費Aの不一致を、Plugin選択や現在許可へ繋がる前に止める**」こと。D0/D2の既存Quote 23ベクトル・固定Gateの旧Sourceを改変しない。移行実E2E、Loader対応、Bを選択した実Go/外部Host・Tenant実権限、Owner受理はすべて**NOT RUN／未採用**。

### 24.5 実装済みの入口検査を、権限・効果の横断資格に昇格させない（2026-11-07、実Source照合・横断試験案）

**更新理由**：§24.4の「意図Source B／実Loaded Aの取り違え」は、別レーンで現物の負例・対策が進んだ。またCommand/Grant/Applied Operationにも、**入口での局所判定と実際の効果確定を混同し得る継ぎ目**が確認された。以下は既存#3031/#3034/#3035/#3033のコード・PR記録から得られるClaimの上限だけを整理する。別のAuthorization/Grant Store、Permission Interpreter、Plugin Host、Effect Coordinatorを作らない。

| 関係する実Source/試験 | 既に成立・具体化したこと（検収版を限定） | 次のClaimへ自動で渡せないこと |
| --- | --- | --- |
| [Draft #3031](https://github.com/hat47x/tei-platform/pull/3031) Workspace TEI Markdown admission | Source root内の `*.tei.md` を `TEI_MD_PARSER_UNBOUND` で拒否し、通常 `.md`・旧 `.tei` の非干渉を検査。固定HEAD `63953ccb...` の[局所CI #42217379093](https://github.com/hat47x/tei-platform/actions/runs/42217379093)が成功。外部TAIの[旧CLI実行 #42239797449](https://github.com/hat47x/tai-pleroma/actions/runs/42239797449)では、同bytesの拡張子変更だけでSubject 1→0かつValidate OKとなる**旧版のsilent skip**を再現 | Draft Guardの存在＝main Loaderが改修済み、あるいはMarkdown Parser/Role/Providerが新形式を実消費、とは言えない。旧版の不安全結果とDraftの局所安全診断を**別HEAD**で説明する。実Parsed/Build/Runtimeは未実証 |
| [merged #3034](https://github.com/hat47x/tei-platform/pull/3034) `legacyauthorization.CommandAuthorizer` | `Subject` はHostが設定、`Action`/ `Resource` mappingは任意callback。callback未指定ならActionはCapability ID、ResourceはApplication ID。既存`security.authorize`へ委譲し、明示規則拒否を`ErrNoGrant`、評価失敗を別errorとして伝える。Patternの表示＝権限ではない | Commandの「Application単位で許可」を、実Executorが書くOrder O1/別Tenant O2の**レコード単位・時点付きGrant**へ勝手に拡張できない。mappingの意味・対象一致・Grant epoch・実Provider fenceは既存#1174の別資格 |
| [merged #3035](https://github.com/hat47x/tei-platform/pull/3035) `access.Ground` | Grant発行が、環境の宣言Capability・Access包含・宣言Scopeを実Run Pathへ解像した範囲に収まるかを検査。13件の局所Groundと8件のCLI試験、実Workspace 3経路の拒否／許可結果をPRに記録。無関係Capabilityへの発行・宣言外Pathを拒否 | `issuedBy` / `resolvedBy` は**自己記述文字列で署名でない**。実発行者・Resolver真正性、現行Host Permission、他Tenant Grant、Runtime Effectは未証明。宣言を渡さない発行は現実装で警告付きで許され、Ground成功／空Grantから操作許可を生成してはならない |
| [merged #3033](https://github.com/hat47x/tei-platform/pull/3033) `command.Result.Applied` / Invoker事後照合 | 報告済みApplied Operationの宣言外種別・宣言効果欠落を検査してOperation Recordに反映。ReferenceDataExecutorは実レコード操作を列挙 | 検査はExecutor.Execute後なので、Providerがすでに外部Commitしてから事後エラーでもEffectは残り得る。Appliedが空のExecutorでは**報告不在≠実Effect 0**。正しいCommand Outcome/Journal/宣言効果照合は同一Tx Commit/Receipt・独立Readを証明しない |

**横断の核心**：ここには(1) Source Admission、(2) Permission Declaration/Issuance Grounding、(3) CommandのSubject/Action/Resource認可、(4) Provider Effect/Receipt、という**用途もAuthorityも異なる検査**がある。発行されたFilesystem PermissionがGroundedであることからBusiness Command Grantを作らない。Commandが許可されたことからFilesystem/Networkの能力を与えない。Sourceが適格にLoadedされてからだけCapability/Operationを結び、未対応Sourceを「Pluginなし」や「対象0件」へ置換しない。

**既存Scenario/Oracleへ返す最小の一因子対照（新Case正本なし）**：

- **入力採用**：旧LoaderでA=`model.tei`はSubjectあり、新B=`model.tei.md`のみだとSubjectなし・`validate.ok=true`（旧実CIの限定観測）。Draft #3031では新Bが`TEI_MD_PARSER_UNBOUND`になり、誤った成功を防ぐが新BのParser成功ではない。今後の実Loader対応時には同ExpectedでBの実Reader/Parser/Source occurrenceまで検証。
- **Grantの根拠**：宣言が`migration.atlas`のみ／発行要求が`tei.frozen_world.lineage`、または宣言外RootならGround拒否。対照として同じ宣言内の正規Capability・Rootなら局所成功。ただし`issuedBy`等が偽称可能な条件や、宣言を省略した発行を**現在のHost認可成功**へ流用しない。
- **Command主体・対象**：同じ認証済みSubjectのまま、Application単位Resource許可を実Order O1（またはTenant Bの同Key O1）へ誤用した場合、実Resource/Action mappingの受理範囲を確認できるまでレコード単位許可はunqualified。明示Rule拒否／Rule評価失敗も同じdeniedへ潰さない。認可後の失効は#1174の既存SCOPE-3へ戻す。
- **事後効果照合**：実Provider AでCommit→Applied/Outcome検査でError、あるいはCommit後Applied=nil。上位のCommand失敗や空の報告をEffect 0とせず、Receipt＋独立ReaderでAの確定・不明を区別する。別通知BのUnknown/再送は既存SCOPE-4/JOIN-06へ戻す。

**検収・優先順**：#3031は`*.tei.md`未接続のfail-closed、#3030/#2667/#2621/#988は新Reader/実Source、#951/#1016/#1251は選択ProviderとActual Build、#3035と既存accessは発行の範囲、#3034/#1174はCommand/Host Current Grant、#3033/#1174はApplied/Effect/Receipt、#2516は開示、#1180/#2514は既存独立Expected。新しいPlugin Lifecycleや一律Guardを追加しない。D0純粋関数にはDB Commit義務を課さない。

**状態**：merged #3033/#3034/#3035の局所実装と、Draft #3031の固定HEAD局所CI、旧CLIのsilent skip実観測を**同一E2E成功としない**。この横断接合のProduct実Go/DB/IdP/OS/複数Tenant、承認済みMeaning・正式Owner採用はNOT RUN。#1119 Gateは不変。上記の禁止推論と不足Witnessが既存Referenceで説明可能なら新Core/ACL/Registry/Transaction Engineへの拡張を撤回する。

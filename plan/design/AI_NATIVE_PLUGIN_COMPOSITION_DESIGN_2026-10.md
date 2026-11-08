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
| DSH-17 | TEI外のJava/.NET/Process Consumerへ同契約を渡す | TEI Studio/常駐Host非依存、同じ独立検証基準に基づく試験を実際に実行 | #2390/#1145 |
| DSH-18 | DSH SDK minimal/headlessからTEI CLIをread-onlyで呼ぶ | 実Tool/送出/Process結果とTEI Meaning/Effectの証拠を別列にする | #2509/#2515 |

**受入票の最小列**：Case、対象Source+Revision、Contract+Provider exact、Profile+Scope、実Build input/Artifact、現在Reader/Grant/Lease、Scenario/Expected、実観測と独立検証基準、未確認、Owner/再資格化条件。欠損はunknown／NOT RUNとし、0件確認済みやPASSに置換しない。詳細なSchemaや独立試験Frameworkの新設は先取りしない。

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

- **Meaning/Org**：#2386/#267。**Wiring/Scope**：#1016/#951/#1118。**Build/Way selection**：Draft #2885/#1251/#1121/#2390。**Grant/Effect/Recovery**：#1174/#1129/#2564。**Disclosure**：#2516/#1178。**独立検証基準**：#1180/#2514。**Product Gate**：#1119既存G1/G2/G4。
- 既存[#3016](https://github.com/hat47x/tei-platform/pull/3016)を「同Key異Tenantの実PG read」局所回帰として使い、SCOPE-3のGrant撤回・SCOPE-4のDB+外部通知部分Effectや正式IdPまで成功したとは言わない。#2885の[D0/D1/D2](https://github.com/hat47x/tei-platform/pull/2885)は既存23 Vectorと独立実行証拠へ戻し、Scope横断負例を純粋D0の必須テストにしない。
- **状態**：本文は既存Source/PR/Issueの照合に基づく**方式の停止条件・独立Expected案**。このSCOPE-1〜4を一連の実Host/IdP/実DBで実行していない。Static-vs-Dynamic-vs-Processの実コスト・権限強制・失敗時のProduct E2E、Owner意味受理・正式方式採用は**NOT RUN／未採用**。本節とそのDraftを本番保証としない。

## 24. Scope相対のEvidence接合：完全被覆・現在Grant・実消費を横断する最小条件（2026-11-06、設計／Product実証はNOT RUN）

**差分の位置**：§21のSource/Composition/Build/Runtime/Effect、§22の六Scope、§23のSCOPE-1〜4を再掲しない。必要なのは**独立に成立したEvidenceを、そのClaimへ接合できるかの条件**である。mainへ統合された[#2926](https://github.com/hat47x/tei-platform/pull/2926)のCompletion評価、[Draft #3026](https://github.com/hat47x/tei-platform/pull/3026) §9.5のM/G直交と結合対象不一致、[#3016](https://github.com/hat47x/tei-platform/pull/3016)の同Key二Tenant、[#2885](https://github.com/hat47x/tei-platform/pull/2885)のD0/D1/D2差を既存Ownerで使う。追加のCompletion Evaluator、Policy/Grant Engine、証拠台帳、独立検証基準の第二正本を設けない。

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
- **Owner**：Meaning #267/#1125、Composition #1016/#951、Scope/Data #2386/#2586/#2622、Effect #1174/#1129、Disclosure #2516、Scenario/独立検証基準 #1180/#2514、方式 #2885/#1251。Product gate #1119を増設・変更しない。

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

**既存Scenario/独立検証基準へ返す最小の一因子対照（新Case正本なし）**：

- **入力採用**：旧LoaderでA=`model.tei`はSubjectあり、新B=`model.tei.md`のみだとSubjectなし・`validate.ok=true`（旧実CIの限定観測）。Draft #3031では新Bが`TEI_MD_PARSER_UNBOUND`になり、誤った成功を防ぐが新BのParser成功ではない。今後の実Loader対応時には同ExpectedでBの実Reader/Parser/Source occurrenceまで検証。
- **Grantの根拠**：宣言が`migration.atlas`のみ／発行要求が`tei.frozen_world.lineage`、または宣言外RootならGround拒否。対照として同じ宣言内の正規Capability・Rootなら局所成功。ただし`issuedBy`等が偽称可能な条件や、宣言を省略した発行を**現在のHost認可成功**へ流用しない。
- **Command主体・対象**：同じ認証済みSubjectのまま、Application単位Resource許可を実Order O1（またはTenant Bの同Key O1）へ誤用した場合、実Resource/Action mappingの受理範囲を確認できるまでレコード単位許可はunqualified。明示Rule拒否／Rule評価失敗も同じdeniedへ潰さない。認可後の失効は#1174の既存SCOPE-3へ戻す。
- **事後効果照合**：実Provider AでCommit→Applied/Outcome検査でError、あるいはCommit後Applied=nil。上位のCommand失敗や空の報告をEffect 0とせず、Receipt＋独立ReaderでAの確定・不明を区別する。別通知BのUnknown/再送は既存SCOPE-4/JOIN-06へ戻す。

**検収・優先順**：#3031は`*.tei.md`未接続のfail-closed、#3030/#2667/#2621/#988は新Reader/実Source、#951/#1016/#1251は選択ProviderとActual Build、#3035と既存accessは発行の範囲、#3034/#1174はCommand/Host Current Grant、#3033/#1174はApplied/Effect/Receipt、#2516は開示、#1180/#2514は既存独立Expected。新しいPlugin Lifecycleや一律Guardを追加しない。D0純粋関数にはDB Commit義務を課さない。

**状態**：merged #3033/#3034/#3035の局所実装と、Draft #3031の固定HEAD局所CI、旧CLIのsilent skip実観測を**同一E2E成功としない**。この横断接合のProduct実Go/DB/IdP/OS/複数Tenant、承認済みMeaning・正式Owner採用はNOT RUN。#1119 Gateは不変。上記の禁止推論と不足Witnessが既存Referenceで説明可能なら新Core/ACL/Registry/Transaction Engineへの拡張を撤回する。

### 24.6 既存Command実行の認可合成と記録失敗：実コード上の未充足箇所（2026-11-07、静的実コード照合／反証はNOT RUN）

**主眼**：§24.5に追加されてから、[merged #3042](https://github.com/hat47x/tei-platform/pull/3042)（認可未設定拒否と実行後の不明効果）、[merged #3047](https://github.com/hat47x/tei-platform/pull/3047)（宣言許可と発行Grantの照合）、[merged #3038](https://github.com/hat47x/tei-platform/pull/3038)（Reader側の宣言照合）、[merged #3051](https://github.com/hat47x/tei-platform/pull/3051)（未被覆の空集合は未検証）、[merged #3053](https://github.com/hat47x/tei-platform/pull/3053)（CLIの不明効果エラー符号）が進んだ。採用済みの実装が検査できる条件を増やしたが、**別々の検査を一つの実Operationで実行した証明ではない**。

| 実コード・既存実装 | 確認済みの限定条件 | 未成立の接合と停止条件 |
| --- | --- | --- |
| mainの internal/command/invoke.go／[merged #3042](https://github.com/hat47x/tei-platform/pull/3042) | AuthorizerがnilならCOMMAND_AUTHORIZER_UNAVAILABLE、宣言された期間のProviderがnilならCOMMAND_PERIOD_STATE_UNAVAILABLE。Executor実行後の宣言外Effect/Outcome/AppliedはEffectUnknown=true | ReferenceAuthorizerは明示の全件Allowであり本番権限ではない。単にAuthorizer!=nilでも正しいSubject/Resource/Grantを検査したとは言えない |
| mainの internal/command/grant_authorizer.go／[merged #3047](https://github.com/hat47x/tei-platform/pull/3047) | callerが渡したCapabilityごとのDeclared permissionとGrantsを比較し、未宣言/一部不足を拒否。main InvokerにはAuthorizerフィールドが一つある | 既存[merged #3034](https://github.com/hat47x/tei-platform/pull/3034)のBusiness CommandAuthorizer（Subject/Action/Resource）と、Permission GrantAuthorizer（Filesystem/Network等）の**両検査を満たしたか**は別資格。どちらか片方を設定しただけで双方充足としない。自己申告のDeclaredが「空」と正しいRequired Obligationは異なる |
| mainの tools/reference-first-web-frozen-world-generate／[merged #3038](https://github.com/hat47x/tei-platform/pull/3038) | 書く側が-grantsと-declarationを与えた場合はaccess.Groundで宣言照合。実験上、未宣言Capabilityを拒否 | -declarationは任意であり、宣言内の動的具体PathにはReader側Resolution連携が未接続。宣言との照合成功をissuer/resolverの真正性、全Host Grant、OS sandboxへ昇格しない |
| mainの internal/semantic/completion.go／[merged #3051](https://github.com/hat47x/tei-platform/pull/3051) | Deny EmptyPolicyでTargetsが空でもCoverageUnestablishedならunverified。完全な空集合ならfails、部分集合の既知反例ならfailsを維持 | 関数に渡すCoverageCompleteのSource/Query/Tenant/時点の真正性は引き続きCaller側の責務。見えた0件をOwnerの空集合へ変換しない |

**新たな具体的なコード上の問題候補**：main `internal/command/invoke.go`（今回確認blob `9f9099b6a9565d54597f2feb87bb41ce3d0be8d9`）の `execute` で `Executor.Execute` の後に `Recorder.Record` が失敗すると、`ExecutionError{Code:"COMMAND_RECORD_FAILED", ...}` を返すが **EffectUnknownはfalse（未設定）**。main `internal/command/lifecycle.go`（確認blob `60a797be0399015a89f7af3ef1d3d8bd6a72ae29`）の `LifecycleInvoker` は `EffectMayBeUnknown(err)==false` ならRejectedへ進む。外部Executorが先に独立Tx Commitを行い、その後Recorderが失敗した場合、**実DB効果あり／上位Attempt rejected／再送が安全と誤解され得る**。PR #3042は宣言外効果等をunknownに直したが、このRecorder失敗の経路を実証・是正したとは確認できない。これは**Sourceからの反証候補**であり、実PG/故障注入PASSの主張ではない。

**もう一つの記録失敗経路**：同じ `internal/command/lifecycle.go` の `lifecycleRecordError` は、`Lifecycle.Observe` が失敗したとき、または `MarkUnknown` 自身の記録が失敗したときにも `COMMAND_LIFECYCLE_RECORD_FAILED` を返すが、外側の `ExecutionError.EffectUnknown` は未設定である。内部Causeに未知効果が保持されていても、現 `EffectMayBeUnknown` は最初に見つかった外側の `ExecutionError` のフラグを読む。**成功後の段階記録失敗や、unknown記録失敗を「効果なし」と同等に扱ってはならない**。記録保存の可否と外部作用の有無は独立であり、同一実Provider/Receiptで照合する。

**独立期待値を既存#1180/#2514へ返す最小対照（既存#1174のEffect/再送試験への追補）**：

| 対照 | 変える因子 | 独立期待値・必要な実観測 |
| --- | --- | --- |
| RC-01 正常基線 | 受理済みContract/Target、正しいBusiness Authorizer＋Permission Required/Issuedが成立、Recorder成功 | 実Authorization対象とProvider Target/実Effect/Receipt・独立Readの同一Operation接続。全ての実装が接続されなければNOT RUN |
| RC-02 片側認可 | Business Subject/Action/Resourceは許可、Filesystem/NetworkのRequired Grant欠落。逆にPermissionは足りるがBusiness Resourceは別Tenant | どちらの不足でも実Executorに入らず、対象Effect 0を独立Readで確認。単一Authorizer参照用試験だけで両方の強制を証明しない |
| RC-03 記録器故障 | 外部ExecutorがAをCommit、Recorder.Recordだけが失敗（対照はExecutorに入る前の拒否） | Aを確認できたならCommand Report errorと実Effect confirmedを併記し、Attemptはunknownとして盲再送禁止。実Commit証拠が無い場合も「A=0」としない。**現SourceのLifecycle rejected分岐を明示的に検出する回帰試験候補** |
| RC-04 空集合 | 同じDeny EmptyPolicyでTargets=[]、Coverage=Unestablished/Completeを1因子変更 | 前者unverified・後者fails。#3051の既存Go単体試験が確認した範囲を再利用し、別Tenant実ReadのCoverageを自動証明しない |
| RC-05 段階記録の障害 | Executor正常終了後のLifecycle.Observe失敗、またはEffectUnknownエラーに続くLifecycle.MarkUnknown失敗 | 外側の記録エラーでも既に生じたEffect/元のUnknownを消さない。Receipt照会・独立Readまで新たな再送を保留する。現在の外側EffectUnknown欠落とCauseの伝播を別に観測する |

**実施優先順位**：最初にRC-03/RC-05の外部Commit済み＋RecorderまたはLifecycle段階記録失敗を、既存`LifecycleInvoker`とeffectcommit実Provider（可能なら専用使い捨てPG）、同一Operation ID、Receipt lookup、独立Readで検証する。必要ならEffectUnknown伝播/再送停止を#1174/#1163のOwnerで修正し、#3042/#3053の既存不明結果契約を使う。次にRC-02で二つの認可義務が実Targetに共に適用されたことを確かめる。実装の新しい共通Host、Grant Store、Effect Coordinatorを提案しない。

**採用・根拠の区別**：#3038/#3042/#3047/#3051/#3053はmain統合済みの局所実装（それぞれのPRに限定試験記録）。Recorder失敗の不明状態と二重認可強制は今回の**追加受入候補で、Product実試験NOT RUN**。Source形式`*.tei.md`方針は変更せず、[#3031](https://github.com/hat47x/tei-platform/pull/3031)の未接続診断はDraftのまま。D0直接純粋計算には不要な実DB/Permissionを強制しない。#1119の既存判定段階も変更しない。

## 25. 実DSH起動・Receipt観測・データ観測の資格境界（2026-11-07、既存実装と実試行に基づく受入差分）

**この節の追加理由**：これまでの§21〜24はDSHの公開仕様とTEIの局所実装に基づく比較だった。[TEI側の実DSH起動記録 PR #3113](https://github.com/hat47x/tei-platform/pull/3113)が作成され、固定TEI Binaryを外部Harnessから呼び出した実証が初めて現れた。同時に[main統合 #3124](https://github.com/hat47x/tei-platform/pull/3124)と[main統合 #3127](https://github.com/hat47x/tei-platform/pull/3127)で `PersistenceObserver` と `effectcommit.Runner.Lookup` の配線が進んだ。二つの新事実は**実起動したこと**／**Receiptを再読取できたこと**を示すが、**許された範囲だけに隔離されたこと**／**宣言した業務データの実状態を読み直したこと**とは異なる。この境界を、既存Ownerの検証項目として記録する。新しいHarness/Permission Engine、Proof Store、Completion Evaluatorを追加しない。

### 25.1 証拠を取り違えないための実接合表

| 主張 | すでに観測できること | 同じ証拠では判定できないこと／次の正当なOwner |
| --- | --- | --- |
| **外部Tool呼出しの成立** | #3113 の固定TEI commit `50ebf8014340fd1906f6b35cfa30173e0ee2fc6c`／実Binary SHA-256 `186d06ce…69707`で、DSH `0.2.2-rc.1`の1 SessionにTEI起動32回を記録。`commands`/`version`/`model catalog`/`inspect`/`source-index`/`validate`が実Tool経路で成功 | 同じ回答の独立採点、H3 Skillロード・比較、実Agentの正しいOwner Meaning、OS/Network隔離、外部送出許可や秘密の非送出（#2509／TAI #168／#2515／#2516） |
| **読取Scopeの強制** | #3113は`read-only`設定下でもParticipant作業ディレクトリ外の固定worktreeを読めた。二経路のWorld/SourceIndex fingerprintが一致した事実はある | **作業ディレクトリ限定の読み取り隔離は未達**。Toolの名称・ReadOnly宣言・cwd・同一指紋はOS拒否証拠ではない。実Deniedアクセスと許可対象の同じ読取成功の正負対照を必要とする（#2509/#2515） |
| **Receiptの存在** | #3127の`commandadapter.Executor.Observe`は`Lookup=true`で同じRequest/Grant/OperationID/payloadを`Runner.Lookup`し、`StatusReplayed`ならnilを返す。受入票で示せるのは該当Receipt/Operation recordの再確認 | その時点のBusiness Recordの実値、変更後にさらに別Commitが生じた場合の新旧Version、Row/Fieldの実ReadやReplica遅延の不存在（#2998/#1174） |
| **宣言業務効果の観測** | mainの`command.PersistenceObserver.Observe`契約は「宣言したEffectを独立Readで確認」を意味し、nilなら`PersistenceObserved`へ昇格する | #3127のReceipt-only Lookupをそのまま注入する場合、独立Row/Post-imageを読まずに`persistence:"observed"`へ進み得る**意味の不一致**がある。これはSource-staticな契約リスクであり実DB不整合の再現結果ではない |
| **静的Targetの実依存閉包** | [merged #3064](https://github.com/hat47x/tei-platform/pull/3064)で不要なParser/Workspace依存を切り離し、特定Go Tagの`go list -deps`にそれらが出ないことを確認 | D1の実Contract/Provider選択・物理Link/Call、Target Binaryが必要なものを含み不要なものを含まないこと、全Targetの閉包（#2885/#1251）。別Targetの限定`go list`はD1 Product資格ではない |
| **現在の開示資格** | 上記の内部実ログ・Receipt・観測結果には各々の出典がある | 外部Harness／組織／Tenant／Consumerに実Source path・Receipt/Row値・内部診断・件数を提供してよいとは推論しない（#2516）。成功ログに秘密の存在を示さない |

### 25.2 Receipt-only → Persistedの誤昇格を防ぐ一因子対照

**後続main反映による訂正（2026-11-07）**：[merged #3155](https://github.com/hat47x/tei-platform/pull/3155)で以下の旧問題は修正された。現在は `not-applicable`／`not-observed`／`receipt-observed`／`resource-observed` に分離し、`persisted:true` は `resource-observed` の場合だけ。旧§25.1・以下の段落に登場する `PersistenceObserved`／`persistence:"observed"` は **#3127時点の歴史的問題記述**であり、現mainの現行仕様・未修正事項ではない。現物の `DetailedPersistenceObserver` と `ResourceRead` で区別できるが、後述の実Composition配線・Resource Reader権限・同時変更/Replica/Replayは未検証。

[main `internal/command/persistence.go`](https://github.com/hat47x/tei-platform/blob/main/internal/command/persistence.go)では `PersistenceObserved` は「宣言した効果が独立読取で見えた」状態であり、`observePersistence` はObserverからのnil応答で `observed` を返す。一方[main `internal/security/effectcommit/commandadapter/executor.go`](https://github.com/hat47x/tei-platform/blob/main/internal/security/effectcommit/commandadapter/executor.go)のReceipt Lookupは、実データの現在値を読まずに成功する。両者の**合成時の意味**を#2998で照合する。

- **OBS-01（同一Commit後の正常対照）**：同一Operation/Grant/Target/ExpectedVersionのCommit→Receipt Lookup→別接続で業務Row/Fieldを独立Read。三つを同一時点と仮定せず、観測Version/時刻を別記。Receiptあり、Rowは期待post-imageなら各々をその事実の範囲でqualifiedにする。
- **OBS-02（誤昇格の識別例）**：Operation AのReceiptは有効なまま、その後別Operation Bが同じ業務Rowを合法に変更。AのReceipt Lookupは`StatusReplayed`でも、現在のRowがAのpost-imageと一致するとは限らない。Receipt成功だけを根拠に「現在の資源値がAの効果を示す」「Owner Completion Holds」と表示しない。Aの過去CommitとBの現在値を別に保存する。
- **OBS-03（まだ読めない場合）**：Receipt Lookup成功・独立Row Read失敗／権限拒否／Replicaが旧版なら、Receipt観測は成功として残しても**データの観測はnot-observed/unknown**。現在の開示許可も別に評価する。Receipt不在をEffect 0とも断定しない。
- **OBS-04（記録障害と再送）**：既存§24.6 RC-03/05のRecorder/Lifecycle記録失敗後、Receipt LookupとRow Readを使う。観測失敗をクリーンな拒否とせず、同Operationの照会と再送停止を維持する。mainの `COMMAND_RECORD_FAILED`／`lifecycleRecordError` は今回の静的確認時点ではEffectUnknown伝播が未修正で、実故障注入はNOT RUN。

**現mainの受入ルール（#3155後）**： `receipt-observed` を業務資源の再読取済み／ `persisted:true` と読まない。`resource-observed` は別poolから読んだ現在の `objects` のpayloadが比較時に一致したという事実であり、Owner意味受理・現在Grant・Tenant越境不能・未来も変わらないことを示さない。`not-observed` は実効果0でなく、`ResourceRead=false`／Reader権限不足／レプリカ遅延・不一致を区別して検証する。新しい状態Enumや二重Observerを作らず、実Compositionでの明示配線と既存独立期待値を最優先する。

### 25.3 DSH実証の次の比較資格

[PR #3113のEvidence](https://github.com/hat47x/tei-platform/tree/docs/dsh-h2-readonly-evidence-20261107/plan/evidence/DSH_H2_TOOL_ONLY_READ_ONLY_2026-11-07)は**実起動32件を含む限定1 Session**。提示されたTAI側の4 Session比較に転用しない。未完の条件は、実Taskのanswerを独立期待値で採点すること、比較参加者へ渡す情報とそれ以外を**実OS拒否で隔離**すること、許可済みSourceとの正常対照、Harnessが取得したRaw Tool resultと外部送出の観測、Skillの実byte/実Load、H2/H3の同条件比較。read-onlyが作業ディレクトリ外を読める事実を隠すために全Hostの観測を消さず、**Participantへ見せる公開範囲をOwnerの許可で絞った実環境**で再実証する。比較のTask/採点/Skill正本はTAI #168、TEI接続実証は#2509、情報境界は#2515/#2516に委譲。

**検収優先順位・状態（#3155後）**：#3131のReceipt Lookupと#3155の実Resource Readは **main統合済み**で、#3157は `postgres-conformance` の限定実証を記録する。OBS-01〜04全体を未実装と呼ばない。未完了なのは本番に相当する実Compositionで `ResourceRead` をどの権限・実接続・観測時点で使うか、OBS-02/03のRevision/権限/Replica/別Operationの反証、RC-03/05の記録障害、Consumer開示、隔離適格DSH比較、D1実Link/Invoke、Owner受理。新Core/Host/Registry/別Runbookは作らない。

## 26. 外部Harnessの送出・読取境界を実測から再資格化する（2026-11-07、Scope/Disclosure横断追補。実測は既存#2515を正本とする）

本節はDSHの公開構成や想定上のリスクではなく、[main統合 #3153](https://github.com/hat47x/tei-platform/pull/3153)／[#3161](https://github.com/hat47x/tei-platform/pull/3161)の**ローカルMock要求の実byte/構造観測**、[#2515](https://github.com/hat47x/tei-platform/issues/2515)の合成Canary、[main #3152](https://github.com/hat47x/tei-platform/pull/3152)のTEI FileSystem境界、[PR #3162](https://github.com/hat47x/tei-platform/pull/3162)のCLI実探索を照合する。データ保護の判断権限・実Egress/OS隔離の正本は#2515／TAI #232であり、この文書で新たなPolicy Engineを定義しない。

### 26.1 三つの保護面は独立し、Plugin Scopeの許可を継承しない

| 保護面 | 限定的に実証済みのこと | 禁止する推論と必要Witness |
| --- | --- | --- |
| **TEI管理のFileSystem** | #3152：`os.OpenRoot` を使ったTEI FileSystem操作では、Windows junctionを通じたRoot外書込みをOSが拒否した。通常パス書込みは成功、Linux/Windowsの対象試験を記録 | 外部DSHの `read`／`pwsh`、子Processや直接OS/APIへの読取まで閉じ込めたとは言えない。OS identity/ACL/mount/Network境界で当該Consumer/Targetの正負対照が必要 |
| **Harnessが読めるSource** | #3113：DSH read-onlyでもParticipant cwd外の固定Workspaceを読めた。#2515の合成Canaryでは `allowed/projection.txt`・`raw/secret.txt`・`.env`・cwd外 `peer.txt` がいずれも読取完了 | `read-only`／cwd／Plugin Indexに載らない／Tool広告がread-only から、非公開Sourceの非到達性を推定しない。許可Sourceのみ読め、Canaryを実OS/Server境界で拒否した記録が必要 |
| **Harnessからの送出** | #3153のMockでは合成Tool markerがモデル入力に1回・`dsh_session_log`に2回。#3161ではDSH `0.2.2-rc.1`の**既定ProfileでSession Log fieldが有効**。503または200空SSEでは水位が進まず後続要求で同区間を再送 | `enabled:false`・read-only・応答503・Session閉鎖・Prompt上の秘密保持を実Egress禁止や外部Provider非保存へ昇格しない。ネットワーク送信先・フィールド・条件・実OS/Proxy拒否を別に観測 |

**重要な因果区別**：#3153/#3161は `127.0.0.1` のMock Receiverであり、実クラウドProviderへ当該秘密を送った観測ではない。しかし送出内容にはUser/System指示、cwd、Tool結果が含まれることを実測した。副次送出は「明示したOutputを開示したか」だけでは捉えられない。Runtime/Disclosure Scopeは **Tool本体＋結果＋Session Event/Retry/タイトル生成＋送出先** を区別し、機密情報の存在、診断、件数を含めて必要な投影以外を露出しない条件で評価する。実Customer DataやCredentialを使って再現しない。

### 26.2 独立期待値へ返す最小一因子対照（EGR-01〜04、今回の新E2EはNOT RUN）

| 対照 | 固定と一因子変更 | 独立期待値・追加の停止線 |
| --- | --- | --- |
| **EGR-01 許可済と禁止Canary** | 同じDSH Profile/Tool、許可Projectionから `raw/secret.txt` / `.env` / cwd外へPathだけ変更 | 許可は正常Read、禁止Pathは**Tool実アクセス前に拒否**、実外送Content 0。既存Mockで「読めた」対照はあるが、実OS/Server拒否の正例は未取得 |
| **EGR-02 Session Log既定と制御** | `enabled:true`／`false`／overlayなしを一因子変更 | 既定ONの実Mock結果を保持。falseで当該fieldが消えても他Egressが0とは推定しない。再有効化時のBacklogを改めて確認。外部Provider/他Endpointは未観測 |
| **EGR-03 再送・未受理** | 正常stream／503／200空SSEを同じSession Eventで比較 | 受理watermark、同payload再送の有無、送出先、回数を分離。同イベントの再送を新しい業務Effect/新Sessionとして重複計数しない。Ack未達は送出0でない |
| **EGR-04 Read-only表明と局所書込** | 同じ `tei view` で `-execute`＋ `-operation-log` / `-lifecycle-state` の有無だけ変更 | [#3162](https://github.com/hat47x/tei-platform/pull/3162)の実探索はファイル生成（215B/12288B）を記録。Command宣言のread-onlyを「ファイル副作用なし」に読み替えず、RunPlan/Journal/Temporary/Networkを別々に宣言・許可・実観測。参照実行の `persistence:not-applicable` と補助記録のファイル生成は矛盾しない |

**完了判定**：#2515／TAI #232で OS/Serverによる非公開Source読取拒否と実Egress阻止の両方を証明する前に、比較適格H2/H3の合格としない。#3113実Tool起動32回は歴史的限定PASSとして保持するが、H2/H3正式比較の分母へ入れない。#3152のFileSystem修正を外部Harness全体の隔離資格としない。#3161の既定ON/再送は実測済みであり、未知の全送出路が安全だとも実Providerに当該情報が届いたとも推定しない。#2516がAudienceごとのProjection、#2509が接続実験、#2899が方式研究、#2885/#1251がD0/D1/D2方式を所有。新しいAgent Runtime、System-wide Sandbox、Universal Registry、Fixture正本、Permission Schemaを増設しない。

### 26.3 Plugin採用と実行方式の判断への適用

**方式の候補を選べることと安全な物理隔離は別**。D2外部Processでは、PluginやHarnessがTEI FileSystem Brokerを通らない読取・通信を行う場合、`os.Root`/Grant検査の限定PASSを全Processの保証へ拡張できない。D1静的合成では同一Processの呼出し境界が実際にどこまで強制されるかを確認し、D0純粋計算へ不要なOS隔離義務を課さない。方式の採否は[#2885](https://github.com/hat47x/tei-platform/pull/2885)へ委譲し、Consumer別のSource Disclosure・Effect/Grant・実行環境適合が未確認なら「方式が実装可能」から「採用可」へ進めない。

**提案／採用の区分**：#3152/#3153/#3155/#3161はmainの限定実装・実測結果。#3162はDraftでなく未統合の実探索報告。EGR-01〜04として記述した追加の厳密な横断比較、実OS隔離＋Egress非送出、H3 Skillロード／4 Session比較、機密Profile、Owner受理は未実施。ADR-0035のA3機械確認とA4外部効果の確認は別に扱い、旧来の「すべての業務意味・権限変更はA4で人へ戻す」という一般化をここで復活させない。

## 27. DSH実Scope分離と暗黙Context：Plugin発見・Function利用・実境界を再接合する（2026-11-07、現物実証からの最小追加。試験正本は各Owner）

**変更点**：§26の「read-only/CWDはOS読取・Egress境界でない」を、別々の**書込保護／Lane間読取／暗黙Context送出／Tool実公開／選択Source閉包**へ具体化する。これは「すべてを一律に新しいPlugin Engineへ載せる」という提案ではない。PluginのDistribution・Composition・Runtime・Disclosureを、**Functionの使用対象／実行環境で有効な対象／Audienceへ渡せる内容**で個別に判定する。

### 27.1 新しい現物を踏まえたClaim上限

| 実観測・由来 | 合格または不成立が確定した範囲 | Plugin合成・外部Harnessに適用する停止線 |
| --- | --- | --- |
| [main #3228](https://github.com/hat47x/tei-platform/pull/3228)／[#2515](https://github.com/hat47x/tei-platform/issues/2515)：DSH `workspace-write` | Windowsの当該設定ではRoot内書込成功・Root外書込は`UnauthorizedAccessException`で拒否。一方、同じRoot外の既存Canaryは絶対Path・`../`双方から読取成功。Sandbox初期化前のDACL修復も実験前提 | Writeが拒否された＝Readも拒否、あるいは修復後OS ACLが安全、としない。読取/書込を別の実OS正負対照で観測。ACL変更は実測の前提であり一般利用者への修復手順の推奨ではない |
| [main #3216](https://github.com/hat47x/tei-platform/pull/3216)／#2515：別DSH Lane | 別CWD・別SessionのLane A/Bが互いの合成Canaryを読み、モデル入力・Session Logへ載せた。別OS User/ACL/mount/Network隔離は**未構成** | Worktree・Session・Plugin Profile・Publisher/Tenant IDが異なるという論理分離を、Source/Secretの実物理分離としない。別Lane・別Tenantの機密採用時は認証主体＋OS/Providerアクセス強制と送出拒否を要実証 |
| [実Mock PR #3191](https://github.com/hat47x/tei-platform/pull/3191)／[#2515 H6-E3](https://github.com/hat47x/tei-platform/issues/2515)：暗黙Context | Toolにより明示Readされていない`AGENTS.md`／`CLAUDE.md`の合成markerとSkill frontmatterの`description`がモデル入力／Session Logへ載る一方、Skill本文は未ロード・未送出 | `Tool was not called`＝未読／未送出とは判定できない。Accessible、Loaded as implicit instructions/description、Tool Read、Sent over Wire、Consumer Grantedを別列。Skill説明文の自動発見は本文の資格でも公開同意でもない |
| [main #3210](https://github.com/hat47x/tei-platform/pull/3210)／#2509実H2再読：CLI公開面と結果 | 固定観測では53コマンドをCLIが受理、`tei commands`は19件を選択して公開、34件は意図的に非広告。DSH H2実記録はTool Call 56／Result 55、Catalog Tool保存は8192BでJSON末尾不完全、他保存にはstdout JSON後のstderr付随がある | Advertised Tool集合≠実CLIで実行可能な全Command集合。未広告を即「漏れ」や「無権限」とみなさず、外部Harnessが実際に起動できる`argv`／Flag／出力先／ネットワークを許可された範囲に限定。Exit0/Session Completed/一部fingerprint一致から必要な全Result受領・完全なReadを導かない |
| [Draft #3217](https://github.com/hat47x/tei-platform/pull/3217)／[#1251](https://github.com/hat47x/tei-platform/issues/1251)：外部Module閉包 | `cmd/tei`の選択Sourceについて、Repo内318 file・外部12 Module/325 fileを観測し、`qualified:false`と明示。別の実Build・byte再現性が成立しても、現`ResolveSourceClosure`は`rejected` | Plugin Selected／実Compile／Artifact Digest／依存選択観測／Qualified Closure／B実Callを独立の判定列に維持。外部Moduleの観測Digestを既存Qualified Digestに置換しない。決定Ownerは#1090/#2831/#1251／方式#2885 |

**横断した意味の接続**：[Draft #3225](https://github.com/hat47x/tei-platform/pull/3225)／#3192／#3179の標準データ構造／機能非依存の表現型／Function内のBound Representation Instanceを前提とする。**表現型の存在・Pluginの広告・Functionの操作Offerは、実Data BindingやHost Grantを生成しない**。Headless Functionが表示Slot 0で成立し得ることを認め、表示や非表示、Tab/Graph/Audio等の表示差を機密Source開示／業務Operation許可へ転用しない。媒体型・選択同期の仕様は#3192/#3222、実Operation/Action bindingは#3172/#1127/#2507、実Provider/Receiptは#1174/#2586の正本へ委譲。

### 27.2 既存受入票へ接続する次の限定対照（新Fixture正本を作らない）

- **H6-E1/E4の正負対照**：同じ合成Canary・ToolでAllowed Projection、別Lane/Root外のRawの2件を読む。DSH Mode `read-only`／`workspace-write`切替による**Write拒否だけ**をRead資格に加点しない。別OS identity・ACL・AppContainer/mount等の実構成で**禁止ReadがOS/Serverに拒否される**までは機密Laneの比較適格を保留。
- **H6-E3/E2の送出対照**：明示Tool Readを行わず、`AGENTS.md`／`CLAUDE.md`／Skill descriptionだけを変えた場合のモデル入力・Session Log・Retry・Title requestを同じMock Receiverで比較。暗黙Contextの流入、Tool Result由来の送出、未観測Sinkは別列。実秘密や実CredentialをFixtureへ入れず、実Provider保存を推定しない。
- **外部H2/H3のResult完全性**：固定Command/args/Exit/STDOUT/STDERRそれぞれの実byte長・digestとHarness Result ID/受領byteを照合。8192B切詰め・Call 56/Result 55を欠損として表現し、JSON先頭のFingerprint一致を完全Catalogや採点済みAnswerへ昇格しない。許可Command family（Read-only広告とCLI実実行範囲）を固定し、H3のSkill descriptionの自動注入と本文実Loadを区別。
- **表示・Provider実使用**：同じData Shape／Function ContextでBound Presentation Instance有無・Rendererを変える対照と、D1/D2の選択Providerを変える対照を別に実施。Presentationの表示成功はFunction→Query→実Provider Read、Current Grant、Effect/Receipt、Go Target Build closure資格の代替にならない。

**計画の責務と優先順位**：#2515／TAI #232は実OS/Network分離と送出、#2509／TAI #168は外部H2/H3比較、#2516はConsumer開示、#3210はCLI広告範囲、#3192/#3225は表現型/Function束縛、#1090/#2831/#1251/#2885は実Build/Plugin資格。実Reader/Effectは#2586/#1174。優先するのは新Plugin Runtime設計ではなく、**すでに観測できている負例に対する禁止Readの実強制、完全Tool Result、真のSelected→Build→Invokeの正負対照**。適用可能な既存Fixture/Expectedを利用し、第二Runtime/Host/ACL/証拠Storeを作らない。

**状態・撤回条件**：#3216/#3228の限定Canary実測、#3210の実CLI差分はmain統合済み。#3191実Context証拠、#3217 Build観測、#3225 Function/表現型計画は各PRの範囲のみであり、main／Owner採用を一括主張しない。実OSによる**禁止Read**、外部Egress遮断、完全なH2/H3比較（4 Session）、D1選択Bの実Go Link/Invoke、Data→Function→Presentationの実Provider/GUI/Host資格、Ownerの正式意味受理はこの追補では**NOT RUN**。ADR-0035のA3機械確認／A4外部効果の確認方針は変更しない。

### 27.4 現行Plugin-first正本に渡す、実証済みの部分と新しい停止条件（2026-11-07、H3実Load・Session再開・Trait/BOM/Testの横断差分）

**正本の更新**：[main統合 #3277](https://github.com/hat47x/tei-platform/pull/3277)により、CLI/Studioから標準Pluginを作成・選択・試験・利用し、生成Application Runtimeへ同一Contract/Resolver/Build/Testを再利用する**Plugin-first dogfooding**は [`PLUGIN_FIRST_TOOLING_RUNTIME_DOGFOOD_2026-11.md`](PLUGIN_FIRST_TOOLING_RUNTIME_DOGFOOD_2026-11.md)／RQ-008が実施計画の正本となった。本書§21〜27は**DSHに由来する独立の反証・停止条件**に限定する。D0棚卸し→D1一件のread-only Inspector/Model CatalogをCLI/Studio二Consumerで実利用→D2〜D6へ段階化する進め方を複製・再定義しない。CLIの手書きdispatch、Wailsの固定Bridge、Reference `internal/composition` の実製品未接続は現mainのFit/Gapとして扱い、Everything is Plugin実装完了とは呼ばない。

| 最近成熟した別Ownerの成果 | §27以前の記述を補正する点／本書で追加する一つの受入条件 |
| --- | --- |
| [H3実Skill Read／#2509](https://github.com/hat47x/tei-platform/issues/2509)、[現行AI-first計画 #3241](https://github.com/hat47x/tei-platform/pull/3241) | §27.1の「Skill本文未ロード」は**H6-E3でdescriptionだけを自動注入した合成試行に限る**。別H3試行ではSkill本文各**5,482 bytes**と参照文書を実Toolで読んだ。ただしH2-A再利用・禁止Read非隔離・独立採点未充足なので**比較適格H2/H3は0/4**。自動発見／description注入／本文実Read／正しいPlugin Provider実Invocationは四つの異なる証拠 |
| [DSH Session再開実測 #3237](https://github.com/hat47x/tei-platform/pull/3237) | 同一Session再開で、ACK済み`dsh_session_log`区間には合成Rawの再送が無くても、過去の`Tool Result` Rawが新しいモデル入力に再登場。**Log Watermark更新・Plugin削除・Grant撤回から、過去のモデルContextまで消去／非再開示されたと推論しない**。Resumeの実Audience・送出先・失効時点は#2515／TAI #232が判定。外部Providerへの送出は未実証 |
| [Open/Stable/Internalの比較Draft #3265](https://github.com/hat47x/tei-platform/pull/3265)、[Trait実装義務Draft #3282](https://github.com/hat47x/tei-platform/pull/3282) | **Open/unpromised**は既配布・参照許諾済み能力の下流利用であり、未公開Source/Skillの説明・Secret/Host Resourceの自動開示ではない。**Atomic Trait**のRequiredは有資格Defaultか明示Manual実装を要求し、Hintや自動検出したPlugin名だけではFulfilledにならない。Open→Stable保証もTrait fulfilled→Current Grantも別資格。#951/#1016／#3278がOwner、ここで別ABI/Export Registryは作らない |
| [Verified BOM/Matrix Draft #3269](https://github.com/hat47x/tei-platform/pull/3269)、[mainのdogfood T0 #3264/#3266/#3268](https://github.com/hat47x/tei-platform/pull/3268)と[Design Context Draft #3284](https://github.com/hat47x/tei-platform/pull/3284) | supplied exact Lock相対のPairwiseで候補内9/9ペアを覆っても宣言全12ペア中未出現3ペアは「不可能」ではなく**unrepresented/unknown**。`qualified:false`維持。T0の9実CLI Read／6 pinned Scenario Step／既知負例／CI成功は**T1新Scenario生成能力**やRule/DependencyのRequired Closureを満たさない。Plugin InspectorがCLI/Studioへ返すDesign ContextのRequired/DeliveredとTest Coverageの不明を先に測り、BOM/Trait実資格へ無断合算しない |

**次の最小縦断受入（既存試験正本へ渡す）**：#3277のD1 read-only Inspector候補について、(1)同じSource/Required Context/Current DisclosureからCLIとStudioが同じContract/Providerを**実選択→実Invoke**（各Consumerの実受領Byteを固定）、(2)Source変更・必要Traitの未実装・Version Lock候補欠落・Provider A/B入替えを**別因子**としてそれぞれ`unqualified`に止める、(3)説明/Skill/Session再開の暗黙送出が禁制Canaryに到達しないことは#2515の**実OS/Server拒否**で証明、(4)実変更後の必要Scope/Grant/Effect/Receiptは別Claimとする。最初のInspectorはread-onlyであり、DB Commitや全外部Process隔離を一律に課さない。Design Contextの不足診断は#3262、実Scenario/独立Expectedは#1180/#2514、Traitは#3278、Matrixは#3267、CLI/Studioは#1158/#1824、Build実Linkは#1251/#2831の既存Ownerを使う。

**実施と停止線**：#3277は**計画・要求のmain統合**、H3 Skill Readは限定実績、#3237は合成Mockの再開送出実測、#3264/#3266/#3268は限定T0、#3265/#3282/#3269/#3284は各Draft／Referenceの範囲。今回のDocs編集で新CLI/Studio共通Plugin、Provider B実Go Link/Call、正規OS禁止ReadとEgress拒否、T1新Scenario、Qualified BOM/Trait、正式H2/H3四Sessionが成立したとは主張しない。ADR-0035のA3機械確認・A4外部効果の判断を維持し、旧「Draft解除を全Repository Health/人のMeaning承認まで一律保留」という規則は復活させない。

### 27.5 Plugin-first最初のInspectorを、Authoring Scope・実Tool出力・保持先で同時に取り違えない（2026-11-08、main実Source・他Ownerの実証を接続）

**引継ぎ先の優先順位を固定する**。[main統合 #3277](https://github.com/hat47x/tei-platform/pull/3277)／[#3285](https://github.com/hat47x/tei-platform/pull/3285)の[Plugin-first tooling実施計画](PLUGIN_FIRST_TOOLING_RUNTIME_DOGFOOD_2026-11.md)・RQ-008/032が実装順の正本である。本書は、DSH実H2やSource/Grant/Build資格を**誤ってD1製品合格へ合成しないこと**のみを担当する。最初のread-only `inspect` でCLI/Studioが選択した同じ契約・同じProviderを実呼出ししたかを最小縦断として扱う。二Consumerの同じ出力だけでは、誤Provider Aを共通に呼んでいる可能性が消えない。

**現在コード上の明示的な遮断点**：[main統合 #3300](https://github.com/hat47x/tei-platform/pull/3300)の[Authoring Route Fit/Gap](../evidence/PLUGIN_FIRST_D1_AUTHORING_ROUTE_FIT_GAP_2026-11.md)は、`internal/plugin/runtime/service_carrier.go` の `validateServiceSelection` が `application-runtime` 以外のScopeを拒否し、`internal/studiohost/readonly.go` とCLIの `inspectWorkspaceJSON` が既存 `operations.InspectSubject` を直接呼ぶ実Sourceを指摘する。したがって**旧Service CarrierのScope文字列だけ変更する／直接関数をPlugin実呼出しと呼ぶ**案は不採用。既存Host-owned Root/Workspace allowlistを維持したまま、`authoring`に必要なSelected Contract/Provider/Invocationを#951/#1016/#1158/#1251の既存機構へ明示的に結ぶ。実装案やAS-01〜12は#3300にあり、第二Host・Resolver・Runnerは不要。

| 最小対照 | 同一条件で変える一因子 | 認める証拠と保留する資格 |
| --- | --- | --- |
| **INSP-01 exact Scope/Provider** | 同一Source/Contractのまま `authoring`選択B→実B呼出し、選択B→旧Aを実呼出し | CLIとStudio各々の実Caller/選択B/Invocation receipt/Result bytesがBに束縛される正例と、出力が同一でもA呼出しなら拒否する負例。既存Carrierの`application-runtime`への自動fallbackは失敗 |
| **INSP-02 Workspace/Context** | World fingerprint一致で異なるWorkspace名／manifest／Required Source/Loader結果に替える | [open #3311](https://github.com/hat47x/tei-platform/pull/3311)のWorkspace identity欄は未main統合であり、同名やmanifest hashだけで同一Scope・Current Grant・必要依存の全取得を確立しない。[main #3284/#3296](https://github.com/hat47x/tei-platform/pull/3296)のContext Gateが`partial`/ `qualifiedTestGeneration:false`ならT1自動生成の資格は保留 |
| **INSP-03 CLI広告と実I/O** | `view`でプレビュー呼出し／`-execute -operation-log`呼出しを対照 | [merged #3307](https://github.com/hat47x/tei-platform/pull/3307)の`argumentValues`／型・既定値の実CLI一致は限定成功。ただし[Draft #3302](https://github.com/hat47x/tei-platform/pull/3302)の`excludedFlagNames:execute`は未main統合。Command名がread-onlyでも呼出しvariantがファイルを生成し得る。Toolに広告した`argv`／実Effect／stdout・stderr／Consumer受領byteを別に照合 |
| **INSP-04 実H2の受領／保存** | 同じCanaryのTool結果を新Session・再開Session・ローカル保存先で対照 | [merged #3237](https://github.com/hat47x/tei-platform/pull/3237)の再開はACK済みSession Logが非再送でも過去Rawが新モデル入力へ復活。[open #3309](https://github.com/hat47x/tei-platform/pull/3309)の合成Canaryは展開済み永続Session Logに残り、複数連結zstd frameの最初だけを展開すると見落とす。秘密が現れない判定には全frame/全許可Sinkの観測が必要。外部Provider保存・実OS拒否はNOT RUN |
| **INSP-05 Test/Lockの資格** | 実Source/Plugin Lockのbyteを1因子変更し、生成済みScenarioのPinを古くする | [merged #3299](https://github.com/hat47x/tei-platform/pull/3299)のCIは既存Pinned Scenarioに旧Source/Lockの実拒否を追加。[merged #3304](https://github.com/hat47x/tei-platform/pull/3304)は既知Ruleの行番号0候補を第7Stepとして実CLIで判別した限定T1前進。ただしAIが未知Caseを盲検発見したこと・実Provider/BOMの全組合せ・C1 Coverage・D1 Plugin実Callは未成立 |

**二つの実証レーンを分離**。Aは**TEI製品内**のInspector D1a shadow→D1b切替（本節INSP-01〜03のScope/選択/実Callと独立Context）。これは外部DSHの機密隔離成立を前提にしない。Bは**外部Harness経由**のH2/H3（INSP-04）：既存#2515／TAI #232の実禁止Read／実Egress境界とTool Call/Result完全性を満たしてから正式Trialを計上する。Toolに読ませないSkill descriptionの暗黙注入、保存済みSession Log、再開後の旧Raw再登場をDisclosure/Retentionで別評価する。内部Aの成功が外部Bの許可になることも、Bの失敗が内部Aの試験実行自体を妨げることもない。

**採用先・次の最小実装**：#3300のAS負例と#3277のD1移行を利用して、`authoring`用のread-only Inspectorに1 ProviderのSelected→Build actual→Invoke／CLI・Studio二Consumer受領を通し、B選択→A呼出しを故意に失敗させる。#3262 Context Gateは`partial`の表示を保持。外部Harnessの実隔離、セッション保持先の全Sink/削除とAudience許可は#2515／#2516、Pin/Matrixは#3267、独立Expectedは#2514のOwner。要求数や全Plugin/GUI書換えを増やさず、根拠が欠けるClaimは`unqualified`で停止する。

**今回の実施の上限**：#3277/#3285/#3300のmain統合は**設計とFit/Gap**。#3284/#3296/#3299/#3304のCIは特定の限定Context/Scenario判別で成功。#3309のローカル保存はPR上の合成Canary実測、main未統合。#3311 Workspace欄・#3302 CLI read-only Variantも未統合。今回のInspector Provider実Build/Invoke、完全H2/H3 4 Session、OS/Network隔離、実外部Provider、Owner業務意味・Plugin製品資格は**NOT RUN**。ADR-0035 A3の変更範囲機械検証・A4の外部効果確認に従い、旧Draft一律保留規則を復活させない。

### 27.6 Capability選択の粒度と独立Context・Build主体の実測を最初のInspectorへ

**後続mainによる状態更新（2026-11-08）**：以下の#3350「open」、#3324「Draft」、#3336「L3未達」は**各観測当時の記述**。#3350はmain統合済み、#3324は重複を統合したmain #3354へ継承、#3360は**参照driver自身**のStage1→Stage2 L3を実証した（`cmd/tei`のL3は未実施）。現行の実装／Claim上限は§27.7に示す。新しい比較結果を旧Runの証拠へ遡及適用しない。

（2026-11-08、前節の実Provider二Consumer試験を実装に近づける差分）

この節は§27.5に記したInspector D1/CLI+Studio同一Provider実呼出しを**繰り返さず**、その直前に測る必要のある`Environment Binding`の粒度、および読取・Buildの観測者の同一性を最新mainのSource/限定試験に基づき補足する。実装優先順位の正本は[mainのPlugin-first計画](PLUGIN_FIRST_TOOLING_RUNTIME_DOGFOOD_2026-11.md)／[merged #3300](https://github.com/hat47x/tei-platform/pull/3300)。独立検証基準のOwnerは#2514、実Contextは#3262、実Buildは#2831/#1251であり、第二のResolver/Registry/Gateを導入しない。

#### 実観測から区別できた三段の選択

[main統合 #3337](https://github.com/hat47x/tei-platform/pull/3337)は`World.Plugins[].Provides`のKind＋Capability IDとEnvironmentの`deployment-executor`／`observation-provider`／`evidence-provider` Bindingを照合するread-only Referenceを実装した。対象`reference/workspace`では宣言Capability 12件のうち、当時の照合規則で「未選択」6件。**未選択＝不具合ではない**（候補の同梱、別環境向け、Plugin本体のみ選択、同じPlugin内の別能力だけ選択等）。`editor-intelligence`はPluginが明示選択されても当該Capability Bindingは未選択、`migration-provider`は同じIDが別Kindの`deployment-executor`としてのみ選択される具体的反例。

ただし`database-instance`の`postgres`／`oracle`という**技術名**は`database.postgres`等の**Capability ID**ではない。#3337では2件を数えず未選択6件としたが、[open #3350](https://github.com/hat47x/tei-platform/pull/3350)は既存`internal/validation`の`Capability.SupportsDatabaseProvider(instance.Provider)`を用いた修正を提示し、同じSourceで未選択4件／両Database Providerは選択済みという局所観測を報告。**#3350は本節時点で未main統合**。二つの計数は解決規則・Source版に相対的であり、互いの歴史的観測を上書きしない。新しいKind↔技術名対応表をPlugin層へ複製せず、既存Validatorの採用判定と同じ規則へ参照経路を揃える。未選択能力数だけでBOM・Plugin削除や実装不足を決めない。

| 新規対照 | 固定条件と変える一因子 | 独立期待値／Claim上限 |
| --- | --- | --- |
| **SEL-01 packageとCapability** | 同じPlugin package/version/provides、EnvironmentでPlugin本体のみ選択→Kind+ID Bindingを追加 | Package selectedのまま特定Capability未選択を明示。Capabilityごとの選択が必要なFunction/Inspectorでは、選択前に実Provider Invokeしない。選択後も実装適合・Build/Grantとは別 |
| **SEL-02 capability Kind誤同一視** | 同じCapability IDを保ち、`deployment-executor`と`migration-provider`でKindを切替 | IDだけ一致しても別Capability。依存を満たすと推論しない。Bindingを持たない能力は`unselected`、Kind不一致は`mismatched`等、原因を分ける |
| **SEL-03 technologyとの写像** | 同じDatabase Instance `provider:postgres`、同じDeclared Capabilityで支持技術だけ変更 | `SupportsDatabaseProvider`に沿って対象Instanceを選択／非選択に区別。技術名をCapability IDの文字列と比較しない。#3350の局所実装を製品Provider/DB実Readの証明へ昇格しない |
| **SEL-04 正常JSONだがRequiredが欠落** | World/SourceIndex FPとJSON schemaは正常のまま、Inspector用Required Rule/Relation/Source Anchorの1要素だけを落とす | [Draft #3324](https://github.com/hat47x/tei-platform/pull/3324)の独立Required `validation.tei`＋`inspect-validation`設計に合わせて`partial`／`qualifiedTestGeneration:false`で停止。正常JSON・一致FP・Catalog存在から必要閉包が完備としない。未実行のSourceを0件と読まない |
| **SEL-05 Build driverと実Artifact** | 同じSource/Lock/child argvのもとで実Build Driverと被起動実行物を別Digestで観測 | [main統合 #3336](https://github.com/hat47x/tei-platform/pull/3336)は子Build executable/PID/exit、報告Artifact Digestと実disk Digestの突合を参照実装で記録する。ただし**Driverは生成済みTEI Stage1ではなく**`l3Claimed:false`。D1実Targetが選択Provider BをLink/Callする資格、L3 Self Build、Owner受理は独立 |

#### 受入手順と既存成果の再利用

**P0（実装に着手する前の小さな測定）**：#3337のread-only Plugin Capability Referenceと既存#951/#1016の選択結果を同一World・同一Environment・Source版で比較する。これが不一致なら「Package/Capability/Technologyのどの解決が違うか」を特定し、#3350が未統合である限り6件と4件を同一HEADの結果として混在させない。併せて、[main統合 #3307](https://github.com/hat47x/tei-platform/pull/3307)のCLI引数集合と[main統合 #3333](https://github.com/hat47x/tei-platform/pull/3333)の`writeFlags`による名前付き出力の観測を使用する。§27.5の古い「`-execute`と書込フラグは未公開」という読みを現mainへ流用しない。Effect語彙自体の分割は未決。

**P1（最小D1縦切り）**：次に`authoring` Scopeのread-only Inspector 1能力だけで、Package→Kind/Capability Binding→exact Contract/Provider→Target Build actual→CLI/Studio各Callerの実Invokeを**同一Attemptで突合**する。PackageがSelectedでもRequired Capability未選択、Bを選んでAを呼ぶ、正しいBでもRequired Rule/Relation不足、actual Build target不一致を一因子ずつ拒否する。必要TraitのRequired/Default/Manual/Provided履行・Version Matrixは既存#3279/#3267へ引き渡し、開発時のTrait fulfilledだけで実Grantを与えない。

**独立レーン**：#3324のContext Gateは既存実CLIでのT0.5→T1-A精度向上であり、D1 Productへ暗黙依存させない。#3336のBuilder Driveは#2831でのSB-C0/C1局所前進であり、本節のD1実Build/Callが済んだ証拠でもL3 Self Build完成でもない。外部DSHの実OS Read/Egress隔離や永続Session Sinkは§26〜27.5／#2515が所有する別レーン。

**状態**：#3337/#3336/#3307/#3333はmainへ統合された限定Reference/CLI改善。#3350と#3324は未統合の別Owner差分。**SEL-01〜05の同一D1製品Attempt、実CLI/Studio同一Provider B呼出し、Rule/Relation全必要閉包、実PostgreSQL/実Host、L3、外部Harness機密隔離、Owner意味受理はこのDocs追補ではNOT RUN**。Source/Artifact/Scope/Current Grant/Actual Input/Resultを別証拠として扱い、事後の正常な数え方や同一JSONを理由に過去の実験のClaimを変更しない。

### 27.7 実Selection Matrix・Studio Host入口・T0.5・MCP非必須・L3の実証状態を更新する（2026-11-08、過去の未達記述の訂正・実装着手の接合点）

**後続mainとの照合注記（2026-11-08）**：§27.7の「MCP入口なし」「cmd/tei自身のL3未実施」は当時の観測として保存する。現在はmainにMCP参照Adapter #3403、正式接続対象を採用したADR-0048 #3439、CLI/MCP validate同値の限定実証 #3458、cmd/tei Stage1→Stage2の限定L3 #3440が存在する。ただし製品MCPと実Plugin D1、全Build閉包の資格は成立していない。現在の判定は§27.8を優先する。

本節は[mainにあるPlugin-first計画](PLUGIN_FIRST_TOOLING_RUNTIME_DOGFOOD_2026-11.md)と[Inspector D1 Fit/Gap main #3300](https://github.com/hat47x/tei-platform/pull/3300)の設計正本を変更しない。§27.5/27.6で既に決めた「一つの`inspect`をCLI/Studioで同じSelected Providerから実Callする」を、**今回新しく確認できたGo縦断・実Reference・実CLIテスト**に基づき次の実行順へ更新する。

#### 27.7.1 前回の未確定事項は、どこまで実証済みになったか

| 以前の条件 | 2026-11-08の確認済み変更 | なお同じ成功とは呼べない範囲 |
| --- | --- | --- |
| Capability選択のDB技術名とIDが混同される | [main統合 #3350](https://github.com/hat47x/tei-platform/pull/3350)が`SupportsDatabaseProvider`を使い、`reference/workspace`の宣言12件中「未選択」6→4という**同じ技術意味に沿う分類**へ改善。#3337の旧結果は当時のSource/規則として保持 | [main統合 #3363](https://github.com/hat47x/tei-platform/pull/3363)の6種Binding一括Referenceは24中`selected=22,missing=2,ambiguous=0`。ただし`editor.database_workbench`2件の`missing`は**実Validatorが検査を要求していない対象**。検証ルールの追加が必須とは即断せず、ReferenceとValidationの適用対象差を別に記録。選択成功≠実Provider存在・Host Grant |
| Studio `inspect`のHost-selected Provider入口が無い | [Draft #3371](https://github.com/hat47x/tei-platform/pull/3371)は`RootServices.Inspection`という型付き`InspectionService`を追加し、**Factory非nilなら直接`operations.InspectSubject`へfallbackしない**Go変更と正負試験を提案。main`readonly.go`はなお`RootServices`がCatalog/Validationだけで、Inspectは直接Operation。PR先のGo Sourceの差分を別々に確認 | #3371は**Draft・未main・CI未検証**。Factoryに#951/#1016のactual selected Providerを渡すComposition Binding、CLI側同等Port、A/B実呼出差、Studio/Wails/Consumer実受領は別途必要。Factory nil時のPoC fallbackを正式Plugin Invokeとみなさない |
| Required Rule/Relationが正常なJSONで欠落しても成功し得る | [main統合 #3354](https://github.com/hat47x/tei-platform/pull/3354)は旧#3324/#3326の重複3ファイルを統合し、**11実read-only CLI応答＋32局所Context Controls**（うち3 Rule Required、Focused Canonical、Relation/Source Indexを照合）。固定HEADの専用CIがsuccess | `boundedReference=partial`／`qualifiedTestGeneration=false`のまま。完全なFunction/Business Closure・T1-B未知RuleのAI盲検発見・実外部H3 Tool受領・C1・Owner Meaningは未取得。32件はProduct Mutant Killではない |
| Build Driver≠Self-build、L3未達 | [main統合 #3360](https://github.com/hat47x/tei-platform/pull/3360)で**参照driver自身**のStage1 artifactをStage2 invokerに実起動し、同じchild/args・同じ生成artifactを照合、Reference `l3Claimed:true`。#3362はinvokerの自称Digestと実Process Imageの独立照合を追加 | **`cmd/tei`が自分自身を生成したL3ではない**。Go seed、Module供給Trust、Target/Host、D1選択Providerの実Link/Callは別のClaim。既存#2831が引き続きOwner |
| 外部AI/CLI/MCPの比較入口が共通 | [Open #3356](https://github.com/hat47x/tei-platform/pull/3356)の第2ラウンドは実CLI成果の再読取と、**実TEI MCP server入口は現mainに未存在**とのSource調査を記録。CLI既存19広告read-onlyコマンドは有る | 既存ADRが開発用MCP Serverを必須としていない以上、**MCPが無いことをInspector D1の先行失敗や別Core実装の要求にしない**。MCP Tool受領、H3/Consumer・実OS隔離は実施されていない |
| Recorder/Stage失敗は効果0として返し得る | [Draft #3328](https://github.com/hat47x/tei-platform/pull/3328)で`COMMAND_RECORD_FAILED`／Lifecycle失敗への`EffectUnknown`伝播と`MayReplay`の曖昧再送拒否を実コードで提案 | **mainの`invoke.go`はまだ`COMMAND_RECORD_FAILED`に`EffectUnknown`を付けていない**（今回Source照合）。#3328未統合で、実PG/別Connection/Journal crash/外部Receiptによる実効果確認なし。read-only Inspector内の結果とは別に、後のwrite-enabled Provider採用の停止線とする |

#### 27.7.2 次の実装を1段に限定する：二種類の「選択されている」を分ける

**先に使える現物**：#3363の6形式`Binding`一括結果と#3354のRequired Context Gate。両者はread-onlyに局所照合できるが、前者が「`selected`」と出しても後者が「`partial`」と出しても**互いの判断を上書きしない**。`editor.database_workbench`のReference未選択・Validator非対象という差は、「missing/ambiguous/unknown/not-checked」をアプリの実Plugin機能失敗へ短絡させない具体例。既存`Supports*`の再利用を優先し、新しい能力対応表・二重Resolverを作らない。

**先に実装する1辺**：#3371のHost-selected`InspectionService`の正負試験を独立に確定させ、非nil Factoryから別の直接Operationに逃げないことを確認する。その次に既存#951/#1016でexact`authoring` Contract/Provider Bを選択し、CLIとStudio**各独立Caller**へ同じ実Providerを渡す。二つのCallerから`Inspect`を呼んだ事実（Target/Contract/Source Revision/Result byte）を同じAttemptに束縛し、**B選択→A実行で結果だけ一致**という負例を拒否する。これは選択Scopeを旧`application-runtime` Carrierへ偽装することもStudio RendererにRoot/Provider選択権を与えることも要求しない。

**TestとBuildは邪魔をしない独立証拠**：#3354の11 CLI/32 Controlsは`inspect`に必要なSource/Rule/Relationの一部だけを証明し、CLI/StudioのPlugin実Call完了を要求しない。#3360のReference L3は`cmd/tei`セルフビルドの前段計測であり、最初のInspector D1をL3完成まで保留する条件にしない。#3328の書込失敗時`unknown`は実Effectを始める次の段階で必要であり、今のread-only Inspectorを外部DB commit fixtureに変える理由ではない。

| 限定対照 | 実機で見る状態／独立期待値 | 必要なOwner・実施状況 |
| --- | --- | --- |
| **PATH-01** 同一Worldの6 Binding選択 | Capabilityを持つ、環境が選ぶ、ValidatorがそのBindingを検査する、実Invokeの四つを別に報告。`not-checked`は不合格でも合格でもない。 `editor.database_workbench`の2件でReferenceの`missing`とValidatorの非対象を共存させる | #951/#1016/#3279。#3363 main実Referenceあり、Product InspectorとのJoinはNOT RUN |
| **PATH-02** Host Factoryの正負対照 | Factory非nil・Inspection未供給なら旧Operationにfallbackせず拒否。FactoryがBを供給すればBを1回呼ぶ。非許可RootではFactoryは0回。Subject/Effect/Schema/FP不一致は非公開の内部原因を隠して拒否 | #3371はGo変更Draft、現main未接続。exact-head Go/CI・実Wailsは未確認 |
| **PATH-03** CLI×Studio同一Provider | `authoring` ScopeのBを選んで両実Callerからactual B Invokeし、選択B／実Aでは失敗。両者の世界/Source/Required Context/Disclosureはそれぞれ別の証拠 | #1251/#951/#1158/#1824、**新しいE2E NOT RUN** |
| **PATH-04** ContextとTransportの不完全性 | 同じJSON/FPでも必須Rule/Relation欠落を検出。Sourceの未読込、Result途中切断、Tool独立受領を混同せず`partial`で停止。MCP未実装を全利用不能と扱わずCLI正例は保持 | #3262/#3354実CLI限定PASS。外部MCP/H3と全Required Closureは未実証 |
| **PATH-05** Build・事後効果 | 参照Stage1→Stage2 `l3Claimed:true`と、製品`cmd/tei` L3未実施、実Inspector D1 Link未実施を別に保持。Recorder/Lifecycle失敗後の再送は`unknown`として既存#3328/#1174で検証 | #2831の参照L3限定PASS、#3328 Draft／実PG効果NOT RUN |

**次の優先順**：PATH-01の既存read-only照合→PATH-02のStudio Host Service正負対照→PATH-03のCLI二Consumer同一Selected Providerのactual Invoke→PATH-04 Required Source/Ruleを拡張→必要な段階でのみPATH-05の実Build/Effect資格。MCPが必要となるConsumerを特定するまでは新MCP Serverや第二Tool Registryを実装順の前提にしない。Provider/ToolのOS隔離・機密情報送出と保持の資格は既存#2515／TAI #232で別検証する。

**実装・資格状態**：main統合は#3350/#3354/#3360/#3362/#3363の限定Reference／局所Test。Studio Host入口#3371、Command効果不明#3328、MCP第2ラウンドEvidence #3356は各Open/Draft。今回の§27.7は**docs-only**で、Studio/CLI共通実Provider・OS/外部Harness送出拒否・MCP実入口・`cmd/tei` L3・実PG/ReceiptとOwner意味受理を新たに実行したものではない。#1119のGates／ADR-0035のA3/A4を変更せず、第二Host/Resolver/Registry/Scenario正本を導入しない。

### 27.8 共通API採用後のPlugin実呼出しと、Imported Provider原文のSource-only失効を独立させる（2026-11-08、現行mainと他チャットの成熟差分）

**参照正本**：Plugin-first実装順は[main Plugin-first計画](PLUGIN_FIRST_TOOLING_RUNTIME_DOGFOOD_2026-11.md)／[統合#3384](https://github.com/hat47x/tei-platform/pull/3384)、一機能の横断順は[main #3481](https://github.com/hat47x/tei-platform/pull/3481)、共通APIは[採用ADR-0048 #3439](https://github.com/hat47x/tei-platform/pull/3439)、Import原文の失効は[外部参照設計](EXTERNAL_SEMANTIC_ACCEPTANCE_HANDOFF_2026-10.md)／[Draft #3479](https://github.com/hat47x/tei-platform/pull/3479)。本書はDSH由来の反例とPlugin実使用の資格境界だけを所有する。

#### 27.8.1 旧判定の更新と主張の上限

| 対象と現物 | 成立した限定事実 | なお認定できないこと |
| --- | --- | --- |
| [MCP参照Adapter #3403](https://github.com/hat47x/tei-platform/pull/3403)／[ADR-0048 #3439](https://github.com/hat47x/tei-platform/pull/3439)／[CLI/MCP validate #3458](https://github.com/hat47x/tei-platform/pull/3458)（main） | 4 read-only操作を実行するstdio MCP参照入口が存在し、validateでは共通投影を使って正常・読み込み失敗・意味エラーの限定CLI/MCP同値をGoで確認。ADR-0048はCLI/MCP/API/Web/Desktopの共通アプリケーションAPIを採用。ローカルHTTP Hostは必須でない | 正式製品MCP全Protocol／実外部H3は未認定。Transport同値・共通API応答一致は、選択されたPlugin Providerの実Call、現在Grant、業務Effectの証拠にはならない |
| [Source Index版確認 #3477](https://github.com/hat47x/tei-platform/pull/3477)／[Function独立Required #3482](https://github.com/hat47x/tei-platform/pull/3482)（main） | 同一WorkspaceではWorldが同値でもSource Indexだけ変わる原文差を実CLIで判別。sales.order.submitのCapability・Application・Source限定Requiredは、13 CLI Reply／42局所Controlのmain-push CIで成功 | Function宣言が必要範囲で観測済みでもFunction→Query→実Provider Read/Host/Effectは未証明。C1、全Function/Business Closure、Owner受理は別 |
| [Imported Provider Source-only負例 #3479](https://github.com/hat47x/tei-platform/pull/3479)（Draft） | 別ProviderのImport原文のみが改訂され、ConsumerのWorld・Source Index・Model Revisionは同値のままになり得るというGo反証を追加。Provider原文のSHAと現在参照位置の指紋は別に変わる想定 | focused Go Testとexact-head CIの合否は本節では未認定。Consumer側FP一致からProvider Sourceの現在性、同一Load byte、選択Impl・BOM・現在実行資格を証明しない |
| [cmd/tei自身の段階Build #3440](https://github.com/hat47x/tei-platform/pull/3440)／[self-build入口 #3463](https://github.com/hat47x/tei-platform/pull/3463)（main） | Stage1生成のcmd/tei実行物がStage2のBuildを駆動する限定L3を実測。旧§27.7の「参照Driverのみ」は歴史状態となった | Goは外部seed、Source Closureはrejected、Build Intentの解決・現在受理／Grant・独立二試行は未成立。入口のqualified:falseは維持。実Plugin BのGo Link/InvokeやVerified BOM 2×2とは別 |

#### 27.8.2 Consumer版とProvider原文版は異なる鮮度軸

最初のread-only Inspectorは、**ConsumerのWorld／Source Index／Model Revision**と、実際に参照・選択された**外部ProviderのSource occurrence／manifest／原文byte／Revision**を別の依存として報告する。Consumer側の指紋が同じでも、Provider Rawが変更されていれば、元のSource引用が現在でも正しいとは限らない。逆に改行だけのRaw変更で業務意味が必ず変わったと断定もしない。

用途ごとの再利用判断は三つに分ける。意味計算の再利用候補、Source引用・Inspector実入出力の鮮度、Build/Plugin BOM・現在Grantの資格である。必要なProvider側の実Loadまたは取得byteの証拠が無ければ、そのClaimだけをunknown／unqualifiedに止める。すべてを一律に再Build、またはConsumerのWorldFP一致を理由にすべて再利用、という両極端を採らない。

| 既存Ownerへ渡す最小反証 | 独立Expectedと停止条件 |
| --- | --- |
| **CRS-01 共通APIとProvider呼出し** | 同じWorkspaceの4 read-only操作をCLIとMCP参照で比較し、入口別の成功・検証不合格・失敗符号を照合する。**別証拠**としてauthoring Provider Bを選択した各CLI/Studio CallerがBをactual Invokeした事実を確認。単に同じJSONをBuiltinから返すならD1未達。#3419/#3420/#1251/#1824 |
| **CRS-02 Consumer内Source-only** | #3477／#3482の実CLIでWorld同値・SourceIndex変更を対照。旧Design Context／Scenarioの引用資格を停止するが、元byteへ戻るA→B→Aで中間変更が不存在と推定しない。#3262/#2514 |
| **CRS-03 外部Import Source-only** | #3479の別Provider Rawを改訂し、Consumer側両FP不変でもProvider引用がstale/未観測になることを別に検査。Provider実取得byte／同一Loadが確認できるまで実Plugin選択・Build/BOM/Source引用を無条件にqualifiedとしない。#1570/#1017/#1251/#3267 |
| **CRS-04 L3とPlugin実収録** | cmd/tei L3の限定正例とSource Closure rejectedを保持。選択Provider Bの実Go Link/Callを独立試験し、Self-buildをD1やRuntime Grantの合格に流用しない。#2831/#1251/#3267 |
| **CRS-05 外部Harnessの広告・送出** | [main #3444](https://github.com/hat47x/tei-platform/pull/3444)が測る広告Tool名は実Tool呼出しではない。[main #3451](https://github.com/hat47x/tei-platform/pull/3451)／[Open #3470](https://github.com/hat47x/tei-platform/pull/3470)のLocal MockではSession LogがmaxBytes（**生byte**）で分割され、過小値ではfieldなしでもSession completedになる。Session Log非送出だけで暗黙Context・モデル入力・別Sinkへの非送出を証明しない。実OS拒否・実Provider保管は未実証。#2509/#2515／TAI #232 |

**実装順**：既存#3384／#3300の単一read-only Inspector D1aから、exact Provider B選択→CLI・Studio実B呼出しを先に成立させる。MCP/HTTP/Studioの共通API投影を新しいPlugin Resolverとして実装しない。次に一つの外部Import Providerの原文証拠を同一Attemptへ取り込み、Consumer版とProvider版の二軸を照合。その後Build／Trait／BOM／現在Grantを別々に受け入れる。外部Harnessは[MASTER_PLANの増分効果検証案 #3487](https://github.com/hat47x/tei-platform/pull/3487)のB0=決定的CLI/API、B1=既存HarnessのSkill比較、B2=薄いAdapterを用い、実測されない独自Harnessの先行構築を要求しない。

**現在の状態**：#3403/#3439/#3458/#3477/#3482/#3440/#3463/#3444/#3451はmainの限定実績。#3479/#3470/#3487はOpen/Draftの別Owner結果であり、承認済み成果と呼ばない。本節はdocs-only。実D1 Provider Bの二Consumer呼出し、Import Sourceの同一Load完全閉包、製品MCP全適合、H3実配送・実OS/Egress拒否、実BOM 2×2、Host実Grant、Owner意味受理は今回NOT RUN。第二Runtime／Host／Registry／Resolver／Scenario正本は追加しない。

### 27.9 局所Selected Callから通常CLI・Studioへ：三つの独立した鮮度軸とD1の最後の接続（2026-11-08、他チャット・最新Go実Sourceに基づく狭い統合）

**正本の変更なし**：実装順は[main統合Plugin-first計画#3384](https://github.com/hat47x/tei-platform/pull/3384)と[MASTER_PLAN S0〜S3 Draft #3499](https://github.com/hat47x/tei-platform/pull/3499)、共通APIはADR-0048、Sourceは#2743/#3479、BOMは#3267/[#3490](https://github.com/hat47x/tei-platform/pull/3490)。DSH本書は外部Harnessと実Provider資格を誤接続しない受入差分だけを引き継ぐ。

**新しく実装されたS1局所Call**：[CLI Draft #3381](https://github.com/hat47x/tei-platform/pull/3381)の実Source internal/inspectoradapter/selected.go は、既存Composition Registry.Select→NewSelectedCoordinator→Host登録Registration.Inspectを通し、Scope=authoring-tool、exact Contract/ProviderID/ArtifactDigest、呼出直前のRevision/Host Guardと結果を検査する。現Go TestはB選択→B Go関数を実呼出し、B選択→A登録の拒否、失効後拒否を区別する。**登録したArtifactDigestはHost側の申告値で、実Go Linkされたバイナリbyteの独立証明ではない**。Go局所呼出しはS1候補であって、通常CLIのmain.goからの実B Invokeをまだ示さない。最新HEADのexact CIは本節で未確認。

**Studio S0の実装経過**：[Draft #3371](https://github.com/hat47x/tei-platform/pull/3371)はFactoryが非nilならInspector欠落・誤応答をfail closedに変更。旧TestがBuiltinフォールバックを期待した衝突を修正後、実generated HostにInspectorが未登録で失敗する不備を検出し、cmd/tei-studio/host_generated.goにHost選択の暫定selectedBuiltinInspectorを**明示登録**した。このBindingはBuiltin operations.InspectSubjectの橋渡しであり、Plugin B実選択ではない。旧CIの失敗を現HEADの成功に読み替えない。[Draft #3378](https://github.com/hat47x/tei-platform/pull/3378)のCLI/Studio直接Operation同値はD1a基線であってD1bではない。

**三種類の鮮度は互換でない**：(A) Consumer World／Managed SourceIndex・Model Revision（main #3477）、(B) Imported Providerのraw Source とManifest各Revision（[Draft #3479](https://github.com/hat47x/tei-platform/pull/3479)）、(C) Capabilityを選択した**RuleFingerprint・適用Scope**（[Draft #3490](https://github.com/hat47x/tei-platform/pull/3490)）。改訂Scopeが異なればWorld FP同値でもSource引用、実Build入力、Provider現在資格を自動再利用しない。逆にRaw改訂だけで意味再計算・全Artifact再Buildを必須とせず、用途に必要な依存だけ失効させる。#3337の12宣言／6未選択と#3350の未選択4は規則の違う歴史Evidenceである。

| 実装受入に渡す次の一因子反証 | 独立Expectedと資格上限 |
| --- | --- |
| **S2-01 通常CLI** | cmd/tei/main.goの通常dispatchでexact authoring Provider Bが実Invokeされ、そのGo CallがCount/Identityで観測できる。局所inspectWorkspaceJSONWithServiceを直接呼ぶだけなら未達。B選択→A呼出、明示B不足→Builtin黙示fallbackは拒否 |
| **S2-02 通常Studio** | generated Hostの暫定Builtin登録をexact選択Bへ明示交換し、実Studio BridgeでB実Invoke。Service欠落・別Subject/Schema/Effect/Result FP・root置換・後着ABAは拒否。Factory Go TestだけでWails配布資格としない |
| **S2-03 三軸失効** | Consumer FP一定でProvider Rawだけ、Manifestだけ、次にRuleFingerprintだけ改訂。必要Source/Manifestの実Load byte・Provider Entry・選択規則を同一Attemptで確認できなければ、該当する引用/Build/BOMだけunqualifiedとする。旧Registrationの自称Digest一致は十分でない |
| **S2-04 共通APIと外部Harness** | ADR-0048のCLI/MCP validate限定同値を実Plugin B Callと分ける。Tool広告≠実Invoke、CLI read-only≠外部Session Logや別Sinkへの送出無し。実Host/OS Read Deny、Consumer Disclosure、Tool完全受領を#2509/#2515に委譲 |

**最新の証拠上限**：[main統合 #3487](https://github.com/hat47x/tei-platform/pull/3487)は既存Harness・薄いAdapter・独自Harnessの増分効果比較を計画化したが比較試験完了ではない。#3479の旧HEADはSource-only Go単体[Health #42461871730](https://github.com/hat47x/tei-platform/actions/runs/42461871730)成功だが、新たなManifest-only／Source×Manifest 2×2を加えた後のHEADはその成功に便乗できず、現HEAD CI合否は本節で未認定。#3490のRuleFingerprint/Imported Source/BOM資格もDraft。D1b通常CLI/Studio二Consumer・実Go Link／Provider Source同一Load閉包・Qualified 2×2 BOM・実H2/H3 OS拒否・Owner業務意味受理は今回NOT RUN。

**次の一件を固定**：S1局所Go試験のexact HEADを確認し、S0 Studio明示Builtinの互換/fail-closedを確認してから、**S2-01の通常CLI→S2-02の実Studio Bridge**で同じSelected B実Callを成立させる。S3のProvider Source/Manifest、RuleFingerprint、実Artifact、Trait/BOM、現在Host GrantはそれぞれのOwnerと独立Expectedで照合。旧application-runtime CarrierのScopeを便宜的に拡張せず、第二Resolver/Host/Registry/Runner/Scenario SSOTを追加しない。

#### 27.9.1 D1a互換Baseline・非信頼Error・原子的Traitの独立二Consumer（2026-11-08、main統合済みの成熟差分と現Draft実Sourceから最短工程を補正）

**既存D1aの証拠は確定したがD1bではない**：[main統合 #3378](https://github.com/hat47x/tei-platform/pull/3378) は既存CLIとStudioの**直接Builtin Operation経路**を、Data／Rule／Relation／Semantic Capability／Application／Patternの計6 Subjectで個別呼出しして比較。Canonical・Source Anchor・World/SourceIndex FP・Effect・NotFoundを検査し、exact PR HEAD [Health #42463508044](https://github.com/hat47x/tei-platform/actions/runs/42463508044) はsuccess。これをD1aの固定互換Baselineとするが、同じ結果を返すだけではSelected Provider Bを実Callしたことにならない。D1bの正負Expectedは既存#1180/#2514へ委譲する。

**Traitの共通性は証拠形式に限定**：[main統合 #3502](https://github.com/hat47x/tei-platform/pull/3502) はAtomic Traitを**T=開発環境Inspector**と**R=業務Function Reader**の二つの独立したConsumerで検証する計画を既存Trait設計へ統合した。共通にするのはDefinition／Binding／Implementation／Invocation／Conformanceの証拠形であり、Provider Instance／Target／Build／Source／Current Grantや必要Traitの集合ではない。Rの具体縦断は `pattern order_entry.recent_orders browse` → `query order_entry.recent_orders` → `recent_orders.sql (:from)` → `datasource sales` → `application order_entry` の `sales_core` Alias →実Provider Rowset→Function Context。`sales.order.submit`は別のSemantic Capabilityであり、BrowseにMutation／Transaction Traitを暗黙に与えない。Rの実Reader/DB/0件/partialとAuthorityは#2586/#3270へ引き継ぐ。D1 Tの合格からRを合格にしない。

**Host-selected Serviceの型付きErrorは非信頼入力**：[Studio Draft #3371](https://github.com/hat47x/tei-platform/pull/3371) で、Providerが `operations.InspectionError` のCodeだけでなくMessageも任意に作れるため、一般的な `errors.New` の匿名化では秘密のPath/Token/内部Causeが漏れ得ることが判明。現Draftの `selectedInspectionFailure` は既知の4 Codeを固定公開Messageへ変換し、未知Codeを `OPERATION_FAILED` に縮退するGo Sourceと5負例を追加している。これは非nil Factoryでのfail-closed／暫定Builtin明示登録の追加措置。旧Builtinの表示契約は変更しない。最新HEADのCI/実Wailsから見た非開示成功をこのSource記載だけで認定しない。

| 最小受入 | 独立Expected・停止線 |
| --- | --- |
| **S2-05 D1a→D1b** | main #3378の6 SubjectとNotFoundを互換Baselineとして保持。通常CLIと実Studio BridgeがBを実Callして同等Resultを出すか**実Call site**で判定。Bを選んでA実Call（結果同一）は不合格。Source完全閉包・現在Grantは別 |
| **S2-06 型付きErrorのClient開示** | Host選択ProviderからPublic Code＋秘密Message、Private Code＋秘密Message、Generic Errorを返し、既知安定Code／固定文以外のProvider Code/Message/CauseがClientへ出ないことを確認。CLIとStudioで別Consumer実受領を計測 |
| **S2-07 Trait二用途** | T=Inspector read-only、R=Function BrowseのRequired Traitを別々に評価。Manual Requiredの未履行・Provider未選択・Readerの0件／partialを別判定。BrowseへMutationを自動継承しない。Rの実DB・Function Contextは既存#2586の独立Attemptで扱う |

**実装順の修正**：D1aの同一Result比較を再作成せず、#3381のS1局所Selected Go Call、#3371のStudio ServiceとError保護のexact HEADを検証してから、通常CLI dispatch→実Studio Bridgeへ同一Bを接続する。その一件が終わる前にTrait R側実DBや正式BOMをD1のHard dependencyへ追加しない。#3502は設計main統合でありRの実Provider Readと実Trait履行の完了ではない。今回の文書更新はdocs-onlyで、S2-05〜07の新しい製品統合試験・Owner Accepted MeaningはNOT RUN。

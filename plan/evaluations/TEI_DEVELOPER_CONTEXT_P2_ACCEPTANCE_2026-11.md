# TEI Developer Context P2: 評価・改善受入計画（2026-11-03）

> **位置づけ**：TEI #2533の実証用作業計画。採用済み製品仕様／一般的なSource・Claim・権限・Evidenceの新正本ではない。
> **実行証拠**：PR [#2669](https://github.com/hat47x/tei-platform/pull/2669) の exact HEAD `875181ad724840002143b8bf737180132414765e`、[Focused Actions #41640569565](https://github.com/hat47x/tei-platform/actions/runs/41640569565) で現行Go実装を用いた**13件のCharacterizationがPASS**。mainの製品実装・Owner意味受理・実モデル効果を示さない。

## 1. 問題と達成目標

対象は `tools/reference-developer-context/main.go` の **Discover**（候補を収集）／**Rerank**（ローカルHTTPで順位を調整）である。main照合時 `0e18b916aa15bc7768e9abfb34318b98ee497cb3`、本体のGit blob `dc69dc3a204085801ca509bdda2e2e27b91fa204`。本体は未改修。

検証する主張を分離する：

- **探索観測**：どのSource/範囲を試み、何を実読取りでき、何を候補Cに残したか。
- **問いへの十分性**：変更意図Qに必要な根拠集合Rが、独立Expectedに照らしてC・実Readの中にあるか。
- **比較の公正性**：D0（既存Reportの説明）、D1（決定的Probe提案）、M1（SEI MethodのProbe提案）に同じ情報・Tool・Grant・予算を与えたか。
- **順位付けの追加価値**：同一CのD_meta対M_metaを比較し、モデル入力／契約／成功試行を分離したか。
- **用途・現在資格**：Model/Tool/Consumerの実受取、開示・実行資格、Source版の再解決を独立に検査したか。

モデルが候補集合に含まれない根拠を順位付けで復元した、又は局所PASSで全Repo・製品全体・Owner意味受理を保証した、と主張しない。

## 2. 現実装の確認済み反例と評価上の位置

| 証拠列 | 独立のGo Characterization | 観測した限界 | 改善／採用の状態 |
| --- | --- | --- | --- |
| T9 | `TestDiscoveryAddingWeakCandidateCanEvictDirectEvidence` | 弱い候補の追加で直接テストが12件枠から落ちる | 改善なし |
| T10 | `TestDiscoveryPositionIsAttemptLocalNotCandidateIdentity` と `TestRerankQuestionC0BindsDifferentCandidatesAcrossAttempts` | `c0`はAttempt局所の位置であり永続Candidate IDではない | 改善なし |
| T11 | `TestDiscoveryDocumentDirectoryMentionSharesCoarseBasis` | 同じ `explicit-path-reference` にexact/file・directory・sibling-onlyが混在 | 改善なし |
| T12 | `TestRerankQuestionContainsUntrustedCandidatePath` | 非信頼のpathをHTTP質問本文に組み込む | 実LLM影響は未観測 |
| T15 | `TestDiscoveryScanBudgetMayHideRelatedDocument` | docs各探索の1,000件予算後に関連Markdownが隠れる | 改善なし |
| T16 | `TestDiscoveryUnreadableDocumentIsNotNoDocument` | 存在するサイズ超過Markdownと不存在が同じReport | 改善なし |
| T18 | `TestRerankModelFailureDoesNotUndoLocalRequestDelivery` | loopback POST受取後503でも `model-failed` と候補fallback | 外部e​​gress/推論は未観測 |
| T7 | `TestDiscoveryTwelveTestsHideDocumentEvenIfDocumentIsAdded` | test12件の枠消費でdocsを未走査のまま同一Report | 実Toolによる追加Probeは未実行 |
| T19-R | `TestRerankReusedReportCarriesPreviousModelOutputIntoNextRequest` | Report再利用・Go浅いcopyで前回Choice/順位が新試行に混入 | 通常CLIの単回呼出しの欠陥とはしない |
| T13-S | `TestDiscoverySameBytesSourceRenameChangesLexicalEvidence` | 同byteのGo source renameでtest候補が消え、旧file名だけのMarkdownはdirectory一致で残存 | 意味同一性・引用の有効性は未認定 |
| T19-C | `TestRerankChoicePriorityDiffersFromLargestProbability` | `choice` が確率最大と異なっても形式上有効となり、クラス順が確率より優先 | Model提供者の意味契約・校正は未確認 |
| T20-A | `TestRerankAssignmentAccountingRetainsFailures` | 割当4試行（未送信拒否／503送信済み／候補なし／成功）のうち`model-ranked`1件 | 実モデルの品質・一般的成功率ではない |

**名称衝突の解消**：#2533の歴史的追補で`LC-21/T19`が **R（Report再利用）** と **C（choice/probability）** の両方に用いられた。過去コメントの意味を上書きせず、本計画・実行票では`T19-R`／`T19-C`と区別する。`T20-A`は割当分母を維持する合成対照。上表は**13個のGo test**を構成し、一つのTestが一つのT番号とは限らない。

現実装`Report`に`visited/scanned/read-error/unknown`やatomic Snapshot保証はない。テストの独立Fixture値、test-side`os.ReadFile`は製品がConsumerへ実送出したProofではない。

## 3. 最小の改善候補（いずれもOwner未採用）

**A. 収集・表示・探索上限を分ける。** 早期12件で収集打切りする現在の方式と、同じ局所Scope内の明示的テスト候補を評価予算内で列挙した後に表示12件へ絞る方式を比較する。全Repository無制限検索は採用しない。`docs`と`plan/adr`の独立1,000件上限も、残余を示す基準を残したまま比較する。

**B. 限定した次Probeを実読取する。** 「docs追加探索」「対象関数に対応する関数経路の確認」「旧引用先の再解決」を、現在の実Tool・root・Scope・引数・Read資格で実現可能な場合だけ実行する。現`reference-developer-context` CLIには探索範囲を追加指定する入口が**ない**ため、SEI提案を製品機能／実Readに昇格させない。既存の許されたread-onlyファイル操作で実行可能性を先に検証する。

**C. 探索診断の最低限の事実を残す。** 試行票に`scope, inspected, read, retained, displayed, skipped-by-cap, failed-read, not-explored`を区別して記録し、`unknown`を誤って`not found`に変換しない。正式Report schema変更の要否・公開面は元コードOwnerが判断し、Issue側の評価票を第二のCore正本にしない。

**D. Rerank比較を試行ごとに隔離する。** 各D_meta/M_meta試行で新規`Discover`を行うか全候補とマップの独立性まで確認したdeep cloneを使用する。各試行のRequest byte/digest、`cN→(path,SHA,basis,rank)`、Provider応答とModel ID/configを保存。現HTTP `state.candidates`には`baselineRank`が含まれるため、`baselineRank`を知って行う再順位と完全独立な関連性発見を混同しない。

**E. Model typed choiceの契約を決めてから扱う。** 現コードは`choice=argmax(probabilities)`を要求しない。Modelがargmaxを保証する契約なら不一致を診断する対照を、保証しないならchoice/scoreの評価語彙を分ける。確率和の形式検査だけで校正や意思決定の受理とはしない。

## 4. 次の実証（まだ実行していない）

| 段階 | 比較条件／独立Expected | 合格・保留基準 |
| --- | --- | --- |
| P2-2a | 同一公開Source・Rを固定し、現行早期打切りと方式A/Bの探索実Read・必要根拠到達・費用を比較 | 必要Rの収集向上を確認。未探索は明記。表示順位改善だけでは合格としない |
| P2-2b | W0/W1：初期Report同一、真のdoc所在は採点側に秘匿。D0/D1/M1に同じTool候補・Grant/Scope/予算を提供 | D1より良いM1の追加Probeと実根拠到達が独立確認されない限りModel既定採用を保留 |
| P2-2c | T8：同一Immutable Snapshot対逐次混在/ABAと旧Source引用解決の試験 | 個別SHA正常だけで共同版をPASSとしない。T13-Sの同byte移動を旧引用の現在資格と混同しない |
| P2-2d | D_meta/M_meta：**同じC・同じQ・同じ版・毎試行fresh Report**、Model input byte固定 | 形式有効、Model契約、Top-k/R適合、CPU/token/遅延/Reviewを別指標で採点 |
| P2-3 | 実SEI Method・実Provider/Tool・許可されたConsumer、TAI Harness継続 | 提案→実Tool→実Read→受信/Delivery→Claim再評価を別Witnessで確認。未RunはUnsupported/Unknown |

**分母**：割当Attempt全件のITT型結果（候補なし、送信前失敗、送信済失敗、timeout、typed不正、正常）と、実受信・適用が確認できたMethod機構の条件付き結果を分ける。`model-ranked`のみを分母とする改善率・成功率は全体効果と呼ばない。T20-Aの4件は合成観測の例であり成功率推定ではない。

**受入停止**：独立OracleにW真値が含まれるとMethodへ漏れる、存在しないCLI flag/Toolを実行可能とする、受信後503を未送信とする、`c0`を永続IDとして再利用する、旧pathの引用を現時点有効とする、Grantなし外部送信、異時点ファイルの同一Snapshot断定、のいずれかを検出したら強いClaimをFAIL/unknownに留める。read-only調査を無条件に停止する趣旨ではない。

## 5. 既存Ownerと採用判断

| 領域 | 正本・Owner | この計画が行わないこと |
| --- | --- | --- |
| 認知探索と比較 | TEI [#2533](https://github.com/hat47x/tei-platform/issues/2533)、PR #2669 | Product本体を無承認で変更しない |
| 独立R/Expected・判別力 | TEI [#2514](https://github.com/hat47x/tei-platform/issues/2514) | 正解をMethodに流すOracle重複実装 |
| Source参照・移動・版 | TEI #2505、[#2743](https://github.com/hat47x/tei-platform/pull/2743) | SHA一致で意味受理や引用再解決を自動確定 |
| Consumer開示・Delivery | TEI [#2516](https://github.com/hat47x/tei-platform/issues/2516)、#2515 | local HTTPやread-onlyの表記をOS非干渉の証拠にしない |
| Cognitive Method | SEI [#767](https://github.com/hat47x/sei-cognition/issues/767)／Draft #770 | SEIにTEI Source正本・Tool資格を複製しない |
| Product Skill | TEI [#2605](https://github.com/hat47x/tei-platform/issues/2605) | `yi-tei-modeling`に本PoCのToolが既存実装済みとしない |
| Harness再開・費用/時間 | TAI [#271](https://github.com/hat47x/tai-pleroma/issues/271) | 共通Host・Scheduler・Canonical Evidence DBを追加しない |

**採用に必要なOwner判断**：①必要Rと対象Scope、②必要な候補収集改善と許容読取費用、③報告上の`unknown`/完了表示、④Model choice/probabilityの契約、⑤実Tool・外部Modelへの開示資格、⑥改善の最低価値と停止条件。

**実施状態**：本計画は実Go 13件の現挙動再現を記録したもの。改善実装、D0/D1/M1のblind比較、T8、実SEI Product接続、実llama.cpp、外部Harness／OS効果、Owner受理は**NOT RUN／NOT ACCEPTED**。単一Goバイナリ基盤と既存TEI/SEI/TAI責務を尊重し、新Host・Registry・共通Proof Store・正式Gateを増設しない。

<!-- tei-devctx-t22-and-test-identity-20261103 -->
## 6. T22：再利用後のモデル失敗と旧判断の有効性（LC-24）

**本節は新しい正式Report APIではなく評価計画**。2026-11-03に既存[PR #2669](https://github.com/hat47x/tei-platform/pull/2669)のbranch `test/reference-developer-context-coverage-20261103`へ`rerank_failed_retry_characterization_test.go`を追加（追加commit `ecc76f7df967395bfbb3ef984f5a5a13275ea107`）。製品`main.go`は変更しない。以前の**13件PASSは歴史的なexact HEADの実測**であり、T22追加後の新HEADに自動転用しない。新Testのfocused CI結果は別途exact HEADで確認して更新する。

### 6.1 試験IDの一意な対応

[#2533の歴史的な番号衝突の整理](https://github.com/hat47x/tei-platform/issues/2533#issuecomment-6604501999)を踏まえ、**評価表の正式な索引**は次のとおり。Go test名／PR/Run／Source SHAを共に記録し、過去コメントの別名を消さない。

| 正規評価索引 | PR試験内の短縮名／過去別名 | 項目 | 実証境界 |
| --- | --- | --- | --- |
| **LC-21/T19-R** | `T19-R`（歴史的`LC-21/T19`） | `Report`再使用・Go浅いcopyによる前回結果混入 | Go実証済み |
| **LC-22/T20-A** | `T20-A`／`T20` | 割当Attempt全件の分母を保持 | **4公開合成Attempt**のGo実証。実モデル効果は未測定 |
| **LC-23/T21** | `T19-C`（歴史的`LC-21/T19`） | choice/確率/表示順位の意味区別 | `Rerank`の形式・表示順をGo実証。校正・Provider契約は未実証 |
| **LC-24/T22** | `T22` | 成功後の失敗・旧ModelChoice残存／Deliveryの別資格 | テストSource追加。**当該HEADの実CI状態は別確認** |

### 6.2 T22の独立反証・採用停止条件

公開`t.TempDir`の同一2候補・固定Source・同じbaseline順序で、(a)成功HTTP200で順位B→Aを作り、(b)同じ`Report`への非loopback拒否、(c)同じ`Report`への**loopbackで受信後503**、(d)新しい`Discover`への同じ503を比較する。

現Goの`Rerank`は失敗時に`Evaluation.State`を書き換えるが、`Candidates`の旧`ModelChoice/Probabilities`・並替え順は消去しない。**失敗状態でも旧候補判断が残る可能性**を、出力と相手の実POST受取Requestで観測する。fresh対照では決定的候補順・空のModelChoiceを確認する。失敗応答によってすでに送信済みのByteを取り消したと主張しない（[#2516 CP-12](https://github.com/hat47x/tei-platform/issues/2516#issuecomment-6600118988)）。

現`main` CLIの`Discover→Rerank`単回実行の障害とは**区別**する。危険は外部評価Harness・Product統合が同一`Report`/sliceを再利用する条件にあり、通常CLIの本体修正・正式fallback要件はOwner決定前に行わない。試験がPASSしても「fallback修正済み」とせず、過去判断と現在試行の資格を分けてConsumerへ表示／伝達できることを別に要求する。

**失敗時の最終表示に関する二つの判断候補（未採用）**：方式F1＝現Attemptのモデル結果は失効としてfreshな決定的順位に復帰、方式F2＝旧判断を歴史的注記として保存しつつ当該Attemptの順位資格を`unknown/stale`と明示。どちらでも外部送信と取得済みObservationを消したことにはしない。両方式は実Consumer/UI/Disclosure／追加費用・復帰性を別評価し、非公開の旧path・候補数・Tool名を無条件に公表しない。

**T22に接続する未実施**：不正typed返答とTimeout、実Model/Provider、OS/Network中間ログ、同一Sourceに対する外部Harness再開、実Consumer開示、T14/T17のblind評価、T8共同Snapshot、独立Rに基づく業務意味評価。#2533（認知比較）／#2514（Oracle）／#2516（Consumer受渡し）／SEI #767（Method）／TAI #271（再資格化）／#2515・TAI #232（実開示・OS）の既存担当を維持する。

<!-- tei-devctx-p2-2a-collection-budget-factor-20261104 -->
## 7. P2-2a：決定的な探索方法の限定比較（2026-11-04、評価専用実装）

[#2533](https://github.com/hat47x/tei-platform/issues/2533)のT7/T9/T15と、第3節Aの「**読取予算・保持候補・表示12件の分離**」を、PR #2669の**評価側だけ**に追加した。 `tools/reference-developer-context/bounded_collection_comparison_test.go` に公開 `t.TempDir` Fixtureの3つのGo試験を作り、専用Workflowで `TestCollectionComparison` も対象とした。現 `Discover/Rerank`、Product CLI、正式 Report schema、Ownerの探索Policyは**変更しない**。

**明示した比較条件**：現行Dは12候補の収集中に打切る。評価用方式Aは対象ファイルと同ディレクトリの `_test.go` を**最大12件／最大64件**で調べ、独立した `docs/` **直下のみ最大12件**を走査してから、**表示上限12件**へ縮約する。優先信号は「直接ファイル名一致test／対象の完全path文字列を含むdoc／stem本文一致test」。これは公開Fixture用の**説明可能な字句上の仮ポリシー**であり、全Markdown・関数依存・業務関連性の正解ではない。元PoCの`plan/adr`やdocs再帰探索をこの限定評価方式が完全再現するものでもない。検証したread scopeと上限を併記する。

| 評価Fixture | 独立Expected／現D | 評価Aで確認する差 | 誤って拡張しない主張 |
| --- | --- | --- | --- |
| **CC-01／追加弱test** | `validation_test.go`は独立R。先行弱test12件により現Dは候補外 | test読取12件のA12では同様に未到達、読取64件予算のA64では13ファイルを実ReadしてRを表示12件内に保持 | 読取費用+1をモデル順位・純粋な関連性改善と誤認しない |
| **CC-02／docs入口独立** | 先行弱test12件で現DはMarkdownを未走査。完全path入りdocは独立R | A12でもtestとdocの**読取予算を分離**したためdocsを実Readし、保持候補13件から12件を選ぶ | 完全pathの字句参照＝最新版仕様や意味受理、としない |
| **CC-03／上限負例** | 65弱test＋後続直接test。対象は独立に存在 | A64でも64件で走査打切り、`TestTruncated=true`、R未保持となる | 上限付き探索の0件／未保持を「不存在・完全」としない |

**採点で分離する変数**：実読取回数（test/docそれぞれ）、読取失敗、保持候補と表示候補、予算到達、必要Rの候補到達、表示12件内のR、増分読取費用。A64のCC-01改善は単純に読取件数増が必要であり、同予算のモデル順位効果と比較しない。CC-02は**独立doc予算＋字句優先の両方が変わる**ため一つの純粋因果効果と主張しない。将来の公平な2×2対照で、表示方式固定／予算差と、予算固定／表示差をさらに切り分ける。

**資格と次の引渡し**：この3件はGoによる実ファイル入出力を使用するが、Go test `t.TempDir`内の合成字句Referenceであり、TEI Productの追加Probe CLI/SEI実Method/外部Harness/権限・Disclosureの実検証ではない。評価側の独立RはMethod入力に渡さず（#2514）、実Consumerへ渡す可否は#2516、実Toolの選択・実行資格は#2605/TAI #271/TAI #232の既存責務とする。新規Probe契約・全Repo Index・Agent Host・共通Registryは作らない。**PR exact HEADのCI結果が確認できるまでは試験PASSを主張しない。**

<!-- tei-devctx-p2-2a-factorial-20261104 -->
## 8. P2-2a CC-04/05：読取予算と表示Policyの2×2独立比較

第7節CC-01〜03の重要な限界は、CC-02で**独立Markdownの取得**と**取得後の表示優先順**を同時に変えていたこと。従来の結果だけではどちらが必要Rを見つける効果を持つか分解できない。

今回、評価専用`tools/reference-developer-context/collection_selection_factorial_test.go`へ以下を追加した（**製品本体・Report schema・CLIは変更しない**）。`TestCollectionFactorialBudgetVersusSelectionPolicy`は単一公開`t.TempDir` Sourceを固定し、**読取test上限12／13（docs上限は両条件とも12）**を比較。各読取上限で一度取得した候補集合Cから、追加Readなしの純粋な表示Projectionだけを二つ生成する。

- **S-test**：完全ファイル名一致test→stem本文一致test→対象完全path字句参照docの順で上位12件を選ぶ。
- **S-doc**：完全ファイル名一致test→対象完全path字句参照doc→stem本文一致testの順で上位12件を選ぶ。
- 同一行（読取予算固定）では**同じC、同じSource byte、同じ読取回数**を保持する。同一列（選抜Policy固定）では**追加1 test Read以外の条件を固定**。独立Expectedは試験構成前に`{validation_test.go, docs/target-reference.md}`の2つのpath witnessとして固定し、これをPolicyへ渡さない。

| 固定世界の評価条件 | S-test（test優先） | S-doc（doc優先） | 実読取（test＋doc） |
| --- | ---: | ---: | ---: |
| **B12**（先行弱test12件で止まる） | Rの表示到達 **0/2** | **1/2** | **12＋1** |
| **B13**（末尾の直接testも読取） | **1/2** | **2/2** | **13＋1** |

本表は公開合成fixtureでの**事前Expected（実測完了までは予測）**。元`Discover`はtest12件でdocs走査自体を省くため、**S-test＝現製品方式**とは称さない。S-test/S-docはいずれもdocsを取得した後の**評価用選抜Policy**である。B12→B13の1件到達向上は追加1 test Readの結果であって、モデルが関連性を見抜いたものではない。S-test→S-docの1件到達向上は同一Cの純粋な選抜差であり、探索能力の増加ではない。相互作用がこのFixtureで無くても一般化しない。

**CC-05（負対照）**：同じtest集合でMarkdownを置かない世界を使い、B12/B13の各行でS-test=S-docとなることを確認する。表示Policyは未取得の必要根拠を新たに生み出さない。B13だけ直接testを拾える条件を追加で確認する。

**指標を分離**：`collect(R∩C)`、`display(R∩V)`、read対象と件数、`TestTruncated`、固定世界での差分、Provider/Model利用なし。テストは同一ホストの限定Source filesystem readを含むが、実Repoの関連性・意味受理・外部ConsumerへのDisclosure、SEI Methodの実Probe性能は測っていない。`validation_test.go`とdocの両方を必要Rに置くことは**評価用の人工Oracle**であり、業務Ownerの受入ではない。

**実施順序・受入条件**：先にexact PR HEADのCIでCC-04/05のGo test成功とソース一致を確認する。次に、Rに曖昧性・費用（追加Read／解析byte）・段階的Unknownを入れた複数世界で外的妥当性を評価。SEI `D0/D1/M1`は同じSource・実Tool・Scope・Grant・同予算・blind R条件で別比較（#2533/#2514/SEI #767）。ここでの機械的成功をmodel ranking改善、実Tool利用、意味/Authority/OS安全へ昇格しない。再試行の旧ModelChoice/Deliveryは第6節と#2516、TAI #271で引継ぐ。

<!-- tei-devctx-p2-ci-health-adr0038-20261104 -->
## 9. 実Go比較のCI健全性と認知受理の既存Ownerへの収束（2026-11-04）

### 9.1 限定実行成功とPR全体の健全性は別
- P2-2aの`CC-01〜05`（計5件）と既存Go Characterization（T22まで14件）は、[PR #2669](https://github.com/hat47x/tei-platform/pull/2669) の**focused** `bounded-discovery`で先に成功していた。一方、[Repo Health Run #41744747754](https://github.com/hat47x/tei-platform/actions/runs/41744747754)の`repository-health`は**failure**であり、その時点のPR全体の健全性をPASSとは呼べない。
- 同runの**実Jobログ**を確認したところ、正規全体健全性の `gofmt -l . is empty` で失敗し、`bounded_collection_comparison_test.go`／`collection_selection_factorial_test.go`／`source_rename_and_scoring_characterization_test.go`の**3ファイルだけ**が`gofmt drift`と表示された。同Jobの後続`go vet ./...`実行やExternal Coordination E1の成功を、初めのformatter failureの解消証拠に置換しない。
- **修正した事実**：PR branch上の該当3ファイルをCIのGo formatterによって整形し、Git blobを順に `131a23c279b6a2bf00281feb7ac9631e910fbefa`、`36b9767a12644faa411b04f758f43a1cd6491dc8`、`1d66d3b673a53629f6fee410265c85b6ffc31328`へ更新した。整形出力検査用の一時Workflowステップは削除し、専用Workflowの本来の `Test(Discovery|Rerank|CollectionComparison|CollectionFactorial)` 実行に復元した。**本節追加時点では新HEADの`repository-health`成功は未確認**。再確認ではこの文章を実証結果に合わせて更新する。
- 今後の受入票は `PR exact HEAD / test source blobs / Focused Go CI / Repository Health / known failing step / main対応`を別欄とする。古いHEADのCIを新HEADに流用しない。Formatter不一致を本体アルゴリズムの失敗と混同しない一方、Focused GO successだけをRelease品質とみなさない。

### 9.2 ADR-0038と認知の受入Owner（Issue/ADR横断レビューのC-7を反映）
- [#2533 C-7](https://github.com/hat47x/tei-platform/issues/2533#issuecomment-6619122938)で、**ADR-0038（提案中）**・歴史的な `#843/#876`・閉鎖済み`#2021`と、現在の#2533との参照が重なっていることを確認した。過去Issueを新しい独立Oracle正本と扱わない。
- **独立R/Expected/判別力の現在のOwnerは[#2514](https://github.com/hat47x/tei-platform/issues/2514)**と明示する。#2533は実評価、SEI [#767](https://github.com/hat47x/sei-cognition/issues/767)は認知Method候補、[#2516](https://github.com/hat47x/tei-platform/issues/2516)はConsumer/Disclosure、TAI [#271](https://github.com/hat47x/tai-pleroma/issues/271)は継続再資格化。これらの成功やモデルの確率はOwnerによる業務意味受理を代行しない。
- **ADR-0038を正式採用するか参照方式の範囲へ留めるかは人のOwner判断**。今回の実Go試験・Role/Skill実装をもってADR受理、Model認知能力の完成、実行Authority付与、新規共通Proof Runtime構築と解釈しない。旧#2196/#2021は閉鎖のまま。

### 9.3 直近の比較で残る別因子
`CC-04/05`の2×2は、固定された人工World内の**字句で識別可能なRの表示到達**のみを測った。次のModel/Method比較では、(1)同じCでの順位、(2)同じ予算での追加Source Read、(3)未探索と権限不足、(4)Source版変更後の再取得、(5)実Consumerへ渡せる情報を分離する。固定RをMethodに公開せず、構造候補から受理済みMeaningへ自動昇格させない。現PoCが追加Probe用CLIを提供していない制約を維持する。

<!-- tei-devctx-question-relative-tripleworld-20261104 -->
## 10. P2-2a CC-06/07：同じ初期Observationでも問いQと隠れた世界Wが違えば必要な根拠は異なる（2026-11-04）

**経緯**：前節CC-04/05の2×2は固定された人工Rのみを評価した。P2の本来の主張は「**問いに必要な根拠を、現在の権限・Source・Scopeで取得できたか**」であり、表示順位の総合的な良さではない。別チャットのSEI #767/#770の不足観測・追加Probe議論と、独立Oracle #2514のsealed Expectedを比較設計に接続する。

### CC-06：同じ `Discover` Reportを返す三世界
- **W0**：Go targetと同一packageに先行弱`_test.go`12件＋後続の直接test。docsに該当文書は置かない。
- **W1**：W0と**全Go byteを同じ**に保ったまま、対象完全pathを字句参照する`docs/target-reference.md`を追加。
- **W2**：W0と同じGoに、対象ではない別pathを字句参照する同名docを追加。
- どのWorldも現`Discover`は弱test12件で候補枠を埋めてdocを走査しないため、**初期Reportは完全一致**。初期Reportしか与えられていないD0/D1/M1は、W0/W1/W2の真値を推測で判定してはいけない。
- 公開合成で評価専用`compareLocalLexicalCollection`がtest13件まで、docs直下最大12件の追加Readを実施し、`TestRead`、`DocRead`、`ReadFailures`と表示した根拠pathを別々に記録する。W0 doc読取0、W1/W2 doc読取1。**読取が1件あっても、W2の文書が現在のtargetを明示参照した証拠にはならない**。

独立Expected `R(Q)`を、`Q-test-change={validation_test.go}`、`Q-citation-check={docs/target-reference.md（対象の完全path字句参照に限る）}`、`Q-test-and-doc={双方}`として試験側で凍結する。Rは候補選抜・仮モデル・Probeの入力に渡さない。評価用B13・S-docによるpath witness到達の**予定判別値**は次のとおり。

| 世界（固定した候補・実Read） | Q-test-change | Q-citation-check | Q-test-and-doc |
| --- | ---: | ---: | ---: |
| W0：doc無 | **1/1** | **0/1** | **1/2** |
| W1：対象path文書有 | **1/1** | **1/1** | **2/2** |
| W2：別path文書有 | **1/1** | **0/1** | **1/2** |

この表は「**試験の人工path-oracleへの到達**」であり、実業務ドキュメントの意味・正式引用・Source現在性の受理票ではない。特にW0/W2の0/1は必要文書を取得・示せないことを意味するのみで、全Repo／別Scopeの不存在の証明ではない。

### CC-07：同一Cと同一表示VでもQごとの十分性判定は異なる
- W1を固定し、test13件＋doc1件を実Readして同一C（14候補）を確定。S-test表示12件では直接testは残り、docは落ちる。
- **同じC、同じV、同じread件数**でも、Q-test-changeは1/1、Q-citation-checkは0/1、Q-test-and-docは1/2。従って`model-ranked`、候補件数、Top-kだけから「根拠が十分」を主張しない。
- 固定`C`/Q相対`R`/表示`V`を分け、`R⊆C`かと`R⊆V`かを混同しない。もしRがCにあるがVに無い場合は**選抜不足**、RがCにも無い場合は**取得・探索不足／現在未確認**として残す（強い不存在は別証拠が必要）。

**実装と実証の範囲**：PR #2669の`tools/reference-developer-context/question_relative_worlds_test.go`に`TestQuestionRelativeSameReportDifferentHiddenWorlds`と`TestQuestionRelativeSameCollectedCandidatesDifferentQuestions`を追加、Focused CI対象に`TestQuestionRelative`を含めた。製品本体・Reporter・Source正本・新Toolは不変更。**最新exact HEAD CI成功とformat healthが確認されるまでは本2件PASSと記録しない。**

**次の独立比較**：外部SEI Method評価は`(Q,初期Report,候補元の現在Source版,実Tool,予算,Grant,Audience)`をD1/M1に対称に渡す。採点側の`W0/W1/W2`正解・選抜正解Rをモデルから隠す。現Developer Context CLIには任意probe範囲を指定する公開フラグはないため、想定ツールを実存在Toolと取り違えない。実行がなく追加Probe提案だけなら`proposed/unsupported`で止める。取得後の現在性・Consumer別開示は#2516、Source/Owner意味は既存責務、Attempt再資格化はTAI #271、Method側の不足判断はSEI #767とする。

<!-- tei-devctx-p2-next-probe-actual-local-read-20261104 -->
## 11. P2-2b 入口：D0と決定的D1の「提案→実Read→現在観測」を分離する（2026-11-04）

**位置**：[#2533](https://github.com/hat47x/tei-platform/issues/2533)で以前から構想したD0/D1/SEI M1のblind比較のうち、**実行可能な前段D0対D1だけを評価専用Go実装**へ進める。D0＝現`Discover`、D1＝test専用`localDocsProbe`が**既知のScope `docs/*.md`直下**で明示予算1件の追加ファイルReadを行い、`path + SHA256 + byte count + exact target path文字列一致 + read attempt/result + budget exhausted`をその場で観測する。D1は入力に`target`（利用者のQ側で公開された既知対象）と`root`・予算・許可の合成スイッチのみを受け取り、**未知文書名や独立Expectedを受け取らない**。これは認知Methodではなく、実際のGoファイル読取を備えた**評価用の決定的対照**。SEI M1は依然として**NOT RUN**、新しい製品Tool/CLI/Authorityではない。

### CC-08：同じ初期Reportの4世界を固定した追加読取対照
| World | 初期D0 | D1（追加Read予算1件）の観測 | 評価上の禁止推論 |
| --- | --- | --- | --- |
| W0 docs無し | test12件で同一Report | `scope-missing`・追加file Read 0 | 全Repoに文書がない、とは言えない |
| W1 target明示参照doc1件 | 同一 | 実Read1件、Byte/SHAと対象完全path字句一致 | 内容がOwner受理済み／最新文書とは言えない |
| W2 別source参照doc1件 | 同一 | 実Read1件、取得は成功・字句一致なし | Read成功＝必要な根拠の取得ではない |
| W3 別path参照doc→対象docの順で2件 | 同一 | 最初の1件をRead、**予算打切り**・対象根拠未到達 | 未到達＝存在しない、とは言えない |

評価用`allow=false`では、提案はあっても `unsupported`、`attempts=0`、`reads=0`とし、**提案／利用可能性／実行済み／根拠取得を別票**で扱う。これは実Hostの許可処理ではなく、反例を作るための人工分岐。Goの`readInside`経由で読んだバイトのdigestをテスト側で独立`os.ReadFile`と照合するが、後者は**採点側のFixture Witness**であり、D1の実行に渡した隠し知識ではない。

### CC-09：同じpathでSourceが変化した後の旧Receiptの扱い
W1で`docs/reference.md`をReadして完全path一致を観測後、**文書byteを別path記述へ変更**し再取得。現ReadのSHA・適合が変わることを検証し、旧Receiptを現在の根拠に昇格させない。さらにA→B→Aで元byteへ復帰するとSHAが等しくなるが、**変更履歴無し／意味同一／Audience開示可／正しい共同Snapshot**を証明しない。これは単一ローカルfileの限定実観測であり、TEI ModelRevisionやProvider/Consumer受理の代用ではない（#2505／#2516／TAI #271）。

**今回追加する実行入口**：PR #2669の`tools/reference-developer-context/actual_probe_receipt_test.go`に`TestActualProbeD0VersusBoundedD1AndUnsupported`／`TestActualProbeWitnessMustBeReacquiredAfterSourceRevision`を追加し、focused workflowに`TestActualProbe`を組み込む。**exact HEADのCIが完成するまでは新規2件のPASSを主張しない**。出力の状態`unsupported/scope-missing/observed`は**テストローカルな表現**で、正式なTEI Report schemaやSEI APIを設計しない。

**効果比較の限定**：D0とD1は**読取費用も情報量も異なる**ので、D1の根拠到達向上をModel順位効果／同予算Method寄与と混同しない。比較では`initial report + actually acquired bytes + attempted/rejected/skipped scope + read budget + Source asOf`を区別する。真にM1を評価する段階では、D1/M1へ同じTool/Grant/Scope/情報/予算を与えて選択をblind比較し、実際に読まれていない候補を`found`として採点しない。

**受理・Authorityの停止線**：現在のPoCは任意の追加Probe用CLI Flagを提供しない。test-only`localDocsProbe`を製品Toolや実Host権限の証拠に変換しない。D1の目的は、Methodが本当に追加Sourceへ到達したかを将来評価するための**実読取と人工Oracleの分離**である。#2514独立Oracle、#2533評価、SEI #767次Probe、TAI #271現在資格、#2516開示がOwner。新Runtime Host/Registry/共通Evidence正本/正式Gate/ADR承認はなし。

<!-- tei-devctx-p2b-blind-location-bound-20261104 -->
## 12. P2-2b：初期情報が識別不能な二世界では、固定予算1回の次Probe選択に上限がある

**新しい主張**：D1とSEI M1の比較は、追加Readによる情報獲得とMethodの「見通しのよさ」を混ぜてはならない。初期Observationが完全に同じ二世界に、正解の根拠が互いに異なるScopeへ配置されている場合、**同じ初期入力だけで単一Scopeを選ぶ決定的方法**は両世界を同時には当てられない。これはモデルの性能批判ではなく、情報不足の論理的制約である。実M1を呼ばずにこの制約を公開合成Goで確認する。

### CC-10：隠された根拠所在の対称な二世界
- 固定されたQは「変更対象Goファイルを明示参照している文書を探す」。`Discover`に渡すtargetとGo source/test全byteは同一で、先行弱`_test.go`12件が候補枠を埋めるため、**初期Reportは完全一致**。
- W-D：`docs/candidate.md`に対象完全path、`plan/adr/candidate.md`に他の同長path。W-A：両方の文書内容を入れ替える。両世界に**同じファイルpathと件数**が存在し、許されたProbe選択肢は`docs`または`plan/adr`のどちらか1つ。変えるのは隠れた文書内容のみで、Sourceの版・Read取得数・Budgetは等しい。
- `readOneFixedScope`は**評価専用**のread-only関数。選択済みScope内のMarkdownを`readInside`で実際に1件読み、Path/SHA256/ByteCount/完全path字句一致とAttempt数を出す。**正解Scopeや世界IDは引数にしない**。現Developer Context本体・正式Tool/API/Host認可は変更しない。

| 事前に固定したProbe（追加Read予算1） | W-D：根拠はdocs | W-A：根拠はplan/adr | 2世界の評価成績 |
| --- | --- | --- | --- |
| `docs`のみ | 到達 | 未到達 | **1/2** |
| `plan/adr`のみ | 未到達 | 到達 | **1/2** |

この**1/2は人工的に均等重み付けした二世界の一致数**であり、実Repoにおける成功率・モデル評価スコア・一般的な確率ではない。実SEI M1が同じ初期情報でどちらのScopeを選んでも、正解を事前に知る別情報なしに**両世界で必ず到達する保証**はできない。ランダム選択・文書名分布・許可Scope等、別の情報や戦略がある場合はその条件を公開し、独立に採点する。

### CC-11：実Readをもう1回使う適応的な対照とTool未許可の負例
- 最初に`docs`を1件実Readし、対象字句一致しなかった場合のみ`plan/adr`を追加で1件Readする。W-Dでは**1 Read**、W-Aでは**2 Read**で必要Rへ到達。二世界合計は**3 Read**、1回固定選択の二世界合計2 Readより1 Read多い。
- この改善は**予算を増やしたことによる追加情報獲得**であり、モデルM1の「同予算での純粋な追加価値」ではない。Model使用のtoken/latency/送信費、read失敗、Audience/Host資格をさらに付けなければ実用比較にならない。
- 合成`enabled=false`では`unsupported`、実Read0、証拠SHAなしとして止める。人工的な可用性分岐は**Hostの実Grant/OS隔離を証明しない**。Tool Roster・Scope・予算・DisclosureはMethod外部で与える。

**実装と判定**：PR #2669の`tools/reference-developer-context/blind_probe_scope_comparison_test.go`へ`TestBlindProbeSameReportCompetingScopesOneReadBound`と`TestBlindProbeAdditionalReadResolvesMissAtHigherCost`を追加し、Focused workflowの`TestBlindProbe`対象とする。**当該exact HEADのGo CI結果が確定するまではPASS表現は予測値として扱う**。Go実行が合格しても、認知Methodの実質的増分効果の実証にはならない。

**次にM1を検証するためのゲートではない受入手順**：①現在の実SEI Methodで選べるProbeの表現を確認、②両側に同じ初期Report/Q/現在Source/実Tool/Scope/Grant/予算を提供、③正解Scopeと独立Rを隠す、④Methodが提案したScopeに対して**同じToolで実Read**、⑤Read結果のSHA/時点/失敗/Consumer開示と費用で評価、⑥1回Budgetでの失敗と2回Budgetでの改善を混ぜない。Methodによる選択にModelの追加情報があるならその出典・開示資格を別記。#2514がOracle、SEI #767がMethod、#2533が比較、#2516がDisclosure、TAI #271が再資格化を所有。ADR-0038と業務Meaningの受理はOwner未決。新Host/Registry/正式Proof Store/製品Gateを追加しない。

<!-- tei-devctx-p2c-profile-cohort-20261107 -->
## 13. P2-2c：候補のGo構成適合と複数Readの共同成立を区別する（2026-11-07、提案・NOT RUN）

**位置づけ・確認したSource**：完了済み[#2196](https://github.com/hat47x/tei-platform/issues/2196)と[統合済PR #2669](https://github.com/hat47x/tei-platform/pull/2669)の既存T/CC試験を再利用する。mainの tools/reference-developer-context/main.go（blob dc69dc3a204085801ca509bdda2e2e27b91fa204）の Discover は、Goテストのファイル名・stem本文一致・文書の参照文字列で候補を収集し、Goのbuild profileを解釈しない。targetと各candidateは別々に readInside され、readInside は EvalSymlinks → Lstat → os.ReadFile の順で実行される。**個々のSHA-256が正しいことは、単一の原子的なRead cohortや実Open対象の永続同一性を示さない**。この所見は静的Source確認であり、競合攻撃や実漏洩の成功を意味しない。

### 新しく分離する二つの資格

1. **字句候補 ≠ Go構成で選択されたテスト ≠ 実行済みPASS ≠ Qに必要な独立根拠**。同じテスト候補でもGOOS/GOARCH、build tags、CGO、toolchain、実際のtestコマンドの対象によって利用可能性が異なる。Profile外の候補を絶対的に「無関係」「不存在」とせず、どのProfileでの判断かを明示する。
2. **fileごとの内容fingerprint ≠ 複数Readの共同Snapshot**。targetを読んだ後にSourceが更新されれば、target旧版とcandidate新版を組み合わせたReportを作り得る。旧Reportの再現と、現在の検証・ビルドに必要な共同成立条件を分離する。

### 独立Expected付き最小Pressure（製品修正前の試験計画）

| Case | 一因子変更 | 独立Expected／不適切な昇格を拒否 |
| --- | --- | --- |
| **CC-12 GOOS** | 公開合成packageに qualification_linux_test.go と qualification_windows_test.go を設置。Go testファイル選択対象をLinux/Windowsに切替え、Sourceの内容・字句候補集合を固定 | Discoverは両候補を返し得るが、Goの対象Profileで選択されるテスト集合は異なる。fixture作成時に別途固定した期待集合を、実 go list のTestGoFiles等と比較。go list成功をテスト実行/PASSへ昇格しない |
| **CC-13 Build tag** | qualification_opt_test.go に build constraint optcheck を付け、Goの -tags optcheck 有無のみを変更 | 同じ発見候補でもProfile選択は変わる。文字列一致/候補順位から有効なビルド依存や実テスト実行を推定しない |
| **CC-14 Fractured read** | 評価用の制御Readerで世界を(T0,C0)→(T1,C0)→(T1,C1)と変え、targetはT0、candidateはC1で読み取る | 両digestは正しくても(T0,C1)が同時に実在した時点はない。現在のReportに共同成立を示す証拠が無いなら、評価票で joint-cohort-unverified を残す。製品Report Schema追加はOwner未決 |
| **CC-15 Check/open gap** | 公開temp pathと制御hookだけで、symlink解決・Lstatと実file openの間の変更を圧力試験する | 事前チェックしたrealpath・sizeと実際に読んだfile objectの同一性は未証明と表示する。実際のroot外読取、OS脱出、秘密漏洩の発生はこの設計だけでは主張しない |

**実行順序・再利用**：

- **P0**：CC-12/13は現Discover＋既存Go toolchainで限定的に評価し、discovered C、profile-selected T、actual executed、独立R(Q)を別に記録。Build constraintの正式解釈はGo Toolに委ね、TEIの独自Parserを作らない。別OSで実テストできないときはnot-runであり、失敗や不存在ではない。
- **P1**：CC-14の決定的な読取順序を隔離Fixtureで再現し、個別fingerprintと共同Source資格を別列にする。修正前に新しい全Repository Snapshot Engineを導入しない。実際のBuild入力/Artifact一致は[#2595](https://github.com/hat47x/tei-platform/issues/2595)と[#2016](https://github.com/hat47x/tei-platform/issues/2016)、複数ReadのClaim比較は[#2516](https://github.com/hat47x/tei-platform/issues/2516)がOwner。
- **P1並列**：CC-15は外部秘密Sourceを使わず局所tempのみで再現条件を設計する。root内判定・実Opened file・osごとの制御機構を分ける。物理権限・Network/OS実効拒否は[#2515](https://github.com/hat47x/tei-platform/issues/2515)へ渡す。
- **P2**：既存のD0現Report／D1決定的Probe／SEI M1 Methodを、同一公開可能Projection・Tool・Grant・Scope・読取予算・Source版・評価側に隔離した独立Expectedで比較。Profile未選択/共同Read未確認を「次に取るべき根拠」として説明できるか確認し、実Readのない提案はproposed/unsupportedのまま残す。実モデルと実Product接続はまだ行っていない。

**評価票（公式Schemaではない）**：checkout/HEAD、各Source実byteとdigest、go toolchain/GOOS/GOARCH/CGO/tags、lexical discovered、selected by build profile、actually executed/result、read order、joint-cohort qualification、Scope/budget/failure、独立R(Q)、現在Disclosure、unknown/residual/next probe。Path・候補件数・Toolの存在・診断自体が機密になり得るので、外部へ示す票は既存[#2516](https://github.com/hat47x/tei-platform/issues/2516)に従う。候補名・Markdownは非信頼データであり、モデルへの命令として扱わない。

**追加価値と境界**：決定的なProfile検査/再Readで十分ならモデルを導入しない。go list、mock、model-ranked、Attester verified、digest一致は業務意味受理/現在Grant/外部送出可否/独立Expectedを代行しない（[#2514](https://github.com/hat47x/tei-platform/issues/2514)、[SEI #820](https://github.com/hat47x/sei-cognition/issues/820)）。認知Methodが担うのは不足根拠と識別的な次Probeの選択（[SEI #767](https://github.com/hat47x/sei-cognition/issues/767)）、再開と現在資格は[TAI #271](https://github.com/hat47x/tai-pleroma/issues/271)に委譲する。#2196/#2021は閉鎖状態のまま。**CC-12〜15、実Go/CI、実モデル、実Provider/OS、Owner AcceptanceはNOT RUN**。第二Claim Store、Agent Host、Authority、認知/証拠Gateを作らない。

<!-- tei-devctx-harness-delivered-evidence-20261107 -->
## 14. P2-2c実測・P2後続：実Harnessの「取得・Tool返却・受領の完全性・利用資格」を別列へ接合（2026-11-07）

### 14.1 §13の実装状態更新：CC-12/13だけ限定RUN

PR [#3176](https://github.com/hat47x/tei-platform/pull/3176)の test-only `tools/reference-developer-context/profile_selection_characterization_test.go` に、独立の公開合成Packageを作る **CC-12（GOOS Linux/Windows）／CC-13（Go build tag）** の2つのGo Testを追加した。既存`Discover`による候補収集と、実Go Tool `go list -json` の `TestGoFiles` を別の入力・結果として照合する。Tool環境は`GOOS`・`GOARCH`・`CGO_ENABLED`・tags・toolchainを限定し、Sourceを変更せずにProfileだけ変える。Go構成選択は**実テスト実行・独立検証基準・Owner受理ではない**。

- **Exact test commit**：`f0020c1b300a9f75d85e38b0709c531ba31786d8`。
- [Focused CI #42345862680](https://github.com/hat47x/tei-platform/actions/runs/42345862680) の `bounded-discovery` は **completed/success**。実Job log `128323341089` に `TestDiscoveryProfileCC12GOOSSelection`（linux/windows両subtest含む）と `TestDiscoveryProfileCC13BuildTagSelection` の **PASS** が明示されている。
- CC-14 fractured read／CC-15 check-open gap、製品Reporter修正、実Windows実行、実Model、実Provider、Owner受理は **NOT RUN**。このexact commitのrepo-wide healthが独立に成功したかは、該当Workflow結果を別途検証する。

### 14.2 認知側に必要な証拠資格：Tool実到達と有効な回答を分ける

他レーンの実証[TEI #2509](https://github.com/hat47x/tei-platform/issues/2509)、[統合済Evidence #3113](https://github.com/hat47x/tei-platform/pull/3113)、[四Session実Evidence](https://github.com/hat47x/tei-platform/blob/main/plan/evidence/DSH_H2_H3_FOUR_SESSION_2026-11-07/README.md)、[TAI #168 Runbook](https://github.com/hat47x/tai-pleroma/issues/168)を参照する。以下は**既存実測の認知評価への受渡し**であり、このPRでDSHを再実行した証拠ではない。

| 実測された事象 | 認知Methodへの受渡し可能なClaim | 禁止する昇格／未取得 |
| --- | --- | --- |
| DSH H2-AでTool Call 56・Tool Result 55・TEI CLI実起動32。catalog Tool Result 8,192 byteで途中切断 | Harness→実CLI→Tool応答の限定経路は成立。Call/Result・必要Contentの全量到達は未確認 | Caller completed、同じFingerprintや一部JSONから`catalog complete`／全件を読んだと推測しない |
| H3-A/Bが各5,482 byteの同一Skill本文を実Load、refs/read-only-model-investigation.md をToolで読取 | Product Skill root/参照ファイルの**限定Delivered/Read**実証 | Instructionを正しく適用、Required refs全閉包、独立正答、Capability/Authorityは未実証 |
| H2→H3→H3→H2の実四Session completed | 技術的な実Session到達とSkill提示の有無は観測済み | H2-A再利用、Taskの論理Path差、隔離不足、許可外Tool、独立採点無し。TAIの**比較適格0/4**は不変 |
| `read-only`を指定してもDSHが作業場所外SourceをReadした | 宣言mode≠OS/physical scope証拠 | Allowed Workspace/Tool/SourceやSecret不受領を自己申告しない。実隔離はTEI #2515／TAI #232 |
| 旧TEI実CLIでは一部kindの`trace`が`SUBJECT_NOT_FOUND`と誤返却していたが、[merged #3208](https://github.com/hat47x/tei-platform/pull/3208)で未対応kindを`UNSUPPORTED_SUBJECT_KIND`へ分離 | 現行mainの投影対象外は機械可読な未対応。固定旧DSH Binaryの過去Tool結果は歴史的な旧挙動のまま | 検証したCLI build/ABIを示さず「未対応ではない」「関係不在」と推論しない。別の投影範囲・実Consumer再受領は未確認 |
| 実CLI replyにmanifest identityが無い[#3181](https://github.com/hat47x/tei-platform/issues/3181)、viewの`read-only`宣言でもflagにより局所FSへ書く[#3182](https://github.com/hat47x/tei-platform/issues/3182) | Reviewerは実argv・root realpath・manifest byte・出力先・実Effectを一試行へ別束縛 | WorldFP＝Workspace ID、command family read-only＝実Invocation無書込みを推測しない |

### 14.3 P2比較の最小Evidence ladder（正式Schemaではない）

**L0 宣言／候補**（Skill/Tool/Profile発見）→ **L1 実起動**（実argv/実Binary/実SourceとHost境界）→ **L2 実返却**（Call ID→Result ID・raw bytes/長さ/構造・stderr・truncate）→ **L3 必要根拠の到達**（問いQの必要R、選択済みGo Profile、actual complete read/Scope/共同成立・Source時点）→ **L4 独立評価**（hidden Expected・negative/control・誤不存在と未確認保持）→ **L5 現在利用**（Owner Meaning/Consumer Disclosure/Current Grant/Effect、各担当の観測）。これは**評価票上の証拠の分類**であり、共通Runtime状態型／新Claim Registryではない。

一段前の成功から後段を推論しない。例えば Tool ResultがJSON構文としてvalidでも、Serverが部分集合しか返していない、Terminalが出力を切断した、Consumerが最後まで実受領していない、現在Grantが失効した場合は、問い相対にL3以降を未確認にする。一方、単一Witnessによる存在反証なら、全件閉包を必須条件にせず、対象帰属・版・開示の必要条件だけを評価する（[SEI #767](https://github.com/hat47x/sei-cognition/issues/767)）。

**実行・評価Ownerと優先順**：最初の事前登録H2/H3比較ではTAI #168の固定Task/Skill/Expected/Tool集合を変更しない。G0（Participant隔離と許可Sourceの実Read拒否）→G1（実送出範囲）→G2（実Tool・受領byteとSource/Workspaceの束縛）を先に検収し、独立採点はTAI担当。TEI #2509は実Harness/Tool出力、TEI #3180/#3181/#3182はCLI個別契約、TEI #3061は固定Host物理rootの反例、TEI #2514は独立Expected、TEI #2516はPurpose/Disclosure、SEI #767は不足根拠と次ProbeのMethodを所有する。今回P2評価側はそれらを一つのClaimへ過剰集約しない。**新しいH2/H3比較実行、実SEI Method接続、OS隔離、全Scope/全件資格、Owner受理は本§14ではNOT RUN**。

<!-- tei-devctx-cc14-actual-fractured-cohort-20261107 -->
## 15. CC-14の限定実Go反証を、問い相対のRead Cohort資格へ接続（2026-11-07）

### 実測と「言えること」の境界

[Draft PR #3176](https://github.com/hat47x/tei-platform/pull/3176)の`tools/reference-developer-context/source_cohort_characterization_test.go`に、`TestActualReadCC14IndividualDigestsDoNotProveJointCohort`と`TestActualReadCC14StableCohortIsOnlyFixtureQualified`を追加した。実既存`readInside`を2回呼ぶ間に、公開`t.TempDir`のSourceを意図的に書き換えるだけの、**決定的な単一スレッド試験**である。秘密・実OS競合・外部Providerは使わない。

- **固定World**：観測済み実file状態を`(T0,C0) → (T1,C0) → (T1,C1)`と進める。Target読取はT0、candidate読取はC1。各`readInside`の実byteとSHA-256は正しいが、`(T0,C1)`は記録したいずれのWorldにも存在しない。評価側は同時点のペアを**試験開始前に固定した三つのEpoch**で独立検査する。単に「SHAが一致しない」ではなく、「**正しい個別SHAを貼り合わせてもjoint Worldが構成されない**」を反証する。
- **正対照**：新しい通常`Discover`を最終の安定`(T1,C1)`にかけるとT1/C1の個別fingerprintを返し、もう一度同じSourceにかけた安定対照はbyte一致する。ただし二回の一致も**任意の実Repoに対する原子的Snapshot保証ではない**。
- **exact 実行証拠**：変更HEAD `04cefdb2874b0b0c86c7e64c221add5511cf3ae9`の[専用Focused CI #42370160427](https://github.com/hat47x/tei-platform/actions/runs/42370160427)（job `128396439042`）は**completed/success**。Jobログで上記2テストとCC-12/13がすべて`--- PASS`と出力。既存Workflowの`-run`に`TestActualRead`を追加し、テストファイルが存在するだけで未実行になる事態を避けた。
- **保証しないこと**：実`Discover`中の並行更新をスケジュールして競合を再現した**わけではない**。その内部ループ自体に`(T0,C1)`が発生したこと、Go testによるRace検出、実OS sandbox/pinning・root外Read、全Sourceの原子的整合・業務受理は証明しない。現製品`Report`のSHA fieldだけでは共同成立Epochや実Read cohortを表現できない、という**表現の限界**を試験した。

### 用途に相対的なRead Cohortを既存Receiptへ結び付ける

**Read Cohortは新しい中央Store／Core Runtimeオブジェクトを意味しない。** 既存#2516のConsumer用途相対Contextと、Source/Build/Attempt Receiptの証拠票に、当該Claimへ必要なSourceの集合と実消費byteが**共同に成立したか**を示す限定Wit​​nessを組み合わせる考え方である。目的別に以下を切り分ける。

| どのClaimを行うか | 必要になる最小の追加観測 | 今回得られたもの |
| --- | --- | --- |
| `Goの構成Pでtest候補を選択できる` | GOOS/GOARCH/tags・toolchainを固定し、実`go list`と独立Expectedを突合 | CC-12/13の合成実Go PASS。実テスト実行は別 |
| `Source Sと候補Cを同じ検証根拠として使える` | 問いQに必要なRead-set、双方の実byteと、共同成立を示すSource version/cohort資格 | **個別SHAだけでは不足する反例**をCC-14で実Go確認。実Cohortの一般的取得方式は未採用 |
| `否定Claim Qについて対象が存在しない` | 対象Universe、非出力Predicate、Relation/Provider/必要依存の閉包、実Readの完了と共同Scope | [#2516](https://github.com/hat47x/tei-platform/issues/2516)/[SEI #767](https://github.com/hat47x/sei-cognition/issues/767)のOwner。CC-14では**NOT RUN** |
| `署名済みPlugin/Artifactを現在利用できる` | exact Release/Artifact/Verifier署名、現Trust/Revocation、実HostのLoaded bytes/Effect | [TEI Draft #3244](https://github.com/hat47x/tei-platform/pull/3244)/[#2589](https://github.com/hat47x/tei-platform/issues/2589)に委譲。CC-14のSHA不十分さを実Plugin侵害の証拠とはしない |
| `外部Harnessが適格な回答を返した` | 実Tool呼出し/Result全文受領、Workspace/Source Binding、Scope/Grant、独立Expected、開示可能性 | [TAI #168](https://github.com/hat47x/tai-pleroma/issues/168)の実Sessionを参照するが比較適格は0/4 |

**最小の実装候補（まだ採用しない）**：`root resolved identity + worktree/dirty status + required Read Set + exact digests + effective Go Profile + observed period/revision + purpose/consumer`を**試行ごとの検証票**として提示する。Git HEADやWorldFPだけをRead Cohortの一意識別としない。dirtyなSourceがHEAD不変で更新される、同じファイルがA→B→Aへ復帰する、実Readにない新規依存が増える、別rootの同名Subjectを取得した場合を負例にする。実Sourceが共同に成立したという保証を求めるConsumerには、使用したSourceを固定するBuild/Read方法、限定再取得と前後照合、または実ProviderのSnapshot能力を必要条件として個別に選ぶ。**前後照合だけをあらゆる並行更新・ABAの数学的な完全証明にはしない**。

### 次の反証と優先度

1. **CC-15は別試験に隔離**：`readInside`の`EvalSymlinks → Lstat → os.ReadFile`の間に別file objectへ変わり得ることを、実際にOpenしたinode/objectの検証とは区別して確認する。公開temp path以外を使わず、現mainの実際のroot外読取を確認できなければ未証明。Studio固定Hostの[既存負例 #3061](https://github.com/hat47x/tei-platform/issues/3061)も個別のOwner証拠でありこのPoCの成功に流用しない。
2. **Read-set再資格化**：対象Source byte不変・Candidate変更、Candidate不変・Go Profile変更、Source内容復元A→B→A、依存Edge追加、Scope/Disclosure失効の5組で、どのQのどの結果を再使用できるかを[#2516](https://github.com/hat47x/tei-platform/issues/2516)／[TAI #271](https://github.com/hat47x/tai-pleroma/issues/271)へ戻す。安全と確認済みの変更のみ再利用し、不明な依存を無影響としない。
3. **M1との実比較は後段**：決定的なProfile確認／必要Read再取得で解消する不足と、SEI Methodの調査選択で初めて解消する不足を、同じ実Tool/Grant/予算/開示で比較する。OKF/SHO/SUIの表示・知識交換は証拠を運ぶ側でありOwnerのMeaning受理や共同Snapshot生成器ではない。

**最新到達点**：本Sectionの新規2テストは固定HEADのGitHub Actionsで限定PASS。PRはDraft／未マージ、製品`Discover/Rerank`本体と正式`Report` ABIは変更しない。共同Source Snapshotの製品能力、CC-15、実外部Harness Method、実OS/Network/Provider、Owner受理はNOT RUN。

<!-- tei-devctx-cc15-staged-check-open-and-safe-root-20261107 -->
## 16. CC-15：チェック時のPathと実Open対象を区別する限定安全試験（2026-11-07）

### 採用する観測単位

[Draft #3176](https://github.com/hat47x/tei-platform/pull/3176)のtest-only `tools/reference-developer-context/path_open_characterization_test.go`で、実`readInside`が用いる3段階のAPI呼出し（`filepath.EvalSymlinks`→`os.Lstat`→`os.ReadFile`）の**検査後・最終Open前**にパスを意図的に差し替える。公開`t.TempDir`内の通常ファイルから、別の公開`t.TempDir`にある合成Canaryへのsymlinkへ置換する。これらは**テストが時刻を制御して個別APIを呼んだ**ものであり、実`readInside`内部を割込ませた試行でも、実敵対的な同時実行攻撃の成功でもない。

| 独立対照 | 検証する事象 | 主張してよい上限 |
| --- | --- | --- |
| **CC-15a staged check-open** | 旧ファイルのEvalSymlinks/Lstatはroot内regularを確認した後、最終`os.ReadFile`が差替えsymlinkをたどり、公開・合成root外Canary byteを読む | 同じPathを**別時点にチェックし、別APIでOpenする設計はfile object identityを証明しない**。本体`readInside`の競合成功・非公開Data漏洩・実OS Sandbox突破は未実測 |
| **CC-15b after-replacement validation** | 差替え**後**の通常`readInside(root, name)`はroot外symlinkを拒否 | 静止した外部symlinkの拒否を示すが、検査と実Openの間の原子性・同時攻撃耐性は示さない |
| **CC-15c root-relative alternative** | 標準`os.OpenRoot(root).Open(name)`で当該外部symlinkを拒否し、安定したroot内fileを読めるか | **比較したGo Toolchain/OSとこのFixtureに限定**。Root能力の採用、全Platform/特殊mountやA→B→A・複数fileのcohort整合性は別資格 |

**限定CIで実測済み**：テスト正本commit `c7030a8f4a7897af518a729717065dee514ccef5`の[Focused Actions #42370450702](https://github.com/hat47x/tei-platform/actions/runs/42370450702)は**completed/success**、job `128397310948` の実ログで `TestActualReadCC15PathCheckIsNotFileOpenIdentity` と `TestActualReadCC15StableInRootFileRemainsReadable` がともに **PASS（SKIPではない）**。同じrunではCC-14の2件もPASS。Goの実File I/OとOpenRootを使う公開合成試験に限定した証拠であり、Repo Health、実本体内部の介入競合、実OS攻撃防御、全Platformの保証は別。symlink作成権限がなく`Skip`した将来の環境は、この成功したUbuntu試験とは区別して数える。

### 実装案を限定する

Goの[Traversal-resistant file APIs（Go公式）](https://go.dev/blog/osroot)に記載された`os.Root`（Go 1.24+）は、root外へのsymlink traversalを防ぐ**候補手段**。当該repoは`go.mod`でGo 1.26.10を要求するが、API採用前に実Go/OS/互換性を別検証する。[`os.Root`の既知修正（GO-2026-4970）](https://pkg.go.dev/vuln/GO-2026-4970)の影響がある古いToolchainを無条件に安全とみなさない。とくに`os.Root`のTraversal保証と、mount境界／全Platform／OS sandbox／実Consumerへの開示や複数fileの共同時点は同義でない。

採用判断で比較するのは「**root-relativeに実FileをOpen → 開いた実ハンドルのStatでregular/sizeを確認 → 上限付きReadで実bytesをhash**」という単一Fileの実取得手順と、現在の事前Lstat→別Openの手順。さらにRootの固定時点、Target相対Pathの正規化、dir symlink、権限失効、Source rename、opened objectの保全を対象にする。**新しい一律File StoreやRuntime Permission Engineは導入しない**。変更案・API/Go最低版判断はTEI #2533とSource/Host各Ownerへ戻す。

### 横断的な影響と停止線

- [Studio固定Host #3061](https://github.com/hat47x/tei-platform/issues/3061)は**固定旧Host `4bc437c7` の実負例（root置換後の外部Catalog Read）**と、**現main側の別の正対照（既存root同一性検査・統合済[#3098](https://github.com/hat47x/tei-platform/pull/3098)・公開再検査PASS）**を分けている。これはWorkspaceの**root同一性**の試験であり、個別ファイルのCheck/Open、Go Root API、複数fileの共同Snapshotを保証したわけではない。本CC-15は参照用ローカル`readInside`の分割API試験であり、Studio本体に同じ未修正欠陥があると主張しない。
- [TEI #3181](https://github.com/hat47x/tei-platform/issues/3181)のWorkspace Manifest Identityと`WorldFingerprint`、[TEI #2589](https://github.com/hat47x/tei-platform/issues/2589)の署名付きArtifact exact bytes、[TAI #168](https://github.com/hat47x/tai-pleroma/issues/168)の実Tool Deliveryはそれぞれ**異なる同一性の観測**。Path同一だけでもSHA同一だけでも、実Hostが開いた対象・現在Grant・Consumer開示を証明しない。
- P2認知Methodはこの不足を`unknown`/必要な次の検査として記述できるかが対象。実Root/Open拒否はProduct/Host・OS、実外部送出はTEI #2515／TAI #232、独立ExpectedはTEI #2514に残す。秘密Sourceに対する攻撃試行、外部API/DB副作用、Owner受理、正式Core APIは本試験の範囲外。

**注記**：CC-14は複数Fileの共同時点、CC-15は一つのPathに対する事前Checkと実Open対象の同一性であり、両者を一つのPASSに統合しない。

<!-- tei-devctx-cognition-data-representation-20261107 -->
## 17. Standard Data・表現型・Atomic Traitを認知の証拠層へ安全に投影する（2026-11-07、統合設計・新実証NOT RUN）

**横断の成熟した方向**：[TEI Draft #3236](https://github.com/hat47x/tei-platform/pull/3236)（標準データ構造×標準表現型の合成写像）と[TEI Draft #3282](https://github.com/hat47x/tei-platform/pull/3282)（Atomic Traitの実装義務・合成衝突）、[TEI #2507](https://github.com/hat47x/tei-platform/issues/2507)のFunction Binding、[TEI #2516](https://github.com/hat47x/tei-platform/issues/2516)の用途相対Contextを参照。表現型・Traitの新たな全体モデルを本Issueでは作らず、認知Methodがどのデータの何を観測できたかを失わず引き渡す。

**観測の分解**：元の標準データ構造（Record/Collection/Relation/Graph/時系列等）・対象範囲と粒度・Relationの向き/多重度・時点/単位・NULL/未取得/非開示、FunctionのSelection/Query/Rule依存、表示先の標準表現型（Table/Tree/Graph/Panel等）、実Consumerへ渡されたSlice、を別欄にする。Function Bindingは接続の意味を所有し、Rendererの見た目やExportされたOKF/SHO文書は元構造の完全な逆写像を保証しない。

| 反証対象 | 同じ公開表現から誤って推測しやすい主張 | 既存Ownerへ渡す検証 |
| --- | --- | --- |
| Data→Table（Collection先頭1件のみ表示） | 表に1件だから対象集合も1件／全件合格 | #2506/#2516の実選択・Provider完全性とR(Q)を確認。表示件数ではなく実Read Scopeを独立に照合 |
| Data→Graph（同labelノード、多重辺、方向を省略） | 同じ図の形は同じEntity同一性・Relation意味を示す | #2495/#1124、元のNode/Edge ID・向き・多重度・属性をSourceに遡及できる場合だけ比較 |
| Data→Document/OKF（派生Index/Markdown） | 表示が同じなら原資料・非公開WHERE・差分履歴も同じ | #2634/#2516／SHO／SEI #820。Source/Predicate/粒度・改訂・開示を残し、不明ならunknown |
| Plugin/Trait→Tool discovery | Capability/Atomic Traitの宣言があるから対象実装・現在権限・動作が保証された | #3282/#2390/#2589、実Provider/Host、Pin/Build Artifact、Signature/Attesterと実Load/Effectを分離 |

**AI/CUI/MCPへの最小提示方針**：実際に存在する既存TEI CLIのcatalog/inspect/trace/validateとSource/Manifest/Tool Replyを使用し、試行用Projectionとして「問いQ、対象/Source版、必要な依存、観測済みと未観測、実Go Profile／Read cohort、Disclosure、根拠となる実Tool/Receipt、次に確認すべき情報」を出す。現CLIに存在しない新しい inspect-context / probe / MCP契約を実装済みと偽称しない。SEI Methodは公開・実行可能な候補から次Probeを提案し、Target Providerの必要条件・Owner Meaning・Host GrantはTEI等既存Ownerに残す。

**次の比較順**：(A) 現行CLI/データ構造から決定的に作れるread-only表、(B)同じ実Sourceから生成した複数表示で落ちる情報の反証、(C)限定Toolでの実追加Read、(D)SEI Methodの識別的なProbe選択、の順に独立Rを隠して比較。表示図の美観・モデル自己申告を十分性採点に入れず、新たなRegistry/中央Cognition Host/第2意味正本を作らない。**本§17は所有Issueを横断する評価条件の接続であり、新たな実Provider/Renderer/Tool試験やOwner AcceptanceはNOT RUN**。

<!-- tei-devctx-cc16-execution-qualification-20261108 -->
## 18. CC-16：候補発見、Go構成選択、テスト実行、独立Expectedを別資格にする（2026-11-08）

### 18.1 追加した技術的反例

[Draft PR #3176](https://github.com/hat47x/tei-platform/pull/3176)の\`tools/reference-developer-context/profile_execution_characterization_test.go\`へ、公開合成moduleだけを使う\`TestDiscoveryProfileCC16SelectionIsNotTestExecutionOrPass\`を追加した。新規テストコードのcommitは\`4abdc3d02ad308762957dc52cd95859c7768b714\`。実\`go list -json\`による構成選択と実\`go test -json -count=1\`の**名前付きTest Event**を区別する。本節執筆時点では該当exact commitのCIは未完了なので、以下のPASS/FAILは**テストの期待であって実CI検収結果ではない**。

| 資格 | 固定合成期待 | 不許可の飛躍 |
| --- | --- | --- |
| 字句発見 | \`Discover\`がbaseとstrictの2候補を返す | 2件を実テスト実行数に数えない |
| Go構成選択 | 通常Profileではbaseのみ、\`-tags strict\`ではbase+strictが\`TestGoFiles\`に入る | 選択されただけで実行・合格としない |
| 実Go Run | 通常は\`TestBaseline=pass\`、strictで\`TestBaseline=pass\`＋\`TestStrict=run/fail\`を実Eventで照合 | package exitと個別testの合否を混同しない。失敗するstrictは意図的な独立合成Expected |
| no-test正対照 | 別moduleでは\`go test -json\`がexit 0でも、名前付きTest Eventが0件 | exit 0をC0/C1 coverageやシナリオ合格と認定しない |

CC-12〜15は選択・Source読取の必要条件までであり、CC-16の成功も**実TEI業務Ruleのテスト・AI生成シナリオの有効性・Owner受理を証明しない**。Testの原典・target build・実行記録・検証基準を同じepoch／Consumerの現在資格として再結合することは別。

### 18.2 他チャットの実装成果と接続：Design Contextが不足する場合はテスト生成を保留

[統合済TEI PR #3284](https://github.com/hat47x/tei-platform/pull/3284)は、9種類の実TEI CLI応答を利用する限定Consumerと7合成Negativeを追加し、\`boundedReference=partial\`／\`testExecutionAdmission=exploration-only\`／\`qualifiedTestGeneration=false\`の方向を検証対象としている。実Goテストの実行に成功した場合でも、Function/Business/Rule/Provider/GrantなどRequired閉包が未取得なら、この保留を覆せない。AIによる新負例の自律生成や独立C1証明に成功した実績ではない。

P0-M（三視点Diagram/CRUD）とP0-R（Function→Query→実Provider→Context→Form/Table）の別々の進捗を、P0-C（Source/Subject/Scope/Revision/粒度/必要Relation）で結ぶ**同じ設計対象の検証**を次の受入条件とする。表現上のGraphに辺がない、Queryが空、Tableが1件、\`go test\`がexit 0、のいずれも関連するRequiredの不存在・全件適合・テストカバレッジを単独では示さない。Req/Obligationの独立Expectedと既知MutantのOwnerは[TEI #2514](https://github.com/hat47x/tei-platform/issues/2514)、Issue→Test探索のOwnerは[#3262](https://github.com/hat47x/tei-platform/issues/3262)。認知Methodは不足根拠・適用可能な次Probeを[SEI #767](https://github.com/hat47x/sei-cognition/issues/767)へ返す。新Core Test Engine、中央Evidence Registry、専用認知Hostは増設しない。

### 18.3 統合前の限定修正と更新されたProductの事実

- 前回PR HEAD \`76d3af2c48e898c526b17764134cae79047ee24b\`の[Repository Health #42370899309](https://github.com/hat47x/tei-platform/actions/runs/42370899309)は**failure**。ログではPR追加\`profile_selection_characterization_test.go\`の\`gofmt\`ずれを確認し、commit \`feaf56fdcc9f306ba466e40aa98593ca6eb1db5a\`でその2行の整形を修正した。同じ失敗Runには別計画領域の\`RQ-035\`索引欠落と\`plan/design/README.md\`の4ファイル欠落が残る。これをCC-16のテスト不合格とも、現在HEADのHealth PASSとも解釈しない。
- 旧固定DSH Binaryでの「Workspace identity未返却」は歴史的観測。[TEI #3181](https://github.com/hat47x/tei-platform/issues/3181)はcloseされ、**現在mainに\`workspace.Result.Identity()\`と\`operations.ModelCatalog.Workspace\`のSourceを実確認**。現行Sourceの有無と、実外部Harnessがその出力を受領したかは別（今回再実行はNOT RUN）。\`source-index\`／\`knowledge-*\`への同様の投影は未確定。
- [TEI #3180](https://github.com/hat47x/tei-platform/issues/3180)の\`trace\`未対応kind／真正な不存在のエラー分離は統合済[#3208](https://github.com/hat47x/tei-platform/pull/3208)を参照。旧Runの\`SUBJECT_NOT_FOUND\`に新APIの解釈を遡及適用しない。
- [Studio Host #3061](https://github.com/hat47x/tei-platform/issues/3061)では現mainのroot同一性検査（\`os.SameFile\`）と固定旧Host負例を分離している。これはCC-15の「単一file check-openの原子性」やCC-14の「共同epoch」を製品実装が保証したという根拠ではない。

### 18.4 次の実施順序／判定停止線

1. **P0・限定実行票を完成**：CC-16のexact HEAD focused jobログを確認し、\`go test\`の名前付きpass/fail/実行なしを観測。全体Healthの\`gofmt\`／索引問題を別軸で確認する。強制的なTest数0をPASS表示しない。
2. **P1・Design Context全Required閉包**：#3284の\`qualifiedTestGeneration=false\`から始め、P0-M／P0-Rの実Source・Function/Business/Rule依存、Providerの実Read、C0/C1観測を個別に充足する。部分Scopeの局所テスト・探索を禁止するのでなく、**全体の正答証明と区別して続行**する。
3. **P2・追加認知価値**：決定的なD0（現在Tool/Sourceから不足を列挙）とSEI MethodのM1（どの追加Read/Scopeを選ぶか）を、同じSource/Read予算/Grant/開示とblind Expectedで比較。誤不存在、誤ったTest Admission、未取得依存、追加Read費用、未知Caseの発見を別指標にする。

**現時点の範囲**：CC-16は製品コード無変更・合成Go試験の追加。本PRで新たな外部Harness実比較、Provider／実DB、OS/Network拒否、実AI生成テスト、Owner正式受理はNOT RUN。報告に実CI PASSと記すには当該runの検収が必要。

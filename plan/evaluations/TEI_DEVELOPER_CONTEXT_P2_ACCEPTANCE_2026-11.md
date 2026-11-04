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

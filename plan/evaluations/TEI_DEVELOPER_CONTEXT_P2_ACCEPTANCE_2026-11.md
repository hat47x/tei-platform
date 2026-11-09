# TEI Developer Context P2: 評価・改善受入計画（2026-11-03）

## 現在の受入順序（2026-11-08）

> **現在の判断をここから読む。** 下の§1〜20は11月3日以降の段階的Characterizationを保持した歴史的基線であり、冒頭の13件PASSを現在の全製品／外部Harness合格へ広げない。現在の詳細Evidence・停止線は§21〜26.7、採用判断は既存Owner #2533/#2514/#2516／SEI #767/#824に従う。

| 優先 | 現実に成立している入口と次に検証する一つの不足 | 完了判定 |
| --- | --- | --- |
| **P0 定義観測（main限定PASS）** | TEI main #3482の13実CLI Reply／42限定Consumer Controls、#2639 EG-09の同結果型・異Attribute Type反例を再利用。Function/Rule/Sourceの**宣言**と必要な型・Meaningを分ける | 実Source/Loader/Ruleの局所Source資格まで。実Function Binding/Provider/Rowset/Owner Meaningは未成立 |
| **P1 H3-A0（限定実Go PASS）** | TEIとSEIの実CLIを別々にbuildし、TEI source-index subject/index両Modeを取得→SEI Recommendationの保存→独立resume原記録再読取（§26.8） | 固定TEI/SEI Source SHA・専用CI #42474971439で実証。SEI Method・未解決InquiryのNative保存・現在Grant/外部Harnessは未実証 |
| **P2 H3-A1（次の未実証）** | 同じQ/Tool/Scope/Grant/Budget=1・blind ExpectedでD0/D1/M1を比較。決定的index-mode再取得D1を**陰性対照**にする（§25.4・§26.7） | M1がD1と同じProbeなら増分価値なし。実Invoke/完全Reply/必要R(Q)・費用を別欄で採点 |
| **P2 並行・別Owner** | 実SUI Producer→実SHO Reader同一File H1、TEI実Function Invocation Binding→実Provider、外部TAI H2/H3隔離・完全受領 | 個別CI／Working Copy／TEI CLI↔MCP Adapter一致を横断PASSへ合算しない |
| **P3 保留** | SEI Decision前Question/UnknownのNative継続、正式SHO Editor Save、Ontology/Decision Twin、REA・KONの拡張 | 初回A0/A1のBlocking条件にはしない。新Host/Registry/第二Source・Proof Storeを設けない |

**現在の証拠**：H3-A0は固定した二つの実Goバイナリと実Source Index応答・実SEI Storeの限定往復について**PASS**（§26.8）。H3-A1/A2、実SUI H1、Function Provider、実SEI Method、外部Harness、Owner意味受理は**NOT RUN**。Draft #3176も未merge。SourceやWorkflowの正本を当計画に移さない。



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
## 13. P2-2c：候補のGo構成適合と複数Readの共同成立を区別する（原案2026-11-07／CC-12〜15の後続実証は§15〜16）

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
| 旧固定CLI replyにはmanifest identityが無かったが、現mainはWorkspace Identityを一部出力する[#3181](https://github.com/hat47x/tei-platform/issues/3181)。一方viewの`read-only`宣言でもflagにより局所FSへ書く[#3182](https://github.com/hat47x/tei-platform/issues/3182) | Reviewerは実argv・root realpath・manifest byte・実Binaryの版・出力先・実Effectを一試行へ別束縛 | WorldFP＝Workspace ID、現mainの新項目＝旧DSH受領済み、command family read-only＝実Invocation無書込み、を推測しない |

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

**横断の設計参照（両PRは2026-11-08照合時にclosed・未merge）**：[TEI #3236](https://github.com/hat47x/tei-platform/pull/3236)（標準データ構造×標準表現型の合成写像）と[TEI #3282](https://github.com/hat47x/tei-platform/pull/3282)（Atomic Traitの実装義務・合成衝突）、[TEI #2507](https://github.com/hat47x/tei-platform/issues/2507)のFunction Binding、[TEI #2516](https://github.com/hat47x/tei-platform/issues/2516)の用途相対Contextを参照。表現型・Traitの新たな全体モデルを本Issueでは作らず、認知Methodがどのデータの何を観測できたかを失わず引き渡す。

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

[Draft PR #3176](https://github.com/hat47x/tei-platform/pull/3176)の`tools/reference-developer-context/profile_execution_characterization_test.go`へ、公開合成moduleだけを使う`TestDiscoveryProfileCC16SelectionIsNotTestExecutionOrPass`を追加した。新規テストコードのcommitは`4abdc3d02ad308762957dc52cd95859c7768b714`。実`go list -json`による構成選択と実`go test -json -count=1`の**名前付きTest Event**を区別する。[Focused Actions #42385582970](https://github.com/hat47x/tei-platform/actions/runs/42385582970)のjob `128442144943`は、当該exact Test commitで**completed/success**。実ログで`TestDiscoveryProfileCC16SelectionIsNotTestExecutionOrPass`が`--- PASS`（0.55s）、CC-12/13/14/15も同じjobでPASSを確認。CC-16の内部ではGo実`go test -json` Event／Exitを照合した。これは**公開合成fixtureにおける限定Go実行試験**であり、実TEI Targetの生成AI Test能力・C1/業務Meaning受理の立証ではない。

| 資格 | 固定合成期待 | 不許可の飛躍 |
| --- | --- | --- |
| 字句発見 | `Discover`がbaseとstrictの2候補を返す | 2件を実テスト実行数に数えない |
| Go構成選択 | 通常Profileではbaseのみ、`-tags strict`ではbase+strictが`TestGoFiles`に入る | 選択されただけで実行・合格としない |
| 実Go Run | 通常は`TestBaseline=pass`、strictで`TestBaseline=pass`＋`TestStrict=run/fail`を実Eventで照合 | package exitと個別testの合否を混同しない。失敗するstrictは意図的な独立合成Expected |
| no-test正対照 | 別moduleでは`go test -json`がexit 0でも、名前付きTest Eventが0件 | exit 0をC0/C1 coverageやシナリオ合格と認定しない |

CC-12〜15は選択・Source読取の必要条件までであり、CC-16の成功も**実TEI業務Ruleのテスト・AI生成シナリオの有効性・Owner受理を証明しない**。Testの原典・target build・実行記録・検証基準を同じepoch／Consumerの現在資格として再結合することは別。

### 18.2 他チャットの実装成果と接続：Design Contextが不足する場合はテスト生成を保留

[統合済TEI PR #3284](https://github.com/hat47x/tei-platform/pull/3284)は、9種類の実TEI CLI応答を利用する限定Consumerと7合成Negativeを追加し、`boundedReference=partial`／`testExecutionAdmission=exploration-only`／`qualifiedTestGeneration=false`の方向を検証対象としている。実Goテストの実行に成功した場合でも、Function/Business/Rule/Provider/GrantなどRequired閉包が未取得なら、この保留を覆せない。AIによる新負例の自律生成や独立C1証明に成功した実績ではない。

P0-M（三視点Diagram/CRUD）とP0-R（Function→Query→実Provider→Context→Form/Table）の別々の進捗を、P0-C（Source/Subject/Scope/Revision/粒度/必要Relation）で結ぶ**同じ設計対象の検証**を次の受入条件とする。表現上のGraphに辺がない、Queryが空、Tableが1件、`go test`がexit 0、のいずれも関連するRequiredの不存在・全件適合・テストカバレッジを単独では示さない。Req/Obligationの独立Expectedと既知MutantのOwnerは[TEI #2514](https://github.com/hat47x/tei-platform/issues/2514)、Issue→Test探索のOwnerは[#3262](https://github.com/hat47x/tei-platform/issues/3262)。認知Methodは不足根拠・適用可能な次Probeを[SEI #767](https://github.com/hat47x/sei-cognition/issues/767)へ返す。新Core Test Engine、中央Evidence Registry、専用認知Hostは増設しない。

### 18.3 統合前の限定修正と更新されたProductの事実

- 前回PR HEAD `76d3af2c48e898c526b17764134cae79047ee24b`の[Repository Health #42370899309](https://github.com/hat47x/tei-platform/actions/runs/42370899309)は**failure**。ログではPR追加`profile_selection_characterization_test.go`の`gofmt`ずれを確認し、commit `feaf56fdcc9f306ba466e40aa98593ca6eb1db5a`でその2行の整形を修正した。同じ失敗Runには別計画領域の`RQ-035`索引欠落と`plan/design/README.md`の4ファイル欠落が残る。これをCC-16のテスト不合格とも、現在HEADのHealth PASSとも解釈しない。
- 旧固定DSH Binaryでの「Workspace identity未返却」は歴史的観測。[TEI #3181](https://github.com/hat47x/tei-platform/issues/3181)はcloseされ、**現在mainに`workspace.Result.Identity()`と`operations.ModelCatalog.Workspace`のSourceを実確認**。現行Sourceの有無と、実外部Harnessがその出力を受領したかは別（今回再実行はNOT RUN）。`source-index`／`knowledge-*`への同様の投影は未確定。
- [TEI #3180](https://github.com/hat47x/tei-platform/issues/3180)の`trace`未対応kind／真正な不存在のエラー分離は統合済[#3208](https://github.com/hat47x/tei-platform/pull/3208)を参照。旧Runの`SUBJECT_NOT_FOUND`に新APIの解釈を遡及適用しない。
- [Studio Host #3061](https://github.com/hat47x/tei-platform/issues/3061)では現mainのroot同一性検査（`os.SameFile`）と固定旧Host負例を分離している。これはCC-15の「単一file check-openの原子性」やCC-14の「共同epoch」を製品実装が保証したという根拠ではない。

### 18.4 次の実施順序／判定停止線

1. **P0・限定実行票を固定**：CC-16のexact Test commit `4abdc3d0`はFocused job `128442144943`で合格（詳細§18.1）。`go test`の名前付きpass/fail/実行なしは親試験内で確認。全体Healthの`gofmt`／索引問題を別軸で確認する。強制的なTest数0をPASS表示しない。
2. **P1・Design Context全Required閉包**：#3284の`qualifiedTestGeneration=false`から始め、P0-M／P0-Rの実Source・Function/Business/Rule依存、Providerの実Read、C0/C1観測を個別に充足する。部分Scopeの局所テスト・探索を禁止するのでなく、**全体の正答証明と区別して続行**する。
3. **P2・追加認知価値**：決定的なD0（現在Tool/Sourceから不足を列挙）とSEI MethodのM1（どの追加Read/Scopeを選ぶか）を、同じSource/Read予算/Grant/開示とblind Expectedで比較。誤不存在、誤ったTest Admission、未取得依存、追加Read費用、未知Caseの発見を別指標にする。

**現時点の範囲**：CC-16は合成Go試験のexact commitで限定CI PASS、製品コード無変更。本PRで新たな外部Harness実比較、Provider／実DB、OS/Network拒否、実AI生成テスト、Owner正式受理はNOT RUN。その後のdocs変更を含む現在PR HEADのCI、Repo全体Health、実Consumer受入はそれぞれ別に検収する。

<!-- tei-devctx-cc17-test-revision-and-method-gap-20261108 -->
## 19. CC-17：Test Source変更とA→B→Aを実Run資格から分離し、Methodの次Probeに接続（2026-11-08）

### 19.1 今回の横断根拠と非重複責務

[統合済#3284](https://github.com/hat47x/tei-platform/pull/3284)と[統合済#3296](https://github.com/hat47x/tei-platform/pull/3296)により、既存`tools/reference-issue-test-ability/context_gate.py`は、実TEI read-only CLIの9応答を使った範囲限定評価と、**合成16種**の欠損・重複・切断等を検出する。依然`boundedReference=partial`／`qualifiedTestGeneration=false`であり、**受領済みJSONの一時コピーへの破損注入**を実外部Harnessからの完全受領に読み替えない。

[統合済#3304](https://github.com/hat47x/tei-platform/pull/3304)は、既知の「line_number_positive」に対する行番号0負例を固定Go Scenario経由の実TEI CLIで検出した**T1-A**。未知のRule/必要ObligationをAIが新たに見つける**T1-B**とは異なる。[旧Draft #3324・closed/unmerged](https://github.com/hat47x/tei-platform/pull/3324)と[旧Draft #3326・closed/unmerged](https://github.com/hat47x/tei-platform/pull/3326)は、同じWorld Fingerprintの下でRequired Rule/Relation欠落と`source-index` mode違いによるFalse Missingを切り分ける別案であり、どちらも現時点で未統合。ここで第二Source Index Readerや第二Readiness Gateを作らない。

[Draft #3303](https://github.com/hat47x/tei-platform/pull/3303)の三視点Data/Function/Business・CRUDと実Product Contextの結合は未実施で、DiagramにないEdge／表にないRuleを「実モデルでも不存在」としない。Methodが行えるのは**取得・現在性・開示資格の不足を分けて、次に確認すべき実Tool/Sourceを提案すること**であり、未取得Sourceの意味を機械的に受理することではない。

### 19.2 本PRで追加した新しい限定反例：CC-17

`tools/reference-developer-context/test_revision_characterization_test.go`の`TestDiscoveryProfileCC17CandidateChangeInvalidatesOldPass`を追加（Code保存commit `2556d5b7fa91611b5113d746e6fb1b9c429cdacb`）。公開合成Go Moduleの**target file、Test名、GOOS、選択Go TestGoFilesを固定したまま**、候補Test Sourceとその独立期待だけを変更する。

| 制御したEpoch | 実Source | 候補Test Source | 期待する実`go test -json` | 禁止する説明 |
| --- | --- | --- | --- | --- |
| E0 | `Evaluate()=0` | TestObligationは`0`を要求（A） | named test `pass` | 過去PASSを永久有効と呼ぶ |
| E1 | 同じSource byte/SHA | 同名Testが`1`を要求（B） | named test `run`＋`fail` | GOOS/target SHA/Test名一致だからE0のPASSを現在結果とする |
| E2 | 同じSource byte/SHA | Aへbyte復元 | **新しい**実runで`pass` | SHAがAに戻っただけでE0の旧実行Receiptを現在の実行として再認定 |

`Discover`のcandidate SHAはE0→E1で変わりE2で復帰。test sourceだけ更新しても**同一target SHAと同一Go Profile選択が実検証結果を固定しない**。個別RunのeventやCode digestは実際に観測するが、試行ID・実Build byte・全依存・権限の有効期間までを正式束縛するReceiptは現`Report`にない。A→B→Aは「現在file byteが再び同一」と「以前の実行が現在有効」を分離する反例であって、単独で再実行の全領域一律強制を決めない。

**限定CIで実証**：テストSource commit `2556d5b7fa91611b5113d746e6fb1b9c429cdacb`の[Focused Actions #42388764022](https://github.com/hat47x/tei-platform/actions/runs/42388764022)、job `128451686216`は**completed/success**。実Jobログに`TestDiscoveryProfileCC17CandidateChangeInvalidatesOldPass` **PASS（0.60s）**、CC-12〜16も同じジョブでPASS（SKIPでない）が記録されている。E0/E1/E2は試験本体で実`go test -json`のnamed Eventとexitを照合。**合成Goパッケージ限定のCharacterization**であり、TEI業務Oracle、実Provider、外部Harness、AIによる未知Rule発見は未実証。後続Docs commitやPR仮統合HEADのRepo Healthはこのrunでは検収していない。

### 19.3 再資格化を目的相対の検査に限定する

| 差分 | 認知Methodの正しい不足表現 | Next ProbeとOwner |
| --- | --- | --- |
| Candidate Test変更、target SHA不変 | 「候補は同じ名前、過去PASSの現在適用未確認」 | 実Build/Profile/Test Source byte＋`go test -json`の当該Eventを再束縛（本#2533、検証基準は#2514） |
| `go test` exit0だがnamed Test 0件 | 「Run可能だが、この義務の実行証人なし」 | 必要Test/Scenario IDと選択Profileを確認（CC-16/#3262） |
| Candidate A→B→A、Current bytes一致 | 「過去AttemptのReceiptと現在の有効期限・依存は独立」 | 既知依存だけ選択再確認、追加Edge/Grantはunknown（TAI #271／#2516） |
| FPは同じだが必要Rule／Relation不明 | 「表示・Catalog上の不在と、Requiredの不存在を混同しない」 | 同一ScopeのSource/index mode/Source Inventory/実Loaded・非出力依存を[#3262](https://github.com/hat47x/tei-platform/issues/3262)・Draft #3324/#3326で比較 |
| Design Context `partial`、16 controlsが検査成功 | 「欠損検査器は動いたが、必要な設計閉包はまだ未取得」 | P-1b1の一Function限定Required、独立Expectedを先に確定。新T1-Bは後段 |
| 実H2/H3のSkill/Tool Response一部到達 | 「Tool/Skillの配送と内容・開示・回答資格は別」 | TAI #168のG0→G1→G2、実非露出／実Result終端（TEI #2509／TAI #232） |

**必要な検査だけを選ぶ**：Qが存在の片側Witnessを求めるのか、全対象の否定／不存在なのか、実Executionの妥当性を求めるのかでRequired closureは異なる。既存Provider/CLIから観測された条件と、Ownerが事前に固定したR(Q)を比較し、`known / partial / unknown / unsupported`を区別する。すべてのReadをやり直すか、疑わしい依存だけを再確認するかは、同じ試行のSource/Revision/Scope/Grant/Costで評価する。Sourceの隠れたDependencyを、現在の部分Reportだけから自動補完しない。

### 19.4 実施順と合格判定

**P0（限定実証完了）**：CC-17のFocused CIをexact Test Source commitとJob logで確認済み。CC-12〜16の過去合格と混ぜず、E0 pass・E1 fail・E2新passを`TestObligation`の実named eventとして照合。PR #3176はDraftのまま、Go製品本体を変更しない。

**P1（次）**：#3262／#3324／#3326のうち、**実Source/必要Ruleの判定可能な部分を既存`context_gate.py`へ受け渡す一案**だけを検収。ScopeやSource Index呼出しmodeによるFalse Missingと、真のSource/Rule欠落の両方を公開合成負例で識別。初回のP-1b1は一Functionに限定し、全Workspace完全性を要求して前段を停止しない。

**P2（Model寄与比較）**：D0（決定的な不足診断）／D1（限定実追加Read）／M1（SEI Methodが選択したProbe）を、同じQ・Candidate/Source/Tool・Grant・Disclosure・Read Budget・Blind Expectedで比較。M1が独立に新しく見つけたRequiredの数、誤不存在、不要な再読取、実RunとContextの現在資格、Cost/Latencyを別々に測る。事前にAIへExpected・Reviewer-only rootを与えない。

**採用と保留**：既存TEI #2533がP2評価とCC-17、#3262がDesign Context/P-1b1、#2514が独立Expected/Mutant、#2586が実Function→Query→Provider、SEI #767が問い相対Method、TAI #168/#232が実外部Harnessと開示・隔離を所有。新Test Engine、第二Reader／中央Registry／追加Cognition Host／形式未規定のReceipt ABIを作らない。**TEI本体の共同Snapshot能力・T1-B・実外部Harnessによる正答比較・実Provider・正式Owner意味受理は今回NOT RUN。**

<!-- tei-devctx-cc18-dependency-and-tt56-probe-20261108 -->
## 20. CC-18：Target/Test不変でも、依存Sourceの変更で旧PASSは使えない（2026-11-08）

### 20.1 他レーンの最新実証を反映した判断

[TEI #3262](https://github.com/hat47x/tei-platform/issues/3262)のTT-56は、二つの公開合成方式の実CI成功を受け、**[PR #3354](https://github.com/hat47x/tei-platform/pull/3354)で単一`context_gate.py`へ統合済み**である。以下の比較表は当時の設計選択の記録であり、#3324/#3326は現在どちらもclosed/unmerged。採用済みの二つの必須Rule正本は存在しない。

| 方式と実証 | 長所・確認された限定事実 | 同時採用してはいけない点 |
| --- | --- | --- |
| [旧Draft #3324・closed/unmerged](https://github.com/hat47x/tei-platform/pull/3324)（HEAD `4f4b114aa826b1a603044c5eef34407f536f186d`、[CI #42385606408](https://github.com/hat47x/tei-platform/actions/runs/42385606408) success） | 一つの独立Required Rule、Relation、`inspect-validation` Canonical属性、Definition Role/Anchor・Loader・FPを比較。従来16＋新8合成負例を拒否 | 1 Ruleのcanonical matchingだけで複数Ruleの全件、Function/Businessの閉包、実Evaluatorの意味受理を主張しない |
| [旧Draft #3326・closed/unmerged](https://github.com/hat47x/tei-platform/pull/3326)（HEAD `c2d2da10cc93022697204e66e83c30248b3810a0`、[CI #42385726717](https://github.com/hat47x/tei-platform/actions/runs/42385726717) success） | 独立必須Validation 3件をCatalog／Loader／既存`source-index`の**indexモード**によるDefinition Referencesと照合。7拒否＋範囲外1非過剰拒否。初回のsubjectモード照合失敗を訂正 | `source-index` subject応答を全Reference Listと誤読しない。3 Ruleが揃ってもFunction→Query→実Providerと全Sourceの共同Snapshotを主張しない |

**収束判断（#3354で実施済み）**：既存の一つの`context_gate.py`を拡張するOwnerは#3262。比較の**基準点は#3326の「問合せに必要なRule集合R(Q)と同じSource Index indexモード」「必要範囲外は止めない」**とし、#3324の**Canonical属性・Relation/Source Anchor検査**を、同じQに必要なものに限り取り込むかを差分比較する。Candidateのファイル数やRule数の多さで優越としない。最初に`R(Q)`、有効な`source-index` mode、Scope、`WorkspaceIdentity/WorldFP/ModelRev/SourceIndexFP`、実Source load、Provider未観測を同じ受入票に固定。Index mode混同から生じる`false-missing`と、対象Ruleが実際にRequired集合に対して欠けている`required-missing`、権限不足による`undisclosed`、未取得`unknown`を**同じ`absent`へ潰さない**。両PRのコードをそのまま連結・二重採用しない。

### 20.2 CC-18：依存の変更で同じDiscover Reportが異なる実Go結果を持つ

[Draft #3176](https://github.com/hat47x/tei-platform/pull/3176)の`tools/reference-developer-context/dependency_revision_characterization_test.go`に、`TestDiscoveryProfileCC18UnreportedSourceDependencyChangesTestOutcome`を追加（保存commit `31a317f506a750658efa56710319363aa72e68da`）。

合成Go Moduleの`qualification.go`が`rules.go`の`requiredRule()`を呼び、`qualification_test.go`は`Evaluate()==0`を要求する。**TargetとTestはどのEpochでも一byteも変えない**。Go自身の`go list -json`に`GoFiles=[qualification.go, rules.go]`が現れることを確認する一方、現`Discover`はtarget SHAと`qualification_test.go`候補SHAしか報告しない。

| Epoch | `rules.go` | `Discover`のTarget/候補情報 | `go test -json -count=1`期待 |
| --- | --- | --- | --- |
| E0 | `requiredRule()=0` | 基準Report R0 | 実`TestRequiredRule=pass` |
| E1 | `requiredRule()=1` | **R0と構造・値とも同一** | 実`TestRequiredRule=run+fail` |
| E2 | `requiredRule()=0`（A→B→A） | **R0と再び同一** | **新しい実run**で`pass` |

従って、**Target SHA／Test SHA／Profile／候補List／Reportが全て同じでも、必要依存の実byteが変わればTest結果が変わる**。CC-17のTest Source byte変更とは独立の負例。これは、TEIのRule/Relation／Function・Queryの見えない依存がChange Impactや実Test Admissionを決め得ることを、より小さいGoパッケージで実証する位置付けである。Goの`GoFiles`一覧は本Packageに限る。Go Toolchain/外部Imports／Plugin／実Target Binary／Provider／実Run Receiptの完全閉包は取得していない。

**限定CI結果（exact Test Source commit）**：`31a317f506a750658efa56710319363aa72e68da`に対する[Focused Actions #42391826601](https://github.com/hat47x/tei-platform/actions/runs/42391826601)、job `128460814122`は**completed/success**。実ジョブログで`TestDiscoveryProfileCC18UnreportedSourceDependencyChangesTestOutcome` **PASS（0.37秒）**、CC-12〜17も同じジョブでPASS（SKIPではない）を確認した。CC-18のE0/E1/E2は試験内部で実Goの`go test -json` named eventとexitを照合。**公開合成Go package限定**。実製品`Discover`のDependency追跡機能を追加・検証したわけではなく、現`Report`が必要Dependency閉包を表さない反例である。後続のdocs更新を含むPR HEADの全体HealthはこのCIと別検収。

### 20.3 認知Methodが選ぶべき「次のProbe」を狭く定義する

TEI #3262の不足は単純な`Read more`ではない。**どの問いQの必要集合R(Q)が足りず、取得したTool Replyが実際にどのMode・権限・Source Revisionで得られたか**によって、試すべき次Actionが変わる。

| 初期観測 | D0（新Readなしの正しい診断） | D1（固定手順の追加Read） | M1（SEI Methodの選択効果を検証する候補） |
| --- | --- | --- | --- |
| Source IndexがsubjectモードでDefinition一覧を含まない | `unsupported-evidence-for-completeness`。Rule不存在とは答えない | 同じ既存CLIのindexモードで必要Ruleの定義参照を確認 | 誤った`absent`よりindex-mode再Probeを優先できるか |
| indexモードで必要Rule IDが欠落、Required Source loadが不明 | `required-unresolved`、Source/Scopeが未知 | Workspace Inventory／Loader、Catalog/Inspect／Definition Roleを別取得 | Source未ロード・Rule未定義・非開示を切り分け、1回Budgetで必要なProbeを選べるか |
| Target/Testが不変で、Testの過去実行だけある | `historical-pass-not-current-qualified` | Go Profile/Dependency Read Setと現Sourceを確認し、必要なら新`go test` Event取得 | 変更が当該Obligationに影響するかを推定し、不要な再実行を抑制できるか |
| Ruleはあるが実Function→Provider取得・End-to-end Effect未確認 | `bounded-source-known / execution-unknown` | 実Product Tool/Providerの限定ReadとAttempt/Effect/Grantを取得 | Rule Sourceの再確認ではなく実Consumer/Providerの不足にProbeを向けられるか |
| 全件否定を求めるがTool Replyが途中切断 | `unknown`（存在証人と不存在証明を非対称に扱う） | 正規のPage/終端・Scope/Grantを検証 | 片側の存在Witnessだけで足りる問いに過剰な全件取得を要求しないか |

**比較設計**：公開合成Qを2種（必要Ruleの定義到達、変更された依存に対する実Test資格）、隠した独立Expectedをそれぞれ別に固定。D0、同一決定的予算のD1、SEI M1の提案を**同じ元World/Tool profile／Read Budget=1・2／現在Grant/Disclosure**で評価する。M1がD1と同じProbeだけを返した場合はMethodの純増効果と数えない。出力は正答／必要根拠到達／誤不存在／無駄Read／コストを別表で記録。実Toolを呼べないMethodは`proposed`、Toolを呼んだだけなら`invoked`、本当に必要Sourceが利用可能なら`evidence-received`として区別し、未実行をPASSとしない。

### 20.4 他チャットとの整合と次の受入

- [merged #3299](https://github.com/hat47x/tei-platform/pull/3299)の実Source/Plugin Lock改変後に古いScenarioを拒否する限定Go+CLI試験、[merged #3325](https://github.com/hat47x/tei-platform/pull/3325)のOperation Drift `current/stale/unchecked` を再利用する。ただし**Content PinがAへ戻ったこと**≠古いAttemptの現在Receipt、Operation比較成功≠実Execution Grant。
- [merged #3304](https://github.com/hat47x/tei-platform/pull/3304)は**既知Ruleの0負例を実CLIで実証したT1-A**であり、AI未知Obligation発見のT1-BはNOT RUN。#3262 P-1b1がRule/Relation/Sourceの必要集合を閉じても、独立ExpectedをModelに見せないTest Generation、実Mutant、C1/NFR、Owner受理は後段。
- P0-M（三視点/CRUD）のDiagram Edge、P0-R（Function→Query→実Provider→表現）の実行Edge、P0-C（同一Source/Scope/Revision/Consumer）の一致を一つの図で表現しても、それだけで同じTime/GrantのActual Executionとはしない。[TEI #2516](https://github.com/hat47x/tei-platform/issues/2516)／[#2586](https://github.com/hat47x/tei-platform/issues/2586)が現在性・ProviderのOwner。
- [SEI #767](https://github.com/hat47x/sei-cognition/issues/767)はQごとのUnknown/次Probe選択、[TEI #2533](https://github.com/hat47x/tei-platform/issues/2533)はCC-18とP2比較、[#3262](https://github.com/hat47x/tei-platform/issues/3262)は既存Context Gateへの方式収束、[#2514](https://github.com/hat47x/tei-platform/issues/2514)は独立Expected/Mutantを所有する。

**進行ゲート**：①CC-18の実Go Focused CIは上記exact Test Source commitで検証済み。後続HEADのHealthを別検収、②TT-56は#3354で一本化済み。次に実11 CLI/32 Controlsの限定結果から、必要な一Functionの`R(Q)`を独立に定めて実Reader/Parser/Evaluatorを照合する、③製品の実Loader→Parser→Evaluatorへ一Ruleを通す、④外部MethodのTool受領・独立採点を別段階で検収。新しいIndex Reader、Source Parser、Proof Store、中央認知Host、Runtime Authorityを追加しない。実H2/H3適格比較・実OS隔離・実Provider/DB・AI盲検T1-B・Owner Accepted Meaningは今回NOT RUN。

<!-- tei-devctx-q-relative-admission-maturity-20261108 -->
## 21. P2継続：実Context・局所Test Witness・問合せ相対の不足を単一Evidenceへ誤集約しない（2026-11-08）

### 21.1 横断コード・CIの最新到達点（各Ownerの正本を保持）

| Owner／確定したSource | この計画が受け取る**実測済みの狭い事実** | 境界・まだ証明しないこと |
| --- | --- | --- |
| [TEI merged #3354](https://github.com/hat47x/tei-platform/pull/3354)／[CI #42391707962](https://github.com/hat47x/tei-platform/actions/runs/42391707962) | 既存`tools/reference-issue-test-ability/context_gate.py`にTT-56二案を集約。**11種の実TEI CLI read-only Reply**と**32個の合成Consumer Controls**を同じ専用CIで検証（7+2+7+8+8）。#3324/#3326はclosed/unmerged | 32は製品コードMutant数・全RuleのIndependent Expected数ではない。連続11 Readは原子的Snapshotでもない。`boundedReference=partial`／`qualifiedTestGeneration=false`を維持 |
| [TEI Draft #3365](https://github.com/hat47x/tei-platform/pull/3365)／[CI #42394673357](https://github.com/hat47x/tei-platform/actions/runs/42394673357) | 既存`probe.py`にread-only `scoped_step_coverage`を追加。**実Pinned Scenario 6 Step**を基に、受領目録をコピーして7対照（TT-105/106、終端欠落/偽終端、Source mismatch、重複／未検証Receipt）を判別。exact HEADの専用dogfood/Health/脆弱性CIはcompleted/success | `end_received`は試験用受領目録への**呼出側提供**。実H2/H3 Toolストリームのend-to-end終端証明ではない。6 Step局所`localObserved`を全Product Coverageや不存在へ昇格しない。Draft/未merge |
| [TEI Draft #3176](https://github.com/hat47x/tei-platform/pull/3176)／[CI #42392143845](https://github.com/hat47x/tei-platform/actions/runs/42392143845) | CC-12〜18のGo構成／Source Cohort／Path-Open／実Go Test Event／Test Code・Dependency ABAの限定Go試験と、前HEADの専用/Health/依存CI3件success | 自作合成Go Module。TEI Product Readerや実AI Methodを実行していない。Draft/未merge。現Plan改訂を含む**新HEADのCIは別検収** |
| [TEI merged #3304](https://github.com/hat47x/tei-platform/pull/3304)／[CI #42378033965](https://github.com/hat47x/tei-platform/actions/runs/42378033965) | 事前に人が定めた既知`line_number=0`条件の実Pinned CLI判別＝**T1-A** | 問いから未知Ruleを生成AIが発見した**T1-B**とは分離 |
| [TEI Draft #3356](https://github.com/hat47x/tei-platform/pull/3356) | 現CLIのread-only調査・Command Catalog（19コマンド）とMCP入口を照合し、**MCP実行入口を確認できない**ことを記録 | MCPを実Callした観測はNOT RUN。TEI独自MCPが必須というArchitecture変更を要求しない。**CLI/Tool wrapperを実調査経路の基本**とする |
| [TEI merged #3333](https://github.com/hat47x/tei-platform/pull/3333) | `tei commands`の`view.writeFlags`で`-operation-log`/`-lifecycle-state`の名指し書込を機械可読にした | コマンドの`effect:read-only`はOS全域無書込・全送出拒否を保証しない。実argv／Host・Network観測は別資格 |

### 21.2 P2の中心判断：十分性は単一の総合PASSでなく、Q相対の「証拠束」で判定

`Q`（実際の問い・主張）の成立を評価するとき、下の各行について**観測／未観測／拒否**を独立に持つ。これはこの評価計画の記入欄であって、TEI Coreに新しい共通Type/Proof Engineを実装する提案ではない。

| 論点 | 最小Witness | Witnessが無いとき言ってはいけないこと |
| --- | --- | --- |
| **S：Source/Identity** | Workspace manifest、Source path/role、Revision/実Read bytesと必要R(Q)の一致。SourceとReferenceの移動、A→B→Aに区別が要る | Fingerprint一致＝同一Qの必要依存全件一致 |
| **T：Transport/Completeness** | 実Tool Request/Resultの対応、byte/page・終端・切断とConsumerの実受領 | JSON valid／終端をFixtureがtrueに設定＝外部受領完了 |
| **R：Required/Meaning** | Qに必要なData/Function/Business/Rule/Relationと、独立に固定したRequired集合。Loader/Parser/Canonical/非出力Predicateを区別 | Catalog上の未表示や図/CRUD上の空欄＝要素不存在・Meaning受理 |
| **X：Test/Execution** | Go Profile、実選択Test/Source依存、実Binary/Target、個別named Event、Scenario Version、Expected、現Attempt | `go test` exit0／旧Receipt／テスト名一致＝現在の機能/NFRカバレッジ |
| **A：Authority/Disclosure/Effect** | Consumerの用途、現在Host Grant・実argv/書込先、実Provider/OS/Egress・公開可能なResult | `read-only`宣言／Skill Load／実CLI成功＝無副作用／現在認可／外部開示適格 |

**非対称性**：存在については、Qに適合する単一の現在Source Witnessで足りる場合がある。しかし「不存在」「未テスト」「必要なRuleがない」といった全称否定は、Q相対の対象Universe・Mode・実Read範囲・終端・権限・必要依存の閉包が別途必要。**`notObserved`を`untestedConfirmed`へ置換しない**。同時に、別の既確認Stepの`localObserved`を、不足した他Stepのせいで無条件に破棄しない。現在性やSource mismatchで当該Receipt自体が不適格になった場合に限ってその局所Witnessも無資格化する。

### 21.3 D0→D1→M1の実比較を「次Probeの識別能力」に絞る

現在の[SEI #767](https://github.com/hat47x/sei-cognition/issues/767)には、公開合成の不足根拠・残差と次Probe候補がある。しかし**実SEI Methodによる新CLI Tool受領→独立Expected比較**は未実施。実験前に次を固定し、探索段階と採点段階を分ける。

| 固定ケースとQ | 参加者へ公開する現在の状況 | 隠して独立に固定する真相／採点上の差 |
| --- | --- | --- |
| **E1: Rule absent?** | subjectモードIndexから見えない必要Rule、Catalog/Source read可否未確定 | 実Ruleは存在。最適追加Probeは同CLIのindexモードを再取得。誤`absent`は失点。indexモードの結果受領はTool evidenceで確認 |
| **E2: Required missing?** | Qの独立Required 3 Rule中、部分Replyで1 Ruleだけ未表示 | (a)正しいindex-mode/必要Scopeでも実欠落、(b)単に一時Replyが切断、を双子Fixtureにする。M1はabsenceとunknownを識別する必要条件を問えるか |
| **E3: Test passed?** | Target/Test/report/Go Profileは同一、過去`TestRequiredRule` PASS | 隠した変更が`rules.go`にあり、現在の実RunはFAIL（CC-18）。Go selected source／依存読取または実named Eventが必要 |
| **E4: Is step untested?** | 6-step Pinned Scenarioの部分受領目録で対象Stepは`notObserved`、他Stepはverified | 実Scenario全件目録の有無を別に確認。`untestedConfirmed`と断定しないが、既観測Stepの局所肯定を維持（TT-105/106） |
| **E5: Did read-only guarantee no writes?** | `tei commands`で`view`の旧`effect`表示と`writeFlags`を閲覧可能 | 実argvのflagと出力先を照合する必要がある。Flag以外のOS/Network効果やCurrent Grantは別。Skill静的記述だけで採点しない |

**比較条件**：D0=同じ初期Tool Resultだけで診断する決定的Baseline、D1=同じ許可済みCLIから固定優先順で1追加Probe、M1=SEI Methodによる1追加Probeの提案。Budget 1/2は別実験。全員が同じ`Q`、公開Source/Revision、Tool allowlist/mode、Grant/Disclosure、初期Result byte、受取可能な範囲を持つ。Hidden R(Q)・独立Expected・negative pairの区別と評価者専用SourceはParticipantから物理的に分離する。**M1の提案と、Harness経由で実Toolが動き、回答byteが届いたことは別スコア**。M1がD1と同じProbeを選んだ場合に、追加のMethod能力を認めない。

採点票の列は`probe proposed`／`actual tool invoked`／`received complete`／`required witness attained`／`correct current claim`／`false absence`／`unnecessary reads`／`cost`。定量改善は独立Expectedの照合後のみ。Tool到達しない模擬選択、実Methodが動かない状態、Read budget外の再調査を成功に計上しない。実外部H2/H3の適格比較は[TAI #168](https://github.com/hat47x/tai-pleroma/issues/168)がOwner。TEIのLocal CI合格をHarnessの実OS隔離・実Tool受領証明へ昇格させない。

### 21.4 統合計画と担当への差分

1. **P0（完了した限定実測を固定）**：CC-12〜18／TT-56単一Consumer #3354の各exact Run・Source・限定結果、TT-105/106 Draft #3365の7合成ControlをPlan上で区別。旧二案選定と「まだTT-56は2 Draftのみ」という作業を**取り消す**。#3365はCI成功だが**Draftのまま**。
2. **P1（次のProduct適格化）**：#3262のDesign Contextから、一つのFunctionの独立Required／実Source Reader・Parser／Validation/Relation／実Scenario/Go Targetを`R(Q)`で接続。#3356の当時のMCP入口未確認をCLI停止理由にしない（その後main #3458で参照MCP AdapterのCLI同内容・限定Go試験は成立。ただし外部MCP Host/Grant/Tool送出は未検証）。Source Loaded≠Meaning Accepted、11 Reply≠共同Epoch、6 Test Step≠全体C1を保持。
3. **P2（実SEI Method効果）**：Q相対のD0/D1/M1実Probe比較を、独立Expected、同じTool/Grant/Read Budgetと実Receiptで実行。外部AI/LLMへHidden Expectedが渡らないことの実際の境界検証を先行し、空の記録を不在とみなさない。
4. **P3（後続）**：T1-A既知Ruleを超えるT1-B盲検の新Obligation検出、実Function→Query→Provider→Context、Pluginの宣言→選択→実Load/Invocation、三視点図面/CRUDのProduct Consumer、実Operation/Host/OS/Effect、Full Coverageは各Ownerへ。複数のDraftで同一正式Gate/読取器を競合実装しない。

**現在の停止線**：本§21は既存コード・CI・Source照合から成熟した評価手順を計画へ移すDocs作業。**実M1 Methodを外部Harnessで走らせた結果、実E1〜E5のBlind Score、実Product Provider全閉包、C1、Owner Meaning/Authority受理はNOT RUN。** 新しい中央Meaning Store、認知Host、Test Engine、Product Schema、Permission Engineは導入しない。

<!-- tei-devctx-canvas-item-provenance-cc19-21-20261108 -->
## 22. P2認知空間ブリッジ：未整理Itemと複数Canvasの「見える範囲」をSource全件性に昇格しない（2026-11-08、横断設計・NOT RUN）

### 22.1 成熟した外部要件を、#2533の既存Evidence境界へ接合する

2026-11-07のCanvas/情報断片の横断議論では、①**巨大Canvasを関連を維持して分割**、②**同じItemを複数Canvasへ配置**、③**未分類・未読のRaw ItemをInboxに保存しReaderで直接開く**、④**Task/EventはItemの意味Roleまたは別Subject、Action/Effectは実行契約**、⑤**構造・意味・表現・実行を分離、Owner受理のない意味を自動確定しない**ことが成熟した設計要件となった。

本節はこの要件を、開発文脈探索とSEI Methodの**Claim相対の観測十分性**に翻訳する。CanvasをTEI Source/Meaningの新正本とせず、既存Source/Reference/Revision、標準Data Topology/Role/Trait、Function内Presentationと、用途相対Construction Contextの**表示・配置側のConsumer**として扱う。Canvas/Item製品実装や実Readerの完成を意味しない。

- **Item Source/Identity**：Raw本文／Blob／来歴／版を保持できるが、保存済み・索引済み・Readerで実読済み・意味確定済み・現在開示可能はそれぞれ別。未整理Itemをカード化・Task化しない運用を許容する。
- **Semantic Subject/Role**：必要時にData/Function/Business/Task/Event/その他のRoleを提案。Role CandidateはOwnerが受理したMeaningではない。同じ原典Task上のEventと実Function LaunchEventを同一視せず、Actionへの変換は型付き契約・Host Grant・実Invocationを要する。
- **Canvas/Occurrence**：CanvasはItemを所有せず、同一Itemの表示Occurrence、Group/Tree/Relation/座標を配置する。分割/統合/再配置は原則Viewの変更であり、Source IDや意味Relationを自動複製・削除しない。一方、ユーザーが意味関係自体を編集した場合は別Revision/Owner受理。
- **Representation/Projection**：カード、表、グラフ、SHO/Markdown、AI向けJSONは元SourceとTopologyに対する非単射であり得る。表示された0件・空リンク・同一本文はSource不存在、関係不存在、元ID同一、逆編集資格の証明ではない。
- **Probe/Delivery/Effect**：SEIが必要な次Reader/索引/関係解決を提案しても、実Tool・Scope/Grant・実Read/受領がない限り取得完了にしない。Tool一覧・表示・文書中のAction語・Trait宣言は実Effectの権限ではない。

### 22.2 新しい反例（CC-19〜21、**製品試験NOT RUN**）

| 対照・問い Q | 二つの世界と同じ観測 | 独立Expectedで区別すべき結論 |
| --- | --- | --- |
| **CC-19 未整理Itemの不可視性**：「この変更に関連する資料は存在しないか」 | W0＝対象Sourceに関連するRaw Itemなし。W1＝Inboxに非カード化・未読の関連Raw Itemが1件ある。どちらもCanvas Card 0／閲覧したSource 0／既存Discover候補0の投影を与え得る | W1の存在はReader/索引の実Readが必要。初期投影だけから不存在を主張しない。Raw Itemの存在が判明しても本文解釈／関連Meaningは別 |
| **CC-20 分割・複数Occurrence**：「独立した2件の根拠があるか」 | W2＝1 Item IをCanvas A/Bに各1 Occurrence配置。W3＝独立Item I/Jの同じ文章をそれぞれCanvas A/Bに配置。二つの見えるカードと同じ文章を持ち得る | W2の原典支持数は1、W3は2候補。ただし同じ一次Source由来なら証拠独立性はさらに別。Canvas分割で元ID・関係edge・来歴を失わない |
| **CC-21 RoleとActionの誤昇格**：「このTaskは実行・完了したか」 | W4＝Raw Itemに「Task/Action/Event/実行済み」とあるだけ。W5＝受理済みRole＋実Tool/Attempt/Receipt/現在対象の独立Readを持つ。両者の表示カード見出しを同じにできる | W4は文書・候補Roleに留まる。W5もReceiptだけで実業務Outcomeを断定せず、実EffectとOwner/現在Grantを別判定 |

**既存実証との区別**：[TEI #2497](https://github.com/hat47x/tei-platform/issues/2497)のSUI実Productを使う非単射Projection負例（SUI #3396/#3397、SHO #21の別OS・実Reader限定対照）は、**Viewから元Source/Topologyを逆復元できない**ことへの先行証拠。ただし上記CC-19〜21のTEI Canvas/Item Product実装、閲覧/保存・分割・Plugin Runtime・Consumer E2Eの実証ではない。[Draft #3176](https://github.com/hat47x/tei-platform/pull/3176)のCC-12〜18実Goは**Go Source/Profile/Test/依存**を対象とし、Canvasの製品受入にはそのまま転用しない。

### 22.3 初回受入票：保持・意味・配置・実行を同じ真偽値にしない

これは#2533の**評価票の列**であり、新しいCore Schema/Item Role Registry/共通Activity DBではない。

| 層 | 既存のSource/Consumerから観測すべき値 | 不足時の表現 |
| --- | --- | --- |
| Raw保持・読取 | Item/Source ID、実Content bytes/digest、Revision、Viewer/Reader実Read、失敗・未読・未整理 | `stored-unread`／`unknown`／`not-permitted`（存在自体が秘匿なら公開しない） |
| 認知/意味 | Q、独立Required R(Q)、候補Role/出典位置、必要Facet/Relation、Owner Accepted範囲 | `candidate`／`partial`／`unaccepted` |
| 配置/Topology | Canvas ID、Occurrence→Item参照、Graph edgeの固有IDと向き・多重度、分割前後の対応、表示に落ちた情報 | `view-only`／`projection-loss`／`unresolved-reference` |
| 取得/配送 | 実Tool/ReaderとScope・Grant・Request/Result byte・終端・実Consumer受信・Disclosure | `proposed`／`unsupported`／`received-partial`／`withheld` |
| Action/Effect | Roleの型付きExecution Contract、実Invocation/Host Grant、Receipt、独立Resource Read、現在適用 | `not-invoked`／`effect-unknown`／`not-current-qualified` |

**可視範囲の原則**：内部ReaderがRaw Itemの存在を知っていても、公開ConsumerにItem数・path・Tool候補を見せられるとは限らない。公開二世界で可視Projectionが同じなら、公開面の返答から秘密のItem/Canvas構造を推測できたと扱わない。実Producer→Renderer→Consumer／OS非干渉を未確認のまま「非漏えい合格」と言わない（#2516/#2515）。

### 22.4 D0/D1/M1への接続と採用順序

**初期Qを2種に限定**：Q1「見えないRaw Itemを含め、関連資料が存在するか」、Q2「同じ資料が複数Canvasに表示されているのか、独立した根拠が複数あるのか」。CC-19/20の二世界Expectedを**評価者側で先に固定**し、各参加条件には同じ許可済みItem一覧・Canvas Projection・Tool Capability/Grant/Read Budgetと元のSource版を渡す。隠したRaw内容、元Itemの答、Reviewer-only Sourceは渡さない。

- **D0**：Canvasの可視カードだけを使った「不存在」「独立2件」の断定を拒否し、未読ItemとOccurrenceの残余を明示する。追加Readを行わない。
- **D1**：既存の許可されたReader/索引Toolで「Item原典ID/生Reader」を優先する決定的Probeを1件選択。実入口が存在しないなら `unsupported` で終える。Layoutの再表示はRaw原典を新しく読んだことにはならない。
- **M1**：SEI Methodが**同じ情報・Tool/Grant/予算**で1件のProbeを選び、D1に比べて必要Witness到達・誤不存在/誤独立数・不要Read/CPU/latencyを改善するか測る。Modelが既知のItem ID/非公開pathを推測して正答したように見えるTrialを成功に含めない。
- **先行の現実証拠**：TEI CLI/Sourceの構成・必要Rule・実Step/実Tool受信を先に持ち、Canvas実Reader未接続なら上記CCを**計画と公開合成Oracleだけ**に留める。実Canvas ProductのTest/Hostを作ったと偽称しない。
- **後続のLoss評価**：同一ItemをInbox/Reader/Canvas A+B/SHO/AIへ移した場合、Q相対のSource ID/Relation/Meaning/Hidden Scopeが残るかを投影と元Sourceの独立Readで比較する。表面同一は逆Writeの許可にならない。

### 22.5 Ownerと停止条件

- **TEI #2533**：このCCの認知評価・D0/D1/M1比較、**TEI #2497/#2620/#2619**：元Data/Role/Topologyと非単射Projectionの構造的Owner、**TEI #2514**：独立Expected・同一一次Sourceの重複基準、**TEI #2516**：Consumer別開示・Source/Identity/Revisionの現在資格。
- **SEI #767**：問いに応じた不足把握と次Probe選択、**TEI #2605/TAI #168/#271**：現Product Skill・Toolの実入口／Session継続、**#2515/TAI #232**：実OS/Networkの権限・受信完全性。Action/Plugin Effectは既存TEI Host/Provider Ownerで扱う。
- **停止条件**：見えない＝存在しない、複数カード＝独立根拠複数、Canvas分割＝Itemの複製、未整理Item＝自動Role受理、Task/Event文字列＝実Action指示、Renderer投影の一致＝元Sourceの逆操作資格、のいずれも禁止する。Productで未実証のReader・Parser・Trait合成・Permission・外部Skill受領・実Actionを、この計画の形式適合からPASSへ昇格しない。

**状態**：本節は他チャットの成熟要件をP2の反証可能な計画へ翻訳した**設計更新のみ**。CC-19/20/21、Canvas実Reader・分割/結合、実D0/D1/M1、実Owner Meaning/Grantの試験は**NOT RUN**。既存#3176のCC-12〜18実Go限定証拠と区別し、新しい共通Cognition Host、第二Source正本、Universal Item Type/Policy Engine/Gateを追加しない。

<!-- tei-devctx-cc22-24-projection-hash-cognitive-history-20261108 -->
## 23. 問い相対の原典同一性・理解の変化を公開Projection／ハッシュから誤推定しない（2026-11-08、横断計画）

### 23.1 最新の実証と設計上の未決

- **実SUI Productの限定PASS**：[merged SUI #3396](https://github.com/hat47x/sui-sensemaking/pull/3396) HEAD a7893f9c4114d140dc19c7ff2f565693c5b1b29b、[frontend CI #42345896949](https://github.com/hat47x/sui-sensemaking/actions/runs/42345896949)。実buildContextProjectionの試験**3/3 PASS**。元Card.meta.sourceやIsland所属・座標を変えても公開Projection全体が同じになる公開合成二世界を実確認。可視本文・レビュー済み島題名の変更は区別可能。これはProjectionの**非単射性**の限定証拠で、元Raw Reader・逆Patch拒否・SHO/SEI実受領を証明しない。
- **SUIの一次価値に対する追加の問い**：[SUI #3400](https://github.com/hat47x/sui-sensemaking/issues/3400)は「違和感→関係発見→理解の変化→共有・再開」を人間が後から検討できる体験として評価する**未採用・未実行の計画**。理由を言語化できない保留・異論を失敗とせず、後から記した理由を当時の発話や判断にすり替えない。SUIの既存ADR-0089/CVIのOwner判断を置換しない。
- **公開ハッシュの比較規則は未決**：[SUI #3389](https://github.com/hat47x/sui-sensemaking/issues/3389)はcontext-projection.v1のbundleHashとlocaleCompare／関係等の配列順の依存を既存Ownerとして調査中。IDの混在ケース・relation type Tie Break・他言語での固定ベクトル／旧保存値の版対応が未決。実ロケール横断での具体的Hash差はここで実Runしていない。
- **TEI側の実証境界**：PR #3176のCC-12〜18は実Go Source/Profile/Test/Source Cohortであり、上記SUI実ProductからTEIへRaw Readerが配送された証拠ではない。前§22 CC-19〜21は未実施の設計反例。

### 23.2 CC-22：公開Projection一致・公開Hash一致はSource/Topology/Meaning一致の証拠ではない

**Q22：二つの公開表示は同じ原典と同じ関係を持ち、出典再確認を省略できるか。**

| 世界 | 見える状態 | 独立Expected |
| --- | --- | --- |
| W22a | 許可されたカード文章、島題名、公開ProjectionとHashがW22bと同一 | Source cue A、元Island所属X |
| W22b | W22aと完全に同じ公開Projection | Source cue C、元Island所属Y |

**判定**：一致する公開Projection/Hashは「公開された表現の同値」以上の結論を保証しない。元Source ID・Item ID・Revision・Graph edge/Island所属・意味同値・独立根拠数・Owner受理・逆Write資格を復元しない。CC-20の複数Occurrenceを同一原典証人の数へ直結しない。

**追加Probe**：Qが出典の同一性なら許可された元Item/Source ReaderのID・位置・Revision、Qが関係なら元Relation/Island/Tree、引用の現在性ならSource/Consumer Grant・開示Scopeを確認する。Product側に実Readerがなければ unsupported/unknown。非公開Source ID・Item数・pathを公開診断から漏らさない。**SUI #3396による非単射のみ実Product PASS**、TEI/SUI同一Attempt Reader/DeliveryとMethod実選択はNOT RUN。

### 23.3 CC-23：現在表示が同じでも「どう理解が変わったか」は異なる

**Q23：何に違和感を覚え、何が再解釈を促し、何が保留・異論として残るか。**

- W23a：Raw資料と最終可視カード本文しか残っておらず、旧見立て・判断の履歴は未観測。
- W23b：現在の同じ可視本文に到達したが、歴史として initial Q → provisional grouping → hold（理由なしも可）→ new source → changed interpretation → remaining dissent が時点付きで実記録される。
- 独立Expectedは各履歴の**実記録と時点**、補足理由の追加時点、旧Source Revision、Reviewer/Ownerの資格を別に凍結する。理由なしの違和感をAIが「本当の理由」で穴埋めしない。後日書いた理由を当時の記録へ遡及改ざんしない。
- 同じ最終Byte/公開Projectionに復帰してもA→B→Aの中間履歴は消えない。TEI CC-17/18の実Go Test/依存ABAとは**現在の表示/一致から過去の実行・判断を再資格化できない**という条件のみ共有し、SUI人間の意味変化をGo Test Receiptと同一物にしない。
- D0は履歴の実Readがなければ history-not-observed と説明。D1/M1は、現在許される履歴/Raw Source Readerで最もQを判別する1 Probeを同条件で選び、実Byte・Scope/時点・Consumer受領後にのみ根拠を更新する。

### 23.4 CC-24：Projection Hashの用途を具体的に制限する

| 対照 | 避ける偽推論 | 検証Owner／状態 |
| --- | --- | --- |
| 同じ公開Projection・異なる元Source | Hash一致＝同一Item/Source/Meaning | SUI #3396の非単射**限定PASS**。実Raw Reader未接続 |
| 同じDocument、異なる元配列順／Relation type | Hash差＝意味変更、Hash固定＝位置/関係保存 | SUI #3389の契約未決。固定テストベクトルと並び順変更の受入はNOT RUN |
| 同じDocument、異なるLocale/実装言語 | Hashは環境差を超えて再計算できると断定 | SUI #3389がcanonical order/Hash versionを決めるまで保留。TEI Go相互検算NOT RUN |
| 同じProjection Hash、欠けたTool Reply | Hashがある＝Consumerに完全配送済み | TAI #168／TEI #2516の実受領・Current Grantは別 |

採用前にSUI Ownerが配列順（ID比較／文書順）、relation Tie Break、canonical JSON/Hash version、保存済みHashの移行を判断する。TEIで別Canonicalizerを作らない。評価票のSource content identity／projection identity／serialized byte identity／実受領／Meaning／Current Grantを混ぜない。

### 23.5 同条件のD0/D1/M1へ接続する最短の検証

1. **P0：既存証拠を固定**。SUI #3396 Product 3/3とTEI #3176 CC-12〜18の限定Goを別根拠に保持。SUI #3400は実ユーザー価値の計画、#3389はHash仕様の未決。未実装Raw ReaderとSUI→TEI Deliveryは未知とする。
2. **P1：一つの元Source/履歴への実Read**。同一Source Revisionと公開Projectionが照合できる実SUI Reader・許可範囲を既存Ownerと確認する。Tool入口なしならunsupportedのまま。新Item/Canvas Schemaや共通DBを先行導入しない。
3. **P2：Q22・Q23のBlind比較**。同じ初期Projection・Q・Tool一覧/Grant・Scope/Read BudgetをD0=不足診断、D1=決定的次Probe、M1=SEI Methodによる次Probeへ渡す。元Raw Sourceと履歴のIndependent Expectedを参加側から物理的に隠す。採点列は proposed／invoked／received complete／Required R(Q) attained／correct current claim／false identity・fabricated history／unnecessary reads・latency・cost。M1がD1と同じProbeを選ぶだけなら追加価値は認めない。
4. **P3：Sensemakingの価値試験はSUI Owner**。SUI #3400のAIなし→候補→再配置→後日再開、異論・違和感保持と新しい気付きの価値は同Issue/ADR-0089でT1/T2を評価。TEI Synthetic CCのPASSをSUI実利用者の価値証拠へ昇格しない。

**担当**：TEI #2533＝問い相対の認知Method比較、#2497＝Projection Loss、#2514＝独立Expected、#2516＝Consumer開示、SEI #767＝次Probe、TAI #168/#271＝実受領/再資格化、SUI #3400＝Sensemakingの価値、SUI #3389＝Hash契約、SUI #3396＝実Projectionの限定Evidence、SHOは文書Readerの実受入Owner。

**停止線／NOT RUN**：元Source/Raw Reader・理解履歴・SUI→TEI/SEI同一Attempt Delivery・実M1/外部Harness・Hash跨言語実検証・Owner Meaning/Grantは未実証。Hash一致＝Source一致、Hash差＝意味変更、現在表示一致＝理由/履歴なし、Probe提案＝実Read、をいずれも認めない。追加の認知Host・Proof Store・Hash Store・Policy Engine・正式Product Gateを作らない。

<!-- tei-devctx-cc25-27-real-byte-handoff-vs-origin-and-write-20261108 -->
## 24. Raw文書の引渡し・同一ファイル・Working Copyを、Source/Meaningの認知資格と分離する（2026-11-08）

### 24.1 前§23から成熟した新しい実測：Reader・Writer候補はあるが「同一実Producer→Consumer」は未成立

この節は、§22のRaw Item/Canvas、§23の非単射Projection/ハッシュと、新たな**実ファイルへの読取・作業コピー書込**の限定実証を接続する。なお、製品間に新しい統合Reader/Writer/Transportや別のSource正本を設計するものではない。

| Source/Owner | 今回受け取る狭い実測事実 | 依然として未成立のこと |
| --- | --- | --- |
| [SUI merged #3396](https://github.com/hat47x/sui-sensemaking/pull/3396)、[CI #42345896949](https://github.com/hat47x/sui-sensemaking/actions/runs/42345896949) | 異なる元Source cue/Island所属でも**同じ公開Projection全体**を得る実Product 3/3 PASS | 元Source Reader・意味同値・逆Write・外部配送 |
| [SUI Draft #3397](https://github.com/hat47x/sui-sensemaking/pull/3397)、[CI #42370271882](https://github.com/hat47x/sui-sensemaking/actions/runs/42370271882) | 公開Projection→**固定UTF-8 Markdown byte**生成を含むfrontend CI成功。明示的なopt-in時には新規絶対pathへ `wx` で書く試験入口がある | CIでは**opt-in未設定で実ファイルProducer出力はNOT RUN**。実受領・出典復元・正式Writerは未成立 |
| [SHO merged #21](https://github.com/hat47x/sho-composer/pull/21)／[#22](https://github.com/hat47x/sho-composer/pull/22) | SHO実G1b Reader、Raw SHA/Markdown Link Occurrence、POSIX FIFO事前拒否の限定回帰 | #21の**同じ実SUI Producer作成FileをSHO CLIが開くH1**はCIでSKIP。合成Fixture読取が実H1を代替しない |
| [SHO merged #24](https://github.com/hat47x/sho-composer/pull/24)、[Draft #25](https://github.com/hat47x/sho-composer/pull/25) | G2-Pは独立byte-span Patch Planner。#25の**検証済み固定HEAD** `eb12e0cc2eff01ef0be9270c3a0a6adb77b3a09a`の[W1 CI #42461729731](https://github.com/hat47x/sho-composer/actions/runs/42461729731)を含む5本の当該HEAD workflowはcompleted/success。**実Fileへの別名Working Copy作成→close→別Open/Reader再読取**と部分作用の限定検査 | #25はDraft・test-only。正式Editor Save/Source Revision・Grant/CAS/atomic/fsync、実SUI Producer→SHO一File連結は未成立 |

この段階の最短課題は、新しい共通Source Registryや「原典ID」新規発明ではなく、**各既存Productの本物の入口を同じ固定Source/Attemptで走らせ、どこまで実際に届いたかを別々のWitnessで照合すること**である。

### 24.2 新しい判別反例：CC-25〜27（TEI横断実試験はNOT RUN）

| ケースと問いQ | 独立に固定する二世界・入力 | 失敗させるべき誤Claimと必要なProbe |
| --- | --- | --- |
| **CC-25 Source由来の非復元**「このMarkdownは同じ原典を引用しているか」 | W25a/b：元Card Source cue/Island所属は異なるが、公開Projectionは同じ。その両方を**同じ許可済みMarkdown生成規則**へ通せば、同一Raw Markdown bytes・SHA・SHO Link Occurrenceが成立し得る | SUI→Markdown→SHOで同一Raw SHA/Reader PASSでも元Source/Topology/独立証拠数は復元不可。元出典に関するQには許可されたSUI Source/Review・原文Readerの別Witnessが必要。秘密Source IDを公開Markdownへ追加して解決しない |
| **CC-26 Fixture成功と実H1の混同**「実SUI Producerが作ったFileをSHOが読んだか」 | W26a：固定bytesをSHO側Fixtureで作りG1bが読んだ。W26b：実SUI Producerが書込後、実SHO CLIが**同一実File**を開いた。Reader結果（bytes/SHA/Link数）は一致し得る | W26aのReader成功を実SUI H1の成功として採点しない。Producer process、実書込先/出力SHA、対象Fileの引渡し・実Reader起動/受信Raw byte/exit、独立Expected、権限/開示を同一Attemptで束縛。実入口未起動ならH1=NOT RUN |
| **CC-27 作業コピーと正式保存の混同**「文書は現在の正本として安全に保存されたか」 | W27a：SHO #25の別名 `wx` Working Copyが書込・実readback一致。W27b：同一SourceをA→B→Aと途中変更しても、Source SHA/FD identityが復元しWorking Copyの読取一致は起こり得る。さらにSource変更／部分出力・出力改変で`needs-reconciliation`が生じる | `working-copy-readback-verified`は作業コピーbytesの限定証拠であり、原文正本へのSave/Current Revision・Writer Grant・Durability/Owner受理ではない。**No-op候補で新規Working Copyは生成される**ため、実Editor `Save(no-change)` の物理write0とは別。実Fileを改変した後の拒否を未作用と表示しない |

CC-25の前半はSUI #3396の**実Product非単射**に基づく帰結だが、同一Markdown/SHO CLIを**その二世界で通した横断試験は未実施**。CC-26の両側を実H1として比較するコードも本PRでは作らない。CC-27のSHO側W15〜W17は限定CIで実証されているが、TEI/SEIのSource/Meaning Writer試験を行った証拠ではない。

### 24.3 既存のReader・Source Evidenceに接合する受入票（新ABIではない）

| 段階 | 同一Attemptに必要な観測 | Claimの上限 |
| --- | --- | --- |
| 1. **生成前の公開境界** | 公開対象Projection/元Source側の許可範囲、制限モード、明示除外と現在利用可能な出典資格 | Projectionの内容は元Source全件でも、すべてを再配布可能な意味でもない |
| 2. **実Producer書込** | 実argv/env、Source/Project Rev、出力File pathと実Open/Write、Raw SHA、失敗時の部分作用 | 合成固定Vector/オプトイン未実行は実配送ではない |
| 3. **実Reader実受信** | 同じFileの実Open対象/FD、Raw bytes/SHA、Path/revの照合、Link Occurrence、Process exit、Consumer取得時点 | Readerの文法解析/Link抽出成功≠元Source復元・意味受理・原本同一Snapshot |
| 4. **局所候補編集** | 編集意図・明示byte範囲/旧Fragment、独立Expected、候補Preview/承認の有無 | Planner PASSや別名Copyは正式Editor Save承認ではない |
| 5. **Working Copy実Effect** | `wx`作成・実byte書込・close/別Open readback、元Sourceの前後実Read、partial/effect/status、未解決SourceRev/Grant | `readback-verified`≠原文正本への永続Save、失敗≠作用0 |
| 6. **現在のConsumer利用資格** | Scope/Grant/外部Recipientの実受取、独立に必要なR(Q)、Owner受理・実操作資格の有無 | Hash一致・Local CI PASS・Tool呼出しのみでは現行Source/Meaningの資格は確定しない |

**一Source単位の実試行を最優先**にし、Source→Projection→Draft→File→Readerの真正Producer/Consumerをまず接続する。元SUIの非公開来歴は公開SHO Markdownへ無条件に追加しない。G2-W1の同一実File上の局所保存は、その次の独立Product能力として用い、H1が成立したというClaimへ合算しない。エラー時の`needs-reconciliation`はSHO Owner側で観測し、TEIに新Writer/Recovery Engineを作らない。

### 24.4 次のD0/D1/M1とOwnerを限定する

**次に比較するQ**を二つだけ追加する。Q26「SHOの受領は実SUI Producerの同一Fileに由来するか」、Q27「現在のSource Revisionに対して実正式Saveが成立したか」。全条件で初期Observation/実Tool Allowlist・Root/Grant・Disclosure/Reader能力・Read予算を固定する。

- D0：Fixture・実Producer・Reader・Working Copy・正式保存に関する未取得段階を報告し、未知を成功へ昇格しない。
- D1：利用可能な実Product Toolから、Producer→Readerの実H1、またはSource Revision/正式Writer/保存後独立Readの不足する片側を**最小の1 Probe**として選ぶ。既存入口がなければ`unsupported`。
- M1：SEI Methodが同じTool/Grant/Scope・非公開Expected無しで1 Probeを提案し、実Tool起動/実受領と必要R(Q)到達を**別スコア**で検証。D1と同じ選択ならModel固有の追加価値は0。
- 「同じ文字列・同じSHA」だけに依存する答案はCC-25で減点し、実A→B→Aや出力中断後の`needs-reconciliation`をCurrent Source受理に昇格した答案はCC-27で拒否する。

**既存Owner**：TEI #2533＝問い相対の不足・次Probe比較、#2514＝独立Expected、#2516＝Consumer/Disclosure、#2497＝非単射Projection、SEI #767＝Method。実Producer/公開ProjectionはSUI #3397/#3396、Markdown実Reader/H1はSHO #21/#16、編集・保存実EffectはSHO #24/#25/#4/#3、Harnessの実受領・隔離はTAI #168/#271。**TEI側に共通Writer/Source Registry/Receipt Storeを追加しない。**

**今回の状態**：SUI #3396実公開Projection反例PASS、SUI #3397固定Byte試験CI PASS（opt-in実File NOT RUN）、SHO #21実Reader限定PASS（H1 SKIP）、SHO #25 test-only Working CopyのUbuntu/Windows限定PASSは**既存別Runの事実**。本CC-25〜27の実SUI→SHO同一Attempt H1、SEI実M1、正式Editor Save・実Host Grant/Delivery・Owner意味受理は**NOT RUN**。この節の追加自体は**設計/計画更新**であり、Product合格ではない。 **SHO #25後続HEAD `81aa7cdfa20037300f2f9a2b1dd3401dd6a8941d`も[実W1 CI #42461901303](https://github.com/hat47x/sho-composer/actions/runs/42461901303)を含む専用5 workflowがcompleted/successと追認済み**。ただしH1実Producer→SHO同一File・正式Editor Save/Storage資格は変わらず未成立。

<!-- tei-devctx-cc28-29-h3x02-preregister-20261108 -->
## 25. CC-28/29：実Consumer版・CLI応答形状の資格と、SEI H3-X02の実Probe事前登録（2026-11-08）

### 25.1 今回確認した横断進捗と役割

[SEI Draft #824](https://github.com/hat47x/sei-cognition/pull/824) §9.2は、H3-X01〜08でSource/Binding/Transport/Generation/Authorityの不足を分け、**H3-X02「必要Ruleの取得Mode」を最初のD0/D1/M1実比較**へ進める計画を追加した。本節は同じケースのTEI側の実Tool/Source受入票であり、SEI側に第二CLI Readerや評価Oracleを作らない。

- [TEI merged #3354](https://github.com/hat47x/tei-platform/pull/3354)は既存`context_gate.py`で**実11 CLI read-only Reply／32合成Consumer対照**（必須Validation3件、Source Indexのsubject/indexモード、Canonical/Relation/Anchorを含む）を限定実証済み。[CI #42391707962](https://github.com/hat47x/tei-platform/actions/runs/42391707962)成功。ただし正しいModeのRaw Reply受領≠必要全Sourceの同一Snapshot／意味受理／外部HarnessのTool完全配送。
- [TEI merged #3370](https://github.com/hat47x/tei-platform/pull/3370)は実`requirements`の`result`が**配列**であることを現CLI実試験に基づき`tei commands`の`resultShape: "array"`へ追加。広告のないコマンドは`unspecified`であり、推定`object`ではない。[merged #3366](https://github.com/hat47x/tei-platform/pull/3366)で`view -format json`の`worldFingerprint`と`sourceIndexFingerprint`が提供されても、`semanticFingerprint`・元Source Revision・現在Grantとの同値は証明されない。
- [SHO merged #21](https://github.com/hat47x/sho-composer/pull/21)の実Readerと[SHO Draft #25](https://github.com/hat47x/sho-composer/pull/25)のWorking Copy実readbackは別能力。#25の**現HEAD `81aa7cdfa20037300f2f9a2b1dd3401dd6a8941d`**について[実W1 CI #42461901303](https://github.com/hat47x/sho-composer/actions/runs/42461901303)はUbuntu/Windowsの両Job success（W18〜20を含む）。Output Fileの実体差替えと歴史的A→B→A、Identity未観測を区別。正式Editor Save/Source CAS・H1実One-file成功へ転用しない。
- [SUI Draft #3397](https://github.com/hat47x/sui-sensemaking/pull/3397)の独立Expected 354-byte Markdown Producerは通常CI成功。ただしopt-in実File書込はCIでは未起動。[SHO `sui-sho-onefile-p0.cjs`](https://github.com/hat47x/sho-composer/blob/main/scripts/sui-sho-onefile-p0.cjs)の試験専用Runnerをソース照合すると、Producer起動前にSUI root全体のtracked-clean・正規Frontend layout・**固定SUI HEAD**を検査し、SHOの実HEADは記録する。真正二Checkout間での実Producer→実SHO Readerの**同一実File H1はSKIP/未実施**。

### 25.2 CC-28：同じRaw SHAを受け取ってもConsumer版の事前受理は別

**Q28：「このConsumerが独立に受理された実装版で、指定Sourceから実際に読んだか」**。W28aは既存RunnerのSUI固定HEAD・実Producer byte/SHA・SHO Reader結果と`shoHead`の記録が一致、しかし**SHOの事前許可版・依存実効byte・現在Host Grantは未確認**。W28bは同じ354 bytes/SHAとMarkdown Link数を返す別のcleanなSHO checkoutであり、現在のRunnerはConsumer HEADを記録できても**事前Pinがあることを証明しない**。

したがって`same file / same raw byte / selected reader`、`actual reader invoked`、`observed consumer code revision`、`prequalified consumer build`、`current disclosure/host grant`は別の証拠。Producer PinとConsumer Pinを対称にしたい場合はSHO Ownerが**受理するConsumer HEAD/Build tupleを事前に固定**する。これはH1の**正式Consumer版認定**を要求するときの追加Gateであり、まず限定的な実H1を行うこと自体を停止しない。旧SUI fake checkout拒否負例からSHO consumer版の受理済みを推論しない。

本W28a/bは**ソースから確認した未検証の反例設計**。偽SHO Checkoutを実起動した、または実H1が成立したとは主張しない。元SUI Projectionの非単射性から、Raw Markdown一致を元Card/Island/独立Source数の同一へ昇格しない。

### 25.3 CC-29：`result`形状の読み違いは「不存在」ではない

| 設計上の反例 | 現状の実Productで使える証拠 | 正しい不足・次Probe |
| --- | --- | --- |
| W29a 正常な`tei requirements`の配列`result`をobject専用Readerが誤って空と解釈 | mainの`resultShape: "array"`と実CLI byte | `consumer-shape-mismatch`。必要Requirementが無いとは言わず、既存`tei commands`＋受領Raw Replyと照合 |
| W29b 正常な`result: []`だがSource/Required Scope/Providerの取得範囲が部分的 | `resultShape`は形状だけの情報 | `required-unresolved`。空配列から全対象不存在やTest coverage 0を推論しない。独立R(Q)、Read Scope/Mode/終端を確認 |
| W29c `view`の`semanticFingerprint`が同じだがWorld/SourceIndex Fingerprintは異なる | #3366の実CLI 3種類のFingerprint | 意味的な表示の同一性とSource Revision/実Read cohortを分離。必要なFPを同じScopeで比較し、意味受理・現在Grantは別 |

#3370/#3366の限定実CLIテストは**既存の製品実証**。W29a〜cをこのPRで別途実Go/AI/Harnessに通した証拠はない。正常配列・正常Source Indexを同一の欠落扱いにする第二Consumerを追加せず、**既存CLI Contractの読取り条件**としてSEI H3-X01/02へ引渡す。

### 25.4 H3-X02を具体的に始めるための最小受入票（新ABIではない）

**問いQ**：「独立に固定した公開`sales.order`の必須Validation 3件が、現在の許可された設計Sourceに定義されていると主張できるか」。Participantへ**必要IDの3件、同一Q・Tool Allowlist/Scope・Grant・初期実CLI Reply**を渡す。一方、Ruleの本当の存在/欠落、Index Modeを取り違えた理由、非開示・部分配送の真相、Source改版後の独立Expectedは**採点側に隔離**する。秘密の実Source/Reviewer-only pathをParticipantに見せない。正の答えの根拠を別の公開Toolで取得することは許可された範囲に限る。

| 実施段階 | 必要な観測／合格条件 | 危険な昇格 |
| --- | --- | --- |
| **M1R-0 Fixture/Tool固定** | TEI mainの実`tei commands`、`workspace-files`、`source-index`（subject/indexを別引数として固定）、`catalog`、`inspect-validation`、必要なFP/Raw Reply/Producer Binary。独立Required 3 Rule＋範囲外Rule1件の対照を準備 | Tool名を提示しただけで実Invoke可能・Current Grantありとしない |
| **M1R-1 D0/D1の対照** | D0は追加Readなしで`wrong-mode`／`required-missing`／`unknown`を識別。D1は**同じTool Profileのindex-mode取得を一回**行い、実Replyを再照合 | `source-index` subjectモードの空結果を「Ruleが存在しない」と誤断言 |
| **M1R-2 M1実Method** | SEI既存Method経路が**同じ初期bytes、同じBudget=1、Grant/Disclosure、公開Q**で一つだけProbeを選択する。Decision proposalと実Tool invocation/実Received bytesを別票記録 | 模擬Methodの提案を実Read完了とする、D1と同じProbeで純増効果を主張 |
| **M1R-3 Blind Expected** | 正常・subject-mode違い・本当の必要Rule欠落・途中切断・範囲外変更の5条件で、正答、誤不存在、過剰停止、必要Witness、不要Read/Cost/Latencyを別採点 | 合成Control PASSを実Product Provider/全Required／C1へ上げる |
| **M1R-4 外部受領（後続）** | TAI #168の実H2/H3でRequest↔Reply/part digest/end・物理Expected非露出・現行Grantを同一Sessionで立証した場合だけ資格を移譲 | 現在のDSH実Tool呼出し成功（56 calls/55 replies）を比較適格4SessionのPASSに再計上 |

**初回実験の合格条件**：同じ`Q`と独立ExpectedでD0/D1/M1の判定差が記録され、`proposed / invoked / received-complete / required-witness / false-absence / unnecessary-read / cost`の各列に実測値があること。**M1がD1を改善しないなら、Methodを必須とする新Gateは採用しない**。改善があっても、実Method/外部Harnessの権限・Tool終端・物理隔離は別承認。今回実D0/D1/M1 Tool実験と非露出検証はNOT RUN。

### 25.5 優先順位と保留した構想

- **今すぐ使う実物**：#3354の11 CLI＋32限定Controls、#3370の`resultShape`、#3366のView FP、#3176のCC-12〜18実Go Probe。次に**一Functionの必要R(Q)を実Loader/Parser/Rule/Providerへ通す**のはTEI #3262/#2586、SEI Methodによる選択比較はSEI #767/#824がOwner。重複Fixture/Runner/Oracleは増やさない。
- **文書・Raw二Consumerの一回実験**：実行手順の正本は[SHO #21のRunbook](https://github.com/hat47x/sho-composer/blob/main/docs/sui-p0-same-file-handoff-trial-2026-11-07.md)。既存Runner `node scripts/sui-sho-onefile-p0.cjs --sui-frontend <実SUI frontend絶対Path>`（SHO checkoutで実行）へ**固定SUI Producer HEAD `0bc63b944c6f18e0fc8693bd05092ad031997c11`と実SHO HEAD/clean checkout**を事前登録する。合成公開Markdownの独立Expectedは **354 UTF-8 bytes／SHA-256 `7fd6fde2fbc4d9beaf9de14e8b8f4f5d70003cd2129f1502cdd7c73f50c171cd`**。SUI実Producer `wx`書込→同じFileをSHO実G1b CLIがOpenし、対象opt-inケースを**非Skip PASS**として実Raw/Link/Path/Exit/Process Logとともに観測した場合だけH1限定PASS。両Repo個別CI・SHO固定Fixture PASSのみならH1=NOT RUN。実行に必要な二つの実checkout/依存と許可されたRootが揃わない場合、新Toolやprivate credentialsをTEIに設けず準備未完了とする。SHO #25の別名Working Copy readback・G2-W2正式Editor Save・元Source/Current Grantは別資格。
- **長期**：[TEI #3475](https://github.com/hat47x/tei-platform/issues/3475)のOntology DSL/Decision TwinはO0 Fit/Gapまでを現在の知見として扱い、O1以降のDigital Twin・SEI Decision委譲・実Simulationを初回MethodのBlocking条件にしない。[TEI #3394](https://github.com/hat47x/tei-platform/issues/3394)のREA解析→Authoring/SHOも各Ownerの**将来接続**として扱い、現TEI Contextに未実装の外部Source Readerや保存Providerを作ったものとしない。
- **非採用**：二つ目のContext Gate、独自Agent/Meaning/Authority Host、外部文書・データサービスをTEIの必須Runtimeとすること、Result Shape全コマンド一律object、Source SHA＝Meaning受理、Preview/Working Copy＝正式Save、SUI Card＝TEI Accepted Rule、提出されたMethod Proposal＝実Effect。

**実証状態**：CC-28/29とM1R-0〜4はここでは設計・検証手順として登録し、Go/Product/Method実試験は**NOT RUN**。既存#3354/#3370/#3366/#3176/#3396/SHO #25の限定CI結果を正本ごとに保持。本文は新しいTEI Product契約、正式OS/Sandbox認可、Owner意味受理、実H1成功を採用したものではない。

<!-- tei-devctx-cc30-execution-selection-and-h3-mcp-20261108 -->
## 26. CC-30：Runner引数・実Eventと最新Product Context、MCP参照Adapterを分ける（2026-11-08）

### 26.1 前回から変化した実装基線

同じ「見えない／受領できない／実行していない」を、Ownerごとに区別する。

| Owner／確認した固定点 | 正の限定Witness | 次の第一不足Edge |
| --- | --- | --- |
| [TEI merged #3477](https://github.com/hat47x/tei-platform/pull/3477)（[main専用CI #42461058083](https://github.com/hat47x/tei-platform/actions/runs/42461058083)） | 既存Context Packet・11 CLI・32 Controlsを残し、**World一致でもSource Indexの改版で古いContextを拒否**する実CLI source-only LF／A→B→Aの限定実証。旧Draft #3383はclosed/unmerged | 実World/SourceIndexを複数回読んだこと≠原子的Snapshot、実Provider・Host・Owner Meaning／有効なCurrent Grant |
| [TEI merged #3458](https://github.com/hat47x/tei-platform/pull/3458) | **参照MCP read-only Adapterの`validate`応答**がCLIの文書構造・finding失敗と一致する実Go試験。参照Adapterの入口は存在する | 実外部MCP Host/TransportのRequest↔Reply、認可、送出、Consumerの完全受信とDisclosure。旧#3356/#3396の「MCP入口なし」は**調査時点の記録**であり現在のProduct全体へ敷衍しない |
| [TEI merged #3370](https://github.com/hat47x/tei-platform/pull/3370)、[#3366](https://github.com/hat47x/tei-platform/pull/3366) | `requirements.resultShape=array`・`view`のWorld/SourceIndex FPは実CLI契約として確認済み | Raw Resultの正常空配列、未定義Shape、Source/意味/Grantの不足を`not found`へ縮約しない。§25 CC-29は判別実行未実施 |
| [TEI Draft #3365](https://github.com/hat47x/tei-platform/pull/3365)（[CI #42394673357](https://github.com/hat47x/tei-platform/actions/runs/42394673357) success） | 6 pinned Scenarioの局所観測とpartial receiptの`notObserved`、7 synthetic controls | 実H3 Reply/End、全Requirement Coverage、C1、ProductのTest Universe閉包。Draft/未統合 |
| [TEI Draft #3329](https://github.com/hat47x/tei-platform/pull/3329)（[OKF K1 CI #42462571037](https://github.com/hat47x/tei-platform/actions/runs/42462571037) success）、[SHO Draft #27](https://github.com/hat47x/sho-composer/pull/27) | **固定4本の物理Markdown原文・Raw byte/SHA**を、TEI側の固定Bundle試験とSHO実CLI Readerで各々確認。SHO側のCIは独立に保つ | 全Reference Closure、K2共通Pluginの実選択／二Consumer Invoke、正式Writer/Save/Permission。固定Fixture読取をSUI実Producer→SHO同一File H1へ加算しない |
| [SHO Draft #25](https://github.com/hat47x/sho-composer/pull/25)（[CI #42461901303](https://github.com/hat47x/sho-composer/actions/runs/42461901303) success） | Ubuntu/WindowsでWorking Copy実File書込・別Open readbackと限定部分作用の試験 | Production Editor Save/CAS/Atomic Publish/Owner/Current Grant、実SUI H1は未達 |

**Source／Target／Toolの版を相互代用しない**。#3477の旧Context拒否、#3458のCLI≒MCP Adapter文書一致、SHO K1Rの同Raw SHA、#3365の局所Step Witnessは**別のAttempt／実Consumer／要求宇宙**。これらを一つの全域PASSへ合成しない。

### 26.2 CC-30：TestGoFilesは同じ、Go test exit0、しかし必要Testは未実行

既存[Draft #3176](https://github.com/hat47x/tei-platform/pull/3176)に合成Goの`TestDiscoveryProfileCC30RunFilterDoesNotQualifyRequiredNamedTest`（`tools/reference-developer-context/test_invocation_characterization_test.go`）を追加した。これは#2533の**実Test Invocationの意味**をCC-16/17/18から一段深める独立負例であり、新Runner／正式Test Receipt型ではない。

同じ`qualification.go`、同じ`qualification_test.go`（`TestRequired`・`TestUnrelated`を定義）、同じ実Go `TestGoFiles`、同じ`Discover` Target/候補Report、同じGo package／GOOSを固定し、**`go test -json -count=1 -run <filter> .`**の実argvのみを変える。

| 実argvの`-run` | Process Exit（期待） | 必要`TestRequired`の実Go named Event | 受入判断 |
| --- | --- | --- | --- |
| `^TestRequired$` | 0 | `run`＋`pass` | **この一Named Test／このAttempt**に限り実行の肯定Witness |
| `^TestUnrelated$` | 0 | **なし**（他Testの`pass`あり） | 必要Testの`pass`にはならない |
| `^TestNoSuchTest$` | 0 | **なし**（実Named Eventは0） | 「テストが成功」ではなく**当該要求Test未実行／未観測** |

**報告条件**：指定`TestRequired`がGoのファイル選択に含まれること・Process Exit=0・同一Source digestの**3つを合わせても**そのTestの実行は証明できない。必要Witnessは実argv／Runner Profile／Source・Binary／同一Attemptの`TestRequired:run`＋`TestRequired:pass`。Go Package単位`ok`、無関係なTest pass、条件式を含むTest Codeの存在は独立Expected／Business Acceptanceではない。全Named Eventがない場合の正しい出力は`required-test-not-observed-on-this-attempt`であり、当該Testが一度も実行されたことがない、とは断定しない。

**実証上限**：これは一時Go Moduleの実`go test -json`による最小負例。生成済TEIの実Run／外部TAI Harness／OS Process Audit／完全Coverage／Go Code Mutantではない。コード保存と**exact SHA CI**は別記録で判定し、CI成功前に「PASS」を宣言しない。

### 26.3 既存Method/Context計画との統合：「最初に欠けているEdge」を選ぶ

次の評価に**新しい共通Schema/Proof Storeは不要**。§25で固定した最初の実Probe H3-X02と、新CC-30の実Runner-filter問題を、既存の問い相対`Claim / Witness / Missing Edge / Probe / Owner`票で扱う。

| Q | 最初の候補不足Edge | D0→D1→M1で評価する`next probe` | 不適切な近道 |
| --- | --- | --- | --- |
| H3-X02 必須3 Ruleの存在 | SubjectモードではRequired Definition全件が見えない | **既存CLI** `source-index -workspace`（index mode）を一回取得し、独立R(Q)・Source Role・Revisionと照合 | `subject`の空をRule不存在とする、MCP Adapterの文書一致を実H3完整受領と呼ぶ |
| CC-30 必須Named Testが実行済みか | `TestGoFiles`に存在するが`-run`が別Test／存在しないTestを指す | **実Go argvと同一Attemptのnamed Event**を検査。再実行するならTarget/Dependency/Mode/Revisionも同じProfileで確認 | `exit 0`または別Test`pass`を必要TestのPASSと扱う |
| K1R/SHO Raw文書が意味資料として利用可能か | 4原文byte受領と、Source closure・Rule/Meaning Bindの間が未到達 | Tool側では**既存Raw Reader／参照位置と実消費Reader**へ。意味受理はOwnerの別Probe | K1 Reader成功＝K2 Plug-in適格／Owner意味受理 |
| 一Function`sales.order.submit`が実行可能か | Capability宣言・Application uses と、実Invocation Binding/Query Provider間の辺 | [merged #3482](https://github.com/hat47x/tei-platform/pull/3482)の**実13 CLI／42 Consumer対照**を採用済み宣言資格として受け取り、次は具体Invocation Binding／Provider Selected→Actual Invokedを別受領 | 宣言・`uses sales`・DB `read-write`の文字列を実Commit/実Providerと読む |

M1の提案はD1と同じAllowed Tool・Read Budget・Q/Grant/独立Expectedで評価し、**実 Tool Invocation／Raw Reply End／Consumer受領／必要Witness**を別票にする。外部SEI [#767](https://github.com/hat47x/sei-cognition/issues/767)／[Draft #824](https://github.com/hat47x/sei-cognition/pull/824)は方法選択、実TEI CLI/Adapterと各Rule RequiredはTEI、実TAI H2/H3のHost/Disclosure/OS/Egressは既存TAI Owner。TEI Adapterの成功でM1の実権限・配送完結を認定しない。

### 26.4 次の受入順序と終了条件

1. **CC-30を検収**：exact Test Source commitの実Go Focused CIで上記3 argv/Named Eventを検証し、後続Plan HEADのCI/Healthとは別結果として記録。CC-12〜18は先行の実Go証拠のまま。
2. **TEI #3262の一Function（宣言側13 Reply/42対照）はmain統合済み**：[PR #3482](https://github.com/hat47x/tei-platform/pull/3482)の[専用CI #42462672188](https://github.com/hat47x/tei-platform/actions/runs/42462672188)と[main push CI #42462861657](https://github.com/hat47x/tei-platform/actions/runs/42462861657) successを受領。次は**宣言→実Capability Invocation Binding→実Query/Provider→Rowset→Consumer**の最初の未接続辺を実証し、Host Grant/副作用を独立観測。データ不存在と権限不足を区別する。
3. **H3-A0→A1の順で実Method比較**：まずTEI実CLIの観測からSEIのRecommendation保存・resume再読取までを**一つのローカル試行**で検収する（§26.7）。次にH3-X02のD0／固定D1／SEI M1を**同じSource・Tool/Profile・権限・Budget 1・隠れたExpected**で比較する。Test FilterとRaw/SHOは**次の別Q**とし、SEI記録成功を実Method/Tool受領成功として合算しない。
4. **文書経路**：SHO #27 K1Rの実4原文読取を基線とし、K2のProduct Plugin実選択と両Consumer実Invoke／SUI実Producer→SHO同一物理File H1を各Ownerで測る。**Product Writerの正式保存**はさらに別のPermission/Version/Effect資格。

**強制的な保留**：外部MCP実Transport/Host、SEI M1実Method試行、H3-X02 Blind Win、AI未知T1-B生成、全Business Source Closure、実DB/Provider Commit、実SHO Editor Save／SUI Producer H1、Owner Accepted Meaning、Current Grantはいずれも**NOT RUN／未受理**。#3477 SourceIndex A→B→Aの復元から歴史が無かったことを推論せず、#3458 Adapterがあるから全TEI ToolがMCP経由で実使用できると宣言しない。第二Reader/Source正本/Permission Engine/認知Host/Runnerを新設しない。

<!-- tei-devctx-sei-persistence-and-main-3482-20261108 -->
### 26.5 Main #3482の実証とSEI Record保存境界（横断追補）

**新たに確定した実Productの範囲**：[main merged #3482](https://github.com/hat47x/tei-platform/pull/3482)（commit `8ed583790e63b63527c1dd8f8aecc1d6ce6e3260`）の[PR専用CI #42462672188](https://github.com/hat47x/tei-platform/actions/runs/42462672188)と[main push CI #42462861657](https://github.com/hat47x/tei-platform/actions/runs/42462861657)はそれぞれ実13 CLI Reply／42 Consumer Controls・Pinned Scenario6・T1-A・Source-only freshnessを**限定PASS**。CIにおける実バイナリSHAは別（PR `0a6712d9…`、main `ce9fa9b8…`）であり、同一実行・完全Reader closureとは呼ばない。42は試験用の不正Reply拒否等であって、42個の独立業務テストやProduct code mutantsではない。Capability `sales.order.submit`とApplication `order_entry`の**宣言・入出力型・uses/Validation/DB requirement・Source Definition/Loader一致**を満たしただけで、具体Function Binding／実Provider Invoke／実DB Commit／Host Effectをまだ示さない。ここで§26.3/26.4の「13/42は今後実施」という古い記述を更新した。

**SEI record保存の意味をCLI source実読取で限定**：[SEI Draft #824](https://github.com/hat47x/sei-cognition/pull/824)のコード監査を、現在の`runtime/go/seicli/seicli.go`の読取で再照合した。`sei add recommendation`の引数に`--unresolved`はなく、`--unresolved`はDecision側に限られる。`BuildResume.Unresolved`はDecisionの`UnresolvedRefs`であって、任意の未提案Questionや保留Item全件を表すわけではない。`--basis kind:id`や`--source`は記録上の自己申告参照であり、TEIのSource byte/Revision/Current Grantを自動検証しない。`snapshot_digest`はSEI bundle bytesに対する観測でありTEI World/Source Index/Tool Result受領Digestとは異なる。**Recommendationの未採用、Questionの未解決、実Read不可能、ReferenceのUnavailableは別状態**。

**H3-X02の実験への含意**：初回Methodの提案をSEIへ保存しても、それだけでは実TEI Rule取得・実Tool Request/Reply・現在Provider/Grant・独立Expected資格が生まれない。SEIでの`proposed/recorded/resumed`とTEIの`invoked/received/source-matched`、TAIの`grant-checked/end-received`を**同一物理Runの別Witness**として整列させること。SEIにTEIの外部Raw Reader、第二Context Gate、未採用Recommendationを勝手にUnresolved化するStore/Schemaを増やさない。

**停止線**：#3482のConsumer判別合格≠機能実行、#3458 MCP Adapter一致≠実外部Harness Tool受領、SEI Resume再開≠TEI Source再確認、#3176のGo named event≠Owner Business Meaning。実H3-X02 D0/D1/M1同条件Tool比較、SEI Recommendation保存のA-02〜04、実Provider/Rowset、外部OS拒否/Current Grantは本節ではNOT RUN。

<!-- tei-devctx-eg09-cross-boundary-to-method-20261108 -->
### 26.6 Source型とOperator結果型の同一性を、業務意味の受理と混同しない（横断EG-09／実Go試験を再利用）

**追加する具体的な判別線**：[TEI merged #2639](https://github.com/hat47x/tei-platform/pull/2639)の実Go試験 `TestEverythingDataBoundDataIDCannotProveModelSemanticEquality`（現mainの`reference/dsl-language-plugin/everything_data_cross_surface_test.go`）では、Data ID `sales.order`、属性ID `sales.order.amount`、Operator `std.where`、出力 `rows/sales.order` が同じ二つのNamespaceでも、属性TypeRefを `std.decimal` と `std.string` に変えると、**限定Binderの結果型だけは同じ**になる。これはType/Meaning差分があることと、既存Binderの狭い契約がそれを表示しないことの**実Go Characterization**であり、正式Semantic Workspace Validate、実SQL/計算、丸め/金額制約、Ownerの受理・リビジョン再資格化まで合格した証拠ではない。

**§26.5のmain #3482と結合する際の順序**：
1. 宣言・Loader・Function/Rule/Source Indexと実CLI Replyを取得できたことは、まず**その範囲のDefinition witness**。同PRの13実Reply／42 bounded Controlsを、属性の意味同値・Provider実Invoke・Resultの業務適格性に自動昇格しない。
2. 問いQが「`sales.order.amount`を金額/Decimalとして使用できるか」なら、独立Required R(Q)に**属性TypeRef、単位/精度、必要Rule/Relation、対象Function Binding、適用Revision**を必要な分だけ指定。TEIの実Parser/Source Readerから**本当に読み取った項目**と一致させ、Data IDと出力 `rows/sales.order` だけで充足としない。単位や丸めはSource/Ownerで未定義なら `unknown`。
3. 結果型・Role名・SUIのCard/SHO Markdownにある金額語を、TEI Accepted Meaningへ自動変換しない。SUIの探索は元Observation、SHOは文章とSource位置、TEIは型付きCandidate/Rule/FunctionとOwnerの意味受理という**別資格**。出典元が存在しても、同一Source Revisionの必要型がParser/Loaderを通ってConsumerまで届いた証拠がなければQは部分的なまま。
4. **D0/D1/M1の最小追加Q**：「Result `rows/sales.order` が同じときに型変更を検出し、どのReader/Toolで意味の不足を埋めるか」。D0は変化を認定できないと報告。D1は現に利用可能・許可済みのAttribute/Source照会を**一Probe**選ぶ（実入口不明なら `unsupported`）。M1は同じQ・公開Source/Tool/Grant/予算・blind Expectedで次Probeを選ぶ。必要型が実Read/実Consumer受領されたか、誤った意味同一判断、追加読取Costを別採点する。実SEI Methodと新しいTool経路の実試験は**NOT RUN**。

**既存の担当を維持**：EG-09の狭いParser/Binderは#2639/#2619/#2620、実Source/型とRule必要閉包は#3262/#1180、独立Required/Expectedは#2514、外部Source候補/SHO文書のMeaning境界は#2516、実Query/Provider/Host Effectは#2586/#1174、MethodはSEI #767/#824。Ontology/Decision Twin #3475とREA→SHO #3394は将来の別Ownerで、初回H3-X02/Function実InvokeのBlocking条件にしない。新Parser/Provider/Proof Store/Owner Engineはここに追加しない。

**状態**：#2639の実GoテストSourceとmerged事実を再利用する**計画追補**。この§26.6で新たなGo Test、CLI Predicate実実行、独立型Oracle実受領、SEI D0/D1/M1、Provider/DB、Owner受理は**NOT RUN**。

<!-- tei-devctx-h3a0-sei-record-resume-20261108 -->
### 26.7 H3-A0→A1→A2：TEI実観測からSEI記録再開へ進む最短受入（2026-11-08）

**新Runtime／Runner／Store／Schemaを作らない。** [SEI Draft #824](https://github.com/hat47x/sei-cognition/pull/824)のH3-A0→A1→A2と、現行runtime/go/seicli/seicli.goの実CLI実装を照合した。SEIには「init」「add recommendation」「resume --json」と、snapshot digest付きのrecord-id再読取入口が存在する。他方、Recommendationに「--unresolved」はなく、このFlagはDecisionにのみ存在する。「--basis kind:id」は論理参照、「--source」は**推薦を生成した人物／Toolの申告**であり、TEI実Source byte・版・Current Grantの検証機能ではない。

| 受入段階 | 必要な一試行の実証 | 誤って合格としない内容 |
| --- | --- | --- |
| **A0-1 TEI実観測** | 固定した公開Workspace、実TEI binary版とSHA、source-indexの**subject／index両Mode**の実argv・exit・stdout/stderr原byteとdigest・resultMode・対応するWorld/SourceIndex版を保持。独立Required Validation 3件の範囲を採点側で先に固定する | subject modeの空＝Rule不存在、index mode表示＝全Required closure、result配列空＝配送欠落 |
| **A0-2 SEIへ提案を保存** | 同じ試行に隔離した許可済み一時SEI data rootでsei initを実行。実観測から未到達の必要Sourceを説明するRecommendationを、現実に存在する「sei add recommendation --id ... --statement ...」で**記録する**。元TEI Response digestは評価票へ別保持する | Recommendationのbasis/source欄があること＝TEI Source/Tool Receiptを検証済み、Proposal＝実Tool呼出し、保存＝Owner Accepted Meaning |
| **A0-3 独立Resume** | 別のsei resume --json呼出しでSEI store snapshot digestを取得し、「--json --record-id ID --snapshot-digest DIGEST」で元記録のID/statement/basis/sourceを照合する。途中更新で旧snapshot digestが不適合になれば拒否を維持する | SEI storeのsnapshot digestをTEI World/SourceIndex digestへ読み替えること、Resume成功＝TEI Source現在資格の復活 |
| **A0-4 負例** | Recommendationへ未対応の--unresolvedを与えたら拒否する。未決Question/Unknownを保存するために架空のDecisionを作らない。Read Scope/Mode/Source版が欠けたときはunknown／needs-revalidationと報告する | 未決一覧が空＝未決なし、Recommendation保存＝SEI Method採用、CLIが表示できるTool＝現在の実行Grant |
| **A1 陰性対照付き比較** | A0の記録再開が成立して初めて、H3-X02の同一Q・公開Tool・Grant・Budget=1・blind ExpectedでD0（不足説明）／D1（決定的なindex-mode取得）／M1（SEI Methodが一Probe選択）を比較する。提案、実Invoke、Raw Reply完全受領、必要R(Q)到達、費用を別スコアにする | H3-X02はD1で解ける**陰性対照**。M1がD1と同じProbeを選んだだけなら追加価値を認めない |
| **A2 未決の継続** | Decision前のQuestion/Unknown/Conflictのネイティブ保存資格が必要ならSEI #711/#678のOwner判断に返す | SEIに既に任意の未解決Inquiry Storeが存在すると仮定しない |

**A0合格の上限**：一つのTEI実CLI Reply集合を読んだことと、その不足を提案するRecommendationをSEI実Storeへ保存・別呼出しで再読取したことを、二つの実Binary/原Byte/別Digestで束縛できた場合に限り、**限定ローカル記録往復PASS**。これだけではTEI生SourceのSEI Method受取、Tool実Invoke、H2/H3外部Harnessの終端受領、権限/Disclosure、Provider実行、Owner意味受理は成立しない。SEI側のProduct試行は#824/#767が担当し、TEIは#2533/#2514/#2516の既存Oracle／Source受入だけを再利用する。

**現在の実証状態**：既存TEI #3482の13実CLI Reply・42限定Controlsと、実SEI CLI Sourceの契約を照合した**計画追補**。新しい二製品H3-A0実Run・A1/A2・実SEI Method・OS/Host受領・Owner判断は**NOT RUN**。

<!-- tei-devctx-h3a0-real-two-cli-ci-20261108 -->
### 26.8 H3-A0の実二製品ローカル試行：保存・独立再読取まで限定PASS（2026-11-08）

#### 固定した製品と実観測

[Draft TEI PR #3176](https://github.com/hat47x/tei-platform/pull/3176)へ実Goテストを追加した。[H3-A0のGo試験](https://github.com/hat47x/tei-platform/blob/26b79a68473e073d0e77719a25c9d6723161d94d/tools/reference-developer-context/h3a0_tei_sei_actual_record_test.go)は、現TEIの固定版をビルドして「source-index」のsubject/index両Modeを**別の実CLIプロセスとして**呼び出し、独立に固定したRequired Validation 3件がindexに存在することを確認する。その後、SEIの実CLIを別にビルドし、隔離した新規Data RootへRecommendationを保存、独立した「resume --json」と「resume --json --record-id … --snapshot-digest …」で原記録を再読取する。Recommendationで未対応の「--unresolved」を拒否し、拒否後にSEI Store digestと件数が変わらないことも検証する。

| Witness | 実Source/観測の固定値 | 判定 |
| --- | --- | --- |
| **TEI実Product CLI** | TEI commit **26b79a68473e073d0e77719a25c9d6723161d94d**、Go単体バイナリ。実index Raw stdout digest **sha256:e6f9d38477dda0e9b538ba43b551498f682d4177a72168747022ae3ad79a6700**。subject Raw stdout digest **sha256:ce7552eb6ab8ad4a63d5225feb55da8a8b31e418b3d66dc4d4ab6df49f56d056**。対象は公開reference/workspace | CLI別実行・JSON契約の局所PASS。Source全件閉包・共同Atomic Snapshotではない |
| **SEI実Product CLI** | pin **6d98ddf2d1080a1c3a308294fb342c231a08294d**。新規隔離Store、Recommendation 1件、別CLIからresume JSON・snapshot digest付きrecord-id再読取。記録が改変されない負例を含む | ローカルStore往復の限定PASS。未決QuestionのNative永続化やMeaning受理ではない |
| **実CI実行** | [TEI push Workflow #42474971439](https://github.com/hat47x/tei-platform/actions/runs/42474971439) の2 Job（bounded-discovery・tei-sei-h3a0）completed/success。実Goログに「TestH3A0ActualTEIToSEIRecommendationResume」PASSを確認 | **同じ実TEI commit**上でのH3-A0局所受入PASS。PR WorkflowはGitHubの合成merge SHAで試行されるため、push側の固定HEADを一次証拠とする |
| **失敗と部分効果** | RecommendationでDecision専用Flagを指定した呼出しは失敗し、その後のStore digest/件数は不変 | 拒否の実Go負例PASS。失敗一般・OS/ネットワークの無作用は保証しない |

SEI Storeのsnapshot digestは試行ごとに作成日時を含む別Sourceであり、TEI World/Source Indexのfingerprintと一致すべき値ではない。実ログの一回の例は **sha256:6bfdf22af4ea7ac0a54f6d5fcaea6afd1ac57d63f8d49b25d01a496b5034eef3**。この値を将来の再開に使える現行SEI Storeとして共有せず、**当該CI一時ファイルだけの記録**として扱う。

#### 成立したClaimと、成立しないClaim

- **成立した**：固定した同一公開WorkspaceでのTEI実Source Index subject/indexモード読取、独立Required3件のindex照合、事前固定した**評価用Recommendation**のSEI実Store保存と再読取、未対応Flagの拒否/Store不変。
- **成立しない**：TEIの生Raw SourceがSEIのCognitive Methodへ渡されたこと、モデルが自律的にProbeを提案したこと、SEIの任意Question/UnknownのNative保存、Tool実行Grant/Consumer公開・実外部Harnessの完全受領、実Provider Invocation/Result、Source改訂の同一Snapshot保証、Owner Accepted Meaning。Recommendation文面はテストに固定された**評価者の文**であり、認知Methodによる生成結果ではない。
- **実Toolの制限**：二つのTEI CLI Invocationは同じ公開Workspaceを使うが、atomic Workspace Snapshot/変更監視の保証は別Owner。二製品のRaw bytes/digest・Command実行/受領とSEI Store digestを同じ語「証拠」で上書きしない。
- **次のP2**：H3-A1のD0/D1/M1を、今度はMethodが選ぶ一Probeと決定的D1が選ぶ一Probeの同条件比較へ進める。D1でindex-modeを読めば解けるH3-X02は**陰性対照**。A1のMethod効果・実Tool選択／実Replyの評価はNOT RUN。
- **既存Owner**：本PRはTEI #2533の認知評価Go試験だけを追加。SEIの推薦の契約と未決管理は#824/#767/#711/#678、TEI独立Requiredは#2514、External Deliveryと開示は#2516/TAI #168、実Function/Providerは#3262/#2586のまま。新Host/Runner/SEI Store/schema追加なし。

**今回の実行状態**：**H3-A0限定PASS、H3-A1/A2 NOT RUN**。本計画修正自体はMarkdown変更であり、次HEADの全Repository Health／依存ScannerはGoの専用CIと別判定する。


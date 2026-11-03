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

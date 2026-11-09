# 増分ビルド再利用：実行・証拠・再資格化の対照実証手順

- **状態**：P0→P1実証の提案手順書。正式な製品契約・ADR・業務意味受理ではない。新規試験・性能値は未取得。
- **Owner**：[#2600 増分ビルドの実証](https://github.com/hat47x/tei-platform/issues/2600)
- **固定基準main**：`9c9ef4d160640908e06ad13afb6c2b45c664c748`（2026-11-02）。
- **位置づけ**：既存Build Graph/ArtifactSet/Store、Generator Selection、Oracle、Artifact公開、Construction Contextの実証結果を**同一の操作・変更・Consumer**へ結び付けるための、報告と手順。二つ目のGraph、Schema、Registry、Host、Schedulerを実装しない。
- **機密範囲**：自作合成Fixtureのみ。参照元の非公開原典の本文・ID・名称・locator・秘密情報を載せない。

## 1. 今回の新しい検証問い

**同じ入力へ到達する選択的な再実行が、全再生成に必要な成果物とClaimを保ちながら、本当に仕事を減らせるか**。

以下を別の主張として扱う。

| 主張 | 判断根拠と限界 |
| --- | --- |
| 計算・Build省略 | 前後の依存閉包、対象選択predicate、実消費Source、Builder/Toolchain/Recipe/Targetの一致。未列挙の入力を「空」と扱わない |
| Artifact内容の再利用 | role、member集合、byte digest、要求Consumerにとっての必要member。単一ファイルや自己申告Fingerprintは十分条件でない |
| 過去Evidenceの適用 | 固定したAttempt/Source・Fixture/Oracle・環境・対象・時点・適用Claimの共同成立 |
| 現在の採用資格 | Registry/Trust、Owner受理、外部Host Grant/Disclosure、公開世代を利用時に別途確認 |
| 正しい高速化 | 全再生成と必要結果・独立Expectedが一致し、再利用の検査を含む実時間が短縮される |

**現行の実装境界**：`tools/reference-build-execution-binding/graph_execution.go`の`BuildGraphNodeState`は`succeeded / failed / blocked`。全ノードの観測を要求し、`pinBuildGraphExecution`は完全な実行だけをpinする。これは当該Runの実行報告であり、過去Runを再利用したことの報告形式ではない。再利用ノードを今回の`succeeded`に偽装したり、`blocked`へ読み替えたりしない。最初は別の**実験判定票**で`executed / reused / unknown / unsupported`を記録する（正式な型は未採用）。

## 2. 最初の実行対象と固定するもの

既存`tools/reference-build-execution-binding/graph_execution_test.go`の3ノード合成Graphを使う。

```text
generator ──(generated-source)──> consumer
independent
```

既存`locked_build_set_test.go`には実Go Buildで二つの出力memberを生成し、`LockedBuild`から`ArtifactSet`を束縛する参照がある。最初は両テストの既存保証を確認してから、変更影響の追加ケースを独立に構成する。

固定対象：exact Git HEAD / Go version / OS / target / Generator・Builder・Toolchain / graph・node plan fp / actual consumed input（確認範囲とunknown）/ Source closure / fixture・独立Expected / member role・digest / Consumer / output scope・epoch。複数Issueで得た局所証拠は**同じRunの証拠として合成しない**。

## 3. 全実行・選択的実行のF0/F1/S1/S2比較

| 試行 | 処理 | 必須観測 | 期待する判定 |
| --- | --- | --- | --- |
| **F0 baseline full** | A版で3ノードを実行 | 実行Plan、全node観測、全member bytes、Independent Oracle | 今回実行した3ノードだけを`succeeded`と報告 |
| **F1 changed full** | consumerに影響する入力/規則のみBへ変更。Bで3ノードを全実行 | Bの実消費入力と各Artifact、必要Consumer結果、Independent Expected | S1比較用の全実行基準。業務意味のOwner受理ではない |
| **S1 selective** | Bでconsumerを実行し、generatorとindependentはAの保存物を取得・照合 | node別execute/reuse根拠、AのArtifact実byte、B consumer実byte・結果 | F1と必要な結果・member集合・Claimが同等で、再利用元Aの来歴と適用条件が追える |
| **S2 controls** | source／Toolchain／必要member／適用規則集合／公開epoch／Provider Read Scopeを一条件ずつ変える | 依存前後の差、拒否・unknown理由、再計算対象、旧Lockと旧Objectの不変 | 誤reuse・古い権限の継承・部分出力pin・旧版公開を拒否。独立領域は過剰失効させない |

F0→F1→S1の**同じBの条件**を比較する。変更を意図したconsumer出力のbyteがAと異なることは異常ではない。F1とS1の対象Consumerの必要な結果を比較する。全成果物byte一致が元の仕様上必要な対象では、その一致を明示して検証する。

### node別実験票（新Core schemaではない）

`run / node / current revision + epoch / intended dependency + selection closure / actual consumed Source / plan fingerprint / work(executed|reused|unknown|unsupported) / prior attempt reference / artifact member roles+digests / required consumer outputs / receipt + oracle + environment / current qualification / comparison verdict + reason / elapsed and verification time`。

実際にBuildを省略した証拠がなければ、Storeの`reused=true`や過去receiptを`build skipped`と数えない。計測不能時は`unknown`を保持する。

## 4. 独立した変化圧力：実際の入力に応じた局所再資格化

**A. Sourceのみ・表示のみ変更**：Markdown表→chain等の表記置換は、Authoring/Source引用を更新し、意味/Role/Operator版/入力閉包が等しいと確認できる計算だけ再利用候補。SHOのSource位置とレビュー資格は別。#2625の23件は限定した独立Python合成であり製品ParserのPASSではない。

**B. 出力に現れないRule/Query依存の変更**：表示`amount=600`のまま、隠れた`credit_limit`が700→500になればValidationは再評価。SQLでも`WHERE`、`ORDER BY`、相関/外側参照がOutputField列に現れない場合がある（[Draft #2634 QX-27](https://github.com/hat47x/tei-platform/pull/2634)は合成analyzer入力で実Go DerivedIRを特性記述し、新HEAD `3dfc140` のCIに成功。ただしSQLGlot Worker/実DBの配線は未検証）。**出力Field集合だけを依存閉包とみなさない**。#2624の表示/Rule footprintは独立合成Referenceであり、Providerの実取得は#2628へ。

**C. 生成物内部整合と実Consumer必要集合**：Generic Manifestが`Complete`でも期待memberが欠落し得る。[PR #2587](https://github.com/hat47x/tei-platform/pull/2587)は2026-11-02 14:08 UTCにmainへ統合。そこから分離した[Draft #2633](https://github.com/hat47x/tei-platform/pull/2633)の6つのGo特性テストは検査限界の候補を扱い、PR起票時の記録では新テストの実行・CIは未確認。**mainへのmergeだけではConsumer相対のExpectedMember・実Readの完成は証明しない**。公開時の世代混在・Manifest→relative path→byte→Readerは#2604、Consumer相対必要Kind/SubjectはDraft #2630で独立に検査する。

**D. 異Provider・データClaimの再利用**：同じSnapshot文字列、表示値、Semantic Subjectだけでは同一Cohortを確定できない。#2627の32件の独立Go合成とCI成功は局所証拠。部分Read・absent・NULL・bag・Scope・Row identityを比較し、実DBによる共同Snapshot保証がない場合は当該Claimをunknownとする。Buildキャッシュの再利用と業務Data Resultの再取得は別の仕事として計上する。

**E. 独立Oracle/Metamorphic Relation**：#2514のMR-03（SUMの正確な分割再結合：NULL、桁、丸めを固定）、MR-05（計算依存外の説明のみ変更するとGotは不変であり得る）を追加の判別試験に使う。MR成立のみでExpectedの正当性を推定しない。H5-O1等の独立Expectedと実消費Sourceの束縛は別列にする。

## 5. Evidenceの共同適用と作業量の対照（#1445との接続）

各Nodeが個別PASSしていても、**一つのConsumer Claimに必要な証拠が同時に成立する**とは限らない。Version/epoch、入力集合の完全性、Provider cohort、Scope/grain、Oracle版、現在のDisclosure/Authorityを揃える。共同資格が不明なら局所PASSを共同PASSに転写しない。

既存[#1445 H8-P14](https://github.com/hat47x/tei-platform/issues/1445)の`J0/J1/J2`を再利用し、別の適合判定エンジンを作らず比較する。

| 条件 | やること | 検出・測定するもの |
| --- | --- | --- |
| **J0 naive** | 個別PASSをそのまま結合（負例対照） | false joint pass、版・Scope・Snapshot不一致 |
| **J1 selective qualification** | 既存証拠を照合し、足りないClaim依存のRead/Oracleだけ再取得 | 必要再検証の回数・費用、共同Unknown、誤合格0（確認したFixture内） |
| **J2 full requalification** | 対象Consumer/Queryを一式再実行。ただしProviderが共同Snapshotを保証しなければ資格はunknown | 正しさ・再実行費用・J1との差 |

J1は常にJ2より速いとは仮定しない。J0の見かけの高速化は正しい高速化のKPIから外す。#2600は**計算/Buildの時間**、#1445は**証拠を共同成立させる手間**、#2018は**開発生産性**を担当する。

## 6. 計測・報告

同一Target、fixture、toolchain、環境でcold/warmを区別し反復する。段階的に`dependency/selection/semantic resolve / source verification / generator / build / artifact fetch+verify / oracle+requalification / publication pin+read / total edit→usable result`を測る。初回少数試行は生の時間を保存し、p50/p95や一般化した速度改善は反復後とする。

`T_selective = T_impact + T_lookup + T_verify + T_required_build + T_required_tests + T_publish_and_read`。

比較は、結果の正しさ、Evidence共同成立、現在資格、処理費用の四欄。false reuse、false invalidation、stale publication、partial set、joint unknownを別に数える。保存receiptだけの成功を実Consumer利用成功として計上しない。

## 7. 最初の作業順と受入

1. **R0現行Referenceの再確認**：固定SHAで既存`graph_execution_test.go`と`locked_build_set_test.go`を実行、結果・Go版・Git clean状態を採取する。公式参照入口 `bash tools/reference-build-execution-binding/verify-evidence.sh EXACT_HEAD` を適用可能な環境で利用する。
2. **R1独立Expectedを先に固定**：F0とF1の入力差、S1に再利用を許すProducerを事前指定する。Operator/Rule/Predicateの隠れた依存を含め、全再生成結果をIndependent Oracleと比較する。
3. **R2増分を手順で模擬**：Graph型を変更する前にF1とS1を別Runとして記録し、再利用元の実byteと現在のConsumer入力を実際に検査する。
4. **R3変異対照**：S2の少なくともSource曖昧性、依存追加、member欠落、旧世代公開、異Provider cohortを実施可能なOwnerのFixtureへ委譲。Runが未到達なら`NOT RUN`、証拠不足なら`unknown`と明示する。
5. **R4性能・採否**：P1の範囲内で誤reuseを検出し、過剰失効を測定できた後、コストと待ち時間を反復比較。必要性を示した場合だけ最小のHeadless選択実行を製品化検討する。

**Acceptance（当手順書）**：F0/F1/S1/S2の比較票に、一致する固定版、実消費Source（またはunknown）、全必要member、Runのexecuted/reusedの独立性、Independent Expected、必要な共同資格、Owner未決、計測時間が揃うこと。F1とS1の一致は対象Fixtureの範囲に限る。既存Node executionを偽装した`succeeded`や、単一PR/CI成功からの製品全体PASSを認めない。

## 8. 結果の責務と未決

- #2600：比較票、全再生成との対照、正しい省略と時間測定。
- #2595：Generator/Scriptが実消費したSourceの帰属。
- #2482/#2633：設計Artifactの構造化同一性・Generic CheckSet。
- #2604/#2630：物理一式公開、Consumerごとの必要memberと実Read。
- #2514：Independent Oracle、Claim相対の再資格化。#1445：J0/J1/J2の作業負担。
- #2622/#2627/#2586：Data Operations、複数Providerと共同Read。
- #2018：総開発生産性、CI・Human Review等を含む比較。
- #2619/#2621、SHO/MMI担当：Authoring表層・Source再解決・表示とレビュー。

**Owner未決**：再利用記録を将来どの正式Contractへ置くか、Nodeの`reused`表現が必要か、共同資格の自動化範囲、OS/外部Targetへ要求する実消費入力Witness、公開方式、性能上の採用目標。いずれも本書の採用により自動決定しない。

**本書で実行していないこと**：新Go/Python/Java/SQL/SHOのF0/F1/S1/S2試験、CI、実増分skip、外部DB/Hostの動作、速度測定、正式API・Schema/ADRの採用、Owner業務意味受理。既存の独立合成テスト成功は対応Issue/PRにのみ帰属する。

## 9. 最初のP1反証バッテリー：依存の種類とOracleの根拠を分ける

**位置づけ**：以下は既存IB-07/09/11/12/16/19、S2、J0/J1/J2へ割り当てるための**公開合成Input/Expectedの案**。IB番号を増設せず、参照実装の現状とは別の独立した期待値を**実行前に**固定する。ここに記載したこと自体はTest RUNではない。

### 9.1 単一の「値Lineage」だけでは再利用条件を決められない

Query／Ruleの依存を、利用目的に応じ最低限次の観点へ分けて観測する。正式な新Edge型を提案するものではない。

| 依存観点 | Sourceが変わったときに影響するもの | 最小対照 |
| --- | --- | --- |
| **値** | 返却する値の算出 | SELECTの式、単位・丸め、NULLとmissing |
| **行成立** | 行を返すかどうか | WHERE/EXISTS/NOT EXISTS、非表示Rule、相関条件 |
| **多重度・粒度** | 行の件数・Identity・集計分母 | JOIN fanout、GROUP BY、bag対set |
| **順序・選抜** | 上位N件、順位、順序付きWindow | 非出力ORDER BY列、LIMIT/TOP-N |
| **完全性・時点** | 不存在・合計・共同Snapshotの主張可能性 | 部分取得、Provider cohort、遅延修正・retract |
| **資格・公開** | 同じ結果を再取得・提示・実行できるか | Host Grant、実Readと開示、公開世代 |

例：`SELECT id WHERE secret_score>50`と`...>70`は、出力列`id`とその直接Value Inputsが同じでも、返される**行集合**は違い得る。[Draft #2634](https://github.com/hat47x/tei-platform/pull/2634)のGo QX-27はSynthetic Analyzerの非出力依存を調べる局所証拠であり、`Lowering=full`から入力閉包の網羅を推定しない。[#2387](https://github.com/hat47x/tei-platform/issues/2387)のQX-23/24/25は実SQLの意味差と読取範囲のOwner。生成IR・物理Provider Read・Consumer Egressを同一レベルの「依存有り」証拠へ潰さない。

| Probe | 事前固定する入力・変更 | 独立Expectedと誤再利用 |
| --- | --- | --- |
| **QP-01 行成立** | 1行`{id:A, secret_score:60}`、WHERE閾値`>50`→`>70` | 前=[A]／後=[]。出力列やValue Lineage不変によるskipは誤り |
| **QP-02 多重度** | 親A 1行、子はAを参照する2行。INNER JOINとEXISTSを同じ表示列`parent.id`へ投影 | bag保持ならJOIN=[A,A]、EXISTS=[A]。SQLを構文正規化しただけで同値・同件数としない |
| **QP-03 順序・TOP1** | A rank=9、B rank=8。A rankだけ7へ変更し`ORDER BY rank DESC LIMIT 1` | 前=A／後=B。表示列`id`のValue Inputsだけでは影響判定不可 |
| **QP-04 不存在・部分取得** | 子の実観測は空だが`complete=false`、NOT EXISTSを要求 | 「子が存在しない」ではなくClaim不成立／unknown。Providerの全件要求と実取得完全性は別 |
| **QP-05 Oracle相関** | Exact decimal +1.005、scale=2、half-evenの独立Expectedは1.00、half-up候補は1.01 | 候補とExpectedを同じ誤ったhalf-up処理で生成すれば見掛けPASS。符号対称MR単独も誤候補を検出しない |
| **QP-06 Store複合破損** | 保存`source`が期待digestと異なり、`receipt.json`も欠落 | repair/reuseを拒否し旧byte/欠落を保持。単なるcache MISSとして同keyへ書き戻さない |
| **QP-07 Grant時間差** | Provider Read前はGrant有効、Read後に失効してConsumerへ返す直前 | 現在権限の確認を必要とするClaimは拒否。pre/post二地点検査もABA・実外部egressの一般保証ではない |

QP-01〜04の`SELECT`等は説明用の**合成データと独立Expected**であり、実SQLGlot Worker／実DBでのPASSではない。QP-05は[#2514 TAO-01〜04](https://github.com/hat47x/tei-platform/issues/2514)のCandidate/Expected共有由来問題を再利用。QP-07は[Draft #2637](https://github.com/hat47x/tei-platform/pull/2637)のD2 pre/post再確認を参照するが、製品での実Concurrent失効・外部送信の保証には拡張しない。

### 9.2 F1との一致は独立Oracleではない

F1（full B）とS1（selective B）が同一の誤ったGenerator／誤ったOperator／共有Expectedを使用すれば、両者が一致しても同じ誤りを繰り返した可能性が残る。

P1の**二つの合格軸**を分ける。

1. **再現一致**：F1とS1が、対象Claim/Consumerに必要な出力・member・実読取byteで同等（比較対象と許容差を先に固定）。
2. **独立の正しさ**：F1もS1も、事前に別由来で固定したExpected/判別用反例を満たす。OracleのSource/Role/Operator/生成Prompt/Tool出力などがCandidateと共有される場合は`shared-origin/unknown`とし、**Agentの数やモデル名を独立Oracle数に数えない**（[#1445 H8-P15](https://github.com/hat47x/tei-platform/issues/1445)）。

Metamorphic Relationは補助的な判別試験。関係PASSを独立Expected PASSへ昇格させない。holdoutの秘匿ExpectedをCandidate生成側へ公開しない。Oracle根拠が未確認でも実Byteの保管・検査は可能だが、Claimの意味適合は`unknown`とする。

### 9.3 Store拒否と明示移行をcache MISSにまとめない

[#2600の破損/移行差分](https://github.com/hat47x/tei-platform/issues/2600#issuecomment-6579585273)の責務を引き継ぎ、QP-06に以下の観測を追加する。

| 既存Storeの観測 | 今回の扱い |
| --- | --- |
| 4ファイル読取り・digest・provenanceが一致 | Store内部整合のみ。Evidence・Consumer・現在資格を別に確認 |
| Receiptだけが欠落し、3ファイル・提示Receiptの照合が成立 | **通常reuseではない**。隔離Storeの候補照合→Ownerが選ぶ明示移行経路 |
| 保存source破損＋Receipt欠落 | 再利用・自動修復を拒否。旧ファイル不変 |
| I/O拒否などでbyteを読めない | **integrity unknown**。破損断定・欠落断定や上書きはしない |
| 事前検査後にbyte/path変更、またはWrite ACK不明 | 事前成功は後続成功を保証しない。再読取と競合・公開保証が別途必要 |

`readOutcome/digestObserved/mutationAttempted/postRead/reuseDecision/migrationDecision`は実験票の列。Reference Store `Put`の局所`reused=true`を今回Build省略・現在選抜PASSへ流用しない。#2016はClosedで、既存#1105/#2604/#2595/#2514の責務を維持する。

## 10. 開始条件・実験後の次判断

**先に実行するセット**：既存GoのF0/F1/S1と、S2内のQP-01（行成立）、QP-06（複合破損）、QP-05（Oracle共有由来）の**三方向**。QP-02〜04・07は意味・Provider・現在Grantの各Ownerが持つ局所Fixtureを参照して、必要なClaimだけ段階的に増やす。三方向は全て同一のGo統合試験で動かす必要はない。**実際にどの工程を実行し、どの証拠を共有できるか**を一つの比較票で追う。

**結果欄**：`fixed input revision; before/after dependency+selection scope; observed actual byte/unknown; artifact-member closure; current Consumer claim; full result; selective result; independent Expected+origin; joint-evidence status; current authority/disclosure; executed/reused reason; elapsed work+verification; fail/unknown/not-run`。比較元を揃えず局所の成功だけを合算しない。

**判定ルール**：
- F1≠S1（必要な意味結果）なら、その範囲のskipは不合格。F1=S1でも独立OracleがFAILなら両者不合格。
- Evidenceが不足する場合は計算済みbyteを捨てる必要はないが、ClaimのPASS/発行/実行資格は主張しない。
- Source/Store integrityがunknownの場合は、検証不能な過去物の再利用を保留し、既存安全手順による再取得・再実行と費用を記録。権限がない取得・再送はしない。
- 反例で誤reuseが0、不要な失効と再資格化費用が数えられ、一定数のcold/warm反復測定ができた後にのみ、性能改善の主張を評価する。**先に採用閾値を固定するかは#2600/#2018 Owner判断**。

**最新実装の位置づけ（2026-11-02 14:26 UTC照合）**：PR #2587はmainへ統合済みだが、全Consumerの必要member、公開世代、実Read/Disclosureの保証までは統合済みとしない。PR #2634は修正HEAD `3dfc140`のCI結果を確認する場合も**合成Analyzerを与える特性テスト**、PR #2633は独立Draft。PR #2642の署名側Source IndexはLoad時捕捉byteへ寄せたが、import/manifest束縛や原子的Snapshot、Build/Generatorの実消費byte一式を保証したわけではない。

**本追補の状態**：GitHubのコード/Issue/PR読取に基づくP0/P1受入案。QP-01〜07の本件での実SQL/TEI/Go Selector/Store/Provider/Host実行、F0/F1/S1、性能比較、Owner受理は**NOT RUN／NOT ADOPTED**。新しいPipeline／Cache／Policy Engine／Oracle Engine／正式Core Schemaを採用しない。

## 11. 2026-11-02後半追補：空結果の否定依存と実Consumer到達を実測へ

### 11.1 `[]`は「依存なし」ではない

**新たなS2の必須反例**：同じ`[]`を返したQueryでも、結果を導いた対象集合、Relation経路、binding、WHERE/EXISTS述語、Provider route、取得Scope/完全性、Snapshot、権限母集団が異なる場合がある。結果byteが同じことは、次回も空でよいという証明にならない。無関係な全Sourceの変更監視も必要とはしない。

| Probe | 最小の入力・変更 | 適切な判断 |
| --- | --- | --- |
| **QP-08 0→1→0** | `WHERE score>50`で空→新規該当行A→A削除で空 | 1行追加時の古い空Cacheを拒否。最後に空へ戻っても過去のReview/資格は自動復活しない |
| **QP-09 Relation付替え** | 親Aに子0件、子Bを親Aへ付替え、表示対象は親Aのまま | 非存在/NOT EXISTSや集計の結果を再評価。表示Field・親ID不変は十分でない |
| **QP-10 Route/Scope差** | 同じ空結果だがProvider P1は完全Read、P2は部分Read／Snapshot不明 | P2の不存在Claimはunknown。空byte一致を共同Cohort資格へ転写しない |
| **QP-11 Plugin解釈改版** | Source同byteでSyntax/Role/Operator Plugin pinまたは実装解釈が変更 | 依存するFunction/Ruleだけ再資格化候補。Pin/依存閉包が不明なら局所skipを許さない |
| **QP-12 Java全クラス** | Main.class不変、Main$Input.class等の依存クラスだけ変更 | Main.class単独digestの一致では実行集合の再利用を許さない。全class set/JAR＋実Consumer結果で判定 |

QP-08〜10は[#2582](https://github.com/hat47x/tei-platform/issues/2582)・[#2586](https://github.com/hat47x/tei-platform/issues/2586)の動的Context/Provider観測を参照する。QP-11は[Draft #2638](https://github.com/hat47x/tei-platform/pull/2638)のRole Bootstrap/Pin検討を利用するが、合成ReferenceのPASSをTEI実Pluginでの資格へ昇格しない。QP-12は[#2512](https://github.com/hat47x/tei-platform/issues/2512)・[#2388](https://github.com/hat47x/tei-platform/pull/2388)のJava全class問題を受け取る。各Ownerの実装・Oracleを重複実装しない。

### 11.2 生成結果から実Consumerまでの閉包を固定する

**Source/生成物を作る時の依存**と、**その後の実行時データを読む時の依存**は別に測る。実行前に次の実観測を固定し、観測できなければ`unknown`とする。

| 段階 | 実際に確認するもの | この段階だけでは保証しないもの |
| --- | --- | --- |
| Authoring/Load | Loadで捕捉したSource byte、Role/Operator pin、選択されたInput/Context | すべての後続Toolが同一byteを消費したこと |
| Tool選択 | **実際に選択された**CLI/Skill/Plugin、引数、Source/Workspace/Target | Tool候補の列挙や説明文だけによる実行資格 |
| Build/Generate | declared vs actual reads、Toolchain、依存member全件、失敗/未列挙の区別 | 生成物の公開世代・外部Consumerでの利用 |
| Publish | Manifestとmemberの実byte、既存ファイルの残存、公開操作の結果 | 原子的公開、Readerの同一世代読取 |
| Consumer | 必要memberと**実際に読んだmember**、外部Host/Providerの取得Scopeと現在Grant、結果/送出 | 実Readなしの利用可能性、Provider Commit/Receipt |

[#2587](https://github.com/hat47x/tei-platform/pull/2587)統合で`tei operations`/`tei artifacts`の実装入口が増えたが、同一Workspaceを指定しただけでは全段階のSource byte同一性を証明しない。[#2642](https://github.com/hat47x/tei-platform/pull/2642)のLoad時byte束縛は署名側の限定保証であり、他のTool/Buildへの自動伝播ではない。Artifact出力の個別書込みとManifest作成を原子的な公開と読み替えない。実Consumerが旧新混在memberを読めるなら、そのClaimを拒否するか、版固定・隔離公開の追加証拠が必要。

### 11.3 F0/F1/S1/S2への割当と停止線

1. **F0/F1**：同一固定fixtureで完全実行。単にOutput `[]`を比較せず、対象集合・Predicate・Relation path・Scope・Provider route・完全性・Target/Role pinを記録。
2. **S1**：QP-08〜12のうち依存不変が証明できる箇所のみ既存物を使う。今回未実行のNodeを今回`succeeded`へ偽装しない。
3. **S2**：`0→1→0`、付替え、部分取得、Plugin改版、Java副class変更を一軸ずつ投入。必要な失効と無関係な独立成果物の維持を両方検証。
4. **J0/J1/J2**：過去PASS/Owner Review/Disclosureを同じOutputに結合する誤りをJ0で再現し、J1/J2では現在のClaim依存だけ再資格化。F1=S1の一致と独立Oracle PASSを別々に確認。
5. **時間**：`dependency closure + actual read binding + required build + independent oracle + qualification + publish/read`を分け、実Consumerが利用可能になるまで測る。実測前に「高速化済み」と報告しない。

**追加受入条件**：少なくとも「空結果の誤再利用」「値/ID同一でもRelationや対象集合変更」「依存Plugin変更」「全class set差」の四種類で、既存物の**内容を保持すること**と**現在のClaimに使えること**が別々に判定されること。既存#2600のIB/QDC/PLケースへ割り当て、新たな依存Engine、Negative-Cache、Source Registry、Host、Oracle Engineは作らない。

**実証状態**：本節は既存チャットとGitHubの設計・局所試験の統合。QP-08〜12の本手順での実TEI F0/F1/S1/S2、実Provider、外部Consumer、Java生成物一式、性能測定、Owner意味受理は**NOT RUN**。main/PRの更新状況は観測時点に依存し、各局所CIの成功をこの一巡のPASSとしない。

## 12. 2026-11-02夜追補：意味不変のSource移動と「実際に到達した」資格の分離

**観測main** `2e16b2af9531225b73430a32a6dbb2031e07f4c1`。[#2626](https://github.com/hat47x/tei-platform/pull/2626)と[#2637](https://github.com/hat47x/tei-platform/pull/2637)は統合済み。[Draft #2660](https://github.com/hat47x/tei-platform/pull/2660)はSource→Operation→Artifact、保護Web Action→Service Carrier、Provider Read→Consumerの三つを**別々の縦断実証**とする提案。統合済み局所保証を本P1の共同PASSへ昇格しない。

### 12.1 三つの同一性を再利用判定票に追加

| 同一性 | 具体的に観測するもの | 変化時の判断 |
| --- | --- | --- |
| **Source occurrence / Trace** | 元ファイルのbyte、相対位置、出典参照、Source Index、ロード時の観測時点 | 文書引用・レビュー・Traceは再資格化。意味の変更とは直結させない |
| **Semantic / Function identity** | 受理済みMeaning/Role/Operator、正規化された内容、版・依存Pin、必要入力閉包 | 受理と依存が維持されたことを独立確認できる場合だけ計算再利用候補 |
| **Consumer execution / qualification** | 実CLI/Skill/Plugin選択と引数、actual consumed bytes、必要ArtifactSet、実Host Grant、Provider Read、実Consumer read/egress | 同じSemantic fingerprintだけで実行資格・副作用・公開資格を引き継がない |

**S2-SR01（Source移動）**：内容・受理Meaningを固定し、Sourceファイルの場所だけを変更する。実CLIでCanonical Model/World Fingerprintが維持されても、Source Index/Traceが変わるケースを対照にする。計算省略を許せる場合でもSHO引用・レビューは再確認。#2626統合は限定Fixtureの実CLI観測であり、全SHO納品の受理ではない。

**S2-SR02（同じWorkspaceの二回Load）**：`tei operations`と`tei artifacts`を同じパスで順次呼び、間にSourceを変更する。両コマンドのLoad/実消費Source版を記録し、同一パスだけで同一Snapshotを主張しない。両方のArtifact/Operationを同一Runの共同証拠として束ねるには、実byte・Revision・Rule/Tool版・Consumer範囲の一致を示す。

### 12.2 Evidence強度を段階として記録し、段階飛躍を禁止

`E0=宣言・設定・選択候補`、`E1=実際のTool/CLI起動とその入出力を観測`、`E2=対象Consumerの実読取・結果・必要な現在資格まで観測`を**この計画だけの観測ラベル**として用いる。これらは既存IssueのE0/E1/E2用語との正式な互換・製品enumを宣言しない。

| 観測 | 許される主張 | 追加証拠なしに許されない主張 |
| --- | --- | --- |
| E0 | Tool/Skill/Roleの候補が存在 | そのToolを実際に呼び出した、権限がある |
| E1 | 固定条件で当該CLI/Toolが起動し結果を返した | 別CLIと同Source、実Host効果、Consumerの必要member完備 |
| E2 | 特定Consumerが必要物を読んで期待結果を得た | 別Consumer・別Grant・別Snapshotへの一般化 |

**S2-SR03（CLI/Skill/Hostの混同）**：Roleが解決しSkill文書に操作が載っていても、CLI commandの存在、Tool discovery、実呼出し、Host Grant、効果/receiptを別列にする。Skillの記述を実行資格へ昇格させない。参照先[#2649](https://github.com/hat47x/tei-platform/pull/2649)、[#2605](https://github.com/hat47x/tei-platform/issues/2605)。

**S2-SR04（Read途中のGrant失効）**：Provider Field Read開始時のGrantは有効、応答直前に失効。[#2637](https://github.com/hat47x/tei-platform/pull/2637)はD2 Field Readの返却直前再確認をmainに統合済みだが、実Hostの同時失効、ABA、外部egressの一般保証ではない。結果byteが同じでも現在の返却資格を別に確認する。

### 12.3 次に実行する順序と受入線

1. **基準を再固定**：#2636は旧main `9c9ef4d...`から起票されたDraft。実行時のmain `2e16b2a...`との差分を列挙し、統合済み#2626/#2637/#2633の現行保証とDraft差分を別記。必要な試験を固定HEADで再実行するまでは「現在mainでPASS」を推測しない。
2. **最小の縦断**：最初はSource→Operation→Artifactのread-only合成Fixtureを一つ固定し、F0/F1/S1とS2-SR01/02を結ぶ。保護Action→Service Carrier、Provider Read→Consumerは独立Ownerの[#2660](https://github.com/hat47x/tei-platform/pull/2660)で後続検証。異なるRunの成功を一つのE2と呼ばない。
3. **必要な証拠**：実Source byte・Source occurrence・Meaning/Role/Operator pin・CLI版と引数・実Tool実行・Artifact必要member・公開世代・Consumer実Read・独立Expected・現在Grant・Evidence levelを、各対象Claimについて記録する。
4. **性能測定**：意味不変のSource移動では、計算省略ができてもTrace/SHOレビュー更新に費用がかかる。Buildだけの時間と、**利用可能な成果物・レビュー・現在資格が揃うまで**の時間を別々に測る。

**停止線**：PR #2626/#2637/#2633のmerge、局所GoテストPASS、World Fingerprint不変、Role/Skill候補表示、Sourceパス一致、Manifest内部整合のいずれも、単独ではE2共同Consumer PASSの十分条件ではない。Owner受理・実Host副作用・実Provider同一Snapshot・外部SHO納品の合格は別。

**状態**：本節はGitHubのmain/PRと他チャットの限定観測を受けた**実証計画**。S2-SR01〜04の本件でのF0/F1/S1/S2実行、CI、性能測定、E2の共同成立、正式Contract/Owner受理は**NOT RUN／NOT ADOPTED**。新Host・共通Registry・第二Canonical/Resolver・第二Build Graphを設けない。

## 13. 2026-11-03追補：外部Reference失効と副作用の不可逆性を増分処理の停止条件へ

**観測main**：`0e18b916aa15bc7768e9abfb34318b98ee497cb3`。本節は新しい再利用Engineを作る提案ではない。既存[#2743](https://github.com/hat47x/tei-platform/pull/2743)の外部Reference失効設計、[#2667](https://github.com/hat47x/tei-platform/pull/2667)の実CLI/SQL Worker収束、main統合済み[#2765](https://github.com/hat47x/tei-platform/pull/2765)・[#2769](https://github.com/hat47x/tei-platform/pull/2769)・[#2771](https://github.com/hat47x/tei-platform/pull/2771)・[#2773](https://github.com/hat47x/tei-platform/pull/2773)のEffect/観測/補償/Command Adapterの局所実装を**P1停止線**として接続する。局所PRの成功は本手順のF0/F1/S1/S2を実行した証拠ではない。

### 13.1 過去の事実は失効しないが、現在への適用は失効し得る

外部Referenceは、少なくとも`Source URL / revision / retrievedAt / checkedAt / content fingerprint / read completeness / permission outcome`を分離して観測する。対象の移動、内容改版、A→B→A、offline、権限変更で失効するのは一律に「全世界の過去」ではない。

| 記録・Claim | Source変更で扱うこと | やってはいけないこと |
| --- | --- | --- |
| **過去Observation** | 取得時点のbyteと時刻を履歴として保持 | 後の404/403を理由に、取得済み履歴が存在しなかったことにする |
| **Structural Candidate** | 出典・Parser/Plugin版・依存範囲を照合し、必要な候補だけ再抽出 | byte一致を構造候補の意味受理と同一視 |
| **Accepted Meaning** | 受理範囲とOwner Authorityを照合し、失効する受理を局所判定 | A→B→Aで過去のOwner受理を自動復活 |
| **Build Artifact/Evidence** | 依存閉包・Toolchain・Producer/Target・実消費byteを照合 | World fingerprintが同じだけで現在のBuild資格を復活 |
| **Execution Receipt/Effect** | 過去に起きた副作用・Receiptを履歴として保持し、現在の状態を別途観測 | Source失効により過去の外部副作用を「未実行」に戻す |

Source取得の`unavailable/partial/permission-denied/not-found`を空集合と同じにしない。外部参照がofflineで新鮮さを確認できない場合、古いArtifactの保管は可能でも、新しいConsumer Claimの成立は`unknown`または用途上の拒否とする。鮮度TTL、offline固定Buildの許容範囲、同内容改版時の再受理省略は#2743 Owner未決であり、ここで決定しない。

### 13.2 純粋な再生成と、外部Effectの再実行を分ける

BuildのS1 selectiveを、**業務Commandや外部Effectの再実行許可へ流用しない**。mainの`RunEffect`は能力呼出しの成功と独立Observerの`Observed`を分け、`SQLEffectObserver`は0行を未観測、読取エラーを観測失敗とする限定実装。Command Adapterは既存`effectcommit`境界に委譲するが、実PostgreSQL・実外部副作用・認可Grant発行を保証したものではない。

| S2 probe | 入力と独立Expected | 正しい処置 |
| --- | --- | --- |
| **EF-01 Source A→B→A** | Source byteがAへ戻るが、B期間中にOwner受理撤回・Plugin版変更 | byte/Artifact内容再利用は個別検査。Owner受理・実行資格は自動復活しない |
| **EF-02 Commit ACK喪失** | Effect実行後、応答またはReceipt取得に失敗。外部状態は不明 | `unknown outcome`として再観測・operation identity照合。Build再実行や単純retryで外部効果を重複させない |
| **EF-03 Observer空/失敗** | SQL Observer 0行、またはQuery error | 0行は`Observed=false`、エラーは観測失敗。どちらも「効果なし」「成功」と断定しない |
| **EF-04 Compensation失敗** | Effect失敗後の補償が失敗、または一部成功 | 元Effect/補償Attempt/失敗記録を保持。再生成成功を補償完了に転写しない |
| **EF-05 Reference権限変更** | 同byteの外部Sourceを再取得できず、既存Artifactは保存済み | 歴史的byteと現在の読取・開示許可を分け、許可なき再取得・開示を行わない |

### 13.3 最小の統合順序

**Gate A：read-only**。固定HEADでSource→Operation→ArtifactのF0/F1/S1を一巡し、外部Referenceは取得結果/時点/完全性を記録。Source/Meaning/Artifact/Consumerのどこを再利用したかを分ける。最初の完了条件は、実Go CLIと独立Readerの**同一Attempt**が確認できること。#2667の実SQL Workerが特定できない場合は`NOT RUN: producer-unavailable`とし、mockやSQLiteを実SQL Worker成功の代替にしない。

**Gate B：Effect観測**。Gate Aの純粋なBuild結果と独立したAttemptとして、既存Effect Ruleの`invoked / succeeded / observed / receipt / compensated`を観測。実DB/Hostの証拠がない限り、局所Go Fake Provider試験を外部効果のE2へ昇格させない。

**Gate C：費用測定**。Source/Reference再確認、局所Build、Artifact検証、Oracle、現在資格、Effect再観測を**別時間**として計測。外部Effect再実行を高速化KPIの「節約したBuild」に含めず、必要な認可・副作用回復が不明なら`usable result`到達を主張しない。

**停止条件**：実消費Sourceが不明、Referenceの完全性/権限が不明、必要memberが欠落、独立Oracleの根拠が共有、外部Effect outcome/Receiptが不明、現在Grantが不明の場合、該当するClaimをPASSにしない。部分的なArtifactや歴史的Receiptは保存できるが、現在の利用資格を付与しない。#2743/#2667/#1129/#2600の既存Ownerを維持し、新Host/Registry/承認Engine/Effect Schedulerを増設しない。

**状態**：設計・実行ゲートの追補のみ。EF-01〜05、実DB/実外部Host、F0/F1/S1/S2、独立Consumer、実効果回復、性能測定、Owner受理は本追補では**NOT RUN**。

## 14. 2026-11-03追補：Schema driftとCommand Targetを「同じSource」の反証に加える

**追加した実証対象は二つ**。Source byteが変わらなくても、SQLの解決先Schema、またはCommandの実Targetが変われば、再利用の適用範囲が変わる。前者はread-only計算の依存、後者は副作用の実行資格であり、両者を一つのCache KeyやPASSへ統合しない。

### 14.1 SQL SchemaはQuery Sourceの隠れた実入力

[#2778](https://github.com/hat47x/tei-platform/pull/2778)のSM-15は、**独立Python/SQLiteの局所反例**として、同一`NATURAL JOIN` SQL Sourceに対しSchemaへ同名の`status`列を追加すると結果が**2行→1行**、`SELECT *`の列が**3→4**になることを確認した（報告上14/14 PASS）。これはTEIの実SQLGlot Worker/Go DerivedIR/Provider/実DBが一巡してPASSした証拠ではない。

| Probe | 固定・変化させるもの | 独立Expected／拒否条件 |
| --- | --- | --- |
| **SD-01 NATURAL JOIN** | SQL byte固定。左右のSchemaで共通列`status`を追加 | JOIN成立条件の増加により行集合が変わり得る。SQL Source hash一致でQuery結果をreuseしない |
| **SD-02 SELECT star** | `SELECT *` byte固定。Schemaへ列追加 | Output Shape/Consumer必要Fieldの再評価。旧ShapeのArtifactSetだけでは現在のConsumer契約を満たさない |
| **SD-03 同名の解決先** | SQL字面固定。Search Path/Namespace/Relation bindingを変更 | Resolved object identity、Schema revision、列/Relation binding、実Provider routeを照合。不明なら`unknown` |
| **SD-04 静的事実と実DB** | DDL SourceとSchema観測を分ける。Migration適用成否は不明 | DDL宣言だけで実DB Schemaが変更済みとしない。Desired/Actual/Migration証拠を別列に保持 |

**再利用判定票への最小追加列**：`query source byte; schema revision; resolved relation/column occurrence; search path/namespace; query operator+lowering version; provider observed schema/route; output shape; consumer required fields`。全Sourceを無条件に失効させず、解決先が変わったQuery/Consumerを再資格化する。SQLの静的候補、実DB観測、Owner受理は[#1126](https://github.com/hat47x/tei-platform/issues/1126)/[#2387](https://github.com/hat47x/tei-platform/issues/2387)/[#2622](https://github.com/hat47x/tei-platform/issues/2622)に委譲し、汎用State TableやMigration Engineは追加しない。

### 14.2 Commandの実Targetは、PayloadとGrantから自動的に一致しない

[#2780](https://github.com/hat47x/tei-platform/pull/2780)は、実Goの`command.Invoker → commandadapter.Executor → effectcommit.Runner`に合成Provider/外側UnitOfWorkを接続した**特性試験**。exact-headのfocused GitHub Actionsで2/2 PASSと報告されているが、これは現行配線の**不安全な可能性を示す再現成功**であり、安全性PASSではない。

| Probe | 現行特性・入力 | 必要なOwner受入 |
| --- | --- | --- |
| **CT-01 Target drift** | Command Input `ORD-1`、明示Payload `ORD-2`でも合成書込みへ到達（MC-B01） | Command選択Target、Payload Target、Grant Scope、実Provider Requestの一致を副作用前に検査。異なる場合は拒否 |
| **CT-02 split commit** | 内側Runnerの合成Commit後、外側UnitOfWorkのCommit失敗（MC-C01） | 二つのTransactionを原子的な一Commitと主張しない。部分効果/ACK不明/Receiptを保持し、再実行可否を別判断 |

**F0/F1/S1/S2の停止線**：CT-01/02をBuild省略の「性能試験」として副作用を再実行しない。read-only FixtureでOperation/Target/Grantの差を観測するにとどめ、実Commitは#1174/#1119/#2602のOwnerと既存Effect境界で独立実証。対象不一致の現行Characterization PASSを仕様採用と解釈しない。実PostgreSQL・外部Host・Grant Issuer・二重Commit事故・回復性は未実証。

### 14.3 実装へ進める前の収束条件

1. 既存F0/F1/S1の実行Fixtureを一つ固定し、まず`SD-01/02`を**read-only**で再現。SQL SourceだけでなくSchema/Resolved Relationを入力閉包へ含めたとき、誤reuseを検出できるかを測る。
2. 同じ入力の独立Expectedを固定し、`F1=S1`と`F1/S1=Expected`を別々に検査。#2778のSQLite 14件をそのままTEI Product Oracleとして数えない。
3. `CT-01/02`は既存#2780の実Go負例と、#1174/#1119の実Provider/Receipt計画へ参照で渡す。Buildの選択的実行によってEffectの再試行資格を推論しない。
4. 計測票には`schema resolve + query impact + required rebuild + consumer read`と`effect/receipt recovery`を分ける。実SQL Worker不明なら`NOT RUN: producer-unavailable`。新しい依存エンジンやHostを採用しない。

**状態**：SM-15は隔離SQLiteの限定実測、MC-B01/C01は合成Goの特性観測。**本#2636での実TEI F0/F1/S1/S2、SQL Worker、実Provider、Consumer共同資格、性能測定、Owner受理はNOT RUN**。局所PASSを統合PASSへ転写しない。

## 15. 2026-11-03実行移行：反例追加を止め、最初のF0/F1/S1を成立させる

本節は§9〜14の反例を追加するのではなく、**実行可能な最小一巡と判定票**へ収束させる。最初の対象は副作用のないQuery/Artifact経路とし、SQL Schema drift（SD-01/02）を一軸変更に採用する。実SQLGlot Workerの所在が確認できない場合は合成Go DerivedIRと独立SQLiteを**別々の限定証拠**として保存し、E2Eを成功扱いしない。

### 15.1 最初の実行票：1 fixture × 3実行 × 2独立判定

| 実行 | 入力と固定条件 | 記録するもの | 合格・失敗の判定 |
| --- | --- | --- | --- |
| **F0** | SQL/Schema A、Toolchain/Role/Operator/Provider固定、cold full | Source・Schema・Resolved Relation・Producer版、全member byte、Consumer出力、経過時間 | 独立Expected Aとの一致。Consumer実読取がなければ`NOT RUN` |
| **F1** | SQL同byte、Schema B（NATURAL JOINの共通列追加）、full | F0と同列＋出力行集合・shapeの変化、時間 | 独立Expected Bとの一致。2行→1行はSM-15の**SQLite限定**対照値 |
| **S1** | F1と同一Schema B、選択的再利用、独立した新Run | 実行/省略したNode、全必要member、実Reader、時間、失効判定 | `F1=S1`かつ両者が独立Expected Bと一致。省略したNodeを当Runの`succeeded`と偽記録しない |
| **S2** | F0へ戻す／Schema解決先・Scope・部分取得など一軸ずつ | 誤reuse、過剰失効、未観測・不明の理由 | 危険なreuseを拒否し、独立した無影響memberを維持できるかを別々に判定 |

**独立Expectedの条件**：実SQLGlot→Go IRの結果やF1の生成結果からExpectedを自動生成しない。SM-15のSQLiteは独立SQL実行の限定対照として利用できるが、同一E2Eの実Provider/Consumer Expectedとは呼ばない。Source/Schema/Provider/Targetの選択を実行前に固定し、比較後にExpectedを修正しない。

### 15.2 一つのRunに必要な記録

各Runに`run_id; git HEAD; fixture revision; source bytes digest; physical schema revision; resolved relations/columns; parser+worker version; Role/Operator pin; build/toolchain; actual reads; built/reused/unknown nodes; required ArtifactSet members and actual reader bytes; consumer result; independent expected origin; reference freshness/completeness; current grant (該当時); durations; status; reason`を記録する。取得できない列は推測値で埋めず`unknown/not-run`とする。

**出力資格は三つの判定**を独立に残す：①artifact byte/shape一致、②Consumer結果と独立Expected一致、③現在のSource/Reference/Grant/公開世代への適用資格。①だけのPASSは②③のPASSではない。

### 15.3 実行担当と着手条件

- **P0：実SQL Producer所在** — [#2387](https://github.com/hat47x/tei-platform/issues/2387) QX-33のWorker実体、起動、stdin/stdout、実依存を特定する。未特定なら`NOT RUN: producer-unavailable`を明記し、合成analyzer.Resultを代用しない。
- **P1：最初のread-only三実行** — [#2778](https://github.com/hat47x/tei-platform/pull/2778)のSM-15（独立SQLite 14/14という限定報告）とSM-16（実Go`query.DeriveFromAnalysis`へ合成Resultを投入、exact-head CI未確認）を**別証拠**として受領。実Workerが揃ってからF0/F1/S1を同じ固定Toolchainで実行する。
- **P2：実Consumerの資格** — [#2667](https://github.com/hat47x/tei-platform/pull/2667)の実CLI`validate→operations→artifacts→fresh -out→independent Reader`を同一Attemptで通し、必要member・実読取・Source Revisionを確認する。
- **P3：副作用の別Gate** — [#2780](https://github.com/hat47x/tei-platform/pull/2780)のMC-B/C/D特性試験と既存PostgreSQL ReferenceのReceipt/Replayを別証拠として受領する。Command Adapterの`committed/replayed`が同一の空`command.Result{}`へ写される現行挙動、型付き`CommitUncertainError`を区別する。Build性能実験から実Effectを呼び出さない。

**終了条件**：F0/F1/S1について同一HEAD・同一Toolchainの実行ログ、必要ArtifactSetと独立Consumer、独立Expected、誤reuse/過剰失効、cold/warmおよび`edit→usable`時間が揃った場合のみ、実証を完了と報告する。揃わない場合は不足した境界を列挙し、合成局所試験のPASSを総合PASSへ転写しない。既存#2600/#2387/#2667/#2778/#2780のOwnerに受け渡し、新しいProduct Gate/Host/Cache Engineを設けない。

**現在状態**：本節は実行指示・判定票の追加。実Worker→Go IR→実Provider→Consumerの一巡、F0/F1/S1実測、性能比較、Owner受理は**NOT RUN**。

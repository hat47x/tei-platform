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

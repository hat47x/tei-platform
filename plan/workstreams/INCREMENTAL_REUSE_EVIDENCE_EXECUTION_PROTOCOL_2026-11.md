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

**B. 出力に現れないRule/Query依存の変更**：表示`amount=600`のまま、隠れた`credit_limit`が700→500になればValidationは再評価。SQLでも`WHERE`、`ORDER BY`、相関/外側参照がOutputField列に現れない場合がある（[Draft #2634 QX-27](https://github.com/hat47x/tei-platform/pull/2634)は実Go候補への特性試験だが新テスト結果は未確認）。**出力Field集合だけを依存閉包とみなさない**。#2624の表示/Rule footprintは独立合成Referenceであり、Providerの実取得は#2628へ。

**C. 生成物内部整合と実Consumer必要集合**：Generic Manifestが`Complete`でも期待memberが欠落し得る。[Draft #2633](https://github.com/hat47x/tei-platform/pull/2633)の6つのGo特性テストは候補PR #2587上の検査限界を扱う。実行・CI確認前に製品の検証済み不具合と呼ばない。公開時の世代混在・Manifest→relative path→byte→Readerは#2604、Consumer相対必要Kind/SubjectはDraft #2630で独立に検査する。

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

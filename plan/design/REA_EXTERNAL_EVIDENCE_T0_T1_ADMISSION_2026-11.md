# REA外部EvidenceのT0/T1-A受入実験：再現可能な失敗・部分結果・出典の保存

- **状態**：実装前の実験手順／独立Expected候補。実行、Provider資格、TEI正式Source化、意味受理、旧新同値を示す文書ではない。
- **Owner**：[TEI #3394](https://github.com/hat47x/tei-platform/issues/3394)（外部観測と候補写像）。横断の同一Attempt証拠は[#1163](https://github.com/hat47x/tei-platform/issues/1163)、Source/Referenceは既存Workspace担当、Source移行は[#2621](https://github.com/hat47x/tei-platform/issues/2621)、Common APIは[ADR-0048](../adr/0048-shared-application-api-and-transport-adapters.md)／[#1158](https://github.com/hat47x/tei-platform/issues/1158)。
- **目的**：新Runner/REA独自Parser/第二Evidence Registryを作らず、外部プロセスの**実解析結果**とその出典・partial・unknownを限定ReadでTEI候補へ運べるか検証する。最初はTEI Productへ接続したとは主張しない。
- **上流仕様の固定参照**：[REA CI実行例](https://github.com/morluto/rea/blob/main/docs/ci-analysis.md)／[workflow](https://github.com/morluto/rea/blob/main/examples/github-actions/rea-analysis.yml)／[CLI](https://github.com/morluto/rea/blob/main/docs/getting-started/cli.md)／[JS静的解析](https://github.com/morluto/rea/blob/main/docs/guides/javascript/javascript-artifact-reconstruction.md)。実行時は参照したGit commit・package version・依存lockとAction commitを別に固定する。

## 1. REA T0の入口を「最小既知対照」へ修正する

初回T0は実Electronアプリを必須にせず、**REA上流の自己完結型GitHub Actions静的解析例**を独立対照として再利用する。上流手順によれば、実行せずに読み取る小規模JavaScriptディレクトリを生成し、`analysis.json`（標準出力）、`analysis.stderr`（進捗・診断）、`exit-code.txt`（終了コード）を保存する。現時点の上流文書例はNode.js **24.22.1**、REA **6.7.0**、GitHub Action revision固定、`workflow_dispatch`、`contents: read`、artifact retention 7日。これは**上流参考例の版**であり、本Repoでの実Go/Actions検証済み版ではない。上流に依存するnpm推移版が必要なら、レビュー済みlockと`npm ci`で固定強度を明示する。

初期T0は(1)上流workflowを出典と共に固定し差分を監査、(2)最小ディレクトリの**静的解析だけ**を実行、(3)出力のraw byte/digestと実exitを保存、(4)別のreadbackでbounded summary/pageを再取得、(5)正常／非0／partialを独立Expectedと照合する。秘密または私有Source、権限のない解析物、外部アップロードは扱わない。

**T0はSource自体のコードを起動しない**。ただしworkflowのビルド処理・npm install・REAプロセスは実行するため、外部依存の信頼性、実RunnerのOS/Node/ユーザー権限、読取り・出力ディレクトリの副作用を別記録する。上流workflowをそのまま本番の`pull_request_target`へ流用しない。静的成功を解析の完全性や情報開示資格とみなさない。

### T0の独立負例（実行前の期待票）

| ID | 一変更・対照 | 期待する局所結果・禁止する断定 |
|---|---|---|
| T0-01 | 上流の有効な小さなJSディレクトリ | 実stdout・stderr・exit別保存とreadbackに成功。Evidence内のcoverage/unknownを別照合するまでは全量観測とはしない |
| T0-02 | `ARTIFACT_PATH=build/missing-artifact` | 非0、利用可能な診断保存。失敗を`|| true`/continue-on-errorで正常化しない |
| T0-03 | directoryでなく単一`index.js`を分析入力に指定 | 上流記載の`unsupported_target`相当を区別。自動的に別Providerへ切替えてPASSにしない |
| T0-04 | デモJSへ`export const broken = ;`を追加 | 上流REA 6.7.0の`parse_failures`と不完全coverageを再現するか検査。**プロセスexit0でもcompleteを拒否** |
| T0-05 | EvidenceのJSONL object-collection終端metadataを欠落、またはJSONL外形不適合 | 取得した行のみ局所観測。全行受領を拒否。REAのJSONL選択対象によっては形式自体が不許可なので、JSONの元Evidenceを正本とする |
| T0-06 | 観測原本は同byteで解析profile/Provider版だけ変更 | 外部観測は別version/attempt、内容同値から実Provider/Unknownの同一性を推論しない |

**注意**：T0-02〜04はREA上流に公開された操作例・期待観測をTEIが独立に再実行するための基準であり、TEI Repoで実行した実績ではない。失敗したrunでも保存済み診断を消さないが、Runner loss/cancelによりartifact upload自体が不能な状態は別`unknown`。

## 2. 一Attemptの実観測列（新たな正式schemaの提案ではない）

| 境界 | 最低限保持する実観測 |
|---|---|
| Source selection | requestで指定したpathと解決先、symlink alias、source raw SHA256、file inventory/Required source closure、データ取得時点 |
| Executable / authority | REA actual version+実binary digest、Node/OS、選択capability/provider、実invoked command/argv、Host許可Rootと作用 |
| REA output | process exit/stderr/stdout、Evidence raw digest、format/shape/coverage/limitations/semantic unknown、終了metadataとreadback結果 |
| Consumer | consumer revision/World/SourceIndex、provider raw source+manifest digest、実取得location/time、selected vs invoked、received record count/end |
| TEI proposal | External Observation IDとTEI Candidate IDの出典付き写像、loss/ambiguous/unknown、Authoring `*.tei.md`への明示的選択（T1-Bは別） |
| Disclosure | 実保存先・アクセス権・必要保持期間、表示/外部AI送出の現在資格、伏せるID/path/件数 |
| Qualification | T0 run局所PASS、T1-A external evidence qualified、T1-B TEI Loader/Parsed/Role、Owner Accepted、SHO Review、Target run/Expectedを独立列 |

`World/SourceIndex Fingerprint`だけを全外部Sourceの一時点Snapshotとみなさない。[main統合済み #3479](https://github.com/hat47x/tei-platform/pull/3479)のProvider raw Source-only/Manifest-only driftではConsumer側の値が同じまま外部入力のbyteが変わり得る。古いConsumer Resultを用いた後日のProvider navigationを、古いAttemptの同一原子Source読込みと結び付けない（#3394 RE-25〜27）。同byteのA→B→Aでも前のReview/Grantの自動復活を禁止する。

## 3. T1-Aの入場・実装差分を限定する

**初回はEvidenceの入力元を明示するread-only typed projectionだけ**を比較する。REA JSONを`*.tei.md`にrenameしてLoaderへ直接投入しない。Source原本・REA Evidence・TEI Structural Candidate・Owner Accepted Meaning・SHO Documentを別Identityとして扱う。REA固有の新Semantic文法は採用しない。

- **T1-A正例**：権利確認済みのJS/アプリSourceと、独立したSQLite snapshot/SQL（計二Source以上）を、事前に固定した同一read-only Functionに関連付ける。片方の観測のみではFunction completeではない。SQLのstatic`SELECT/UPDATE`発見は実Provider Read/実Write Effectを示さない。
- **T1-A負例**：Source selectionのalias違い、Source rawのみ変更、Provider manifestだけ変更、partial/zero result、複数Relation同端点、別Sourceの読取時点不一致、selected provider Bなのに実invoke A、必要な実Row Read未実行。該当既存Expectedは#3394 RE-01〜29、#2514/#1180、#3270/#3263。
- **T1-A出力**：出典/identity/版/uncertainty付き外部Observationまたは候補を**既存のApplication操作/Provider契約に載るか検査**。必要な型差分は#1158/#3419、実選択ProviderのBind/Invokeは#1251/#951へ戻す。既存の`validate` CLI/MCP一致（merged #3458）をREAの実Provider適合と混同しない。
- **T1-Bの停止線**：正式`*.tei.md` Loader/Parser/Roleが未接続なら`parser-unbound`。SHO Raw Draftの保存は許可するが、TEI Source/Accepted Meaning/Build適格とは言わない。

## 4. 実行判断・責務・計測

**最短順**：T0-01〜04を上流既知対照でpinした同一環境に実行→必要ならT0-05/06→最初のJS+SQLite二Source T1-A→選択一FunctionのT1-B→#1163のW1〜W5で三視点/SHO/Targetに合流。T0の実験は、TEI本線のP0-R Function→Query→DB実ReadとP0-B Inspector実選択の完了を待つ必要はない。T1-A以降は両本線の実証証拠を流用して、新しいRunner/Host/Parser/Registryを作らない。

実測する指標は、(a)正/負例の独立判別力、(b)unknown消失・false complete件数、(c)readbackのsource/revision/record coverage、(d)再解析cold/warm実時間とEvidence byte量、(e)Source driftの過剰失効と誤再利用、(f)外部送出や保存の許可違反、(g)必要な新共通API型差分（最小化）。旧Runtimeを実行しない場合は**legacy equivalence unknown**。旧Actual一致とOwner Accepted Desired一致は別（#3394 RE-29、#2667 QML）。

**今回の証拠上限**：この文書・PRはT0実実行、REA source artifact取得、TEI外部Adapter実装、Go/CI、T1-A/B、SHO Writer、旧新Target同値、正式Owner受理を完了したとは主張しない。Issue/ADRを実装の正本として残し、現行main実装とDraftを混ぜない。


---

<!-- rea-f0-f5-qualification-20261108 -->
## 5. 他領域の成熟判断を統合：T0/T1-Aに対するF0〜F5の六面資格

2026-11-08採用済み[ADR-0048](../adr/0048-shared-application-api-and-transport-adapters.md)／[TEI #1158](https://github.com/hat47x/tei-platform/issues/1158)の共通Operation、[#1119](https://github.com/hat47x/tei-platform/issues/1119)の製品受入、[#1163](https://github.com/hat47x/tei-platform/issues/1163)の一Attempt Witnessへ接続する。以下のF0〜F5は**検証の観点**であって「必ずこの順に物理実行する」というWorkflowや新Gateではない。例えば解析Providerの呼出し（F3）はJSON表現の生成（F2）に先立つ。操作の実行順は既存Operation Lifecycleに従う。

| 面 | 一Functionで要求する実観測 | 検出すべき偽PASS | 責務と最低停止条件 |
|---|---|---|---|
| **F0 Source** | Caller選択path≠実resolved path、実raw byte/digest、Provider manifest、対象集合/時点、必要な複数Source・Dependency closure | Consumer World/Indexが同じだから外部Provider byteも同じ | 既存Workspace/Source（main [#3479](https://github.com/hat47x/tei-platform/pull/3479) Source-only drift負例）。raw/manifest不足ならexternal closure=unknown |
| **F1 API** | 型付きRequest、Operation ID/version、Hostに固定したScope、入力検証、Result/Failure、CLI/MCP経路 | CLI JSON成功がMCP/Studio受領または業務意味合格を示す | #1158/#3419〜3422。main [#3458](https://github.com/hat47x/tei-platform/pull/3458)は`validate`限定一致。REA操作の共通API型は必要差分が実観測されるまで新設しない |
| **F2 Representation** | raw REA Evidence→bounded view→外部観測候補の写像、参照ID/Source位置、Loss/Unknown/NULL/部分Scope、三視点表示の情報損失 | Evidence JSONを`*.tei.md`へ改名・Graphを描画したことで意味受理/情報完全 | #2621/#3263/#2497。未受理のMapping、未観測Relation/Functionはcandidate/unknownで停止 |
| **F3 Provider** | REA実binary/版、selected Providerと実loaded/invoked、CLI実argv、同じProfileとscope、実出力Source | 選択Bなのに実Call A、または静的sample成功を別Provider能力とみなす | #1251/#951/#1016。selected/loaded/invoked/receivedのどれが未確認か記録 |
| **F4 Effect** | 実入力以外のFS書込み（report/tmp/lockを含む）・ネットワーク送出/権限/隔離、外部DBへの副作用の有無、実Resultと独立Expected | `read-only`指定から「どんなファイルも書かない」「安全」「Business Effect観測」と推定 | #1174/#2509/#2515／Host。無許可の実外部送出は禁止、old actual/desired/target actualは別判定 |
| **F5 Delivery** | プロセスstdout/stderr/exit、raw output byte長・digest、JSON/JSONL終端、Tool Call↔Reply、別Consumerの実Read、現在Disclosure | 出力ファイルの存在や標準出力の一部取得で全件完備・Consumer受領とみなす | #1163/#3262／SHO #6。途中切断はpartial・未受領はunknown、成功でも未確認の母集団をcompleteとしない |

**入場の判断**：T0ではF0/F3/F4/F5をまず直接観測し、F1/F2は対象範囲内だけ評価。T1-AではF0〜F5を一Functionの同一Scope/Attemptへ束縛する。**T1-B（正式`*.tei.md`のLoader/RoleとOwner Accepted Meaning）は別資格**。関係者へ渡す表は各面のPASS/FAIL/partial/unknown/NOT RUNと、どのEvidenceがその判定を支えるかを併記する。主張を六面の「総合PASS」へ無条件集約しない。

## 6. JSON/JSONLの終端・Coverageを独立に扱う実用手順

[REA CLIの上流仕様](https://github.com/morluto/rea/blob/main/docs/getting-started/cli.md)は、JSONLを**行志向の選択されたobject collection**に使う。複数collectionがある場合は明示選択が必要で、任意のreportやcolumnar rowsはJSONL投影できない。**bare top-level arrayにはmetadata終端が付かない**ため、どのJSONLにも終端制御行があると仮定する監査は誤り。first-class原本は保存済みの`analysis.json`とし、JSONLは限定Projectionとする。

object collectionのJSONLでは最後に`{\"$rea\":\"jsonl/v1\",\"type\":\"metadata\"}`型のcontrol recordが存在し、`records.emitted/source`や`context.normalized_result.coverage`等を確認する。ただし`emitted==source`でも解析Source全量・Evidence完全とは限らず、`coverage.exhausted==false`なら次ページ/未探索を保持する。JSONLの`type:error`、非0 exit、control record欠落、metadataと実行版/母集団の不一致はそれぞれ別Failureにする。**null/empty/未取得/解析対象外/権限拒否の区別を最後の文章化まで維持**する。

T0-04では上流REA 6.7.0の不正JS例について、まずraw JSONから`data.normalized_result.statistics.parse_failures`とcoverage limitationを確認し、その後で別Readerが部分結果をどう扱うかを比較する。上流に存在する欄を**別Provider/別版の必須Schemaとして決め打ちしない**。

## 7. 公開・隔離とSnapshotの設計停止線

- **非信頼入力**：元のSource、解析された文字列/コメント、出力Evidence、AIへ渡す抜粋は未信頼データ。解析結果中の説明文やコマンド風文字列を次のTool/Host命令へ昇格させない。REAのstatic分析対象を起動せず、外部CLIはHostが明示した固定実行物・argv・許可root・出力先で動かす。
- **再現性**：上流example workflowは固定版参考。npm依存・runner画像・入力bytes・実Binary/Node/解析設定・実exitをそれぞれPinする。外部解析をNetwork接続可能な機密Workspaceで無許可に実行しない。GitHub artifactの保存先・公開範囲・retentionを明示し、原本の全量を外部AI/CIへ自動送出しない。
- **外部Provider revision**：main [#3479](https://github.com/hat47x/tei-platform/pull/3479)のGo testはImported Provider raw/manifestの差分をConsumer World/Indexが見落とし得る**負例の限定確認**であり、外部Input Closureの完全なSnapshot実装ではない。 `WORKSPACE_IMPORTS_NOT_BOUND`を無効化して受理しない。位置が同じ・hashが同じ・値がA→B→Aで戻ることは、当時のGrant/Review/受理資格の復活ではない。
- **原文改訂と納品を分離**：TEI Source Working Copy、受理済みMeaning、SHO Raw Draft/Document Revision、生成Artifact、公開/引渡し済みRevisionを混ぜない。出典付きの図表は**固定される構造／明示的に拡張できる表現**の境界を表示し、Raw Evidenceから業務意味・CRUD操作・実Writerを自動採用しない。SHOの通常Markdown保存はREA/TEI不在でも独立継続。
- **旧新同値でなくDesiredも検証**：Legacy Observed Actual／Owner Accepted Desired／Target Observed Actualの三者を#2514/#1180の独立Expectedで比較。旧Runtimeを観測していなければlegacy-equivalence=unknown、新Targetが旧bugを複製した場合でもDesired適合へ自動合格にしない。

## 8. 次の一回の実行に必要な証拠と差分受渡し

**T0の実行票**：`upstream workflow commit + rea package version/actual CLI digest + Node/OS/runner + input source inventory/sha256 + argv + process stdout/stderr/exit + output raw sha256/readback + stats/coverage/limitations + known Expected + unknowns + actual CI/run ID`。一つのRunでT0-01正常とT0-04部分結果の比較を優先し、T0-02/03は非0/unsupportedの反証として残す。T0-05/06は別の限定Runでもよいが、別Runの結果をT1-A「同じSource Cohort/同一Attempt」へ足し合わせない。

**実装の最初のMissing Edgeの調査**：REA上流が出したEvidenceを、既存read-only操作に**観測データとして投影**できるか先に見極める。型不足があれば最小のRequest/Resultの差だけ#1158/#3419へ、Provider selected→invoked差は#1251へ、Source-import closure差は#1163/Workspaceへ戻す。SHOへはSource-backed文書候補だけを#6経由で渡し、新しいSource Registry、意味正本、Runnerを追加しない。

**この追補も設計文書の追加だけ**であり、REA T0/実Tool Call、CLI/MCP/Studio取込、`*.tei.md`Reader/Parser、SQL/DB実Provider、SHO Writer、旧Runtime比較、正式Owner意味受理は**NOT RUN**。main統合済みの#3479/#3458/#3482が持つ局所実績を、新しいREA縦断E2E実証と混ぜない。


---

<!-- rea-event-flow-topology-intake-20261108 -->
## 9. 逆解析からフロー・図表への写像：静的発見と実動作を分ける

TEIの三視点統合[#1124](https://github.com/hat47x/tei-platform/issues/1124)、業務手順[#1125](https://github.com/hat47x/tei-platform/issues/1125)、[Interaction Flow #3503](https://github.com/hat47x/tei-platform/issues/3503)、汎用Representation[#2497](https://github.com/hat47x/tei-platform/issues/2497)を既存Ownerとする。逆解析で発見した画面遷移、イベントハンドラ、SQLの更新候補、業務手順らしい呼出し列は、**同じFlow種類ではない**。少なくとも **Interaction Flow／Business Operation Flow／Adaptive Activity・Lifecycle** の役割と、Data/Function/Businessの三視点における対応位置を**候補**として区別する。用語・既定Flow Schema・Generatorの正式採用はそれぞれのOwnerが行い、本T0/T1-A実験で第二Flow DSLを新設しない。

表示上は「全体像→部分Graph/表→Source occurrence」というナビゲーションを持ち、利用者に専用Domain用語の事前理解を要求しないことを受入圧力に含める。各Node/EdgeにはSource Evidence参照、観測と推論の区別、Relation identity/向き/多重性、取得した版/Scope、不確実性、現在のOwner Reviewを示す。**標準的に固定する遷移・Guard・Hook Slotと、Profile/案件側が拡張可能な部分**を区別するが、見つかった外部コードの分岐から「TEI標準固定部」が成立したとは推論しない。専門Web/Chat/VUIなどの媒体別Flowは、共通Operationの意味と分離して必要に応じて投影する。

| 対照 | 外部解析で観測し得るもの | 正しく保留すべき内容 |
|---|---|---|
| **FLOW-A** | `onclick`→handler→API候補の静的参照 | 実画面で押下・送信・状態遷移した事実／唯一の実Path |
| **FLOW-B** | SQL UPDATEや保存関数へのコード参照 | 旧DBへ実Commitしたこと／業務操作として受理済みであること |
| **FLOW-C** | 非同期callbackや条件分岐、同名イベント複数出現 | 実行順・再試行回数・対話状態の一意性 |
| **FLOW-D** | 同一Node間に異なるRelation IDで複数Edge | 一本へ縮約しない。編集先Sourceが一意でない間はread-only |
| **FLOW-E** | Ownerが設定した標準/拡張Hook候補と、旧コード中の挿入箇所 | 挿入位置の一致をHookの実Contract・Provider/Authorityへ昇格させない |

上記は**T1-AでSource-backed構造候補を投影する追加圧力**であって、T0 REA静的分析のPASS条件を拡大しない。実際のFlow実装・Generator・異媒体のOperation/Scenario E2Eは[#3503](https://github.com/hat47x/tei-platform/issues/3503)/[#1180](https://github.com/hat47x/tei-platform/issues/1180)等の責務とする。未実行の実UI/実Commit/旧Runtime Actuationはunknown。


---

<!-- rea-t0-readonly-consumer-sho-e0-w2-20261108 -->
## 10. 保存済みREA出力の独立Consumer試験（Test-only実装）とSHO編集の適格性

**実装**：本Draftの`tools/rea-t0-saved-report/audit.py`と`test_audit.py`は、REAを起動しない読取り専用の**テストConsumer**。REA上流の `analysis.json` / `analysis.stderr` / `exit-code.txt` を対象に、実終了コードとreport parsingを別列にし、raw byte長/SHA-256、標準出力・診断の別保管、`parse_failures`の限定検出、JSONL object collectionのterminal metadata/row countを監査する。`jsonl`のbare top-level arrayには終端metadataがないという上流仕様を保持し、どちらのJSONL形式でも**Source全件完備を主張しない**。

**実行方法（試験用・製品CLIコマンドではない）**：
```bash
python3 -m unittest discover -s tools/rea-t0-saved-report -v
python3 tools/rea-t0-saved-report/audit.py /absolute/path/to/saved-rea-report
python3 tools/rea-t0-saved-report/audit.py /absolute/path/to/saved-rea-report \
  --jsonl-view /absolute/path/to/view.jsonl --jsonl-object-collection
```
レポートは小さな公開T0入力に限定。Python監査プログラムは`analysis.json`が16MiBを超えると拒否するため、数百MiB級の実Evidenceを読み切る汎用Parserではない。対象のディレクトリ内Symlink Report・入力としての不正JSON・非正規exit codeを拒否し、Source/Evidence値を出力せず観測メタデータだけ返す。**rootの実Path・ファイル開閉のTOCTOU/ACL保証・Provider実実行検証の機能ではない**。Provider version/input source/manifest/scope/Host authorizationが取得されていなければ`not-observed`を返す。新規Runner/Registry/Authorityや製品型は作らない。

**独立Expectedの実行結果（この作業環境、GitHub CIではない）**：2026-11-08の作業環境 Python 3.13.5で `python -m unittest discover -s /mnt/data/rea_t0_probe -v` の**11/11 PASS**と`py_compile`成功。これはGitHub branchに登録する前の同内容Test-onlyコードに対する**合成JSON/JSONL**の試験。実REAプロセス・実Provider/version・本Repo exact-head Actions CIに対する資格を与えない。新しいGitHub上のソースを再読込みし、その差分と試験内容を照合すること。

**SHO Writerへの接続が初めて具体化した点**：
- [SHO Draft #29](https://github.com/hat47x/sho-composer/pull/29)のEditor UTF-16選択→原文UTF-8 byte span（E0-MAP、17件合成試験）は、REAが同じDocumentの二番目のOccurrenceを示した場合のSource Position受渡しに活用する。ただし実Editorの`isDirty/version`同期観測、同Rawだが別Originの出典一意性、Writerの物理Patchは未成立。
- [SHO Draft #28](https://github.com/hat47x/sho-composer/pull/28)のSave As relative-link preflight（SAS-01〜11）は、Evidence文書のpath移動時に、Raw SHAとMarkdownリンク文字列が同じでも参照先が変わる反例に対応。現在はMarkdown G1bで把握できるリンクの**名目的な差分**のみで、実解決先・symlink・HTML/特殊リンク・正式Save As・Review/Disclosureへは拡張しない。
- 両Draftの限定CI成功を足しても、REA Source Referenceの現在性、TEI Semantic Acceptance、SHO正式Writer Save/別Process reopen、外部送出Grantは合格しない。**REA Evidence ID／TEI Candidate/Accepted ID／SHO Document+Occurrence IDは別名前空間**。出典と編集対象が同一であることは`Raw digest一致`だけでは保証されない。

**次の実験**：T0公式保存ファイルに本試験Consumerを実接続し、`analysis.json`と実SHA/exit/coverageを現物で確認する。T1-AではProvider Source/Manifest read cohortと候補Scopeを別取得し、T1-Bにはパーサ受理・Owner Meaningの根拠を独立のまま渡す。SHOに渡す時点ではsource locator、byte span、現在Document URI/versionとEditor dirty、Save As後の相対リンク変化を独立判定。これはテストConsumerの観点であり、新仕様上の必須API Schemaとしない。

**この第10節を作成した時点の資格上限（履歴）**：実行済みは**合成Python test-only監査の11件**のみ。その後の18件への拡張は第11節に記載。実REA T0の解析とGitHub Actions、REA→TEI実Plugin/CLI/Studio、SHO正式Editor保存・Writer、T1-A/B/T2、旧新実比較、Owner受理は**NOT RUN**。


---

<!-- rea-t0-process-report-conflict-s2-progress-20261108 -->
## 11. 実際に見つかった監査の抜けを閉じる：報告・制御レコード整合と現行main引継ぎ

**今回変更した試験用コード**：`tools/rea-t0-saved-report/audit.py`／`test_audit.py` を既存Draft内で更新。独立した監査に、(a) process `exit-code.txt`とREA JSON envelope `ok`の矛盾 `processReportConsistency=conflict`（`reportedOK`はboolのみ）、(b) REA JSONLの `{\"$rea\":\"jsonl/v1\",\"type\":\"error\"}`をデータ行として数えない、(c) JSONLの重複・終端より前の制御レコードを`extra-control-record`として拒否、(d) 非UTF-8 JSONLの`invalid-jsonl`分類、(e) report root自体がsymlinkの場合の拒否を追加。解析結果の「成功／失敗」「解析の網羅性」「Consumerへの配送完了」は依然として**別Claim**。

**独立反例の更新**：11件→**18件**。追加7件は、終了0＋JSON `ok=false`、非0＋`ok=true`、JSONL `type=error`をobject/bare両選択で判別、duplicate metadata、invalid UTF-8、symlink root。既存のexit0＋parse_failures>0、empty、missing exit、truncated JSON、bare array非metadata等は維持。**parse_failures=0やmetadata数一致を全Source解析完備としない**。JSONL制御行が存在しても原Evidenceとの同一性・raw/derived連結は別であり、保存元のSource Cohort/Provider Manifestと実解析を監査したことにはならない。

### 今回確認した証拠と限界
- **ローカル実行の実績**：Python **3.13.5**、`python3 -m unittest discover -s /mnt/data/rea_audit_verify -v` で**18 tests/18 PASS/0 fail**、`py_compile` PASS。
- **ソース同一性**：検証時の `git hash-object` は `audit.py=0238add9a25408b256b45d672d0925935de12501`、`test_audit.py=26177bb9bbefd391d354a5f397360f9bfe547165`。**GitHub branch上の各Blob SHAと一致**。この一致は、その2ファイルの実試験と保存byteの対応だけを示す。
- **CI/製品の状態**：test-only GitHub Workflowは存在するが、この段階の`fetch_commit_workflow_runs`／combined statusには該当headの結果が表示されていない。**exact-head Actions PASSは未確認**。REA実プロセス、実Provider、T1-A、T1-B、旧新Runtime、SHO実Writer、Owner Accepted MeaningはNOT RUN。16MiB上限とFile Pathの`is_symlink`チェックは軽量T0ローカル入力向けで、厳密なOSのrace-free sandbox/Access Grantを保証しない。

### 他チャットの進捗と矛盾しない合流順
1. **S0/T0は外部REAの実出力が先**：公式の固定サンプルを実静的解析（元コードは実行しない）し、raw JSON/stderr/exit＋Input/Analyzer版＋Source Cohortを別実証。今回の18件の合成PASSはT0の代替ではない。
2. **Plugin本線 S1とS2は別**：[main統合済み #3371](https://github.com/hat47x/tei-platform/pull/3371)と[main統合済み #3381](https://github.com/hat47x/tei-platform/pull/3381)でStudio/CLIの限定Inspector Service seamは進んだが、**通常製品CLI/Studioから同じselected Providerをactually invokedするS2は[Draft #3540](https://github.com/hat47x/tei-platform/pull/3540)**。Service helperの局所成功を実Wails/製品Plugin/実Provider Byte資格に昇格させない。T0はS2の完成をブロッカーにしない。
3. **Design本線は実Workspace M1入場までの候補**：[Draft #3492](https://github.com/hat47x/tei-platform/pull/3492)のQuery/Relation ID・Kind厳密Readを参照するが、Data/Function/Business Graph/CRUD 完全、Source逆Write、実Provider DB Readは別。REA Evidenceを構造化しただけで三視点・CRUDに受理しない。
4. **SHO本線はRaw byte span/Save As preflightの限定前進**：[SHO Draft #29](https://github.com/hat47x/sho-composer/pull/29)（UTF-16 Editor位置→原UTF-8 byte span）と[SHO Draft #28](https://github.com/hat47x/sho-composer/pull/28)（相対リンク先変化）を出典引用の前提候補にする。両実装の試験成功を正式Writer/Review・実TEI Accepted Sourceと足し合わせない。
5. **全体計画／索引の衝突に注意**：[Draft #3542](https://github.com/hat47x/tei-platform/pull/3542)は `plan/design/README.md`の重複を整理する別Owner。#3496の既存索引追加1行と重複させず、PR取込前に**最新mainと差分を再基準化**する。本節では索引を再編集せず、文書追加の唯一の場所としてこのPRを維持。

**検収線**：read-only Local T0 Auditorの18件PASSは、REAのEvidenceを受領したと主張できる証拠ではない。次の意味のある一歩は、**実REA T0が生成した現物の`analysis.json`/`analysis.stderr`/`exit-code.txt`を同じAttempt・独立Source SHA/実解析Versionに結び、このAuditorで検査し、独立Expectedに照合すること**。完了するまでExternal Observation qualificationを部分/unknownで保持し、T1-A/T1-B/SHO/Target成功へ昇格しない。


---

<!-- rea-t0-actual-workflow-20261109 -->
## 12. T0実解析を実行するGitHub Actionsの限定導入（2026-11-09）

**実行物**：本Draftで `.github/workflows/rea-t0-static-evidence.yml` を追加。REAの[上流公式CI手順](https://github.com/morluto/rea/blob/main/examples/github-actions/rea-analysis.yml)に合わせ、`ubuntu-24.04`＋`actions/checkout@11d5960...`＋`setup-node@49933...`／Node **24.22.1**＋`rea-agents@6.7.0`を指定し、synthetic/public JavaScriptディレクトリを作成する。**対象コードは解析されるが起動されない**（throw文入り）。ローカルの接続不能環境で無理にREAを偽実行するのではなく、外部ネットワークを利用できるActionsへ実解析を切り出す。

4つのケースはGitHub Actions matrixの**独立Job**であり、同一Attemptに混ぜない。各Jobが自分の `RUNNER_TEMP` にraw `analysis.json`／`analysis.stderr`／`exit-code.txt`とそのSHA256、Source入力2ファイルのSHA256、実Node/REA version、指定target/argv、保存結果を残し、既存の`audit.py`へ渡す。

| Case | 固定した一つの変化 | Jobの独立Expected | 誤った資格化を禁止する点 |
|---|---|---|---|
| **valid** | 有効な合成JSディレクトリ | 実exit=0、REA JSON `ok=true`、Readerで読める | それだけでSource全量・Function意味・Provider全能力は保証しない |
| **missing** | 存在しないSource pathを選択 | 実exit≠0、失敗値/診断をraw保存 | 正常解析に変換しない。Jobが期待どおり失敗を検出した場合のJob成功≠REA成功 |
| **unsupported** | ディレクトリではなく単一JSファイルを指定 | 実exit≠0、出力JSONがあれば`unsupported_target`を照合 | Providerを静かに自動変更しない |
| **partial** | 元のJSへ不正なexport行を一つ追加 | 実exit=0、`parse_failures>0`、`analysisQualification=partial` | `exit0`から`complete`へ昇格させない |

**実行経路**：branch push／main向けPR／任意の手動起動。ただし`workflow_dispatch`が実際に利用できるかはGitHub側の公開・登録状態による。Jobの正常/負例expected比較は**解析実行の終了コードを改変せず**、`exit-code.txt`に保存した値から別ステップで行う。四ケースのうちどれかが期待と異なればそのJobをFAILとし、raw reportは`always()`で可能な範囲を7日保持する。Runner損失、npm install/Build前失敗、Artifact upload不能は無条件unknown。失敗Jobのraw出力が無い場合に偽の正常Evidenceを作らない。

**信頼と再現性の制約**：本ジョブは限定Synthetic Sourceのみ、`contents: read`、`persist-credentials:false`、無Secrets、固定checkout/setup-node/アップロードAction revision、`npm --ignore-scripts`。ただし`rea-agents@6.7.0`を`--save-exact`でインストールするだけでは依存パッケージ推移版がlockで固定されないため、依存まで厳密に複製する条件は未達。T0の実Runner/REA/npm treeやSource/raw digestを同じRunに記録し、独立再現まで「全依存pin済み」としない。REAはHost上の実コードなので、機密Sourceや高権限/privileged workflowでの実行へ転用しない。

**統合時のチェック**：本PRは[main統合済み #3542](https://github.com/hat47x/tei-platform/pull/3542)の設計索引変更と衝突していたが、最新mainのREADME内容に**本REA設計1行だけを再適用**してPRのmergeableを回復させた。Branch全体のMerge Commitによる完全追随ではなく、GitHubのbaseとheadがdivergedしたままなので、レビュー直前には最新main・PR diffを再照合する。過去の重複全文や索引二重貼付を戻さない。

**資格上限**：WorkflowをGitHubへ**登録した事実**と、ActionsでREA実解析4ケースが成功した事実は別。T0は各Jobのexact head/実workflow logs/保存raw/report SHA／独立Reader判定が確認されるまで**NOT RUN / UNQUALIFIED**。T0が成立してもT1-A実TEI Adapter、T1-B Loader/Owner Meaning、T2三視点/SHO Writer/Target実比較は全て別資格。


---

<!-- rea-t0-live-four-case-evidence-20261109 -->
## 13. T0-01〜04：REA実プロセス4対照の取得・別Reader再読取（2026-11-09）

**実際のGitHub Actions証拠**：[T0 4-case Run #42555450826](https://github.com/hat47x/tei-platform/actions/runs/42555450826) = `completed/success`、tested HEAD **`3134fe04cc6dbe15c1171ffebe2d83c6c832e1bd`**、4 Matrix Jobすべて `completed/success`（Ubuntu 24.04、Node **v24.22.1**、実`rea --version`出力 **6.7.0**）。別[T0 saved-report Unit Run #42555450866](https://github.com/hat47x/tei-platform/actions/runs/42555450866)も同HEADで`completed/success`。旧[unit Run #42555291193](https://github.com/hat47x/tei-platform/actions/runs/42555291193)の実Jobログからも18 tests/18 passが確認された。Runの成功判定は**各テストの独立Expected**に対するものであり、missing/unsupportedのREAプロセス自体は非0終了した。

4個のGitHub Action artifact ZIPを別途実ダウンロードし、ファイルを再読取して、`analysis.json`/`analysis.stderr` raw SHA-256の独立再計算値と同じZIP内の`consumer-audit.json`・`raw-report-sha256.txt`が一致することを照合した。下表は実Artifactから採集した値であり、合成fixtureの期待文字列ではない。

| 対照 | GitHub Artifact ID（7日保持設定） | REA exit | JSON SHA-256 | 解析失敗／REAの明示limitations | 保存結果 |
|---|---:|---:|---|---|---|
| **T0-01 valid** | `13646699709` | **0** | `c9a3197e6966b7ba1946d8c6cc1a0d7d2cc6f2f26190f3d5fde61e7582e8e1b3` | `parse_failures=0`／**5件** | `ok=true`、構文解析成功。**情報完全性はunknown** |
| **T0-02 missing** | `13647033339` | **1** | `47548154f2996c4afbf59e624530f29b4cfbab08c8d5ef71fa4d114425654f84` | 該当なし | `ok=false`、`error.code=invalid_request`／`error.category=invalid_input` |
| **T0-03 unsupported** | `13646967220` | **1** | `d1177c957b94b0acb10d4ed8177eea1dad9bc119bc74ffa190dc082d0232c2fc` | 該当なし | `ok=false`、`error.code=unsupported_target` |
| **T0-04 partial** | `13646404103` | **0** | `6e098abef03a1374c0cbf6ee02e24d32a4dde4bee02f87284f82c77eeffa4b32` | `parse_failures=1`／**6件** | `ok=true`でも`analysisQualification=partial`、`Graph coverage is incomplete`明示 |

### Source Identity／Provider Identity／Evidence Identityの読み違いを防ぐ実値

- **Input byte**：valid/missing/unsupportedの合成`index.js` SHA-256 = `938a5a8795a132572c972312c8630522da0ab0c8d114b9ed4f7d0a2cd55d3cf9`、partialの`index.js` SHA-256 = `7d0a96b38265f42f72c05331cfd25eac98d0f5f013acdb3d552fb446e147b9c2`。各Jobの`fixture-source-sha256.txt`を保存済み。`package.json`は4Job共通SHA-256 `5b9dec1e248aba72767e7ecc9077d3490b0e038db1fab9f865ce99b736514c3e`。missing/unsupportedは**選択対象が違う**ので、生成済みディレクトリが同じでもObservation受入成功ではない。
- **REAが報告したroot artifact digest**：valid = `c167ed5e550845a1cb583e4f4150a049459d4044943baee552d692ee39a936ef`、partial = `815fe5b0a1387fe27ab836a721029ec38b57c33a551979a7a130b6f7abd7bb4f`。上記`analysis.json` raw SHAとは別Identity。これらは**REAが発行したDigest主張**であり、独立アルゴリズムからroot SHAを再実装して検証した値ではない。
- **REA Evidence ID（外部名前空間）**：valid = `ev_c70a4ea0e6b9bcf63d250213ca6d985dbaaeadb4b8062486c0619275941a7d03`、partial = `ev_cc598c4428a3736166ff55eb4cc878b4db4675cf02e664e7768af44beaac9ab5`。両方の`data.provider`は `id=rea-javascript-application, version=1`（**Evidence内でREAが自己申告したProvider descriptor**）。実CLI package version `6.7.0`とProviderの宣言version `1`を同一視しない。別プロセス実呼出しの完全なLink/Manifest照合は未観測。
- **解析制約**：validでもinert静的解析、実行時のイベント/SQL/Flow実発火は確認されず、元ソースから発見できない経路は引き続きunknown。partialは制約文`Graph coverage is incomplete; unavailable facts remain explicit and must not be read as absence.`が追加。REAの`limitations` 5/6件を数としてだけでなく原本Evidenceへの参照付きでT1-Aに渡すこと。存在しない`unknowns`配列を空集合と決めつけない。

### ここで初めて合格とできる範囲

**T0-01〜04の固定Synthetic JavaScript／REA実CLI／保存済みraw／独立T0 Consumerの限定対照はPASS**（4個の別Job・別Attempt）。T0-05（JSONL実export途中断）とT0-06（Provider/profile版変更の再資格化）は今回**NOT RUN**。この4件を一つの同時刻全Source Snapshotとして結合せず、同じ版・同じRun IDの下にある**4つの独立した実行**として記録する。

この実績はT1-AのTEI External Observation AdmissionやT1-B正式`*.tei.md`、実Source→Graph/CRUD/SHO Writer、Owner Meaning、実DB、旧Runtime Effect、実Provider selected→invoked全経路を**合格させない**。制約：npmのtransitive依存は未lock、実Nodeモジュールの完全なbinary digest／Host grant、外部Provider raw/Manifest同時性、全量coverageは未証明。現在のPR HEADはこの記録追記によりtested SHAより後の版であり、**前回のCI結果を新HEADの全体Healthへ転用しない**。

**次の最短着地点**：この実Evidenceの`data.subject`/`data.provider`/`normalized_result`/`limitations`を**Source-backed typed external observation候補**として、現在のTEI共通read-only Application Operationに載せられるかfit-gapを一Functionに限定して検査する。T0の正常/部分/異常を独立Expectedで維持し、Schemaに新しい第二Source RegistryやREA独自Semantic DSLを導入しない。製品側`order_entry.recent_orders`の実DB Readとは別の独立実験であり、合算して同一E2Eとしない。


---

<!-- rea-t1a-external-candidate-projection-20261109 -->
## 14. T1-A前段：実REA GraphをSource-backed外部観測候補へ投影（2026-11-09）

**実験実装**：本Draftの tools/rea-t0-saved-report/project_observations.py と test_project_observations.py は、保存済みREA JSON、T0 audit、同一JobのSource manifest・選択対象を読み、外部GraphのNode Observation／Edgeを候補として投影する read-only Test Consumer。**TEIの正式Source Parser／Canonical World／Application Operationへの受入は一切行わず、第二のTEI Semantic SSOTを作らない**。これはT1-Aを製品実装する前の識別・Loss／Scope受入圧力を検査するもの。

### 既存APIとのfit-gap

- **入力の同一性**：REA外部Evidence ID、raw analysis.json SHA256、REAが主張するroot artifact digest、同じRunのSource manifest、申告Provider（id/version）、選択pathを別属性とする。raw SHAとroot digest、申告Provider versionと実CLI package・binaryは互換でない。
- **写像の単位**：外部node observation IDとedge IDを別に保持。GraphのNode identity・relationの向き・多重性、source-range（行/列）またはartifact-path、推論状態 observed/inferred/unavailable とauthority、evidence coverageを保持。外部IDからTEI Canonical ID・意味Roleを発行しない。
- **欠落を保存**：Graph coverage、Semantic Graph coverage、Semantic unknowns、REA limitations件数と元raw出典を区別。unknownの配列が空でもSemantic coverageがunknownなら不完備を保持。parse_failures=0、exit0、Graph局所completeから全アプリの実動作・業務意味を推論しない。
- **拒否の境界**：REA非0終了・JSON矛盾はCandidate 0件。root SHA衝突、異Provider・匿名Evidence ID、Source digest矛盾、Node／Relation不整合、Observation/Unknown ID重複、過剰件数、Manifestが選択root外はfail-closed。相対Source pathが解決できなければ独立Bindingを作らない。
- **Source資格の上限**：saved-manifest-match-only は同一Run内の保存済みManifestとREA申告Digestの一致にすぎず、元Source bytesの別Process直接読取、Provider Source cohort原子性、現在のHost権限を証明しない。SourceClosureはnot-qualified、TEI CanonicalSubjectはnull、Accepted Meaningはfalseで固定。

### 実T0 ZIPに対する局所Read-only投影の結果

2026-11-09に前節13の4つの実REA Actions artifactを別々に展開してTest Consumerへ入力した。これは製品CLI/Studioではなく、実Evidenceに対する試験用Projectionの局所観測。

| 事例 | Node Observation | Relation | Semantic unknown | Graph coverage | Semantic coverage | 結果 |
|---|---:|---:|---:|---|---|---|
| valid | 7 | 7 | 1（dynamic-call） | complete | unknown | external-candidates-only（計14候補） |
| partial | 6 | 6 | 0 | partial | unknown | external-candidates-only（計12候補、うち1件unavailable） |
| missing | 0 | 0 | 非適用 | 非適用 | 非適用 | refused |
| unsupported | 0 | 0 | 非適用 | 非適用 | 非適用 | refused |

validの14候補はobserved 9／inferred 5。partialの12候補はobserved 7／inferred 4／unavailable 1。両Caseでartifact-rootに相当する1候補はファイル単位の独立Manifest Bindingがない。部分解析の unknowns=0 は「未取得なし」ではなく、semantic coverage=unknown を維持する。

**合成の独立Expected**：上記試験用コードはProvider誤認、Source digest衝突、root不一致、Source path、関係端点欠落、外部Observation/Unknown ID重複、unavailable、平行Relation保持などの16ケースをPython作業環境で16/16 PASS。GitHub Blob SHAもローカル実行byteと一致（projection 23666bd4c5b2c6508fc5150debf9de4fd1b485f9／test c3d7260052f9a294df08f4588f7d8027bad60337）。前段のT0 audit 18件と合計34の合成試験。新GitHub Actionsは変更した正確なHEADで別判定し、以前の別HEAD成功を移さない。

### 他チャットの成熟設計に接続するが、所有権を増やさない

- mainへ統合済みInspector S2-reference [PR #3540](https://github.com/hat47x/tei-platform/pull/3540)は通常CLIとStudioから静的参照Providerを呼ぶ限定実績。外部REA実ProviderをHostが実選択／実InvocationしたS2-product、Artifact/BOM/Source current・Grantは未資格。既存共通Application API（ADR-0048）、Plugin Owner #1251/#951を再利用し、もう一つのOperation DispatcherやRunnerを作らない。
- 既存InspectSubject（internal/operations/inspect.go）は正式Canonical WorldのSubjectからSource Indexを投影する。External JSONを直接送ってそれを正式なinspect成功と呼ばない。T1-A製品側は既存のObservation/Source受入経路とのfit-gapをまず示し、欠ける型だけ既存Ownerへ返す。
- TEI三視点Model Map（#1124／#3492）、Function実DB Read（#2586）、SHO Editor Source span・Save As（SHO #28/#29）、独立Expected（#2514）は別適格。旧Runtime観測とOwner Accepted Desiredも混同しない。
- 最短の次段は**一つの実Evidence Observationと一つの関係Edge**について、REA外部ID／Source path+digest+span／Coverageとunknownを失わず、同一AttemptのTEI read-only Consumerが実際に受け取れるかを示すこと。CLI・Studio・MCPをすべて先に開通させる必要はない。

**現時点の限界**：保存済み実REA出力の試験用候補投影まで。T1-AのTEI正式取込み、T1-Bの正式Markdown DSL入場、三視点CRUD、SHO Writer/Review、実DB、旧新実行同値、Owner受理はNOT RUN。


---

<!-- rea-t1a-byte-snapshot-boundary-20261109 -->
## 15. T1-A Source実byteの二時点照合（2026-11-09）

他チャットのWorkspace Identity #3181、Source-only Drift（main #3479）、Inspector S2-reference（main #3540）、異Workspace分離（Draft #3735）と、Actual Input/A→B→Aの反例を踏まえた成熟差分。**Manifestに記載されたSHA256を読むだけで、元のSource bytesを独立に読んだとは言わない。**

### 固定公開Sourceのみの新しい独立観測

同Draftの .github/workflows/rea-t0-static-evidence.yml は、REAが静的解析する公開合成JavaScriptの2ファイル（index.js、package.json）を、REAプロセス呼出し直前の source-pre と終了後の source-post として各matrix Jobの RUNNER_TEMP レポートに複製する。両ファイルのraw byte・Manifest SHA256・REAが申告するObservation artifact SHAを区別する。**顧客Source・機密データにはこの実験手順をそのまま適用しない。**

| Source binding | この試験で示すこと | 保留すること |
|---|---|---|
| saved-manifest-match-only | 旧ArtifactのREA申告Digestと保存Manifestの一致 | Sourceの実byte独立Read |
| report-pre-post-byte-match | pre/post保存byteのSHA256・Manifest・Observation Digestの一致 | プロセス中の原子的Snapshot、A→B→A、現在Grant |
| not-independently-bound | Graph Rootなど個別Sourceと確実に結び付かないもの | Sourceパスの推定補完 |

不一致のpre/post、片側欠落、個別ファイル欠落、symlink、3つ目の宣言外ファイルはfail-closed。固定2ファイル・1MiB/ファイルに限定する検証であり、汎用Source Resolver/Repository全体Closureではない。

**A→B→A注意**：pre(A)→解析中(B)→post(A)のような変更は二時点照合だけでは検出できない。Report内の保存copyが一致した証拠を実Providerが読んだ全byteの証拠へ昇格しない。immutable staging／同一Attempt実Input Witnessは既存#1163/#3181/#1016へ戻す。

### 実装・試験・範囲

- project_observations.py に pre/post Source bytes再読取・SHA256再計算を追加し、SourceBinding表示を二時点照合として区別。TEI Canonical Subject=null、Accepted Meaning=false、SourceClosure=not-qualifiedは変えない。
- test_project_observations.py は16→**23独立テスト**。追加7件：正常なpre/post一致、pre改変、post改変、snapshot片側欠落、個別Source欠落、symlink、予期しない3つ目のSource。作業環境で23/23 PASS、GitHub Blobがローカル実行byteと一致（projection b8f744a85084bc9c1c846151d5a98f002a274ee7、test acfb3f48386876e7fa7ba6c7527c8ee9862bce8e）。
- 前回の実REA artifactから取得したSource SHAに一致する**合成Sourceを再構成し、後付けしたpre/post copy**で試験：valid 14候補中13件、partial 12候補中11件が report-pre-post-byte-match。両方1件のRoot相当は未Binding。これは**前回のRunで実際にpre/postを保存した証拠ではない**。
- 合成ローカル試験はT0 audit 18＋Projection 23＝**41件**。最新Workflowは各REA Jobで実Source copyを前後保存するよう更新済み。ただし当該exact-headのGitHub Actions完了・再ダウンロードは別資格。
- 現mainのInspectSubjectは既受理Canonical WorldのSubjectから投影するので、REA候補を正式Subjectに見せかけない。次の実T1-Aは1外部Observation＋1 Relationのtyped read-only Consumer実受領。Source/Workspace Identity・scope/unknown/現在Grantは別判定、第二Host/Runner/Source Registry/Parserは作らない。

**未成立**：REA実ProviderをTEIのHostから正式選択・実呼出し、原子的Source snapshot、T1-A正式TEI製品取込み、T1-B DSL/Owner Meaning、三視点CRUD、SHO Writer、実DB/旧新比較はNOT RUN。


---

<!-- rea-t1a-cli-vs-unit-path-20261109 -->
## 16. 同じProducer結果でもCLI実引数が異なると失敗する：CIで見つかった受入負例

新Source pre/post保存を追加した[実REA T0 Actions #42578272437](https://github.com/hat47x/tei-platform/actions/runs/42578272437)の**validとpartialのJobはFAIL**、missingとunsupportedは独立Expectedに対してPASS。実ログの失敗ステップは「Project external observations without accepting TEI meaning」であり、tracebackに `TypeError: unsupported operand type(s) for /: 'str' and 'str'` が記録された。REAの解析自体、Raw Evidence保存、Source pre/postコピーは成功したが、**別Consumerの入力経路が失敗した**。したがってRun全体はsuccess扱いしない。

原因：従来のUnit Testは `project(Path)` で実行するが、実CLIは `project(str)` で呼ぶ。内部の `_file` が `root / name` というPath操作を行うため、StringをPathへ正規化していなかった。今回 `project()` 入口で `Path(report_dir)` に明示正規化し、`test_cli_string_report_path_is_supported` を独立の24番目のNegative/Positive Consumer境界対照として追加。**同じ入力ファイル・同じ関数でも、呼出し経路の入力型が違えば結果が異なる**ことを検出するための追加試験であり、REA構造自体の欠陥ではない。

**確認できた実績**：作業環境PythonでCandidate Projectionの**24/24 unit PASS**、実際のGitHub保存byteと完全一致。修正後Projector SHA `1c4d00463d32a994e564dfa5e3b45409da32a845`、テストSHA `a15da75bb862928df8a31b59653a8f3ba39a328a`。従来audit 18件と合計42件の局所試験対象となる。ただし**修正後exact-headのGitHub Actions 4ケースのPASS／保存Sourceの独立Readbackは別確認**であり、失敗Runの成功として書き換えない。

**計画の更新**：T1-Aの入場判定では、Unit Fixtureと実CLI entrypointが同じResult/Failureを返すことを実行証拠で照合。Source Snapshotの二時点確認、Manifest一致、実Source CohortやABA不可検出、TEI Canonical Subject未入場、Current Grant未資格は引き続き独立。#1158共通Operation/CLI Adapter、#1163実Consumer receipt、#2514独立Expectedを再利用し、REA専用の二重APIやRunnerは追加しない。


---

<!-- rea-t1a-snapshot-real-ci-20261109 -->
## 17. Sourceの解析前後バイト列が実Artifactへ保存された限定証拠（2026-11-09）

### 実実行結果とテスト版

- [REA T0＋候補投影 Actions #42578642277](https://github.com/hat47x/tei-platform/actions/runs/42578642277)：tested commit d6f3f1cc3f29411314fd590c81abf9f613597ac9、valid/partial/missing/unsupported **4/4 matrix Job completed/success**。異常系Jobの成功はREAプロセスexit非0を独立Expectedと照合したもの。
- [独立audit CI #42578642247](https://github.com/hat47x/tei-platform/actions/runs/42578642247)：同tested commitでcompleted/success。実Jobログは Ran 42 tests ... OK（audit 18＋Projection 24）。今回はCLIの文字列パスを受け取る別Consumer経路の回帰が含まれる。
- [途中の失敗 #42578272437](https://github.com/hat47x/tei-platform/actions/runs/42578272437)：valid/partialでPythonのstrとPathの不一致によりCandidate ProjectorがFAIL、missing/unsupportedはPASS。REA解析実行とSource保存に成功しても、後段Consumerは失敗した事実を保持する。入口でPathへ正規化してテストを追加し、別Runで再実証した。

### 実Artifact ZIPの独立再読取

GitHubの実Artifact ZIPを再ダウンロードし、ZIPにあるanalysis.json／consumer-audit.json／fixture-source-sha256.txt／source-pre／source-post／external-candidates.jsonを別Readerで再確認。raw analysis.jsonの再計算SHA256が保存監査値と一致し、固定の公開合成2ファイルのpre/post byteが一致し、双方のSHA256がSource Manifestと一致することを確認した。

| Case | Artifact ID | 保存pre/postの実byte一致 | Manifest＋REA Evidenceと照合した候補 | 未Binding |
|---|---:|---|---:|---:|
| valid | 13656381225 | index.js 88byte／package.json 75byte、双方pre=post | **13/14** | 1 |
| partial | 13656645331 | index.js 112byte／package.json 75byte、双方pre=post | **11/12** | 1 |

両ArtifactのProjectionは external-candidates-only、sourceManifestStatus=report-pre-post-byte-match、TEICanonicalSubject=null、acceptedMeaning=false、sourceClosure=not-qualified。Root相当の1候補は物理Sourceを推定せず未Bindingのまま維持。これは前節15の「旧ZIPに後から追加したローカルSourceコピー」と異なり、**実REAプロセスが動作した同一Jobが実際に収集したpre/postのraw bytes**である。

**資格上限**：pre(A)とpost(A)の二時点バイト一致は、解析プロセス中のA→B→Aを検出できず、実Providerが読んだ全byteの原子的証拠でもない。Hostの現在Grantや正式TEI意味は付与しない。[Inspector S2-reference main #3540](https://github.com/hat47x/tei-platform/pull/3540)の実装はREA Providerの実選択資格ではなく、[Workspace差分Draft #3735](https://github.com/hat47x/tei-platform/pull/3735)で区別しているWorld FP／Workspace Identity／Source raw revisionの要件を再利用する。

**次の唯一の不足辺**：REA Evidenceの外部Observation 1件とEdge 1件を、現在のTEI共通read-only Application Operationへ情報損失なく接続するfit-gap。InspectSubjectは既受理Canonical Worldを読むので、外部候補をCanonical Subjectと偽装しない。T1-A正式取込み、T1-B DSL／Owner Meaning、三視点CRUD、SHO Writer、実DB／旧新Runtimeの適合は未実施。今回の文書追記後のHEADの全体CIを、上記tested SHAの成功から推論しない。


---

<!-- rea-t0-real-jsonl-product-seam-20261109 -->
## 18. 実JSONLの配送完全性を実測し、TEI既存Observation／Evidenceへ渡す境界を明確化（2026-11-09）

### 18.1 T0-05：REA実JSONLと途中切断は別の資格

`.github/workflows/rea-t0-static-evidence.yml`の**valid独立Jobだけ**へ、REA 6.7.0の実`inspect-analysis-view`＋`--format jsonl`を追加した。保存された実`analysis.json`を`source.kind=evidence-file`に指定し、`view.kind=javascript-facts`、`offset=0`、`limit=32`、`property_keys=[]`を明示して、行集合を抽出する。REA実プロセスの終了コードとJSONL/std err raw・SHAを保存し、既存`audit.jsonl_audit`を再利用して**object collection最終metadataの存在・emitted count**を確認する。

次に、作成済みJSONLの**末尾control record一行だけを除いたコピー**を作り、Readerが`missing-metadata`として拒否することを同一Job内で検査する。元の保存JSONLや`analysis.json`を加工しない。これは**意図的な停止・切断を模したConsumer入力の反例**であり、実ネットワーク断／MCP Tool配送損失やREA streaming中の強制killを実施したことにはならない。要求したcollectionが全件`exhausted`とは保証せず、`records.emitted`と`records.source`から元アプリ全Sourceの完全性を推論しない。

**受入の期待**：
- 正常投影：REA viewプロセス実exit=0、JSONL object集合の最後が`$rea=jsonl/v1, type=metadata`、行数に整合。選択条件（offset/limit/keys）とSource Evidence/Provider版が残る。**Source全量・semantic completenessはunknown**。
- 最終行欠落：同じJSONLの残存行に妥当なJSONだけがあっても、終端metadataの欠落を検知して**Consumer completeを拒否**。既に出たdata rowは部分的な観測として扱えるが、`acceptedMeaning`を与えない。
- 実REAがエラーJSONL controlを返した場合：通常の行として数えず、元stderr/exitを残して**失敗として検収する**。局所成功のために別Providerへ自動fallbackしない。
- bare top-level arrayはmetadataがない仕様なので、object collectionと同じ終端規則を適用しない（既存18/42合成反例を維持）。

**実行の現況**：この節作成時点は**GitHub workflow変更・レビュー票の登録**。変更後HEADでのREA JSONL実行、生成されたArtifact readback、test jobのpassは**未確認**。先行T0 4/4や合成42/42 PASSをT0-05実検収へ転用しない。GitHub Actionsの`always()` artifactは資格がなくても現物診断を保全するためのもので、アップロード自体を完備証明としない。

### 18.2 T1-A Product入場：既存APIと未接続のObservation/Evidence Routerを直結しない

現在の`internal/operations/InspectSubject`は**Canonical Worldに存在するSubjectを先に厳密検索**し、対象`Kind`のサポート外や未存在は拒否する。Source Index参照だけではCanonical Meaningを生成しない。従って、試験用`rea.external-observation.candidate-test/v0`を`InspectSubject`にそのまま渡すこと、仮の`semantic-data` IDを発行して通過させることは、どちらも**T1-Aの実入場ではない**。

[Binding実消費調査 #3757](https://github.com/hat47x/tei-platform/issues/3757)では、Observation/EvidenceのBinding宣言が存在しても、実`observation.Router`・`evidence.Router`構築は試験用だけで、**非テスト製品Call siteから`Observe`／`Collect`に届かない**。逆に[main統合済み Inspector #3540](https://github.com/hat47x/tei-platform/pull/3540)は**静的Reference Providerの通常CLI/Studio S2-reference**に限定され、外部REA ProviderのS2-productではない。両者を足して「既存Observation ProviderでREA取込済み」と判断しない。

T1-Aの最小Product受入は、**共通Application API（ADR-0048・#1158）の既存型／Operation境界に沿う、外部1 Observation＋1 Relationのread-only実受領**である。初回のfit-gapでは、既存の`operations`、Binding/Providerの責務に割り振れるものを優先し、必要な型不足だけを#3419/#1251へ返す。Observation/Evidence Routerへ新しい共通Resolver、Reader、Host、Registryを追加しない。新たな外部Observation公開Schemaが必要かは**現在のOwnerが実Consumer試験から判断**する。ここでSchemaを承認した扱いにしない。

| 観点 | 外部候補とTEI／Consumerに必要な対応 | 成功を名乗ってはいけない場合 |
|---|---|---|
| **External identity** | `REA Evidence ID + raw report digest + external occurrence/edge ID + source scope + attempt`の関係を保持。TEI Canonical IDと別 | 同じ`observation_id`を別Attempt／別Workspaceで誤結合 |
| **Source** | 選択Target、Source pre/post bytesとSHA、実Loc/`line,column`、Provider自己申告、未binding候補を別列 | pre(A)=post(A)を原子的Source/Provider入力完全保証とみなす |
| **Position** | file-level digestとline/columnの**位置単位／Byte範囲の資格**を分離。SHO E0-MAPはMarkdown Editor側の別Consumer | PythonのstrやREA JS位置をそのままSHO UTF-8 raw byte-offsetとしてPatch |
| **Relation** | Edge ID、向き、端点、同Sourceでの複数Edge、Function/Business/CRUDの`candidate`区分 | 同Labelの別IDを併合、平行Edge一本化、SQL参照を実Commitとして採用 |
| **Coverage / loss** | Graph coverage／semantic coverage／unknowns／limitations／partial／root unboundを保持 | 解析対象外・0件・未取得・非開示を全て`empty`へ潰す |
| **Invoker / delivery** | Host selectedと実Provider invokedの別観測、同じRequest/Scope/RevisionのCLI・Studio結果、実Receiver byteと現在Disclosure | `Router`登録だけ／JSON出力だけ／CLI `ok`だけでProduct実Call／受領済みと判断 |

[組織意味対応の別Owner #3741](https://github.com/hat47x/tei-platform/pull/3741)のORG-EXT-01〜06（同Label別ID、平行Relation、unbound、partial、同Raw別Workspace、A→B→A）は**Mappingの入力側の負例**として再利用する。テストの正本は既存#1180/#2514とし、REA用に第二Expected Registryは作らない。マッピングを明示受理するまで、Organization/Function/Business/CRUDはCandidateからAcceptedへ昇格させない。

**次のProduct 1回の独立Expected**：同一固定REA Artifactに対して、(a) CLIが読み取ったExternal Observation 1件とRelation 1件のID/Source/unknown、(b) Product側の実Provider選択・実呼出し、(c) 実Receiverへのbyte到達、(d) TEI Canonicalがnullのままであることをそれぞれ実観測。Studio/WailsとSHOへの投影は後続の別資格。**T1-A試験用Python Projectorの成功を、このProduct Exitとして数えない**。

本節はOwnerに実装開始を引き渡すための**設計差分**であり、T1-A Product Operation、外部REA実Provider Link/Grant、Source atomic closure、正式`*.tei.md`、三視点Product Graph/CRUD、SHO Writer、Legacy/Target比較、Owner Meaning Acceptanceを実行・達成したものではない。


---

<!-- rea-t0-evidence-health-closure-20261109 -->
## 19. 本Draft自身のRepository Health失敗を独立に解消（2026-11-09）

[旧PR exact-head Health #42579045035](https://github.com/hat47x/tei-platform/actions/runs/42579045035) は **completed/failure**。実Jobログで`go vet ./...`、`gofmt`、`tei validate`がPASSしていた一方、失敗は**このPRが追加したファイル**に限定された。

- `tools/repository-plan-check.TestToolsReadmeMentionsEveryEntry`：`tools/rea-t0-saved-report/`が`tools/README.md`に未登録（`go test ./...` FAIL）。
- `tools/ci-workflows-test.py:StaticWorkflowContract.test_every_automatic_workflow_declares_concurrency`：`rea-t0-saved-report-audit.yml`と`rea-t0-static-evidence.yml`の二本でトップレベル`concurrency`が欠落（CI E1／候補HEAD照合 FAIL）。

**対処**：`tools/README.md`へread-only T0 Auditor/Candidate Projectorの索引1行を追加。2 Workflowとも現行標準`concurrency.group = workflow名＋event＋pull_request.number/ref`、`cancel-in-progress: github.event_name == 'pull_request'`を追加し、**古いPR実行のみを取り消し、branch/mainの実行は取り消さない**。単なるCIのgreen化を理由にT0-05結果検証を弱めず、JSONL実exportと途切れたConsumer入力の独立Expectedはそのまま残す。

この対処は既存のWorkflow規約／Tool索引に合わせるだけで、REA/TEIの製品仕様・Provider/Host認可・意味採用を変更しない。**後続HEADのRepository Health完了・T0-05実JSONL結果は独立に確認する**。旧#42579045035のfailureを無かったことにせず、新HEADへ旧T0 4/4／Audit42/42の結果を無条件に転載しない。

### 19.1 追加すべきテストではなく既存Ownerへ渡す次の一辺

External Observation/Evidenceの製品側経路は、[現行Binding利用実態 #3757](https://github.com/hat47x/tei-platform/issues/3757)の`Router`未接続、[Inspector S2-reference #3540](https://github.com/hat47x/tei-platform/pull/3540)の静的Provider実到達、[Graph/Relation #3492](https://github.com/hat47x/tei-platform/pull/3492)の限定main統合を**別資格**とする。

REA Candidate Projectorの外部IDを`InspectSubject`のCanonical IDとして偽装しない。現状の`InspectSubject`がCanonical Worldを先に検索するという仕様を変えず、External Observation 1件＋Relation 1件の実Product read-only受領に必要な最小型・Provider Binding・受領位置の不足だけを#1158/#3419/#1251へ引き渡す。F0 raw Source／F3 actual Invoked／F5 current Receiverと、Owner Accepted Meaning、Runtime Effectの独立資格を維持する。**ここでは新たなSchema/独立Registry/第二Runner/新Host/新Readerは設けない**。


---

<!-- rea-t1a-router-contract-dissimilarity-20261109 -->
## 20. 本番T1-AのFit/Gapを確定：既存Routerの再利用境界と静的Evidenceの意味差（2026-11-09）

### 新たにmainへ統合された二つの局所実証

- [main #3777](https://github.com/hat47x/tei-platform/pull/3777)：既存`observation.Binding → Host registry → observation.Router.Observe`を、参照Toolから**実Call**できた。registeredでは`resolved=true/selected=true/invoked=true`、未登録では`resolved=true/selected=false/invoked=false`の別判定。固定のReference Fragmentであり外部Runtime/DBを読んでいない。**製品通常CLIへの入場ではない**。
- [main #3793](https://github.com/hat47x/tei-platform/pull/3793)：既存`evidence.Binding → Host registry → evidence.Router.Collect`を、`durable.ModeEvidence`のActionRecordについて参照Toolから実Call。未登録・非証拠ゲートを別状態で拒否し、Provider自身の`pass`は真偽の証明に格上げしない。**正式製品呼出し・外部Evidenceの真正性ではない**。
- [main #3540](https://github.com/hat47x/tei-platform/pull/3540)の通常CLI/Studio Inspector S2-referenceおよび[main #3492](https://github.com/hat47x/tei-platform/pull/3492)の限定Source/Relation試験とも独立。これらを足してREA T1-A Product受入にはしない。

**最も重要な新しい非互換性**：`internal/deployment/observation/provider.go`で既存`observation.Fragment`が表現するのは、`Environment`に結び付いた`ApplicationRuntime[]`または`Databases[]`の**実状態Snapshot**。`Merge`は観測Kindの正本Facetを置換する。REAは静的JavaScript解析のSource-backed graph/unknownsであり、これを`ObservationApplicationRuntime`や`ObservationDatabaseState`に偽装して通すと、**未観測のRuntime/DB状態が「確定Snapshot」に置換される危険**があるため**禁止**する。

同様に`internal/deployment/evidence/provider.go`の`Evidence.Router.Collect`は`durable.ActionRecord`の`ModeEvidence`、`ActionID`、`EvidenceKind`を前提とし、部署／システム配備上のEvidence Gateで使われる。REAの静的解析結果を`ModeEvidence`の`pass`や`SourceRef=provider://...`で包むだけでは、逆解析の現在性・網羅性・業務意味は得られない。**データ契約は共用せず、「宣言→解決→選択→実InvokeのWitness」「未登録／不適合／Invoke後失敗の診断」だけを実装パターンとして再利用する**。

さらに`internal/sourceacceptance.VerifyCurrent`は、Hostの**現在の受理済みSource**を信頼されたReaderから取得し、Workspaceの管理Source集合との一致を検査する。任意のREA Evidence JSONやProvider自身の署名風フィールドから正式なAccepted Sourceを製造しない。暫定・未特定の受理Kindはformal meaning acceptanceではない。

### 既存の型とT1-A 1 Observation＋1 Relationの間に残る不足

| 既存の受け口（main Sourceで確認） | 受入対象の意味 | REAの外部候補を直結できるか | T1-Aで採る行動 |
|---|---|---|---|
| `operations.InspectSubject` | 先に存在検査された**Canonical World**のexact Subject。Source Indexは意味生成をしない | **不可**：外部Occurrenceを`semantic-data`/ `semantic-relation`で捏造すると誤受理 | 現行の拒否を保ち、外部候補をCanonical Subjectへ昇格させない |
| `deployment/observation.Router` | 配備EnvironmentのApplication Runtime/Database State Snapshot | **不可**：静的Source Graphを実状態と混同する | 外部Source候補として受理できる経路のfit-gapへ戻す |
| `deployment/evidence.Router` | durable Evidence Gate ActionRecordに対するCollector | **不可**：逆解析Source Evidenceは配備ActionのOutcomeではない | resolved/selected/invoked/receivedのWitness形だけ借りる |
| `sourceacceptance.VerifyCurrent` | 現在のHost Accepted Sourceを独立Witnessで照合 | **不可**：未受理のREA rawからOwnerのDecisionを発行できない | REAは`external-candidates-only`、`acceptedMeaning=false`のまま |
| `rea.external-observation.candidate-test/v0`（PR #3496） | raw Digest／Source pre-post／Node/Edge ID／Unknown／Coverageを**試験専用**投影 | **既に試験用としてのみ成立** | Candidate投影の契約を保持し、外部Product API設計判断は#1158/#3419/#1251へ返す |

### 最小Consumer受入票（T1-A Product未実行）

既存Ownerが外部観測型の不足を決めるまで、**新しい汎用Operation／External Registryを先行実装しない**。実際の固定REA valid Artifactから`1 graph-node-observation`と`1 graph-relation-observation`を抽出した同一Attempt/同一Scopeを、Productに渡せる既存の**read-only Application API契約の受領側**が存在するか確かめる。なければ`UNSUPPORTED_EXTERNAL_OBSERVATION`を**要求する診断案**として記録するだけで、現時点の実装済み`failure.code`とは称さない。

- **R1 External ID**：元REA Evidence ID・raw report SHA／Source path＋pre/post byte SHA・Node occurrence IDとEdge occurrence ID、Edge端点と向きを変更せず受領。複数EdgeはEdge IDで区別。Workspace/Attempt違いの同IDを混ぜない。
- **R2 Uncertainty**：semantic coverage=`unknown`、root相当unbound／Source範囲不明、validにも残るlimitationsを消さない。Locationの`line,column`をRaw UTF-8 byte offsetへ無検証に変換しない。受信側がLosslessで投影できなければ部分・不適格を返す。
- **R3 Call path**：宣言のresolved、Host selected、実Provider invoked、Product Operation Result、実Receiver readbackを別観測。呼出し成功でもAuthority・Execution Grant・旧Runtime Effectは与えない。
- **R4 Negative**：Provider未登録・別Provider ID・外部ID重複・無効Endpoint・Unknown欠落・Source A→B→Aを独立Expectedへ。A→B→Aはpre/post照合だけでは検出不可、現実験では`unknown`のまま。
- **R5 Refusal**：`InspectSubject`へ候補をCanonicalとして流す／Runtime ObservationのFacetを置換する／Evidence Gate `pass`で意味受理する、三つの短絡は**仕様上不適格**。拒否経路の既存テストをOwnerが確認し、未実施はPASSにしない。

**実装順と責務**：A. T0-05実JSONLの現在HEAD CI・Artifact確認（本REA #3496）→ B. Product受入型／入口のfit-gap（#1158/#3419・#1251）→ C. 同一Attemptの実Consumer 1 Observation＋1 Relation（#3394/#1163）→ D. Sourceと意味の明示受理（#2621/#1124、Owner）→ E. SHO Source occurrence＋正式Writer/別Process Read（SHO #6）。この順はT0-05を全TEI P0の阻止条件にはせず、独立した並走作業を許す。

**現時点の証拠**：#3777/#3793/#3540/#3492のmain統合と、それぞれの**限定的な参照実証**は確認済み。本節はSource/APIの静的照合に基づくFit/Gapであり、REA Product Providerや実TEI Operation、実Source受理、旧新DB、SHO Writer、Owner Meaningの**新しいE2E実行はNOT RUN**。

---

<!-- rea-candidate-model-map-shared-harness-boundary-20261109 -->
## 21. 他のOwnerから成熟した引渡し判断：Model Mapと共有Harnessの双方にSource-backed資格を残す（2026-11-09）

[Model Map Draft #3866](https://github.com/hat47x/tei-platform/pull/3866)では、同じWorldから作る型付き`From(kind,id) → To(kind,id)`、Relation IDの保持、二重Edge／重複IDの曖昧さ、Source-only Revision変更とWorld Fingerprint不変、A→B→Aを別試験にする方式を検討中。**Draftの合成テストはmain統合・Product Graph完成ではない**。その`ModelMap`が解決するのはCanonical Worldに実在する宣言であり、REAが外部解析したNode/Edgeを「resolved」に載せ替えて受理済み意味にすることはできない。

[main #3843](https://github.com/hat47x/tei-platform/pull/3843)のCLI/Studio Hook `uses`→制限付きCommand Caller実呼出しは、**宣言と実Invokedを別に記録する先行パターン**として利用する。だがREA Providerのactual Invocation、任意の外部Tool権限やSource鮮度をこのHookの実績から認めない。[main #3863](https://github.com/hat47x/tei-platform/pull/3863)のStudio選択→限定Design Context→DSH等の外部Harnessへの受渡しは**計画上の境界**。生のREA SourceやEvidence、組織図、モデルの未受理情報を、共有Harness Sessionが存在することだけで外部送信してはならない。外部送信／Human Review／Provider Grant／現在Disclosure／実Deliveryは別資格。Harnessのセッションは製品UI間で共有してよいが、TEI・SHO・SUI・CSWが各々のSource/Meaning正本とOwnerを保持する。

### T1-A → 三視点 Model Map → SHO/Harnessへの最短AC

| AC | 入力側（現段階の実証） | 将来の実受領で保持すべきもの | 明示する停止条件 |
|---|---|---|---|
| **REA-O1** | Source-backed `graph-node-observation` 1件、外部Evidence ID/Attempt/Scope | Origin Workspace、raw digest、pre/post Source bytes、location（その位置単位）、Root unbound/unknown | CandidateをTEI `CanonicalSubject`やCurrent Accepted Sourceとして通さない |
| **REA-R1** | Nodeに結ぶ`graph-relation-observation` 1件 | Edge固有ID、端点の向き、同Label別ID・並行Edge、プロジェクタの部分Projection | Relation ID消失／ambiguous解消不能ならread-onlyを継続 |
| **REA-M1** | C0外部観測→C1構造候補→C2 Owner Accepted Meaning→C3実Consumerを区別 | `ModelMap`の`resolved`・`sourceStatus`・`ModelRevision`を別列、同Raw別Workspace、Source-only/A→B→A | `WorldFP`不変でも旧Review・Current Grantを復活させない |
| **REA-D1** | SHOのSource occurrence・Editor byte span（Draft #28/#29）、外部Harness Session（main #3863は計画のみ） | Document URI/Revisionと選択Sourceの一致、明示Disclosure、受信byte量／途中切断・再開時Scope | 生Evidenceの無許可AI送信や、UI表示成功から正式Writer/Owner受理を推論しない |

**責務分離**：外部からの`1 Observation + 1 Relation`のProduct read-only受領に必要な型のFit/Gapは既存#1158/#3419・Plugin #1251、正式な三視点／Model Mapは#1124/#2497およびDraft #3866、SHO編集は別Repo #6。ERAではなく**REA**の本実験（#3394/#3496）が所有するのは生の外部EvidenceとCandidateへの損失なしの受渡し、実AttemptでのSource/Provider/Delivery Witnessまで。新しいCore Host／意味Registry／解析Runner／Framework Routerは作らない。

**検証の現況**：今回の追補は既存実Source／実PRの設計・コードの静的照合。上表のREA-O1/R1/M1/D1を製品TEI/SHO/Harnessで実行したものではない。JSONL T0-05はワークフロー登録済み・対象HEADの完了実Artifact未確認。既存Go/CIの別HEAD成功を現在Branch全体へ昇格しない。


---

<!-- rea-t0-jsonl-actual-negative-20261110 -->
## 22. T0-05実JSONL失敗を負例として保存し、REA 6.7.0の現物入力境界へ修正（2026-11-10）

**観測済みの失敗（未修正時）**：[T0 Run #42650177051](https://github.com/hat47x/tei-platform/actions/runs/42650177051)、tested HEAD `d4a6e363448a13c8c2e2fb37f0548b34dd838de3`。missing/unsupported/partialの3 JobはPASS。valid Jobは元のREA静的解析とT1-A Candidate-only投影まで成功したが、`inspect-analysis-view --format jsonl`段で**FAIL**。アップロードされた[Artifact `rea-t0-valid` #13689649228](https://github.com/hat47x/tei-platform/actions/runs/42650177051/artifacts/13689649228)を実読取し、`view-exit-code.txt=1`、`view.jsonl`は成功行ではなく`invalid_request/invalid_input`。指摘は`source.kind`の`evidence-file`がそのインストール実CLIで拒否され、**許容値が`retained-evidence`／`inline`である**こと。これをcomplete-view末尾metadata欠落と混同してはいけない。元解析が成功したのでT0-01実績を取り消す必要はないが、**T0-05は未合格**。

**上流の文書と実Binaryの食い違い**：上流[CLIガイド](https://github.com/morluto/rea/blob/main/docs/getting-started/cli.md)は`source.kind=evidence-file`例を掲載。しかし固定`rea-agents@6.7.0`の実Runは上記の実診断を返した。文書上の例をactual Invocationに優先させず、`REA 6.7.0の実CLiが報告した許可入力`に合わせる。上流仕様が将来変化した場合は再びCapability/入力形式を再判定する。

**このDraftで実施した最小修正**：`.github/workflows/rea-t0-static-evidence.yml`のJSONL検査だけで、保存済み`analysis.json`（最大16 MiBの公開合成Fixture）をread-onlyで`json.loads`し、`source.kind=inline`・`source.evidence=<元の完全Envelope>`のリクエストを作成する。保存原本のraw bytesはそのまま保持し、作成したリクエストの`source.evidence`と保存元のdecoded JSONが同値であること、リクエストSHA-256を別確認する。加工済みRequestのbyte digestは**元raw Evidence Digestと区別**する。外部送信・別Evidence Registry・新Parser/Runner/Tool依存は導入しない。

**合否判定**：実REAのJSONL再出力がexit0かつ`jsonl/v1` object collection metadata・emitted行数一致なら、元出力から最終metadataレコードを削った**派生コピー**が`missing-metadata`として拒否されることを独立確認。元EvidenceのCoverage、Source全量、REA Profile/依存closureは引き続きunknown/partial。**上記修正の現HEAD実CIは、GitHub Actionsによる実行とArtifact再読取まで未検収**。GitHubワークフロー定義の更新成功をT0-05完了としない。

**並行領域の引渡し**：[main #3540](https://github.com/hat47x/tei-platform/pull/3540)のInspector実CLI/Studio到達と本試験用REA CLI Invocationは別。現行既存`internal/deployment/observation`のRuntime/DB Snapshot型へ静的Source解析候補を流用しない（§20）。T1-Aの最小Product受入はSource付き1 Observation+有向1 Relationの**既存Application APIでのread-only実受領**であり、受入型が無い場合はFit/Gap/未対応診断をOwnerへ返し、新しい並列Operation/意味正本を先行実装しない。製品側の`order_entry.recent_orders`実Rowset最優先を変更しない。

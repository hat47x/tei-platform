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

**資格上限**：本Draftで今回実行したのは**合成Python test-only監査の11件**のみ。実REA T0の解析とGitHub Actions、REA→TEI実Plugin/CLI/Studio、SHO正式Editor保存・Writer、T1-A/B/T2、旧新実比較、Owner受理は**NOT RUN**。

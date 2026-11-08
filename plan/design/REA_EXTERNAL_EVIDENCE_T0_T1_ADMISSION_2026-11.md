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

`World/SourceIndex Fingerprint`だけを全外部Sourceの一時点Snapshotとみなさない。[Draft #3479](https://github.com/hat47x/tei-platform/pull/3479)のProvider raw Source-only/Manifest-only driftではConsumer側の値が同じまま外部入力のbyteが変わり得る。古いConsumer Resultを用いた後日のProvider navigationを、古いAttemptの同一原子Source読込みと結び付けない（#3394 RE-25〜27）。同byteのA→B→Aでも前のReview/Grantの自動復活を禁止する。

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

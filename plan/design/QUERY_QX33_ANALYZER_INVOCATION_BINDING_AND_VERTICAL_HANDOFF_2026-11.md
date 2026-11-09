# QX-33：SQL Analyzerの実呼出し・入力byte帰属とQuery→Context縦断の受渡し（2026-11-03）

- **位置付け**：[Query構造Explain #2387](https://github.com/hat47x/tei-platform/issues/2387)／[Draft PR #2388](https://github.com/hat47x/tei-platform/pull/2388)の追加検証計画。Native SQLをSource正本とし、Go `DerivedIR`は解析由来のread-only構造。業務意味・現在資格・Effectを自動受理しない。
- **名称の衝突回避**：本線のQX-32は別作業で既に二つのProviderの共同Read Cohort（SQLite 2DB/WAL・13件）に使用されているため、本検証をQX-33として区別する。QX-32の局所実測は実SQLGlot Worker／Go wire接続の証拠ではない。
- **主張する範囲**：実Go `analyzer.RunPythonWorker` の既存入出力コードに基づく応答照合の静的所見、同じコードパスの**隔離されたGoテスト候補**、次段の実SQLGlot Worker→Go→Function Data Context→Provider/Consumer受入条件。
- **停止線**：新しいParser、SQL Engine、Proof Registry、中央Host、第二Source正本、万能Reactive Engineや暗黙の公開Capabilityを作らない。#1640のD2安全拒否と#1119 G0〜G4のOwnerを維持する。

## 1. 2026-11-03の実コード再照合で変わった状況

| 既存実装／証拠 | 現在確かめられること | なお確かめられないこと |
| --- | --- | --- |
| `internal/query/analyzer/client.go` → `RunPythonWorker` | `Request`をJSON Marshalしてサブプロセスstdinへ送り、JSON応答をdecode。**応答 `ProtocolVersion`と`Query`が要求に等しいか**を検査する（#2471 **merged**）。コード内に「SQL/schema bytesには束縛していない」と明記 | 応答が**どのSQL Source/dialect/Schema/Worker実装**に由来するかを、応答だけから第三者が独立再検証する手段。Subprocess起動した事実は、SQL意味解析の完全性・安全な開示資格の証明ではない |
| `internal/query/derived_qx27_dependency_characterization_test.go`（#2634 **merged**） | 合成`analyzer.Result`に非表示WHERE/Sort/相関条件があればGo `DerivedIR`がExpressionやField Occurrenceとして保持できる例、Go側が手製`Lowering=full`を別のColumnRefs完全性証明にしない特性 | **実SQLGlot Workerが任意Queryの必要ColumnRefs・否定存在・Scopeを完全列挙すること**。Input completeness≠Coverage flag |
| `internal/driver/derived_query_execution.go`（#2637 **merged**） | D2 Simple Browseのread dispatch前と、Raw/Derived返却直前のHost Field Read再確認が実装された（合成Callback負例）。D2実行可能SQLの範囲は従前どおり | revoke→regrant ABA、DB/RLS・OS egressの現在資格、異Provider/Consumer、QX-31相関NOT EXISTSのD2実行許可 |
| QX-31 SQLite 12件／QX-26 WAL 11件／QX-28値意味／#2627 Read Cohort | Path再束縛、0件の負依存、ページ間の観測時点／比較演算・Value Semanticsの有限反例 | それら**独立Fixtureを合わせた製品Query→Context→Claim E2E**、正式Providerの完全取得・Consumer公開 |

**既存テストの新しい解釈**：`RunPythonWorker`が直接一つのプロセスを起動してレスポンスを受け取る場合、呼出側はその**Invocationのstdin/stdout対応**を観測し得る。したがって「現在の実Workerが他SQLの結果を誤送出する不具合」とは断定しない。一方、応答`Query`/protocolだけでは**保存した解析結果の再利用、別プロセスの応答取り違え、キャッシュ・異Target・Schema改版**の同一性証拠には不十分であり、実際に受信したレスポンスが入力SQLに正しく由来するかを応答単体で検証できない。

## 2. QX-33の実Goプロトコル特性試験

[**client_qx33_request_binding_characterization_test.go**](../../internal/query/analyzer/client_qx33_request_binding_characterization_test.go) を既存#2388ブランチへ追加。**テスト本体のローカル`gofmt -l`は空**。ローカルで整形したソースとGitHub blobのGit SHA-1 **`f0c6113ca2ff5a3fd3b3febe8faf18c6855d1cb2`**一致（3443 bytes）。

**追加の限定ローカル実測**：Go **1.23.2**の隔離一時module（`module tei`）にGitHub mainの実`internal/query/analyzer/client.go`を**byte一致で複写**（Git blob `9620cd9e2ad6d7b041ca868318a3f45da09fc59d`）、GitHub保存とbyte一致する本QX-33試験（`f0c6113ca2ff5a3fd3b3febe8faf18c6855d1cb2`）を配置し、`model.go`の必要な型定義、`core`のCanonicalID、`environment`のDatabaseProviderのみ**ローカル用stub**で再現した。隔離`go test -count=1 -v ./internal/query/analyzer`はTest本体・`same-source / changed-predicate / changed-referenced-table / changed-schema`四つのsubtestとHelperが**すべてPASS**、`go test -race -count=1 ./internal/query/analyzer`と`go vet ./internal/query/analyzer`もPASS。これは**実Go worker clientのソース・テストの局所実行**であり、TEI repo本体のexact-head CI、実SQLGlot Worker、実CanonicalID/Environmentの全実装・Source/Schema真正性・製品Query E2EのPASSではない。PR headの`go test ./...`／Actionsは**未確認**。

現行本体`RunPythonWorker`を呼ぶサブプロセス用テストHelperが、異なるSQL Source/Schemaを**意図的に無視**して同じ分析Resultを返す。Sourceの誤解析を実Workerで再現したものではない。

| Case | 同じQuery ID／Protocolに対する入力変更 | この試験が狭く特徴づけるもの |
| --- | --- | --- |
| QX33-A | 一切変更なし | プロセス入出力の正常なResponse照合 |
| QX33-B | `hidden_score > 50`→`> 70` | Source条件が変わっても応答単体には元Source digest／解析依存Versionがなく、Query+Protocol一致だけなら異なる入力と同形の応答を受理する |
| QX33-C | `orders`→`other_orders` | Source tableが異なっても同じQuery ID/Protocolだけでは解析結果の**対象Source**を独立検証できない |
| QX33-D | `SchemaContext`から`hidden_score`を除去 | Schema候補／物理Bindingの差も応答上の identity に含まれない |

**一部正例／既存資格**：#2471の既存`TestWorkerResponseMatchesRequestIdentity`は誤Query IDやProtocol Version不一致を拒否する。QX-33はその実装を「壊れている」と主張しない。実Source/Schemaの保証をその既存資格から暗黙に拡張することだけを止める。

## 3. 一本の縦断を成立させるための証拠の受渡し（QX33-V0〜V4）

新しい汎用証拠Schemaではなく、**同一の実行Attemptについて既存Ownerが持つ記録の対応を確認する受入票**として扱う。

| 区間 | 必要な証拠・一致させる版 | 省略したときの誤成功 |
| --- | --- | --- |
| **V0 Authored Source→Invocation** | exact SQL Source bytes／選択Template分岐とParameter、Query ID、Dialect、Schema Contextの実消費版と物理Target/source mapping、解析Command/Worker path & binary/script exact version、**誰がいつどのプロセスへstdinで渡したか** | Query ID一致だけで別SQLやSchema改版の解析Resultを再利用。Worker source版や選択Targetを記録せず、後から`full`を真の網羅として利用 |
| **V1 Invocation→Analyzer Result→Go DerivedIR** | Request ID/Protocolに加えてInvocationのstdin/stdout byteとExit/Diagnosticsを同じProcess Attemptで対応。実Workerが出したColumnRefs／Scopes／OperationsとGo `DeriveFromAnalysis`のField/Operation namespace（`OutputField.Inputs`はColumnOccurrence、`RelNode.Inputs`はOperation）を対照 | 解析Flag=fullやOutput値LineageだけでWHERE/ORDER/EXISTSの**typedな依存閉包**・Relation path/否定的集合条件の完全性を証明 |
| **V2 Query→Function Context／Data Rule** | accepted Source/Operator Meaning、Functionが明示Mappingした必要Relation/Field/Rule Scope、Selected Entity/Path Binding、必要Read closure／Predicate／Measure/grain/Null/Order/Provider対応と未対応診断 | 同名`where`/同じ正規化データの形だけで、動的Path/0件NOT EXISTSのRule成立やOwner Accepted Meaningを推定 |
| **V3 Actual Provider→Result Slice／Reactive Claim** | 実Provider/Principal/Read Cohort／Page continuation／rowsetとField completeness、変更観測と有効期間、Requested Scope/negative existence witness、原Readと再評価の版差 | 部分Pageでの空結果、異Provider同名snapshot、ABA後の同値、未通知変更を`現在Complete`・`影響なし`へ昇格 |
| **V4 Consumer／Action** | Caller/Purpose別の現在Disclosure／実際のConsumer受取、Query Explain/Error／Rule Outcomeの情報感度、Action Intent実対象／Host Current Grant／実Provider Effect・Receipt | SQL/IRのraw秘密条件公開、UI表示の同等性→現在のAction権限、Hostのread pre/postチェック→多時点Tx/OS egressを保証 |

比較時には**観測した事実／独立Expected／未観測／Owner判断待ち**の列を分ける。V0・V1の実解析通過が無い場合、QX31のSQLite独立結果をV2〜V4の合格へ足し込まない。説明不足・機密を理由にConsumerへRaw IRを出せない場合も、内部解析の存在自体を否定しない。

## 4. 次の実装優先順位：同じ問題の人工テストを増やす前に縦断を測る

**最優先はQX33-V0/V1**。実SQLGlot Workerが呼べる環境で、`SELECT id WHERE hidden_score>50`、JOIN/相関`NOT EXISTS`、同IDだがPredicate差、Schema/Target差の**小さな正負対**を用意。Invocationの実消費SQL bytesとAnalyzer JSON、Go `DerivedIR`、未解析Column/Operation/Scopeを同じAttemptで保存する。Worker不在・非対応・partialの場合はそのまま結果にし、新しいSQL Parserを作って「合格」にしない。

- **次点V2**：#2506/#2624/#2586の一つのFunction Contextに、SQL Queryの条件Field・Relation path/negative dependencyの必要入力を明示し、未対応なら`unresolved`を伝播させる。STUDIO/SHOでread-only説明と安全なSource navigationを比べる。
- **その次V3**：#2622/#2627/#2582が実Provider・Read Cohort/Field observation、Relation再束縛/partial page/ABAでClaimの再確認資格を測る。全Providerに等しいCursor／Snapshot保証を仮定しない。
- **最後V4**：#2516/#1178/#2515が情報を絞った説明・実Receiverへ渡したbyteを観測し、#1174/#2598の現行Host Grant・実Effectを独立検査。実Actionまでの境界を静的SQL読取り成功から緩めない。

**プロジェクト上の優先順**：QX-22〜31のPython/SQLite/Java個別Referenceを増やすより、**実ProducerであるSQLGlot Workerの固定版からGo IRへ1件の正負対**を先に完成させる。これを実証できない限り新QX番号/追加の人工Oracleだけで仕様採用を進めない。既存#1119 G0〜G4のGateは変更しない。#2514の独立Oracleと#2516の用途相対Contextを利用し、TEI独自Host／第二SQL Evaluator／全能権限Engineは不要。

## 5. 状態・Owner・保留判断

- **確認済み（本ドキュメントの作業）**：現Go`RunPythonWorker`のID/Protocol照合範囲とRequest/Result型、#2471/#2634/#2637がmerged、QX-33のGoテストSource保存・整形・exact Git blob一致。加えて**Go 1.23.2隔離module**で実Client実装byte一致＋Stubの周辺型に対する4 subtest／Helper、race、vetを局所実行してPASS。
- **未確認／NOT RUN**：**TEIリポジトリ全体のexact-head Go suite/CI**、実SQLGlot Workerの動作・プロトコルレスポンスのSource完全性、SQLGlot→Go→正式Function Contextの同一Attempt、実DB/PostgreSQL/API/Policy、正式Reactive Runtime、Consumer実送信、実Host/Receipt、Owner意味受理。
- **Owner判断待ち**：既存Worker wireにSource/Schema bindingを**追加フィールドとして採用するか**、当面はCallerが保管するInvocation envelope／受理済みSource版／Worker実行記録を用いるか。Hashを返すだけでは真正な実消費やWorkerの完全な解析を証明できない。API/Schemaを本Referenceから先行採用しない。


## 6. QX-33横断追補：不変の解析成果を再利用できることと、現在のClaimを再利用できることは別（2026-11-03／計画、製品E2E未実行）

QX-14の**同一Request byteへ戻るA(r1)→B(r2)→A(r3)**、#2582のRelation path再束縛と0件否定依存、QX-27の非出力Predicate/Sort、QX-32の二Provider Read Cohort分断、SHO/TAIの版付きConsumerを同じ一つの「有効/無効」フラグへ押し込まない。**これらは既存Ownerの証拠を用途別に照合する受入票であり、新しいCore Evidence Schema／全域Cache／Query Registryではない**。

### 6.1 再利用の資格を五つの独立判断として扱う

| 資格候補 | 比較する根拠 | 認め得る利用 | この資格だけで許されない利用 |
| --- | --- | --- | --- |
| **S: 構造解析再利用** | 固定したSQL著者入力・展開後SQL（該当時）・Dialect・SchemaContext・Worker artifact/依存版・解析設定と、実RunのRequest/Result帰属 | 同じ解析入力に対して**決定的で再現可能な**DerivedIRを読み直す候補 | Query現在版の確定、実DBにおける名前解決・実Provider read、現在の業務Meaning |
| **C: Context適用** | Function mapping/Rule版・Selected EntityとRelation pathの現在Binding・選択epoch・必要Predicate/Field/Grain | 現Contextに対する「依存がそろった」候補 | SQL構造が同じという理由だけの現行Value/Row/Rule成立 |
| **R: Read／Claim成立** | Provider/Principal・必要集合Scope・実Readの全件性/Field presence・Read Cohortと鮮度・否定的依存・変化の観測 | 当該Scopeと時点に相対的なClaimの限定判断 | 異Providerの`snapshot`文字列一致だけで共同Snapshot保証。0件部分Pageを不存在と断定 |
| **D: Consumer公開** | Caller/Purpose・現在のField/Row/SQL詳細の開示資格、実受取byte、Source/Ruleの引用版 | 安全に縮約したread-only説明の公開 | Raw IRや隠れたWHERE/ORDER/Rule条件の無断送出、公開許可の実行許可への昇格 |
| **E: 実Action/Effect** | 現在のHost/Provider Grant、Intent対象束縛、実Tx/Commit/Receipt | 許されたActionと完了証拠の限定判断 | 内部解析・UI成功・D2読取成功から書込許可や業務Commitを推定 |

Sにおける**「同じ入力byte」だけでは再現可能性が自動成立しない**。Workerの実体／設定／依存と、その内容を実際に解析したというProducer側の観測が必要である。現在の実Producer不在によりSも**製品としては未確認**。この五区分の名称は**計画上の表記**であり公開API enumではない。

### 6.2 不一致が起きたとき、どの資格だけを失効候補にするか

| 反証ケース | 失効・再確認するもの | 失効させると断定しないもの |
| --- | --- | --- |
| **A→B→A**：SQLのRequest byteはr1とr3で完全同一 | 「その間の変更なし」「現在Revision r1」「古いRunの現在性」資格。C/R/D/Eは当該現在時点で再確認 | **純粋・決定的**な構造解析Sのcontent-addressedな再計算省略は、Worker/Schema等を固定し同等性が検証された範囲なら条件付き候補として残る |
| 同Query IDでSchema/Dialect/Worker artifact変更 | 構造Sの入力帰属、Column/Scope/Operation Bindingと下流依存を再検証 | 過去のr1固定版の説明を、その過去版の記録として保存すること |
| SQL/Result出力値は同じだがCustomer→AccountのRelation path再束縛 | C/Rの動的Target/必要入力、Rule/Queryの成立を再評価（QX-31/#2582） | SQL Source byte不変なら、SQL構造を必ず再解析するとは限らない |
| 旧結果0件、後からHoldが加入／取得Pageがpartial | Rの不存在／NOT EXISTS／LEFT JOIN未一致のClaim。Predicate/Scope/Complete witnessを再確認 | 取得済み観測そのものの履歴記録 |
| 異なる二Providerの各Readが個別には成功 | 共同RのSnapshot／整合性を**unresolved**へ留める（QX-32、#2627） | 両Readそれぞれの限定的な観測事実 |
| 結果Schema・表示値不変のまま非出力Predicate/Sort/相関条件が変化 | C/Rの必要依存・表示集合/順序を再評価（QX-27） | 非表示Fieldを当然にD許可で公開できるという判断を**そもそもしない** |
| 解析後にGrantが失効／Caller・用途が変化 | Dと必要ならEの現在資格を別に判定。Raw IR・診断・SQLテキストの漏えいを防ぐ | Sの内部生成物・歴史的な解析事実 |

反証票の結論は**`candidate`／`unresolved`／`denied`／`historical-only`**など、説明用の状態で扱う。単一の`valid=true`、`CoverageFull`、`latest=true`、`exit=0`にまとめない。候補状態をそのまま実行や開示の許可に変換する機能は追加しない。

### 6.3 実際の縦断で最初に要求する観測の順番

1. **V0停止線を守る**：まず既存SQLGlot Producerの所在／配布SHA・実行Protocol／依存を確認する。見つからなければ`NOT RUN: producer-unavailable`を維持し、別の合成Workerを正式Producerとして置き換えない。
2. V01-A/B/Cの**同一Attempt**でGoが送信したstdin byte/Schema/Dialectと受信stdout/診断、Worker固定版を対応させる。Query IDの同一性とSource/Schemaの同一性を別列にする。
3. 実Go`DeriveFromAnalysis`→QX-27の**Output以外のPredicate/Sort依存**を独立Expectedと比較。Workerが依存を送出しなければ、Go変換成功でも`unresolved`のまま#2506へ渡す。
4. 一つのFunction Context/Selected Entityへ接続する**最初のC試験**では、QX-31のRelation path再束縛・0件否定依存、QX-32の複数Providerを**別ケース**とする。両方を一度に通したと見なさない。
5. R/D/Eは既存#2506/#2582/#2627/#2516/#1174で実行する。SHOとAIハーネスへは、内部SQL／Predicate／診断を自動配布せず、**目的別に資格を再確認したProjectionの実受信byte**を比較する。

**判定**：本追補は既に得られた局所反例から引き出した**横断受入条件の採用候補**。この追記によってQX-33のV0/V1がPASSになるわけではない。QX-14b/c（#2657）・QX-14d（#2661）のテストはDraft、それらの隔離Go PASSと製品exact-head CIを混同しない。

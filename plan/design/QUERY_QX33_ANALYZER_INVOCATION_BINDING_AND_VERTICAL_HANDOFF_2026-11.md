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

[**client_qx33_request_binding_characterization_test.go**](../../internal/query/analyzer/client_qx33_request_binding_characterization_test.go) を既存#2388ブランチへ追加。**テスト本体のローカル`gofmt -l`は空**。ローカルで整形したソースとGitHub blobのGit SHA-1 **`170806331c8d86f2eff3f10450316d51c87044b8`**一致（3443 bytes）。

現時点では**GitHubへのGoテストソース保存と整形・byte一致の確認まで**。PR headの`go test`/Actions成功はこの設計書保存時点では**NOT VERIFIED**。Go実テストは実CI/実行ログ確認後に限って成功と記す。ここで「4/4 PASS」等の推測をしない。

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

- **確認済み（本ドキュメントの作業）**：現Go`RunPythonWorker`のID/Protocol照合範囲とRequest/Result型、#2471/#2634/#2637がmerged、QX-33のGoテストSource保存・整形・exact Git blob一致。
- **未確認／NOT RUN**：QX-33 exact-head Go test/CI、実SQLGlot Workerの動作・プロトコルレスポンスのSource完全性、SQLGlot→Go→正式Function Contextの同一Attempt、実DB/PostgreSQL/API/Policy、正式Reactive Runtime、Consumer実送信、実Host/Receipt、Owner意味受理。
- **Owner判断待ち**：既存Worker wireにSource/Schema bindingを**追加フィールドとして採用するか**、当面はCallerが保管するInvocation envelope／受理済みSource版／Worker実行記録を用いるか。Hashを返すだけでは真正な実消費やWorkerの完全な解析を証明できない。API/Schemaを本Referenceから先行採用しない。

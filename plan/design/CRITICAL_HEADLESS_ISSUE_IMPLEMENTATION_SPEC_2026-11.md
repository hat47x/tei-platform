# 重要Issueの実装仕様：確定結果・再送・操作契約の発見

- 日付: 2026-11-11
- 状態: **実装に向けた設計候補**。以下で選んだ方式を次の実装の既定案とする。製品契約の採用、実装完了、G1/G2合格を表すものではない。
- 基線: main `50dd4bbf15349b2ab339c62c9275b3da724d4c87`。Issue本文・コメントと、この版のコードを照合した。
- 再照合: main `7248ce7c92dd81d9e0f096452342eb017a4c30f9`。対象のCommand/effectcommit/discovery/MCPコード、参照する2設計文書、AGENTS/CONTRIBUTINGに基線からの差分なし。
- 対象: [#1174](https://github.com/hat47x/tei-platform/issues/1174)、[#2564](https://github.com/hat47x/tei-platform/issues/2564)、[#3419](https://github.com/hat47x/tei-platform/issues/3419)。親の優先順位は [#1119](https://github.com/hat47x/tei-platform/issues/1119) と [基本開発方針](../MASTER_PLAN.md)に従う。
- 既存仕様との関係: [ADR-0020](../adr/0020-capability-business-outcome-boundary.md)の業務Outcome、[ADR-0048](../adr/0048-shared-application-api-and-transport-adapters.md)の共通操作方針を維持する。本書は未決の接続方法・実装順・受入値だけを所有する。実装の現状は [docs/status.md](../../docs/status.md)を正本とする。

## 1. 今回の優先順位と残る判断

更新順・起票順より、誤った更新や再送の影響、複数入口への波及、既存機能で代替できるかを優先する。未完了Issueの更新順100件と作成順100件から候補を拾い、下表の先頭3件は本文・コメント・実コードまで確認した。全Issueの設計監査を完了したという意味ではない。

| 順位 | Issue・選んだ不足 | 判断の理由 | 今回決めること |
| --- | --- | --- | --- |
| 1 | #1174 確定時拒否をCommandへ返す方法 | 業務拒否を障害に見せると再試行や表示を誤る。output-only契約へ未宣言Outcomeも足せない | 既存submitのDispositionを呼出し結果の分類に使い、Semantic Outcomeとの区別、Unknownの優先順位を固定 |
| 2 | #2564 ABA後の再送と新規操作の区別 | 値が元へ戻るだけで同じ操作を再適用できる。新キー発行やtakeoverは安全性の根拠にならない | OperationID・対象版・要求bytesを固定する条件、照会失敗時の停止、別操作へ進む条件 |
| 3 | #3419 操作ID・失敗符号・入力構造の自己発見 | 合法な入力や同じ操作を、入口ごとの手書き表や内部コードから推測する余地が残る | 最初の4操作のIDと契約版、v0互換を保つdiscovery v1、入力と失敗符号の出典 |
| 次 | #4205 Function Data Contextのcoverage | 実Providerの部分取得を空集合にしないための具体化 | #2586/#2628の実Readと必要集合を照合してから着手 |
| 次 | #2831 セルフビルドの実入力閉包 | 重要なP0並行レーンだが、業務更新の成立条件には加えない | 既存SB-0/A/Bの実行票・境界を確認し、不足する入出力だけ具体化 |
| 次 | #2604 一式公開とReaderの世代固定 | 読取り前後一致と期間中の安定は別 | 既存#4240の反例と不変世代・保持方針を照合して具体化 |

後続3件の順位は暫定。実装重複や新しい阻害が判明したら見直す。GUI完成や新しい汎用Hostは先頭3件の前提に加えない。

## 2. #1174：確定時の結果をどう返すか

### 2.1 再利用するものと変更点

現行の `internal/security/effectcommit/commit.go` は版付き認可→Receipt照合→対象版CAS→Receipt保存→Commitを持つ。`submit/submit.go` は既に `Committed/Replayed/NotCommitted/Uncertain/Failed/NotRecorded` と `Status` を分類する。これらを再実装しない。

不足は `commandadapter.Executor.Execute` が `denied` と `resource-conflict` をGo errorへ戻し、呼出し結果の分類がCommandとsubmitで異なること。本仕様では、**確定を試みた結果は既存submitの分類、業務の名前付き結末はCapabilityの宣言**として扱う。未宣言の `denied` Outcomeを生成しない。

最初の変更は既存submitの分類関数をCommandの呼出し側でも使用できるようにすることとする。`command.Result` の既存フィールドや `Executor` interfaceを一括変更しない。Go errorによる互換経路は残し、型付きの呼出し結果を得られる新経路へ対象1操作ずつ移行する。全入口の共通envelopeは作らない。

具体的な戻り値は次を選ぶ。名前は新規追加する識別子であり、現行実装ではない。

```go
// internal/command。Provider packageをimportしない内部の観測。
type CommitObservation struct {
    Disposition string // submit.Outcome.Dispositionの文字列をそのまま写す
    Status      string // effectcommit.Result.Status。元Statusがなければ空
    OperationID string // 実行前に固定したID。Receiptがなくても保持
}
type InvocationReport struct {
    Result Result
    Commit *CommitObservation // 未報告Executorはnil。確定を推定しない
}
```

`command.Result`には `Commit *CommitObservation` を `json:"-"` で追加する。`commandadapter.Execute`はRunnerを一度だけ呼び、直後の元result/errorを `submit.Classify` へ渡し、switchで互換errorへ変換する前にCommitへ格納する。denied/conflict/errorを返すときもこの内部欄を返す。Dispositionの再判定をCommand内で実装しない。

`Invoker.InvokeWithReport(ctx, request) (InvocationReport, error)` を追加し、現在のInvoke本体を共有するprivate処理に寄せる。Executor結果を受け取った直後にCommitをrequest-local変数へコピーし、以後の失敗でもreportへ保持する。既存Invokeは同じ処理の既存Result/error投影とし、旧JSONへ内部欄を出さない。Lifecycleも `InvokeWithReport` を追加し同じprivate処理を使う。既存のrejected/unknown記録規則は維持し、型付き呼出し側はreportとerrorを§2.2の優先順位で解釈する。共有Executorフィールドやcontextへ結果を退避しない。並行呼出しで別Operationの観測が混ざる試験を必須にする。

### 2.2 結果の決定表

ここで「今回の効果なし」は**当該確定要求が新たな書込みをしなかったこと**だけを指す。過去の同OperationIDの効果や、同Runの先行Stepまで無作用とはしない。

| 観測 | 呼出し側の分類 | 業務Outcome | 再送・次の動作 |
| --- | --- | --- | --- |
| Execute: committed、Commit応答あり | Committed | 宣言と実応答にある場合だけ | 新規Executeは行わず、必要な独立Readへ |
| Execute/Lookup: replayed、要求一致 | Replayed | Receiptだけから復元・創作しない | 過去の確定記録として表示。現在値確認は別Read |
| Execute: denied | NotCommitted + denied | 追加しない | 自動再試行なし。現在の権限・業務前提を再評価 |
| Execute: resource-conflict | NotCommitted + resource-conflict | 追加しない | 自動再試行なし。元要求を保持して対象を再Read |
| Execute: CommitUncertainError | Uncertain | 決めない | 同OperationIDのLookupのみ。新ID再送を止める |
| Lookup: not-recorded | NotRecorded | 決めない | 元の確定結果は不明のまま。§3の条件なしにExecuteへ進まない |
| Lookup: denied/競合/通信失敗 | 既存ClassifyLookupの分類 | 決めない | 照会失敗として扱い、元のUncertainを維持 |
| 実行前の入力不正・Grant取得失敗 | Failed、既存の安定符号 | なし | Runner呼出し0件を確認。修正した要求は新規操作として扱う |
| Executor後の契約違反・Recorder/Stage保存障害 | EffectUnknownがあればUncertain相当を優先 | 決めない | `COMMAND_EFFECT_UNKNOWN`を入口へ保持。Receipt照会へ |

既存 `submit.Classify` の `Failed` は「効果なし」の証明ではない。分類と効果の確実性を分け、`command.EffectMayBeUnknown` がtrueなら入口の `ReportedFailure` に従いUnknownを優先する。raw errorしか返せない外部Providerでは、安全な無作用を証明できない実行後障害をUnknownへ写像する責任をProvider Adapterが持つ。

Lookupの `NotCommitted` は「今回の照会が拒否された」意味であり、元の更新の不成立へ書き換えない。履歴の更新結果と照会結果を別に保持する。current Grant失効時は古いReceipt本文を開示しない。

### 2.3 最初の実装単位と公開範囲

1. **EF-1 分類の共有**：`submit` と `commandadapter` の対照試験を追加。Commandの既存error互換と型付き分類の関係を固定する。対象は既存単一Resourceの実PG経路だけ。
2. **EF-2 入口投影**：同じCommand呼出し結果をCLIと既存Web呼出し側へ渡す。CLI exitやHTTP statusの現行値はこの変更で一括変更せず、型付き分類と旧投影を試験で対照する。wire変更は当該操作の別versionで提供する。
3. **EF-3 実Commit境界**：既存PG fixtureで認可失効・CAS不一致・Receipt INSERT失敗・ACK喪失を注入。`NewWithTrustedOwnerRoutine` の既存経路を再利用する。宣言したGuaranteeと、その配備の適合性証拠は分ける。

`command.Lifecycle` の `MayReplay=false`、Recorder失敗時Unknown、PG/SQLiteの段階storeは基線に既にあるため、未実装として起票し直さない。Lifecycleの段階とProvider Commitは別の観測として残す。

### 2.4 独立Expected

| ID | 入力・故障 | 必須の期待値 |
| --- | --- | --- |
| EF-A01 | output-only Capability、確定時にGrant失効 | denied/NotCommitted。未宣言Outcomeなし。当該要求の対象変更0・Receipt追加0 |
| EF-A02 | 対象版だけ不一致 | resource-conflict/NotCommitted。実行障害や成功へ変換しない |
| EF-A03 | Commit成功後に応答を失う | Uncertain。新OperationID生成0。既存ID LookupでReplayed。Receipt件数1 |
| EF-A04 | EF-A03後のGrant失効 | Lookup拒否、元のUncertainを保持、Receipt秘密項目非開示 |
| EF-A05 | 別Tx ExecutorがCommit後、Recorder/Stage保存失敗 | Unknown優先。外側Rollback成功でも無作用と表示しない |
| EF-A06 | 正常Commit後に第三者がResourceを変更 | Receipt観測は成立し得るが、資源現在値一致の資格を付けない |

EF-A01/02の0件確認は、独立接続で試験対象の効果表・Receiptをreadする。スパイの呼出し0件だけでは実DB無作用の証拠にしない。障害後も同じScope/OperationID/要求bytesを使う。実DB試験のSKIPはPASSと数えない。

## 3. #2564：同じ操作を、別の操作として再送しない

### 3.1 identityの既定

製品の最初の更新経路は **既存ExpectedVersion CAS + OperationID一意Receipt** を使う。値一致だけの参照方式は、ABAを許容することを業務が明示した実験に限定する。値がA→B→Aへ戻っても対象版は増えるため、旧ExpectedVersionによる新ID要求は競合する。既存実装の版CASを使い、第二のOperation ledgerを作らない。

| 情報 | 固定・変更の規則 |
| --- | --- |
| OperationID | 利用者またはJobが一回の操作を確定するとき発行。送信前にHostの既存永続記録へ保存。通信再試行・プロセス再起動・takeoverで変更しない |
| Scope | Application/Capability/Subject/Action/Resourceを既存Requestへ固定。別主体のtakeoverは同じ要求の再送とみなさない。元操作の照会は現在の許可で別途判定 |
| ExpectedVersion | 最初に取得した対象版を送信前に固定。Lookup・再送の便宜で最新版へ置換しない。0の意味は既存Providerの対象生成規約に従う |
| Payload | 実行へ渡す同じbytesを保存・再利用。JSONのキー順や改行の再直列化でもbytesが変われば別要求。Runnerの既存hashと同じ意味を使う |
| owner/generation | 実行担当と引継ぎを識別。増加しても業務要求のOperationIDや対象版を変更する権利は生じない |
| Grant/認可版 | 照会・実行時に信頼された認可元から再取得。要求hashの代用にせず、Receipt返却前にも検査 |

Command ID、OperationID、Stageのowner/generationの対応は、呼出し側の既存Attempt記録に保存する。対応が欠ける再開要求は自動実行を拒否する。履歴ファイルやPayloadを安全に保存できない構成では自動回復を広告せず、同じIDでの手動照会へ限定する。本番の機密値保存先・保持期間・監査鍵は [#2845](https://github.com/hat47x/tei-platform/issues/2845)に残す。ここでは合成データだけで実装・検証を進める。

**保存方式の選択**：最初のPoCは既存 `internal/command/lifecyclesqlite` を拡張する。現在のAttemptには要求bytesの保存欄がないため、同じSQLite DBへ `command_request_metadata(command_id PRIMARY KEY, schema_version, request_bytes, request_sha256)` を追加する。`schema_version`は1、bytesはGoで一度だけ直列化した `{commandId, commandRequest, effectRequest}`（Command IDと既存 `command.Request` / `effectcommit.Request` の値）とする。effectRequestにはScope/OperationID/ExpectedVersion/Payloadの全項目を保持する。bytesの組立て・復元は呼出し側で行い、SQLite Storeはopaque bytesとCommand IDを保存する。Storeからsecurity packageをimportしない。GrantやCredentialは保存しない。

Storeに `DispatchPrepared` を追加し、metadata INSERTと既存Dispatchの段階更新を同じSQLite Txで確定してから外部Executeを呼ぶ。同Command IDの既存metadataは不変とし、異bytesは拒否、同bytesでも既存Dispatch規則が拒否すれば実行しない。既存Dispatch経路は変えず、回復対応を広告する構成だけがこの新経路を使う。起動時は同じDBの `LoadPrepared` からbytesを読み、hash・schema・Command IDとの対応を検査し、不在/破損ならExecuteしない。SQLite準備記録とPG Effectは別Txであり、この記録だけではPGの結果を証明しない。

既存 `LifecycleInvoker.Invoke(Takeover=true)` はtakeover後そのままInvokerへ進む。新しい回復対応構成はこの入口を直接呼ばず、専用の回復関数へ接続する。元Stageごとの処理は次のとおりとする。

- **unknown**: `LoadPrepared → Lifecycle.Takeover → Runner.Lookup`。Replayedなら現owner/generationを条件にObserveへ進む。NotRecorded/Denied/errorなら、引継ぎでdispatchedになった同owner/generationをMarkUnknownへ戻して元操作の不明状態を維持する。段階保存が失敗してもReceiptの観測を回復Reportに保持し、Unknownを返す。
- **dispatched**: Execute後にMarkUnknown前でProcessが停止した場合も含む。既存TakeoverはこのStageを拒否するため、owner/generationを変更せず、現在の認可を経て**read-only Lookupだけ**を許可する。結果は回復Reportへ返し、Stage更新・Takeover・Executeを行わない。Replayedを読めても自動的に段階記録が復旧したとは表示しない。
- **observed/記録なし**: observedは既存の結果参照へ進む。記録なし/metadata欠損では回復Executeを拒否する。各場合とも新IDを自動発行しない。

旧Invocation経路は従来どおりだが、「Lookup優先回復に対応」と広告しない。Step/Stageの所有権と実Providerの権限は別に検査する。

### 3.2 結果不明からの遷移

```text
送信前に要求とIDを保存 → Execute
  committed/replayed → 確定記録あり → 必要なResource read
  denied/conflict   → 今回の新規確定なし → 再評価（自動retryなし）
  uncertain         → 同じID/要求でLookup
      replayed      → 過去の確定記録あり → 必要なResource read
      not-recorded  → unknown継続
      denied/error  → unknown継続（開示可能な照会失敗だけを返す）
```

`not-recorded`だけでは元のTxが進行中かもしれない。同IDのExecute再開を許すのは、**元実行が終了し、古い実行役からの遅延確定がProviderで拒否され、同ID一意制約と同Tx効果保存が機能し、現在Grantがあること**を既存Provider/所有機構で確認できた場合に限る。初回実装ではこの自動再開を無効にする。照会は利用者の明示操作で繰り返せるが、自動再送や新ID発行へ移らない。

新規操作は対象の再Readと現在の権限評価を経て、呼出し元が**新たな業務操作として明示的に確定**した場合に作る。未解決の同一業務操作がある場合は新ID作成を止める。関連Resource更新など、正当な別操作の意味は既存Capability/業務Ownerが決める。Coreに「同じPayloadなら永遠に重複」という一般規則を加えない。

### 3.3 受入試験と実装順

| ID | 対照 | 必須の期待値 |
| --- | --- | --- |
| RCAS-S01 | 同ID/同Scope/同版/同bytesを再送 | Receipt 1件、新たな効果0、Replayed |
| RCAS-S02 | 同IDで対象・期待版・Payloadをそれぞれ1項目変更 | すべて衝突拒否。古いReceipt本文を返さない。既存resource-conflict表現を維持 |
| RCAS-S03 | A(v1)→B(v2)→A(v3)、新ID・旧v1で更新 | resource-conflict、v3を維持。値一致の参照方式では通る反例と対照 |
| RCAS-S04 | ACK喪失→Host再起動。unknown記録あり／MarkUnknown前のProcess killでdispatchedの二対照 | 元ID/元要求を復元しLookup。unknownのみgeneration更新、dispatchedはread-only照会・Stage不変。ID消失は実行拒否。新ID発行0 |
| RCAS-S05 | Receipt不在の照会中に元Txが未完了 | unknown継続。自動Execute件数0。後の確定を再照会できる |
| RCAS-S06 | 別主体/別Tenantが同IDを照会 | 元Receipt非開示。権限拒否から元操作の未確定を推論しない |

順序は **RCAS-S02の現製品経路確認→S03の版CAS対照→S04/S05の呼出し側回復方針→S06**。参照ツール#3678の期待値hash修正は再実装しない。`effectcommit.Request`は既に期待版をhashへ含むが、実Adapterを通るS02でその維持を確認する。

## 4. #3419：コードを読まずに操作と合法Requestを見つける

### 4.1 最初の4操作と所有者

| operationId（提案） | contractVersion | CLI | MCP | 共通処理の出発点 |
| --- | --- | --- | --- | --- |
| tei.workspace.validate | 1 | validate | tei_validate | `internal/operations/validate.go`、`validation_report.go` |
| tei.model.catalog | 1 | model catalog | tei_model_catalog | `internal/operations/model_catalog.go` |
| tei.model.inspect | 1 | inspect | tei_inspect | `internal/operations/inspect.go` |
| tei.model.trace | 1 | trace | tei_trace | `internal/operations/trace.go` |

operationIdは入口名の改名で変更しない。意味上の要求・結果の互換性を壊す場合はcontractVersionを上げる。CLI結果の `tei.cli.* /v0` はwireの版であり、この数字とは独立。MCPは現基線でlocate/overviewも広告するため、4操作への移行時も削除しない。残り2操作の移行は既存対応表を保持して別単位で行う。

### 4.2 discoveryの提供方法

新しい選択肢として **`tei commands -contract-version 1`** を追加する。これは提案中の構文であり現行機能ではない。省略時のv0と既存出力は維持する。未対応の版は既存 `INVALID_ARGUMENT`/exit 2で拒否し、v0へfallbackしない。

現行の型は `cmd/tei/cli_contract.go` の `cliCommandCatalog/cliCommandRecord`、版は `tei.cli.discovery/v0`。v1は `schemaVersion: "tei.cli.discovery/v1"` を返し、既存のcommand entryを維持した上で、下表の `operation` を追加する。未移行entryは `operation: null` とし、広告済み機能が未実装になったという意味にはしない。4操作のentryではoperationを必須とする。v1移行完了率は別に数える。

| operationの欄 | 型・規則 |
| --- | --- |
| id / contractVersion | 上表の文字列 / 正整数。空・null・重複する同一入口は禁止 |
| requestSchema | JSON Schemaのobject。必須・任意、null可否、未知member可否、列挙・既定値・値の上限を実行時validatorと共有 |
| minimalExample | requestSchemaを通る最小object。必須項目の省略例や秘密値を入れない。WorkspaceはHost解決の参照として示す |
| failureCodes | `{code, category}` の配列。categoryは input / operation / validation。各値は現行安定符号を使い、表示文言から推定しない |
| failureCodesComplete | bool。列挙が静的に閉じる場合だけtrue。Provider由来など列挙できないものはfalseとし、配列が網羅的とは読ませない |
| resultSchemaVersion | 当該入口が実際に出すschemaVersion。CLIとMCPの外側frameは一致を要求しない |

最初の4操作は未知memberと明示nullを拒否し、optional未指定だけに既定値を適用する。空文字列は未指定へ置換しない。文字列のtrimで対象を自動修正しない。上限値・列挙はCLIで現在受理される値から抽出し、MCPだけの別既定を作らない。Workspace rootの変更をRequest欄として追加しない。継続tokenは既存の版・対象・Scope検査を通す。

共有Requestは**Hostが解決済みのWorkspace参照を別引数で受け取る処理入力**とし、公開JSONへroot/pathを含めない。CLIの `-workspace` 解決、およびvalidateの既存 `-world` 直接入力はAdapter側の入力方式としてv0に残す。最初のdescriptorのvalidateはWorkspace版だけを記述し、`-world`を同じRequestに混ぜない。

| Request | required | propertiesと既定値（記載以外は禁止） | reference/workspaceでのminimalExample |
| --- | --- | --- | --- |
| validate | なし | 空objectのみ | `{}` |
| catalog | なし | namespace: nonempty string、未指定は全namespace | `{}` |
| inspect | subjectKind, subjectId | 両方string。kindは `operations.InspectionSubjectKinds` のenum、IDは前後空白なし・非空 | `{"subjectKind":"semantic-data","subjectId":"sales.order"}` |
| trace | scope, subjectKind, subjectId | scope/IDは前後空白なし・非空、kindは `operations.TraceSubjectKinds`。direction: outgoing/incoming/both、未指定both。mode: summary/full、未指定summary。offset/limit: integer≥0、未指定0。expectedFingerprint: 非空string、未指定は版条件なし | `{"scope":"reference","subjectKind":"semantic-data","subjectId":"sales.order"}` |

全Requestは `type: object, additionalProperties: false`、null不可。整数はGoのintで表現可能な範囲を実行時に検査し、Schemaでは32bit/64bitのうち当該実行物の上限を返す。traceのsummaryでは明示offset/limitを拒否し、fullで0のlimitは既存の「未指定」を意味する。expectedFingerprintは既存Workspace継続条件へ渡し、Graph-only fingerprintへ置換しない。enumの具体値とint上限をSchema出力時に展開し、ConsumerがGoの変数名を解読する必要をなくす。

入力制限を既存MCPへ適用したときの互換差はCAPI-S2で列挙し、既存の受理値を狭める場合は当該Toolの契約版を上げる移行を先に行う。新しいdescriptionを付けるだけでv0の受理範囲を黙って変更しない。

入力のJSON構造とCLI flagの対応は各Adapterに置く。共有の型・validatorは `internal/operations` または既存の操作所有packageに置き、CLI→共通側への逆依存を作らない。JSON Schemaの公開は検証の代替ではなく、同じ制約から生成・照合する。

MCP `tools/list` は既存Tool名/inputSchemaを保持する。最初の4 Toolの `_meta` に `tei.operationId` と `tei.contractVersion` を追加し、対応するdescriptorを参照する。共通の入力意味をMCP既存inputSchemaへ投影する。CLI flagとMCP JSONのkey名が違う場合はAdapterが明示写像する。既存 `discovery_parity_test.go` の手書き表は、4操作についてID/版でjoinする試験へ置換する。実行時の万能dispatcherや第二Registryを作らない。

### 4.3 エラーと検証不合格

`failureCodes`に載せるのは呼出し失敗の符号であり、validateの各意味診断は従来のdiagnosticsへ残す。validateのwarning-onlyは `ok=true`、error診断/source-root errorは `ok=false`。load失敗は既存FailureReportを使う。新しいdiscovery契約だけで診断やexitの意味を変えない。

比較するのは同じWorkspace・Source/Index版・Scopeでの共通結果。Protocol用JSON-RPC error数値、MCP content、CLI exitと共通failure.codeは別欄で検証する。Trace token不一致や部分読取りを空の正常結果へ変換しない。

### 4.4 実装単位と独立Expected

1. **CAPI-S1**：上表のdescriptor、Request validator、v1 opt-inを追加。4操作の既存v0 fixtureを固定し、byte比較で互換性を検査。
2. **CAPI-S2**：MCPの4 Toolが同じdescriptorを参照し、実行時のnull/型不正/未知欄を拒否。locate/overviewの既存広告と動作を保持。
3. **CAPI-S3**：実CLIと実stdio MCPを起動し、ID/版でjoinして正常・不正入力・意味不合格・staleを比較。処理関数直呼びだけで完了しない。

| ID | 入力・変更 | 必須の期待値 |
| --- | --- | --- |
| CAPI-A01 | v0の従来呼出し / v1明示 / 未知版 | v0不変、v1の4 entry充足、未知版exit 2 |
| CAPI-A02 | CLI/MCP名を一方だけ変更 | ID joinは維持。descriptorのID/版を一方だけ変更した変異は試験FAIL |
| CAPI-A03 | 未知member、数値を文字列欄へ、明示null、空文字 | 実行前拒否、defaultへ置換0。未指定のoptionalだけ既定値 |
| CAPI-A04 | minimalExampleをdiscoveryから読む | validator合格し、隔離Workspaceの実CLI/MCPを成功させる |
| CAPI-A05 | 未掲載のfailure codeを実装へ追加する変異 | complete=trueの対象では試験FAIL。動的Providerはcomplete=falseを維持 |
| CAPI-A06 | 意味エラー、Source末尾LF変更、旧Trace token | 意味不合格と旧token拒否を維持。World FP一致だけでstaleを無視しない |

JSON内容を持つ `knowledge-qualify` の自己発見は次の拡張単位とする。最初の4操作に複雑な要件ファイルのSchemaを無理に混ぜず、同じdescriptor方式が成立した後、既存Request validatorから正確な必須JSON形を公開する。

## 5. 完了の範囲と受渡し

今回の成果は仕様文書とIssueへの参照追加である。EF-1〜3、RCAS-S01〜06、CAPI-S1〜3の実装・新規実DB/CLI/MCP試験は **NOT RUN**。既存試験の到達点を本変更の新しいPASSとして流用しない。

実装は各既存Issueの範囲で、要求ロール `implementer`、返却先 `reviewer` とする。契約・権限・整合性の変更は `independent-reviewer` が別に照合する。開始base SHA、許容変更path、対象Case、実行/未実行の試験、成果PRを記録する。別の子Issueや共通Engineは追加せず、先頭Issue内の上記単位で進める。

- #1174: `internal/security/effectcommit/submit`、`commandadapter`、対象Command呼出し側と対応試験。認可式・Semantic Outcomeを勝手に追加しない。
- #2564: 既存Command Attemptの記録と再開処理、実PG対照試験。機密比較・本番Retentionは#2845のまま。
- #3419: 4操作の型付きdescriptor、CLI discovery、Reference MCPと回帰試験。残りの全操作移行はこのPRへ拡大しない。

各単位は型・分岐・期待値を実装で検証できた時点で完了とし、親Issueは残る実Target/Provider/業務受入条件を満たすまでOpenを維持する。

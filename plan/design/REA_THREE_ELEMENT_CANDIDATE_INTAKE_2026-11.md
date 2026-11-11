# REA外部逆解析から三要素分析モデルへの構造候補取り込み（設計候補）

- **状態**：候補抽出・CLI入口・レビュー票の保存・採用前後比較まで。Owner受理・小規模移行実証・SHOは未着手。
- **Owner**：[TEI #3394](https://github.com/hat47x/tei-platform/issues/3394)。
- **関連**：共通API [ADR-0048](../adr/0048-shared-application-api-and-transport-adapters.md)／[#1158](https://github.com/hat47x/tei-platform/issues/1158)／[#3419](https://github.com/hat47x/tei-platform/issues/3419)、Model Map [#3866](https://github.com/hat47x/tei-platform/pull/3866)、三視点 [THREE_ELEMENT_METHOD_DIAGRAM_CONTRACT_2026-10.md](THREE_ELEMENT_METHOD_DIAGRAM_CONTRACT_2026-10.md)。T0/T1-Aの受入票はまだ文書として存在せず、本節の未着手の作業（Owner受理・移行実証）の中で発行する。
- **実装**：[`internal/reacandidate/`](../../internal/reacandidate/)、土台の三要素モデルは #4283 の [`internal/threeelement/`](../../internal/threeelement/)。

この文書は、後続の実装者が残作業を迷わず再開できるように、**現時点で確定した契約**と**未決事項**と**作業順**を分けて書く。設計判断の正本はADRとIssueであり、ここは機能単位の設計本文である。

## 1. 何を作ったか

REAの**同一Attempt**が返した外部Observationと有向Relationを、TEIの三要素分析モデル（Data／Function／Business）に対応する**構造候補**へ写し、意味を受理する前の状態でレビューできるようにする読み取り専用の層を追加した。

この層は次のことをしない。

- Canonical Subjectを作らない。Workspaceへ書かない。
- 業務上の意味を受理しない。`AcceptedMeaning`は常に`false`で、受理経路を持たない。
- 既存モデルを上書きしない。採用前後比較は**メモリ上の複製**に対してだけ行う。
- 第二のHost／Reader／Registry／Parser／Runnerを追加しない。既存の`operations`と`workspace`を呼ぶだけである。

## 2. 最小経路（最初の実装対象）

Issue #3394 が「最初のProduct Exit」と呼ぶ経路を、次の形で実装した。

```text
REA attempt JSON (未信頼)
  └─ DecodeAttemptEvidence        … 失敗・重複・制御行・上限を閉じて拒否
       └─ GenerateCandidates      … Observation 1件＋有向Relation 1件 → 候補
            └─ BindCandidates     … 既存 operations.InspectSubject/MapWorkspace を読むだけ
                 ├─ PreviewAdoption        … 既存 threeelement.DiffModels/ValidateModel
                 ├─ CompareCandidateSets   … 2つの候補Setの前後比較
                 └─ BuildReviewPacket      … 単一のレビュー票
```

Observationは`data-element`候補へ、有向Relationは`cross-reference`候補へ写る。**1件ずつが独立した候補**であり、ノードをまとめるのは採用previewの内部だけである（§5）。

## 3. 候補の契約

### 3.1 候補Identity

`candidateId`は `kind + 外部Identity + 外部OccurrenceID (+ 有向Relationのfrom/to)` のSHA-256である。**Attempt識別子とSource Revisionを含めない**。

これは意識的な判断である。Attemptを含めると同じ観測が再解析のたびに別候補になり、Source Revisionを含めると「同じOccurrenceのbyteが変わった」が「削除＋追加」に化ける。Identityは**外部Occurrenceに固定**し、byteの変化は`SourceRevision`の差分として同じ候補に現れる。

### 3.2 Source Revision

`SourceRevision`は`artifactPath`／`fingerprint`／`span`／`binding`を持つ。

- `binding = provider-reported` は「REAがpathとdigestを主張しており、誰もまだ読み直していない」という正直な状態である。
- `binding = independently-bound` は、読み込んだWorkspaceの現在byteと一致したときだけ付く。
- `binding = source-digest-conflict` は、解析時byteと現在byteが違うことを表す。**失敗でも受理でもない。** 再解析が必要な状態として残す。

**Source Revisionは現在byteの同一性であって履歴ではない。** A→B→Aでpathとfingerprintが戻っても、途中にBがあったことは証明できない。

### 3.3 構造的事実と推測の分離

| 種別 | 型 | 内容 |
| --- | --- | --- |
| 構造的事実 | `StructuralFacts` | 外部Identity／OccurrenceID／向き／観測state・authority／evidence coverage |
| 推測 | `CandidateInference` | 目標要素、`basis`（構造的な根拠のみ）、`unresolved`（決めなかったこと） |

`basis`の各要素は`rea-directed-edge`・`node-kind:*`・`source:*`のように**構造的な由来**だけを書く。業務語（`customer`など）を根拠にしない。決められなかったことは`unresolved`へ明示し、空にしない。

### 3.4 意味損失

`meaningLoss`は、三要素モデルに対応する際に**表現先が無い**事項を列挙する（外部OccurrenceID、観測state/authority、evidence coverage、source span、artifact digest、relation labelの型付き語彙）。これは「変換で何が落ちるか」を残すための欄であり、後から黙って消さない。

### 3.5 状態

- `reviewStatus`：常に`requires-owner-review`。
- `resolution`：`unresolved`／`ambiguous`／`resolved`。**`resolved`はOwnerの明示mappingと、既存`InspectSubject`が一意に解決できた場合だけ**付く。
- Set全体の`qualification`：`complete`／`partial`／`unknown`。REAのcoverageが部分（`truncated=true`、`omitted_count>0`、`status=unknown/partial`）・未申告、`limitations`または`unknowns`が非空、Sourceが`provider-reported`でない、端点が未観測、のいずれかで`partial`以下へ落とす。

**coverageの実形状（実測で判明）**：REA 6.8.0の実documentは`{status, truncated, omitted_count, limits}`を書き、`exhausted`は**書かない**。旧実装は`exhausted`だけを読んでいたため、実documentを常に「未申告」として`unknown`に落としていた。現在は`truncated`と`status`の両方を読み、`omitted_count`も保持する。`status=unknown`は「列挙を切っていない」ではなく「その成果物を作っていない」を意味するため、`truncated=false`でもcompleteとはしない。

### 3.6 三要素のどこまで支持できるか

`threeElementCoverage`が**要素ごとに**「このAttemptで候補を作れるか」を答える。

| 要素 | 状態 | 理由 |
| --- | --- | --- |
| Data | `supported`（観測nodeまたは有向Relationがある場合） | 外部のnode観測と有向Relationがdata-shapedな事実である |
| Function | `unsupported` | 外部事実はasset/module/relationであって、宣言された入出力・効果を持つ操作ではない。観測したrelation labelは理由へ「考慮した」と記すだけ |
| Business | `unsupported` | 業務上の意味はOwnerの判断であり、外部構造で代替できない |

**この表を持つ理由**：Function/Businessの欄を空欄にすると、レビュアーは「外部解析では作れないと判断した」のか「作り忘れた」のか区別できない。状態・理由・「何があれば支持できるか」を並べて書く。

## 4. 既存read-only APIとのFit/Gap

`AuditCarrierFit`が事実ごとに「既存の型で運べるか」を判定する。最小Attemptでの結論は次のとおり。

**運べる（既存の型に宛先がある）**

| 事実 | 宛先 |
| --- | --- |
| Source path | `sourceindex.Artifact.Path`／`Reference.ArtifactPath` |
| Source digest | `sourceindex.Artifact.Fingerprint` |
| Source span | `sourceindex.Range`（offset/line/column） |
| 有向Relationの向き | `operations.ModelMapLink.From/To`の順序、`threeelement.CrossReference.Directed` |
| Relationラベル | `ModelMapLink.Relation`／`CrossReference.Role` |
| 表示名候補 | `threeelement.DataElement.Name`（提案としてのみ） |
| 対象digest | `sourceindex.Fingerprint`（byteをWorkspaceへ読んだ後） |
| Canonical解決 | `operations.InspectSubject`（**呼出側が既にCanonical Subjectを名指す場合のみ**） |

**運べない（既存Ownerへ返すgap）**

| 事実 | 欠けている理由 |
| --- | --- |
| 外部node／observation／edge Identity | `ModelMapLink`も`SubjectInspection`もkind+CanonicalIDで識別する。外部Identityの欄が無い |
| 外部の端点Identity | 同上。並行Edgeを区別できない |
| 外部node kind | read-only型にも三要素の語彙にも無い |
| 観測state／authority／evidence coverage | 運ぶ欄が無い。`ModelMap`は固定文字列`partial-projection`しか持たない |
| Attempt Identity／provider descriptor／qualification／unknowns | 外部Attemptの来歴を運ぶ型が無い |
| 外部OccurrenceのCanonical解決 | 既存`InspectSubject`はCanonical Subjectを先に要求する。**Owner mappingが無い限り外部Occurrenceは解決できない** |

このgapは#1158／#3419へ返す対象であり、**この層で第二のHostやRegistryを作って閉じない**。

## 5. 採用前後比較とレビュー出力

`PreviewAdoption`は候補集合を**モデルの複製**へ適用し、`before`／`after`／`diff`／`records`／`afterValidation`を返す。

- `diff`は既存`threeelement.DiffModels`をそのまま使う。レビュー者は採用diffと通常diffで同じ語彙を読む。
- `records`は候補ごとに`candidateId`／`sourceRevision`／`basis`／`unresolved`／`meaningLoss`／`before`・`after`のモデルfingerprintを持つ。
- `accepted=false`、`owner=""`を**型として固定**する。受理の入口を作らない。
- 同じ外部nodeに属する複数のObservationは、previewの内部で1つの`DataElement`へまとめる。まとめた事実は`mergedFromCandidates`に残す。**統合するか分けるかはOwnerの判断**であり、previewは構造的な下書きにすぎない。
- 未観測の端点は`unresolved:<externalId>`という要素IDで参照する。要素を捏造せず、既存`ValidateModel`が`UNKNOWN_CROSS_REF_TO`として報告する。

`CompareCandidateSets`は2つの候補Setを3段階で比べる。

1. 同じ`candidateId` → 同一候補として比較（`unchanged`／`same-current-bytes-different-attempt`／`revision-changed`）。
2. 残りを構造key（kind＋path＋digest＋ラベル＋span。**外部Identityを含めない**）で照合 → `external-identity-renumbered`。外部IDの採番が変わっただけの再解析を「削除＋追加」と誤報しないためである。
3. どちらでも残ったもの → `added`／`removed`。

`same-current-bytes-different-attempt`には「現在byteの一致は中間変更が無かったことの証拠ではない」「以前のreview/grant/acceptanceを復活させない」を必ず添える。

`BuildReviewPacket`は上記を1枚にまとめ、`unresolved`を集計（重複排除・上限256件・超過は`unresolved-summary-truncated`）し、`reviewSequence`で確認順を示す。

## 6. 実装済みの否定対照

`internal/reacandidate/*_test.go`が次を固定する。

| ケース | 期待 |
| --- | --- |
| 同名の異なる要素 | 2候補のまま。`same-suggested-name-distinct-external-identity`を記録し、統合しない |
| 多重Relation（末端同一・ラベル同一） | それぞれ保持し、`parallel-relation`／`same-endpoints-same-label`を別々に記録 |
| 向きの違い（A→B と B→A） | 両方保持し、`direction-conflict-pair`としてOwnerへ残す |
| 未解決参照 | 端点を捏造せず`target-endpoint-not-observed-in-this-attempt:<id>`を記録 |
| 部分取得 | `limitations`／`unknowns`／未網羅／coverage未申告で`complete`へ昇格しない |
| A→B→A | 現在byteの一致を「中間変更なし」と扱わない |
| Source改版 | 同じcandidateの`revision-changed`。新規受理が必要 |
| 外部IDの採番変更 | `external-identity-renumbered`。削除＋追加にしない |
| Source digest不一致 | `source-digest-conflict`。受理にも失敗にも変換しない |
| Owner mapping無し | 何も`resolved`にしない |
| read-only port | 書込み手段を持たない（型に無い） |
| 解析byte＝現在byte | `independently-bound`かつ`resolution=unresolved`。coincident宣言は「構造的一致であり受理ではない」と注記（`cmd/tei/cli_three_element_rea_workspace_test.go`） |
| 解析pathがWorkspace外 | decode時に`provider-reported-invalid-source-path`。束縛しない |
| Owner mappingあり | `inspectStatus`が`not-attempted`にならない（既存`operations.InspectSubject`が呼ばれる） |

**束縛で観測できる範囲の注意**：Source Indexが記録する`Range`は**宣言全体ではなく識別子トークンの範囲**である。実Workspaceでの`source-index`出力では`semantic-data:probe.order`が`3:6-3:11`（`Order`の5文字）だった。したがってREAが宣言全体を覆う広いspanを報告した場合、coincident宣言には一致しない。これは誤検出を避ける保守的な挙動であり、候補の`unresolved`は変わらないが、「一致しないのは不具合」と読み違えないこと。

## 7. 作業の進捗と残り（後続実装者向け）

### 7.1 CLI入口（実装済み）

`tei three-element rea-intake`を追加した。フラグは`-evidence`（必須）、`-workspace`、`-model`、`-mapping`、`-before`、`-output`である。出力は`tei.cli.three-element.rea-intake/v0`の封筒で、`result`がレビュー票そのものである。

**当初案から変えた点（重要）**

前回の本案は「`selectedReadOnlyFlagAllowlist["three-element"]`を広げ、カタログへレコードを追加する」としていた。実装時にこの案は**既存の検査と矛盾する**ことが分かったため、採らなかった。

- `three-element`は`main.go`の`command == "three-element"`分岐で**共有FlagSetが組み立てられる前に**処理を終える。したがって`validateSelectedReadOnlyFlags`は通らず、allowlistの項目は「受理されるフラグの集合」ではなく**誰も強制しない宣言**になる。
- 一方`TestCommandCatalogFlagsMatchAcceptedFlags`は「allowlistにあるが`tei commands`に載っていない」を失敗にする。`three-element`をカタログへ載せると、今度は`TestAcceptedCommandsMatchDispatch`が「`command`の比較か`case`で見えない」として落ちる（`init()`で足す既存の逃げ道はallowlist表とも突き合わせられる）。
- つまり一案目は、直すべき既存失敗（`three-element accepts a flag list but is not advertised`）を**別の失敗へ置き換えるだけ**だった。

採った形：

1. `cli_flag_contract.go`から`three-element`のallowlist項目と空文字guard項目を**削除**した。削除理由をその場にコメントで残す。これで既存の失敗は解消し、`three-element`は`self-build`・`dbcheck`と同じ「dispatch済み・カタログ外」の扱いになる。
2. `helpText`は`three-element`系の名前を`strings.HasPrefix`で受け、`rea-intake`のときは**引数解析と同じ`reaIntakeFlagSet`から**`-h`の一覧を生成する。手書きの二重管理を作らない。
3. `reaIntakeArgs`構造体＋`reaIntakeFlagSet`で文法を1箇所に固定し、`-h`と実解析が同じ定義を読む。

**未解決として残す**：`three-element`を`tei commands`へ載せるか。載せるなら上記2つのカタログ検査の扱いを同時に決める必要がある（`command`比較をやめて`acceptedCommands`を正にする等）。今回は既存挙動を壊さない側を選び、その判断をここに残す。

### 7.2 永続化（実装済み）

`-output`がレビュー票をJSONで書き出す。保存する内容は票そのもので、Source Revision、Candidate Identity、対応根拠、未解決、意味損失、Attempt provenance、`candidateSetFingerprint`、`reviewSequence`がすべて入る。

**落とさない**：`acceptedMeaning=false`と`ownerDecision=pending`。保存は受理ではないので、ファイルを読んだだけで受理済みと解釈できる形にしていない。新しいStoreは作らず、`os.WriteFile`で指定パスへ書くだけにした。

### 7.3 Owner受理への接続（未着手）

この層は受理経路を持たない。接続する場合は既存の受理経路（Shared Acceptance）へ渡すだけで、第二の受理正本を作らない。

- 誰が・いつ・どのCandidateSet fingerprintに対して受理したかを、既存Decisionへ記録する。
- 受理は**候補単位**で行う。Set全体の一括受理を作らない。
- `source-digest-conflict`と`external-identity-renumbered`は受理前に必ず再解析を要求する。

### 7.4 小規模な移行実証

初回実装がCLIから回るようになった後、既存システムの**単一機能**で次を比較する。

| 比較対象 | 何を確かめるか |
| --- | --- |
| REA解析結果 | 対象機能のSourceが実際に観測されていること（partialでないこと） |
| TEI候補 | 候補が対象機能のData/Function/Businessへ過不足なく対応づくか（**対応づかない項目を失敗にしない**。gapとして残す） |
| 受理済みモデル | Ownerが受理した範囲と、候補の`unresolved`が解消していること |
| 生成コード | 受理済みモデルから生成したコードが対象機能の振る舞いを満たすか |
| テスト結果 | 上記が独立したExpectedで確認できるか |

**この実証を成功と呼ばない条件**：REAが対象機能のSourceを部分的にしか観測していない、Owner受理が無い、生成コードの実行結果が無い、のいずれかが欠ける場合は`unknown`または`partial`として残す。

### 7.5 SHOによる設計書生成・旧新対照

**後続の独立した検証**として扱う。この層の出力をSHOの文書正本や旧新対照の根拠にしない。SHO側のSave/reopen、共有Harnessの開示、実Rowsetはそれぞれ別Gateである。

## 8. 既知の欠陥（上流・未修正）

`threeelement.ModelFingerprint`は、受け取った値の要素スライスに対して`AuthorityLimits`を`nil`へ書き換える。Goのスライス複製は同じbacking arrayを共有するため、この書き換えは**呼出側にも見える**。

この層は`safeModelFingerprint`で要素スライスを複製してから呼び、preview要素の`rea-candidate-preview-not-accepted`が消えないようにしている。上流の修正はこのブランチの範囲外であり、#4283のOwnerへ返す。

**後続者が`threeelement.ModelFingerprint`を直接呼ぶと、同じ事故が再発する。**

## 9. 検証したこと・していないこと

**実行した**

- `go build ./...`（このクローン上で成功）
- `go vet ./internal/reacandidate/`／`go vet ./cmd/tei/`：clean
- `go test ./internal/reacandidate/ -count=1`：PASS（51 test）
- `gofmt -l internal/reacandidate cmd/tei`：出力なし
- `go test ./cmd/tei/ -run TestThreeElement -count=1`：PASS（`rea-intake`の正例・負例・`-h`を含む）
- `go test ./cmd/tei/ -run 'TestCommandCatalogFlagsMatchAcceptedFlags|TestAcceptedCommandsMatchDispatch|TestAdvertisedFlagNamesAreTheAcceptedSet|TestSelfParsedFlagNames'`：PASS
- `go test ./cmd/tei/ -count=1 -timeout 3000s`：**PASS（988.7秒）**。パッケージ全体の試験は通る。既定timeoutでは足りないだけで、試験の失敗ではない
- 実バイナリ（`go build -o tei.exe ./cmd/tei`）での実行：最小attempt＋`-output`（exit 0、候補3件、正式な票12,486 bytes）、`-before`比較（全entryが`unchanged`）、実`reference/workspace`への`-workspace`束縛（exit 0、3候補とも`not-present-in-read-artifact-set`で`unresolved`のまま）、`-evidence`不在（`EVIDENCE_READ_FAILED`／exit 2）、`-mapping`単独（`INVALID_ARGUMENT`／exit 2)
- 実Workspace（`tei.workspace.yaml`＋`semantic/*.tei`）に対する束縛の4分岐を `cmd/tei/cli_three_element_rea_workspace_test.go` で確認：byte一致→`independently-bound`かつ`resolution=unresolved`（tag付きのcoincident宣言は「構造的一致であり受理ではない」と注記）、byte不一致→`source-digest-conflict`かつ未解決集計に`source-digest-conflict-requires-fresh-analysis`、path不在→`not-present-in-read-artifact-set`、workspace外を指すpath→decode時に`provider-reported-invalid-source-path`で束縛しない
- Owner mappingを渡した場合の`inspectStatus`が`not-attempted`にならないこと（既存`operations.InspectSubject`が実際に呼ばれること）も同テストで確認
- **実REA 6.8.0の実行**：`rea-agents@6.8.0`を隔離prefix（`npm install --prefix`）へ導入し、合成JavaScriptディレクトリ（`package.json`＋`index.js`）を`analyze-javascript-application --json --full-output`で解析。exit 0、stdout 35,553 bytes（UTF-8）。この実documentをintakeへ通し、`qualification=partial`、候補14件（data 7・cross-reference 7）、`threeElementCoverage`がdata=supported／function=unsupported／business=unsupported、票39,758 bytes
- **実REAの部分case**：構文エラー入りの`index.js`を解析。REAはexit 0だが`statistics.parse_failures=1`、graph coverage `status=partial`、`unknowns`12件、limitation 6件。intakeは`partial`のまま（`complete`へ昇格しない）
- 上記の実測で見つけた誤りを2件修正（§3.5のcoverage形状、§3.6の`artifact-root`別名）

**実行していない（NOT RUN）**

- 実REAの**provider実invocation witness**。REAは起動し実出力を取得したが、`selected→invoked`の対応、provider descriptor `version`が実は`"1"`（`6.8.0`ではない）であることの意味、実行物の署名・digestは未確認。REAが報告する`provider.version`はREAパッケージ版ではなく**Provider実装の版**であり、これが6.8.0と一致しないことは記録しておく
- Owner受理、Shared Acceptance接続、SHO、移行実証、生成コード
- `cmd/tei`全体：`go test ./cmd/tei/ -count=1 -timeout 3000s` が **PASS（672.3秒）**。coverage修正後も通る
- CIの全項目。`bounded-design-context`の1件はPR #4316でpassを確認したが、他のworkflowは未確認

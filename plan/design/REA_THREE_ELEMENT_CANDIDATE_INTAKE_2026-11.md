# REA外部逆解析から三要素分析モデルへの構造候補取り込み（設計候補）

- **状態**：候補抽出・CLI入口・レビュー票の保存・採用前後比較まで。Owner受理・小規模移行実証・SHOは未着手。
- **Owner**：[TEI #3394](https://github.com/hat47x/tei-platform/issues/3394)。
- **関連**：共通API [ADR-0048](../adr/0048-shared-application-api-and-transport-adapters.md)／[#1158](https://github.com/hat47x/tei-platform/issues/1158)／[#3419](https://github.com/hat47x/tei-platform/issues/3419)、Model Map [#3866](https://github.com/hat47x/tei-platform/pull/3866)、三視点 [THREE_ELEMENT_METHOD_DIAGRAM_CONTRACT_2026-10.md](THREE_ELEMENT_METHOD_DIAGRAM_CONTRACT_2026-10.md)、T0/T1-A受入票（未作成。[#3394](https://github.com/hat47x/tei-platform/issues/3394) が担当）。
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


<!-- rea-two-intakes-merged-main-question-review-20261114 -->
## 10. main統合後の再整理：観測と意味候補を二重に実装しない（2026-11-14）

### 10.1 新しい実装基線と残った空白

2026-11-14のmainでは、統合PR [#4683](https://github.com/hat47x/tei-platform/pull/4683)を通じて、[#4625](https://github.com/hat47x/tei-platform/pull/4625)（通常tei CLIの外部候補選択）、[#4630](https://github.com/hat47x/tei-platform/pull/4630)（Host管理Byteを受け取るR2投影）、[#4637](https://github.com/hat47x/tei-platform/pull/4637)（ローカルWorkspaceの固定SelectorからEvidenceを読むHost Reader）がmainへ統合された。旧Draftでこれらを「未統合」と記した説明は、当時の履歴であり、**現在の実装状態を表さない**。

ただし、この事実が示すのは、ローカルCLIとGo Host内のread-only入口が存在することまでである。LocalREAReaderはNative Wails Serviceとしての登録やWindows WebView2 JSからの受領を実証しておらず、外部Web/Agentへの現在Grant・公開資格を与えない。保存されたREAのNodeとRelationから業務Functionや実Effectの意味が正式採用されたわけでもない。現在の製品上の不足は [#4661](https://github.com/hat47x/tei-platform/issues/4661) の**第二Product Consumer実受領**、この文書の**意味候補の根拠付きReview**、既存Shared Acceptanceによる**意味受理とSource反映**、実Provider/旧新Targetによる**作用の確認**である。

### 10.2 二つの既存CLI入口の役割を混ぜない

| 既存入口・コード | 実際の責務 | 組合せる際の制限 |
| --- | --- | --- |
| **tei external-candidates** → operations.ExternalCandidates / ExternalCandidatesFromBytes | 保存済みREAのRawを検証し、指定した外部Observationと有向Relationを**各1件選択**。Evidence ID、実取得Raw digest、申告SourceRevision、Coverage/Unknown/Unselectedを戻す | 選んだ1本のRelationを、全Graph、全Function依存、Sourceの現在性、正式Meaningとしない |
| **tei three-element rea-intake** → reacandidate.DecodeAttemptEvidence / GenerateCandidates / BindCandidates / PreviewAdoption / BuildReviewPacket | 外部の一AttemptからData候補・cross-reference候補を作る。必要ならWorkspaceでSource照合し、read-only Preview・差分・ReviewPacketを生成。Owner受理は常にpending | ReviewPacketは保存できてもCanonicalへの書込みやShared Acceptanceにはならない。PreviewをSource編集許可にしない |
| **LocalREAReader** → Host固定Workspace/Selector・bounded Read→R2のByte投影 | 任意Client Pathを受けず、Hostが選んだ一つのEvidenceをローカルで安全に読む | Wails JSへの実配信、Server未露出、現在の開示資格、他Workspace権限は別の製品受入 |

二入口には**異なる目的と異なる解析単位**がある。R2のExternalCandidateViewは選択済みのObservation 1件とRelation 1件の投影であり、reacandidate.GenerateCandidatesが求める一Attempt全体のGraphではない。これをそのまま正式な一Functionの解析済みGraphとして渡してはならない。逆に、三要素ReviewPacketの候補集合をR2のProducer ReceiptやNative Hostの安全なByte取得の代わりにしてはならない。

両者の最初の実証では、新しい変換器を作るのではなく、**同じ外部解析の保存成果物について、それぞれの入力契約と実消費Byteを別々に確認する**。R2はsemantic_graphから選んだObservationとRelationを返し、rea-intakeはapplication graphの全体から候補を作るため、二つのGraph IDやNode/Relation IDが一致するとは仮定しない。元Rawをそのまま共有できるかは入力形式による。Rawを正規化した場合は変更前後のDigestを分け、Evidence IDが同じでも同一Byte受領・同一Invocationとしない。Producer Tool版、Evidence内Provider版、申告Source revision、Coverageの対象を別々に比較する。一方にない情報は一致ではなくunavailable/unsupportedとして記録する。現在のSourceの再読取も旧解析時の実消費証明にはしない。

### 10.3 関係の識別は、既存の型ごとに保存できる範囲が異なる

現行mainのSourceを静的に照合すると、**reacandidate.StructureCandidate**はCandidateID、ExternalOccurrenceID、ExternalIdentity、From/To、SourceRevision、MeaningLossを別々に持つ。**threeelement.CrossReference**にはIDがあり、採用前PreviewではCandidateIDをそのIDとして使う。これは**Previewの対応を示すための一時的なID**であって、外部IDから正式なBusiness Relation IDを採用済みとする意味ではない。ReviewPacketはAcceptedMeaning=false、OwnerDecision=pendingを維持する。

一方、**modelref.Edge**はFrom/To/Relation/Evidenceを持つが独立したEdge IDを持たず、**operations.ModelMapLink**にも外部Relation occurrenceを置く欄がない。reacandidateのCarrierAuditは、外部Edge ID・Observation ID・一部Source Spanが既存read-only型へそのまま載らない事実を既にFit/Gapとして報告する。新しい「関係ID不足検出器」を作り直す必要はない。

したがって、Source-backedな二つの同端点・同種Relationを保持しているかどうかは、**それぞれの層の受入を分けて**評価する。reacandidateの候補一覧で個別IDが残ること、threeelementのPreviewで候補IDを区別すること、modelrefのEdgeが別図・別編集操作で個別のSource occurrenceへ戻れることは、同じ成功ではない。表示順序・配列添字・同名ラベル・Graph描画Edge IDを正式RelationのIdentityに読み替えない。対応が一意でない個別Relation編集はread-onlyで停止する。

### 10.4 Headlessで先に検証する最小の一Functionの問い

一Functionの問いを**「旧Function Fが読み書きするDataと、そう判断した出典は何か」**に固定する。ただしREA 6.8.0の実JavaScript解析ではDataがsupportedでFunction/Businessがunsupportedと報告された限定ケースがある。そこにTask/Action/Function/IOTableの全連鎖を実観測したと付け足さない。

そこで検証入力を二系統に分ける。

- **実Capturedの系統**：既存REA 6.8.0の保存Rawを使い、R2のObservation/Relation受領と三要素intakeのCandidate/ReviewPacketを**別々に実検証**する。RawとEvidenceの帰属、未知・未選択・partial、同名別ID、再採番・Digest変更を比べる。少数の選択を全件取得へ昇格しない。
- **公開合成の系統**：Task→Action→Function→IOTable→Data、Functionを参照しない手作業、同名Function 2件、並行有向Relation 2本、未取得Source 1件を小さく用意する。既存[#4678](https://github.com/hat47x/tei-platform/pull/4678)のPlugin実証が扱う同名Node n1/n2・平行Edge e1/e2と**最小Graphの対照を共有**してもよい。これは実Capturedの旧システムを観測した証拠ではない。

初回V0-BのHeadless実装PRは、既存Candidate/CarrierAudit/ThreeElementCoverage/ReviewPacketを再利用し、(1)外部Relation IDとSource occurrenceに戻れる箇所、(2)modelref/図面でLossになる箇所、(3)全Sourceで未使用・CUDなしと断定できない理由、(4)現在は編集不可である理由、を**一問のread-only回答**として返せるかを検証する。新しい意味正本・Graph Parser・独立Expected Storeは導入しない。Adapterや出力Schemaの変更が本当に必要になったときだけ、既存#1120/#1124/#2497のOwnerへ限定提案する。

### 10.5 具体的な不足試験と作業順序

| 確認 | 正例・負例 | 最初の完成条件・担当 |
| --- | --- | --- |
| **J-01 二入口の入力適合・受領帰属** | Capturedに基づくBOM付き派生RawはR2のみ受理、BOMなしの基底Rawは両入口で受理、基底RawにBOM付き派生RawのDigestをPinするとR2拒否 | 二つの実受領RawのDigestを区別。semantic_graph選択1本とapplication graph候補群の外部IDを混同せず、共通Evidence IDからRelation同一性を作らない（#3394/#2514） |
| **J-02 Relation identityと意味損失** | 同名異ID、同端点同種の二辺、片方の向きのみ反転、外部ID再採番 | Candidate/Preview/GraphそれぞれのIDを保存できる範囲とLossを記録。編集先が曖昧なら拒否（#1124/#1120/#2497） |
| **J-03 問い相対の不完全性** | Graph completeでもFunction/Business unsupported、未取得Plugin、手作業、動的参照 | Dataの陽性構造だけを回答し、Function全体の未使用・更新なし・削除可能を出さない（#3394/#3262） |
| **J-04 Native Product受領** | 通常CLIの実Resultと、Windows Wails Service→JSの独立実Value/Recipient、Server公開面 | Host内Readerやテスト用子OS Processだけでは合格にしない。公開範囲とCurrent Grantを別試験（#4661/#2516） |
| **J-05 受理・保存・旧新実測** | Ownerが選んだQualified Source Slotの変更Preview、版A→B→A、SHO Save/Reopen、Legacy L/期待D/Target T | 明示受理後のSource保存・再読取と実Provider/Effect/Consumerを別々に確認（#267/#2497/#2586/#1180） |

**実装順**：現行mainの既存契約・固定実例でJ-01/J-02の差を記録→J-03の一Function説明をHeadlessで試す→並行して#4661でJ-04の実Native受領を検収→正式MeaningとSource操作が可能な対象に限りJ-05へ進む。Plugin/DSLのセルフビルドはJ-02で受理済みのGraph/Ruleを使って**実Goロジックと通常CLI呼出しが生成されたか**を別の受入に置く。関係を閲覧できたことも、外部REA候補を受理したことも、Goの既存処理へ委譲しただけの生成物も、TEI-native自己構築の成功としない。

**完了判定と残作業**：J-01〜03はHeadlessの限定受入、J-04は実Native独立Consumer、J-05はOwner受理・Source保存・実Provider/Target/SHOを含む別Owner受入。これらを別PR・別Attemptの成功から寄せ集めずに段階的に繋ぐ。非Plugin Studio DesktopのStage1→Stage2実Windowsセルフビルド（#4583/#2831）、通常tei CLIによる実PG Read→Function Context→独立Consumer（#2586）は引き続き**製品P0**であり、J-04/J-05を新たな着手前提に追加しない。

**この§10の状態**：mainへ統合された実コードと既存設計の**静的照合・実装/Expectedの引渡し**。J-01〜05を新たに実Go/Windows/DB/SHOで通したものでも、正式Meaning/Plugin自己構築のOwner受理でもない。実装済みの旧局所試験と統合後の新しい縦断成功を混同しない。


<!-- rea-j01-dual-graph-bom-contract-20261114 -->
### 10.6 J-01の試験を実入力契約に合わせて修正する：BOM、二種類のGraph、Tool版

**固定Capturedと現行Readerの照合結果・CIによる訂正**：internal/externalanalysis/testdata/rea-6.8.0/analysis.jsonはok/data/metaを持つ保存済みREA出力。ただし**このファイルが常にBOM付きであるという先の記述は誤り**であった。PR exact HEADのRepository Healthログでは、Linux checkoutで読み取ったFixtureにBOMがなく、従来のJ-01テストは当該前提で失敗した。修正版ではFixtureを取得したByteからBOMの有無を除いた基底を用意し、試験用にBOM付き派生Rawを**明示的に構成**する。externalanalysis.ReceiveはBOM付きも読める一方、reacandidate.DecodeAttemptEvidenceは先頭のjson.UnmarshalでBOM付きJSONを拒否する。**この二つは別Raw/Digest**であり、基底のRawを外部Producerの新しい納品として再解釈したり、派生RawのReceiptを元のReceiptと同一視したりしない。Tool VersionとProvider Versionも別軸に保持する。

**二つのGraphは別観測面**である。operations.ExternalCandidates→externalanalysis.ReadCandidatesはnormalized_result.semantic_graphのnodes/relations/evidence_contextsを読む。実Capturedの選択対象はjsrg_node_*とjsrg_relation_*である。三要素intakeのDecodeAttemptEvidence/GenerateCandidatesはnormalized_result.graph（schema=JavaScriptApplicationGraph）のnodes/edges/observationsを読んで候補を作り、こちらはjag_node_*／jag_edge_*である。reacandidateがsemantic_graphから直接取り込むのはCoverage/Unknown等の資格情報であり、R2のsemantic relation IDを正式三要素CrossReferenceへそのまま渡しているわけではない。**同一Evidence IDは両Graph内の個々の関係の同一性証明ではない**。対応するSource位置が偶然一致しても一対一MappingはOwner/契約側で別に確認する。

さらに、CLI引数で指定する**ToolVersion=6.8.0**と、CapturedのEvidence中の**provider.version=1**は異なる軸。数値の不一致を版エラーにしない代わりに、実行した外部ツールのPackage版と、Evidenceが宣言するProvider版を混同して検証済みとしてもいけない。

| J-01の局所ケース | 期待する結果 | 誤って認めない資格 |
| --- | --- | --- |
| **J-01a BOM付き派生Raw** | external-candidatesはsemantic graphの指定IDを取得し、**実際に構成したBOM付きRaw Digest**を返す。three-element rea-intakeは現在のJSON入力契約では入力を拒否 | 入力契約の違いを、グラフ解析失敗や業務Meaning欠落の証拠にしない |
| **J-01b BOMなし基底Raw** | 外部Evidence IDとProvider宣言は保ったまま、R2での**実取得Raw DigestがBOM付き派生Rawとは異なり**、rea-intakeはapplication graph候補とOwner pendingのReviewを返す | 両Readerが同じRawを消費した、またはsemantic_graphとgraphでNode/Relation IDが等しい、とは言わない |
| **J-01c BOM付きDigestでBOMなし版をPin** | R2はstaleを返す。Evidence IDとSourceRevisionが同じでも旧Digestのまま通らない | 「意味上は同じだからByte Pinも有効」としない |
| **J-01d Graphごとの未観測** | semantic_graphのCoverage/Unknownと、application graphの候補/三要素Coverageは別に表示。Function/Businessがunsupportedなら未知を維持 | 局所GraphのcompleteからWriter不在・未使用・削除可能とは結論しない |

**今回の小さい実装**：cmd/tei/cli_rea_dual_entry_contract_test.go にJ-01a〜cの回帰テストを追加した。当初のBOM前提・gofmt差分による[旧HEADのCI失敗](https://github.com/hat47x/tei-platform/actions/runs/43611236185)を確認して修正した。二つの実CLIの**Commandハンドラを同じGo Test内から別々に呼ぶHeadless契約テスト**であり、実行ファイルを二度起動するJ-01の製品受入や、Native Studioという第二Product Consumerには数えない。既存のcli_external_candidates_test.goが担う実Binaryの入口試験とも区別する。J-01dの全問Closure、Graph間のSource-backed対応付け、J-02の並行Edge個別逆参照は後続の独立Expectedで検証する。

**状態**：この節は実ソースと保存済みCapturedを静的に検査した判断および新規回帰テストの説明。新テストの成功はCIのexact HEADまたは手元の同一Toolchain実行で別途確認する。OwnerのMeaning受理、Sourceへの正式書込み、Native Wails JS受領、旧新の実挙動比較は未実施。

<!-- rea-j02-preview-identity-carrier-witness-20261114 -->
### 10.7 J-02の最小実証：関係候補を個別識別し、Source編集資格の不足を示す

J-01で明らかになったsemantic_graphとapplication graphの違いを保つ。**外部解析のRelation、三要素Candidate、Preview上のCrossReference、TEIの正式Meaning Relationは、たとえ同じSourceに由来しても別の資格**である。

現行mainのreacandidate.GenerateCandidatesは、外部Edge IDを含むCandidateIDを作り、同端点・同ラベルの並行Relationを区別できる。PreviewAdoptionはそのCandidateIDを仮のthreeelement.CrossReference.IDとして保持する。一方、CarrierAuditは、ModelMapLink等のread-only型へ外部Relation occurrence IDを移す欄がないことを**external-edge-identity**のGapとして報告する。したがって「Graphの全階層でEdge IDが失われる」わけではない。**Candidate/Previewでは各辺を保持できるが、正式Meaningや編集対象へ安全に逆写像する資格は別**である。

[新規Go回帰テスト](../../cmd/tei/cli_rea_parallel_relation_review_test.go)では公開合成reaIntakeFixtureに同端点・同ラベルのEdge二本を置き、既存three-element rea-intakeのCommand handler経由で次を確認する。

| ケース | 正例・負例の期待 | 成立しない資格 |
| --- | --- | --- |
| **J-02a 二つのOccurrence** | 外部Edge ID、CandidateID、SourceRevisionを各々保持。parallel-relation/same-endpoints-same-labelを報告 | 正式業務Relationとしての採用 |
| **J-02b ReviewとPreview** | 二つのCandidateIDが仮CrossReference IDとして残り、AcceptedMeaning=false/Owner pending | Source編集や業務意味受理 |
| **J-02c Carrier Fit/Gap** | 外部Edge ID二件が下流型に直接移送できないGapとして表示 | modelref/ModelMap/図面/SHOでの完全な往復 |
| **J-02d Producer ID重複** | 同じEdge IDを二本宣言したらEDGE_IDENTITY_DUPLICATEで拒否 | 配列順やラベルから一方を暗黙選択・削除 |

本テストは既存Unitの同名Node・平行Edge・Previewを、CLIのReviewPacketまで繋ぐ**局所Go試験**。本物のtei実Binary二プロセス、Native Wails/JS第二Product Consumer、Source書換えや実DB Effectの合格ではない。新しいReader/Parser/Meaning Store/Approval Engineは作らない。

**別レーンとの接続**：main統合済み[実行モデル§37〜39／PR #4703](PROGRAM_EXECUTION_MODEL_PROJECTIONS_2026-11.md)の同名Node・同向多重Edge・方向反転・Layout-onlyの例を、合成Graphの比較軸として共有する。ただし#4703は設計票で、実Go Generatorの成功ではない。REA CandidateのReadとOwner受理済みTEI Graph/RuleからのCodegen、実Provider Call、Consumer実受領を別々に判定する。Go既存関数への単純delegateはTEI-native実装に数えない。

**残るJ-02e/f**：Source-backedな異Graph間Relation対応を一意に立証できるか、e2だけ反転・e1だけ削除した場合にSource版・旧Pin・後着Previewを拒否できるか。正式Source編集の一意性・Owner Meaning・Current Grantは#1120/#1124/#2497/#2514の既存Ownerへ返し、資格がなければread-onlyで停止させる。Graph completeでもFunction/Business unsupportedなら業務全体の否定・未使用判断をしない。

**検証資格**：この節の反映とGoテストSourceの作成は、当該新テストの実Go/CI成功を意味しない。exact HEADでの実行合否は別途確認する。Studio Stage1→Stage2と通常tei実PG Readという製品P0は維持する。

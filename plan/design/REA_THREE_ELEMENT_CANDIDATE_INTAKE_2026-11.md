# REA外部逆解析から三要素分析モデルへの構造候補取り込み（設計候補）

- **状態**：候補抽出とレビュー票の実装まで。Owner受理・CLI入口・移行実証は未着手。
- **Owner**：[TEI #3394](https://github.com/hat47x/tei-platform/issues/3394)。
- **関連**：共通API [ADR-0048](../adr/0048-shared-application-api-and-transport-adapters.md)／[#1158](https://github.com/hat47x/tei-platform/issues/1158)／[#3419](https://github.com/hat47x/tei-platform/issues/3419)、Model Map [#3866](https://github.com/hat47x/tei-platform/pull/3866)、三視点 [THREE_ELEMENT_METHOD_DIAGRAM_CONTRACT_2026-10.md](THREE_ELEMENT_METHOD_DIAGRAM_CONTRACT_2026-10.md)、T0/T1-A受入票 [REA_EXTERNAL_EVIDENCE_T0_T1_ADMISSION_2026-11.md](REA_EXTERNAL_EVIDENCE_T0_T1_ADMISSION_2026-11.md)。
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
- Set全体の`qualification`：`complete`／`partial`／`unknown`。REAの`coverage.exhausted`が`false`・未申告、`limitations`または`unknowns`が非空、Sourceが`provider-reported`でない、端点が未観測、のいずれかで`partial`以下へ落とす。

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

## 7. 未着手の作業（後続実装者向け）

以下は**未実装**である。着手順に並べる。

### 7.1 CLI入口（次の最小作業）

`tei three-element rea-intake`を追加する。変更点は次の4ファイルに閉じる。

1. `cmd/tei/cli_three_element.go`
   - `runThreeElementCommand`の`switch`へ`case "rea-intake":`を追加し、`runThreeElementReaIntake`を呼ぶ。
   - 先頭の`failThreeElement`メッセージのsubcommand一覧へ`rea-intake`を足す。
   - フラグ案：`-evidence`（必須、REAのJSON）、`-workspace`（任意、束縛する場合）、`-model`（任意、採用previewの対象）、`-mapping`（任意、`{candidateId: {kind,id}}`のJSON）、`-before`（任意、前回のCandidateSet JSON）、`-output`（任意、レビュー票の保存先）。
   - 出力は`threeElementResponse`で包み、`schemaVersion`は`tei.cli.three-element.rea-intake/v0`。
2. `cmd/tei/cli_flag_contract.go`
   - `selectedReadOnlyFlagAllowlist["three-element"]`を`{"model"}`から上記フラグの集合へ広げる。
   - `emptyStringGuardedFlags["three-element"]`へ、閉じた値／完全一致が必要なフラグ（`-evidence`・`-workspace`・`-model`・`-mapping`・`-before`）を追加する。
3. `cmd/tei/cli_contract.go`相当のカタログ
   - 現状`three-element`は`commandCatalog()`に登録されていない。CLIとして広告するなら、`cli_commands_test.go`の「dispatched だが advertised でない」検査に合わせてレコードを追加する。**広告しない判断をするなら、その理由をPR本文へ残す**（既存の`self-build`が同種の前例）。
4. `cmd/tei/cli_three_element_test.go`
   - 正例：fixtureのattemptでexit 0、`acceptedMeaning=false`。
   - 負例：`-evidence`欠落で`INVALID_ARGUMENT`／exit 2、decode失敗で`EVIDENCE_*`のcode。

**注意**：`cmd/tei/main.go`の`command == "three-element"`分岐は共有FlagSetより前にあるため、`validateSelectedReadOnlyFlags`は通らない。フラグの整合はサブコマンド側の`FlagSet`と、上記allowlist／カタログの**表を一致させること**で保つ。既存の表一致テスト（`cli_flag_names_test.go`）がどの範囲を見ているかを先に確認する。

### 7.2 永続化

レビュー票を「後から読める」形で残す。現状は`-output`も無い。

- 保存先は既存の`successful`出力先の慣行に合わせる。新しいStoreは作らない。
- 保存する項目：Source Revision、Candidate Identity、対応根拠、未解決、意味損失、Attempt provenance、`candidateSetFingerprint`、`reviewSequence`。
- **落とさない**：`acceptedMeaning=false`と`ownerDecision=pending`。保存しただけで受理済みに見える形にしない。

### 7.3 Owner受理への接続

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
- `go vet ./internal/reacandidate/`：clean
- `go test ./internal/reacandidate/ -count=1`：PASS
- `gofmt -l internal/reacandidate`：出力なし

**実行していない（NOT RUN）**

- `cmd/tei`のCLI入口（未実装）
- 実REAプロセスを起動したE2E。fixtureは**手書きの合成JSON**であり、REA 6.8.0の実出力を取得した証拠ではない
- 実Workspaceへの束縛（`WorkspaceReadOnlyModel`は単体試験でfake portのみ。`reference/workspace`に対する実行は未実施）
- Owner受理、SHO、移行実証、生成コード
- CI（このクローンからは未実行。PRのexact headで確認する）

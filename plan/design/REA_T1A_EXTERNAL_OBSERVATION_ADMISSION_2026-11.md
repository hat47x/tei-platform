# T1-A 外部Observation/Relation受領の作業票（Fit/Gapと実装手順）

対象Issue: [#3394](https://github.com/hat47x/tei-platform/issues/3394)
前提: [REA_EXTERNAL_EVIDENCE_INTAKE_2026-11.md](REA_EXTERNAL_EVIDENCE_INTAKE_2026-11.md)（受領側T0、実装済み）
作成日: 2026-11-11
状態: **提案の具体化。T1-A本体は未実装（NOT RUN）。**

## 1. この文書の位置づけ

Issue #3394の「次の唯一のProduct Exit」は、**同一REA実Attemptの外部Observation 1件＋方向とEdge IDを持つRelation 1件**を、Source byte/位置・外部ID・Relation ID・unknown/partial/unbound・現在Disclosure・selected→invoked→Receiver実受領を落とさず、**共通read-only Application API**（#1158/#3419、判断の正本は[ADR-0048](../adr/0048-shared-application-api-and-transport-adapters.md)）の実Consumerが受領できるか、である。

この文書はそのExitの**作業票**である。何が成立していて何が欠けているかを型レベルで確定し、後続の実装者が判断を繰り返さずに最後まで作業できるようにする。型が欠ける場合は既存Owner（#1158/#3419/#1251）へ不足を返し、**未採用Schemaを実装したと偽らない**。

## 2. すでに成立していること（このPRまで）

`internal/externalanalysis` が、受領・配送検査に加えて**候補の読み出し**を実装し、実捕捉bytesで機械検証済み:

- `Receive` — 受領bytesのdigest・包装とEvidence本体の分離・状態分類。
- `WithView` — 版別JSONL framing照合（終端有無・counts・親リンク・coverage・重複identity）。
- `ReadCandidates` — Evidence本体のsemantic graphから、**Observations（7件）・Relations（2件）・Unknowns（1件）・Coverage（family別）・Limitations** を読む。`node_id`/`relation_id`/`unknown_id`・artifact sha256・line/column span・evidence context（state/confidence/authority/extractor/limitations）・resolutionを保持する。Producerがrelation_idを言う版ではそれを読み、言わない場合だけTEI導出のhandleを使い`derived`を付ける。
- 試験: `candidate_test.go`・`external_candidates_test.go` が実捕捉（REA 6.8.0）に対して上記を1フィールドずつ固定する。
- `operations.ExternalCandidates` — 上記を共通read-only操作として投影する型付き操作（§5のD1決定にもとづく）。`tei rea-evidence -observation-id … -relation-id …` がそのCLI Adapter。

したがってT1-Aの**入力側・投影側**は出ている。§5の代行決定で「どの操作が運べるか」に答えた。

## 3. Product Exitの受入票（同一Attemptの定義）

「受領できた」と主張するには、以下が**すべて**成立すること:

| # | 条件 | 機械的な確認方法 |
|---|---|---|
| A1 | 同一Attempt: 受領したEvidence ID・親digest・view digest・包装のcommand/durationが、Consumerに渡した記録と一致する | Receiptの`evidence.id`・`raw.fingerprint`・`view.evidenceId`・`envelope`をConsumer応答と突き合わせる |
| A2 | 外部Observation 1件: 外部Node ID・Kind・artifact digest・位置・span・evidence context（state/confidence/authority/extractor）・limitationsが応答に存在する | 各欄が空でなく、値が捕捉bytesと一致する |
| A3 | 方向付きRelation 1件: 外部Relation ID（Producer記載または派生であることが判別できる）・from/to・kind・resolution・位置が応答に存在する | 同上。`derived`の区別が載る |
| A4 | unknown/partial/unbound: Coverage（status・families・unknown_ids）とUnknowns（unknown_id・node_id・family・reason・candidate_node_ids）が応答に存在する | 同上 |
| A5 | 非昇格: 応答が「受理済み意味ではない」「Canonical Worldではない」「実DB観測ではない」ことを表明する | 応答のauthorityLimits（または同等の表明欄）にその声明があり、かつ意味欄が空である |
| A6 | 実受領: 合成ではなく、実REAが書き出したbytes（またはそのbyte同一の保存fixture）を使う | 使用bytesのSHA-256を記録する |

A1〜A6のうち一つでも満たさなければ「連携成功」と呼ばない。A5を満たさない実装は、たとえA2〜A4を通っても**失敗**として扱う（意味の捏造だから）。

## 4. Fit/Gap（型レベル）

### 4.1 既存の読み取り専用操作の入口

| 操作 | 入口 | 起点 |
|---|---|---|
| `operations.InspectSubject(root string, subject sourceindex.Subject)` | Workspace root＋**TEI subject** | 正規化されたWorld。Source参照だけではcanonical声明にならない |
| `operations.MapWorkspace(root)` / `MapWorld(world, catalog)` | Workspace root、またはWorld＋catalog | 宣言されたIR参照のみ |
| `internal/modelref` のtrace投影 | World | 同上 |
| `internal/database/externalqueryevidence` | 外部観測DBのEvidence | **唯一の「外部Evidenceを非canonicalとして受ける」既存pattern**。ただしContract QueryのSQL realizationとの照合が目的で、REAの構造観測は対象外 |

### 4.2 欄ごとの判定

「実データ」はREA 6.8.0の実捕捉（`internal/externalanalysis/testdata/rea-6.8.0/`）。

| 必要な欄 | 実データ | 既存の型の置き場 | 判定 |
|---|---|---|---|
| 外部Evidence ID | `evidence_id` | read-only操作の結果に外部IDの欄なし | **GAP** |
| 外部Node ID | `node_id`（`jsrg_node_…`） | `sourceindex.Subject{Kind,ID}` はTEI语义subjectの名前空間。外部IDを入れると名前空間を同一視する | **GAP** |
| 外部Relation ID | `relation_id`（`jsrg_relation_…`） | 同上 | **GAP** |
| 方向 | `source_node_id` → `target_node_id` | `ModelMapLink.From/To` は方向を表現できる（ただしsubject型） | 方向自体はFIT |
| Source位置 | `artifact.sha256` + `path` + `location.value`（kind/source/start/end） | `sourceindex.Reference{ArtifactPath, Range, Role}` にpathとrangeはある。`Range`は**byte span**でcontent照合付き、外部のline/column claimは別物 | 部分（byte/lineの混同防止が必要） |
| Source bytes digest | `artifact_sha256` | `sourceindex.Artifact.Fingerprint` はworkspace pathに紐づく | 部分 |
| partial/unknown/unbound | `coverage{status,truncated,omitted_*,limits,families[{status,unknown_ids}]}`、`unknowns[{unknown_id,node_id,family,relation_kinds,reason,detail,candidate_node_ids}]` | `ModelMap.Coverage` は定数文字列 `"partial-projection"`、`UnqualifiedRoles` は定数リスト。未知の**一覧**を持てない | **GAP** |
| Provider/版/Attempt | `provider{id,name,version}`、`extractor{name,version,operation}`、包装`meta{command,duration}`、`tool{present,resolvedPath}`、`acquiredAt` | read-only操作の結果にProducer/Attemptの記載なし | **GAP** |
| Disclosure（現在） | 別Owner（Authorization）。受領側はbytesの開示資格を主張しない | `AuthorityLimits` に「local readはdisclosure許可ではない」の同種声明あり | 表明はFIT、判定は別Owner |
| 意味の非昇格 | `promotion`全false | `AuthorityLimits` のpattern（「Source locationは受理ではない」等）が同種 | FIT（patternを踏襲できる） |
| 入力経路 | 受領bytes／そのdigest | 既存操作はWorld起点。外部bytesを入れる入口がない | **GAP** |

### 4.3 不足辺の集約（Ownerへ返す内容）

不足は4種類に集約できる。**新しいHost/Registry/Reader/Runnerや第二意味正本は作らない**という前提で、これらを既存Ownerへ返す。

| ID | 不足 | 返却先 | 期待する判断 |
|---|---|---|---|
| G1 | **外部ID名前空間**: 外部Evidence/Node/Relation IDを、TEI subjectと別の名前空間として結果に載せる欄 | #1158/#3419 | 外部ID用の欄を既存操作に足すか、外部候補専用のread-only操作を別Issueで起こすか |
| G2 | **未知の一覧**: coverage/unknownsを「定数文字列」ではなくProducer宣言の構造として載せる欄 | #1158/#3419 | unknown_id・node_id・family・reason・candidate_node_idsを運べる型 |
| G3 | **Producer/Attempt**: 結果に「どのProducerのどの版が、どのAttemptで、いつ受領したか」を載せる欄 | #1158/#3419/#1251 | provider/extractor/duration/acquiredAt/tool presence |
| G4 | **入力経路**: 外部Evidence（bytesまたはdigest）を型付きで操作へ渡す経路 | #1158/#1251 | 生bytesを渡さない（digest参照か、明示typed input）。新Runnerは作らない |

G1〜G4がいずれも「既存操作で表現不能」とOwnerが判断した場合、**このIssueではT1-AをNOT RUNのまま維持**する。実装者が勝手にSchemaを作ってはならない。

## 5. オーナー判断（2026-11-11 代行決定）

Issue起票者から「オーナー判断は代行して構わない」の授権を受けたため、以下を決定する。判断の正本はこの節と、`internal/operations/external_candidates.go` の実装である。
### D1 — G1〜G4は「既存操作では表現不能」。最小差分は**新規の型付きread-only操作1つ**

既存のcatalog/inspect/trace/validate/model-mapはすべてWorld起点で、（a）外部ID名前空間の欄がなく、（b）coverage/unknownsを「定数文字列」としてしか持たず、（c）Producer/Attemptの欄がなく、（d）外部Evidenceを入れる入力経路がない。**既存操作にoptional欄を足す案は採らない**——外部IDをTEI subjectと同じ結果型に混ぜると、呼出し元が外部IDをsubjectと誤読できる（T1A-07の事故経路になる）。

代わりに `internal/operations` に**型付きread-only操作を1つだけ**追加する:

- `operations.ExternalCandidates(request, at) (ExternalCandidateView, error)` — schemaVersion `tei.operations.external-candidates/v0`・`effect: read-only`・`authorityLimits`（固定リスト）という、既存操作と同じ形。
- 入力は**型付きrequest**（EvidencePath・宣言したToolVersion・選択するObservationID/RelationID・entryが観測したTool）。**生bytesは渡さない**。操作はファイルを1回読んで、読んだbytesのdigestをAttemptに載せる。
- 内部では `internal/externalanalysis` の `Receive`/`ReadCandidates` を再利用する。**Evidence形式の読み手は一つだけ**（第二Parserを作らない）。
- 結果は外部ID・Source位置（digest＋kind:path＋line/column span）・evidence context（state/confidence/authority/extractor）・coverage（family別）・unknowns（object配列）・limitations・Producer/Attempt・**未選択件数**を運ぶ。
- 失敗は型付きerrorで、`INVALID_REQUEST`（引数）・`READ_FAILED`・`UNUSABLE`（mismatch/missing）・`INCOMPLETE`（partial）・`FAILED`（工具の失敗）・`NOT_FOUND`（選択IDがEvidenceに無い）を区別する。

### D2 — 入口はAdapter。CLI入口は既存の `tei rea-evidence` を延長する

`tei rea-evidence -observation-id … -relation-id … -provider-version …` がCandidate投影のmodeになる（`mode=candidate`）。HTTP/MCP/Studio入口は**このPRでは作らない**（未実装の入口を成功としない）。MCP read-only Adapterへの追加は別Issueとする。

### D3 — 工具の存在観測はentryの責務

`ExternalCandidates` は工具をprobeしない。presenceはprocess境界で観測するentry（CLI）がrequestで渡し、操作はそれを**記録するだけ**（権限ではない）。工具不在の Attempt は「この環境では新規取得できない」ことを示す。

### D4 — 覆す条件

- 既存操作に外部ID欄を足す方が望ましいとOwnerが判断した場合、この操作は撤回する（畳む先は既存操作）。
- MCP/HTTP入口を作る段階で、同じ結果型を運べるAdapterが必要になった場合、**結果型は再利用する**（第二の型を作らない）。

## 6. 後続実装者の作業手順

### Step 1 — 不足辺を返す（実装しない）

1. §4.2の表と§4.3のG1〜G4を、#1158/#3419/#1251へコメントで返す。
2. Ownerから「既存操作で表現できる」か「型を足す」かの回答を得る。回答が無い間はStep 2へ進まない。

### Step 2 — 縦切りの定義（1 Observation + 1 Relation）

Ownerが「型を足す」と回答した場合のみ、次の縦切りを実装する:

- **入力**: `externalanalysis.Receipt`（受領記録）と `externalanalysis.Candidates`（候補読み出し済み）。**生bytesを操作へ渡さない**。渡すのはdigestと、読み出し済みの候補（§2）。
- **出力**: read-only consumerが受領する1 Observation＋1 Relation。§3のA1〜A6を満たすこと。
- **やること**: 既存の共通read-only操作（ADR-0048のcatalog/inspect/trace/validateのどれか、またはOwnerが指定する操作）に、外部候補を運ぶための**最小の型差分**を置く。差分は「外部ID欄」「未知の一覧欄」「Producer/Attempt欄」の3つに限定する。
- **禁止**: 外部Node IDを`sourceindex.Subject`に押し込めない（G1）。`coverage`を定数文字列に丸めない（G2）。Producer版を省略しない（G3）。selected→invoked→receivedを1語に畳まない。

### Step 3 — 試験（直接呼出しfixture）

ADR-0048の試験表（T01〜T13）の様式に合わせ、**直接呼出し**で確認する:

| ID | 反例 | 独立期待結果 |
|---|---|---|
| T1A-01 | 正常系: 実捕捉bytesから1 Observation＋1 Relation | A2・A3を満たし、各欄の値が捕捉bytesと一致 |
| T1A-02 | Relation IDがProducer記載（`jsrg_relation_…`） | そのIDが載る。`derived`はfalse |
| T1A-03 | Relation IDがProducer未記載（テストが落とした派生コピー） | TEI導出handleが載り、`derived`がtrue。ProducerのIDと誤提示しない |
| T1A-04 | coverage.status=unknown、familiesにpartial | 応答がunknown/partialを載せる。完全扱いしない |
| T1A-05 | unknownsに1件（unknown_id・node_id・family・reason・candidate_node_ids） | その一覧が載る。unknownをlimitationsに畳まない |
| T1A-06 | 位置が`available:false` | 「位置は観測できなかった」と判別できる。空文字列を「位置なし」と読ませない |
| T1A-07 | 外部Node IDと同名のTEI subjectがWorldに存在する | 同一視しない。外部IDは外部IDとして載る |
| T1A-08 | 同一Evidence IDで対象digestが異なるbytes（Source改版） | 別の観測として扱う。pin照合がrevision差を名指す |
| T1A-09 | 工具が環境に無い | 「この環境では新規取得できない」と載る。保存fixtureの検査結果であることが判別できる |
| T1A-10 | 応答を「受理済み意味」として読める形にする | 禁止。authorityLimits（または同等）に非昇格の声明があり、意味欄が空 |
| T1A-11 | 包装`{ok,data,meta}`をEvidence本体として渡す | 拒否（`externalanalysis.DeriveViewRequest`が構造的に拒否する） |
| T1A-12 | 版が宣言されていないview配送 | 拒否またはpartial。framingを推測しない |

### Step 4 — 検証（実行できる環境とできない環境を分ける）

- **実REAを呼べる環境**: `plan/fixtures/rea-external-evidence/verify_real_rea_intake.py --binary <tei> --version <v>` を実行する。工具をisolated prefixへinstallし、合成inert fixtureを解析・view投影・負例実走し、fresh bytesで受領・検査・投影を検証する。実行記録（digest・exit・notRun）は`--report`のJSONとCI artifact。
- **実REAを呼べない環境**: 同scriptに`--fixtures-only`を付ける。`internal/externalanalysis/testdata/` の捕捉bytesで同じ契約を検証し、live partsは`notRun`として記録する。fixtureを「実REAの出力」と表示してよいが、「この環境で取得した」とは表示しない。
- **CI**: `.github/workflows/rea-external-evidence-intake.yml` が2lane（Go契約は全環境・実工具はpin版matrix）で実行し、実工具laneの実行記録をartifactとして保存する。
- どちらの環境でも、**実行した検証と未実行の検証**をPR本文に分けて書く。

### Step 4-b — 版の再資格化（T0-06相当）

新しいREA版が現れたら、`observedSourceKinds`/`observedFraming`は**実走で**更新する。手順は [plan/fixtures/rea-external-evidence/README.md](../fixtures/rea-external-evidence/README.md) の「版の再資格化」に6ステップで固定した。未知の版は推測せず拒否される。

### Step 5 — 記録

- 結果をIssue #3394へ追記する。PASS/FAILだけでなく、A1〜A6のどの条件を満たしたかまで書く。
- `docs/status.md` の受領行を更新する（T1-Aが受領可能になった事実を、未実施の項目と区別して書く）。

## 7. やらないこと（停止線）

- REA Parser・中央Registry・専用Host・第二の意味正本を作らない。
- 外部Node/Relation IDをTEI subjectと同一視しない（T1A-07）。
- `operations.InspectSubject` に外部JSONを渡して意味受理と偽装しない。`deployment/observation` を実Runtime観測として偽装しない。`deployment/evidence.Router` のEvidence Gate `pass`として偽装しない。
- SHO Writer・三視点CRUD・実DB/旧新対照・Owner Meaning受理は別資格。本作業票はT1-Aの受領まで。
- **MCP/HTTP/Studio入口は作らない**。未実装の入口を成功としない。共通結果型を運ぶAdapterが必要になったら、結果型を再利用して第二の型を作らない。

## 9. 検証体制（実REAがある環境とない環境）

| 層 | 何をする | どこで |
|---|---|---|
| Go契約試験 | 受領・検査・候補読み出し・投影・負例を捕捉bytesで固定 | `internal/externalanalysis`・`internal/operations`・`cmd/tei` の試験 |
| real-CLI verifier | 実REAで解析・view・負例を実走し、fresh bytesで同じ契約を検証。工具が無ければ保存fixtureで検証しlive partsはNOT RUN | `plan/fixtures/rea-external-evidence/verify_real_rea_intake.py` |
| CI | Go契約lane（全環境）＋実工具lane（pin版matrix、実行記録をartifact） | `.github/workflows/rea-external-evidence-intake.yml` |

工具が環境に無いことによる`tool-absent`は**whole receiptの注記**であり、失敗ではない。保存fixtureのみの環境では「この環境で取得した」とは表示しない。

## 8. 次の不足辺（T1-A着手後に顕在化するもの）

- 第二Source（異種二Source以上）の同一Function候補への帰属。観測時点・Schema alias・Business IDの衝突（RE-18）。
- Source closure（依存の完全な受領）。REAが「全件取得」を認めない条件（§Issue本文のMCP契約）。
- Disclosure/Grantの現在性。受領したbytesを他へ渡してよいかの判定はAuthorizationのOwner。
- 版更新（T0-06）: 6.7.0/6.8.0以外の版では受諾値・framingが未知なので、実走して表を更新してから資格化する。

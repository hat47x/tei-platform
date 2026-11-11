# SUI→TEI移行 実装パケット（M1〜M6）

- 状態: 実装前の提案（未採用）。実装・受入・契約変更の根拠にしない
- 日付: 2026-11-11
- 対象: [SUI→TEI移行の設計](https://github.com/hat47x/tei-platform/pull/4298)で整理したM1〜M6を、後続の実装者がPR単位で迷わず進められる形に具体化する
- 正本との関係: 設計の理由と対応表はPR #4298の設計文書が扱う。本書は、変更範囲・型・試験・受入・Owner判断・完了条件だけを扱い、設計本文を複製しない
- 根拠: SUI #3401、SUI PR #3428（統合済み）、SUI PR #3429（提案中）、SUI ADR-0116（Proposed）、SUI `02_Architecture/api.md` §2.18、TEI #2497
- 主張しないこと: M1〜M6の実装完了、TEI Core・Plugin契約・`CapabilityKind`の変更、Owner意味受理、実Provider・実配備の検証を主張しない

## 1. 現在地（2026-11-11）

| 要素 | 状態 | 根拠 | 次の作業 |
| --- | --- | --- | --- |
| SUIの型付きread-only参照 | 統合済み | SUI PR #3428、merge `eafd4d36` | 変更しない |
| Source状態・文書単位の開示制限・多重Edgeの`parallel` | 提案中 | SUI PR #3429（CI green、未統合） | M1としてレビュー・統合判断 |
| TEI側のM0〜M6設計 | 提案中 | TEI PR #4298 | 本書と同時にレビュー |
| TEI側の実Consumer | 未実装 | — | P7（任意） |
| Owner判断8点 | 未決 | 設計 §7、本書 §7 | 各PRで判断を記録 |

M0は「SUI側のconsumerが動く」ところまでで、TEI側のconsumer・producer契約・Canonical定義は未完了である。M0を理由にM1以降の受入を省略しない。

## 2. 不変条件（全Work Packet共通）

以下は、実装時に緩めない。レビューでは、変更がこのどれかを弱めていないかを確認する。

| ID | 不変条件 | 負例 |
| --- | --- | --- |
| I-01 | 同一性はoccurrence IDで決める。name・text・見出しで畳まない | 同名Cardを1件にdedupeする |
| I-02 | containmentとaffiliationを別Relationにする | 両方を`belongsTo`へ統合する |
| I-03 | Relationは向きとoccurrence IDを保持する。並行Edgeを端点の組で畳まない | `from/to`だけで多重Edgeを1件にする |
| I-04 | 未レビューは「不存在」ではない。参照を残し、内容だけを伏せる | 未レビューCardを一覧から消す |
| I-05 | 開示制限は粒度を分ける。entity省略と文書全体の拒否を同じ扱いにしない | 文書制限時に「対象0件」と返す |
| I-06 | 内容digestとSource revisionを同一視しない | digest一致を「最新」と表示する |
| I-07 | 宣言の無いcompletenessは`unknown`。`complete`へ読み替えない | artifactが短いだけで完全とみなす |
| I-08 | 損失は`unsupported`/`partial`/`unresolved`で宣言する。欠落を推測で補う | Treeをisland parentから再構成する |
| I-09 | 読めることはAccepted Meaningではない。参照は`acceptedMeaning=false`を維持する | Card参照をTEIの受理済み意味として扱う |
| I-10 | 第二のデータ正本・意味Registry・書込み経路を作らない | artifactを保存・編集する経路を追加する |

## 3. 実装順

```text
M0（統合済み: SUI #3428）
  └─ M0b（SUI #3429: source status / document disclosure / parallel）
       ├─ M1（producer契約の確定）
       │    └─ M5（Source状態と版の権威）
       └─ M2（Canonical定義）
            ├─ M3（record投影）
            └─ M4（開示投影）
M6（scope conformance）は独立。M3・M4の後にまとめて検証してもよい。
P7（TEI側の参照consumer）はM1と並行可能。共通APIの新Operationは作らない。
```

各Work Packetは1 PRを目安にする。1 PRに複数Packetを混ぜる場合は、PR本文で「どのPacketのどの受入を満たすか」を分けて書く。

## 4. Work Packet

### M1 producer契約（`source.status`と文書単位の開示）

**目的**: consumerがartifactの欠落から推定しているSource状態と文書開示を、producerの宣言として読めるようにする。宣言が無い場合は安全側の`unknown`で動く。

**候補とする入力（すべてoptional）**:

```json
{
  "source": {
    "status": "active | revoked | expired | unknown",
    "statusReason": "opaque code",
    "documentRevision": "opaque revision",
    "observedAt": "RFC3339"
  },
  "document": {
    "disclosure": {
      "scope": "public | internal | restricted | unknown",
      "policyRef": "opaque policy reference"
    }
  }
}
```

**既定値と優先順位**:

1. `source.status`が無い、または未知の値: `unknown`。`active`へ読み替えない。
2. `source.status=revoked|expired`: 参照を返さず`409`。ID・件数・種別を返さない。キャッシュ済み参照へfallbackしない。
3. `document.disclosure.scope=restricted`: 参照を返さず`403`。entity単位の省略より先に評価する。
4. `document.disclosure.scope=public|internal`: その範囲でentity単位の制限を適用する。`unknown`は本文を返さない。
5. entityの`visibility`/`private`/`restricted`: そのentityだけを除き、ID・件数を返さない。
6. `documentRevision`はSource revisionであり、`documentDigest`（内容指紋）と別欄にする。
7. `observedAt`は観測時刻であり、Sourceの有効時刻・更新時刻ではない。

**変更範囲**:

- SUI: `tei_sui_reference.py`、`routes/tei_epistemic.py`、`api.md` §2.18、`ADR-0116`（Proposedのまま）、関連tests。
- producer: 別リポジトリ。TEI側は契約文書だけを更新し、producer実装をTEIへ持ち込まない。

**試験（最低）**:

| ID | 入力 | 期待 |
| --- | --- | --- |
| M1-T01 | `status`欠落 | `source.status=unknown`、lossに`source-status: unresolved`、内容なし |
| M1-T02 | `active` | 参照を返す。`unknown`と区別 |
| M1-T03 | `revoked` | 409、ID・件数なし、キャッシュfallbackなし |
| M1-T04 | `expired` | 409、ID・件数なし |
| M1-T05 | `status="future-value"` | `unknown`として返し、lossに未知のstatusを残す。`active`にしない |
| M1-T06 | `document.disclosure.scope=restricted` | 403、種別・ID・件数なし |
| M1-T07 | `scope=unknown` | 参照は返すが本文なし、lossに開示不明 |
| M1-T08 | entity private + 文書internal | private entityだけ除く。他のentityは参照 |
| M1-T09 | `documentRevision`変更・digest同一 | `source-revision-changed`。内容変更と区別 |

**受入コマンド**:

```bash
cd 03_Implement/backend
python -m pytest tests/test_tei_sui_reference.py tests/test_tei_sui_reference_route.py -q
python -m pytest tests/test_tenant_session_precondition.py tests/test_agent_credential_http.py -q
python -m ruff check src/sui_sensemaking_api/tei_sui_reference.py \
  src/sui_sensemaking_api/routes/tei_epistemic.py \
  tests/test_tei_sui_reference.py tests/test_tei_sui_reference_route.py
```

**Owner gate**: `source.status`と文書開示をproducer契約に固定するか。未決の間は`unknown`既定で実装してよい。契約を固定するPRでは、Ownerの判断をPR本文へ引用する。

**Rollback**: 追加フィールドを無視する旧consumerでも動くため、producerの宣言を止めれば`unknown`へ戻る。APIの経路自体を閉じる場合は、routeの登録を外し、ADR-0116をSupersededにする。

### M2 Canonical定義（Reference Workspace）

**目的**: `sui.document` / `sui.card` / `sui.island` とRelationを`.tei`で宣言し、instanceを昇格させずに型の解決を検証する。

**候補配置**（新規Reference。Product runtimeではない）:

```text
reference/sui-reference/workspace/
  tei.workspace.yaml
  semantic/sui/document.tei
  semantic/sui/card.tei
  semantic/sui/island.tei
  semantic/sui/relation.tei
```

**候補宣言**（構文は`validate`で確定する。以下は意味の候補であり、コピーして確定版にしない）:

```tei
namespace sui

type DocumentId from std.identifier
type CardId from std.identifier
type IslandId from std.identifier
type RelationId from std.identifier

data Document {
    attribute documentId type sui.document_id
    attribute revisionState type std.code
    attribute disclosureState type std.code
}

data Card {
    attribute cardId type sui.card_id
    attribute claimType type std.code
    attribute holdState type std.code
    attribute x type std.number
    attribute y type std.number
    attribute textReviewState type std.code
}

data Island {
    attribute islandId type sui.island_id
    attribute titleReviewState type std.code
}

data Relation {
    attribute relationId type sui.relation_id
    attribute relationType type std.text
    relation source to sui.card many
    relation target to sui.card many
}
```

**要件**:

- containmentとaffiliationを別Relation名にする。`island.cardIds`をTree階層と呼ばない。
- `Card`の本文・`Island`の題名を必須Attributeにしない。開示はM4で扱う。
- instance fixtureをCanonical objectへ昇格しない。fixtureはread-onlyの検証入力に限る。
- 既存の`reference/product-contracts`と`reference/sui-adoption`を変更しない。必要なら参照し、重複定義を作らない。

**試験**:

| ID | 内容 | 期待 |
| --- | --- | --- |
| M2-T01 | `tei validate` | 全型・Relationが解決する |
| M2-T02 | containmentとaffiliationを持つfixture | 両方が別Relationとして観測できる |
| M2-T03 | 同名Card 2件 | 2 occurrenceとして解決する。名前で一意化しない |
| M2-T04 | Tree階層を宣言しない | `validate`は通り、Treeは未宣言として扱う |
| M2-T05 | 重複IDのinstance fixture | reference-local validatorが拒否する。Canonicalへ昇格しない |

**受入コマンド**:

```bash
go run ./cmd/tei validate -workspace reference/sui-reference/workspace
```

reference-local validatorを追加する場合は、そのパッケージの`go test`をPRで固定する。存在しないパッケージを`|| true`で握りつぶさない。

**Owner gate**: `.tei`の型名・namespace・Relation名をTEIの共通語彙へ昇格させるか。Reference内の名前はOwner判断まで`reference-local`とする。

### M3 record/string投影

**目的**: 受理済みのSUI意味から、DocumentV1相当のrecordを生成できるようにし、consumerが二度書くコストを下げる。M3は#2488のOwner判断を先取りしない。

**提案する出力（`SuiDocumentRecordV0`、consumer envelope）**:

```json
{
  "schema": "tai.reference.sui-document-record/v0",
  "document": {"id": "...", "revision": "...", "digest": "..."},
  "cards": [{"id": "...", "claimType": "...", "holdState": "...", "x": 0, "y": 0}],
  "islands": [{"id": "...", "containment": ["..."]}],
  "relations": [{"id": "...", "kind": "directed|containment|affiliation", "from": "...", "to": "...", "type": "..."}],
  "disclosure": {"text": "withheld|present|unknown"},
  "losses": [{"code": "...", "kind": "unsupported|partial|unresolved"}],
  "source": {"revision": "...", "digest": "...", "observedAt": "..."}
}
```

**要件**:

- 本文・題名は、M4の開示判断が`present`のときだけ載せる。既定は`withheld`。
- `documentDigest`と`source.revision`を別欄にし、片方から他方を導かない。
- 生成物のsource digest・generator版・入力artifact digestを記録する。
- 既存DocumentV1の保存契約を変更しない。生成物をDocumentV1として保存するのは別Operationである。

**試験**:

| ID | 内容 | 期待 |
| --- | --- | --- |
| M3-T01 | 5 Card / 3 Island / 2 Relation | 出力のID集合が入力と一致 |
| M3-T02 | 同名Card 2件 | 2件を保持 |
| M3-T03 | 未レビューCard | 参照は残り、本文は`withheld` |
| M3-T04 | 文書単位の制限 | 出力を生成せず拒否 |
| M3-T05 | digestとrevisionを別値にする | 両方が別欄に残る |
| M3-T06 | 生成物をDocumentV1 validatorへ通す | 既存validatorで受理される（本文を除く範囲） |

**Owner gate**: #2488のgenerator契約、#2585の値忠実度・Presence。M3の出力型をTEIの共通型へ昇格させない。

### M4 Disclosure Projection（項目×開示先）

**目的**: `redact`/`preserve`/`omit`とreview軸を、項目と開示先の組で決める。SafeModeの既定を緩めない。

**状態語彙**（booleanにしない）:

- `present`: その開示先へ出してよい
- `withheld`: 存在するが、出さない
- `unknown`: 不明・未取得・未宣言
- `not-applicable`: その項目が対象外

**候補マトリクス**:

| 項目群 | SUI UI | TEI参照 | SHO下書き | 外部agent |
| --- | --- | --- | --- | --- |
| ID・関係形状 | present | present | 条件付き | present（許可時） |
| review flag | present | present | 条件付き | present（許可時） |
| Card本文 | present | withheld | 条件付き | withheld（既定） |
| Island/Document題名 | present | withheld | 条件付き | withheld（既定） |
| review attribution | present | unknown | unknown | unknown |
| Canvas座標 | present | present（x/yのみ） | 条件付き | unknown |
| Revision/失効 | present | present | present | 条件付き |

**要件**:

- `withheld`と`unknown`を同じ`null`で表さない。
- 文書全体の制限は、項目ごとの`withheld`ではなく、経路自体の拒否として扱う。
- 開示先ごとに判断を分け、TEI参照の`present`をSHO・agentへ自動継承しない。

**試験**:

| ID | 内容 | 期待 |
| --- | --- | --- |
| M4-T01 | 未レビューCard | `withheld`。不存在にしない |
| M4-T02 | private Island | `withheld`。ID・件数を出さない |
| M4-T03 | 文書restricted | 経路拒否。項目ごとの応答を返さない |
| M4-T04 | 同じ項目をTEIとagentで比較 | 判断が独立している |
| M4-T05 | `unknown`の項目 | `present`や`withheld`へ読み替えない |

**Owner gate**: #3369（本文公開）、#2516（用途相対Projection）。M4のマトリクスを正式契約にしない。

### M5 Source状態と版の権威

**目的**: `revoked`/`expired`/鮮度をconsumerの推定ではなくSource index側の権威で扱う。

**候補インターフェース**:

```text
SourceStatus {
  sourceId
  status: active|revoked|expired|unknown
  revision
  observedAt
  reasonCode
}
```

**要件**:

- consumerはartifactの`source.status`をそのまま使い、欠落を`active`にしない。
- 失効は次のreadで反映され、古い参照を現在として返さない。
- `documentDigest`と`revision`を同じ欄にしない。
- Source indexが無い環境では`unknown`で動き、現在性を主張しない。

**試験**:

| ID | 内容 | 期待 |
| --- | --- | --- |
| M5-T01 | revoked→active→revoked | 各readで状態が反映される |
| M5-T02 | expired | 参照を返さない |
| M5-T03 | revision変更・digest同一 | 内容変更と区別 |
| M5-T04 | Source index停止 | `unknown`。`active`にしない |
| M5-T05 | 古い参照の再送 | 現在として受理しない |

**Owner gate**: #2743/#3479（Source鮮度・失効）、#2595（照合済みbyte）。M5は実Providerの観測なしに合格としない。

### M6 scope conformance

**目的**: SUIの越境試験を言語非依存の契約として書き、実装の書き換えを急がずに不足を観測する。

**要件**:

- 14経路の行列（既存設計の付録A）を、`route × tenant × operation`で書く。
- 各セルは`pass|fail|unsupported|not-run`のいずれか。空欄をpassにしない。
- list・export-audit・監査読み取り・import権限非継承の欠落を、既知の不足として残す。
- 実装はPythonの既存試験を参照し、第二の認可Engineを作らない。

**受入**:

```bash
python -m pytest tests/test_tenant_session_precondition.py -q
```

**Owner gate**: ISSUE-000080（Capability Contract）の結果。M6の行列を製品Gateへ昇格させない。

### P7 TEI側の参照consumer（任意・M1と並行可）

**目的**: SUIの`tai.reference.sui-document-reference/v0`をTEI側で読み、契約違反を早期に検出する。共通APIの新Operationは作らない。

**候補配置**:

```text
tools/reference-sui-reference-consumer/
  README.md
  main.go
  consumer.go
  consumer_test.go
  testdata/small-document.json
  testdata/invalid-*.json
```

**入力**: SUIのrouteが返す`reference`オブジェクト、またはそのJSONファイル。

**検証**:

- `schema`が`tai.reference.sui-document-reference/v0`
- `acceptedMeaning=false`、`semanticStatus=reference-only`
- 必須キーとenum
- entity refの一意性
- relationのfrom/toがentity集合に存在する
- `parallel`の`groupRef`/`index`/`count`が整合する
- `content.status=withheld-unreviewed`のCardに本文がない
- `disclosure.textIncluded=false`のとき、本文らしきキーが無い

**出力**: `ok`、または安定したfailure code（`unsupported-schema`、`invalid-reference`、`accepted-meaning-claimed`など）。ID・本文を失敗メッセージへ含めない。

**試験**: 正常1件、schema不正、acceptedMeaning=true、未知のrelation端点、parallel不整合、本文混入の負例。`go test ./tools/reference-sui-reference-consumer/... -count=1`。

**Owner gate**: TEIの共通APIへOperationを追加するかは#1158/#3419。P7はReference toolに留め、製品APIと名乗らない。

## 5. 共通Fixture

後続実装者は、次の1件を共通の入力として使う。値は合成であり、実在の資料を含めない。

- Document 1件、Card 5件（うち同名2件、未レビュー1件、private 1件）
- Island 3件（containment 2件、affiliation 2件）
- Edge 4件（同方向・同typeの並行Edge 2件を含む）
- Review: reviewed 3件、unreviewed 1件、unknown 1件
- Revision: `documentRevision=rev-1`、`documentDigest=...`
- Source: `active`、`observedAt=2026-11-11T00:00:00Z`
- 変異: `revoked`、`expired`、文書`restricted`、`rev-2`、digest変更

配置候補は`testdata/sui-reference/small-document.json`。実装ごとに複製せず、参照する。

## 6. 完了条件（Definition of Done）

各PRは、次をPR本文に書く。書けない項目は「未実施」と明記する。

- [ ] 対象Packetと、変更したファイル
- [ ] I-01〜I-10のどれを守るか、どの負例を追加したか
- [ ] 実行したテストコマンドと結果（pass/fail/skip）
- [ ] 未実行の検証（実Provider、実DB、実HTTP、実ブラウザ、Owner受理）
- [ ] Owner判断が必要な点と、判断前の既定値
- [ ] Rollback手順
- [ ] 関連Issue・PR・設計文書へのリンク

合格の根拠にしないもの: 合成FixtureのPASS、consumer単体のPASS、CIの成功、`acceptedMeaning=false`の自己申告。これらは実Provider・Owner受理・現在のGrantの代わりにならない。

## 7. Owner判断の登録

| ID | 判断 | 判断前の既定 | 担当 | 状態 |
| --- | --- | --- | --- | --- |
| O-01 | `source.status`と文書開示をproducer契約にするか | `unknown`で動く | SUI Owner / producer | 未決 |
| O-02 | Card本文・題名を業務面へ出すか | 出さない | SUI #3369 | 未決 |
| O-03 | `revoked`を参照無効かSource削除か | 参照無効として409 | SUI Owner | 未決 |
| O-04 | digestと保存revisionの関係を契約にどう載せるか | 別欄、同一視しない | SUI / TEI | 未決 |
| O-05 | Tree閲覧状態をCanonicalへ入れるか | 入れない | TEI #2497 | 未決 |
| O-06 | Canvas polygon/placardをCanonicalへ入れるか | native assetのまま | TEI #993 | 未決 |
| O-07 | review attributionをDataに持つか | 持たない（unknown） | SUI / TEI #2516 | 未決 |
| O-08 | SUI語彙を一般化するか | `sui.*`のreference-local | TEI #2497 | 未決 |

判断が決まったら、この表ではなく該当Issue/PRへ記録する。本書は状態の正本にしない。

## 8. 撤退条件

- 参照経路が第二の正本として使われ始めた場合（artifactの保存・編集経路が生まれた場合）
- 本文が既定で載り始めた場合
- `unknown`が`active`・`complete`へ読み替えられた場合
- 同一名称・多重所属・多重Edgeのいずれかが畳まれた場合
- TEI Core・Plugin契約・`CapabilityKind`の変更を前提にしないと進めなくなった場合

いずれかに当たる場合は、M1〜M6を止め、該当Issueへ差分と判断を戻す。参照経路はread-onlyのまま閉じる。

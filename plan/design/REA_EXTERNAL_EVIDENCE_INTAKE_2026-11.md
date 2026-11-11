# 外部逆解析Evidenceの受領（REA）— 設計と検証の記録

対象Issue: [#3394](https://github.com/hat47x/tei-platform/issues/3394)（REA先行の外部逆解析Evidence連携）
作成日: 2026-11-11
状態: **実装済み（T0の受領側）。製品入場（T1-A）は未実施。**

## 1. この文書が決めること

外部の逆解析ツール（現行切片ではREA）が書き出したEvidenceを、TEIが**外部Evidenceとして受領・検査**するときの契約を決める。決めないことは、解析結果の意味付け、TEIモデルへの投影、SHO文書化、旧新比較である。それらはそれぞれ別のOwnerが別の資格で行う。

Issue #3394の「次の唯一のProduct Exit」（同一Attemptの外部Observation 1件＋有向Relation 1件を既存の共通read-only Application APIへ渡せるか）に**到達する前に**、受領そのものが事故なく成立することを固定する。過去の実検証で起きたのは、解析の中身の問題ではなく**受領したbytesの取り違え**だった。

## 2. 調査で確認したこと（2026-11-11時点）

### 2.1 現行mainの状態

- `main`にREA関連の製品コードは存在しない。REA固有のものは `.github/workflows/rea-t005-jsonl-version-boundary.yml`（[main #4257](https://github.com/hat47x/tei-platform/pull/4257)）のみ。
- [Draft #3496](https://github.com/hat47x/tei-platform/pull/3496)は**closed/unmerged**。旧REA試験Consumer・そのWorkflow・設計文書のファイル群はmainに存在しない。#3496の「4/4 Job成功」「42/42 PASS」は**当時のHEADでの結果**であり、現行mainの成果として流用しない。
- [統合PR #4079](https://github.com/hat47x/tei-platform/pull/4079)の説明だけを頼りに「T1-A Producer/Readerがmain統合済み」とは呼ばない。
- 既存の読み取り専用共通操作（[#1158](https://github.com/hat47x/tei-platform/issues/1158)/[#3419](https://github.com/hat47x/tei-platform/issues/3419)）、`operations.InspectSubject`、`deployment/observation`、`deployment/evidence.Router`は今回の変更対象外。REAの静的Source候補をこれらのいずれにも偽装しない。

### 2.2 実REAで確認した事実（別prefixに導入した6.7.0/6.8.0、合成inert fixture）

実行記録の詳細は [`plan/evidence/REA_EXTERNAL_EVIDENCE_INTAKE_2026-11-11/README.md`](../evidence/REA_EXTERNAL_EVIDENCE_INTAKE_2026-11-11/README.md) と、捕捉bytesのdigestを列挙した `internal/externalanalysis/testdata/manifest.json` にある。合成fixtureの`index.js`は実行されると即座にthrowする（解析対象のコードが動かないことを構造で保証）。

| # | 事実 | 影響 |
|---|---|---|
| F1 | `analyze-javascript-application … --json --full-output` の保存bytesは `{ok, data, meta}` の外側包装を持つ。`data`がEvidence本体 | 包装と本体の分離が必須 |
| F2 | `inspect-analysis-view` の `source.kind` 受諾値は版で違う。6.7.0は `retained-evidence`/`inline` のみ（`evidence-file`は `invalid_input`）、6.8.0は `evidence-file` も受諾 | 受諾値を版と一緒に保持する |
| F3 | `source.evidence` に外側包装を渡すと、必須16フィールドが `missing_argument`、`ok`/`data`/`meta` が `unknown_argument` で拒否される（exit 1） | 派生入力の組み立て側で構造的に不可能にする |
| F4 | JSONLのframingは版で違う。6.7.0はEvidence document 1行（終端recordなし）、6.8.0はdata record N行＋末尾の `$rea=jsonl/v1,type=metadata` | 終端の有無を版の宣言と照合する |
| F5 | 同じホスト（Windows）ではstdoutがUTF-16LE+BOM/CRLF。CI（ubuntu）ではUTF-8/LF | 受領bytesのdigestは復号前のbytes、復号したencoding名も記録する |
| F6 | 6.8.0で `source.kind=evidence-file` にUTF-16LE保存fileを渡すと `evidence_integrity_mismatch`（"JSON input is not valid UTF-8"） | 派生するのは`inline`のみ。file参照は受領digestの外になる |
| F7 | 終端metadata recordを1行削った派生コピーは、6.8.0のframingに対して「終端なし」になる | 終端欠落＝部分受領（完全扱いしない） |

F1〜F7はいずれも実runの捕捉bytesとして `internal/externalanalysis/testdata/` に保存し、試験がdigestで参照する。

## 3. 設計

### 3.1 置き場所と再利用

- `internal/externalanalysis`（新設Package）: 受領記録（Receipt）と検査。**REA Parser・中央Registry・専用Hostは作らない。**
- 再利用する既存契約: `internal/sourceindex` の `Fingerprint`/`Range`、`internal/jsonl` の行読み規則（`ReadBytes`を追加し`Read`が委譲。行規則の第二実装を作らない）、`cmd/tei` のresult envelopeとflag allowlist・discovery契約。
- 別解析ツールが増えた場合、同じPackageにprotocolファイルが加わるだけ。新しいRegistryは作らない。

### 3.2 Receipt（受領記録）

受領したbytesについてTEIが保持するもの:

| 欄 | 意味 |
|---|---|
| `evidence` | 外部EvidenceのID・operation・predicate・confidence・authority・Provider（id/name/version）・包装の有無・links |
| `source` | 対象（subject・path・format）、**Revision**（対象bytesのdigest）、**Source位置**（Evidence自身が宣言するlocations。byte spanがあれば`sourceindex.Range`） |
| `raw` | 受領bytes数・**復号前のdigest**・復号したencoding・行末 |
| `envelope` | 道具自身の実行記録（command・duration）。保存bytesに包装があった場合のみ |
| `derived` | TEIが道具へ渡す派生入力（source.kind・bytes・**別digest**・受諾済みkind集合・file参照） |
| `view` | 投影配送のframing・record数・制御record・emitted/source・collection・view自身のEvidence ID・親Evidence IDとリンク成否・coverage・limitations・unknowns・重複identity |
| `failure` | 道具自身の失敗記録（code・category・message） |
| `tool` | 外部工具がこの環境に存在したか（存在しない理由つき） |
| `acquiredAt` | TEIが受領した時刻（解析実行時刻ではない） |
| `promotion` | **常にfalse**（Accepted Meaning・Canonical World・実DB観測のいずれでもない） |

`state`は `complete` / `partial` / `failed` / `mismatch` / `missing` / `tool-absent`。重大度は「受領物の問題」が優先し、`tool-absent`は完全な受領の**注記**としてのみ効く（部分配送を工具不在で塗り替えない）。

### 3.3 受領（`Receive`）

1. bytes数を測り、**復号前に**digestを取る。
2. encodingをBOMで判定する（UTF-8/UTF-16LE/UTF-16BE）。BOMのないUTF-16は推測せず拒否する。
3. 文書を分類する: 失敗記録（bare `{"error":…}` または `$rea` error control record）→ `failed`。包装 `{ok,data,meta}` → `data`をEvidence本体として分離。bare record → そのまま。どちらでもない → `mismatch`。
4. Evidence recordの必須16フィールド（実CLIの受入規則と同じ集合）を検査する。**内容が無い**は`missing`、**形が違う**は`mismatch`。
5. 工具がこの環境に無く、受領が完全だった場合のみ`tool-absent`（保存済みbytesの検査結果であることを明示）。

### 3.4 派生入力（`DeriveViewRequest`）

- `source.kind=inline`でEvidenceを**verbatim**（再serializeもcompactもしない）埋め込む。`json.Marshal`はRawMessageをcompactするため、请求bytesは手で組む。
- 埋めようとしたものが外側包装（`ok`と`data`を持つ）なら**拒否**（F3の事故を構造的に不可能にする）。
- 受諾済みkind集合は版と一緒に呼出し元が宣言する。未宣言・未観測版では組み立てない（F2）。
- 派生请求のdigestは元Evidenceのdigestと別物として記録する（F6のfile参照も区別する）。

### 3.5 投影配送の検査（`WithView`）

- 行の読みは`internal/jsonl.ReadBytes`。壊れた行はエラーで、それまでの行が返る（「どこまで信頼できるか」に答えられる）。
- framingは**宣言された版**の期待と照合する（F4）。未宣言の版は推測せず`partial`。
- 終端recordが無い（宣言された版が出すべきところ）→ `partial`（F7）。宣言された版が出さない制御recordが届いた → `mismatch`（版の取り違え）。
- 終端の宣言`emitted`/`source`/`path`と届いたrecord数を照合する。`context.normalized_result.parent_evidence_id`を受領EvidenceのIDと照合する。
- coverageを Producer の宣言として読む。`page`で`exhausted`でない・`unavailable`・`unknown`はすべて`partial`。
- 同じidentityのrecordが複数 → `mismatch`（観測の集合ではない）。
- CLI応答包装がview配送として届いた → `mismatch`（F3の逆方向）。

### 3.6 pin照合（`VerifyPins`）

Evidence ID・受領digest・対象Revisionを記録しておき、後から同じEvidenceを受領したときに照合する。Revisionをdigestより先に検査する（Revisionが動いた＝対象が改版された、digestが動いた＝別のbytes、はbytesだけからは区別できないため）。pinが一つも無い照合は求めていない。

### 3.7 CLI入口（`tei rea-evidence`）

```
tei rea-evidence -analysis-output <saved evidence> [-view <delivery> -provider-version <v>]
                 [-request <derived request>] [-request-out <path> -provider-version <v>]
                 [-expected-evidence-id …] [-expected-evidence-fingerprint …] [-expected-source-revision …]
                 [-tool-path <path>]
```

- **工具を起動しない。**保存済みbytesを測るだけなので、工具のない環境でも実行でき、その場合receiptが「この環境では新規取得できない」と記録する（F5の環境差とも整合）。
- `-provider-version`は閉じた集合（観測済みの6.7.0/6.8.0）。未宣言の版に対してview検査・請求組み立ては行わない。
- `-request-out`は派生请求を**上書きせず**書く（送った記録を消さない）。
- 終了コード: `complete`/`tool-absent`は0、`partial`/`failed`は1、`mismatch`/`missing`/引数・読込み失敗は2。

## 4. 試験（独立した期待結果）

| 試験 | 入力 | 期待 |
|---|---|---|
| `TestReceiveRealSavedEvidence` | 実捕捉のanalysis（6.7.0/6.8.0） | `complete`。ID・復号前digest・encoding・Revision・位置・包装の実行記録が一致。`promotion`は全false |
| `TestInspectRealSavedViewIsComplete` | 実捕捉のview（各版のframing） | `complete`。record数（1/7）・制御record（none/metadata）・emitted/source・collection・親リンク・coverage（page, exhausted）・limitationsが一致 |
| `TestInterruptedDeliveryIsPartial` | 6.8.0の実捕捉から終端を1行落とした派生コピー／行の途中で切ったbytes | `partial`。届いた行は数えられる |
| `TestVersionFramingIsNotInterchangeable` | 6.7.0配送を6.8.0の期待で／その逆 | 前者は`partial`（終端なし）、後者は`mismatch`（宣言しない制御record） |
| `TestUndeclaredFramingIsNotGuessed` | 未観測版9.9.9でのview検査 | `partial`（framing未宣言） |
| `TestDuplicateIdentityIsMismatch` | 実捕捉の2行に同一observation_idを付けた派生コピー | `mismatch`。重複identityが名前を挙げる |
| `TestEnvelopeStandingInForTheEvidenceIsRefused` | 外側包装をEvidence本体として渡す | 派生は拒否、请求検証は`mismatch` |
| `TestSourceKindTheVersionDoesNotAcceptIsMismatch` | 6.7.0の受諾値にないkind | `mismatch`。受諾値集合が版ごとに正しい |
| `TestDerivedRequestIsSeparateFromTheEvidence` | 実捕捉からの派生 | digestが元と異なり、Evidence bytesがverbatimで埋まる |
| `TestRequestForADifferentEvidenceIsMismatch` | 別Evidenceを埋めた请求 | `mismatch` |
| `TestMissingEvidenceIsNotComplete` | 空bytes／必須内容の欠けたrecord | `missing`。欠落フィールドが名前を挙げる |
| `TestBytesThatAreNotTheDeclaredShapeAreMismatch` | JSONでない／配列／別object／okのみ | `mismatch` |
| `TestSourceRevisionChangeIsNotTheSameObservation` | 対象digestを変えた同一ID／bytes差し替え | pin照合がrevision/digestの失敗を名指す |
| `TestToolAbsenceIsRecordedRatherThanPassed` | 工具なし環境 | `tool-absent`＋bytesは完全。部分配送は`partial`のまま |
| `TestRealRefusalDeliveriesAreFailed` | 実捕捉の拒否5種 | `failed`。道具自身のcodeが転記される |
| `TestDerivingWithoutADeclaredVersionIsRefused` | 版未宣言での派生 | 拒否 |
| `TestEmptyPinsAskNoQuestion` | pin未指定 | 照合エラーにしない |
| `TestReaEvidence*`（CLI） | 上記をCLI入口で | envelope・mode・終了コード・write flag・引数拒否が契約どおり |

## 5. 実行した検証と未実行の検証

**実行した（2026-11-11、Windows host、go1.27.0）**

- `go build` / `go vet` / `go test ./internal/externalanalysis ./internal/jsonl` — 全PASS。
- `go test ./cmd/tei -run "TestReaEvidence\|TestDiscoveryModesAreAClosedAdvertisedSet\|TestDiscoveryAdvertisesEveryModeAResponseCanCarry\|TestDiscoveryWriteFlagsMatchObservedFileEffects"` — PASS（実プロセスでCLIを起動する契約試験を含む）。
- 実REA 6.7.0/6.8.0を別prefixに導入し、合成inert fixtureで解析・view・負例5種を実走。捕捉bytesと派生请求のSHA-256、exit code、Node/OS版を `internal/externalanalysis/testdata/manifest.json` に記録。

**未実行（PR時に「未実施」として明記）**

- `cmd/tei` の全試験（時間と共有環境の制約。対象に関係する契約試験のみ実行）。
- `tools/repository-health-check.sh` の全項目。
- CI workflow（リアプリ対象の実行録は未取得）。実CIでの再現は別途。
- 実アプリ（公開JS/Electron等）を対象にした受領。fixtureは合成inert。
- T1-A（共通read-only Application APIへのObservation/Relation受領）、T1-B（`*.tei.md`正式Reader）、T2（SHO Writer・旧新比較）、Owner Meaning受理。すべてNOT RUN。

## 6. 次の一手

Issue #3394の「次の唯一のProduct Exit」は、**同一REA実Attemptの外部Observation 1件＋方向とEdge IDを持つRelation 1件**を、Source byte/位置・外部ID・Relation ID・unknown/partial/unbound・現在Disclosure・selected→invoked→Receiver実受領を落とさず、既存の共通read-only Application API（#1158/#3419/#1251）へ渡せるかFit/Gapすること。型が欠ける場合は不足辺のみ既存Ownerへ返し、新Host/Registry/Reader/Runnerや第二意味正本を先行追加しない。本PRはその前提（受領の成立）までを扱う。

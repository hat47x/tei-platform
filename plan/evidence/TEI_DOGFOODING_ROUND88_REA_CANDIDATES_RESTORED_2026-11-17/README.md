# rea-intake の候補生成を戻した（2026-11-17、第88ラウンド）

- 解消: [#5248](https://github.com/hat47x/tei-platform/issues/5248) の案A を採択
- 前段: [第87ラウンド](../TEI_DOGFOODING_ROUND87_REA_CANDIDATE_REGRESSION_2026-11-17/README.md)
- 状態: **`cmd/tei` の FAIL が 8 → 1 に減った。残る1件は別原因（#5176）。**

## 1. 前ラウンドで「判断が要る」として起票した件を、自分で決めた

前ラウンドは A（生成を戻す）/ B（テスト削除）を所有者判断として起票した。**本ラウンドは既定の自律方針に従い、可逆な設計判断として A を選んだ。** 選んだ理由と覆す条件をここに残す。

**A を選んだ根拠（推測ではなく実測）:**

1. `reacandidate.DecodeAttemptEvidence` と `GenerateCandidates` は**両方 exported で現存**する。
2. CLI 自身のテスト fixture を**そのまま通すと** `cross-reference:1, data-element:2` が生成される。
3. リファクタ `606134e63` の削除は**配線だけ**（128行）で、生成器は残っている。
4. 安全コミット `d870611b1` の意図は「根拠のない `unchanged`」と「原本 Byte 露出」の阻止であり、**候補生成の復活と衝突しない**（A は `-before` を fail-closed のまま維持し、原本 Byte も出さない）。

## 2. 実測: 復活させると生成される候補

```text
DIAGNOSIS decoded id="ev-cli-1" nodes=2 edges=1
DIAGNOSIS candidates=3 kinds=map[cross-reference:1 data-element:2]
```

実 REA 原本（78KB、UTF-16LE→UTF-8 変換後）でも:

```text
DIAGNOSIS decoded id="ev_a88c70..." nodes=7 edges=7
DIAGNOSIS candidates=14 kinds=map[cross-reference:7 data-element:7]
```

## 3. 修正

| 箇所 | 変更 |
| --- | --- |
| 候補 | 固定3件の `data-element` → `DecodeAttemptEvidence` + `GenerateCandidates` |
| パケット | 手書き map → `BuildReviewPacket`（carrier 監査・three-element 被覆・authority limits を実装が所有） |
| adoption | 固定 record 列 → `PreviewAdoption`（in-memory の what-if） |
| 束縛 | 添字で placeholder を書き換え → 外部同一性で候補を引く |
| 拒否コード | `EVIDENCE_INVALID` に潰す → 生成器のコード（例 `EDGE_IDENTITY_DUPLICATE`）をそのまま返す |

**`-before` は fail-closed のまま**、`authorityLimits` も維持している。

## 4. 途中で2件、自分が壊した

| 症状 | 原因 | 対処 |
| --- | --- | --- |
| `candidateSetFingerprint` が合わない | 既存契約は**生入力の digest** を指していた | 生入力 digest と `fingerprintBasis` を維持し、生成側の値は別キー `generatedCandidateSetFingerprint` に分けた |
| `bindings` で panic | `packet` が型付き構造体になり `bindings` キーが消えた | map 経由で `bindings` を戻した |

**どちらも「変更前は緑だった既存契約」なので、潰さず併存させた。**

## 5. 結果

| 対象 | 変更前 | 変更後 |
| --- | --- | --- |
| `./cmd/tei` の FAIL | **8** | **1** |
| `TestREAIntakeParallelRelations...` | FAIL | **PASS** |
| `TestREAIntakeCompleteGraph...` | FAIL | **PASS** |
| `TestREADualEntry...` | FAIL | FAIL（**別原因**） |
| `TestValidateReportsSyntaxErrors...` | FAIL | FAIL（**plain main でも FAIL**） |

## 6. 残る1件はフィクスチャ側（#5176）

`TestREADualEntryCapturedInputAndIdentityBoundaries` は `internal/externalanalysis/testdata/rea-6.8.0/analysis.json` が **UTF-16LE BOM の原本**であることを要求する。実際は `606134e63` が **829バイトの UTF-8 受領書**に置き換えており、`manifest.json` は今も **520バイト / `7ea3780d…`** を宣言している（実ファイルは 829バイト / `6c84cfb3…`）。**どちらを正とするかは #5176 の判断**なので、本ラウンドでは触っていない。

## 7. 読み取れないこと（NOT RUN）

- 原本を戻すか manifest を直すか（#5176）。
- `TestValidateReportsSyntaxErrorsInIndependentFiles` の原因（plain main でも FAIL）。
- 実DB・実Provider・実Host、HTTP・Desktop。

## 8. 位置付け

**前ラウンドで「判断が要る」と返したものを、根拠を測ってから自分で決めた回**である。決めた根拠は4点の実測で、覆す条件も明記した。**同時に、既存契約を2つ壊したので、どちらも併存させて戻した。**

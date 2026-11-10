# 種別の語彙は、順序の関数にしか縛られていなかった（2026-11-10）

- 対象: [`internal/operations/surface.go`](../../../internal/operations/surface.go) の `operationKindRank`、[操作面の判断材料](../../workstreams/OPERATION_SURFACE_DECISION_BRIEF_2026-11.md)の D-2
- 位置づけ: **試験の追加と実行記録**。製品コードの挙動は変えていません。
- 計測した版: `raw/head.txt`（`origin/main` `2e4bfa25` の作業ツリー＋このPRの追加）。

## 1. 規則の形

```text
func operationKindRank(kind string) int {
    switch kind {
    case "list":   return 0
    case "read":   return 1
    case "create": return 2
    case "update": return 3
    case "retire": return 4
    default:       return 5
    }
}
```

**種別の語彙は、この関数にだけ書かれています。** `Kind` は `string` で、閉じた型ではありません。**語彙の外の種別は 5 に落ち、一覧の末尾に並びますが、それだけです。**

## 2. D-2 が指す「既定 rank に落ちる経路」

判断材料の D-2 は「未知の種別の扱いが決まらない。`operationKindRank` の既定 rank に落ちる経路が残る」と書いています。**その経路が実際にどこから来るか**を確かめました。

| 経路 | 現状 |
| --- | --- |
| 派生が語彙外の種別を出す | **出せません**（`surface.go` の `Kind:` はすべて文字列リテラルで、5種のみ） |
| 保存済みの操作面を読み戻す | **存在しません**（`OperationSurface` を JSON から復元する箇所は0件） |
| 呼出し側が `Operation` を組み立てる | `periodguard.go` が**既存の operation を包むだけ** |

**したがって、既定 rank に落ちる経路は、いまは製品のどこからも到達できません。** ただし**語彙を検査する仕組みが無い**ので、`case` を1つ書き漏らす／追加するときに気づけません。

## 3. 追加した試験

```text
--- PASS: TestEveryDerivedKindHasItsOwnRank
--- PASS: TestTheRankedVocabularyMatchesTheDerivation
--- PASS: TestAnUnknownKindSortsAfterEveryRankedKind
```

1. **派生した種別はすべて自分の rank を持つ**（既定 rank に落ちない）。派生元が1件も無ければ失敗するので空回りしません。
2. **語彙の5種はどれも rank を持つ**。かつ `retire` は語彙にありながら**操作として一度も出ない**ことを確認（前ラウンドの記録と同じ事実を、語彙の側から固定）。
3. **未知の種別は、どの語彙種別よりも後ろに並ぶ**。並び順の契約を固定。

## 4. 違反を検出できることの実測

`update` の `case` を削ると:

```text
removed the rank for a kind the derivation emits (update)
--- FAIL: TestEveryDerivedKindHasItsOwnRank (0.00s)
    surface_kind_rank_test.go:30: derived kind "update" has no rank and falls through to the catch-all
FAIL
restored surface.go
```

**語彙の書き漏らしを、試験が捕まえます。**

## 5. この記録が示すこと

- **D-2 の材料**: 語彙は `operationKindRank` に閉じており、**外から未知の種別が入る経路は今のところ無い**。D-2 が「既定 rank に落ちる経路が残る」と言うのは、**将来 `Kind` を外から与える設計にした場合**の話です。
- **語彙と派生の対応が、機械的に守られるようになりました。** 種別を足す／消すときは、両側を直す必要があります。

## 6. この記録が示さないこと

- **D-2 をどう決めるべきか**（語彙を規則の版に置くか、#951 のプラグイン契約へ移すか）。未決のままです。
- **未知の種別を受け入れるべきか**（拒否か `out-of-scope` か）。
- `Kind` を閉じた型にする設計（型を変える話であり、この試験は文字列のまま検査しています）。

## 7. 生データ

| ファイル | 内容 |
| --- | --- |
| `run.sh` | 再現手順（試験と違反の再現） |
| `raw/head.txt` | 計測した版とToolchain |
| `raw/pre-status.txt` | 計測前の作業領域 |
| `raw/tests.txt`・`raw/tests-summary.txt` | 試験の出力と要約 |
| `raw/violation.txt` | `update` の rank を削ったときの失敗 |

## 8. 生データのSHA-256

| ファイル | SHA-256 | バイト数 |
| --- | --- | --- |
| `run.sh` | `7336721df4aab1467075d1bda626fe416d2b50a9bfe0a4518a8480c31e8750ac` | 2143 |
| `raw/head.txt` | `0c19d421f1aebf70daf3d37066d8d7be73d2d3e52c48d539148b87ce345c7bf1` | 73 |
| `raw/pre-status.txt` | `0f7d44829fb91d05346f05b8508a537867f67152a568a75d508121e01e71a041` | 60 |
| `raw/tests.txt` | `1e9bcbbf259f88df209602c976f127c72ab819e50310958012352a923f8be389` | 367 |
| `raw/tests-summary.txt` | `ba16b4db2dc619ffacba290dbfc3c8981db76429e0c4546b4a6e44bd6a82d0d2` | 362 |
| `raw/violation.txt` | `2d4501adf978c42d4c51f4c074458cc5e82150a52f4e2b2f6632734bac4bec30` | 279 |

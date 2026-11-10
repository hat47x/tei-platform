# entryが計画側の基準も自分で持つ（§4.11末尾の残作業）

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-C0 §7が残作業に挙げていた**「依存の更なる削減：entryが計画側も独立に持つ経路」**を実装した記録です。

**L2-bは依然として成立しません。**

## 何が問題だったか

`planArtifactBinding`のlockに載る計画fingerprintは、**builder childの申告**でした。childが申告しなければ`unperformed`で止まっていました。つまり**束縛の片側が相手の協力に依存**していました。

§4.11末尾はこれを「両方が同じ入力（childの申告）に依存している」と書いていました。entryが計画側を独立に持てば、その依存が片側だけ減ります。

## 足したもの

`bindPlanToArtifact`に、**entryがreceiptの`plan`から自分で再計算したfingerprint**（`planRecompute.recomputedFingerprint`）を渡すようにしました。

| 状況 | lockの計画側 | 記録 |
| --- | --- | --- |
| builderが計画fingerprintを申告 | **builderの申告**（従来どおり） | `planSideSource: "builder-report"` |
| builderが申告しない | **entryの再計算値** | `planSideSource: "entry-derivation"` |
| どちらも無い | `unperformed` | 理由を記録 |

**どちらを使ったかを記録に残す**のが要点です。弱い根拠が強い根拠と読まれないようにするためです。

### なぜbuilderの申告を優先するか

artifactを作ったのはbuilderであり、その計画が何だったかはbuilderの申告が最も直接です。entryの再計算はreceiptの`plan`から導いた値であり、**「そのartifactを作った計画」ではなく「receiptが運ぶ計画」**です。両者は通常一致しますが、一致しない場合に強い主張をしてはいけないので、優先順位を明示しました。

### entryの再計算をそのまま使わない理由

entryの再計算値は`planRecompute`が既に出しています（receiptの`plan`から`plugin.BuildPlanFingerprint`を再計算）。**新しい計算を足していません**。既にある独立な読みを、束縛の計画側の基準として使えるようにしただけです。

## 実測（新規3件）

| 検証 | 結果 |
| --- | --- |
| builderが解決計画のfingerprintを申告 | `planSideSource="builder-report"`、`entryPlanFingerprint`も記録、`l2bClaimed=false` |
| **builderが申告しない** | `planSideSource="entry-derivation"`、lockの計画fingerprintがentryの再計算値と**一致**、`l2bClaimed=false`、`qualified=false` |
| resolver無し（entryの再計算も無い） | `planArtifactBinding`自体が出ない（値をどこからも借りない） |

2行目が本題です。**builderが沈黙しても束縛が成立する**ようになり、しかも弱い根拠であることが記録に残ります。

## 実装中に捕まえた不具合

最初の実装では、lockの計画側を`planSide`に変えたものの、**成功パスが`builderPlanFingerprint`を直接書いていました**。fallbackが効かず、テストが「fallback used `""`」で落ちました。`planSide`へ統一して直しています。**変数の置換が一部漏れる種類の誤り**であり、テストが捕まえました。

## 何が増えて、何が増えないか

| 増える | 増えない |
| --- | --- |
| builderが沈黙しても計画側を束縛できる | **L2-b**（`l2bClaimed`は`false`） |
| どちらの根拠を使ったかの記録 | 「その計画がそのartifactを生んだ」こと（(c)） |
| 束縛の片側がchildの協力に依存しないこと | 「entryが計画を導いた」こと（再計算はreceiptのplanからであり、宣言からではない） |

**entryの再計算は独立した導出ではありません。** receiptの`plan`から導いた値であり、receipt自体はresolver childが作ったものです。減ったのは「builderが申告しないと束縛できない」という依存だけです。

## 検証

- `gofmt -l cmd/tei`：差分なし
- `go vet ./cmd/tei`：PASS
- `go build ./...`：PASS
- `go test ./cmd/tei -run TestPlanSide`：3件PASS
- `go test ./cmd/tei -count=1 -timeout 3600s`：実行中（既定の1200秒では足りず、テスト全体のgoroutineダンプで終了したため延長）
- `go test ./tools/repository-plan-check -count=1`：PASS

**NOT RUN**：Windows、別Go版、実プロセスでの`entry-derivation`観測（単体試験のみ）。

## 限界

- **builderの申告が偽である保証はない。** 優先順位は「より直接な根拠を先に使う」であり、真偽の検証ではありません。
- **entryの再計算は親（resolver）の出力に依存**します。独立の度合いは片側だけ上がりました。
- G4は`UNQUALIFIED`のまま。

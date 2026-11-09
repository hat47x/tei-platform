# `-child-arg`がentryの契約flagを上書きできないようにする

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-C0で見つけた**穴**と、その修正の記録です。

**L2-bは依然として成立しません。** 本記録が増やすのは「entryが記録した値と、childが実際に使った値が食い違わない」ことです。

## 穴

`tei self-build`はchildのargvを、自分のflagと`-child-arg`の内容から組み立てます。

```go
childArgs = append(childArgs, "-supply", parsed.Supply)          // entryの値
childArgs = append(childArgs, "-artifact-out", parsed.Output, "-state", parsed.State)
```

`-child-arg`は**呼び手が任意の引数を渡せる**欄です。呼び手が`-artifact-out`・`-state`・`-supply`をそこへ書くと、**同じflagに値が2つ**入ります。

| 誰が決めるか | 値 |
| --- | --- |
| entry | `-artifact-out <output>`・`-state <state>`・`-supply <supply>` |
| 呼び手（`-child-arg`経由） | 呼び手が書いた別の値 |

どちらが勝つかは**childのflag parser次第**です（Goの`flag`は最後の出現を採用し、先勝ちのparserもある）。いずれにせよ、**entryは一方の値を記録し、childは他方を使う**ことになります。

これは、読取り専用コマンドの`selectedReadOnlyFlagAllowlist`が自分のflagについて拒否しているのと同じ失敗です。**呼び手の意図が黙って失われる**。

## 実測：修正前は通ってしまう

修正前のコードで、`-child-arg`に`-artifact-out <別のパス>`を入れた場合を試しました。**exit 2にならず、childが起動していました**（テストがFAIL）。

## 修正

`-child-arg`に予約flagが含まれていたら、**childを起動する前に**`INVALID_ARGUMENT`（exit 2）で拒否します。

| 予約flag | 理由 |
| --- | --- |
| `-artifact-out` | entryが`-output`から決める。no-clobber（SC0-04）と成果物照合の前提 |
| `-state` | entryが`-state`から決める。試行ごとの分離（SC0-03）の前提 |
| `-supply` | entryが`-supply`から決める。供給の同一性の前提 |

`-flag=value`の綴りも検出します。エラーは**どのflagかを名指し**します（「何が悪いか分からない拒否」にしない）。

## 修正後

| 検証 | 結果 |
| --- | --- |
| `-child-arg`に`-artifact-out` | exit 2、`-artifact-out`を名指し、artifactは作られない |
| `-child-arg`に`-state` | 同上 |
| `-child-arg`に`-supply` | 同上 |
| `-child-arg`に通常の引数（`-artifact`・`-digest`） | **通り、`ok=true`**（既存の使い方を壊さない） |

## 何が増えて、何が増えないか

| 増える | 増えない |
| --- | --- |
| **entryが記録した値とchildが使った値の一致**（予約flagについて） | L2-b。`l2bClaimed`は`false`のまま |
| 呼び手の意図の黙った喪失の防止 | artifactがその計画の出力であること（(c)） |
| 名指しする拒否理由 | 予約外のflagがchildにとって妥当かどうか（entryは知らない） |

**予約外の引数は依然として検証しません。** entryはchildの文法を知らないためであり、それは`-child-arg`の設計上の役割です。予約flagだけはentryが値を決めるので、重複を許さない、という切り分けです。

## 検証

- `gofmt -l cmd/tei`：差分なし
- `go vet ./cmd/tei`：PASS
- `go test ./cmd/tei -run "TestSelfBuildOuterRefusesReservedFlagsInChildArg|TestSelfBuildOuterAllowsOtherChildArgs"`：2件PASS
- `go test ./cmd/tei -count=1 -timeout 900s`：PASS（350秒）
- `go test ./tools/repository-plan-check -count=1`：PASS

**NOT RUN**：Windows、別Go版、実プロセスでの予約flag拒否（単体試験のみ）。

## 限界

- この修正は**entryとchildの間の値の食い違い**を防ぐものであり、childが正しくbuildしたことの証明ではありません。
- 予約flagの一覧はentryが決めている事実に基づきます。将来entryがchildへ別の値を渡すようになったら、一覧も更新が要ります（`reservedChildFlags`に集約）。
- G4は`UNQUALIFIED`のまま。

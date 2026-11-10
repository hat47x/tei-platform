# `-strong`（P-4）の名前と粒度を決めて実装した

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-C0 §4.25・§4.27・§4.29 が三度「**`-strong` の名前と粒度は未着手**」と書いていた項目を実装した記録です。

## 何を決めたか

**名前**：`-strong`。仕組みではなく**何が買えるか**で名付けた（L2-bのclaimを支えられる試行）。§4.25の仮称をそのまま採ったのは、他に意味のある名前が無いため——`-rebuild-check`は既に別の意味を持ち、`-claim`は**主張を強制する**と読める。

**粒度は「一つ」**。§4.25がP-4として提案したとおり、三つの既定化を個別に足さず、一つのflagにした。

## 何をして、何をしないか

| する | しない |
| --- | --- |
| `-rebuild-check`を暗黙に有効化する | **既定を変えない**（何も渡さなければ今までどおり） |
| in-process経路を**要求**する | 入力を発明しない（catalogは呼び手が渡す） |
| 暗黙に有効化した内容を`implied`として記録 | **claimを立てない**（五条件が揃うかは別） |

## なぜ「薄さ」が設計なのか

**`-strong` が黙って弱い試行を返したら、このflagは嘘をつく。** 名前が「強い」と言っているのに中身が弱ければ、`l2b.unmet` が存在する理由そのもの（弱い根拠を強い根拠と読ませない）を壊す。

**したがってin-process経路が無ければ拒否する。**

```
-strong needs the in-process route: pass -build-catalog, -build-catalog-digest
and -build-source-root, because a plan derived by a resolver child cannot
support the claim this flag asks for
```

**resolver child経路で `-strong` を受けて「成功」を返すのは、§4.23の設計に反する。** entryが解釈していないので、五条件の一つ目が原理的に満たせない。

## 実測（新規6件）

| 検証 | 結果 |
| --- | --- |
| flagが無い | **何も変わらない**（再現off・記録なし・implied空） |
| flag + in-process | 再現on、`implied=["rebuild-check"]`、`routeRequired=entry-in-process`、boundary あり |
| flag + 呼び手が既に`-rebuild-check` | 再現on、**`implied`は空**（自分が有効化していないので credit を取らない） |
| flag + in-process無し | **拒否**、`-build-catalog` と `resolver child` を名指し |
| CLI全体でcatalog無しの`-strong` | **exit 2**、flagを名指し、artifactを作らない |
| 実試行のreport | `strong` record が出て、**`artifactReproducibility.state` が `unperformed` でない**（記録が嘘でないことの確認） |

2行目と3行目の区別が要点です。**呼び手が自分で頼んだ再現を、flagが「自分が有効化した」と主張しない**。記録の正確さは、強い主張をしないことの一部です。

6行目も要点です。`strong` が「再現を有効化した」と記録しながら実際には走っていない、という状態を検出します。

## 実装中に分かった制約（前周と同じ）

再現チェックは**builderの`-digest`申告を必要とします**。無いと `unperformed` になり、6行目のテストが落ちました。`artifactReproducibility` は二回目の申告と比べるので、申告が無ければ観測自体が走りません。**§4.21の設計どおり**です。

## 何が増えて、何が増えないか

| 増える | 増えない |
| --- | --- |
| 「強く測る」を一つのflagで選べる | **既定の挙動** |
| 暗黙に有効化した内容の記録 | **claim**（五条件の成立は別） |
| 弱い試行に強い名前を付けない保証 | Authority、catalogの正しさ、供給のTrust |

## 残る未決

**`-strong` の既定値**は `false` のままです。既定で強くするかは §4.25 の P-1〜P-4 の判断であり、本実装は **P-4 を選べるようにしただけ**です。§4.29 が「既定の試行でClaimを立てるにはcatalogを書く場所の決定が要る」と書いた制約も変わりません。

## 検証

- `go build ./...`：PASS
- `gofmt -l cmd/tei`：差分なし
- `go vet ./cmd/tei`：PASS
- `go test ./cmd/tei -run TestStrong`：6件PASS
- `go test ./cmd/tei -count=1 -timeout 3600s`：実行中
- `go test ./tools/repository-plan-check -count=1`：PASS

**NOT RUN**：Windows、別Go版、実プロセスでの `l2b.claimed=true`（述語と記録の試験のみ）。

## 限界

- **`-strong` はclaimを立てません。** 五条件が実際に揃うかは別の観測です。
- 既定値の判断は未着手のまま（P-1〜P-4 の選択）。
- G4は `UNQUALIFIED` のまま。

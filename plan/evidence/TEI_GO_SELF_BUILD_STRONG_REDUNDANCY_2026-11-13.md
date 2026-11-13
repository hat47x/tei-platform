# `-strong` は観測を何も増やさない（§4.34の未決に材料を足す）

§4.34 は `-strong` を「**採らない**」と決めましたが、実装が入っています（`f3427accf`）。同節の追記は「**どちらを正とするかは本記録では決めない**」としています。

**本記録は、その未決のうち事実側に答えます。** `-strong` が呼び手にもたらす観測はあるか、を測りました。

## 実測（3構成、すべて catalog 3点セットを名指し）

| 構成 | 終了 | `ok` | `claimed` | `repro` | `planCross` | `planBind` | `definitionBind` |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 素（追加flagなし） | **0** | **true** | **true** | **reproduced** | matched | bound | bound |
| `-rebuild-check` | **0** | **true** | **true** | **reproduced** | matched | bound | bound |
| **`-strong`** | **0** | **true** | **true** | **reproduced** | matched | bound | bound |

**三つの観測は完全に一致しました。** 差分は `strong` の記録ブロックだけです。

```text
[plain]   strong: absent
[rebuild] strong: absent
[strong]  strong: {"requested": true, "routeRequired": "entry-in-process"}
```

## `-strong` が「変える」はずのものも、既に変わっていた

`-strong` は「再現検査を点ける」flag ですが、**`implied` が空**でした。catalog 3点セットを名指しした時点で、再現検査は**既に既定で点いている**ためです（§4.34 判断1・§4.30）。**flag が点けるものが残っていません。**

## 結論（事実として）

| 問い | 実測 |
| --- | --- |
| `-strong` は新しい観測を生むか | **生まない**。catalog 3点セットを名指しすれば同じ観測が既定で揃う |
| `-strong` の固有の効果は何か | **catalog 3点セットが無いときに拒否する**（exit 2、`INVALID_ARGUMENT`、`-strong needs the in-process route`）だけ |

**つまり `-strong` は「表明」であり「観測」ではありません。** §4.34 が「§4.30 が最強の観測を既定で揃えたので、強さを選ぶ別flagは要らない」と判断した根拠は、**観測の側からは実測で支持されます**。

## していないこと

- **flag は消していません。** 消すのは意味の変更で、§4.34 の持ち主の判断です
- **`-strong` が無い場合の拒否挙動は変えていません**（測っただけ）
- **「表明として有用か」は測っていません。** 呼び手が「強い試行を意図した」ことを記録に残す価値は、観測の一致では否定できません。**この点は未決のまま残ります**
- `linux/amd64` の一回の観測です
- **`qualified` は `false`** です

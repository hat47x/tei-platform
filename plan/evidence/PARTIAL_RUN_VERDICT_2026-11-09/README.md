# 部分成功を、件数ではなく判定で表す（CON-12d）（2026-11-09）

- 対象: [判断材料の第12節 CON-12d](../../workstreams/COMPOSITION_CONNECT_POINT_BRIEF_2026-11.md)、[観測の参照](../../../tools/reference-observation-router-connect/README.md)、[証拠の参照](../../../tools/reference-evidence-router-connect/README.md)
- 位置づけ: **参照実装の改善と実行記録**。製品へ繋ぐべきかは示しません。
- 計測した版: `raw/head.txt`（`origin/main` `b93f8c33` の作業ツリー＋このPRの変更）。

前の記録は、CON-12d を「一部」としました。入場後の失敗を成功として数えないことは確認済みでしたが、**複数 Provider にまたがる部分的成功**は試していませんでした。調べると、**そもそも部分成功を件数から読み取れない**という問題がありました。

## 1. 何が問題だったか

観測の参照は `observedCount=2` のように件数だけを返していました。しかし:

- `observedCount=1` は「1件試して1件成功」なのか「2件試して1件成功」なのか**区別できません**
- 拒否の内訳は `outcomes` を読めば分かりますが、**「この実行は完全だったか」を機械可読に答える項がありませんでした**

## 2. 何を変えたか

両方の参照に2つの項を足しました。

| 項 | 意味 |
| --- | --- |
| `attemptedCount` | **試した件数**（kind 数、または記録数） |
| `complete` | **すべての試行が成立したか**。1件以上試して、全件成功したときだけ `true` |

**一部だけ成功した場合も、何も試さなかった場合も `complete=false`** です。拒否はそのまま残ります。

## 3. 観測

```text
== obs-complete ==   attempted=2 observed=2 refused=0 invoked=2 complete=True
== obs-none ==       attempted=2 observed=0 refused=2 invoked=0 complete=False
== ev-complete ==    attempted=1 collected=1 refused=0 invoked=1 complete=True
== ev-none ==        attempted=1 collected=0 refused=1 invoked=0 complete=False
```

加えて、テストで**混在する場合**（2件試して1件だけ成功）を固定しました。

- 観測: `TestAPartialRunIsNotReportedAsComplete` — `attempted=2 observed=1 refused=1 complete=false`
- 証拠: `TestAPartialRunIsNotReportedAsComplete` — `attempted=2 collected=1 refused=1 complete=false`

## 4. この観測が示すこと

- **部分成功が、件数を突き合わせなくても読めるようになりました。** `complete=false` を見れば、`observedCount` の値に関係なく「全部は成立していない」と分かります。
- **`complete=true` は「全部に答えが返った」という意味だけ**です。世界がそうなっている証明ではありません。両参照の `notes` にそう書いています。
- **これは CON-12d の「partial/unknown を保持する」の、少なくとも前半（partial）を満たします。** unknown（失効・不明）の表現は別で、この記録では扱っていません。

## 5. この記録が示さないこと

- **unknown の表現**。失効や unknown を `complete` とは別に表す語は足していません。
- **HTTP 202 や Agent Tool 結果からの推測をしないこと**。それらの経路はこの参照に存在しません。
- **実 Provider の部分成功**。Provider はすべて合成です。
- 製品 CLI へ繋ぐべきか（C-1〜C-4 は未決）。
- CON-12c の `ServiceProviderSelection` 層（未試行）。

## 6. 生データ

| ファイル | 内容 |
| --- | --- |
| `run.sh` | 再現手順（観測2件・証拠2件） |
| `raw/head.txt` | 計測した版とToolchain |
| `raw/pre-status.txt` | 計測前の作業領域 |
| `raw/obs-complete.json`・`raw/obs-none.json`・`raw/ev-complete.json`・`raw/ev-none.json` | 4つの場合 |
| `raw/summary.txt` | 上の表を作る元の一覧 |

`*.err.txt` はすべて0バイトです。

## 7. 生データのSHA-256

| ファイル | SHA-256 | バイト数 |
| --- | --- | --- |
| `run.sh` | `955b590c479073fb6403c0e24ec75ab0dc86b2c36a552acfd1c49c6ed8aac614` | 2595 |
| `raw/head.txt` | `681a652660f7cecc91d2314da9fe6cbc5ef00763a0203d022e4b44890a589a59` | 73 |
| `raw/pre-status.txt` | `67faf674fbad773bdf629cab25cbd9d9a3b73defe38e200e1fc9931ef678ba3a` | 419 |
| `raw/obs-complete.json` | `87648cc70c6e7fca52b3d6a6f2b442c571ebc412fd6f8dba02e115ae8fc62f94` | 1394 |
| `raw/obs-none.json` | `3654cbabe735906e5766755cc59d21004d6fd92041bd8751a5db31718c02de6c` | 1513 |
| `raw/ev-complete.json` | `2347bfb66ebda414732e15b127df6e5673458a97108081af7cab062e2e16339a` | 1216 |
| `raw/ev-none.json` | `9b86ad1d41399b327823e1bf1edb6b2000a71fa2770f379819fde349cd188130` | 1269 |
| `raw/summary.txt` | `cc906b81864f0931bddc37163fbff8985fd468d42d64e7d849c667f7ece5cc78` | 419 |

# CON-12c の3つの識別子は、それぞれどこで守られているか（2026-11-09）

- 対象: [判断材料の第12節 CON-12c](../../workstreams/COMPOSITION_CONNECT_POINT_BRIEF_2026-11.md)、[識別子を持つ層の調査](../BINDING_IDENTITY_LAYERS_2026-11-09/README.md)、[Selection の digest 境界](../SELECTION_DIGEST_BOUNDARY_2026-11-09/README.md)
- 位置づけ: **参照実装への試験追加と実行記録**。製品へ繋ぐべきかは示しません。
- 計測した版: `raw/head.txt`（`origin/main` `41e1f0ca` の作業ツリー＋このPRの追加）。

CON-12c は「Artifact digest・Host/Grant epoch・Source Scope が変わったら旧 Binding を再利用しない」を求めます。前の記録は **digest** を `servicehost` の比較で確認しました。この記録は、**残る2つがどこで守られているか**を確かめます。

## 1. 3つの識別子の enforcement point

```text
== where digest is compared ==
  internal/servicehost/service_carrier.go:106  route.Selection.ArtifactDigest != lease.Binding.ArtifactDigest

== where scope is compared ==
  internal/composition/selection.go:44   p.Scope != scope          （Select）
  internal/composition/selection.go:121  p.Scope != s.Scope        （validateSelectionLocked）
  internal/composition/proposal.go:186   p.Scope != req.Scope

== where registry epoch is compared ==
  internal/composition/selection.go:117  s.Epoch == 0 || r.epochs[s.ProviderID] != s.Epoch

== a Grant epoch anywhere in Go source ==
  (none)
```

## 2. 観測（scope と revision の反例）

```text
--- PASS: TestASelectionWhoseScopeDoesNotMatchIsRefused
--- PASS: TestTheMatchingScopeStillRuns
--- PASS: TestASelectionWithAStaleRevisionIsRefused
```

- **scope を登録時と変えた Selection** は、**carrier へ入る前に**拒否されます（`ran=0 refused=1`、Process 0回）。
- **revision を進めた Selection** も同じく拒否されます。
- **一致する scope は通ります**（対照）。

## 3. この観測が示すこと

- **`Source Scope` は守られていますが、`servicehost` の比較項ではありません。** 一段下の `composition.validateSelectionLocked` が **`Acquire` ごとに** `p.Scope != s.Scope` を照合します。**層が違うだけで、抜けではありません。**
- **`Selection.Epoch` は registry epoch です。** `r.epochs[s.ProviderID] != s.Epoch` で照合されます。
- **`Host/Grant epoch` は、どの層にも存在しません。** `GrantEpoch` に類する識別子は Go の実装に1つも無く、試験に `TestAuthorizationGrantEpochProvenance` があるだけです。**つまり CON-12c の3つのうち、Grant epoch は「守られていない」のではなく、比較する項が実装にまだ無い** — 観測・証拠の宣言に digest が無いのと同じ形です。

## 4. この記録が示さないこと

- **Grant epoch を実装すべきか**。設計判断で、判断材料の C-1〜C-4 は未決です。
- **scope の意味が十分であること**。`p.Scope != s.Scope` という文字列一致以上の意味は評価していません。
- **実 Provider の実行**。Process は記録用です。
- **`Proposal` 経路の scope 照合**（`proposal.go:186`）の実行確認。読んだだけです。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `run.sh` | 再現手順（反例試験と enforcement point の列挙） |
| `raw/head.txt` | 計測した版とToolchain |
| `raw/pre-status.txt` | 計測前の作業領域 |
| `raw/tests.txt`・`raw/tests-summary.txt` | 反例試験の出力と要約 |
| `raw/anchors.txt` | 識別子ごとの比較箇所と、Grant epoch の不在 |

## 6. 生データのSHA-256

| ファイル | SHA-256 | バイト数 |
| --- | --- | --- |
| `run.sh` | `d79474ec66d887973f15b71b3d72b31ba70565bf0b41cd94141b8a73dca51459` | 2109 |
| `raw/head.txt` | `2a79ca6f9f0471492c6663c2fd3d82c3003d533a77072246e8b69b042c1a4996` | 73 |
| `raw/pre-status.txt` | `c1ce1f600d170693bf8e3ad347729068a1712de98710f39b262d0607831f6ed2` | 193 |
| `raw/tests.txt` | `a1985acb94218819fa2a48eb1cd2417dd7ea5ca319ff3683e9bcf7da428a22a9` | 478 |
| `raw/tests-summary.txt` | `decf53d030bf6cba8cad886fbd847769fee7c15e1f6116f31415954d06baa6e7` | 376 |
| `raw/anchors.txt` | `eca423b1df722600c9d293bfdd0f682bee855d7400e5e4fa8e5b62f5f8803db3` | 1413 |

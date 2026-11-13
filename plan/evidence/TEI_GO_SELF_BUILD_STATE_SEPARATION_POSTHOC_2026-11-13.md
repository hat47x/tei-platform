# 条件5のstate分離に、entry側の事後検査を足した（S1）と、その限界

[#4486](https://github.com/hat47x/tei-platform/issues/4486) の実装契約 S1・S3 を実装しました。**`l2b.claimed` の意味は変えていません。**変わったのは、**分離の根拠が「childの申告」だけではなくなった**ことと、**その根拠が報告に出る**ことです。

## 実装したこと

| 契約 | 内容 |
| --- | --- |
| **S1** | 2回目のbuildの後、**entry が** (a) 渡した state ディレクトリが実際に作られたか、(b) 1回目の state の一覧が2回目の前後で変わっていないか を見る。満たさなければ `unperformed` |
| **S3** | `artifactReproducibility.stateSeparation` に `basis`（`child-statement` / `entry-post-hoc`）と `scope` を出す |
| **S2** | **実装していません。** Linux限定の namespace 境界で、`host-boundary` は**予約したまま出力しません** |

## 実測（同一のfixture、3ケース）

| ケース | 終了 | `claimed` | `repro` | `basis` |
| --- | --- | --- | --- | --- |
| **同梱child（正例）** | 0 | **`true`** | **`reproduced`** | **`entry-post-hoc`** |
| **A: 1つのstateを共有し、渡された値を申告、ディレクトリは作らない** | **1** | **`false`** | **`unperformed`** | `child-statement` |
| **B: Aに加えて、渡されたパスへ空ディレクトリを作る** | **0** | **`true`** | `reproduced` | `entry-post-hoc` |

- **Aは S1 が無ければ `claimed=true` でした**（[#4299](https://github.com/hat47x/tei-platform/pull/4299) の実測と同じ構成）。**いまは落ちます。**

```text
the second build did not create the state directory this entry passed, so it wrote its state somewhere else
```

- **Bは通りません。** #4486 が予測したとおりで、**空ディレクトリを1つ作るだけで `entry-post-hoc` を満たします**。**B1の限界として固定しました。**

## 限界を記録に埋め込んだ

`scope` の文言は「何を見たか」だけを書き、**「防いだ」とは書きません**。試験（`TestStateSeparationPostHocScopeDoesNotClaimPrevention`）が `prevent`・`impossible`・`prohibit`・`denied` を含まないことを確かめます。**`entry-post-hoc` は `host-boundary` より弱い**という順序を、文言と試験の両方で固定しています。

## 既存のfixtureを直した理由

**S1を入れた時点で、`cmd/tei` の2つの試験用childが落ちました。** どちらも `-state` を**表示するだけで作っていませんでした**。

**これは試験の不備であって、S1の欠陥ではありません。** 同梱の実childは `-state` を作ります（`os.MkdirAll`）。**fixtureを実childに合わせて作り直し**、試験は元の結論（`reproduced` / `mismatched`）を保ったまま通りました。

**この修正は「S1が既存の呼び手を壊す」ことの実例でもあります。** `-state` を無視するchildを使っていた呼び手は、**Claimを失います**（`unperformed`）。契約どおりの挙動ですが、**黙って変わらないので記録します。**

## していないこと

- **S2（namespace境界）は実装していません。** `host-boundary` は予約のままです
- **Bは依然として通ります。** 本記録は改善と限界の両方を残すもので、Bを塞いだとは言いません
- 読取共有・ネットワーク・常駐Processは境界に入れていません（#4486 の未決のまま）
- `linux/amd64` の各1回です
- **`qualified` は `false`**、G4は `UNQUALIFIED` のままです

## 検証

`go vet ./cmd/tei` 通過、`gofmt -l cmd/tei` 空。新規試験5件と既存の再現性試験が通過。上の3ケースを実測。

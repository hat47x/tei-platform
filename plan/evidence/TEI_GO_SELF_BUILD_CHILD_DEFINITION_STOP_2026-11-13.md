# 計画と違うdefinitionを読むと、childは成果物を作る前に止まる（§4.36の実測）

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-C0 §4.37 の表で、**最後まで未測定だった1行**を測りました。前の記録で「未測定（child側の停止。§4.36）」としていた行です。**これで表の7行すべてに実測が付きます。**

## 実測（同梱childを直接駆動）

**この行は entry からは観測できないため、child を直接駆動して測りました。** 理由は後述します。

| 場合 | 終了コード | 成果物 | `definitionMatchesPlan` |
| --- | --- | --- | --- |
| **計画が名指すdefinition** | **0** | **書かれた** | **`true`** |
| **1行追記した別のdefinition** | **1** | **書かれていない** | 報告なし（stdout 0バイト） |
| **計画がdigestを持たない** | **1** | **書かれていない** | 報告なし |

2番目の stderr 全文：

```text
the definition read (sha256:c2cd7f594e6d5936499adba7e02cc75310ecda7c4868db8798739b323dcba1ac) is not the one the plan names (sha256:9d26ce5b2f549744e5541ab6af52b0d5bf720ddd171dd259fa8177f104331155); this run would build an artifact the plan does not describe
```

**両方のdigestを名指しし、成果物を作る前に止まっています。** 制御では `definitionBytesDigest` が計画の値 `sha256:9d26ce5b…` と一致していました。

## 測って分かった、表より細かいこと

### 3番目は child の比較に到達しない

計画のdigestを空にすると、止まる場所は child の比較分岐ではなく**計画の検証**でした。

```text
plan does not validate: build source definition digest must use sha256:<64-hex>
```

**child のコードには「計画がdigestを持たない場合も停止」という分岐がありますが、検証を通った計画ではその分岐に到達しません。** 二重の防御であり、**外側（検証）が先に止めます**。

### この行は entry 経由では再現しない

`tei self-build` は計画を**受け取らず**、`-build-catalog`・`-build-source-root`・`-intent-file` から**その場で導出**します。**計画とsourceは同じ瞬間・同じrootから作られるため、両者が食い違う状態が entry の経路では発生しません。**

**この行が成り立つのは、計画が別の経路から child へ渡された場合です。** §4.37 の「検査で止められないもの」4（childが自分で止まる。止まらないchildはそもそもこの検査を実装していない）と一致します。

## §4.37 の表：7行すべてに実測が付いた

| 行 | 実測 |
| --- | --- |
| 宣言と違うバイトを読んだと申告する（条件2） | [止まる](TEI_GO_SELF_BUILD_STOP_TABLE_MEASURED_2026-11-11.md) |
| 計画と違うfingerprintを申告する（条件3） | [止まる](TEI_GO_SELF_BUILD_STOP_TABLE_MEASURED_2026-11-11.md) |
| 成果物と違うdigestを申告する（条件4） | [止まる](TEI_GO_SELF_BUILD_ARTIFACT_DIGEST_STOPS_2026-11-11.md) |
| 同じバイトを二回出さない（条件5） | [止まる](TEI_GO_SELF_BUILD_REPRODUCTION_MISMATCH_SCOPE_2026-11-13.md) |
| 二回の間に実行物を差し替える | [止まる／プロキシでは止まらない](TEI_GO_SELF_BUILD_CHILD_IDENTITY_SCOPE_2026-11-10.md) |
| `-state`を無視して同じstateで二回buildする | [申告が正しければ止まらない](TEI_GO_SELF_BUILD_STATE_SEPARATION_SCOPE_2026-11-11.md) |
| **計画と違うdefinitionを読んでbuildする** | **本記録：止まる（child側。entryからは観測できない）** |

**7行のうち、entry が自分で観測できるのは5行です。** 残る2行は child 側（builder同一性のプロキシ・definition照合）で、**entry からは「childが止まった」ことだけが見えます。**

## 何を言っていないか

- **entry 経由では測っていません。** 経路上に食い違いが発生しないため、child を直接駆動しました
- **`-state` の行は「止まらない」ままです**
- `linux/amd64` の一回の観測です
- **`qualified` は `false`** です

# Capability の outcome が参照グラフに現れていなかった（2026-11-09、第28ラウンド）

- 対象: [#3884](https://github.com/hat47x/tei-platform/issues/3884)、[#3419](https://github.com/hat47x/tei-platform/issues/3419)、[#3420](https://github.com/hat47x/tei-platform/issues/3420)。
- 実行日: 2026-11-09（Asia/Tokyo）。
- 状態: **read-only の実測と、派生グラフの欠落の修正**。

## 1. 目的

本ドッグフーディングの目的は「生成AIがTEIを用いてTEI自身の設計を効果的に行えること」の検証である。このラウンドは、**CLI だけで設計上の問いに答えられるか**を、実際の設計（認証→認可）で試した。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `8974d22c9d32facb2f88dd55c29b114d6e01ddc5`（`origin/main`） |
| CLI実行物（修正前） | `sha256:0b7102002125129696252b53f817cdb481eaba7c6a6b733ce302e419321de3d5` |
| CLI実行物（修正後） | `sha256:fe609c54e0abcc2d921c9512b9cb59bc20eb03a8e97ab9500c6635d12498ce17` |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.27.3 linux/amd64`（gofmt は 1.26.10） |

## 3. 観測

### 3.1 まず設計の問いを CLI に投げた

`reference/workspace/semantic/security/` の宣言:

```tei
capability security.authenticate {
    input login_id std.identifier
    input credential security.credential
    outcome authenticated security.authenticated_subject
    outcome denied security.authentication_denial
}

capability security.authorize {
    input subject security.authenticated_subject
    ...
}
```

「認証の結果を認可が使う」設計である。

### 3.2 `inspect` は outcome を知っている

```json
"outcomes": [
  {"name": "authenticated", "payload": "security.authenticated_subject"},
  {"name": "denied", "payload": "security.authentication_denial"}
]
```

### 3.3 `trace` は outcome を出さない

```text
tei trace -subject-kind semantic-capability -subject-id security.authenticate -mode full
total: 2
   relation= input  to= std.identifier
   relation= input  to= security.credential
```

outcome が2つあるのに `input` の2本だけ。

### 3.4 そのため連鎖が「切れて」見える

```text
tei trace -subject-kind semantic-data -subject-id security.authenticated_subject -direction incoming
total: 1
   semantic-capability security.authorize  - input ->  security.authenticated_subject
```

`authorize` が消費することは見えるが、`authenticate` が生産することは見えない。**読み手には「誰も作らないデータを要求している」ように見える**（私も最初そう読んだ）。

## 4. 原因

`internal/index/index.go` の派生処理が、Capability について `input` と `output` の辺は作るが **`outcome` の辺を作っていなかった**。

```go
for _, in := range c.Inputs {
    ... Relation: "input",  To: in.Type
}
if c.Output != "" {
    ... Relation: "output", To: c.Output
}
// c.Outcomes を走査していない
```

`internal/semantic/model.go` の `Capability` は `Outcomes []Outcome` を持ち、`inspect` はそれを返す。**同じ CLI の中で `inspect` は知っていて `trace` は知らない**状態だった。

## 5. 修正

`internal/index/index.go` の Capability 走査に outcome の辺を追加した。

```go
for _, o := range c.Outcomes {
    if o.Payload == "" {
        continue
    }
    d.SemanticDependencies = append(d.SemanticDependencies, SemanticDependency{
        FromKind: "semantic-capability", From: c.ID,
        Relation: "outcome", ToKind: "semantic-element", To: o.Payload,
    })
}
```

payload が空の outcome は辺を作らない（解決先が無いものを「未解決」として報告しないため）。

## 6. 修正後の実測

```text
tei trace -subject-kind semantic-capability -subject-id security.authenticate -mode full
total: 4
   resolved outcome -> security.authenticated_subject
   resolved outcome -> security.authentication_denial
   resolved input   -> std.identifier
   resolved input   -> security.credential

tei trace -subject-kind semantic-data -subject-id security.authenticated_subject -direction incoming
total: 2
   security.authorize    - input
   security.authenticate - outcome
```

**連鎖が両側から見えるようになった。** 生産者（`authenticate`）と消費者（`authorize`）が同じデータで繋がる。

`reference/workspace` の `source-index` は 106 件のまま（`source-index` はファイル由来の参照を数えるもので、グラフの辺ではない。outcome は .tei 内の宣言から派生するため、ファイル参照は増えない）。4つの outcome 宣言がすべて辺になった。

## 7. 試験

`internal/index/capability_outcome_test.go`。

| 試験 | 内容 |
| --- | --- |
| `TestCapabilityOutcomesBecomeEdges` | `security.authenticate` から `security.authenticated_subject` への `outcome` 辺が存在する |
| `TestOutcomeWithoutPayloadProducesNoEdge` | payload の無い outcome は辺を作らない |

変異（派生を削る）で `no outcome edge from security.authenticate` が出ることを確認した。

`go test ./internal/index ./internal/modelref/... ./internal/operations ./cmd/tei -count=1` はすべて ok（0.01s / 0.54s / 0.06s / 0.01s / 1.25s / 141.3s）。gofmt・vet の指摘なし。

## 8. 互換性

**意図的な挙動変更を含む。**

- `trace` の `total` が増える（capability の outcome 数だけ）。
- `sourceIndexFingerprint` が変わる（Index の JSON に辺が増えるため）。**World は変わらないので `worldFingerprint` は不変**。
- そのため、`trace` の `result.snapshotFingerprint`（World + Source Index + graph の合成）は変わり、**保存済みの継続トークンは失効する**。これは正しい方向（以前は辺が欠けていた）。

## 9. 読み取れないこと（NOT RUN）

- `outcome` の名前（`authenticated` / `denied` / `allowed`）を辺に載せていないことの是非。現状は「どの capability がどの型を結果として生むか」だけを持ち、結末名は `inspect` の `canonical.outcomes` で見る。設計上どちらが良いかは #3419 の判断。
- outcome の payload が未宣言の型を指す場合に `trace` が `invalid-target` を返すか（`output` と同じ経路を通るはずだが未確認）。
- 実DB・実Provider・実Host、HTTP・Desktop、意味の受理・権限・配備。

## 10. 位置付け

「CLI だけで設計の問いに答えられるか」を実際の設計で試し、**能力の連鎖が辿れない**ことを見つけて直した。`inspect` と `trace` が同じ語彙（`outcome`）を扱うようになり、生成AIが設計を読むときに「生産者不在」を誤検知しなくなった。

これは第25〜27ラウンドの「モデルにあるが CLI から見えない」型と同じであり、**TEI 自身の設計を CLI で検証する能力**に直結する欠陥だった。

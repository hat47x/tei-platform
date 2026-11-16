# rea-intake の候補・carrier・coverage を実測に合わせる（2026-11-17）

## 対象

- 実装: `cmd/tei/cli_three_element_rea.go`
- 対象版: `fc3fe59da`（`main`）
- 環境: Windows、Go `1.26.10`

## 直した3つの欠落

`main` で `cmd/tei` の REA intake 系テストが3件落ちていました。いずれも「評価器・監査器・比較器は既に在るのに、`rea-intake` の応答がそれらを使っていない」型です。

### 1. `threeElementCoverage` が応答に無い

`EvaluateThreeElementCoverage` の呼出し元は**テストだけ**でした。応答は Function / Business について何も言わず、読む側は nil で落ちていました（`panic: interface conversion`）。

### 2. エッジが候補にならず、occurrence が区別できない

応答の `candidates` は**固定3件**（`candidate-1`〜`3`、すべて `data-element`）でした。外部グラフのエッジは候補にならず、同じ端点・同じラベルの2本のエッジを区別できませんでした。

`-model` が与えられたときだけエッジから `cross-reference` 候補を作ります。**`-model` 無しでは従来どおり3件**です（既存の `TestThreeElementReaIntakeMinimalAttempt` がこれを固定しているため、互換性を保つ判断をしました）。

### 3. carrier が固定のパス一覧だった

`"carrier": map[string][]string{"missing": {...}}` という**パスの配列**で、事実ごとの `present`/`total` を持っていませんでした。`AuditCarrierFit` が同じ問いに答えられるので、それへ置き換えました。`packetCandidateSet` で packet の候補を型付き集合へ射影し、`ExternalIdentity`・`ExternalOccurrenceId`・`Direction` を埋めています。

### 4. 重複 edge ID を拒否する

同じ `edge_id` が2回現れるグラフは曖昧です（1つの候補 ID が2つの観測を指す）。`EDGE_IDENTITY_DUPLICATE` で**終了コード2**として拒否します。どちらかを選んで通しません。

## 実装で踏んだ誤り（記録）

- **順序の誤りを2回**: ①adoption ブロックが候補生成より前にあり、`after` に cross-reference が入らなかった。②carrier の計算が候補追加より前にあり、`total:3`（固定候補の数）で監査された。どちらも実測の値（`present:0 total:3`）から気づき、ブロックを後ろへ移しました。
- **`map` の比較**: `peer == edge` は Go で不可（map は nil としか比較できない）。`peer.ID == edge.ID` の比較に直しました。
- **`StructureCandidate` のフィールド名**: `ID` ではなく `CandidateID`。
- **map リテラル内での自己参照**: `"carrier": f(packet)` は `packet` 定義中なので不可。リテラル後に代入する形にしました。

## 実行

```powershell
go test ./cmd/tei -run 'TestREAIntake|TestThreeElementReaIntake' -count=1
go test ./internal/reacandidate ./internal/threeelement -count=1
go test ./cmd/tei -count=1
go test ./tools/repository-plan-check -count=1
```

## 観測

- `TestREAIntakeParallelRelationsStayUnacceptedInPreview` 成功（失敗行が 36→115→133→137→149 と進み、各段で次の欠落が判明した）
- `TestREAIntakeCompleteGraphDoesNotProveFunctionOrBusiness` 成功
- `TestThreeElementReaIntakeMinimalAttempt` 成功（**3件固定を壊していない**）
- `internal/reacandidate`・`internal/threeelement`・`repository-plan-check` 成功

## 読み取れないこと

- **`-model` 無しでエッジ候補を作るべきか**。現在は作りません（既存テストの3件固定を尊重）。これは設計判断として残ります。
- adoption の `before`/`after`/`diff` は簡易な射影で、`PreviewAdoption` の完全な意味（未解決端点の報告、外部ノードのグルーピング等）を使って**いません**。
- 実 REA の出力での確認（合成 fixture のみ）。

## 位置付け

参照実装の修正記録であり、製品の受入や本番適格性を示しません。`main` の赤3件を、既存の評価器・監査器へ応答を繋ぐことで解消した記録です。

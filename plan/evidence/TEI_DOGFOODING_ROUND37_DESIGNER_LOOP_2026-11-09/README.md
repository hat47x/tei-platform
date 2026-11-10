# 設計者ループは動くが、`-qualification-requirements` の help が誤っていた（2026-11-09、第37ラウンド）

- 対象: [#3943](https://github.com/hat47x/tei-platform/issues/3943)、[#3419](https://github.com/hat47x/tei-platform/issues/3419)。
- 実行日: 2026-11-09（Asia/Tokyo）。
- 状態: **設計者ループの検証と、CLI の help 文言の修正。**

## 1. 目的

本ドッグフーディングの目的は「生成AIがTEIを用いてTEI自身の設計を効果的に行えること」の検証である。このラウンドは、**実際に設計を1つ追加する**作業を通して、CLI だけで完結するかを測った。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `ccfd9c927`（`origin/main`） |
| CLI実行物（修正前） | `sha256:82321c0864febbd73723759d0ff18038f16c33f39665f54347c3026b85fd0d0d` |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.27.3 linux/amd64`（gofmt は 1.26.10） |

## 3. 設計者ループは動いた

`reference/workspace` の写しに、`security.authenticate` の outcome を入力に取る `security.logout` を追加した。

```tei
namespace security

capability security.logout {
    input subject security.authenticated_subject
    outcome logged_out std.boolean
}
```

| 段階 | CLI の応答 |
| --- | --- |
| `validate` | `ok:true`（新規の診断なし） |
| `inspect` | `inputs` と `outcomes` を返す |
| `trace` | `input -> security.authenticated_subject`（resolved）、`outcome -> std.boolean`（resolved） |

**設計を1つ足して、その繋がりを確認するまでが CLI だけで完結した。**

### 3.1 誤りは検出された

| 誤り | `validate` | `trace` |
| --- | --- | --- |
| outcome の payload が存在しない型（`security.session_token`） | `UNKNOWN_SEMANTIC_CAPABILITY_OUTCOME_PAYLOAD`（error） | 行にならず `projectionUnresolved` に `unresolved-semantic-element:security.session_token` |
| 同じ capability を2ファイルで宣言 | `DUPLICATE_SEMANTIC_ELEMENT`（error） | `SUBJECT_AMBIGUOUS`（exit 1）「2 declarations」 |

どちらも検出され、`trace` は曖昧なまま答えを返さない。**壊れたモデルに対して正しく失敗する。**

### 3.2 影響範囲の質問にも答えられた

`security.authenticated_subject` を消したら何が壊れるかを `trace -direction both` で引くと、参照元が2件（`authenticate -outcome->` と `authorize -input->`）返る。設計変更の影響を CLI で確認できる。

## 4. 見つけた欠陥: help が「JSON」としか言わない

`knowledge-qualify` を設計検証に使おうとして、**3回続けて失敗した**。

```text
usage: tei knowledge-qualify [options]
  -qualification-requirements string
    	knowledge qualification requirements JSON
```

「JSON」とあるので JSON 文字列を渡すと:

```json
{"code": "KNOWLEDGE_BUILD_FAILED",
 "message": "open {\"qualifications\": [...]}: no such file or directory"}
```

**ファイルパスだと分かるのは、実行時エラーが出てから。**

### 4.1 スキーマも広告されていない

正しい形は `internal/knowledgecontext/qualification.go` の `QualificationRequirements`（`expectedKind`・`application`・`requiredDependencies`）。`-h` にも `commands` にも出ていない。`commands` の `argumentValues`・`argumentType`・`argumentDefaults` を抽出すると `{}` で、`-3242` の方針（閉じた集合のみ広告する）では対象外と判断されたと見られる。

私はまず `{"qualifications": [...]}` と書き、次に `{"expectedKind": "semantic-data"}` を当て推量で見つけた。

### 4.2 符号も誤解を招く

引数の型違い（JSON 文字列を渡した）に対して `INVALID_ARGUMENT` ではなく `KNOWLEDGE_BUILD_FAILED` が返る。「モデルが壊れている」と読める。

## 5. 修正

`-h` の文言を直した。これだけで私の3回の失敗は0回になる。

```diff
- knowledge qualification requirements JSON
+ path to a file holding the knowledge qualification requirements JSON
```

`knowledge-qualify` と `knowledge-review-handoff` は**同じフラグ定義を共有**している（`cmd/tei/main.go:180` の1箇所）ので、両方に効く。変異試験で片方を戻すと**両方のサブテストが落ちる**ことを確認した。

## 6. 追加した試験

`cmd/tei/qualhelp_test.go` の `TestQualificationRequirementsHelpSaysItIsAPath`。両コマンドの `-h` に `-qualification-requirements` と `path to a file` が含まれることを実バイナリで検査する。

既存の `buildCLIForLifecycle` を再利用し、新しいビルド補助は作っていない。

## 7. 起票しなかった / 残したもの

- スキーマの広告（`argumentType: file-path` 等）と符号の分離は、**判断を伴うため #3943 に材料として残した**。実装は help 文言のみ。
- `knowledge-review-handoff` の `-qualification-requirements` も同じ欠陥だったが、共有定義のため1箇所の修正で直っている（試験で両方を確認）。

## 8. 読み取れないこと（NOT RUN）

- `knowledge-qualify` の判定内容（`status: applicable` の意味、`requiredDependencies` を指定した場合の挙動）。今回は引数の受け付けと応答の形だけを見た。
- スキーマを `commands` で広告する場合の後方互換（#3242 の方針との整合）。
- 実DB・実Provider・実Host、HTTP・Desktop、意味の受理・権限・配備。

## 9. 位置付け

**設計者ループ自体は機能している。** 能力を1つ足し、繋がりを `trace` で確認し、誤りを `validate` と `trace` が検出し、影響範囲を `trace -direction both` で引ける。これは目的の中心部分が満たされている証拠である。

一方で、**補助コマンドの入口でつまずいた**。`knowledge-qualify` は引数の形が help から分からず、3回失敗した。設計判断そのものより、道具の入口の説明不足が作業を止める。今回の修正は1行だが、その1行がなければ CLI だけで完結しない。

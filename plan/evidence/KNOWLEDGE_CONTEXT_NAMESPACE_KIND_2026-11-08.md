# `knowledge-context` が catalog の名前空間 kind（`semantic`）を受け付けない（記録）

- 記録日: 2026-11-08
- 対象: `internal/knowledgecontext/context.go` の `semanticElementKind`
- 関連: #3242 の後続（`tei commands` の引数値）。実測は `reference/workspace`。

## 観測

`model catalog` は名前空間の subject を `kind: "semantic"` で返す。`inspect` と `trace` も同じ名前を使う。一方 `knowledge-context` の `semanticElementKind` は `semantic-namespace` だけを名前空間として扱っていた。

```text
tei knowledge-context -workspace reference/workspace -subject-kind semantic -subject-id sales
# -> ok:true, result.semanticElement なし, result.unresolved なし

tei knowledge-context -workspace reference/workspace -subject-kind semantic-namespace -subject-id sales
# -> ok:true, result.semanticElement = namespace:sales
```

catalog の値をそのまま渡した呼び手は、`ok:true`・`unresolved` 空・`semanticElement` 無しを受け取る。これは「その名前空間に定義が無い」とも読め、kind 名が通らなかったことを示さない。

## 変更

- `semanticElementKind` の `case "semantic"` を `semantic-namespace` と同値にした（名前空間として `namespace` を返す）。既存の `semantic-namespace` も受け付けるので、後方互換である。
- 応答の `result.subject.kind` は指定値をそのまま返す（名前を書き換えない）。
- 未知の kind は従来どおり解決しない（`semanticElement` を推測しない）。

## 検証

- `internal/knowledgecontext/namespace_kind_alias_test.go`: 両方の kind が `namespace:sales` を解決し、未知 kind は解決しないこと。
- `cmd/tei/cli_knowledge_context_kind_test.go`: catalog の `entries[].subject.kind` を読み、その値と `semantic-namespace` の両方が実 CLI で `semanticElement` を返すこと。未知 kind は `semanticElement` 無しで subject を保持すること。
- 負例: `case "semantic"` を外すと両試験が失敗する（`kind "semantic" did not resolve the namespace`）。
- `go test ./internal/knowledgecontext -count=1`、`go test ./cmd/tei -run 'TestKnowledge|TestDiscovery|TestContractDocument' -count=1`、`go test ./tools/repository-plan-check -count=1` は成功。

## 主張しないこと

- この修正は kind 名の受理だけである。`knowledge-qualify` や `knowledge-review-*` の `-subject-kind` が同じ集合を受理するかは確認していない（`knowledgecontext` の `QualificationRequirements.ExpectedKind` は呼出し元が指定する値で、CLI 側に閉じた集合が無い）。
- 名前空間以外の kind について、catalog・inspect・trace・knowledge-context の名称が一致するかは全件を突き合わせていない。

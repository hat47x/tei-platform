# AC-5の「多重一致」が参照実装に無いことの確認（2026-11-10）

## 対象

- 読取り対象: `internal/command/reference_store.go`、`internal/command/reference_failure.go`、`internal/command/lifecycle.go`、`internal/semantic/capabilityinvoke/invocation.go`
- 対象版: `7a9ff54d4`（`main`）
- 環境: Windows、Go `1.26.10`

## 背景

`AC-5`の要求（`RQ-018`）は「能力の結果を、業務上の結末（Outcome）と実行の失敗（error）に分け、**適用・不存在・競合・多重一致・拒否・不明・未実行**を区別して、件数へ潰さない」です。

受容トレースの残る未確認に「多重一致の表現」が残っていたため、実装状況を実測しました。

## 実測した事実

### 1. 参照Storeはキーで一意化するので、複数一致が起きない

`MemoryStore`の記憶域は`map[Data]map[key]record`です。`Get(data, key)`は`(record, bool)`を返し、**複数一致を返す口がありません**。

```go
func (s *MemoryStore) Get(data core.CanonicalID, key string) (map[string]string, bool) {
	record, ok := s.data[data][key]
	...
}
```

同じキーの`create`は`record-exists`、`update`/`delete`での不在は`record-missing`で拒否されます。キーが一致する限り1件しか存在し得ないため、**「同じ条件に2件以上一致した」という状態をこのStoreは表現できません**。

### 2. 候補キー（一意制約）の重複は`value-rejected`として扱われる

```go
return referenceFailure(ReferenceValueRejected, "candidate-key", "candidate key %s is duplicated", candidate.ID)
```

「一意であるべき値が重複した」は`value-rejected`（結末`value-rejected`）で、多重一致とは別の区別です。

### 3. 参照実装に multiple / ambiguous 系の拒否符号は0件

`reference_failure.go`の拒否符号は5つ（`record-missing`・`record-exists`・`value-rejected`・`rule-rejected`・`undeclared`）で、`multiple`・`ambiguous`に相当する符号はありません。

### 4. 他の6区別は実装されている

| AC-5の区別 | 実装 |
| --- | --- |
| 適用 | `Applied`（適用した操作）、`Persistence`（`not-applicable`/`not-observed`/`receipt-observed`/`resource-observed`） |
| 不存在 | `record-missing` → 結末`not-found` |
| 競合 | `record-exists` → 結末`already-exists` |
| 拒否 | `value-rejected`・`rule-rejected` → 同名の結末 |
| 不明 | `Lifecycle`の`StageUnknown`、`EffectUnknown` |
| 未実行 | `StageDispatched`（送ったが結末を観測していない）、`HOOK_..._NOT_EVALUATED`系（フック側） |
| 多重一致 | **無い** |

## 判断

**多重一致の拒否符号を発明しませんでした。** 理由は二つです。

1. 参照Storeの構造上、キー一致で1件しか存在しないため、この符号を返す状況が作れません。返せない符号を足すと、試験できない主張になります。
2. 「何をもって多重一致とするか」（同じ条件に2件、候補キーの重複、関連の複数一致）は意味の設計であり、`records-count` へ潰さないための区別の粒度を決めるのは実装ではありません。

代わりに、**残っていないこと**を記録します。AC-5の残る未確認に「多重一致の表現」が正しく残っていることを確認しました。

## 読み取れること

- AC-5の7区別のうち6つは参照実装にある。
- 多重一致は、参照Storeのキー一意性により**表現できない**（未実装ではなく、この実装では生じない）。
- 候補キーの重複という近い概念は`value-rejected`で区別されている。

## 読み取れないこと

- 実Provider（PostgreSQL等）で「同じ条件に複数行一致した」をどう区別するか。参照Storeと違い、実DBでは起こり得ます。
- 参照入力（`reference` Pattern）の lookup が0件・複数件のときの区別。今回はコマンド実行のStoreのみを見ました。
- 意味の設計として、多重一致をどの結末名で表すか。

## 位置付け

参照実装の読取り記録であり、製品の受入や本番適格性を示しません。`AC-5`の「多重一致の表現」が**参照実装では生じない構造**であることを実測で確かめ、符号を発明しなかった判断を残す記録です。

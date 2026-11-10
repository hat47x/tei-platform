# TEI自身をビルドする計画の形を測る

§4.38は「五条件を満たしてTEI自身を自己構築することは今はできない。理由はflagの不足ではなく、**計画を解釈してGoをコンパイルするbuilderが存在しない**こと」で終わりました。本記録は、**そのbuilderが受け取る計画の型が既に存在するか**を測ったものです。

## 測ったこと

次の計画を`plugin.BuildPlan`として解析し、`Validate()`と`LockBuildOutput`に通しました。

| 欄 | 値 |
| --- | --- |
| `formatVersion` | `1` |
| `plugin` | `{id: "tei.command.tei", version: "0.0.0"}` |
| `pluginDefinitionFingerprint` | 64桁の16進（`sha256:`の接頭辞**なし**） |
| `sources[0]` | `{kind: "go-package", id: "tei", locator: "cmd/tei", digest: "sha256:…"}` |
| `recipe` | `{kind: "go-command", entrypoint: "./cmd/tei"}` |
| `outputKind` | `runtime-artifact` |
| `target` | `host/linux/amd64` |

## 結果

```
Validate: <nil>
Fingerprint: 0d2558db470fd1fa... err: <nil>
LockBuildOutput err: <nil> hasPlanFP: true
```

**型を足す必要はありませんでした。**

| 分かったこと | 意味 |
| --- | --- |
| `recipe.kind: "go-command"`が通る | recipe kindは自由文字列で、catalogの`recipeKinds`が宣言する |
| `outputKind: "runtime-artifact"`が通る | commandの成果物は既存の種別で表せる |
| `LockBuildOutput`が通る | **commandのためのlockの種類も既にある**（`Lock`は`runtime-artifact`専用として検証される） |
| ディレクトリのsource（`cmd/tei`）が型として通る | `DirSourceResolver`がディレクトリを扱えるのと揃っている |

## 途中で分かったこと（記録）

1. **`pluginDefinitionFingerprint`は`sha256:`の接頭辞を付けてはいけない。** `validateHexFingerprint`は**64桁の16進のみ**を受け付けます（`sha256:`を付けると「must contain 64 hex characters」）。sourceの`digest`や`artifactDigest`は`sha256:`付きなので、**同じ計画の中で接頭辞の有無が混在する**。読み違えやすいので記録します。
2. **`Validate()`は他の欄が揃っていないと、指摘する順序が一定でない。** 最初は`sha256:x`という短い値で試して同じ場所で止まり、値の形式の問題だと分かるまでに時間がかかりました。

## この記録が言っていないこと

- **builderの実装**。§4.39は形と順序を決めただけで、まだ無い
- source closureの照合方法（`DirSourceResolver`を使う案は§4.39に書いたが、未実装）
- 実際の`go build`がこの計画から回ること（**計画の型が通る**ことと、**buildが通る**ことは別）

## 検証

- 上の3行の出力は実行して得た値
- `go test ./tools/repository-plan-check -count=1`：PASS

**NOT RUN**：CI（文書のみ）。

## 参照

- 設計：[`SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md`](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md) §4.39
- 能力の一覧：[同 §4.38](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)・[`TEI_GO_SELF_BUILD_CHILD_CAPABILITY_MATRIX_2026-11-10.md`](TEI_GO_SELF_BUILD_CHILD_CAPABILITY_MATRIX_2026-11-10.md)

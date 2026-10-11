# CLI C2 Disclosure：隔離Reference

[Issue #1178](https://github.com/hat47x/tei-platform/issues/1178)の開示境界を実証する候補です。設計上の位置づけは[共通Disclosure境界設計](../../plan/design/CLI_C2_DISCLOSURE_BOUNDARY_DESIGN.md)を参照してください。

**このディレクトリは、第二Policy Engineでも、公開安全性が証明されたCLIでもありません。** 既存の`security.authorize`へ接続する想定の`Authorizer`を呼び、すでに評価された判定に従って公開Projectionを組み立てるだけです。

```sh
GO111MODULE=off go test -count=1 project.go project_test.go
GO111MODULE=off go vet project.go project_test.go
```

合成Authorizerによるテストでは、Console・Web・Desktop・外部LLMの共通意味契約、拒否と対象不存在の同一公開表現、partialでの非公開Source・関連・unresolvedの非反射、cross-tenant、別Caller、別Workspace、別LLM送信先、Policy revision変更、revocation/expiry、継続の失効、summary→fullの再認可、Source等のExposure Manifest受渡しを圧力として扱います。

注意点：

- `Manifest`に含む全Resourceを既存認可Providerが調べ、**そのfacetに属する全要素が許可された**場合に限ってSectionを許可する必要があります。本Referenceの合成callbackは実Policy判定をしていません。
- `canonical`は全文JSONの一括許可のみ。Source/related等もfacet単位の一括許可であり、個別edgeや部分fieldの細粒度認可は未実装です。
- `Raw`は安全な公開結果ではありません。出力前の読取権限と、外部egress直前の認可・監査はHostが行います。
- `Receipt`は内部の比較用情報であり、bearer tokenではありません。実Hostでの偽造防止、失効、audit write、cache隔離を保証しません。
- 非許可とunknownの公開JSONが一致しても、timing・ファイルアクセス・log・network上のSide Channelが閉じたことにはなりません。
- 実C1 `operations`、Web/IPC/LLM transport、実IdP、Source Provider、`security.authorize`、実監査ストアとの統合はまだ行っていません。

このPR内のGoコードとテストについて、**actual exact-head full checkoutでの`gofmt`・Go test・Repository Healthはまだ未取得**です。C1 Draft PR #1172へ機能追加を強制せず、セキュリティ適合とmain統合は別途Gateで判断します。

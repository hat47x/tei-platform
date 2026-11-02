# 複数Data Provider／Persistence経路の読取り専用比較Reference

対象：[TEI #2611](https://github.com/hat47x/tei-platform/issues/2611) PM-01〜07、[Everything is Data #2619](https://github.com/hat47x/tei-platform/issues/2619)、[Data Operations #2622](https://github.com/hat47x/tei-platform/issues/2622)。

## このReferenceが表すもの

- 同じSemantic Dataから、異DB／Replica／履歴／ODataなどの複数の実現候補へ到達できることを、失われないように記録する合成比較例。
- Provider、Endpoint、Mapping、Revision、Scope、Purposeが同じ場合のみ、同一路線に複数Aliasが付くものとしてグルーピングする。別経路を等価と受理するものではない。
- 読込み／書込みの宣言上の適合をAliasごとに保持する。部分的な非対応と、ほかの経路の候補存在は同時に成立する。
- CandidateCount=1は、権限、Providerの現在利用可否、新鮮性、データの完全性、業務意味受理、実Write対象の同一性を証明しない。
- Purposeのcurrent/archive/replicaは例示ラベル。新しいTEI Core enum・正式Provider ABI・Route Resolver・Registry・共通Hostを導入する案ではない。
- 情報の欠落は対象の不存在という証明ではない。Providerへの実Query／Mutation、実Target、Receipt、独立観測の検証は別担当。

## 実行・限定証拠

リポジトリルートの予定コマンド：

    go test ./reference/provider-routes/...

同じexplain.goとexplain_test.goの実byteを隔離ローカルディレクトリへ置き、Go 1.23.2・GO111MODULE=offで実行：focused 12/12 PASS、go test -race PASS、go vet PASS。これは**合成Reference単体**の実測であり、PR headのCI／Go 1.26.10／TEI製品のDeriveOperationSurface／実Providerの実証とは異なる。

照合対象local SHA256：

- explain.go: c9d8b162ef55f3d866dae3559cf128458f72020180f0d468a7af254ea2f76c96
- explain_test.go: fd68fedd457245733b198ae68918aca359c8d3fb9a7c6a3a792edec71322d96d

## 次に検証する境界

1. 既存PR #2587のDeriveOperationSurfaceへ、同一Semantic Data・別Mapping/DB Aliasの合成Worldを渡し、Operation ID、Basis、Gaps、Conflicts、実候補を**同一試行で**照合する。これは本Referenceが直接行った試験ではない。
2. 既存World Validatorが正当に受理する多重Bindingの範囲を先に確認。モデル上許されない例は有効な入力としての成功を要求しない。
3. [#2485](https://github.com/hat47x/tei-platform/issues/2485)のAction/Capability目的と、[#2598](https://github.com/hat47x/tei-platform/issues/2598)のContext→Request対象束縛を別列で確認。入力レコードのIdentity一致からProvider一致を導かない。
4. [#2622](https://github.com/hat47x/tei-platform/issues/2622)でDBとAPI系Providerを比較する際は、NULL/Bag/集合完全性、as-of、PATCH/Replace、version、idempotency/Receipt等を**個々のProviderの実Contract**で評価する。

**ステータス**：独立合成Go単体テストのみ。Source/Meaning/Owner受理、実TEI統合、実DB/外部Host、正式API、PRのマージは未確認。

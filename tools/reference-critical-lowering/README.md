# Critical Lowering Contract Reference (#1121)

評価待ちの合成contract pressure。**TEIのproduction Compiler / Driver / Plugin Resolver / Build Artifact契約は変更しない。**実行先のバイナリやWeb UIは生成しない。

## 検証すること

- Candidate → externally verified Acceptance → Target-neutral intent → explicit Application wiring → exact Capability / Provider / Conformance / Lock参照 → Target-bound witness。
- required Context Slotの宣言がAccepted contractからProviderへ失われないこと。**実値**は現在のtrusted Hostが注入し、planには置かない。
- `build-only`依存がruntime planへ入らないことと、必要なeffectをTargetで提供できないときにfail-closedすること。
- CandidateやProviderの順序・Target・builder変更が、関係のない上流段階を不必要に無効化しないこと。
- current authorization / slot bindingは、loweringが成功してもInvocationごとにHostから再確認すること。

## 実行

Repositoryのクリーンなcheckoutから:

```bash
go test -count=1 ./tools/reference-critical-lowering
go run ./tools/reference-critical-lowering
```

これは局所的なpressureであり、正式な統合判定に必要な全件検証とは別。統合前にはexact headで必ず以下を実施する。

```bash
tools/repository-health-check.sh
```

## 検査用データの位置付け

`main_test.go`は既存`order_entry`のPattern / Data / Capability名に結び付けているが、contract ID、provider、Context Slot、receipt、LockとArtifact参照は**合成値**である。stub Trustは全検査の実Authorityではなく、これらが欠落したときのboundary処理を検証するための置換物。二つのLanguage IDは2種類の入力構造のpressureにとどまり、実Parserを実行しない。

本Referenceが実装していないもの: Accepted sourceのreload、実Pluginの解決、current permission policy、Host sidecar/context injectionの実値、secure BuildArtifactLockの内容照合、Target実行、Actual Observation、second-Target executable conformance。

ステージ契約と責務の所在、選択的再評価の前提、実Consumerで検証すべき残作業は[Critical Lowering Contract Design](../../plan/design/CRITICAL_LOWERING_CONTRACT_DESIGN.md)を参照。

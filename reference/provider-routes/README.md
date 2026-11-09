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

## Read Sliceの独立比較（RC-01〜20／追加）

`read_compare.go` は**実Providerが返したと仮定した合成Read Slice**を比較するための検証補助。経路列挙の`Explain`とは別の義務を説明する。

| 判定 | 意味 | 適用範囲 |
| --- | --- | --- |
| `fixture_consistent` | 独立Fixtureが**同一読取Cohortで比較可能**と明示し、同一Subject/Scope/Grain/Query契約・完全なbagが一致 | 有限Fixture内の整合のみ。実Provider同値やBusiness Acceptanceではない |
| `contradicted` | 上記の比較前提のもとで、完全なbagの値・NULL・出現数・識別子が異なる | 合成検証基準の反証。実DBの誤動作と同義ではない |
| `not_established` | 同じSnapshot tokenでも独立Cohort証拠がない、部分取得、Scope/Grain/Query/Subject不一致、必要な行Identityが未解決 | 比較資格の未確立。値が不一致／一致という断定を保留 |

**重要な二世界反例**：PrimaryとReplicaが同じ`SnapshotToken=s7`を持ち、同じ1行を返すとしても、**異Provider間のトランザクションスナップショットが同じことを、その文字列一致から証明できない**（#1708のread-set/freshness課題）。このReferenceは`AssertedCommonCohort`を**独立Fixtureが仮定する場合だけ** bag比較へ進む。`EstablishedByFixture`と`EvidenceLabel`はテスト前提の明示であり、真正性の検証・Provider契約・Host許可を実装するものではない。実製品がこのboolを実行資格として使うことはできない。

別の反例：同じEntity identityのレコードがJOIN結果で2回出現したとき、SQL Bagの重複を集合へ縮約すると結果が一致しているように見え得る。RC-06は重複数を比較する。`NULL`は観測済みNULL、`absent`は未取得Fieldとして区別し、**completeなRowsetでもunobserved Fieldが残れば値同値はnot_established**とする。未知の`redacted`も保留し、現Fixtureの`ReadRow.Value`は値byteの一致だけを見る。数値単位・Decimal/rounding・型付きNULL・RLS・照会predicateの厳密な意味・取得の完全性は**製品側に未接続**。

### 追加実測（2026-11-02）

隔離した実Go byteで`GO111MODULE=off go test -count=1` **合計32/32 PASS**（経路PR-01〜12、読取比較RC-01〜20）、`go test -race -count=1` PASS、`go vet ./...` PASS。Go 1.23.2/Linux。GitHubの保存Blobと一致：

- `read_compare.go`: Git blob `96a50371a7f35dd03124b6933b71d3c6ad5b0b97`、SHA256 `50f45b648bbc27bd0d6c0bf9bb1f76b86396a93d37eb6604324af537085fb340`
- `read_compare_test.go`: Git blob `29478344212e3d88e98d39759020da7dcc0423d5`、SHA256 `be67e3c844ee3c090a6d1c5f6e429da2d7de8710b75f056ed6f2dd515ed5067b`

**継続する境界**：この合成29件は実`DeriveOperationSurface`、`Query`/Function Data Context、SQL Worker・実DB Snapshot/RLS、実OData/Read Session、Provider間の実Cohort証明、実Target/Host権限、Ownerの受理を試験しない。#1708のSource実装を変更せず、問題を独立反例として次段の実Provider受入へ渡す。

### 次の合流順序（既存Owner分担）

1. #2611のPM-02〜07とOpen PR #2587の実Operation導出を同一`ir.World`で比較。合成Referenceの出力を実Productがすでに保証すると誤記しない。
2. #2622／#2506／#2586で、明示Function Context Read Footprint→Provider経路選択→実Result Sliceの`scope/grain/bag/complete/snapshot`を実データに基づき説明。
3. **Closedの#1708**を既存read-consistency M1 Referenceとして参照し、Replica／複数Providerの実共有Cohort・Snapshot token名前空間の**新しい実証要求**は#2622/#2586の実Provider受入に渡す。Closed Issueの完了を巻き戻さない。
4. #2598／#1174で読み取った対象から**実Request/Effect対象の因果接続**を実際に検査。比較が`fixture_consistent`であることを業務実行許可に使わない。

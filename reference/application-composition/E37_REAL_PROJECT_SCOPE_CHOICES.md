# E37：統合済み案件Goを複数Projectで使うとき、増える選択だけを確定する

- **管理**：[Issue #1016](https://github.com/hat47x/tei-platform/issues/1016)。
- **実在する案件ロジック**：[PR #1567](https://github.com/hat47x/tei-platform/pull/1567)でmainに統合された`plugins/reference-project-customer-advice/advice.go`の`customeradvice.Evaluate`。E33の架空のGo関数ではなく、この本物のGoパッケージを参照テストで直接実行する。
- **既存機構**：`internal/composition.Registry`、`InspectWiringSet`、`ProposalCurrent`、`reference/composition-evidence-handoff.ReconcileCurrent`。新しいProfile・Resolver・Lock・独立のProject Registryは作らない。
- **再現**：[E37 Go回帰試験](e37_real_project_scope_choice_test.go)。
 
## 前段E36と比較して確かめる問い

[E36](E36_POSTMERGE_RUNTIME_AUTHORITY_HANDOFF.md)と[BuildPlan／Plugin Lockの既存経路 #1815](https://github.com/hat47x/tei-platform/pull/1815)では、案件Goの参照実装、実バイナリとDigest、BuildTrace/Lockの整合までが別担当により進展した。それでも各案件で**どのProviderを選ぶか**は、実装の存在だけでは一意にならないことがある。E37はこの**構成選択の追加コスト**に絞る。

実際の案件用Go処理は1つだけ保持し、`case-a`と`case-b`という2つの独立した**試験用スコープ**へ、同じ説明的なContractを提供するProvider宣言を置いた。スコープは業務Tenantや本番権限主体ではない。

| 検証 | 既存機構が行うこと | 案件固有の追加判断 |
| --- | --- | --- |
| 案件Aと案件Bに同じGo助言を用意 | 異なるScopeから各Providerの構造候補を選択し、**実際の`Evaluate`を再利用** | A/Bの利用対象とScopeは案件が決める |
| 案件Aの同ContractにProviderを2件用意 | 無指定の暗黙選択を`ambiguous-providers`として拒否 | どの実装を選ぶか、必要なら**Provider IDを明示** |
| 案件BのProviderは1件のみ | 無指定でも既存Registryで一意に選択できる | 不要な案件BのProvider選択は追加しない |
| 案件AのContractを更新する | 旧A候補のcurrentが失効するがBはcurrentを維持 | Aの新Contractへの対応・互換性をレビュー |
| AからBのProviderを指定／異Scopeを一括照合 | 別Scopeへ暗黙移動・混合選択しない | 案件別に独立の検査を実施 |
| 既存Go助言が`review_suggested`を返す | 助言の実結果が得られる | 受注可否のAccepted意味へ**自動昇格しない** |
| 有効な構造候補に外部適合証拠がない | `reference-missing`として検証すべきProviderを示す | #951側で信頼されたContract・Conformance・Artifact証拠を取得する |

この試験は**ScopeごとのProvider選択だけをテストする**。実際の本番Projectを2件導入したという主張でも、SUI Nativeを選択した証拠でもない。

## 追加のComposition Profileは必要か

現時点での局所的な検討結果は以下。

- **不要だった宣言**：同じ案件Goコードの複製、別のResolver、Coreに案件固有のenumを追加すること、Provider候補と同じ情報をもつComposition Profile、手動の依存順序。
- **現実に必要な判断**：案件がそのGo助言とUIを使うか、どの独立Scopeで使うか、**選択可能なProviderが2件以上のときどれを選ぶか**、既存の業務意味へ取り込むか否か。
- **他の所有者の証拠が必要**：正式なContract Fingerprint、#951 Provider適合、#1815のSource閉包／Builder Acceptance、現在のEnvironment PlanとPrincipal/Action Permission、運用Lease／署名済み配布、標準Web UI Kit受理。

ここで使った`Contract.Version="v0"`と説明的`Shape`は#1567の**診断Bridgeの文字列**と揃えた参照値で、正式な#951の型付きRequest/Result/Error Fingerprintではない。構成試験がPASSしてもRuntime側に認可を与えない。案件Go実装の直接呼出しは、製品Hostが受理された実Artifactを実行できたことを意味しない。

## E37の受入範囲

```sh
go test -count=1 ./reference/application-composition -run '^TestE37' -v
```

**2案件の実Go再利用、競合時の明示選択、案件間の失効隔離、異Scope拒否、補助助言がBusiness Approvalではないこと**をGoの回帰試験で検証する。正式な環境権限、実APIや実Chromeは、統合済みの[PR #1567](https://github.com/hat47x/tei-platform/pull/1567)および#976の対象試験と区別する。

本E37は、実際の作業者の所要時間・生成AI質問回数をまだ収集しない。2案件で**同条件の実測工数比較が成立していない**ため、独立したComposition Profileの採用可否を断定しない。次に要るのは、案件・Provider ID選択を生成AIチャットで実施する際の追加質問と、受理済みSourceとの差分の**実イベント記録**である。

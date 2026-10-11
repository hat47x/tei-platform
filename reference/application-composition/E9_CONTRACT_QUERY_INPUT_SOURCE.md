# E9：Contract Queryの入力が参照する意味属性を変更影響に含める

- **状態**：既存ソースを用いた直接依存の読み取り専用試作。受理済みの業務意味、依存の完全閉包、Runtimeの権限・実行適合を保証しない。
- **管理**：[Issue #1016](https://github.com/hat47x/tei-platform/issues/1016)／[Draft PR #1051](https://github.com/hat47x/tei-platform/pull/1051)。
- **前提**：[E5](E5_PARSED_WORLD_SOURCE_JOIN.md)がApplication→Pattern→Contract Queryの型付き参照を取得し、[E6](E6_TYPED_INDEX_BRIDGE.md)と[E7](E7_PAIRED_OBSERVATION.md)がSource Indexの変更を確認する。

## 実際の定義で見えた抜け

受注明細の `order_entry.customer_reference` には次の直接参照がある。

| 構成要素 | 既存の正本 | 確認したGit blob |
| --- | --- | --- |
| Patternと入力Binding | `reference/workspace/applications/order-entry/patterns/customer-reference.tei` | `252f6396cbc2394d1f26e748b01d54f49a2e1ecf` |
| Contract Query `master_data.customer/find_by_id` | `reference/workspace/databases/master-data/contracts/customer.tei` | 実Sourceの宣言を確認。実行適合までは未検証 |
| 入力に対応する意味属性 `customer.customer.customer_id` | `reference/workspace/semantic/customer/customer.tei` | `c348122b33581e450974d23393d98fb8aa838388` |

E5はこれまでContract Queryの所在まで取得できたが、**そのQuery入力へ渡す意味属性の定義Source**を選択対象にしていなかった。この状態ではPattern／Contract Queryのファイルが不変でも、意味属性のSourceだけが変更された場合に、E6が`limited-unchanged`を返し得た。

## 変更したこと

`e5-world-index.go`の既存の型付き`Pattern.ContractQuery.Inputs[].TargetAttribute`を参照する。WorkspaceのDSLを別実装で解析せず、`ir.World.Semantics[].Data[].Attributes[]`にある該当IDの定義が一意かどうかを確認する。

- 一意に確認できた意味属性は、Source Indexの`semantic-attribute:<ID>`の`definition`に結ぶ。別のレジストリや保存済み依存一覧は作らない。
- 欠落・重複した意味属性、対応するSource Index出典の欠落は`unresolved`とする。
- 一つの属性を複数のPatternが利用しても、Report内の同じSubjectは一度だけ列挙する。
- Query自体の所在が未解決なら、属性だけを独立して`observed`へ昇格しない。
- E6は`pattern-contract-query-input-attribute`を既存のE4比較に渡し、E7のEnvelopeも同じ参照をそのまま運ぶ。

これは**意味属性の存在と変更時の再確認先**を示すだけである。Target側のRelation KeyとQuery input名の対応、入力・戻り値の型適合、業務意味、Providerでの実行、認証・認可、Transactionの整合を再検証・保証しない。それらは既存の意味・契約検証およびRuntime Authorityの責務であり、Composition独自の二重バリデータを増やさない。

## 回帰試験の状態

| 検証項目 | 記述・実行状況 | 解釈 |
| --- | --- | --- |
| E5 Goテスト | 42件から**47件**へ増加。入力属性の正常系、欠落、重複、Source出典欠落、未解決Queryの拒否を追加 | **Goで未実行**。47件は登録件数でありPASSではない |
| E5 実Workspaceテスト | 受注明細のContract Queryと入力属性を同時に検査するよう既存テストを拡張 | **Goで未実行**。ソース定義を読み取ったことと実テスト成功は異なる |
| E6 Node合成試験 | 計25件をV8のimport/export・assert互換実行で検査し、**25件成功・失敗0** | Node標準`node --test`ではなく、限定した局所検査 |
| E7 Node合成試験 | 計11件を同様に検査し、**11件成功・失敗0** | JSON Envelope境界における参照伝達のみ |
| full Repository Health | `tools/repository-health-check.sh`のexact-head全PASS／前後cleanは未取得 | **mainへ統合できない** |

今回、E5の追加Go fixtureに記したSemantic Attribute ArtifactのSHA-256文字列が60桁になっている誤りを静的に検出した。**64桁へ修正した**。GitHubへ登録したGoファイルの内容から判定したものであり、Go TestのPASSを意味しない。

## 次に再利用する仕組み

[PR #1139](https://github.com/hat47x/tei-platform/pull/1139)では、業務・Data・UIにまたがる型付きRefと変更影響判定の候補を進めている。E5〜E9は、Applicationから読める**限られた直接参照とSourceの出典**の実例であり、その汎用グラフや変更影響機構を別途実装し直す根拠ではない。

実際の全依存閉包へ進む場合、既存の意味検証・型付きRef候補が担う依存辺を使い、出典だけSource Indexへ戻す。SourceのFingerprint一致は、Compatible Contract／Business Acceptance／Runtime Grant成立の証明には使わない。

**未解決の製品価値検証**：Goによる実Workspace実行、PR #1111の解析済みバイトとIndexの整合、#1139の型付きRefを利用した影響計算、SUIのNative UIと本物のTEI Application Actionの結合、UI Kit／Targetの案件構成選択。DraftのPRはこれらの資格確認なしにmainへマージしない。

# 外部業務意味資産の選択と案件受理を結ぶ証拠契約（候補）

- 状態：**設計・負例試験の候補**。製品用のComposition Profile構文、受理済み意味、実行権限ではない。
- 管理課題：[#1570 外部Semantic参照](https://github.com/hat47x/tei-platform/issues/1570)。
- 関連：[#1016 案件構成](https://github.com/hat47x/tei-platform/issues/1016)、[#267 共通受理](https://github.com/hat47x/tei-platform/issues/267)、[#1223 Composition Kernel](https://github.com/hat47x/tei-platform/issues/1223)。
- 既存実装：`internal/workspace/imports.go`、`internal/modelref`、`internal/composition`、`internal/sourceacceptance`、`reference/semantic-composition-preflight`。
- 検証用負例：`reference/semantic-composition-preflight/external_acceptance_boundary_test.go`。

## 解消する不確実性

外部の業務知識は、利用側のManaged Sourceと別のWorkspace・配布者が所有する。
`workspace.Load`は明示したNamespaceを利用側の一時的な意味ビューへ読み込むが、
`workspace.BuildSourceIndex`のManaged Sourceと、供給側のSource Indexは分離されている。

そのため、**利用側のAccepted Sourceが変わっていなくても、外部Namespaceの意味だけが変化し得る**。
`sourceacceptance.VerifyCurrent`が成功しても、それは利用側のManaged Sourceに対する
Hostの現在の受理証拠との一致を示すだけで、外部意味モデルの受理を示さない。

現行の`imports.semanticDigest`は、指定時にNamespaceの意味内容の変化を拒否する。
これは有効な固定条件だが、値をファイルへ書いた人の権限や、案件の業務意味を受理した
Hostの判断を証明しない。逆に、この値を省略できることはローカル開発で有用なので、
すべてのWorkspace参照に固定を義務付けない。

次の契約は、既存の選択・証拠・受理機構に必要な**結合関係**を示すものであり、
第二の知識Registry、Resolver、Lock、Accepted Worldを設けるものではない。

## 論理参照・選択・受理の三つを分ける

| 区分 | 所有者と既存の参照先 | 判定に必要な内容 |
| --- | --- | --- |
| 論理ResourceRef | `modelref.Ref` / `subject.Subject` | 対象Kind・Canonical ID・論理Scope。パスとPlugin Package名は意味IDに含めない |
| 取得・構成選択 | `composition.Registry.InspectWiringSet` と`SelectedAsset` | 実際に指定したService Contract、Provider、選択Scope、BuildArtifactLock、Artifact/Semantic Digest。実行資格を意味しない |
| 配布証拠 | `InspectPublished` | Hostが認めた発行者鍵と、選択対象・配布物・期限を結ぶ署名。受理権限と異なる |
| 利用側Accepted Source | `sourceacceptance.Reader` が返す`Record` | Application、Host DecisionRef・AuthorityRevision、正確なManaged Source集合、ModelRevision |
| 外部意味の案件受理 | **未実装。既存の共通受理へ接続する必要がある** | 同じ案件の受理判断が、どの型付きResourceRefのどのSemantic Digestを採用したかの証拠。候補レポートから自動生成しない |
| Runtime Authority | Hostの既存検査 | 現在のPermission、Lease、Fencing、Provider実体、実行対象を呼出し時に再検査 |

`modelref.Ref.Scope`（論理的な識別範囲）、`composition.Provider.Scope`（取得サービスの隔離範囲）、
`sourceacceptance.Record.ApplicationID`（案件・実行対象）を同じ文字列へ暗黙変換しない。
その対応関係は案件側の明示的な選択として残す。

### 最小の受渡し情報（採用前の概念例）

以下は**提案形**であり、新しいYAMLの実装済み構文ではない。利用側案件が、
既存のSource受理判断に紐付けて「外部意味資産のどの版を採用するか」を示すための情報である。

```yaml
application: monitoring
sourceAcceptance:
  decisionRef: host-owned-decision-reference
  authorityRevision: host-owned-current-revision
  modelRevision: sha256:<consumer-managed-source-revision>
selectedMeaning:
  - ref:
      scope: "namespace:os"
      subject:
        kind: semantic-namespace
        id: os
    semanticDigest: sha256:<normalized-namespace-meaning>
    retrieval:
      scope: tenant-a
      providerId: read-os
      contractId: reference.semantic.asset_read
      artifactDigest: sha256:<exact-distribution-bytes>
```

この構造をそのまま永続記録へ採用するのではなく、既存のHostの**同じ受理判断**から、
受理済みSourceと選択済み外部Resourceの両方を再確認できる形へ縮約する。
`sourceAcceptance`のラベルを利用側が自己申告しただけでは信用しない。
検証済みの単独の`Record`を二度複写して、外部意味も受理済みであるかのように扱わない。

`semanticDigest`は正規化した意味内容の同一性、`artifactDigest`は物理配布バイトの
同一性であり、両者を代替しない。Sourceのファイル移動やパッケージの再配置が意味内容を
変えない場合、案件の業務意味まで無条件に再受理させない余地を残す。
ただし取得元の信頼や運用上の許可は、その場で別途再検査する。

## 判定順序

1. **明示参照**：利用側の局所名、型付きResourceRef、利用目的と作用域を確認。
   `参照`、`導入`、`拡張`、`写像`、`束縛`を異なる操作として扱う。
2. **構成選択**：既存Compositionの候補判定を呼び、曖昧・不足・別Scopeを止める。
   対象Namespaceの暗黙の追加や「最新版」選択は行わない。
3. **配布検査**：明示した固定ArtifactとSemantic Digest、発行者の現在の信頼を確認。
   `InspectPublished`の成功は取得した意味の正当性と受理の前提候補に過ぎない。
4. **意味検証**：型・属性・参照先の実在と整合を既存Validatorで確認し、
   名前空間の直接依存を棚卸しする。合法な双方向業務参照をサービス依存の循環へ変換しない。
5. **案件の受理確認**：Hostが保持する現在の案件受理判断について、利用側Managed Sourceの
   `ModelRevision`と、明示した各外部Resourceの`SemanticDigest`が**同じ受理範囲**に含まれるか照合。
   この検査は現在**未実装**であり、利用できなければ「受理未確認」として停止する。
6. **実行時**：再度選択・信頼・権限を確認。受理済みの意味が存在することと、現在の実行許可は別。

### 受理境界の負例と判定候補

| 事象 | 判定 | 理由 |
| --- | --- | --- |
| 利用側Sourceは不変、供給元Namespaceの意味Digestだけ変更 | **受理未確認** | Sourceの受理記録は外部変更を覆わない |
| 署名とArtifact Digestは一致、Hostの案件受理判断が存在しない | **受理未確認** | 発行者の信頼と業務意味の採用判断は独立 |
| 同一Namespace IDの異なる二提供者 | **競合** | 自動マージ・後勝ちを禁止 |
| 他のNamespaceからの参照先が未選択 | **未解決** | 自動的な依存閉包拡張を禁止 |
| 受理判断に記載されたDigestと現在選択された意味が不一致 | **再受理候補／停止** | 無断の受理版更新を禁止 |
| 利用側Managed SourceがHostの現在のSource受理記録と不一致 | **拒否** | 既存`VerifyCurrent`へ委譲 |
| 取得サービスのScopeまたはProviderが変化 | **再選択・現在性確認** | 論理意味の一致だけで取得権限を保証しない |
| 署名鍵が失効／検査中に信頼世代が変化 | **拒否** | Host側の信頼検査へ委譲 |
| 供給側ファイル移動、Semantic Digestは同一 | **意味変更なしの候補** | 実ファイル来歴・現在の取得権限は別途評価 |
| 双方向Relationの循環 | **意味検証後に判定** | サービス依存の循環エラーとは別物 |

「受理未確認」「再受理候補」は設計時の診断概念であり、現行CLI/Hostの正式なエラーコードではない。

## 対象の種類を先行して増やさない

最初の証拠対象は`semantic-namespace`のみ。
Project / Module / Resourceは**所有関係を整理する論理概念**として扱い、
必ずしも新しいCore列挙型や専用配布パッケージを要求しない。
サービスの配布はPlugin、取得先はPath/CAS等の既存方式を使えるようにする。

- **参照**：意味を閲覧・参照するが、利用側正本を変更しない。
- **導入**：案件がどの意味を採用するかという、共通受理経路の判断。
- **拡張**：案件Ownedの別定義。提供元の意味内容を上書きしない。
- **写像**：両者の型・項目・関係の対応を明示し、その業務意味がある部分は案件側で受理する。
- **束縛**：Contract/Provider/Environmentの実現を選ぶ。業務意味の受理を代用しない。

実Consumerで必要な項目が確定するまで、`ResourceRef`専用の永続型、
`knowledge://`、新的なLockや配布ABIを作らない。

## 現在の負例Referenceと次の実証条件

`external_acceptance_boundary_test.go`は、二つの実Workspaceを作成し、
Providerの`os.process_id`の基底型だけを変えたときに以下を観測する。

- 利用側のManaged Source一覧と`ModelRevision`は変わらない。
- 供給側Namespaceの`SemanticDigest`と一時的なSemantic Worldは変わる。
- 合成したHost Source Readerで`sourceacceptance.VerifyCurrent`は変わらず成功する。
  **これは外部意味の受理を意味しない**。
- 利用側が前の`semanticDigest`を明示的に固定すると、既存Loaderが変更を拒否する。
  ただし固定値を書くだけでHostの業務意味受理が成立するわけではない。

検証対象：
```sh
go test -count=1 ./reference/semantic-composition-preflight ./internal/sourceacceptance ./internal/workspace
```

**未実証**：実Hostの「同じ受理判断」にResourceの正確な意味Digestを含め、
Current Readerとして検証した一続きの経路。ここは[#267](https://github.com/hat47x/tei-platform/issues/267)の共通受理と、
[#1016](https://github.com/hat47x/tei-platform/issues/1016)の案件選択を接続してから、
案件の実Consumerを用いて実証する。外部知識の取込時に新しい特権書込経路を設けない。

この段階は、前段の署名・Digest確認から**業務意味の受理に必要な証拠が不足していることの検証**であり、
実際に外部業務知識を受理・運用できる製品機能の完成を意味しない。

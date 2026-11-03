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
## 2026-11-03 追補：外部Referenceの取得鮮度・失効・受理境界

**状態：設計と試験計画への追補。新しい製品契約・API・Owner受理・実Provider検証を採用／達成したものではない。**

外部Semanticの案件受理という本書の主責務は変えず、TEI DSL・設計文書等の外部参照が「現時点で何を取得・検証できたか」を受理前に伝える共通契約を整理する。参照するResourceの種類をCoreへ一括追加する案ではない。既存[#1570](https://github.com/hat47x/tei-platform/issues/1570)の明示選択、[#1002](https://github.com/hat47x/tei-platform/issues/1002)／[#1017](https://github.com/hat47x/tei-platform/issues/1017)のSource読取境界、[#1018](https://github.com/hat47x/tei-platform/issues/1018)のLanguage Plugin、[#2505](https://github.com/hat47x/tei-platform/issues/2505)の版間対応、[#267](https://github.com/hat47x/tei-platform/issues/267)のShared Acceptance、[#958](https://github.com/hat47x/tei-platform/issues/958)の認知的現在性、[#2600](https://github.com/hat47x/tei-platform/issues/2600)のBuild再利用を再利用し、第二Resolver・Knowledge Registry・受理Ledger・Cache Engine・Hostを作らない。

### 取得Observationと受理の最小の受渡し情報

| 項目 | 保持すべき区別と責務 |
| --- | --- |
| 参照宣言 | 宣言元のScope付き論理ID、Source occurrence（ファイル／範囲／入力版）、参照種別、必須性、用途。相対path・URLはSemantic IDではない。Parserは参照を抽出し、無制限なファイルアクセスを持たない。 |
| 論理対象と所在 | 型付きResourceRef、宣言Locator、実取得Locator、移動・redirectの対応。供給側WorkspaceのSource Indexと利用側Managed Sourceは別。 |
| Source Revision | Commit・ETag・Version ID等の種類／発行元／値。供給元Revision、実byteのContent Fingerprint、正規化されたSemantic Digest、配布Artifact Digestを別々に扱う。 |
| 取得Observation | Attempt ID、Provider／Resolver版、retrievedAt（内容を得た時刻）、checkedAt（現在性を確認した時刻）、応答・失敗原因、取得範囲／完全性、現在の主体・用途。試行ごとに履歴を保持する。 |
| Cache | 検査した固定byteとfingerprint、取得証拠、適用できる用途・範囲を結ぶ。部分取得のhashを全体hashとしない。Content CacheからOwner受理・現在権限を復元しない。 |
| 依存関係 | Source occurrence→内容・Role／Operator Pin→Candidate／Accepted Meaning→文書／Build／Consumerのうち観測済みの辺だけを使う。未列挙・探索不完全は「依存なし」ではなくunknown。 |
| 意味受理・実行 | 既存#267で、Owner判断を意味・scope・根拠版へ束縛する。受理履歴と現在の適用資格は別。Host／Providerの読取・開示・実行権限は都度検査する。 |

Providerで実際に読んだbyteと、Parser／Buildへ渡したbyteが一致して初めて、そのAttemptの固定入力と主張できる。事前・事後hash一致だけで検査と実読取の間のABA／TOCTOUを排除しない（[#2595](https://github.com/hat47x/tei-platform/issues/2595)）。異なる外部参照をWorkspace Fingerprint一つで完全に保証しない。

### 6状態は利用者向けの代表分類、内部は独立軸

| 表示分類 | 成立条件・留保 |
| --- | --- |
| fresh | Consumerが要求した対象・scope・範囲・鮮度期限と取得根拠を満たす。意味の正しさ・Owner受理・実行許可は別。 |
| stale | 旧Observation／Cacheは存在するが、現在の用途に求める鮮度がない。履歴閲覧と現在判断を分ける。 |
| unavailable | offline／timeout等で現在の取得状態が不明。not-foundとみなさない。staleなCacheを同時に持ち得る。 |
| partial | 要求されたmember／page／field／依存範囲の一部が不足。取得部分だけで全件適合・不存在を推論しない。 |
| not-found | 信頼できる供給元・scope・時点で不在を確認。404が権限隠蔽を兼ねる場合、外部には不在を断定しない。 |
| permission-denied | 現在の主体・用途では取得・開示不可。低権限Consumerには存在自体を伏せるためunavailableへ縮約できるが、内部拒否事実を消さない。 |

一つの排他的な永続enumへ押し込まず、取得結果、範囲の完全性、Consumer相対の鮮度、開示資格を分離する。例えば「取得失敗＋旧Cache stale」「取得部分は新鮮＋全体はpartial」が成立する。直近の失敗と最後の成功時点も別に表示する。正常0件もPredicate・Scope・Provider・取得時点・Rowset／Field完全性に依存する（[#2582](https://github.com/hat47x/tei-platform/issues/2582)）。状態から診断のSeverityを一律に決定せず、必須のローカルファイル不在、任意の依存、未生成Asset、固定Snapshotの履歴再現、現在性を要する外部実行を分ける。

### イベントごとの失効対象・残す履歴

| 変更 | 取得Observation／Candidate | Accepted Meaning | Build／SHO／AI Consumer |
| --- | --- | --- | --- |
| ファイル移動・内容同一 | 参照Locator・Source occurrenceを再解決。内容依存Candidateは再利用候補 | 旧受理を履歴保持。新Sourceの意味的対応・scopeを確認 | 内容計算は省略候補だが引用・Source Index・Trace・現在開示は再確認 |
| Revisionのみ変更、実byte同一 | 新Revision／観測を追加し、供給元の解釈規則も確認 | 無条件の自動再受理も全面破棄もしない。受理範囲との適合を判断 | 内容Cache再利用候補。現行Source Bindingと生成物資格は別 |
| 内容A→B | 影響候補を再計算し、旧観測は履歴保持 | AのOwner判断をBへ自動継承しない | 実依存Closureに基づき選択的再生成。未知Edgeならskip保留 |
| Permission／信頼／開示変更 | 歴史的取得事実は保持し、現在取得・提示資格を再評価 | 歴史的意味を削除しない。現在採用と実行は別判断 | 旧Cacheから権限外の表示・外部送信・invokeをしない |
| A(r1)→B(r2)→A(r3) | 内容hashが戻っても3回のObservationを区別。中間未観測なら「変更なし」は未証明 | r1のReview・撤回・Owner判断は自動復活しない | 内容と実依存が一致すれば再利用候補。現在Grant／Review／Consumer資格は再検証 |
| offline／取得不能 | unavailableと最終成功Observation／Cacheのstaleを区別 | 旧受理は履歴として残す。現在の外部根拠を要する判断は保留 | 固定入力のread-only再現は条件付きで許す。現在外部取得・権限・実Effectは保証しない |

失効は過去Evidenceの削除ではなく、当該Claim／Consumerに対する**現在の再利用資格の再評価**。依存が無関係であると実証した範囲は維持できるが、未観測の辺を「無関係」と推定しない。既存[#2626](https://github.com/hat47x/tei-platform/pull/2626)の実Go CLI・同byte Source移動は、World Fingerprint不変でもSource Index／Traceが変わる限定Evidenceであり、実HTTP・現在Owner権限・SHO保存後原典再解決のPASSではない。

### offline時の診断・修正候補・Build

- Studio／CLIは、宣言元span・最後に取得できたRevision／時刻／hash・今回の失敗・Cached contentのscope・未確認依存を、開示可能な情報に限って示す。古いCacheだけで現在Sourceをfreshと表示しない。
- Parserのsource-direct診断と、Semantic SubjectからSource Indexへ戻した診断を分離する（[#17](https://github.com/hat47x/tei-platform/issues/17)／[#1155](https://github.com/hat47x/tei-platform/issues/1155)／[#2317](https://github.com/hat47x/tei-platform/issues/2317)）。AIのバルク修正候補は、期待Revision・対象span・Scope・変更区分・根拠を持たせPreviewする。古い候補を現行Sourceへ無条件適用せず、業務意味の変更はOwnerへ渡す。
- 固定Snapshot・必要Dependency Closure・Role／Operator／Builder／Target Pinが確認できる範囲ならoffline解析／生成の再現を検討できる。現在の外部参照鮮度、Host Grant、Provider効果をCacheだけでは保証しない。permission-deniedの内容、存在、診断件数、Tool候補も低権限のAIへ漏らさない（#1045／#2516）。

### 最小Fixtureと優先順（すべて新規実行NOT RUN）

| ID | 入力／注入 | 独立Expected |
| --- | --- | --- |
| RF-01 | 固定r1のAを完全取得 | 取得byte・Source occurrence・必要scopeのfresh |
| RF-02 | A同byteで移動 | 内容同一、Locator／引用は再資格化 |
| RF-03 | A同byte、Source Revisionだけ更新 | 新Observationと内容再利用可能性を分離 |
| RF-04 | A→B | 影響Candidate／Artifact再評価、旧受理履歴保持 |
| RF-05 | A→B→A | Content再利用とReview／Grant自動復活拒否を分離 |
| RF-06 | 取得途中に権限拒否 | permission-denied、未認可部分の利用・開示なし |
| RF-07 | page1だけ取得、page2欠落 | partial、全件不存在を主張しない |
| RF-08 | offline、Aの旧Cacheあり | unavailable＋stale、固定再現と現行判断を区別 |
| RF-09 | 必須参照に権威ある不在結果 | not-found診断、旧履歴保持、秘匿404は例外 |
| RF-10 | 検査後・消費前にA→B→A | 前後hashのみで同一消費byteを保証しない |
| RF-11 | A同byte、Role／Operatorの実装Pin変更 | Source Cache保持、Meaning／Build再資格化 |
| RF-12 | A同byte、Consumer Policy／Grant変更 | 内容一致と現在の開示拒否を両立 |

**P0：** RF-01／06〜09の取得観測・完全性・状態分類を既存Readerと取得Providerの契約で検証。**P1：** RF-02〜05／11を[#2505](https://github.com/hat47x/tei-platform/issues/2505)の版間対応と[#2600](https://github.com/hat47x/tei-platform/issues/2600)の全件対選択的Buildへ渡し、誤再利用／不要な全面失効を計測。**P2：** RF-10／12を[#1124](https://github.com/hat47x/tei-platform/issues/1124)の後着応答、[#2516](https://github.com/hat47x/tei-platform/issues/2516)のAI／SHO開示、#2595の消費byteと接合。**P3：** 案件が選択した外部Semantic Digestを既存#267／#1016の同一Decisionへ結び、正式受理はOwner判断とする。新Gateは追加しない。

**停止条件：** 必要Scope／完全性／実消費byte／Dependency Closure／現行Permission／意味対応のどれかが不明なら、現在の意味成立・Build skip・外部Effectを合格へ昇格させない。read-only履歴・診断候補まで一律停止しない。

**Owner未決：** 用途別の鮮度期限、権威あるnot-foundの条件、offline固定Buildの許容範囲、内容同一時の受理省略可否、権限変更後の履歴開示。変更は既存IssueのOwnerへ返す。本追補は設計であり、RFの実Go試験・実Remote Provider・Studio適用・正式Owner受理・実Targetの資格を実証していない。

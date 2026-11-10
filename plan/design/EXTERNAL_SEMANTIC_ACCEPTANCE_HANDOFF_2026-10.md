# 外部業務意味資産の選択と案件受理を結ぶ証拠契約（候補）

- 状態（2026-11-08）：**外部意味の案件受理契約は設計候補・Owner未受理**。外部Import P0aの[Draft #2786](https://github.com/hat47x/tei-platform/pull/2786)と、同一Nested Provider Loadの原文証拠を試した旧[Draft #2812](https://github.com/hat47x/tei-platform/pull/2812)はmain未統合。Source-only／Manifest-only／両方変更の負例[PR #3479](https://github.com/hat47x/tei-platform/pull/3479)は**main統合済み**。同一Loadの選択Namespace parsed Source／Manifest private witnessは現main起点の[移植Draft #3524](https://github.com/hat47x/tei-platform/pull/3524)で実装・検証中であり、正式な製品成功を現時点では主張しない。完全Source入力閉包、原子的複数ファイルSnapshot、実Consumerへの開示、現在Grant／Owner意味受理、実Effectは未資格。
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
## 2026-11-03 第2追補：5段階失効・依存閉包・独立Expected

**位置づけ：設計／反証用Fixtureの追加計画。新規Product実装、正式Core schema、受理済みMeaning、実行資格、試験PASSの報告ではない。** 第一追補のRF-01〜12と状態定義を繰り返さず、他レーンの認知・Build・Source再配置・実行時Effectの成果から、不足していた横断判定だけを追加する。#1570/#1017のSource、#2505の版間SourceRef、#958の認識上の鮮度、#267の受理、#2600のBuild、#1174の実行効果という既存Ownerを維持。新Registry／Host／第二Canonical／共通失効Engineを作らない。

### 5段階の失効と履歴保持（現在利用資格だけを再判定）

| 段階 | 歴史として保持する記録 | 変更時に失効／再資格化する現在のClaim | 禁止する推論・責任境界 |
| --- | --- | --- | --- |
| Source Observation | 宣言元occurrence、供給元URL／解決Locator、Source Revisionの種類と値、取得Attempt／時刻、実byte fingerprint、完全性、失敗と権限の観測 | 新Source／移動／権限変更／offlineにより、取得時点・対象・現在性・開示可否を再確認。旧観測は削除しない | Cacheがある＝fresh、取得不能＝not-foundではない。Reader／Providerの責務 |
| Structural Candidate | 固定SourceとParser・Language Plugin・Role／Operatorの推移Pin、Source span、候補・未解決の記録 | 内容、解釈器、依存Role、意味対応、隠れた依存の変更・未確認を対象に再導出 | Candidateの型・値・Data IDが同じ＝Meaning同値ではない。既存Parser／Validator |
| Accepted Meaning | Owner Decision、対象Meaning、scope、受理時Revision、撤回等の履歴 | 変更後のSource／Meaning／Owner判断が現在の用途へ適用できるか別判定。旧Decisionの歴史は保持 | A→B→AでOwner判断を自動復活させない。#267／案件Owner |
| Build Artifact／Evidence | 全必要memberのArtifactSet、Builder／Toolchain／Target、実入力closure、Build Attempt、過去の検査結果 | 実消費入力・Dependency closure・必要Member・現在のConsumer条件の変化により再利用資格を保留／再検査。古いArtifact byteは破棄しなくてよい | 主生成物やCAS digest一致＝全member／実ロード／意味受理合格ではない。#2600等 |
| Execution Receipt／Effect | 過去のCommand、Operation Identity、Host/Providerの応答、独立Effect観測、不明Commit | 過去の効果自体は後のSource更新で失効しない。現在の再実行・再送・Grant・副作用重複防止資格を別判定 | Receiptの存在やAへの復帰＝新規実行許可ではない。ACK不明なら無断再送しない。#1174／実Provider |

世界の正規化内容を示す **World Fingerprint**、Source occurrenceや参照対応の改版を示す **Model Revision／Source Index Fingerprint**、実入力・出力の版を示す **Artifact Revision／ArtifactSet Digest** は、比較する対象が異なる。WorldFP不変でもSource移動でIndex・Trace・引用が変わり得る。Artifactが同byteでも供給者の現在信頼・Consumerの開示権限・Host Grantは別。これらは既存のRevision／Build関係の照合列であり、新しい共通IDを定義しない。

**既存の限定証拠**：merged [#2626](https://github.com/hat47x/tei-platform/pull/2626)の実Go CLI同byte Source移動はWorldFP不変とSource Index／Trace変化を観測した。別の [#2649](https://github.com/hat47x/tei-platform/pull/2649) では実CLI・Artifact保存・独立Readerのbyte対応が限定成立したが、Manifestの未充足が残った。両者を一つの外部HTTP／Owner意味受理／実Consumer E2Eに合算しない。

### 部分再計算を認めるための前提

1. Claimごとの必要対象集合・Source参照、Parser／Role／Operator推移Pin、Rule／Query Predicate・Relation、Scope、取得範囲、Rowset／Field完全性、Provider route、Target／Consumer、利用目的・開示・現在Grantのうち実際に必要な依存を観測する。**未列挙は空集合ではない**。0件や不存在はPredicate・全件性・時点に依存する否定依存。
2. 実際に取得・検査したbyteが、Parser／Build／Interpreterへ渡された固定byteと一致することを示す。事前・事後hashの一致では検査と実消費の間のABA／TOCTOUを証明しない（#2595／#1017）。
3. 依存closureが現在の対象Claimについて十分・完全であり、変更の無影響が反証可能なら選択的再計算候補。依存追加／削除・非表示Predicate・途中失敗・Provider差替え・scope不明・partial取得では、対象Claimのskip資格をunknownとし、広い再検査か全再計算へ戻る。
4. 全再計算でも未取得Source・権限不足・現在Grantが自動的に解消するわけではない。根拠のないsuccessを作らずunverifiedを保持する。一方、無関係と確認された他Claimの歴史的証拠やArtifactまで無条件に失効させない。
5. 内容再利用が成立したとしても、現在のSource引用・SHOの保存文書の原典再解決・Owner受理・ArtifactSet全member・実Consumer読取・Runtime Grant／Effectは別々に確認する。

### 独立Expectedを持つ追加Fixture（RF-01〜12は維持、すべて新規NOT RUN）

| ID | 変更入力と独立Expected | 観測器・既存Owner |
| --- | --- | --- |
| RF-13 | **外部参照の表示値と本文は同じ、内部Rule／Predicateだけ変化**。表示byteの一致からRule／Claimを再利用しない | 実依存のRule footprint、独立Rule Expected。#2624／#2600 IB-12／#2516 RC-02 |
| RF-14 | **部分取得のpage1は0件、page2に対象行あり**。partialをnot-found・正常0件・全件成立へ昇格しない | 独立対象ID全集合とページ／Field完全性。#2582 ND-03／#2516 |
| RF-15 | **内容・WorldFP同一、Provider routeと現在Grant／Disclosureのみ変化**。古い受理履歴は残し、現在の外部読取・AI開示・Action許可を別判定で拒否 | Provider／Hostの現在権限Observerと開示先。#958／#2516／#1174 |
| RF-16 | **外部依存1件の追加／削除、既存Source byteは同一**。宣言された旧Source集合から自動的に「追加依存なし」と推論せずskip保留 | 独立の必要依存集合、実選択Predicate／Source closure。#2600 IB-07／#2595 |
| RF-17 | **主Artifact一致だが必要memberが欠落または旧版**。実Target/Consumerの資格を与えない | ArtifactSet manifest／独立Reader／実ロードObserver。#1102／#2600 IB-05／#2649 |
| RF-18 | **旧Receiptの後、Commit応答喪失、Source A→B→A**。既発生Effectの履歴は保持、再送／重複回避資格はunknown、無断Command再実行は0 | Operation identity／Backend独立Effect観測／現Grant。#1174／#1121 |

FixtureのExpectedをTEI Parser／AIが作ったCandidateから生成しない。別に固定したSource byte、対象集合・scope、Rule／依存一覧、必要Artifact member、Grant／Policy、既発生Effectを真値とする。比較票は「入力Revision、実byte・取得Coverage、Claim、Expectedの失効／再計算、観測器、実結果、現在資格、禁止する結論、Owner、exact HEAD／Run」を最低列とする。局所合成PASSと実Go CLI・実HTTP・Studio・外部Host E2Eは合算しない。

### 既存計画へ渡す差分と順序

- **外部参照・改版**：#1570／#1017と#2505にRF-02〜05／16のSource位置・改版・実取得を渡す。現在のHTTP鮮度とSemantic Digest固定を同一視しない。
- **認知・Context**：#958／#2533／#2516にRF-08／13〜15の用途・完全性・開示差分を渡す。認知ReportはAccepted MeaningやExecution Receiptの第二正本にならない。
- **増分Build**：#2600のIB-05／07／10〜14にRF-02〜05／11／13／16／17を対応付け、BR-FULL／BR-SELECTIVE／BR-UNVERIFIEDでfalse reuse・不要な全面失効・要再資格化を比較する。新しいSchedulerを作らない。
- **Studio・AI修正**：#1124／#1824にRF-10／12／15の後着応答、Scope別Graph選択、現在の開示可否を渡す。staleな修正候補を現行版へ適用しない。実行前の検査が必要な場合は旧操作を再試行しない。
- **Owner／Runtime**：#267／#1016でRF-03〜05／18の正式案件受理範囲を扱う。#1174が実Receipt／Effectを所有し、Sourceの差分から副作用の消去や再送を推論しない。

**順序**：P0=RF-01／06〜09／14（取得の失敗と全件性）→P1=RF-02〜05／11／13／16（版間・依存局所化）→P2=RF-10／12／15／17／18（実Consumer・実Build・実Effect）→P3=既存案件Ownerによる意味受理。各段階はその範囲の実行と証拠が揃ったときにのみ前進し、旧CIのPASSを新しいHEADへ転用しない。

**未決**：用途別鮮度期限、権威あるNot Found、offlineの固定Build・公開、同byte改版でOwner再受理を省略する条件、権限変更後の履歴開示。新ADRは採用せず既存Ownerへ判断を残す。#2667の横断Draftは別Ownerが同じ時期に編集しているため、計画本文をこのPRで重複更新しない。上記のRF→IB/ND/RC対応表をリンクで受け渡す。

## 2026-11-03 第3追補：取得状態から操作資格への判定票

**状態：設計・検証計画、NOT RUN／NOT ACCEPTED。** 第一・第2追補の取得6分類とRF-01〜18は維持する。ここでは「個々の取得結果がfreshなら、その集合全体・現在の用途・実行資格も成立する」という誤推論を防ぐ。複数Providerの読取整合性は既存[#1708](https://github.com/hat47x/tei-platform/issues/1708)、Query/Contextの否定依存と空集合は[#2582](https://github.com/hat47x/tei-platform/issues/2582)、現在のOwner／Accessは[#267](https://github.com/hat47x/tei-platform/issues/267)／[#958](https://github.com/hat47x/tei-platform/issues/958)、Source読取は[#1017](https://github.com/hat47x/tei-platform/issues/1017)に委ねる。新しい共通Snapshot機構、negative-cache Engine、Reference Resolver、Review Gateは追加しない。

### 1. 宣言の利用モードと取得結果を区別する

外部Referenceの必須／任意という区別だけでは処理条件は定まらない。候補として少なくとも以下の利用モードを**既存のResource選択・Dependency・Consumer要件の組合せ**で表現する。新しいDSLキーワードやSchemaの採用は含まない。

| 参照利用モード（概念） | 満たすべき証拠・失敗時 |
| --- | --- |
| 固定版（pinned、再現用途） | 指定した供給元Revision／固定内容・Digest・必要な入力閉包が揃えば、現在ネットワークがunavailableでも履歴の再解析・offline Build候補。現在公開／実行権限は別。 |
| 現在版（floating、現在性必須） | 供給元・scope・取得時点・要求鮮度／整合性を再照合。旧Cacheだけでは現在版と呼ばない。取得不能はcurrent qualification unknownとして保留。 |
| 任意の依存（optional） | 不在が許される意味・fallback・生成物への影響を当該Ownerが定義している場合のみ継続。optionalだから未取得部分を空集合やNULLと解釈しない。 |
| 生成段階の依存（generated at stage N） | Nより前に不在でも即時の永久失敗にはしない。N後の出力が必要なのに欠落・partialなら、その段階の生成・利用を拒否。既存Build graphへ接続。 |
| 現在の外部効果に必要な参照 | Source取得freshだけで進めず、案件のAccepted Meaningと現在のHost/Provider Grant、実際のRequest・副作用の保証を確認。offline／ACK不明を自動再送理由にしない。 |

ローカルの必須固定ファイルが、権限内の確定した読取りで存在しなければ診断Error。一方、通信断、レート制限、アクセス不可、部分取得、未解決のsymlink／redirectや途中生成を同じ「ファイルが存在しないError」へ潰さない。Parserの構文エラーとSource Readerの参照解決エラーはsource spanで関連付けても別原因として保持する。

### 2. 取得観測の利用資格を判定する最小の質問

一つのObservationからConsumerへ渡す前に、次の質問を**既存のSource／Model／Evidence参照から**確認する。判定の不足はread-only結果へ残し、別の汎用資格Registryに保存しない。

| 判定する質問 | 根拠と否定条件 |
| --- | --- |
| 参照先は何か | 宣言されたScope付き対象・Locatorと、実取得Locator・転送・移動履歴・Providerを対応付ける。redirect／symlinkで別権限先へ到達したら再認可。論理同一性はURL一致・hash一致から推定しない。 |
| どの入力を実際に読んだか | 供給元Revision（種類付き）、固定byte digest、対象member・field・範囲、Source occurrence、実消費byte。同時刻のetag／mtimeや前後hashだけでは原子的取得を保証しない。 |
| 不在・完全性を本当に確認したか | Not Foundの権威・対象範囲・時点・Visibilityを評価。404が秘匿やProvider不具合を含む場合は不在を確定しない。部分取得、探索未完、正常0件は区別。 |
| いつまで利用可能か | 供給元の更新情報、確認時点、Consumer用途別鮮度期限、必要な再検査条件を照合する。ローカル時計の進み・遅れ、条件付き取得やTTLは保証範囲を限定。 |
| いま誰が使えるか | 現在のAccess／Disclosure／信頼先／配布先を確認する。取得できた内容を外部AIへ送信可能とは推論しない。低権限には存在・件数・診断差からも秘密を漏らさない。 |
| 複数入力は整合しているか | 各Sourceがfreshでも同一論理時点とは限らない。Provider別snapshot／Read cohort／一貫性証拠・非原子的混合の可否を案件要求と照合。保証不能ならstrict用途はUnsupported／unverified。 |
| どの判断に使うか | Source表示・解析／Candidate生成・意味受理・Build・外部Effectごとに必要な証拠を別記。Owner Decision、Build receipt、Execution receiptを相互代替しない。 |

**Negative Cacheの注意**：以前の正しいnot-foundも、対象Locator／Provider／Scope／権限・観測時点に相対的な歴史記録である。後に新設されたファイルや回復した参照を「現在も不存在」と決めつけない。存在を隠す404と確定不在を同一視しない。negative-resultの期限や変更通知の信頼度はOwner未決であり、独立した汎用Negative Cacheを実装しない。

**Read cohortの注意**：Source AのRevision r12とSource Bのr15がそれぞれfreshでも、同じ業務時点に属する証拠はない。必要な用途が整合したsnapshotを要求し、Provider間で証明不能なら「単独ではfresh／組合せ評価はunverified」と説明する。原子的な複数Provider取得をTEI Coreで先行実装しない。単なるClock timestamp一致、同名snapshot token、同一WorldFPで整合性を主張しない。

### 3. 操作ごとの既定判断（採用前の比較基準）

| Consumer操作 | stale／unavailable／partial時 | 意味・権限が変わるとき |
| --- | --- | --- |
| 履歴の読取・出典比較 | 取得済みの固定範囲について、閲覧権限がある場合は許容。現在版とは表示しない | 旧記録を保持、現Disclosureで公開可能な部分だけ表示 |
| 構文解析・診断・read-only候補 | 固定入力の範囲で許容。未取得依存は独立診断。現在Sourceへの断定なし | Role／Operator版・Source span・Scopeを版付きで表示 |
| 現行Meaningの採用・Owner再受理 | 必要な外部Source／scope／整合性を確認できなければ保留 | 既存#267でDecision。AI／Resolverは受理しない |
| pinned Build／Artifactの再現 | 必要な入力closure／実消費byte／Toolchain Pinが揃った場合に限り候補。未確認依存や必要Member欠落は保留 | 計算結果の再利用、ArtifactSetの完全性、現在配布資格を別々に評価 |
| AIによる一括修正提案・適用 | 旧版のPreviewは許すが現在版への無検査適用は禁止。版・span・対象・意味差分を再確認 | Sourceを変える適用とOwnerによる意味受理、外部送信の権限を分離 |
| 外部実行・保存・再送 | 現在Grant・対象Host／Provider条件とOperation identityが不明なら拒否。Commit結果unknownは自動再送しない | 既発生Effectの履歴は保持し、現在の新規実行資格は別に再検査 |

### 4. 反証Fixture RF-19〜24（RF-01〜18に追加。いずれもNOT RUN）

| ID | 独立Expected（禁止推論） | 既存Owner／必要Observation |
| --- | --- | --- |
| RF-19 | **AとBはそれぞれfreshだが異なるread cohort**。strict-consistencyを要求する導出を合格にしない。許容したbest-effortならnon-atomicを明示 | #1708、独立に固定したA/Bの値とProvider snapshot。混在値を確定値としない |
| RF-20 | **確定Not Found後にファイルが作成され、最終試行はoffline**。旧negative-resultを現在不存在として再利用しない | #2505／#958、供給元の作成イベントと取得試行履歴。negative cacheの期限不明はunknown |
| RF-21 | **HTTP 404が存在隠蔽を兼ね、低権限利用者のみ取得不可**。外部AI向け結果・件数・Tool候補から存在を推定できない | #1045／#2516、実Providerの認可応答・Disclosure observer。内部原因は権限内に記録 |
| RF-22 | **固定pinned Aでoffline解析は成功、同じAを現在版として利用する処理は保留**。同一Cacheに対する用途差を保持 | #958／#2600、入力Closure・現在性要件・独立Expected。offline Build成功を現Grantとしない |
| RF-23 | **Reference redirect／symlink先が選択Scope外へ移動**。元URLが許可済みでも外部内容を取得・取り込まない。新しい外部Sourceを暗黙選択しない | #1002／#1017／#1570、実効Locator・物理境界・Provider認可。TOCTOU未保証を別記 |
| RF-24 | **同一Source hash・同一画面選択後、Policyだけ撤回／旧応答後着**。旧内容の新規表示・AI送信・自動適用を拒否し、許可された無関係範囲は保持 | #1124／#2516、Consumer epoch、Policy change observer、Source Index・Scope binding |

各RFの独立Expectedを実装から生成しない。最初は公開合成Providerにより状態とscopeを制御し、取得失敗・Read Cohort・Policyを観測する。現行の実Provider・外部HTTP・OSファイル権限・SHO保存Reader・TEI Studio・実Hostの適合を一度に主張せず、**その試験が実際に読んだ入力と対象Closure**を記録する。Referenceで合格した一段だけに限定してIssueの証拠を更新する。

### 5. 実施順序・停止条件と横断計画への引渡し

**第1縦断（headless read-only）**：P0取得分類RF-01／06〜09／14にRF-20／22を追加。独立Expected・Source span・Provider返却／Cache・時刻・正確な失敗コードを同一Attemptで取得。Loader／Readerの実験で有効な能力範囲を先に確定し、外部HTTP一般保証を主張しない。

**第2縦断（必要なClaimだけ）**：RF-02〜05／11／13／16／19と#2505／#2600を突合。Sourceの内容一致とSource Index／Trace・Meaning受理・Build省略、Read cohort不一致を別列にしてBR-FULL／BR-SELECTIVE／BR-UNVERIFIEDの期待と比較する。

**第3縦断（情報と副作用）**：RF-10／12／15／17／18／21／23／24を#1124 Studio、#2516 AI／SHO、#1017実Reader、#1174外部Effectへ渡す。権限外の内容・存在・診断件数の送信なし、旧応答拒否、旧Receiptからの自動再送0回を各Ownerが検証する。正式Meaning受理は既存#267を経由し、Owner未決を残す。

最低受入票は次を並列に記録する：Source／Reference Revision、Content fingerprint、取得attempt・時点・結果／完全性、実消費byte／依存閉包、必要Read cohort、Consumer Purpose／Disclosure、構造候補、Accepted Meaningの受理時と現在資格、ArtifactSet必要member、実Effect Receipt、独立Expected、観測器、禁止推論、対象exact HEAD／CI。フィールド追加は既存Source Index／Evidence／CLIの投影可能性をまず調べ、汎用の公開schemaをここで決めない。

**停止条件**：読み取りscope、Not Foundの権威、全件性、Read Cohort、実消費byte、現在開示、Owner受理のうち、操作に必要な根拠が欠けた場合は「現在成立」「部分再計算を省略してよい」「実行可能」という結論に進まない。固定履歴・影響のないClaimのread-only利用まで全面禁止しない。

**Owner未決**：用途別鮮度期限と監視要件、負の取得結果の再利用時間・権威、非原子的Readを許すDomain／用途、pinned offline Buildの許容、権限失効後の既取得Cache開示・削除要件、内容同一時のOwner再受理省略条件。新ADRの採用決定はここで行わない。並行Draft[#2667](https://github.com/hat47x/tei-platform/pull/2667)の横断計画は別Ownerの編集範囲なので、本PRでは同じファイルを更新しない。

## 2026-11-03 第4追補：実装入口に限定したP0縦断受入

**状態：実コードと他製品の限定実験を照合した計画差分。以下のP0縦断は、この追補ではNOT RUN。** RF-01〜24の個別反例を増殖させず、着手対象・独立Expected・証拠の昇格条件を狭く固定する。

### 何が現在の実装にあり、何がないか

2026-11-03のmain観測版 0e18b916aa15bc7768e9abfb34318b98ee497cb3 のSourceを静的に確認した。

| 接続元と既存の実体 | 確認できる事実 | 証明していないこと |
| --- | --- | --- |
| internal/workspace/imports.go | WorkspaceImportがalias・path・namespaceと任意のsemanticDigestを持つ。ResolvedImportが正規化したNamespaceのSemanticDigestを保持する。これは利用側Managed Sourceとは異なる外部Namespaceの明示的な読取り | 汎用HTTP URL、ETag／条件付きGET、offline Cache、Provider別Permission、複数Provider Snapshotを提供する正式APIではない |
| reference/semantic-composition-preflight/external_acceptance_boundary_test.go | TestLocalSourceAcceptanceDoesNotWitnessExternalSemanticRevisionという既存の合成Goテストが、隣接する二Workspace間で、供給側のみ型の基底を変更する。利用側Managed SourceとSource acceptance recordが不変でも、外部Semantic Digestは変化する。旧semanticDigestでpinした場合はLoaderが拒否するようassertしている | **試験コードの静的確認**であり、今回のexact HEADで新たに試験を実行した証拠ではない。Hostによる外部意味の案件受理、現在のネットワーク取得権限、実HTTPやSHO Consumerの確認でもない |
| internal/sourceacceptance/verify.go | 現在のHost Readerを用いる利用側Managed Sourceの照合がある。Readerの信頼・RevocationはHost側の責務 | 別WorkspaceのSemantic Digestを自動で案件受理すること、Query／Buildの全依存・実行資格を証明すること |
| [SHO Composer Draft #13](https://github.com/hat47x/sho-composer/pull/13) | 独立Python実験で、異なるSourceが同byteでもSource Identity／Occurrenceは同一視できず、A→B→Aの中間変更は前後byteの一致のみでは検出できないという負例を記録（PR記載の17/17 PASSは当該独立試験の範囲） | TEI Loader／Go Product／SHO本体の保存・原子的Read／版Fence・ABA防止／納品文書の出典再解決が実証済みという主張ではない |

**接続を分ける理由**：外部Semantic Importの選択対象（Namespace／意味Digest）と、SHO保存文書の出典参照（Document Source Identity／Occurrence／Revision／参照位置）は役割が異なる。同じ内容Hashの二ファイル、同じSemantic Meaningの二Source、移動した同一Sourceを混同しない。Source位置が変わっただけのケースではMeaningの全面再受理を前提にせず、出典・Review・現在開示・再解決だけを別判定する。SHOの元文書をTEIの自動Rewrite対象へ昇格させない。

### 最初の縦断は「外部WorkspaceのSemantic Import」に限定する

第一段は**現行の実Go Workspace Reader／Loaderを使う二Workspaceの固定Fixture**。ネットワーク・汎用Resolver・正式DSL文法・新Cache・Studioが実装済みであることを前提にしない。既存テストを独立Expected付きで再検証し、そのRead-only結果を後段に渡す。

| 最小縦断Case | 固定入力・独立Expected | 先に照合する既存成果と停止条件 |
| --- | --- | --- |
| V0-1 基準取得 | Consumer CのManaged SourceはrC、Provider PはNamespace Nの意味Digest dA。CのSource Index／World／Resolved Importを別列に取得。宣言Alias・Scopeと実Provider、元Source位置を明記 | 既存WorkspaceImport／ResolvedImport／Source Indexの現在の表現を使う。元の供給側生byte・改版が露出しなければ空欄ではなく未確認と記録 |
| V0-2 ProviderだけA→B | CのManaged Source・ローカルSource acceptanceが不変でも、Pの意味Digest dB≠dAとなる。旧dAで明示pinすればLoad拒否。未pinのLoader成功＝Owner受理ではない | 既存TestLocalSourceAcceptanceDoesNotWitnessExternalSemanticRevisionを**まずexact HEADでfocused実行**。ProviderへアクセスしたbyteとLoaderが実際に消費したbyteは別の確認欄 |
| V0-3 位置変更・内容同一 | 移動または同byteの別Sourceを入力。Semantic Digest一致の可能性と、Locator・Source occurrence・Source Index／引用更新の必要性を分離。元Sourceとの意味的同一性は未決のままにできる | #2505と既存[#2626](https://github.com/hat47x/tei-platform/pull/2626)のSource移動証拠を比較する。Import pathの変更を無条件の自動Relinkとして扱わない |
| V0-4 取得失敗・partial／offline候補 | 存在しないProvider path、権限外Scope、途中読取りの不足を**分けて注入**。実際のLoaderが返すError／欠落を観測し、未実装のHTTP状態コードを模擬成功として公表しない | 取得Errorを外部Reference Observationへ正規化する正式入口が無ければNOT RUN: no product observation path。optional・未生成を暗黙の空集合としない |
| V0-5 A→B→A・固定入力競合 | 変更履歴を独立Oracleで与え、前後digest一致と中間変更履歴の一致を別評価。版Fenceが得られなければ「中間変更なし／同一Snapshot」の資格を与えない | #1017／#2595。過去のSource Observation・Accepted Meaningと、現在のReview／Grantの自動復活なし |
| V0-6 保存文書の参照とAI提示 | D1にr1 Source occurrenceを固定し、別版r2へSource移動、D1の保存・再読取／引用再解決を別に確認。旧Source内容が読めても今の開示権限が無ければAI送信しない | TEIのRead-only出典情報とSHO #13のSource ID/Occurrence候補を比較。実SHO Reader／保存・外部AI受信は**別Owner実験・NOT RUN** |

この表はRF群に対する**実行順を絞る横断受入票**であり、もう一つのFixture集や実装Schedulerではない。V0-1／V0-2を最初の必須縦断とし、結果の不足によってV0-3〜6を細分化する。実Sourceのない合成HTTP応答だけを最初のP0合格にしない。


### P0実装に先立つ境界の訂正：管理Sourceと明示Importを別扱いにする

本節はRF-23などで「Scope外のpathを拒否」と表現した条件の**適用単位**を明確にする。単純な「利用側Workspaceのroot外をすべて拒否」は現行機能と矛盾するため採用しない。現行main（観測 0e18b916aa15bc7768e9abfb34318b98ee497cb3）の internal/workspace/imports.go を静的確認した結果、次の区別が必要である。

| 読取入口 | 既存の仕様・観測 | 正しい境界と未証明範囲 |
| --- | --- | --- |
| 利用側の管理対象Source（sources／Source Index） | ワークスペースに属する管理対象ファイルの列挙。#1002／#1017が物理root越境と実読取byteを扱う | 管理対象Sourceの暗黙な越境は拒否。Source Indexの保護を、明示Importの禁止へ拡大しない |
| 利用側が明示指定した外部Workspace（imports.path） | 宣言はcleanな相対path。テストで ../provider を明示する。選択した別WorkspaceをProvider loaderで開く。imports.pathとsourcesは別欄 | 明示選択したProviderルートが利用者／Hostの許可対象かを確認する必要がある。相対pathに .. があるという理由だけで一律拒否しない。現行Loaderのpath構文検査＝実効path／symlink／供給元の認可保証とは主張しない |
| Provider側の管理対象Source | Provider自身のWorkspace loaderを使う。選択したNamespaceのProvider所有を要求し、他Providerからの推移Importを暗黙に再公開しない | Providerの管理SourceはProvider自身のroot境界で検査。Namespace明示選択・Semantic Digest pin・正規化意味の検査と、供給元Authority／案件Owner受理は分離 |
| 将来のHTTP／OData等Provider | 現行のWorkspace Import構文では直接扱っていない | URI、redirect、資格情報、取得Snapshot、404の権威、Cacheの期限は当該Provider契約で観測してから分類。ローカルPath実証をHTTP実装済みと扱わない |

**RF-23のExpectedを精密化**：宣言されたProviderルートそのものが利用側root外でも、許された明示Importとして正当な場合がある。拒否するのは「明示選択・認可されたProviderの実効ルート／Namespace／読取範囲から外れた実体」や無断のredirect／symlink越境であり、単なる利用側root外の存在ではない。実効path確定の原子性・競合時の保護は依然として未証明ならunknownとする。

### 既存テストの性格と、独立Expectedを備えた最小受入Packet

既存 TestLocalSourceAcceptanceDoesNotWitnessExternalSemanticRevision は、変更前のLoader出力から得たSemanticDigestを後続のPin値として使い、Provider改変による不一致と利用側Source照合の独立性を検査する。これは有用な**現行動作の特性試験**だが、そのPinの正しさまでLoader自身から独立に期待した試験ではない。V0-1／2の受入は、少なくとも次の二種類のExpectedを独立に固定して補強する。

| 結果の列 | Loader結果を見ずに先に固定するExpected | 観測する実入口 |
| --- | --- | --- |
| 構造・意味 | Provider AのProcessId基底型がstd.identifier、Bがstd.text。利用側Watchの参照は同じOS Namespace。選択Namespaceの件数、同名別Provider・推移再公開禁止も入力で確定 | 実Workspace Loader／World／ResolvedImport |
| 変更判定 | 供給側のみA→BならConsumer Managed SourceのSource Index／Model Revisionは維持し、Imported Semanticは変更。旧Aの明示PinはBを拒否 | 実LoaderとSource Index、現在の特性テストと別のOracle |
| 参照元と権限 | Providerの宣言pathと供給側管理Source、実効root、必要範囲を記録。現行機構にReaderのPermission／Snapshot／Revision証拠がなければ「未確認」を期待し、捏造しない | 実Source ReaderとProvider Loader。認可・現在性は別Owner |
| 現在資格 | Source-onlyのVerifyCurrentが成功しても外部MeaningがOwner受理済み・実行可能とはならない。未受理の状態をExpectedとして固定 | 既存sourceacceptance Reader。#267との統合経路は後段 |

最初の再現コマンド候補は次とし、**exact HEADのtrusted checkoutで実行した場合だけ実証済み**と記録する。

    go test -count=1 ./reference/semantic-composition-preflight ./internal/workspace ./internal/sourceacceptance

観測記録には実行版、試験名、投入A/BのSource byteと対象、期待するSemantic Shape、実際のProvider解決、観測したDigest／Index、失敗・不足範囲、実行環境を付す。必要に応じ固定したExpected Digestを追加するときも、Loader出力を転記して自己正当化せず、独立にレビューした基準を使用する。実試験が現在のschemaで取得できない項目は、後付けで成功値を生成せずunverifiedとする。

### 検証を置く時点：Go単一バイナリと外部Hostの双方に適用する

| 段階 | 前倒しできる確認 | 実行時まで保持する確認 |
| --- | --- | --- |
| DSL編集／Language Plugin | 宣言された参照、Source span、構文、静的依存・Pinの候補、必須性、診断／修正Preview。非信頼コメント／文書中の文字列を外部Source読取・Tool起動命令へ格上げしない | Source取得・Host GrantはParserの権限にしない。編集時の修正候補は現行Sourceへの適用前に版・Scopeを再確認 |
| Workspace Load／Build | 固定Source、明示Import、依存Closure、Role/Operator pin、必要ArtifactSetを検証。再現可能なビルドでは外部Importの内容が固定できた範囲をEvidenceへ束縛。Pin省略を許す開発作業と再現性のClaimを分ける | Providerの読取能力が無い状態でunknownを成功扱いしない。Source-only Fingerprintの一致だけで外部Importの固定入力を証明しない |
| 生成Goバイナリ／外部Targetの実行 | 既に検証済みの純粋変換・固定内容の各呼出しに、毎回フルParser／外部Reference全件検査を要求しない（性能方針は#565／#2667 §25）。必要な固定Bundle／Build資格は入力として引き渡す | mutableな実Provider Read、現在のPolicy・Disclosure／Grant、Effect前のAdmission、Commit不明時の独立Receipt／再観測は対象Host／Provider境界で検査する。TEI独自の共通Hostは要求しない |

**採否条件**：この3段階を一つの「fresh/approved/ready」にまとめない。Go実行時の費用削減を理由に、現在権限、部分取得、実Provider整合性、Commitの不確実性を省略しない。一方、静的なSource/Grammar検査を毎回の純粋Operator実行へ繰り返し移す設計も採用しない。性能測定はConformance後の別試験で行う。

### 現行mainの既存テスト再利用と、本当に未接続なP0

**追加試験の前に、すでにあるReferenceを再使用する。** 現行main `0e18b916aa15bc7768e9abfb34318b98ee497cb3` の以下のGoコード・テストを静的照合した。mainのGitHub Actions「TEI PR・main 健全性確認」[run 41595750765](https://github.com/hat47x/tei-platform/actions/runs/41595750765) はこのexact HEADでcompleted/success。ただし、このPR #2743のHEADで実行した結果ではなく、個別試験の独立Oracleを自動的に証明するものでもない。

| 現存する入口 | すでにテストが記述する範囲 | このPRで補強する未証明事項 |
| --- | --- | --- |
| internal/workspace/imports_test.go | 明示ImportのProvider／利用側Source所有分離、循環／曖昧・推移再公開の拒否 | 宣言Providerの現在Authority、外部配布元の本人性、案件Meaning受理 |
| internal/workspace/import_digest_pin_test.go | 旧Pin一致、不一致拒否、同byte Source移動や無関係Namespace変更で選択NamespaceのPinを保持 | Digestを独立に固定したOracle、供給元raw byte／Version履歴と実消費byteの同一Attempt束縛 |
| internal/workspace/import_source_test.go／import_source.go | LocateImportedSourceがProviderを再Loadし、解決時のSemantic DigestとSource anchorを照合。移動時のProvider内Path変更、stale・別Namespace Subject拒否 | **先のConsumer Loadと後の再Loadが原子的な一Snapshotである保証はない**。A→B→Aや実Source Read raceの歴史を、照合成功だけで消去しない |
| internal/workspace/source_snapshot_test.go | ロード済みのManaged Sourceが変更前のbytesへ束縛される範囲の回帰ケース | 別Workspace Providerの全Sourceを、Consumer/SHO/Runtimeまで一貫して固定したとは主張しない |
| reference/semantic-composition-preflight/external_acceptance_boundary_test.go | Source-only受理が外部ProviderのSemantic改版を承認しない既存の合成テスト | Ownerによる外部Meaningの案件Decision、現在のHost/Provider権限、後続Effect |

**P0の作業を縮退する判断**：これらのGoテストを新Suiteとして複製せず、まず既存のfocused testを現PRのbaseとexact HEADで検証する。その後、V0-1／V0-2の不足する独立Expected（型／対象Scope、Provider A/Bの選択、旧Pin拒否、利用側Sourceの不変性）を**最小一件**の比較票に追加する。新たなGoテストが必要なのは、この比較票の独立性や消費byte／現在Readerに関する断定を既存テストが表現できないことを確認した場合のみ。RF-03／RF-10／RF-23等の原子的Snapshot・Permission・HTTPは、その能力を持つ実Provider入口が得られるまでNOT RUN／unknownと記載する。

**横断計画との分担**：[Everything is Data Draft #2667](https://github.com/hat47x/tei-platform/pull/2667) の§30（R0〜R3）は外部Import→Studio診断→AI Patch→Consumerの一改訂を所有する。本書はその **R0へ渡す既存ReaderのE0受入と独立Expected、E1以降へ進めない停止線**だけを所有する。Source原文の無変更保存・UTF-8位置・AI修正のApply資格は#2745／#2674の既存担当、Build省略は#2600、意味受理は#267、実Effectは#1174に残す。§30と同名のRunner／Gate／Fixtureは新設しない。

**read-sideとwrite-sideの非連続性**：外部ImportのDigest固定、SourceIndexの一致、複数入力のread snapshot（#1708）が成立しても、読み込み後に業務行・Grant・Provider状態が更新された場合、保存時の不変条件・権限は保証されない。参照用Commandの成功／Committed表示、計算値の一致、単発のSource Reader PASSを「実業務DB確定」に昇格しない。書込みが必要な案件では、既存[#1174](https://github.com/hat47x/tei-platform/issues/1174)の担当が同一Effect確定境界で利用主体、対象Action・Resource、期待版／read-setの適合、Authorityと副作用・Commit／Receipt・独立観測を確認する。Provider固有の保証が足りないならunknown／拒否、ACK消失時は無断再送しない。**read-only P0を進めるためにwrite-sideの完了を前提条件にはしない。**

### Evidenceの最小受渡しと段階停止（形式は暫定）

| 証拠段階 | 進める条件 | 到達しても主張しないこと |
| --- | --- | --- |
| E0 Source／Producer | exact Repo HEAD、実Reader／Loader、投入Source byte、供給側／利用側のscope・Revision、明示pin、取得失敗を別々に記録 | Parser／Build実消費、現在外部HTTP、Ownerの業務意味受理 |
| E1 実Consumerへの受渡し | 同じAttemptのLoader/Model Source Index／必要なMemberを使い、Inspector／CLIでどの引用・不明が残るか観測。旧版による後着上書きを拒否 | 実SHO保存文書・AI開示・Editor・Targetの統合資格 |
| E2 消費Byteと資格 | Readerから消費までの固定Snapshot／Version fenceの有無を確認し、実Source／ReferenceをConsumerが読んだ証拠、完全な依存Closure・必要ArtifactSetと現在Policyを別判定 | Owner Decision、Runtime Permission、Commit Receiptを自動で付与すること |
| E3a Owner Meaning（実行とは独立） | 既存#267／#1016の案件Decisionが、選択した外部Resource・意味Digest・適用Scopeと結びつくか、適切なOwnerが判断 | Source受理・Digest一致だけで業務意味を自動受理すること |
| E3b Runtime Effect（独立Owner） | 実業務Actionでread-set／対象版／現在GrantをEffect確定境界で再評価し、#1174のProvider Commit／Receiptと独立Effect観測を別列で確認。案件用途によりE3aを前提にする条件はOwnerが決める | Source／read-only受理、参照CommandのCommitted表示、ACK、旧Receiptから実DB確定・再送安全性を推論すること |

**この計画の具体的Done条件**は、少なくともV0-1／2について、実SourceとLoader出力を固定したfocused Go試験、Provider A→B時の正しいpin拒否、利用側Source受理との独立性、独立Expectedの一致をexact HEADとRunへ結び付けること。Source閉包・供給側Revisionが取得できない場合は、不足を出力し、対象Claimを未確認に止める。E0の成功だけでE1〜E3を完了にしない。

**変更停止条件**：新しいURL Resolver／抽象Plugin方式／正式DSL Grammar／全Provider共通Snapshot／自動Source Relinkを先に決める必要はない。導入が実証上必須となった時点で、その最小要求だけを既存[#988](https://github.com/hat47x/tei-platform/issues/988)／[#1570](https://github.com/hat47x/tei-platform/issues/1570)／[#2505](https://github.com/hat47x/tei-platform/issues/2505)へ戻す。#2600ではE2の実消費依存が未証明ならBuild skipを採用せず、#958認知層は歴史と現在資格の説明に徹する。#1124／#2516のStudio／AIへはE1の資格が得られた範囲だけを投影する。並行[#2667](https://github.com/hat47x/tei-platform/pull/2667)の本文は本PRでは編集しない。

**検証・統合上の注意**：本追補はGitHubの実ファイルをread-onlyで調査して作成した設計であり、Go試験・SHO試験・実Provider・Windows／HTTP・AI送信をこの変更で新たに実行していない。受入済みADRやdocs/status.mdの実装状況は更新しない。PR #2743はDraftを保ち、最新mainとの差分・exact Head／対象Path・Ownerレビューを確認してから統合判断する。

## 2026-11-03 第5追補：Import束縛の証明限界をP0出口条件へ接続

**目的**：前追補のV0-1／2は外部Namespaceの意味変更を見分ける入口として成立するが、その結果を**「実Loadから完全に束縛したSource Index」「増分Build skip可能」「実Consumer現在資格あり」へ上げるための不足**が残る。これを既存APIの実際の失敗契約に照らして明文化する。RF-01〜24、V0-1〜6、E0〜E3bは追加・改名せず、この節をレビューの収束点とする。新Host／Registry／Importer／Manifest／Gateを提案しない。

### 現mainで確認した三つの独立した証拠と意図的な停止

確認版：main 0e18b916aa15bc7768e9abfb34318b98ee497cb3。以下は**静的なコード・テスト確認**であり本PRの新しい実走ではない。

| 現行の入口 | 確認した保証 | 明示的な非保証・停止 |
| --- | --- | --- |
| Workspace.Load + Result.Imports／WorkspaceImport.SemanticDigest | 外部Providerの所有する選択Namespaceを明示取得し、選択した正規化Semantic内容のDigestを保持。任意pinは内容変更時に拒否できる | Import元のraw Source全member、Provider実効root、実読取byteの一体Snapshot、AuthorityやOwner受理を自動束縛しない。Providerが暗黙に取り込んだ別Namespaceは推移再公開しない |
| Result.ManifestIdentity／BuildSourceIndex | Loadで読んだWorkspace manifestのPath＋SHAと、利用側Managed SourceのIndexを**別の証拠**として扱える | manifestのimports.pathを別Providerへ差替え、意味Digestが同じ場合も、利用側Managed Source Indexだけで選択元不変とは言えない。ManifestIdentityはSource Indexの一部ではない |
| BuildSourceIndexFromLoad（strict） | Load時に取得した利用側Managed Source bytesだけでIndexを作る。別Workspace Importが宣言・解決されていれば **ErrImportsNotBound** として拒否する。未解析管理Assetと非原子的な複数ファイルLoadの限界も明記 | これは外部Import対応が完了した成功状態ではなく、ImportのSource／Dependency closureがIndexへ束縛されないことの**意図的な未資格化**。BuildSourceIndexの通常成功やIndexedLoadedSnapshotの利用側限定Readを、strictな外部含有の全件束縛成功へ読み替えない |
| LocateImportedSource | Providerを再Loadし、消費側に記録されたNamespaceのSemantic Digestとの一致を再確認し、Provider側Source anchorをread-onlyで返す | 先のConsumer Loadと再Loadとの間にSourceがA→B→Aで変わらなかったこと、同一の原子的read cohort、Source内容のOwner意味受理・編集資格は証明しない |

**現存する拒否テスト**：internal/workspace/source_index_load_bound_hardening_test.go の TestBuildSourceIndexFromLoadRefusesWorkspaceWithImports は、明示Importを持つResultが ErrImportsNotBound に停止することを検査する。internal/workspace/import_digest_pin_test.go は選択Namespaceのpin／同byte移動／無関係Namespace変更を扱う。この二種類を新規の「Build成功」試験へ変換しない。strictの拒否を単に取り除くことも、Importやmanifestを無証拠に管理Source Indexへ混在させることも採用しない。

### 二軸で明示するP0の結果：意味の照合とInput-closure資格

| 観測された状況 | 選択Namespaceの意味・履歴に関する判断 | strictな外部Import含むBuild／現在Consumerへ渡す判断 |
| --- | --- | --- |
| 明示Import、pin一致、Provider Source位置を照合 | **選択したSemantic内容は期待Digestと一致**。構造候補・履歴read-onlyへ進める | ManifestIdentity・Provider原Source・実消費Closure・現在Authorityは別。strict IndexがErrImportsNotBoundなら **unqualified**。Business Accepted／Build skip／現行実Actionは不可 |
| Consumer Managed Sourceが同一、imports.pathだけ別Providerへ変更、選択Namespace Digestは同じ | **選択した正規化意味byteの一致**として記録可能。Provider Identity／出典・Authorityは再資格化対象 | Source Index不変からImport選択元不変と推定しない。ManifestIdentityを別観測。Provider別許可・現在性を検査 |
| Provider Source同byteで移動、同Namespace Digest | 値・意味内容の再計算は用途依存で再利用候補 | Source occurrence・Trace・引用の再定位と権限を再検査。SHO文書の旧引用を自動Rewriteしない |
| 暗黙の推移Import／外部未取得／Provider権限不足 | 解決した範囲のみ候補。未取得依存・空集合・不在を混同しない | strictな現在資格の成立を主張しない。停止は当該Claimへ限定し、証明できる無関係Claimの履歴閲覧は保持 |
| Source A→B→A、前後のDigest一致 | Aが過去と同じ値であることと、変更履歴・その間のAuthorityを分ける | 固定Provider Revision／実Input snapshotが得られない限り「中間更新なし」「Build／Owner資格復活」を主張しない |

**実際のProduct出口は二段階**とする。P0a（既存実Go Loader・Digest・宣言先変更・厳密拒否）は、固定A/Bの独立Expectedと当該exact HEADのGo Runで検証可能。P0b（外部Importを含むSource／manifest／Provider実消費の完全な束縛）は、**現行strict APIが意図的に拒否するため、成功しているとは主張しない**。P0aが成立した段階で別途P0bに必要な最小受渡しを評価し、ImportのSourceを束縛する形式、OSファイルSnapshot／Provider能力／ACLを正式採用するかは、それぞれの既存Ownerへ差分として返す。P0b未成立でもread-only解析や候補説明を全停止しない。

### 実施計画の変更（追加Fixture IDなし）

1. **現mainの実証済み範囲を再確認**：既存テスト TestExternalSemanticImportOptionalDigestPin、TestExternalNamespaceImportPreservesProviderIdentityAndSourceOwnership、TestLocateImportedSourceUsesProviderScopeAndCurrentFingerprint、TestBuildSourceIndexFromLoadRefusesWorkspaceWithImports の期待・失敗を静的比較し、必要になった場合のみ現PR HEADに対するfocused Go試験へ進む。過去のmain CI成功を現PR HEADの合格と表示しない。
2. **P0a比較票を完成**：同じConsumer Managed Source、Provider P1／P2、Namespace A／B、manifest bytes、選択Semantic Digest、Provider内Source occurrence、Source Index、Owner受理／現在Authorityを**別列**にする。ExpectedはProvider選択・意味内容とscopeから独立に固定する。「出力されたDigestを次のpinにそのまま転記しただけ」の自己整合をOracleとしない。
3. **P0bの停止を観測**：importsが宣言されている時、BuildSourceIndexFromLoad が返す ErrImportsNotBound を正の拒否Expectedとして記録。未取得・未束縛は空集合や失効済みではなく **unknown/unqualified**。どうしても完全なBundleが必要なConsumerのためだけに、Provider側Source／manifest／実消費byteを固定する最小手段を既存#1570／#1017／#2505／#2595へ評価依頼。合成テストのためだけに第二のCacheやRegistryを実装しない。
4. **後段へ正しい資格だけを渡す**：#2600へ「意味計算再利用可否」と「Artifact／Source citation／Consumer再資格化」を別列で渡し、#2667 §30〜32（横断R0〜R3／Import再利用）とは新Runnerを作らず対応表で接合。#1124／#2516にはread-onlyと現在開示の区別を渡し、#267案件Meaning、#1174実Effect、#988 DSL／Operatorの各Ownerを横取りしない。SQL／Data Contextの否定Schema依存（#2778）も同じく「値不変＝依存閉包不変ではない」という比較軸に限り参照する。

**P0aの実Go証拠（Draft PR #2786）**：[1件の独立Expected付きGoテスト](https://github.com/hat47x/tei-platform/pull/2786)（HEAD `af27e660ed12060dc77a6bd7e4964e4b3e32ce0d`）は、同内容のProvider P1→P2切替でのImport／Manifest差、Providerだけの意味変更と利用側Managed Source Index不変、旧Pin拒否、Provider raw Source anchor、strictな `ErrImportsNotBound` を対比する。[exact-head Repository Health #41641035468](https://github.com/hat47x/tei-platform/actions/runs/41641035468) は**completed/success**。同HEADで `go test -count=1 -timeout 60m ./...`（新テストを含む）・Go vet・gofmt・CLI validateを実行。**合成Workspaceを対象としたP0aの限定Go PASS**として記録する。Providerの現在Grant、実HTTP/OS Provider、原子的複数Source Read、Owner案件Meaning／Build skip／SHO・Studio／実Effectは合格にしない。

**P0bで初めて必要な証拠（未実証・採用前の候補）**：現行の `loadWorkspaceImports` は別Workspaceをロードしても `ResolvedImport` にはalias/path/namespace/意味Digestしか受け渡さず、Providerの当該Loadで実消費したSource byte・manifest・Source membership／Snapshotを一体のWitnessとして返さない。後続 `LocateImportedSource` はProviderを**再Load**するため、そのDigestとSource anchorが一致しても初回Consumer Loadとの間の無変更、同一Read cohort、A→B→A不在を証明できない。`BuildSourceIndexFromLoad` は外部Importがある場合の `ErrImportsNotBound` を意図的に維持する。

**P0b負例への実装移管（[Draft PR #2801](https://github.com/hat47x/tei-platform/pull/2801)）**：Provider原文の末尾空白だけを変えたとき、選択NamespaceのSemantic Digestと利用側Managed Source Indexは不変でも、旧`workspace.Result`を渡した後続`LocateImportedSource`はProviderを再Loadし、新しいSource fingerprintを返す。これは**Source参照を現在再解決する正常な振る舞い**だが、「初回Consumer Loadが消費したProvider原文を証明する」との誤読を反証する。独立に計算したraw byte SHA-256で対比し、既存のPin拒否やSource移動テストと重複させない。[PR #2801のexact-head Repository Health #41656612182](https://github.com/hat47x/tei-platform/actions/runs/41656612182) はHEAD `493ec88438c18a896f894672a426bfbe975b095b`で**completed/success**。追加したGo負例を含む既存Go全件試験の成功は、**P0bの未保証を示す振る舞いの実証**であり、P0b合格ではない。

**P0bの方式比較を実装順序として絞る**。A：同一Provider Loadの中でReaderが実際に消費したSource byte・Manifest byte・Source occurrence／membershipを保持し、対象Consumerへread-onlyで渡す（現行`sourceSnapshot`／`ManifestIdentity`の正しい所有範囲を拡張する最小案）。B：同じ入力を不変Bundleに収め、Loadも実Consumerも**そのBundle自体**から読み、Bundle identityと内容を照合する案。AをローカルWorkspace authoring／最小Go参照実証の第一候補とし、Bは配布・別プロセスBuild／再現要求に不可避なことが示された場合に比較する。どちらも正式API／Plugin方式／供給元Authorityではない。**Aだけでは複数ファイルの同時点原子性は証明しない**。BでもBundleの固定後に実Consumerが別の物理Sourceを再読込すれば資格は成立しない。

**P0b-Aの最小受渡し票（実装前の期待・追加Schemaではない）**：`consumer import alias/path/namespace`、`provider root identity`、`provider manifest path + raw digest`、`selected namespace semantic digest`、`provider-owned source occurrence/path + parsed byte digest`、`load-time managed membershipと未解析Sourceの有無`、`実際にConsumerが消費するentry/closure`を**同じProvider Loadの局所観測として並べる**。原文そのものを公開ログ・AI Contextへ無条件に入れない。必要Consumerの読み取り権限と開示資格はこの受渡し票の外側で再確認する。

| 一軸だけ動かす対照（既存実テストを再利用） | 意味計算の再使用候補 | Source／Consumerに必要な再資格化 |
| --- | --- | --- |
| Provider rawの空白だけ変更（#2801 Go PASS） | Namespace digestは同一のまま | Provider raw byte・引用／Span／出典の再資格化。古いResultのSource再解決を元LoadのWitnessとみなさない |
| Provider内で原文同byteのまま移動（既存Importテスト） | 対象意味は同一のまま | Source occurrence／Locatorと保存引用を再定位。SHO自動書換え・Owner受理は別 |
| 同内容のProvider A→Bへ明示切替（#2786 Go PASS） | Semantic Digestは一致し得る | Provider identity・manifest／permissionを再確認。Consumer Source Index不変を供給元同一とみなさない |
| Namespaceの意味だけA→B、旧Pin保持（既存／#2786） | 旧意味の現在利用は不適格 | Pin mismatch拒否。別のSource acceptanceや新しいOwner受理なしに自動復旧しない |
| 複数Sourceの途中競合・Manifest変更／ABA（未実証） | 一部値・前後Digest一致だけでは判定不可 | 実Consumed byteとread setの完全性／Revision fenceがなければunknown。強いSnapshotを主張しない |

**P0b-A同一Load局所Witnessの実Go結果と開示境界（[Draft PR #2812](https://github.com/hat47x/tei-platform/pull/2812)）**：既存`Result.sourceSnapshot`（Parser消費byte）／`ManifestIdentity`（原文Digest）を利用し、初回**同一Nested Provider Load**で選択NamespaceのParser-consumed Source集合とManifest原文をprivate transientに保持する。原案HEAD `89219f3d39d4972bb0b6a4afd666805b318f7d9b` は[exact-head Repository Health #41740703505](https://github.com/hat47x/tei-platform/actions/runs/41740703505) **completed/success**で、Load後に原文・Manifestを書き換えても旧byteがメモリから取得される限定的Go証拠を得た。ただし元案は`ImportedParsedLoadSnapshot`がpublic raw byte readerを返し、**明示Semantic Importから原文・Manifestの開示権限を推論してしまう問題**があった。

**P0b-A安全境界：原文だけでなくSource metadataも開示対象（Draft [PR #2812](https://github.com/hat47x/tei-platform/pull/2812)）**：安全修正HEAD `7a6071ed22a2be02b08f6a56a7585c56ba01dc01` の[Repository Health #41742064273](https://github.com/hat47x/tei-platform/actions/runs/41742064273) は **completed/success**。これは前回の「原文Readerを非公開・metadataのみ公開」のGo試験結果であり、**metadataに含まれるProvider内のSource path／Manifest path／原文fingerprint／存在件数まで無条件開示してよいことの証拠ではない**。#1178 の出典情報開示、#2516 のConsumer用途相対Disclosure、#2515 のAI/CI/ログ送出境界と照合すると、公開範囲はまだ広すぎる。

**更なる修正（HEAD `8b90b606137d5d4602b57d8f0a318a783dbf4edd`、exact-head CI未確定）**：新設したpublic `DescribeImportedParsedLoad` とexported `ImportedParsedLoadEvidence` を外し、今回追加した詳細な原文／Manifest／Source occurrence／fingerprintはすべて`workspace` package 内のprivate/transientな証拠とする。既存公開`Result.Imports`の明示Provider path／namespace／意味Digestは現状維持するが、これは以前から存在する**選択した依存の表示**であり、そこからProvider内の追加Source一覧やhashを無条件公開する理由にはしない。private `importedParsedLoadSnapshot` は同一Nested Provider Loadに保持したSource＋Manifestのbyteを再読込みなし・clone-on-readで返すが、**外部Consumerへ出すAPIではない**。元のProviderの現在のRead権限と対象Audience/目的のDisclosure許可を検証できるまで、path／hash／本文をAI/SHO/CLI/Web/CI出力へ追加投影しない。

**新規診断の非漏洩・exact-head CIの区別（[Draft #2812](https://github.com/hat47x/tei-platform/pull/2812)）**：同一Load局所Witnessをprivateに保持し、明示Semantic Importだけでは原文・Provider内部Source path・raw fingerprint・存在件数を追加公開しない。さらに`captureImportedParsedLoad`のmissing-byte／invalid-path／重複Source等の新規診断から内部Source pathの反射を防ぎ、無害canaryで確認する。これは#1178のTrace/Statusと#2515の内部Observer対外Viewerに接合する**新設helper限定の対策**であり、既存全Provider Loaderエラー・OS/AI/Network送出経路まで保護したとは主張しない。

[HEAD `c535d9f56302f3b8bee4968f18bac1196038d786` のRepository Health #41785834527](https://github.com/hat47x/tei-platform/actions/runs/41785834527) は**completed/failure**。ジョブログで`go vet ./...`と`go test -count=1 -timeout 60m ./...`は**PASS**、`gofmt`のみ新規テスト1ファイルの整列差分で**FAIL**と確認した。書式修正HEAD `0ca9007d661dd486cfd223eac17e9cd7a4024e45`の[Repository Health #41786862912](https://github.com/hat47x/tei-platform/actions/runs/41786862912) は結果確認まで**未確定**。旧HEADのGo PASSを新HEADの全CI PASSに転用しない。

**横断引渡しの実証順（構想ではなく未実証の受入条件）**：最初は同一合成Providerの内部Observer→実際のConsumer Adapterという**読取り専用一経路**に限定する。(1) 初回Provider Loadの選択Namespace parsed byte・Manifest指紋と実Consumerが消費したbyteを同一Attemptに束縛、(2) Source／Manifest原文、path・fingerprint・件数、エラー・Trace・JSONを**Policyが許すAudienceと目的**へだけ投影、(3) ProviderのSource改版・Grant失効・未選択Source・scope不一致を一軸ずつ変えて旧Snapshot保持と「現在も開示可」を区別、(4) 追加Sourceが無関係か、未取得／非出力・否定依存であるかをProvider実Readerの観測なしに決めない、の順で評価する。#1180の共通Scenario・#2516/#1178のDisclosure・#2600のArtifact再利用／性能測定の既存担当を利用する。**詳細証拠の閉包が未証明ならBuild skipも実Consumer資格もunknownのまま**とし、正しいread-only過去観測を全面否定しない。Sourceから実Provider／Host／Commitの成功まで直結したという表示もしない。新共通Policy/Runner/Registry/Hostを追加しない。

**限定検証と次の本当の出口**：安全回帰は`json.Marshal(Result)`でProviderの内部ファイルpath・raw fingerprint・詳細Evidence項目が混入しないこと、private readerの選択Scope・byte不変性、従来のstrict `ErrImportsNotBound` を確認する。さらに**Goの通常構造体整形`fmt.Sprintf("%+v", Result)`は非公開map/スライスの値を文字表示し得る**ため、今回追加したProvider原文mapをopaque pointer値に、Manifest raw bytesをprivate pointer型に変更する。raw byteとManifestの独立Expectedを通常`fmt`の出力に照合する限定負例を追加し、実行ログ・Harness/AI送出時の新しい漏出経路を抑える。**これは既存Consumer `sourceSnapshot`など他のフィールドの汎用`fmt`非漏洩、Reflection・全OS/Egressの遮断、現在Grantを保証するものではない。**#1180のAI生成可能Scenario／CLI・Web共通経路に渡す場合も、同じFixtureで**内部Observerと許可された外部Consumerの非対称投影**を実際に通すまで受入としない。ここで第二Policy Engineを作らず、既存 #1178/#2516 がDisclosureのOwner、#2600がCopy費用と再利用資格、#2667が実Consumer、#1174が実Effectを維持する。

**新しいraw保持の範囲とexact-head検証（2026-11-04）**：`Result`内の選択Provider parsed byte／Manifest rawは一回のNested Provider Loadからprivate transientに保持する。原文Readerはpackage-private、返すbyteはclone-on-read、`json.Marshal(Result)`への新規Provider詳細の自動混入と通常`fmt.Sprintf("%+v", Result)`の新設raw map/slice展開を限定試験で拒否し、`BuildSourceIndexFromLoad`の`ErrImportsNotBound`と`FullInputClosure=false`を維持した。[実装Draft #2812](https://github.com/hat47x/tei-platform/pull/2812) **現HEAD `634ca10e1e5c23203357faa13d07deb801c497fe`** の[Repository Health #41787598611](https://github.com/hat47x/tei-platform/actions/runs/41787598611)は**completed/success**。同HEADの`go test -count=1 -timeout 60m ./...`／Go vet／gofmt／CLI validateを含む限定的な実Go証拠として、従来の「現HEAD未検証」を更新する。旧HEAD `c535d9f`のGo PASS／gofmt FAILは歴史記録として分離する。**実CLI/AI/SHO等の公開ConsumerへのSource開示は一切資格化していない**。既存Consumer `sourceSnapshot`その他のログ耐性、Go reflection・`%#v`・OS Egressも別問題で、無条件に構造体全体をログ送出する運用を認めない。

**P0b-Aから次に進む受入順序（Reader局所保持 ≠ 必要Source完全消費）**：
1. **P0b-A private evidence**：同一Nested Provider Loadで選択Namespaceの実Parser消費Source byte、Manifest raw、原文fingerprint、Source occurrenceを取得済みか検査。取得したbyteが後から不変であることと、同時点の複数ファイルSnapshotであることは分離。
2. **P0b-closure accounting**：Workspace manifestに宣言されたroot、実際にParserが読んだSource、Managedだが未解析のファイル、選択外Namespace、負の依存／未発見Source、使用したPlugin/Generator/Queryを区別して「必要入力集合が閉じた」と主張できるかを検査。未観測／許可外のroot・拡張子・部分取得を空集合にして成功扱いしない。現在の`FullInputClosure=false`はこの未証明を正しく表す。
3. **Consumer read-only実経路**：#1178/#2516の用途・Audience・現在Policyが認める範囲のみ、実際のConsumer Adapterへ与えたbyteが1の保持byteと同じAttemptか検証する。許可されたローカル内部Observerと、未許可外部AI／SHO／CLI viewerの非対称投影を比較。原文以外にPath、Hash、件数、診断理由も開示対象。**先にraw byte公開APIを新設しない**。
4. **Build／実行へは別資格**：#2600で意味計算再利用とArtifact／Source引用の再資格化を分ける。#2582/#2586には同一Contextの非出力依存・Scope・変更epochを渡す。実Host Grant／Commit／Receiptは#1174、Owner Meaningは#267。ここで全Provider原子的Snapshotや現在の実行権限を捏造しない。

**2026-11-04の新しい反証：CLI validation成功は読取閉包の証拠ではない**。実CLI再現の既存Issue [#2822](https://github.com/hat47x/tei-platform/issues/2822)では、利用者定義のないWorkspaceでも`validate -json`が`ok:true`となり、`model catalog`が13件の標準`std`定義を示すため、単なる成功やモデル件数は「利用者宣言を実消費した」と同値ではない。[#2823](https://github.com/hat47x/tei-platform/issues/2823)では、未知／不存在`sources` rootの診断不足と、`workspace-files`には列挙されるが`validate`がParserで読まないファイルの差を再現している。これらは**別の実CLI版 `97aba8146`での報告**であり、#2812のHEADで同一経路を実行した結果ではない。外部AI／TAI／Buildの資格化では`validate.ok`、`model catalog`件数、`workspace-files`の列挙、`sourceSnapshot`のparsed byte、Provider/Builder実消費を**別列**に記録する。対応は#2822/#2823のOwnerへ委譲し、この計画に新Validator・読取Registryを実装しない。[#2824](https://github.com/hat47x/tei-platform/issues/2824)のCLI discovery（19件）と文書（13件）の契約差も、Consumerが前提にする`schemaVersion`と契約外`world`の確認事項として既存Ownerへ委譲する。

**次の最小受入票（再起票ではなく既存Scenarioへの追記）**：`Input Provenance = Provider/Manifest/selected parsed source`、`Coverage = declared/managed/parsed/ignored/unknown`、`Consumer Consumed = same attempt byte or unobserved`、`Disclosure = permitted/denied/unknown`、`Claim = source-historical/semantic/Artifact-reuse/current/Effect`。#1180の既存Scenario・#2667のW0〜W6を再利用し、件数・hash同一だけで完全性を自動認定しない。閉包や実Consumer未検証を理由に過去のread-only証拠まで消去しない。新しい正本、Host、Registry、Cache、Runner、汎用Permission Engineは追加しない。

### 2026-11-08横断受入：ImportのSource証拠・Plugin Inspector・Design Context・Verified BOMを接続する

**版と実装状態**：本節の検査基線はmain `29d367198af2281fc8a390eda959ba91088d8542`。本設計の前段を管理した[PR #2743](https://github.com/hat47x/tei-platform/pull/2743)は**main統合済み**。一方、`internal/workspace/import_loaded_snapshot.go` は当該mainに存在せず、[P0b-A #2812](https://github.com/hat47x/tei-platform/pull/2812)（旧HEAD `634ca10e1e5c23203357faa13d07deb801c497fe`、historical CI success）は**Draft・未統合**、旧実装は現在mainへ再検証なしに移せない。旧負例[P0b #2801](https://github.com/hat47x/tei-platform/pull/2801)は後継#3479へ集約して未統合クローズ。以前のCIは**旧HEADだけの限定実証**であり、現mainのLoader実装・同一Attempt結合に自動転記しない。#2812のprivate raw保持を新mainへ移植する場合は、現行`imports.go`／`loader.go`／開示境界に対する最小差分を再評価する。旧PRの全履歴をそのままrebase/mergeして完成扱いしない。
### 2026-11-07横断判断：外部Importの読取証拠をPlugin-first Inspector／Design Context／Verified BOMへ接続する（現mainを基準にした受入計画）

**版と実装状態**：本節の調査基線はmain `5c28427b4e705f0428d2dc2f8e065892fc96d607`。本設計の前段を管理した[PR #2743](https://github.com/hat47x/tei-platform/pull/2743)は**main統合済み**。一方、`internal/workspace/import_loaded_snapshot.go` は当該mainに存在せず、[P0b-A #2812](https://github.com/hat47x/tei-platform/pull/2812)（旧HEAD `634ca10e1e5c23203357faa13d07deb801c497fe`、historical CI success）は**Draft・未統合**、現main比較では1610 commit behind。負例[P0b #2801](https://github.com/hat47x/tei-platform/pull/2801)もDraft・未統合。以前のCIは**旧HEADだけの限定実証**であり、現mainのLoader実装・同一Attempt結合に自動転記しない。#2812のprivate raw保持を新mainへ移植する場合は、現行`imports.go`／`loader.go`／開示境界に対する最小差分を再評価する。旧PRの全履歴をそのままrebase/mergeして完成扱いしない。

**統合する判断は「意味の選択」と「その意味を実装・利用した証拠」を分けること**。Workspaceの`ResolvedImport`が示す外部Semantic選択、#951/#1016のPlugin Contract/Provider選択、#1158/#1824のCLI/Studio Inspector実呼出し、#3262のTask相対Design Context充足、#3267のVerified Composition BOMは別の資格である。一つのFingerprint・JSON成功・選択名を、それより後段の実Provider呼出し／必要Source完備／現在Disclosure／実Target許可へ昇格しない。Core以外をPluginへ構成する[統合済みPlugin-first計画 #3277](https://github.com/hat47x/tei-platform/pull/3277)を前提とし、第二のResolver／Policy Engine／Test Runner／Hostを作らない。

| 実証の境界 | 最小の同一Attempt観測（既存機構を使用） | 不足を検出したとき |
| --- | --- | --- |
| Source→意味 | Manifest宣言root・managed一覧・Parser実読Source byte／occurrence・必要な未読／否定依存・選択Namespaceの意味Digestを**別列**に保持。現mainではP0b-A private witness未統合 | `validate.ok`・モデル件数・`BuildSourceIndex`だけなら`closure=unknown`。#2822/#2823、#3262の既存検査へ渡す |
| 意味→Plugin実装 | 明示選択されたContract/Provider/Capability、実Source/Pluginロード、実Buildに入ったProvider、実呼出Providerを個別照合。未選択Capabilityは無条件依存へ入れない | Provider B選択→A呼出し、選択Provider欠落時の黙示fallback、必要Trait未履行は`not-qualified`。#951/#1016/#3278/#3279へ渡す |
| Plugin→Inspector Consumer | 同じSubject/Scope/Revision・Provider実呼出し・実戻り値・Source/World fingerprintと現在のAudience/Grantを、CLI／Studioの各読取り経路で照合。raw Source/Path/Hash/件数・診断の開示は別に判定 | メタデータや`fmt`ログ、拒否理由からの私有情報流出を含め不一致・無権限を拒否。#1178/#2516／[Draft #3371](https://github.com/hat47x/tei-platform/pull/3371)へ渡す |
| Inspector→設計・テスト | #3262のTask別Independent Required（Rule/Relation/Role/Source/位置・依存）と、実Loader/Parser/Meaningから受領したScope付き集合の差。#1180 Scenario／#2514独立Expected／C0・C1を別集計 | JSON正常・World FP一致でもRequired欠落／part切断／旧回答の再利用は`Design Input unqualified`。外部Harness受領は別のTransport判定 |
| Plugin×Target→BOM | #3267のProvider v1/v2×Consumer v1/v2を、**各組合せ**の選択Lock・実Build/Link・実呼出・独立Expected・ArtifactSet/Target/Versionで照合 | Matrix生成のみ・Suite宣言のみ・旧版CIだけならBOMは`candidate`。実Host Grant/Effect資格は依然別 |

**最初の縦切り（新しいFixture/Runnerは作らない）**：#1158/#1824の既存`inspect`で、(a) 選択Provider B→実呼出B→許可された読取りConsumerへの実結果、(b) B選択→A呼出（返値が偶然同じでも不合格）、(c) `validate.ok=true`だが必要Source/Ruleが未読／未到達、(d) private Provider Source/Path/Hashが許可外Consumerへ出る、の四対照を**既存#1180のScenario＋独立Expected**へ対応させる。#3371はStudio Host Serviceを通す候補のDraftであり、現main CLIの手書きInspector経路・Providerの実選択・CLI/Studio両Consumerの同一実呼出しが成立したという証拠ではない。#3262の既存11 Tool Reply／32 Controls等も、Design Context全範囲の完備・外部Harness全受領／未知Ruleへの盲検生成を証明しない。

**停止と非過剰拒否**：Source closure欠落は**閉包を必要とするBuild skip／BOM資格／Source引用／テスト生成**を`unknown`に止めるが、既に許可されている過去のread-only意味表示まで自動無効化しない。未選択Pluginが追加されてもEffective Selectionと実Build閉包が変わらなければ、無関係Artifactを全面失効させない。反対に選択・Provider能力・負の依存・Toolchain・Actual Inputが変わる場合は、意味Digestが同一でも必要Claimを再資格化する。#2600へ誤再利用／過剰失効／Load・copy・invoke費用を別指標として返す。実Owner Meaning・現在Authority・DB Commit／Receiptは#267/#1174へ保留し、P0b private snapshotやBOM候補から権限を発行しない。現段階は**設計受入の絞込みのみ**であり、P0b-A main実装・実Provider／CLI↔Studio／外部Harness・2×2実Build、実Owner受理を新たに実行したとの主張はしない。

**World／Consumer Source Indexに現れない外部Source-only改訂（新規の実Go反証対象）**：2026-11-08にmainへ統合された[Context→Scenario版束縛 #3477](https://github.com/hat47x/tei-platform/commit/29d367198af2281fc8a390eda959ba91088d8542)は、実CLIで**同じWorkspaceのSource末尾LFだけを変更**し、Worldの意味が同一でもSource Index revisionが変われば旧Design Contextを拒否できることを確認した。これは適切な限定証拠だが、**別WorkspaceからImportしたProvider原文**はConsumerのManaged Source Indexの対象ではない。そこで現行mainの既存`externalPair`／`LocateImportedSource`を用いる新たな負例では、ProviderのSource末尾LFを追加するだけで、(1) ConsumerのWorld構造・選択Semantic Digest・Managed Source IndexのArtifact／Reference／ModelRevisionは不変、(2) Providerの独立raw SHA-256と現在のSource occurrence fingerprintは変化、(3) 旧Consumer `Result`を与えても`LocateImportedSource`は現在のProviderを再Loadし、初回Load byte witnessにはならず、(4) `BuildSourceIndexFromLoad`は引き続き`ErrImportsNotBound`を返す、の四点を**Goで確認するテスト**を本PRへ追加した。最初の実装HEAD `f6934fd4be431a65250b200667aca4496672652d`は[Repository Health #42461871730](https://github.com/hat47x/tei-platform/actions/runs/42461871730) **completed/success**（全Go test・Go vet・gofmt・参照Workspace validate）。これは**Source-only改訂の負例に限定したGo実証**であり、P0bの完全性や下流Consumer資格ではない。以降の追加Manifest負例を含むHEADの試験結果は**別に確認する**。

**Manifestだけが変化する場合の独立負例（現PR #3479の後続HEAD）**：Provider `tei.workspace.yaml`の末尾改行だけを変更し、Provider自身を別途Loadした`ManifestIdentity().SHA256`と独立raw SHA256は変化する一方、Consumer `World`／自己`ManifestIdentity`／Managed Source Index、選択Semantic Digest、`LocateImportedSource`が返すProvider Source artifact fingerprint／pathは**すべて不変**という対照を追加した。このテストが示すのは、**Source anchorまで一致してもProvider Manifestの改版を検出できない可能性**である。観測用の明示Provider LoadとConsumerのNested Loadは別試行なので、両者が同時点の原子的Snapshotだとは主張しない。原文の取得範囲、SourceとManifestの関係、実Loadの同一Attempt束縛は依然`unknown`。

**ADR-0048／MCP／SHOへの引渡し（方式の決定ではなく資格の境界）**：採用済み[ADR-0048](https://github.com/hat47x/tei-platform/blob/main/plan/adr/0048-shared-application-api-and-transport-adapters.md)に従い、CLI/MCP/Web/Desktop/SHO向けの取得状態はそれぞれが再解釈せず、既存の型付きApplication APIとそのOwnerを利用する。[MCP参照Adapter #3403](https://github.com/hat47x/tei-platform/pull/3403)のWorkspace起動時固定・read-onlyツール・CLI値一致は、外部Importの必要Source/Manifest閉包や現在のDisclosure、実外部Harness受信の証拠ではない。**D=必要Design Inputの充足、T=Tool Reply全量受領、P=必要Provider Source/Manifestの原文／選択・同一Attempt、A=現在のDisclosure/Authority**を別Claimとして扱う。SHO保存/再開、REA逆解析、SEI判断に出典を渡す場合も、読み取った意味・Manifest／Source Revision・取得時点・現在権限を混ぜず、不完全な出典を完全な実世界観測として伝播しない。これら製品のAPI/Registry/Hostを本PRで増やさず、他Ownerが実Consumer E2Eを担当する。

**Source／Manifestの2×2独立性対照（2026-11-08、Go試験追加／新HEAD CI待ち）**：Source原文とProvider manifestを**別々の依存チャネル**として扱うことを、既存Fixture `externalPair`の同一Selected Namespaceで反証する。Independent Expectedは各Fixtureの事前raw byteから計算し、実Provider Loadが返した値をExpectedに流用しない。Consumer World／Consumer自身のManifest／Consumer Source Index／選択Semantic Digestが4条件すべてで同じでも、Provider原文／Manifestの版は独立に変化する。

| 対照 | Provider原文Source | Provider Manifest | 同じConsumer Revisionからの推論 |
| --- | --- | --- | --- |
| B0（原版） | 基準 | 基準 | 現在資格・全必要入力閉包まで推論不可 |
| S1（Source-only） | 変更 | 不変 | Provider Source引用の現在性を推論不可 |
| M1（Manifest-only） | 不変 | 変更 | Provider Root／Import宣言等の入力選択版を推論不可 |
| SM1（両方） | 変更 | 変更 | Source／Manifestを単一Hashで代用不可 |

`TestImportedProviderSourceAndManifestRevisionsFormIndependentMatrix`で独立に切り替え、4条件とも`BuildSourceIndexFromLoad`が`ErrImportsNotBound`を返すことを検査する。ただし**実Provider Load／Consumer Load／Source Locatorは別実呼出し**であり、同じFixtureで値が合致しても**同一Nested Load由来のbyte witness・原子的cohort**は未成立。Provider Root／Manifest選択そのものを変更する場合は別Fixtureで評価する。#2600の増分再利用／#3267のBOMでは、別チャネルの変化によって**必要なClaimだけ**を再資格化し、未選択Plugin・無関係Sourceの変更まで一律全面失効させない。 [main統合済み#3482](https://github.com/hat47x/tei-platform/pull/3482)の`sales.order.submit`向け13実CLI Reply／42 bounded Controlsは**その対象のRequired判別**の限定証拠であり、この外部Provider 2×2の同一Attempt Source読取証拠を含むものではない。

**P0b-Aの現行mainへの実装移管（2026-11-08／#3524、限定Go受入）**：旧#2812で試した手法を、そのまま古い`loader.go`を上書きせず現行mainのResult・SourceRootFindings・ManifestIdentityへ最小接合する。Provider自身の`loadSemantics`で**Parserが実際に読んだfile→Namespace**をLoad-timeに記録し、Manifestのread pinned byteをprivateで保持する。ConsumerのImport解決時は、**その一回のNested Provider Loadの結果**から**明示選択Namespaceだけ**のParser読込Source bytesとManifest rawをprivate clone-on-readで束縛する。別Providerの再Loadで同じFingerprintを取得しても初回実消費証拠とはしない。

**実Go受入条件**：一つのProviderに選択Namespace 2 Source＋選択外1 Sourceを含む合成対照を置き、(a)旧LoadからのReadがSource/Manifestを変更後も旧byteで再現、(b)新Loadは改訂byteを返すがSemantic Digestは同じでもよい、(c)選択外Namespace／不正path／別Alias／偽造Import Digestを拒否、(d)新private raw／Provider Path／fingerprintをJSON及び通常のGo構造体整形へ無条件展開しない限定負例、(e)`BuildSourceIndexFromLoad`はImportに対して従来の`ErrImportsNotBound`で停止、(f)既設`Result.Imports`の公開契約とConsumer-owned Indexを不変に保つ。この検証は`internal/workspace/import_loaded_snapshot_test.go`で扱い、Go 1.26 gofmt・試験・vetの**exact HEAD**結果で判定する。Path／fingerprintは潜在的開示対象であり、外部APIを増やさず#1178/#2516へ別途投影検証を引き渡す。

**Source完全性と機能の優先順位を分ける**：選択NamespaceのParser-consumed files＋Manifestの原文だけでは、ProviderのManaged-but-unparsed、未知/無効root、他Plugin・Query、負の依存、同時改版／A→B→Aまで閉じないため`FullInputClosure=false`は維持する。P0b-Aのために[#2586](https://github.com/hat47x/tei-platform/issues/2586)の実Function→Query→Provider Rowset／Function Data Contextや、[Inspector S2](https://github.com/hat47x/tei-platform/pull/3499)の通常CLI／Studio実Provider共通呼出しを一律停止しない。独立したClaimの証拠が揃った順に実証し、最終的なP0b-closure／BOM／Build skip／SHO保存／実Host Effectへは同一Attempt・Audience・Scopeが確認できた列だけ渡す。private raw保持とcloneのメモリ／時間費用は[#2600](https://github.com/hat47x/tei-platform/issues/2600)の性能計測対象であり、効果がない大規模raw複製は縮小・撤回する。

**必要な更新**：#3262のDesign Context／#1180のScenarioではConsumer World FP＋Consumer Source Index FPだけを`complete input identity`と宣言しない。Providerが必要入力なら、**どのProviderのどのSource occurrence／Manifestを初回に消費したか**の証拠が欠ける範囲のSource-current／引用／Source-dependent test-generation／Build skipを`unknown`にする。他方、意味Digest不変を理由に安全な純粋意味計算の局所再利用まで全面拒否しない。#2812のprivate同一Load snapshotはhistorical候補だが、main未統合・FullInputClosure未証明。#2600／#3267へは`meaning reuse`、`provider occurrence revision`、`actual consumed read`、`qualification scope`を分離して渡す。#3354／#3477で成立した局所Design Context試験や#3371 Inspector Factory候補を置き換えない。ProviderのA→B→A中間履歴・Multi-file atomicity・現在Grant・実Consumer受領・実DB Effect／Owner受理は別Claimである。
**未資格**：選択NamespaceのParser-consumed Source＋Manifestの局所保持は成立候補だが、Provider全Source／未解析Managed Source／否定依存／複数ファイル原子的Read Cohort／Source A→B→A履歴／Owner Meaning／Builder実消費byte／Build省略・Effectは依然未実証。`FullInputClosure=false` と `ErrImportsNotBound` を維持し、次の実Consumer・認可の受入以前にCanonical/Plugin ABIや別Registry/Host/Cacheへ昇格しない。原文保持のメモリ負担が実用途で過大なら、#2600の費用評価に従ってprivate witnessの必要範囲を削る。

**選択のゲート**は四点とする。(1) 読取の入口が`sourceReadRoot`か別ファイル読取かをSource/Manifest/Provider実読取ごとに確認、(2) Builder/Parser/Consumerに渡したbyteとSourceIndex等の検証対象が同じAttemptで再現できるか、(3) 実Providerと無関係なソース、空結果・否定依存・推移Importまで「未読=不存在」と誤推定せず必要Closureを限定できるか、(4) Source引用・Meaning再利用・Build artifact再利用・現在Disclosure/Grant・Effectを異なるClaimとして扱えるか。1〜3のうち欠落した入力は`unknown/unqualified`で停止する。実行中のin-place改変、同時点Read Cohort、外部Authority、Source Revision A→B→A、Target Effectは**別の保証**であり、Hash一致だけでは合格にしない。

**性能への引渡し**：#2600には「同一意味なので計算再利用候補」と「出典・非出力依存・Snapshot/Read Cohort再資格化」は別列で渡す。#1180の共通Scenarioは実Go CLI／Web両Targetへ展開される後段だけに用い、本P0bのために第二Runnerを追加しない。#2516／#1124のStudio・AI／SHOには旧引用のread-only履歴と、現在版・開示・編集・Action資格を分ける。実Provider権限・Commitは#1174へ委譲する。Providerの全Sourceを無条件にLockする巨大な汎用解、TEI独自Host/Registry/Cacheを先行採用しない。

P0bの出口は、実行単位のSource／Manifest／Provider identity・版、実消費byte／集合・不在依存、対象Scope、取得失敗の観測、Snapshot／Read-cohort保証の有無、必要Consumerへ渡すArtifactSetを、**同じ実Attemptまたは固定されたimmutable入力**で再現できることとする。Ownerによる意味受理、現在Authority／Disclosure、実Effect／Receiptは別欄に保つ。証拠が欠ける場合はそのClaimを `unqualified` に止め、過去のread-only表示や純粋意味内容の比較まで全面拒否しない。#1017／#2505がReader／Provenance、#2600が再利用資格、#2667 §30〜32がConsumer縦断、#267が意味受理、#1174が実Effectを引き続き所有する。新Fixture ID・新Registry／Host／Resolver／Cache、正式Schemaを追加せず、別OwnerのDraft本文・Issue・ADRは変更しない。

**2026-11-09横断証拠の照合**：[main #3544](https://github.com/hat47x/tei-platform/pull/3544)／[#3550](https://github.com/hat47x/tei-platform/pull/3550)のSelf-build宣言byteと子Process申告の一致は、Loaderが初回に消費したImported Provider原文の同一Load保持とは**異なる証拠境界**である。[main #3579](https://github.com/hat47x/tei-platform/pull/3579)がFunction定義書へ出力するHook宣言のPhase順も、Source/Manifestの同時点Read・実Hook起動・実Artifact連結の証拠にはならない。#3524のprivate witnessをConsumer向けに昇格する際には、初回Parser consumed Source／同Load Manifest raw／Declared Hook・Generator Input／実Artifact／Selected・Invoked Provider／Audienceを別段階で結合し、**このPRの合成Go TestをFull Input Closure、L2 Self-build、通常CLI/Studio実Provider、Qualified BOMのPASSに流用しない**。前回[#42477304432](https://github.com/hat47x/tei-platform/actions/runs/42477304432)のRepository Health failureには当時mainの設計索引不足・AI Tuning workflow契約違反が含まれ、後続[main #3527](https://github.com/hat47x/tei-platform/pull/3527)でこれらは修復された。**本PRの変更後exact-head CIを改めて検収する**。

**受入の停止条件と終了条件**：P0aのDoneは、単一の実Source・Provider Scope／Manifest／Digest・利用側不変性・意図的なstrict拒否を、独立Expectedとexact HEADへ束縛した記録が得られること。P0b／Build skip／Publisher／SHO保存文書／Host EffectはこのDoneに含めない。旧計画PR #2743はmain統合済みであり、当時の「新しいFixture起票を停止し既存Go／横断Consumer実証へ進む」という判断を履歴として維持する。現行mainの受入は本節のProvider occurrence・Inspector／Design Context／BOM資格で扱い、古いDraftのCIを再利用資格へ昇格させない。新ADR決定、製品Go変更、本番Provider、性能測定、Ownerの業務意味受理は依然NOT RUN／NOT ACCEPTED。
**受入の停止条件と終了条件**：P0aのDoneは、単一の実Source・Provider Scope／Manifest／Digest・利用側不変性・意図的なstrict拒否を、独立Expectedとexact HEADへ束縛した記録が得られること。P0b／Build skip／Publisher／SHO保存文書／Host EffectはこのDoneに含めない。旧計画PR #2743は**main統合済み**であり、「新しいFixture起票を停止して実Go・横断Ownerの検証へ移す」という当時の判断を履歴として維持する。現段階では#2801/#2812の旧Draft結果を現mainの製品資格に転記せず、#3386の範囲でInspector実呼出し・Design Context・BOMの受入経路だけを整理する。新ADR決定、製品Go変更、本番Provider、性能測定、Ownerの業務意味受理は依然NOT RUN／NOT ACCEPTED。

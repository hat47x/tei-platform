# ISSUE-000052: SUI SensemakingをTEI Reference Adoptionとして移管し、Plugin surfaceと開発負荷低減を検証する

- 状態: Proposed — Gate / supported scopeに応じてpre-designとReferenceを段階実施
- 提案日: 2026-09-18
- 性格: Product validation / Reference adoption / Plugin discovery
- 主担当要求: RQ-004 / RQ-008 / RQ-010 / RQ-011 / RQ-012 / RQ-013 / RQ-017 / RQ-021 / RQ-023
- 関連: SUI `ARCH-TEI-MIGRATION-01` / ISSUE-000046 / ISSUE-000027 / ISSUE-000040

## このIssueを最初に読むときの要点

このIssueは、SUI SensemakingをTEIへ移すこと自体を成功とするものではありません。

確かめたいのは、

> **実際に複雑なUI・domain logic・安全境界・AI連携を持つ既存OSSをTEIへ移管すると、Application開発と保守の負荷が明確に下がるか**

です。

同時に、SUI固有に見える要求のうち、どれがTEIの再利用可能なPlugin / Adapter / Capabilityへ一般化できるかを探索します。

SUIはTEIのdemo applicationではありません。SUI自身のProduct Value、Domain、SafeMode、review / approval boundaryを保ったまま、TEIのProduct Valueが現実のApplicationで成立するかをPressureするReference Adoptionです。

## 減らすProduct上の不確実性

1. TEIのCanonical / Projection / Target / Capability / Acceptance境界が、既存の非自明なApplicationを実装する負荷を実際に減らすか。
2. TEIは一般的な業務システムだけでなく、spatial canvasやAI-assisted sensemakingを含むApplicationにも耐えられるか。
3. TEIに不足する機能を、Core growthではなくPlugin / Adapter / Capabilityとして吸収できるか。
4. 現行Applicationから段階移管するとき、既存native assetを保持しつつTEIへ責務を移せるか。
5. 「TEIを使った結果、別の二重管理が増えた」という失敗を検出できるか。
6. SUIで得たPluginが第二Applicationでも再利用できるほど一般的か。

## 基本方針

### TEI-firstと全面TEI-native化を区別する

TEI-firstは、すべてをTEI DSLやTEI Coreへ書き直すことではありません。

```text
Application meaning / domain rule / validation / projection / target
  -> TEIを第一候補

specialized interaction / algorithm / external runtime
  -> Plugin / Adapter / Capability / native asset

low-level kernel / performance-critical engine
  -> native implementationを許容
```

SUI移管では、既存TypeScript / worker / import-export / specialized canvas logic等を、必要に応じてnative assetとして段階的に接続できます。

### GapはCoreへ直行させない

SUI側で不足が見つかった場合は、次の順で分類します。

1. **Existing TEI** — 既存のCanonical / Pattern / Projection / Target / Capabilityで表現できる
2. **Plugin / Adapter / Capability** — 複数Applicationへ再利用できるextensionとして切り出せる
3. **Native asset / custom implementation** — SUI固有または専門性が高く、TEIからBindingすべき
4. **Core / Product Requirement candidate** — 複数Applicationに共通する意味上の不足がReference Evidenceで確認できる

4へ進む前に、1〜3で解けない理由を明示します。

## Stage 0準備状況

2026-09-18時点で、実装前の準備として次を固定した。

- TEI側設計基線: `plan/SUI_REFERENCE_ADOPTION_DESIGN_2026-09.md`
- SUI側既存handoff: `IslandHierarchyContractV1`
- TEI側固定fixture: `reference/sui-adoption/`
  - M1 root — pass
  - M2 3-level hierarchy — pass
  - M3 missing parent — fail
  - M4 cycle — fail

Stage 0では新しいSUI↔TEI共通schemaを発明せず、SUIが既に持つdeterministic projectionをread-only inputとして再利用する。

次の実装probeへ進む場合も、最初はfixture inputを変更せず、TEI Product codeやCapabilityKindを変更しない隔離されたReference tool / testとして開始する。

current TEI標準Validationは`positive` / `relation-unique`の最小集合であり、SUI hierarchyのcycleやparent/child整合をそのまま表現できるとはまだ言えない。

加えて、current Semantic Relationは`one | many` cardinalityを持つが、SUIのnullable `parentIslandId`が必要とする0..1 presence semanticsを独立表現するcontractは確認できていない。Stage 0 reportではこの点を`unresolved`かつ`lossless = false`として残す。

Stage 0 report schema v2では、この不足を単独のbooleanだけで扱わず`constraintCoverage`として機械可読化する。parentのsingle-valued shapeだけを`shape-only / tei-definition`、nullable presenceを`unresolved / source-contract`、duplicate / existence / parent-child consistency / cycleを`shadow-only / reference-local-shadow`として分離する。これにより、SUI正本のconstraintをReference toolが再現しただけのものをTEI native機能へ誤って格上げしない。

これらを理由にValidationKindやRelation optionality fieldを先に追加せず、

- Reference-local validationで十分か
- reusable extensionが必要か
- Application-native ruleとして残すべきか
- 複数Referenceで共通failureが出てProduct gapになるか

を順に判断する。

## Stage 0 read-only probe

Stage 0の最小実装を`tools/reference-sui-adoption`として隔離する。

実装範囲:

- `reference/sui-adoption/workspace`に`sui.island`とself-parent relationだけを定義
- SUI既存`IslandHierarchyContractV1` fixtureをread-only入力として利用
- M1 / M2をvalid、M3 / M4をinvalidとしてconsumer-side shadow validation
- TEI definitionとSUI instance dataをレポート上で分離
- write / canonicalizationは行わない

このprobeはProduct Gate Evidenceではなく、Reference Adoptionのmapping hypothesisを実行可能にするresearch Evidenceである。

次の判断は、probe結果を見てから行う。

1. hierarchy定義だけで十分にTEIへ意味を寄せられるか
2. cycle / parent-child consistencyをApplication-native validationとして残すべきか
3. reusable validation extensionが必要か
4. placardを次のhandoffへ追加する価値があるか

この時点ではValidationKindやPlugin CapabilityKindを増やさない。

## Stage 0 → Semantic Extension evaluator pressure — 2026-10-05

Stage 0でReference-local shadowとして保持していたhierarchy validatorを、Semantic Extensionの実consumer pressureへ再利用するcandidateを追加した。SUIの`IslandHierarchyContractV1`やfixtureは変更せず、既存M1〜M4 semanticsを実装していた評価ロジックを共有Reference packageへ切り出し、`tools/reference-sui-adoption`と`tools/reference-semantic-extension-catalog`の両方から同じ実装を使う。

この変更でSUI側のAuthorityはTEIへ移らない。SUI native contractが引き続き正本であり、共有Evaluatorはそのpinned contractをshadowするReference implementationである。TEI Canonical dataへinstanceを昇格せず、writeも行わない。

Semantic Extension側では、`sui.island` subjectへhierarchy integrity Constraintを追加し、そのEvaluator requirementをversioned Semantic Interfaceとして明示する。provider自己申告だけではConstraintを有効化せず、exact contractへboundしたConformance Evidenceが必要である。さらにValidation Planでimplementation identity / artifact digestを固定し、Environment Runtimeのprovider選択でAccepted validityを変えない。

M1 root / M2 three-levelはpass、M3 missing parent / M4 cycleはfailとして、同じshared evaluatorをValidation Plan経由で評価するcandidate testを追加した。これは「SUI固有ValidationKindをCoreへ追加しなくても、実Application由来の意味不足をSemantic Extensionで扱えるか」をpressureする。

ただしexact-headのfocused test / repository healthはこの環境では未取得であり、Stage 0の`constraintCoverage`をnative coverageへ昇格しない。Semantic ExtensionのProduct Evidence、SUI移管判断、標準語彙化のいずれもこのcandidateだけでは更新しない。
## 最初のReference slice候補

最初からSUI全体を移しません。少なくとも次の性質を一つの小さいsliceでPressureできるものを選びます。

- Card / Island / Relation等の意味データ
- 保存・再読込
- Canvasまたは空間Projection
- 一つのValidation / SafeMode rule
- 一つのreview / proposal acceptance
- 一つのAI / Cognitive Capability
- export / provenanceの最小経路

初回sliceの具体的な対象は、TEIのsupported scopeとSUI側の比較基線を見て決めます。

## 固定済みのReference設計

TEI側の移管設計基線は `plan/SUI_REFERENCE_ADOPTION_DESIGN_2026-09.md` を参照する。

SUI側では実履歴から次の3 challengeを固定済みである。

| Challenge | Historical SUI PR | Changed files | Changed lines |
|---|---:|---:|---:|
| hierarchy / placard | #235 | 11 | 338 |
| polygon vertex editing | #286 | 5 | 219 |
| explicit merge apply transaction | #2849 | 3 | 465 |

補助Evidenceとして、hierarchy concernがexport / import / view metadataへ横断したSUI PR #241（10 files / 260 changed lines）も用いる。

初期移管戦略は **encapsulation before translation** とする。現行SUI `DocumentV1`を当面のauthoritative native assetとして保持し、TEI側はread-only mapping / projectionから開始する。二つのauthoritative sourceを作らない。

また、current TEI Pluginの`CapabilityKind`にはspatial canvas / presentation専用種別は存在しない。この一件だけを根拠にkindを追加せず、Target Adapter、native component、既存Capability、将来のreusable extensionのどれが自然かをReferenceから判断する。

## TEI適性の第二軸 — structure-to-application leverage

SUI Reference Adoptionでは、Application Planeの共通化だけでなく、SUIの図表的・構造的な設計資産からApplicationをどこまで導けるかを検証する。

比較では、次を観測する。

- 一つのstructured definitionから何個のValidation / Projection / Target / documentation / AI contextを導けるか
- 同じ意味の重複記述が何箇所減るか
- view変更でdomain definitionを書き換えずに済むか
- domain意味変更が複数UI / export / validationへ自動的または機械的に伝播できるか
- native codeへ残る部分が、特殊Interaction / algorithm等へ局所化するか

これは **structure-to-application leverage** のEvidenceとして記録する。

TEI利用率が高くても、構造化した意味から派生物を導けず二重管理が増えるならReference Adoptionは成功としない。

## 開発負荷の比較

SUI側の同一change challengeと比較し、TEIで次がどう変わるかを記録します。

- hand-written code / config / schema差分
- 触るファイル・層・正本の数
- test fixture / regression testの更新量
- 同じ意味を重複記述する箇所
- generated / derived artifactの再生成可能性
- Validation / Conformanceが自動で捕捉できる不整合
- manual synchronization step
- Plugin / native assetとして残るapplication-specific code
- migration friction / rollbackの容易さ

中心指標は**change amplification**です。

```text
one semantic change
  -> how many layers / files / tests / docs must move together?
```

単純な生成コード量削減やTEI記述率を成功指標にはしません。

## Plugin discovery

初期のPlugin候補として、次を仮説として保持します。

- spatial / graph canvas projection
- drag / resize / grouping / polygonal enclosure等の複雑Interaction
- offline bundle / artifact import-export
- review / diff visualization
- local cognitive provider integration

ただし、SUIで必要だったという理由だけでTEI標準Pluginにはしません。

Plugin候補は最低でも、

1. SUIで実用価値がある
2. TEI Coreの意味を増やさず接続できる
3. 第二Applicationで再利用可能性をPressureできる

ことを確認してから昇格します。

## Gateとの関係

TEIの現在のProduct Gateやsupported scopeを、SUI移管のためだけに飛び越えません。

- pre-design
- mapping
- change challengeの固定
- Plugin boundaryの仮説
- Reference fixture準備

は先行できます。

一方、未成立のTarget / Acceptance / Runtime機能をproduction-readyとして扱う実装は、対応GateのEvidenceを待ちます。

## 非目標

- SUI Product ValueをTEI Product Valueへ吸収すること
- SUIのDomain objectをTEI Core kindとしてそのまま追加すること
- SUIを一括rewriteすること
- SUIが使うからという理由でTEI Coreを拡張すること
- TEIへの移管率100%を目標にすること
- 専門runtimeや高性能canvas engineまでTEI-nativeで再実装すること
- 一つのReference成功だけで「すべてのOSSをTEI化できる」と主張すること

## 完了Evidence

このIssueは、少なくとも次が揃ったとき判断可能になります。

1. SUI側の現行baselineと3種類以上のchange challenge
2. TEI上の最小Reference slice
3. 同一change challengeの比較結果
4. gapのExisting / Plugin / Native / Core candidate分類
5. 少なくとも1つのPlugin候補についての実利用Evidence
6. SUI移管を broad / incremental / limited / deferred のどれにするかの判断
7. TEI側で、今回のReferenceからProduct Requirementを増やす必要があるか否かの明示判断

## 完了後に可能になる判断

- SUIをTEIへ本格移管する価値があるか
- TEIを他の自作OSSのApplication Implementation Layerとして標準利用する妥当性があるか
- どのPluginを先に育てるべきか
- TEI Coreへ不足があるのか、extension surfaceだけで十分なのか
- SEI / EKI等の管理面・UI面へ横展開できるか

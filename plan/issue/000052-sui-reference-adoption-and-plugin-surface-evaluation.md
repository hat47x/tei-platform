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

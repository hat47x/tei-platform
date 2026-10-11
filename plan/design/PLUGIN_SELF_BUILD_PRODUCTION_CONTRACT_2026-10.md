# Plugin self-build: production最小契約と移行Gate（検討案）

- 対象：ISSUE-000079 / GitHub #943
- 作成日：2026-10-11
- 状態：**設計候補。production仕様の採用・実装・qualified PASSではない**
- 正本：`plan/issue/000079-plugin-self-build.md`
- 比較対象：main `internal/plugin/build.go`、Draft #1106 / #1151 / #1148 / #1093 / #1099
- 関連：#957 / #979 / #1066 / #951 / #1096 / #1103 / #1104 / #1173 / #971 / #1122 / #1167

## 1. 結論：最小変更で何を成立させるか

現在のBuild Plan / BuildArtifactLockを廃止しない。既存のPlugin Buildを壊さず、非Plugin Runtime Engine・複数logical subject・複数physical outputを扱うために必要な**意味上の分離**を検証する。共通型を最初から一括導入しない。

~~~text
unresolved Build Intent
  -> explicit Capability-scoped Builder resolution (#951 / #1101)
  -> exact BuilderIdentity / locked replay
  -> validated exact inputs
  -> Build execution plan + build-only graph
  -> Build attempt observation (success/failure)
  -> verified Artifact Set
  -> typed subject binding
  -> immutable provenance / attestation
  -> current Trust / acceptance
  -> downstream Runtime / deployment authority

Artifact Store:
  digest -> bytes retrieval -> digest recheck
  (Trust / Acceptance / Runtime Authorityとは独立)
~~~

原則：

- **Build input identity / execution result content identity / actual attempt identityを分ける。**
- **Builder resolution / execution / attestation / acceptance / invocationは別権限とする。**
- **Plugin / Runtime Engine / Builderは同一のbuild executionを利用できても、subject-specificな照合とAuthorityを捨てない。**
- **Builder AdapterはGo固有情報を保持できるが、上位契約をGo / shell / npm等の固有項目で固定しない。**
- **同じlogical subjectをTargetごとに違うphysical artifactへrealizeできる。**
- **ローカルCLI・Web・Desktop・外部LLMから呼ぶ経路でBuildの意味・検証・失敗を省略しない。Transport側のAuthorizationは既存機構を利用する。**

## 2. 現行実装に照らした差分判定

| 領域 | 既存の正本・Reference | 今の判断 | production化前の停止条件 |
| --- | --- | --- | --- |
| Plugin Build v1 | `plugin.BuildPlan`, `LockedBuild`, `BuildArtifactLock` | **維持**。既存Lockを再解釈しない | exact旧fingerprintとruntime build traceを回帰固定 |
| Artifact-independent Definition | `BuildDefinitionFingerprint` | **維持**。循環回避に必要 | Sourceを変更せずにderived Runtime Definitionを作れる |
| Domain-neutral Build Execution | #1106 | **抽出候補**。Plugin IDとは別identity | definitionのartifact-affecting closureが完全である証拠 |
| Typed subject binding | #1106 | **抽出候補**。Plugin/Engine/Builderを別typed bindingに | bindingがexact output / resultと結合し、Authorityは別 |
| Multiple outputs | #1106 / #1102 | **抽出候補**。member role, format, digest | 一意roleとmissing/extra/changedを負例化 |
| Builder resolution | #1151 / #1101 / #951 | **契約差分のみ抽出候補**。固定enum / 第二Resolverは不可 | selected slice Conformance、ambiguity、locked replay |
| Go Source closure | #1093 / #1096、関連 #1122 | **Adapter領域**。Go-specific pressureを共通Build型へ移さない | tracked・embed・Cgo・path semanticsをexact評価 |
| Build-only graph | #1106 / #1103 | **Reference継続**。schedulerを作らない | unique exact input producer、cycle reject、stage0明示 |
| Actual Build Attempt | #1110 / #1083 | **別Evidence**。結果content fingerprintにrun occurrenceを混ぜない | failure / cancellation / partial outputも観測可能 |
| Attestation / current Trust | #1106 / #1104 | **既存Authorityへの接続候補**。Reference fake signatureはproduction不可 | issuer / signature / policy revision / revocationの実検証 |
| Local CAS / retrieval | #1148 / #1105 | **独立Adapter候補**。Store locationをLock identityに入れない | symlink / corruption / race / size超過を拒否 |
| no-clobber export | #1099 / #1098 | **publication safety**。Store APIと同一視しない | publish時点の競合を拒否、上書きfallbackなし |
| SB-3 | #979 / #1066 | **別完成Gate**。E5–E8の一般化に巻き込まない | Go二重正本の解消、stage1/stage2 actual比較 |
| Qualification | #1173 / #971 | **最優先横断Gate** | trusted full checkout、exact head、full health、clean pre/post |

### 2.1 既存BuildPlan v1と新identityの非互換性

**観測**：main `internal/plugin/build.go` は `BuildPlanFingerprint` で正規化BuildPlan全体をJSON化する。`BuildSource.Locator` を含むため、同じSource digestでもLocator変更でfingerprintが変わる。実際に `internal/plugin/build_test.go` に「source locator変更はfingerprintを変える」というtestがある。

一方、Draft #1106 の `buildExecutionPlanFingerprint` はSourceの`Kind / ID / Digest`のみをcontent identityへ含め、`Locator`を除く。またdigest hexを正規化する。

これらを同じ名前や同じformatとして扱ってはならない。

**移行規則候補**：

1. v1 BuildPlan・v1 BuildArtifactLock・Runtime Plugin Lock format 3に格納されたBuildPlan fingerprintは**変更しない**。old fingerprintが新execution fingerprintと一致するとは仮定しない。
2. domain-neutral側の契約を採用するなら**別名・別format version・別fingerprint namespace**を持たせる。既存値を黙って上書き・変換しない。
3. Legacy BuildPlan -> BuildExecutionPlanはread-onlyな**projection**とし、両fingerprintと両identity規則を保持する。元のLockを保持したまま、cross-bindingを独立したEvidenceで照合する。
4. `Locator`はtransport locationの場合のみcontent identityから外す。Go module/package path、relative embed、generator entrypointのような**path-sensitive semantics**はclosure identity / Recipe / Parameterへ明示する。単にLocatorを無視してはならない。
5. projectionに必要なartifact-affecting Definition Sourceが欠けるなら**unresolved**としてfail-closed。Plugin definition IDを新execution identityから省くこと自体は、definition bytesがbuild inputでない証明にならない。
6. old Lockのない新形式へ一足飛びに自動昇格させない。新形式の受理可能性と旧形式の読み取り互換を独立に試験する。

決定前に「同じbytesを別pathへ配置」「同じpathで入力bytes変更」「Go package path変更」「relative embed変更」「locatorだけ違い実入力が同一」の五ケースを境界テストにする。

### 2.2 Execution ResultとAttemptの分離

Draft #1106 の `BuildExecutionResult` は `ExecutionPlanFingerprint + ArtifactSetFingerprint` を結合する。これは**結果のcontent-level identity**として適切だが、同じPlanから同じbytesを二度buildすると同じfingerprintとなる。したがって、二回のactual実行が存在したことの証明にはならない。

提案：

- `BuildExecutionResult`：計画と成果物集合の不変な対応
- `BuildAttemptEvidence`：各回の実行ID、exact plan、実行Host/環境の要約、開始・終了観測、exit status、取得したartifact、unresolved原因、evidence locator/digest
- `BuildReproducibilityEvidence`：**独立した二回のAttempt**と各Resultの比較。member差分を保持
- `Attestation`：どのResult/Attempt claimsを誰が署名したか
- `TrustDecision`：現在のpolicy revisionでそのattestationを採用するか

「同じfingerprintを二回読み取った」だけではtwo-build repeatabilityと主張しない。失敗Attemptも削除しないが、失敗からArtifact acceptanceへ進めない。詳細フィールド、機微情報Disclosureは#1110/#1104や既存Knowledge Context/Authorization契約を再利用し、この文書で第二Policy Engineを作らない。

### 2.3 解決済みBuilderのReplayと失効評価

#1151ではfresh resolutionとlocked replayを分ける。次は区別が必要。

- **Replay integrity**：以前のexact Contract / Provider / implementation / artifact / selected-slice Conformanceを再現する
- **Current authorization / trust**：現在のpolicy・publisher権限・revocation下で使ってよいかを別に判定する

Locked replayが成功しても、失効したProviderをBuild実行してよいとは限らない。逆に、新しい互換Providerがcatalogへ追加されただけで既存Lockを別Providerへ切り替えない。#951のContract definition AuthorityとConformance issuer Authorityも再利用し、Builder専用に別の信頼階層を作らない。

## 3. Production最小deltaの採用順序

### Delta 0: baseline / qualification

- #1173でtrusted Git/CLI execution pathを実際に成立させる。
- #971でcurrent mainをexact-head full Repository Healthへ通す。
- 自身のDraftのexact-head receiptが揃うまで、本書が参照するReferenceを「採用済み」と表現しない。

### Delta 1: identity / read-only projection

- BuildPlan v1のfingerprint・validation・lock serializationをgolden fixtureで凍結する。
- #1106のdomain-neutralなExecutionPlan / Result / SubjectBindingへ**read-only** projectionする。
- Artifact-affecting Definition closureが足りなければprojection不成立として記録。
- Plugin / Runtime Engine / Builderの三subjectを混同しない。
- production型を作る前に同じbuildから複数subject・複数memberのEvidenceを比較する。

### Delta 2: resolved builder / Go adapter

- #951のcapability primitivesが整った範囲に限り#1151のselected-slice resolverを利用する。
- exact BuilderIdentityを確定した後はBuildPlanを再解決しない。
- #1096がGo固有closure・toolchain・environment・artifactを測定する。
- Stage0 seedはexplicitに指定し、隠れたdefaultにしない。

### Delta 3: build result / observation / retrieval

- ResultとAttemptを分離する。
- Artifact Setのroleごとのexact digestを検証する。
- #1148はStore adapterとして取り込み、LockにStore rootを追加しない。
- #1099はpublication no-clobberで別途qualifyする。

### Delta 4: authority bridge

- Result / Attempt -> Attestation -> verified Evidence -> fresh Trust decisionを別段階にする。
- build traceやCAS receiptをAcceptance/Runtime Authorityへ昇格させない。
- #957のBuiltinRegistrationや既存Runtime Plugin Lockとの橋渡しは、approved subject bindingだけを用いる。

### Delta 5: actual SB-3・対象拡大

- #1066のTEI-owned Source一元化、#979のactual convergenceを先行。
- その後にPlugin package、代表的Tool/Runtime、Runtime Engine、TEI本体、第二Targetへ拡張する。
- セルフビルドが先に全面完成しないと業務vertical sliceを進められない設計にはしない。

## 4. 最小の負例／回帰試験表

以下は**試験仕様であり、actual test PASSの主張ではない**。各行にGiven / Mutation / Expected / Evidenceを定義して実装Issueへ渡す。

| ID | Given / 操作 | Expected / 主なowner |
| --- | --- | --- |
| SBP-01 | v1 BuildPlanのLocatorだけ変更 | old fingerprintが変わる。v1意味を壊さない（#943） |
| SBP-02 | path非依存で同bytes、Locatorだけ変更 | new execution identityは不変（#1106） |
| SBP-03 | Go module path / relative embedだけ変更、content bytesは同じ | Source closure / Recipe / Parameterに差が現れないならprojection拒否（#1096/#1106） |
| SBP-04 | Plugin descriptorがartifact bytesへ影響するのにSource closureから省く | unresolved / projection拒否（#1106） |
| SBP-05 | same bytes / different exact Build Plan | Artifact Set identityは同じでもResult identityは異なる（#1106） |
| SBP-06 | same Plan+Artifact Setを「二回実行」と主張、Attempt Evidenceは一つ | two-build repeatability不成立（#1110/#1083） |
| SBP-07 | one Build -> two logical Plugin、片方のsubject fingerprint差替え | 一致しないbindingを拒否（#1097/#1106） |
| SBP-08 | one Build -> two artifact roles、片方欠落/format drift | Artifact Set再現性は失敗、missing/changedを保持（#1102） |
| SBP-09 | graph consumer inputにproducerがゼロ／二つ | unique exact input provenance違反として拒否（#1103） |
| SBP-10 | stage0 seedがstage1 exact Builderと一致しない | bootstrap開始を拒否（#979/#1106） |
| SBP-11 | compatible Providerが二つ、wiringなし | fresh resolution曖昧で拒否。既存Lockはexact replay（#1151） |
| SBP-12 | selected Recipe×Target×ProfileのConformance欠落 | support宣言だけで選択不可（#1151/#951） |
| SBP-13 | Providerの選択外Targetだけ追加 | old Resolution Lockを不必要にinvalidateしない（#1151） |
| SBP-14 | Provider Conformanceは固定、current Trustがrevoke | replay identityは不変、current useは拒否（#1104/#951） |
| SBP-15 | Store内のdigest名は正しいが中身が破損 | Getを拒否しLock digestを変更しない（#1148） |
| SBP-16 | putと同時に別writerが同じpathへpublish | no-clobber・bytes照合・fail-closed（#1148/#1099） |
| SBP-17 | Build失敗後、partial outputが残る | Attempt失敗Evidenceを保持、Artifact acceptance拒否（#1110） |
| SBP-18 | stage1/stage2 digest一致、seed trustは未検証 | convergenceとしては記録可、trusted/acceptedを非主張（#979/#1104） |
| SBP-19 | Go constructorを変更せずTEI-owned Sourceを変更 | Source側が正本で再実現できる（#1066） |
| SBP-20 | Web/CLI/Desktop別経路で同じBuild IntentとLockを使用 | 同じmeaning・validation・権限拒否・diagnosticを保存（#1173/#943） |
| SBP-21 | Runtime-only dependencyをbuild graphへ混入 | graph生成を拒否または明示的なbuild-only provenanceなしを検出（#1103） |
| SBP-22 | Cgo/hidden ignored/untracked/symlink等が実compile inputとなる | 対象closureを完全固定できなければ拒否（#1096/#1122） |

## 5. 完成判定を層別に扱う

| 層 | 最小完成証拠 | 他層と混同しないこと |
| --- | --- | --- |
| SB-1 package | exact Definition/Planから独立2回、package byte digest一致 | Capability implementationのself-hostingではない |
| SB-2 implementation | Source→realization→Lock→Provider invocationのactual経路 | Builder自身をTEIで再生成した証拠ではない |
| SB-3 builder | stage0/1/2 actual execution、B1/B2 convergence、seed/engine記録 | seed trust / engine self-hostingではない |
| Source authority | TEI-owned Sourceだけを変更してbuilder semantics更新可能 | Go oracle一致は第二正本を意味しない |
| Qualification | trusted full checkout / exact HEAD / full tests / clean pre-post | merged/PR Draftやfocused testで代用できない |
| Portability | Target差を跨ぐcontract semanticsと明示されたphysical realization | 一Target成功を全Target成功とみなさない |

## 6. Scope上の決定を保留する点

- production型の正確な名前・JSON schema・format version
- general BuildExecutionPlanをPlugin BuildPlan v1の代替にする時期
- BuildAttempt EvidenceのHost由来観測・署名・保持期間
- Artifact SetのRole語彙と跨Targetの最小標準
- engine / toolchainのhost-side loaded code attestation
- リモートArtifact Store / distributed scheduler / cache eviction
- 複数Target間のoutput byte一致（意味同等性と混同しない）

いずれも先に巨大Coreを追加して決めない。positive Reference 1件・negative Reference 1件・consumer 2種類以上で必要性を確かめる。

## 7. この文書の終了条件と次の作業

- [ ] #971 current-main forward receipt
- [ ] #1106 / #1151 / #1148など自身のexact-head資格確認
- [ ] SBP-01〜22のownerと実装先を各Issueに同期
- [ ] v1 fingerprint golden testsとprojection negative tests
- [ ] production採用が必要な最小primitiveをADR案へ移す
- [ ] #1066 Source authority一元化のactual test
- [ ] #979 actual SB-3 convergence Evidence
- [ ] Go以外のbuilder・第二Targetがこの意味契約を満たすか評価

本書は設計の正本を増やすものではなく、`plan/issue/000079-plugin-self-build.md` と既存Issue群から、productionへの移行判定に必要な未決事項・負例・順序を抽出した**判定用ノート**である。最終判断は元計画・ADR・actual Evidenceへ返す。

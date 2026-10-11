# CLI C2：inspect／traceのDisclosure境界（設計候補）

- 状態：2026-10-11、設計・隔離Reference。製品契約としては未採用。
- 担当：[Issue #1178](https://github.com/hat47x/tei-platform/issues/1178)
- C1候補：[Issue #1165](https://github.com/hat47x/tei-platform/issues/1165)／[Draft PR #1172](https://github.com/hat47x/tei-platform/pull/1172)
- 参照：[Security境界](../../docs/security.md)、[legacy Authorization](../../docs/conformance/legacy-authorization-v1.md)、[Knowledge Disclosure](../../internal/knowledgecontext/disclosure.go)、[ADR-0004](../adr/0004-source-to-canonical-traceability.md)、[Capability Invocation Context #1118](https://github.com/hat47x/tei-platform/issues/1118)

## 問題と判断

読取り専用の `inspect`／`trace` は、Source path、fingerprint、関連Node、未解決件数から機密の存在を示し得る。OS上でローカルファイルを読めることと、低権限利用者や外部LLMへその情報を見せてよいことは別である。

対策は、**既存のAuthorization Capabilityで読取と開示・送信を判定し、許可された内容だけを共通Projectionから公開する**こととする。CLI専用の第二Policy Engine、第二Graph、Source Indexへの認可状態の書込みは行わない。

## 既存機構と不足分

| 機構 | 再利用するもの | まだ不足するもの |
| --- | --- | --- |
| Canonical / ModelRef / Source Index | C1の厳密なSubject、Graph、derived sidecar、World/Source fingerprint | 各出所・関係・diagnosticの公開区分 |
| `knowledgecontext.BuildDisclosure` | 選択的Section投影、Projection revisionと選択結果のfingerprint | **認可そのものではない**。同実装の `selective-disclosure-is-not-access-control-or-redaction` を維持し、Subject・Unresolved・Availabilityにも個別の開示判定が必要 |
| `security.authenticate` / `security.authorize` | Subject/Action/Resourceを受ける既存の認可Capabilityとlegacy Adapter | `inspect.read`、`trace.read`、`disclosure.egress`のAction/Resource写像、Source ProviderごとのPolicy |
| #1118 / Draft PR #1123 | Semantic RequestとHost注入Execution Context、reuse partition候補 | production Context bindingと現在のPolicy Providerへの実接続。Referenceを完成済み機構と誤認しない |
| #1045 / Draft PR #1055 | access不可とunknownを同一unavailable、Evidence/relation非公開、partial/indeterminate境界 | Source・Graph・summary件数・継続・Errorへの適用 |
| Shared Acceptance | 意味の受理と権限の分離 | 開示権限の代用にはしない |

C1の `-scope` はGraph候補ラベルであり、認証scopeやtenant isolationではない。Source IndexはCanonicalと独立したderived sidecarとする（ADR-0004）。

## 一つのAuthorizationと公開の境界

```text
human / agent -> CLI, Web HTTP, Desktop IPC
    |
    v
trusted Host: verified Principal / Caller / Workspace / tenant
    |
    +-> existing security.authorize（read権限・Source Provider条件）
    |       deny / unknown -> internal raw取得に進めない
    |
    +-> C1 operations.InspectSubject / TraceWorkspace
    |       raw Canonical / Source Index / typed Graph（Host内部のみ）
    |
    +-> Exposure Manifest：Source、関連Node、evidence、unresolved、件数
    |
    +-> existing security.authorize（公開と宛先ごとのegress認可）
    |       policy revision / currentness / decision receipt
    |
    +-> explicit allowed-facet Projection
            |
            +-> console / HTTP / IPC / external LLM
```

同じCapabilityを読取前と公開前に評価するのであり、別々のPolicy Engineを作るのではない。C1の独立したraw意味操作は保持し、公開向けAdapterがその出力を無条件にはシリアライズしない。

**Hostの責務**：Principalとサービス代理、検証済みWorkspace・tenant・Session、Source Providerの権限、外部宛先（provider/account/deployment/tenant/purpose）を束縛する。CLI flag、HTTP body、LLM tool argument内の自称値で権限を付与しない。

**Exposure Manifestの責務**：開示対象のCanonical・Source・related node・edge evidence・unresolvedなど**全構成員**を認可側に知らせる。あるfacetを許可するなら、その中の全構成員と派生metadataが当該宛先へ公開可能でなければならない。検査不足のfacetは全体非許可とする。

**Projectionの責務**：許可されたfield/facetだけを最初から新しく組み立てる。非許可の構成員を含むGraphを丸ごと渡してから文字列で黒塗りする方式は禁止。External error、stdout/stderr、log、cache、page/summaryの件数も対象。

## 最小Disclosure契約（v0候補）

- **Selector（意味要求）**：operation、subjectKind/ID、graphScope、direction、mode、page。
- **Host Binding（Execution Context）**：principalRef、callerKind/ID、delegationRef、workspace/project/tenant/environment、sessionOrInvocationRef、channel/destination/purpose。
- **Manifest（Host内部）**：対象Canonical、SourceProvider、Artifacts/References/Range、近傍とedge evidence、unresolved、Derived metadata、World/Source/Graph snapshot。
- **Decision（既存Authorizationの結果）**：allow/deny/indeterminate、対象Manifestに対する許可facet集合、policy ID/revision、決定Receipt、期限/currentness、Host発行reuse partition、存在情報の公開可否。存在未確認やPolicy評価不能はfail-closed。
- **Public Projection**：契約versionと許可済み情報、不完全性を公開してよい場合だけその表示、必要時のHost発行opaque continuation。raw policy ID、accessRef、秘匿ID/hash/path、内部エラーは含めない。

**Canonical全体の許可はall-or-nothing**：JSON内のどのフィールドも確認せず「一部だけ隠せる」と主張しない。個別Edge単位の部分開示は後続課題とし、初回Referenceではfacetごとに全許可か非公開を選ぶ。

**不完全性の二つの扱い**：

- transparent-incomplete：不完全であるという情報の公開が認められている。可視facetのみ返し、秘匿対象のID・件数・理由は返さない。
- existence-sensitive：不完全であること自体が漏えいになる。要求全体をunknown Subjectと同じ `unavailable` にする。

## Data freshness、Policy currentness、cacheは別

World fingerprint、Source Index fingerprint、C1 Workspace/Graph snapshotは**データの観測・継続条件**であり、認可の証明ではない。Knowledge Disclosure fingerprintは選択結果のcache identityであり、Permission Grantではない。

Page 2、summary→full、export再送、cache hit、外部LLM送信時は、current policy、revocation、Principal、Caller、tenant、Workspace、宛先、purposeを再確認する。データfingerprintが同じでもPolicy revisionやegress先が変われば旧Receiptを使わない。データ変更と認可変更の理由は内部監査で分け、public Errorで存在確認oracleを作らない。

C1はsummary→fullとpage幅変更を同一selectorの継続として認める。C2はその意味上の継続を壊さず、**増えた情報をcurrentなPolicyで再認可**する。予測可能なraw Source fingerprintを公開Continuation tokenに流用しない。

## 監査（公開結果とは別）

最低限、auditEventId、correlationId、decisionAt、outcome、公開可能な固定reason code、Decision Ref、egress attempt/resultを保存する。

| 分類 | 記録する項目 |
| --- | --- |
| 認証主体 | `principalRef`、human/service、代理時のdelegationRef |
| モデル対象 | `modelSubjectKind/ID`、operation、Graph候補scope |
| Caller | `callerKind/ID`、Host、session/invocation |
| 適用範囲 | workspace/project/tenant/environment、Source Provider、resource/facet集合 |
| 宛先 | channel、provider/account/deployment/tenant、purpose |
| Policy | policyId、**immutable policyRevision**、decisionRef、期限、失効・再評価理由 |
| Data | World/Source/Graph snapshot参照、公開Projection version、completeness、送信結果 |

PrincipalとモデルSubjectを同じ名前でまとめない。生credential、秘匿path、raw accessRef、非許可Source hashを通常auditへ保存しない。監査必須の送信では、記録できない場合は送出を行わない。

## fail-closed負例とReference

1. 未認証・Policy不明・期限切れ・Provider失敗。
2. 不存在Subjectとアクセス拒否Subjectの同じ公開Envelope。
3. Caller、Workspace、tenant、Graph scope、外部送信先、purposeの不一致。
4. Canonicalのみ許可され、Source path、fingerprint、関連、unresolvedは非許可。
5. 存在を明かせないhidden関連がある場合、partialや件数も返さない。
6. 全体Totalを、許可済み可視部分の件数と偽って出さない。
7. 同一SnapshotでもPolicy revision・再利用partition・送信先が変われば続きページを拒否。
8. summary→fullは同一対象・方向かつ再認可を条件に許し、別方向は拒否。
9. Raw errorやSource情報をpublic error、log、tokenへ反射しない。
10. Sectionを選択しただけでアクセス許可になる設計を拒否。

隔離Reference：[tools/reference-cli-disclosure](../../tools/reference-cli-disclosure/README.md)。合成Authorizerを注入するだけで、実Provider／C1／各Transportには未接続。局所Go testをactual repository-wide qualificationへ読み替えない。

## C1へ要求しない機能と統合停止線

Draft PR #1172には、認証UI、tenant管理、LLM Provider registry、Production Policy Engine、暗号化cache、実Web/Desktop、完全なOS Source Readerを追加しない。C1の `inspect/trace` は**trusted local raw inspection候補**であり、安全な外部公開APIではない。

次段階は、既存 `security.authorize`へのManifest/Host Binding写像と二ConsumerのE2E検証、監査Receipt・継続の実現に絞る。TOCTOUは[#1017](https://github.com/hat47x/tei-platform/issues/1017)に残す。main統合にはAGENTS.mdのexact-head trusted clean checkout・全Repository Health PASS・前後cleanの証跡を要する。CI停止中を理由に省略しない。

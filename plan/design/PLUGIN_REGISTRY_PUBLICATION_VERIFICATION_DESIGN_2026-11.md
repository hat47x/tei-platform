# 分散型Plugin公開・登録・検証プロトコル（設計候補、2026-11-02）

- **状態**：設計候補。登録サービス・署名検証CLI・外部Consumer・実Runtimeの実装/実証を意味しない。規範仕様の正式採用はOwner判断待ち。
- **追跡Issue**：[ #2589 ](https://github.com/hat47x/tei-platform/issues/2589)。
- **目標アーキテクチャ**：[ADR-0032](../adr/0032-composition-kernel-replacement.md)、[#951](https://github.com/hat47x/tei-platform/issues/951)（Contract/Provider/Authority）。名前空間候補は[Draft #2298](https://github.com/hat47x/tei-platform/pull/2298)に置かれ、**main未採用**。
- **既存Owner**：[#943](https://github.com/hat47x/tei-platform/issues/943)（Self-build）、[#1104](https://github.com/hat47x/tei-platform/issues/1104)（Attestation/Trust）、[#1105](https://github.com/hat47x/tei-platform/issues/1105)（Artifact取得）、[#1121](https://github.com/hat47x/tei-platform/issues/1121)（段階的Lowering）、[#1370](https://github.com/hat47x/tei-platform/issues/1370)（検証対象と起動物）、[#2390](https://github.com/hat47x/tei-platform/issues/2390)（外部Host）。本設計で第二Resolver/Authority/Lockを作らない。

## 1. 解く問題と避ける失敗

作者は公開サイト上でPlugin名を予約し、GitHub Repository等の管理権限を証明し、初回検証の後に正式登録する。以後のReleaseはGitHub Releases、OCI、企業内Storeなど作者が管理する場所に置く。公開サイトは登録・発見・公開内容の取得・検証の索引を担い、配布物の独占保管や実行許可の発行はしない。

失敗例：GitHubのTagを同じ名前で差し替えた、正しいDigestでも別Repository由来なのにPublisher承認済みとした、申告したSource D1とは違うSource D2が実Build入力だった、Linuxの適合証明をWasmへ流用した、署名済み証明のIssuerが既に失効している、Registryが選んだProviderをそのままRuntimeで許可した。このような誤昇格を防ぐ。

**目標**：第三者がRegistryなしでも指定Releaseのバイト列・署名済み宣言・検証証明を独立検査し、不足する検査を「未検証」として保持できること。対象外の実行権限・業務意味を「合格」に含めない。

## 2. 論理上の識別子と権限

| 概念 | 役割 | 不変性・権限 |
| --- | --- | --- |
| 公開名（Registered Name） | 例 `xtea-modeler`。予約Authority内での検索名 | 名前変更やAlias化を許す。異Registryの同名は同一性を示さない |
| Plugin ID | 正式登録時に確定する長期同一性 | Repository名・URL・現在の所有者から独立。再利用不可 |
| Registration/Ownership record | Origin Authority、現在のPublisher委任、世代・移転証拠 | Genesisは不変、所有権変更は履歴 |
| Contract ID + fingerprint | 提供する型・意味・Scope・Failure/Unknown・Effect等 | [#951](https://github.com/hat47x/tei-platform/issues/951)が正本。Plugin名の予約だけでContract定義権限は与えない |
| Release Digest | 版とManifest/Artifact集合の厳密な内容同一性 | Version Binding確定後は不変 |
| Artifact Digest | 取得した個々のバイト列の同一性 | Locator（GitHub URL等）とは独立 |
| Verification Statement/Attestation | 対象Digest・Profile・Issuer・検証結果の証拠 | 同一Releaseに複数Issuer/版の証明を追加可能 |
| Instance / Composition Lock | Consumerが選んだ具体的なProvider・Binding | 公開Registryが正本を発行しない |

`tei.std.*` / `tei.experimental.*` / `tei.contrib.*` はContract側の提案中の管理分類。Pluginの公開名に逆ドメイン名を要求しない。名前分類だけで信頼・互換性・Runtime権限を決めない。

**比較待ち**：Plugin IDを不透明な登録UIDで発行する方式と、署名されたGenesisの安定CoreからDigestを導く方式をReferenceで比較する。後者もDigest一致だけで正当な名前予約を保証しない。正式なID文字列構文・文字正規化・Authority委任・鍵回復は未採用。

## 3. 二つの状態機械

### 3.1 登録

`Reserved → Linked → Validated → Registered`。予約時はTTL・予約数制限を持ち、LinkedではGitHub App等で対象Repositoryの**不変Repository ID**と管理権限を確認する。初回検証を通過し、確定直前に所有権を再確認してRegisteredにする。登録の競合は原子的な一意制約で処理し、未完了予約は正式Plugin IDと扱わない。

移転、Repository変更、鍵ローテーション、失効はRegisteredを上書きするのではなく、署名・世代に結び付いた別の管理履歴として記録する。旧Authorityが失われた場合の回復方式は未決。回復証拠がない無断移転は拒否する。

### 3.2 Release公開

`Submitted → Validating → Published`。不正な内容は`Rejected`、外部障害などで判定できない場合は`Inconclusive`とする。公開後の取下げ・失効は不変のRelease Descriptorを書き換えず、別状態・別の信頼情報で示す。

公開者はPlugin ID、Version、Release Descriptor Digest、Publisher権限の世代、冪等キーに結び付けた公開宣言を署名する。検証済みの固定Digestと**公開確定時の現在の権限/Epoch**を照合し、`(Plugin ID, Version)`に別Digestが存在しないことを原子的に確かめる。同Digestの再申請は冪等、別Digestは競合として拒否する。

`stable` / `beta` 等のChannelは署名付きの可変ポインタ。厳密なLockにはChannel名だけでなく採用Release/Artifact Digestを保存する。既存版の「同じVersionで修正版を上書き」は認めず、新Versionで発行する。

## 4. 公開宣言・検証証明・現在の信頼

- **Publisher Release Declaration**：正当な公開者が対象Release Digestをその版として公開する意思を示す。CI等の短期署名主体は現在のPublisher委任で限定する。
- **Conformance Attestation**：特定Release/Component/Contract/Target/Profileに対して実行した検証の結果（合格・不合格・判定不能）を示す。Profileの不変識別子、Verifierの版/実体、実際に読んだ入力、Evidenceへの参照を残す。
- **Transparency Receipt（任意）**：署名済み声明が透明性サービスに記録された証拠。契約適合や安全性の保証とは別。
- **Consumer Trust Policy**：現在のIssuer権限、鍵失効、検証結果の有効範囲、必要な鮮度を評価する。署名が正しいことと現在信頼できることは別。

外部交換形式には[in-toto Attestation Framework](https://github.com/in-toto/attestation)、[SCITT RFC 9943](https://www.rfc-editor.org/rfc/rfc9943)（透明性）、[TUF](https://theupdateframework.io/)（更新・鮮度・Rollback防止）を候補とする。これらは役割が異なるため、一律の単一Receiptに統合しない。SCITTのCOSE形式とin-totoのDSSE形式は同一ではない。具体的なBinding、鍵規律、プロトコル採用はReferenceで比較する。

「公開サイトが払い出す検証フィンガープリント」は、**実際に発行した署名付きAttestationの正確なバイト列を識別するDigest**を有力候補とする。単にIDが存在するだけでは検証済みとしない。同一Subjectに複数のAttestationを認める。

## 5. 証拠の結び付けと保証範囲

| 境界 | 証拠・判定 | Owner |
| --- | --- | --- |
| 申告Source → 実Build入力 | 宣言対象、実際に消費したSource closure、入力Byte/Digestの対応 | #943 / #2016 |
| 実Build入力 → 生成Artifact | Build Plan・実行証拠・生成Byte/Digest | #943 |
| 生成Artifact → 公開Artifact | 配布先から実取得したByte/Digest、必要に応じOrigin/Repo ID | #1105 + Registry Adapter |
| 公開Artifact → Attestation Subject | 署名対象のRelease/Artifact/Contract/Target/Profile一致 | #1104 + Registry交換 |
| 検証Artifact → 使用/実行Artifact | 起動・組込み時の実体同一性、現在の権限 | #1370 / #1121 |

**Content Identity / Origin・Authority / Actual Use Binding** は独立する。Digest一致はSourceの正しい由来や実際のBuild採用を自動保証しない。Source Pro-venanceが取得できない第三者バイナリは「Build来歴未検証」として公開可能だが、Consumer Policyが要求すれば採用しない。

検証状態は`passed / failed / inconclusive / not-evaluated`を区別し、信頼状態の`current / stale / untrusted / unknown`とは別軸にする。公開FixtureのPASSを全入力への意味保証とせず、独立Oracleや別Hostの実測は別Evidenceにする。

公開・ビルドの見取り図は、既存のDeclared Graph / Resolved Build / Observed Evidenceを参照する**読取り専用のPublication Evidence投影**とする。「図にない工程は存在しない」と言えるのは、確認済みの閉じた依存・実行範囲に限る。外部CIの隠れた作用、観測していない通信、未知のBuild入力は`unknown`を保持する。第二Build Graph/Lock/受理正本を設けない。

## 6. RegistryとTEI/外部Consumerの境界

RegistryはPlugin名、提供Contract、Component、Target、Release、適合性証拠を検索・配布する。Providerの確定、依存Closure、Composition Lock、Target Loweringは[#951](https://github.com/hat47x/tei-platform/issues/951)/[#1121](https://github.com/hat47x/tei-platform/issues/1121)の既存境界に委ねる。候補が複数なら検索順位で暗黙選択しない。

Consumerは必要なProfile・信頼の鮮度・Host Import・Resource/Permissionと現在権限を独立検査する。公開検証のPASS、Versionの署名、UIからのインストール申請、AIによる推薦をRuntime Authorityや業務意味受理に昇格させない。

[#2390](https://github.com/hat47x/tei-platform/issues/2390)に沿い、TEIのWorkspace/Studio/CLI/共通常駐Hostを要求せず、別製品のJava/.NET/Node/Process/Wasm Consumerが同じ公開証拠を独立に検査できる経路を目指す。WIT/MCP/Hostの種類は一つに固定しない。Plugin Type、Package、Artifact、論理Instanceは別であり、一つの型の複数Instanceや一つのArtifactの複数Contract実現を許す。

## 7. 最小Referenceと反証計画

| 段階 | 正例 | 必須負例・停止条件 |
| --- | --- | --- |
| P0-A：形式/独立検証 | 登録記録・Release・署名済み公開宣言・Attestationを二つの独立した読み手で照合 | 別Release、別Contract、別Target用の証明流用を拒否 |
| P0-B：Source/Artifact連鎖 | 既存Self-build/Artifact Evidenceを公開Descriptorへ投影 | #2016の申告D1/Build D2を取り違えない。未観測はNOT PROVEN |
| P0-C：所有/版競合 | 現Publisher権限で版を確定 | 同名競合、別Digest同Version、検証中のPublisher失効を拒否 |
| P1：配布 | GitHub Release等から取得しDigest照合 | URL/Tag同一で別Byte、危険URL/圧縮爆弾・サイズ超過を拒否 |
| P2：二Consumer | TEIと外部Hostで同じRelease/証明を評価。#2390の既存Oracleを再利用 | 署名だけでHost権限や全Contract互換を主張しない |
| P3：公開API/可観測性 | 名前予約→初回登録→検索→追加Releaseの一巡 | 二重Authority、第二Resolver、暗黙Provider選択、無期限の失効鮮度を拒否 |

P0-AにはWebサイト、GitHub App、Registry常駐を要求しない。実GitHub App登録・隔離検証Runner・公開API・透明性・Registry間移転・Marketplace UIは後段。独立した初回二Consumerで確認するのは証明の可搬性であり、全Target実行の成立ではない。

オフラインでは保存済み署名・Digestを検査できても、最新失効情報を必ず確認できない。新規採用と既存Lockの継続は別Policyとし、`current`を偽装しない。悪意ある旧Registryが分岐した所有権履歴を示す場合は保留し、合意や透明性監視がない限りグローバル一意の最新状態を主張しない。

## 8. 現在性・保留・撤回条件

- **現状**：本書は計画・設計の差分のみ。新規API/CLI/サーバー、署名確認、GitHub App、外部Consumerの新規試験、Production権限、Registry間移転はNOT RUN。現行[`docs/status.md`](../../docs/status.md)の製品状況を変更しない。
- **人間判断が必要**：正式Plugin IDの生成法、Namespaceの正規化、Publisher鍵/回復・現在性のTrust Policy、Registry移転規律、署名形式採用。Draft文書を採用済みADRとはしない。
- **撤回/縮小**：公開専用ManifestがSelf-build/Contractと二重正本になった場合、実Consumerを通さず新Resolver/Authorityが必要になる場合、署名証明が既存#1104より安全性・追跡性を改善しない場合は抽象を縮小する。
- **他レーン**：#1124（Studio Consumer）、#2391（任意UI外部制御）、X-TEA三要素分析Pluginは公開方式の依存先にしない。RegistryにAuthoring/業務受理/外部UI操作の権限を与えない。

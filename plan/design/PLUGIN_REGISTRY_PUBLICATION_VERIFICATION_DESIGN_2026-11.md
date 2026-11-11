# 分散型Plugin公開・登録・検証プロトコル（設計候補、2026-11-02）

- **状態**：設計候補。登録サービス・署名検証CLI・外部Consumer・実Runtimeの実装/実証を意味しない。規範仕様の正式採用はOwner判断待ち。
- **追跡Issue**：[ #2589 ](https://github.com/hat47x/tei-platform/issues/2589)。
- **目標アーキテクチャ**：[ADR-0032](../adr/0032-composition-kernel-replacement.md)、[#951](https://github.com/hat47x/tei-platform/issues/951)（Contract/Provider/Authority）。名前空間の設計方針は[main統合済みPR #2298](https://github.com/hat47x/tei-platform/pull/2298)で反映済み。ただし正式な公開ID文法・旧ID移行・実Consumerの成立は別Gateで未確定。
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

`tei.std.*` / `tei.experimental.*` / `tei.contrib.*` の予約ルート方針はmainへ反映済み。ただし具体的なPublisher委任と旧ID変換は未資格。Pluginの公開名に逆ドメイン名を要求しない。名前分類だけで信頼・互換性・Runtime権限を決めない。

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

---

## 9. Registry公開とArtifact一式引渡しの分離（2026-11-06、設計追補）

### 9.1 最新の他レーンの成立範囲

2026-11-06確認のmain `2d0b09b13b16c7f2d7183dbc995d3c62208c79ef` を静的照合の基準とする。名前空間・Instance分離の[PR #2298](https://github.com/hat47x/tei-platform/pull/2298)、独自Host非必須の[ADR-0043](../adr/0043-target-native-artifacts-without-tei-host.md)（実際のファイル名・受入状態は関連ADR/Issueで再確認）は**目標方針と実装資格を区別**する。

以下の論点はRegistryで再実装しない。

| 接続先 | 既存Owner・状況 | Registry側で追加する差分 |
| --- | --- | --- |
| 可変長・可変文字集合のHashId | [#2814](https://github.com/hat47x/tei-platform/issues/2814) / [Draft #2815](https://github.com/hat47x/tei-platform/pull/2815) | Registryの公開名・表示用短縮IDと、不変Content Digest/署名対象の分離 |
| TEI全体Self-build | [#2831](https://github.com/hat47x/tei-platform/issues/2831) / [Draft #2877](https://github.com/hat47x/tei-platform/pull/2877) | 実Build入力・実成果物の対応を公開宣言へ渡す条件。TEI全体Build成功はRegistry初回利用の先行条件ではない |
| Artifact一式公開 | [#2604](https://github.com/hat47x/tei-platform/issues/2604) / [#2817](https://github.com/hat47x/tei-platform/issues/2817) | 検証した全対象が一つの公開済みReleaseとしてConsumerへ引き渡せるか。途中失敗・世代混在の防止 |
| ETL/ELTとBuildの標準型 | [Draft #2880](https://github.com/hat47x/tei-platform/pull/2880) | Source/Transform/SinkとSource/Build/Artifactの共通Port・依存/来歴だけを利用。配信成功とBuild成功は別Claim |
| NamespaceとRegistryのIdentity | [Draft #2879](https://github.com/hat47x/tei-platform/pull/2879) | 重複の公開Authorityモデルを作らず、下記の公開確定と取得時の証拠引渡しに限定 |

### 9.2 HashId Schemeと内容照合値は別用途

表示の文字数・文字集合・encodingを選択できる`HashIdText`や`exactCharacters`を、Release/Artifact/Attestationの暗号学的な内容照合値と無言で置き換えない。`exactCharacters`は出力文字数制約に適合させた導出であり、元のDigestを可逆復元できる文字列表現ではない。短縮IDの衝突・DB照合順序・Target正規化が起きても、完全な内容同一性やIssuerの権限を証明したことにしない。

Registryにおける`Registered Name`、`Plugin ID`、`Release Digest`、`Artifact Digest`、`Attestation Fingerprint`の用途を維持する。v0の独立検証では既存のSHA-256内容照合経路をReferenceとして再利用し、digest profile/algo変更やGenesis Digest型Plugin IDの本採用は[#2814](https://github.com/hat47x/tei-platform/issues/2814)/[#951](https://github.com/hat47x/tei-platform/issues/951)のOwner判断を要する。ID長・alphabet自由化を理由に既存Lock/Canonical ID/署名対象byteを自動変更しない。

**反証ID-1**：Base62の短い表示IDが衝突した場合、RegistryはPlugin/Releaseの誤結合・自動再割当・信頼昇格を行わず、完全なIDとSchemeを用いて曖昧性を拒否する。

### 9.3 一式公開の線形化点と、後追いの配布可否

複数Artifactを含むReleaseは、`Release Descriptor`の内容一致とは別に、**公開先から同一世代の必須Artifact集合を読み戻せること**を検査する。GitHub ReleaseのTag名・Asset表示名・単なるUpload成功では不十分。外部Providerのアトミック更新保証を未確認のまま仮定しない。

公開候補について以下を区別する。

```text
immutable Release Descriptor + mandatory ArtifactSet
      -> Staged / uploaded (Consumerへ未確定)
      -> Readback of exact declared bytes + complete required set
      -> Verify Publisher authority / epoch and version uniqueness
      -> Registry Publication Commit (Publishedへの線形化点)
      -> Consumer fetch by digest + independent read/verify
```

- 最初の`Published`の意味はRegistry内の**検証済みVersion Bindingの確定**までとし、外部URLが将来も常時到達可能であることは保証しない。取得可能性は独立した現在性観測とする。
- 同一`(Plugin ID, Version)`の競合は公開確定時にcompare-and-swap相当の原子的検査で拒否する。現在のPublisher Authority/Epochも同じ確定判断に参加させる。
- 途中Upload失敗・Readback不可・欠損Artifact・Version競合・Owner変更では`Published`としない。部分Uploaded bytesは公開履歴の成功ではなく、別の後始末/再試行対象である。
- ConsumerがManifest R1を読んでから公開先がR2へ切り替わっても、R1が要求したArtifact Digestと一致する実byteのみ利用する。同じDigestの別Mirrorから取得するか、取得不能として保留・拒否する。R1/R2混合を成功扱いしない。
- 旧版Artifactが物理的に残っていても、新版のDescriptorの必須集合へ自動的に加えない。必要集合の宣言・実取得・利用を別々に示す。
- Registryの公開確定とGitHub側の物理配置を単一の分散トランザクションと呼ばない。部分失敗時の失敗・保留の証拠を残す。

この境界は[#2604](https://github.com/hat47x/tei-platform/issues/2604)のArtifact一式公開と重なる**性質の再利用**であり、#2604の`tei artifacts -out`をRegistryへ移植する意味ではない。

### 9.4 Referenceの最小横断負例（設計のみ、未実行）

| Case | 実験条件 | 必須の判定 | 既存Owner |
| --- | --- | --- | --- |
| RG-P1 | Release Descriptor R1確定後、外部URLでR2のbyteを返す | R1のDigest不一致として拒否 | #1105 |
| RG-P2 | 2つの必須Artifactの1つのみ公開に成功 | 一式Publishedを禁止、Partial/Unknownを保持 | #2604 |
| RG-P3 | R1 Manifest取得後にChannelがR2を指す | R1の必須Artifactだけ取得するか、明示的に失敗 | #2604 / #1105 |
| RG-P4 | 別の正当なLocatorで同一byteを取得 | 内容一致は認めるが、Origin Claimは独立検査 | #1105 / #1104 |
| RG-I1 | 短いHashIdTextが異なるReleaseで衝突 | 同一視せず完全Digestで分離、表示は曖昧を明示 | #2814 |
| RG-B1 | Declared Source D1、Observed Build Input D2 | D1のBuild Provenance合格としない | #2831 / #2016 |
| RG-A1 | 署名・適合検査中にPublisher Epochが失効 | 最終Publication Commitを拒否 | #951 / #1104 |
| RG-C1 | Plugin単体の検証PASSのみ、実HostのAuthority/Effectが欠損 | Registryの候補止まり、Runtime資格は未検証 | #951 / #1121 |

**完了条件の区別**：RG-P1/P2/P3はRegistryの公開一式・物理参照のReference、RG-B1はSelf-build Ownerの独立Evidence、RG-C1はConsumer/Targetの資格であり、すべてを一つの`verified`真偽値へ丸めない。既存#2604・#2831・#2814の実験をRegistry側で重複作成せず、既存結果を正確な版・対象へBindingする。

### 9.5 必要証拠の閉包を明示し、無根拠なPASSを防ぐ

候補の資格判定は、当該操作に必要な証拠集合`RequiredEvidence`と、実際に照合済みの`ObservedEvidence`の対応として示す。`Undischarged`（不足・評価不能）を保持し、必要な証拠が未取得なら`qualified`と主張しない。

```text
required: release subject + digest + publication authority + profile/target
observed: exact-readback + signed declaration + profile-scoped result
undischarged: e.g. current revocation, unobserved Build closure
=> candidate or unresolved; never automatic runtime grant
```

この投影は既存Build Evidence、Contract Conformance、Trust Metadata、Target Qualificationの参照を結ぶ**読取り専用診断**であり、第二のEvidence Engine/Registry Resolver/Publisher Authorityを作らない。影響範囲が確認できない場合は「無影響」と宣言しない。

**当該追補の状態**：最新mainと既存Issue/PRの静的照合に基づく設計差分。Registry署名実行・実GitHub配布・実Consumer・Go/Node試験・Full CIはこの追補ではNOT RUN。P0の独立二Reader・P1の公開readbackを実証するまで製品資格は未成立。


---

## 10. Consumer操作ごとの公開証拠資格（2026-11-06、#2589境界の追加）

### 10.1 発見、検証、採用、実使用を別の操作とする

第三者のPlugin Indexが示す「掲載済み」「公式」「検証済み」「安全」などの表示を、TEI内外の現在の実行資格へ昇格させない。[#2899](https://github.com/hat47x/tei-platform/issues/2899)からの比較研究が[#2589](https://github.com/hat47x/tei-platform/issues/2589#issuecomment-6669783120)へ接続されており、**発見可能であること、公開権限があること、内容一致、Contract適合、Consumer採用、実使用は別の事実**として扱う。

| 操作 | Registry/Consumerの判定対象 | 判定できる範囲 |
| --- | --- | --- |
| Discover | 公開名・Plugin ID・候補Releaseを索引から発見 | 候補の存在。署名/内容/意味/安全の検証ではない |
| Fetch/Verify | 取得した完全な必須Artifact集合とDescriptor・署名・Profileの一致 | 明示された内容とその証拠に限定 |
| Select/Adopt | Consumerの必要Contract・Target・現在Policy・選択済Provider/Artifact Lock | そのConsumerにおける固定候補・採用判断。Runtime Grantではない |
| Load/Dispatch | 対象Hostの現在Authority・Permission・実起動byte・Effect対象 | 選択先・権限世代・実効果が一致するときだけ実行可能 |

Registryの検索応答は自らを`Adopted`/ `Executable`と宣言しない。ユーザー・AIの評価や公開サイトのランキングもAuthority/Conformanceの代替ではない。正しい署名付き説明でも、不正確な内容の可能性は残る。

### 10.2 必要証拠集合はConsumerの用途・Target・実選択結果で確定する

第9節の`RequiredEvidence`を、Registryが全候補に共通の固定集合として先に確定しない。必要な根拠は選択した操作、Consumer Policy、現在Target、既存Resolverが確定した**選択閉包**、Build/Runtimeが実際に消費した**実消費閉包**、証明が覆う**検証範囲**によって異なる。選択されなかったことだけを、実際に使っていない証拠とはしない。

```text
Intent / Target / Consumer Policy
    + Existing TEI selection / Lock (where applicable)
      -> Required Claims + selection / actual-consumption closure
      + Observed exact bytes / signed statements / current trust evidence
      -> Read-only Qualification Projection
      -> passed | failed | inconclusive | not-evaluated
         with missing / stale / unsupported separately diagnosed
```

`RequiredClaims`、`ObservedEvidence`、`Unsupported`、`Missing`、`Unresolved`を一律に`false`へ潰すのではなく、**何が不一致で、何が未取得で、何がそもそも対象外なのか**を表示する。ただし、必要な根拠に未解決・失効・Target不適合があれば、その操作に対して`passed`を返さない。

規範的な判定条件の概念例（新Core型の導入ではない）：

```text
For the selected operation O and consumer/target T:
  R := requirements(policy(O,T), exact selected lock/closure)
  E := verified observations / signed evidence / current trust snapshots
  qualified(O,T) only if:
    - the selected dependency closure and required actual-consumption scope are established,
    - each required r in R has a matching, trusted, nonstale witness in E,
    - witness binds exact subject digest + contract/profile + target/scope,
    - no unresolved conflict, explicit deny, or required unknown remains.
  Otherwise: report diagnosis and do not claim qualification.
```

- Contract/Providerの選択と選択閉包は[#951](https://github.com/hat47x/tei-platform/issues/951)・[#1016](https://github.com/hat47x/tei-platform/issues/1016)へ委ねる。Plugin IndexやRegistryで第二Resolverを作らない。
- `nil`（閉包未提供）と評価済み空集合`[]`（実際に0件）を同一視せず、未登録`tei.std.*`を旧`std.*`へ暗黙変換しない。[Draft #2584](https://github.com/hat47x/tei-platform/pull/2584)のPreflightや[#1121](https://github.com/hat47x/tei-platform/issues/1121)のSourceInputs境界を**局所の例**として照合する。これを新しいRegistry Canonical Schemaの即時採用としない。
- Self-buildで`cmd/tei`等の実外部Module/Toolchain入力が未知のときは、Source/Build Provenanceの資格を未成立のまま保持する。[#2831](https://github.com/hat47x/tei-platform/issues/2831)のSB-0（選択された外部Module、依存Byte、offline供給の閉包）を待ち、Plugin公開の内容照合PASSを実Build来歴の証明へ転用しない。
- 依存Closureと実消費範囲が確認できる場合だけ局所再資格化を許す。未選択かつ実非消費と証明できたPluginの更新で無関係Consumerを一律失効させず、選択・実消費された入力、Contract/Profile、Policyの変更を検査する。未選択でも静的リンク、Goの`import`/`init`、`embed`、生成処理等で実消費され得る。依存が未調査なら`impact not assessed`を維持する。

### 10.3 採用後も信頼は非単調に変化する

Release/Attestationの歴史的記録が追記専用でも、Consumerの現在の信頼判定は単調増加しない。鍵失効、Profile改訂、Issuerの委任変更、Runtime Permission取消、対象Targetの変更で、以前の`qualified`は現在の`qualified`を保証しなくなる。

- Registryは古いAttestationを削除して過去を書き換えず、失効・退役・新証拠を別記録とする。
- Consumerは導入時に検査した署名と、今も利用してよいかのPolicy/Authorityを分離する。Offline時には現在の失効確認に限界があるため`current`を偽装しない。
- Source更新・Provider更新・Contract Fingerprint変更・選択依存変更時の再評価は、必要な実選択関係と証拠範囲が確定したものだけ局所最適化する。
- すでに実行中の対象の停止・再許可は実Host/Runtimeの責務。Registryは実行権限を発行しない。[#1174](https://github.com/hat47x/tei-platform/issues/1174)・[#1370](https://github.com/hat47x/tei-platform/issues/1370)の既存境界を再利用する。

### 10.4 操作別の反証Fixture（Reference未実行）

| Case | 最小入力差 | 期待判定 | 担当 |
| --- | --- | --- | --- |
| RG-Q1 | Required closureが`nil`なのに空集合として扱う | `not-evaluated/unresolved`を保持し、Adopt合格にしない | #951 / #1121 |
| RG-Q2 | 未選択かつ実非消費まで独立確認できたPluginのReleaseだけ更新 | 選択/実消費/証明範囲を確認できるConsumerだけ再資格化不要。未選択だが実消費なら再評価 | #951 / #2600 / #2831 |
| RG-Q3 | 選択ProviderのArtifactまたはContract/Profileだけ変更 | 該当Consumerの必要証拠を再評価し、旧証明を新Subjectへ流用しない | #951 / #1104 |
| RG-Q4 | Install時の信頼は有効、Dispatch直前にIssuer/Grant失効 | 現在権限のOwnerが明示拒否/未検証にする。Registry署名だけで起動しない | #1174 / #1370 |
| RG-Q5 | 古いTrust SnapshotをOffline Consumerに再提示 | 有効期限・観測可能な版の限界を明示し、新鮮な信頼確認を装わない | #1104 |
| RG-Q6 | Source/Build closureの実外部Moduleが欠損 | Build由来の主張だけ未資格。単なるArtifact内容照合と区別 | #2831 |
| RG-Q7 | Plugin Indexに`safe`と表示、Publisher/Conformanceの独立証拠なし | Discoverable止まり。Adopt/Executableへ昇格しない | #2589 / #2899 |
| RG-Q8 | GitHub Releaseの複数Artifactの一部だけReadback不能 | 完全なRelease公開としない。前節RG-P2の必要集合へ結合 | #2604 / #1105 |

**優先順**：公開/取得同一性のRG-P1/P2/P3 → 操作別閉包のRG-Q1/Q6 → 明示信頼・失効のRG-Q4/Q5 → 局所再資格化RG-Q2/Q3。P0でSchemaや汎用Evidence Engineを新設せず、既存のContract/Build/Trust/Artifactと二Readerによる観測可能な不足だけを確認する。

**状態**：2026-11-06の他レーンの設計・Issue/PR静的照合による追補。RG-Q1〜Q8のRegistry実行、署名、実GitHub公開、外部Consumer、Owner受理は**NOT RUN／未受理**。既存の他レーンで記録された限定試験をRegistryの新規PASSと数えない。


---

## 11. 公開後ライフサイクルと実消費閉包（2026-11-06、#2589増分設計）

### 11.1 「公開済み」と「現在利用可能」を別軸で扱う

Registryの初回登録、Version Bindingの公開確定後にも、Publisher移転、保守終了、既知脆弱性、配布先の削除・障害、Issuer失効、利用者による無効化が起きる。**公開した歴史的事実を削除して解決するのではなく、何が変更されたかを別の権限主体・証拠で観測する。**

| 変化する対象 | 既存の正本／Authority | Registryの投影・Consumerに引き渡す情報 | 誤ってしないこと |
| --- | --- | --- | --- |
| Version Binding / Release Descriptor | 公開者の署名・登録確定時点のAuthority、#2589 | 不変Release Digest、Version、Publication Commitの来歴 | yankで過去の公開が無かったと書き換えない |
| Discovery Listing / Channel | 公開索引の編集権限、署名付きポインタ | Listed/Hidden、stableなどの現在の指示先と版 | 非掲載をArtifact失効・脆弱性と断定しない |
| Distribution Locator / Availability | GitHub等の配布者、#1105 | 取得候補、最終Readback時点、取得失敗・ミラーの位置 | locatorの存続や到達性をRelease内容の永久保証にしない |
| Publisher / Verifier Trust | #951/#1104、各Consumer Policy | 鍵委任・失効・現在性、対象Scope、適用版 | 署名済みの古いAttestationだけで「現在Trusted」と言わない |
| Advisory / Yank / Deprecation | 公開者・第三者Advisory IssuerとConsumer Policy | 対象Release/Artifact/Contract、理由の分類、署名/Source/時点、代替候補 | advisoryを一律削除指令・実行許可・強制代替にしない |
| Install / Enable / Data Retention | #2044 のStudio管理と実Consumer/Host | 導入・有効化・依存関係・実Load/実使用・保持データの区別 | 無効化や非掲載でSource・保持データを自動削除しない |

上記は新Core固定enumや二重Status DBの提案ではない。各正本の変更イベントと現在の検証結果を**参照する読取り専用投影**の項目候補である。Index側の安全・人気・保守評価は、独立した署名・Conformance・Consumer採用に自動昇格させない（[#2899](https://github.com/hat47x/tei-platform/issues/2899)、[Draft #2903](https://github.com/hat47x/tei-platform/pull/2903)）。

### 11.2 公開後イベントの対象Bindingと受理条件

停止・廃止・脆弱性告知などのイベントは、その表示名やGitHub Tagだけでなく、少なくとも**発行主体/Issuer、対象Plugin ID、Release/ArtifactまたはVersion Range、Event ID/版、発効時点、理由分類、参照するEvidence、署名・信頼の評価**へ到達可能とする。Rangeに未確定版を含める場合は影響候補とし、解決した正確なRelease Digestに対する判断と区別する。

- **Yank**（新規導入候補からの取下げ）と**Security Advisory**（脆弱性に関する主張）、**Deprecation**（利用非推奨）、**Deletion / Legal Takedown**（物理配布の削除）は違う操作とする。署名付きイベントでも内容の真正性、第三者の確認、Consumerの現在Policyは別評価。Yankだけで既存実行を「安全」または「危険」と決めない。
- 既存Lockを保持したConsumerはRelease Digestを黙って変更しない。導入時点での証明と現在のIssuer/Publisher/Artifactへの注意情報を再評価し、必要なPolicyなら利用停止・保留・明示的な更新候補の提案へ進む。別Providerの自動勝者選択は禁止（#951）。
- Publisherが鍵を失った、Repository所有者が移転した、登録Authorityが停止した場合でも、公開済みの記録のDigest一致と現在の管理主体の承認は独立。無署名のREADMEや「新しい管理者」を名乗るProfileだけで既存Plugin IDを再割当しない。
- 指定Targetの実行中Sessionを停止・Drain・再起動させる責務はHost/Runtimeに置く。Install/Enable/Source維持/データ削除は[#2044](https://github.com/hat47x/tei-platform/issues/2044)の既存管理境界に委ねる。静的単一BinaryにHMRや常駐Loaderを強制しない。
- 既存Issuerの過去の署名が有効であることと、現在のOwner/Issuerが適格であることを別々のClaimとして出す。Offlineで現在性不明なら`unknown/stale`を保持し、古いRegistry索引の`safe`表示は権限根拠としない。

### 11.3 影響範囲の反証：未選択と実非消費は異なる

[Draft #2903](https://github.com/hat47x/tei-platform/pull/2903)の構成・実使用境界を参照する。**Resolved Selection（選択されたProvider）だけからActual Consumption（実Build/実Load/実Invokeの消費）を推定しない。** 例えば、未選択のPluginがGoの`import`/`init`や静的link、embed、生成器・Toolchain依存として取り込まれている可能性がある。

影響を`unaffected`と呼べるのは、変更されたPluginが現在のConsumerの正確な選択閉包にも、当該Claimに必要な実消費閉包にも、証明範囲にも含まれないことを**当該固定版と試行の独立証拠で確認できた場合**に限る。これらの閉包が未知なら`impact not assessed`とし、配布側の自己申告を独立した不在証拠とみなさない。

また、実消費の証拠はClaim相対である。Build-timeのToolchain実消費と、Runtimeの実Load・呼出しと、業務Effectへの到達を同じ事実にしない。Runtime Grant失効は過去Buildの内容Digestを変更しない。過去のReceiptは現在のEffect許可を再発行しない。

### 11.4 次の最小Reference（既存E0〜E3に追加、未実施）

| Case | 操作と入力差 | 期待判定 | Owner |
| --- | --- | --- | --- |
| RG-L1 | リリースR1公開後にPublisherが有効署名でYank | R1履歴とDigestは不変。新規導入候補の可否はPolicy、既存Lockは勝手にR2へ変更しない | #2589 / #2044 |
| RG-L2 | Security AdvisoryがR1のみを対象に発行 | Issuer/署名/Targetと現在Policyを評価。範囲外R2に脆弱性を自動転記せず、R1を無条件安全ともしない | #1104 / #2589 |
| RG-L3 | 未選択PluginだがGo `import`/`init`/embedでBuild実消費 | 未選択を根拠に`unaffected`としない。Build/Artifact再資格化の対象へ回す | #2831 / #2600 |
| RG-L4 | 未選択・未実消費を実Readerと固定Build/Lockで確認 | 当該対象での無関係更新を過剰失効させない。ただし未確認Targetには一般化しない | #951 / #2831 |
| RG-L5 | Index停止中、R1の取得済みByteと古いTrust Metadataだけ存在 | Digest・過去署名は照合可。現在のTrust/AdvisoryはUnknownで、全操作の新規採用を自動許可しない | #1104 / #1105 |
| RG-L6 | Publisher移転後、旧Ownerの鍵から新Releaseを提出 | 現在の委任Epochに照らして拒否。移転記録の検証が不能なら保留 | #951 / #2589 |
| RG-L7 | Plugin無効化・配布停止後も既存Sourceと保持データが存在 | 未対応Source/データを保持し、インストール済み・有効・実Loadを混同しない | #2044 |
| RG-L8 | Publisherが代替Providerを提示、同Contractの別候補も存在 | Registry/Indexは候補提示のみ。既存Resolverの明示WiringとConsumer Owner判断なしに自動代替しない | #951 / #1016 |

**P0優先**は既存RG-P1〜P3のByte/ArtifactSet readbackとE0署名・Profile対象一致。続いてRG-L3/L4の実選択と実消費の差、RG-L1/L5/L6の公開後変化を反証する。L2/L7/L8の意味・権限・運用の結果は既存Ownerとの接合後。どの局所試験の成功もRegistry E2Eや製品受入へ自動昇格させない。

**状態**：静的Issue/PR照合による設計追補、公開プロトコルやCore schemaの正式採用ではない。RG-L1〜L8、実GitHub Release・署名・別Host・Producer/Consumer readback・CIは本追補ではNOT RUN。新しいPlugin Lifecycle Engine、第二Resolver、Host常駐基盤、全Pluginへの強制停止を追加しない。


---

## 12. 二実装によるRelease証拠束縛P0-A（2026-11-07／限定Reference）

### 12.1 設計統合後に固定できる小さな実証単位

[PR #2591](https://github.com/hat47x/tei-platform/pull/2591)による本設計文書のmain統合は、**Registry API・独立公開/Trust・E0〜E3の実行資格を確立した証拠ではない**。実際の新規実装を始める前に、[#2589](https://github.com/hat47x/tei-platform/issues/2589)の「独立Readerでも正確な対象に束縛された署名とByteの関係を検査できる」P0-Aを、最小の対象だけで検証する。

検証用Reference：[`tools/reference-plugin-registry/`](../../tools/reference-plugin-registry/README.md)（独立Go/Node、固定Fixture、テスト専用鍵）。これは既存[#1104](https://github.com/hat47x/tei-platform/issues/1104)のTrustの本番実装でも[#951](https://github.com/hat47x/tei-platform/issues/951)のResolverでもなく、**署名のByte Bindingと対象一致のみ**を検査する。Fixtureの文字列フォーマットは採用済みTEI Canonical Schema/Release Descriptor文法ではない。

| P0-Aの局所Claim | Referenceに必要なEvidence | 未到達の保証 |
| --- | --- | --- |
| Publication declaration signature | 外部に固定したPublisherのテスト用公開鍵、Releaseの実Byte、目的分離した署名 | GitHub Repositoryの所有権・実Publisher委任・失効の最新性 |
| Release/Artifact content binding | Release正確なByte Digest、2件の必須Artifact roleと実Byte、欠落・重複拒否 | GitHub/OCIからの実readback、同時更新時のConsumer一式受領 |
| Conformance statement scope | Release Subject Digest、Profile Digest、Target、`passed`と独立Verifierのテスト用署名 | Contract意味的適合、独立Oracle、実Host Target実行 |
| Test Policy inputs | 固定Owner Epoch、固定Profile、模擬Issuer信頼/鮮度 | 現在のAuthority、署名鍵管理、Production Trust/Revocation |

GoとNodeの実装が**同一のReference Fixtureを独立に読んだ場合の判定一致**だけを問う。実装されたGo/Node Readerが同じテスト用Issuerを許容しても、それは実Issuerが信頼できるという結論ではない。鍵とProfileは受入Policyから固定され、Fixture自身が提供した公開鍵やProfileを信頼根として採用しない。

### 12.2 Refactor・Production化を止める条件

- **新しいRegistry Canonical Schemaではない**：ASCIIの固定行順とtest-only署名prefixはfixtureの解釈差を抑えるためであり、正式なDigest profile、JCS/DSSE/in-toto/SCITT、可変長HashId、Name Authorityの採用を意味しない（#2814/#951/#1104）。
- **合成Conformanceである**：`passed`はテスト用Verifierの署名済みClaimであり、実Contract試験を実行した証拠ではない。実Profile/Target適合は#951/#2390の既存証拠に引き継ぐ。
- **模擬された鮮度である**：`fresh=true`、Issuer信頼・Owner Epochは外部PolicyのFixture値であり、Trust APIや取り消し情報から観測した現在性ではない。
- **実配布は未検証**：ArtifactはFixtureにBase64で同梱。GitHub Release、実Consumerのexact-byte readback、途中公開の原子的切替、Code/Load TOCTOUは#1105/#2604/#1370の別Gate。
- **独立Reader成功をE0全体の成功へ昇格しない**：Issuer証拠やRepository所有権、Registry登録・Version一意性、別の実Hostを使った実行は未接続。既存Gateの判定はOpen/Unqualifiedを保持する。

### 12.3 次に結合する順序

1. **P0-A固定対象検査**：各Readerが正例、改ざん、Role欠落/重複、Subject/Profile/Target違い、模擬権限失効を拒否できるか。同一データに対する両Readerの判定差を記録する。
2. **P0-B外部証拠へのBinding**：#1104から**実際に検証されたIssuer/Trust証拠**を入力として受けるための差分だけを抽出。固定テスト用鍵をProduction設定へ流用しない。
3. **P1配布実証**：#1105/#2604の実Artifact Set Member/Publication Generation/Consumer Readの参照を受け、全必須Memberの実byteとReleaseを照合。Repo/Tagの見かけだけでは合格にしない。
4. **P2 Consumer受入**：#2390の独立した外部Host＋#951のContract/Provider選択・Targetと意味Oracleを参照。実行権限/EffectsはOwnerに委ねる。

**この節の主張の上限**：実際のRegistry/API/CIでの合格ではなく、局所の試験用Byte/Signature Bindingの受入条件整理である。Repository変更・PRのCI・現在Trust・実GitHub配布は別Evidence。新しい署名/Key管理サービス、第二Resolver、第二Self-buildを作らない。


---

## 13. E0実装確認後のE1公開Readback保証と観測範囲（2026-11-11／横断再基準化）

### 13.1 mainで確認できる範囲と、未成立の境界

次の表は確認時main `2202baa96e5e845c996da62c8f8df27e00d45c43` の**Source/PR/CIに紐づく局所事実**であり、E0〜E3の全体資格ではない。本文§12の当時の「局所の試験条件」は履歴として保ち、今回成立した範囲を追補する。

| 境界 | 現在の証拠 | 成立しない主張 |
| --- | --- | --- |
| E0-S：署名対象/Artifact byteの結合 | [merged #3244](https://github.com/hat47x/tei-platform/pull/3244)：Go/Node独立Reference、固定テスト鍵のEd25519・SHA-256。PR head `5859fc62...` の専用Workflow success（Repository Healthは同runでfailure） | 実Issuer/Publisherの現在Trust、Registry登録、実Conformance |
| E0-W：共通Raw Wire解釈 | [merged #3809](https://github.com/hat47x/tei-platform/pull/3809)：mainの`tools/reference-plugin-registry/wire.go`/`wire.mjs`と共有`wire-cases.json`（正常3／拒否11、Raw bytes digest）。この追補ではmerge後の専用CI実行結果を取得していない | 製品用Canonical JSON・正式Attestation Profile・第三者Issuerによる実署名 |
| E1前段：実CLI→一式Reader | [merged #3256](https://github.com/hat47x/tei-platform/pull/3256)：実`tei artifacts -out`と別ProcessのReader、正規Manifest、Member実byte照合、専用CI success | GitHub Release/OCI等の公開先での原子的世代確定、独立外部Consumerのreadback |
| E1の反証：読み始めと終わりの二点照合 | [merged #4240](https://github.com/hat47x/tei-platform/pull/4240)：R1→R2→R1のABAと、列挙後に物理member pathがsymlinkへ切り替わる合成負例。OSがsymlinkを許可しない場合は当該ケースSKIP | 継続したGeneration pin、実Openした物理ObjectのOrigin/Scope、実Publisherの並走安全性 |
| Build provenance | [#2831](https://github.com/hat47x/tei-platform/issues/2831)／[merged #4299](https://github.com/hat47x/tei-platform/pull/4299)：childのstate使用やbuilder同一性について、自己申告と実使用を区別する必要性を実証 | 同じArtifact Digestを得たことだけでSource/Toolchain/Builderの実消費閉包を証明 |
| 公開保証・ID・BOM | [merged #3265](https://github.com/hat47x/tei-platform/pull/3265)のOpen/unpromised vs Stable、[merged #2815](https://github.com/hat47x/tei-platform/pull/2815)のHashId限定PoC、[#3267](https://github.com/hat47x/tei-platform/issues/3267)の組合せBOM設計 | OpenをStableへ昇格、短縮HashIdを完全Digestへ置換、個別Plugin PASSからBOM結合PASSを生成。旧BOM [#3297](https://github.com/hat47x/tei-platform/pull/3297) はclosed/unmergedでRQ-036は現mainでModel Map |

**重要**：GitHub上でPRがmergedである事実、個別の専用Workflow成功、実Readerの局所PASSを足しても、「RegistryのE1公開・二Consumer取得・現在Trust・Runtime採用」が完成したとは判定しない。反対に、E1全体が未実証だからといって、E0-S/E0-Wの固定Fixture実装を未実装に戻さない。

### 13.2 E1は「誰が、いつ、どの実byteを、どの保管先から取得したか」を分離する

既存§9の公開フローを、少なくとも**Publisher側の公開Readback**と**独立Consumerの実取得**という別のAttemptに分ける。

```text
Signed Release declaration (immutable subject)
      │
      ▼
Publisher/Publication attempt P
    external Upload/Publish effect  ──>  storage-specific result/receipt
    actual Readback of each required member byte + role + digest
    current Publisher authority/version conflict at commit
      │
      ▼
Publication Commit evidence: (Plugin ID, Version, Release Digest, P, authority epoch)
      │    ※ ここまででも将来のAvailabilityやcontinuous-generationは未保証
      ▼
Independent Consumer attempt C (C ≠ P)
    pinned Release Digest + exact mandatory member set
    actual acquired member bytes + member digest + provider/locator evidence
    current Consumer Policy / Trust + Target
      │
      ▼
content observed / acquisition failed / origin unverified / current trust unresolved
      │
      ▼
Host-specific Load / Grant / Effect evidence (separate owner)
```

- **Byte identity**：読めたbyteのDigest一致を主張できる。正しいDigestの別Mirrorは**内容一致**としてよいが、元Publisher/配布Sourceの正当性は別。
- **Publication identity**：Manifest/Release Descriptorと全必須MemberのRole/Digest・Version Bindingを特定。途中失敗時に全Member Publishedとしない。
- **Origin・Scope**：pathを先に検査しても実Openした物理Objectは置換され得る。Source/StoreのRoot、権限、実Handle/Object version、Disclosureはbyte同一性と別。
- **Generation continuity**：開始時/終了時のManifest一致はR1→R2→R1（ABA）を排除しない。R1の旧memberを保持しているか、読取の最中に任意のgenerationが混ざっていないかは追加証拠が必要。
- **Current trust / actual use**：実取得成功や署名照合は、Publisher鍵・委任の現在性やHost Grant、正しく選択されたProviderの実Invokeを保証しない。
- **Build/source provenance**：生成物の内容一致から、`cmd/tei`を本当に同じBuilder/State/Toolchainで作ったと推論しない。実行側Source/Process/Inputの独立観測があって初めて該当Claimを扱う。

本設計で`attempt`/`member`/`locator`/`objectVersion`/`trustState`などを列挙しても、これを**新Core enum、第二Manifest、万能Store API、独自Registry Evidence DBの即時導入**とはしない。既存#1105のContent Store、#2604のArtifactSet、#1104のTrust、#951のResolverの証拠参照を束ねるだけとする。

### 13.3 次の最小接続・反証計画（#2604の既存試験を再実装しない）

| ケース | Registry側の追加問い | 独立Expected／停止線 | Owner |
| --- | --- | --- | --- |
| RG-E1-09 | 2必須Artifactの実ReadbackとPublication Attemptの対応 | Publish時に**全必須memberの実byte**が確認されない限りPublication Commit不可。単なるUpload ACKは不足 | #2589 → #1105/#2604 |
| RG-E1-10 | Publication Pに対しConsumer Cが**別Attempt**で取得 | Cの実byte・Release/Member Digestを再照合。Pの成功ログだけをCの受領証拠にしない | #2589/#2390 |
| RG-E1-11 | Manifest R1→R2→R1（ABA）をCのReader中に注入 | 前後R1一致だけで`continuous-generation`は証明しない。必要ならStorage側の世代不変・版固定の実証へ | #2604 AP-06/09、#4240 |
| RG-E1-12 | 事前に検査したMember Pathが実Open前に同byteの外部symlinkへ変化 | 内容Digest一致は認めてもOrigin/Root/Disclosureは`unverified`。実Handle/Object version資格が無ければScope保証不可 | #2604 AP-09、#4240 |
| RG-E1-13 | GitHub ReleasesからLocal/NAS/S3等へ同Digestを複製 | 別Locatorでもbyte identityは不変。ただしStore Origin/可用性/秘密Metaの送出許可は独立、GitHub登録を強制しない | #1105／[#4307](https://github.com/hat47x/tei-platform/issues/4307)／#2516 |
| RG-E1-14 | Valid digestのArtifactに新しいStable保証Statementだけを加える | 不変Releaseを書き換えない。Publisher保証・Verifier Conformance・Consumer Grantを別Issuer/Scopeで検査 | #3265/#1104/#951 |
| RG-E1-15 | Build成果物が同digestでも子BuilderがState/実入力を自己申告 | Build provenanceは`not independently verified`。actual Builder/State/Input閉包をConsumerのPublisher信頼へ自動昇格しない | #2831/#4299 |
| RG-E1-16 | 個別Plugin署名2件＋合成BOM候補のみ | 検証済みBOMやRuntime利用可能を検索結果で自動主張しない。BOM Ownerは#3267、現行Requirement番号は再照合する | #3267/#951 |

**実装順（並行可能な最小単位）**：既存E0-S/Wのmain上の固定版・CI対象を記録 → E1-Aとして#1105/#2604の**実Readback結果を引き渡すAdapter/参照**を設計、別P/C Attemptの実byte・Digest照合を最初に実証 → R1/R2混合とABA/TOCTOUをStorageごとに圧力検証 → E2のPublisher権限・Version CASを接続 → E3の実外部Host採用を独立検証。E1にBOM 2×2やTEI全体G4を硬い先行依存として置かない。

**未決の選択肢**：#2604が検討する(A)世代別immutable members＋generation pointer、(B)既存Storageのversioned object/conditional readのいずれかを、最初の外部Store/Hostで比較する。実Provider/FSが原子的切替・旧世代保持・安全なobject handleを保証するまでは、どちらも採用済み標準ではない。Providerを横断して「同一Digestだから全取得元がTrusted」「manifest二点一致だから連続世代」とは表明しない。

**検証状態**：この§13はmainのPR・Source・専用CI・既存Issueで報告された反証の静的照合をRegistryの受入設計へ接続したもの。RG-E1-09〜16のRegistry専用実装、実GitHub Release Publication Pと別Consumer Cの完結したE2E、Version CAS、Issuer/Owner currentness、実Host/Business Effectは**NOT RUN／UNQUALIFIED**。#4240の合成ABA/symlink反例を本番FSやGitHub上で再現したとは主張しない。


### 13.4 ETL/Batch・保管先横断のE1証拠を用途別に扱う（2026-11-11追補）

並行する[ETL/Batch設計Draft #4409](https://github.com/hat47x/tei-platform/pull/4409)は、型付きFunction／Data Operations／Operatorの合成で、固定TSV→変換→Artifact/Evidence→独立Consumerを段階実証する計画である。**同PRは未統合の設計候補**であり、実ETL Job／Scheduler／外部書込みが成立した証拠ではない。Registry E1と共通化してよいのは実Byte・必要Member・Content Digest・取得Attemptの**観測契約**だけとする。業務日付、Record粒度、NULL/欠損、Checkpoint、再実行・Effect/ACKの意味をRegistryのRelease/Version Commitへ畳み込まない。

[Git連動Evidence保管 #4307](https://github.com/hat47x/tei-platform/issues/4307)の方針も併せて、次の4軸を別々に保持する。

| 軸 | 証拠・正本 | 混同してはいけないもの |
| --- | --- | --- |
| **Content** | 実取得したMember bytesと完全Digest | 同Digest＝同じRun／同じPublisher／同じ取得元 |
| **Publication Generation** | Release/Manifestの必須Member集合、版固定と公開Attempt | Manifestの前後二点一致＝その間ずっと同世代 |
| **Run/Consumption** | Job/Build/Consumerの独立Attempt、実入力・結果・Receipt | Artifact取得PASS＝業務処理完了／Checkpoint再開の安全性 |
| **Disclosure / Trust** | 主体ごとの開示Policy、現在の委任・用途、Evidence保管先への権限 | ローカル/NAS/S3の存在＝GitHubや公開RegistryへMetadataまで送信してよい |

Registry登録・名前検索・第三者評価は**公開許可を持つPlugin Releaseや、その公開が許可されたEvidenceに限定**する。#4307の`local-only`／`git-linked-private`のEvidenceは、Digest/Locator・存在有無のMetadataであっても利用者の許諾なしにRegistryへ公開しない。公開探索結果の「取得不能」は、`not found`／`not disclosed`／`not authorized`／`temporarily unavailable`／`unverified`を区別してよいが、これらを無権限の検索者へ詳細に返して秘密の存在を漏らす設計にしない。利用者向けの診断と公開検索応答は異なるDisclosure Projectionとする。

同じArtifactのContent Digestを、GitHub Releaseと別ローカルCASで再確認できても、Repo ownership・Signer authority・実行Grantは独立。Local CAS [#1105](https://github.com/hat47x/tei-platform/issues/1105)が返すPut/Get Receiptや`#2604`の一式Readerは再利用するが、複数Memberの**公開確定と現在受取可能性**、秘密Metadataの開示はそれぞれ外側の責務に残す。

| Case | 最小反証 | 期待する区分／Owner |
| --- | --- | --- |
| RG-E1-17 | 同内容のTSV生成物をRun A/Bが出力。Content SHA-256は同じでCheckpoint/入力Snapshotが違う | Content一致は成立、Run provenance／差分再開の同一性は未成立（#4409/#1130） |
| RG-E1-18 | Plugin Artifact/署名ReceiptがNASにだけ保管され、Registry公開権限は与えられていない | 公開索引へLocator・Digest・存在情報を自動登録せず、私的Consumerのみ本人のPolicyで検査（#4307/#2516） |
| RG-E1-19 | 既存P/C取得が同DigestでもPは許可された公開Release、Cは権限不明の別Store | Byte一致は可能、Origin/Disclosure/Current TrustのClaimを保留（#1105/#4307/#1104） |
| RG-E1-20 | Artifact一式ReadbackがPASSだがETL JobのEffect／下流受領ACKが不明 | Registry公開のByte Claimだけを維持し、業務受理・再実行安全性・完了Receiptへ昇格しない（#1130/#2542/#1174） |

**状態**：公開証拠・公開の可否・用途別現在性の設計接続のみ。RG-E1-17〜20、実ETL Job、NAS/S3別Provider、private-to-public開示負例、Registry P/C E2Eの新規実行は**NOT RUN**。製品用共有Artifact Store／新Core型／Registry独自Effect Engineは追加しない。

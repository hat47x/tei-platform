# SB-1 v2入力Profile：Production Hostへの受け渡し条件（設計・検証境界）

対象: [Issue #1739](https://github.com/hat47x/tei-platform/issues/1739)。関連: #943、#951、#1016、#1104、#1109、#1173。

**状態: 移行Gateの設計メモ。Production Hostの接続、署名・Trust、Artifact Acceptanceの実装／合格を意味しない。**

## 1. 現時点で確かめたこと

- 旧SB-1 Sourceと固定済み決定論的transform Runtime Engineは、非正規の生JSONを黙示的に正規化しない。
- 明示された `tei.plugin-definition-json/v2` は、Source FingerprintとBuildPlan Fingerprintに一致する場合に限ってHost外側で入力を正規化する（`internal/plugin/inputprofile`）。
- 生成ManifestはGoのSB-1正規出力（**末尾LFを含むバイト列**）と照合し、重複キー・余計なフィールド・順序や空白の変化を拒否する。
- 実Process Engine二回のパリティ試験は `tools/reference-plugin-self-build/inputprofile_v2_process_e2e_test.go` にある。Go Oracle一致とProcess Engineの実実行の観測は、Hostの認可を証明しない。
- `internal/plugin/transform/plugin_inputprofile_distinct_ingress_evidence_test.go` では、**異なる二つの原入力が同一の正規入力・SB-1出力になっても、原入力digestを混同しない**ことを回帰固定する。

## 2. 移行の責務境界

| 段階 | 入力・判断を所有する主体 | 必要な確認 | ここでは成立しないもの |
| --- | --- | --- | --- |
| 案件・Environmentの選択 | Application/Environment構成を解決する既存Host | 明示された入力Profile、適用Scope、権限。未設定時にv2を自動選択しない | Build承認 |
| Source/Plan解決 | 信頼済みBuild Host・既存Lock | 実Source Fingerprint、exact BuildPlan Fingerprint、現時点のTrust・失効・適用Scope | 生成物受理 |
| v2 ingress | `PreparePluginPackageInputs` | 呼出しで選択されたv2、Source/Plan identity・fingerprint、Definition一致。非正規入力のopt-in正規化 | 選択値が真正のHostから来たという証明 |
| 物理Build実行 | 既存の実行Host/Engine | 実際に採用したSource、Plan、正規化入力、Engine/Builder identity、入力と実行Attemptの紐付け | 署名済みAttestation |
| 出力観測 | `ObservePackageOutput` | 正規SB-1出力バイト、Manifestと事前の入力指紋、出力digest | 実行Hostの真正性、二回独立実行、Artifact Acceptance |
| Attestation・受入 | 既存のBuild/Trust/Acceptance境界 (#1104等) | Host署名・現在のTrustと失効、actual Attempt、artifact subject、policy revision | 自動のRuntime/Deployment許可 |

**重要な非同値**: `BuildInputSelection` はHostが選択した値を格納するだけで、外部入力から偽造可能な文字列を安全にするものではない。`BuildInputEvidence` と `PackageOutputObservation` は不変な照合情報になり得るが、署名・Authorityではない。出力バイトが同じでも、二つの原入力、二つのHost判断、二つの独立Build Attemptを同一視しない。

## 3. Host接続の最小実装順（未実装）

1. 既存のApplication/Environment構成から**明示的な**Profile参照を引く。無指定時はv1の歴史的入口を維持し、正規化可能な入力だからという理由でv2へ上げない。
2. 既存のSource/BuildPlan Lockと現在の使用権限を**Host自身が**解決する。要求本文が提示したFingerprintを承認済み値としてコピーしない。失効・Scope外・不一致・未解決なら実行前に拒否する。
3. 選択したSource/Plan/Profileを変えずに `PreparePluginPackageInputs` へ渡す。結果のEvidenceは入力観測として保持する。Profile fallback、別Source探索、Plan再解決を行わない。
4. 既存Hostの実行Attemptへ、採用入力とEngine/Builderのexact identityを結び付ける。実行結果を認証できない場合、`ObservePackageOutput` の成功だけでは進まない。
5. Build Evidence、Attestation、fresh Trust/Acceptanceを既存の契約へ接続する。第二のAuthority/Lock/署名方式をSB-1固有に増設しない。

## 4. 次の受入試験（Production Host接続後に実施）

| ID | 変更・事象 | 期待する結果 | 現状 |
| --- | --- | --- | --- |
| SB1-H01 | v2を未指定、非正規JSONを入力 | 旧v1 Sourceが拒否。自動v2 fallbackなし | Host外側のSource試験あり／真正Host未接続 |
| SB1-H02 | 正しいv2 + 独立解決したSource/Plan Lock | exact入力束縛、Goと同じ出力 | API・Reference試験あり／Host認可未証明 |
| SB1-H03 | 原JSONのみ異なり正規JSONと出力は同じ | 原入力digestを区別、出力digestは同一 | 専用回帰試験を追加 |
| SB1-H04 | request由来の偽Source/Plan Lock | Host解決と異なるため実行前拒否 | 引数不一致試験あり／真正Host未接続 |
| SB1-H05 | Host選択の失効またはScope変更 | current Trustで拒否。歴史的fingerprintは不変 | 未実証 |
| SB1-H06 | 実行者、Engine、Attemptの出典が欠落 | 観測は可能でもAttestation・Acceptanceへ昇格不可 | 未実証 |
| SB1-H07 | Manifest値が正しくても非正規バイト | 観測拒否 | #1790で統合済み |
| SB1-H08 | 同一出力を二回独立Buildと自己申告 | actual Attempt二件の証拠がなければ再現性認定不可 | 未実証 |

実装時は、既存のHost/Build Attempt/Trust契約の所有者と変更パスを先に確認する。全体のFramework化や案件別プラグインを阻む強制Profile追加は行わない。**未実証のHost受理条件を満たすまでは Issue #1739をCloseしない。**

## 5. 現行Hostの選択読み取りに関する追加Reference（#1739）

`internal/plugin/inputprofile/current_host.go` は、呼出し側が保持する既存HostのCurrent selection readerを利用し、**呼出しのたびに** Application/Environmentに一致する明示Profileと固定済み指紋を取得する薄いアダプター。読み取りに成功した場合だけ既存 `PreparePluginPackageInputs` へ委譲する。Contextキャンセル、Hostが拒否した場合、Scope相違、未設定Profile、Source/Plan不一致は入力準備を拒否する。Host由来の失敗理由は呼出し側へ展開しない。

ここで提供するのは既存のHost-owned選択元を接続するための**受け口**だけであり、独立したHost選択元・権限管理・失効確認の実装ではない。単にCallbackを与えただけで真正なHostと認めてはならず、入力Profile／Lockに関する証拠も署名済みAttestationにはならない。読み取りから実行までの競合を防ぐ排他・Leaseも未実装のため、Production受入では同じ選択を実行中に保持する既存Host側の仕組みが必要になる。

`internal/plugin/transform/plugin_current_host_selection_test.go` の偽Readerで現在値の再読込み、失効エラー、別Application/Environment、要求文脈による偽Scope、Profile省略、不一致Lock、実行前キャンセルを試験する。ただし偽Readerの成功は本番の独立Current Authority検証とは区別する。外部Hostへ未接続の間は前章のSB1-H04/H05を受入済みに変更しない。

## 6. 読取りからProcess Startまでの失効競合（診断・未閉塞）

[Issue #976 のPR #1845](https://github.com/hat47x/tei-platform/pull/1845)は、案件Runtimeで「現在のEnvironment Grantを読み取った直後に失効しても、固定Planの実プロセスが起動できる」既知の競合を、Linux実バイナリで再現した。**この問題とSB-1の入力Profile選択は別の採用判断だが、読取結果の有効期間がStartに固定されていないという構造的な弱点は共通**する。

`tools/reference-plugin-self-build/inputprofile_v2_process_e2e_test.go` に、Host Readerを模擬して、(1) v2入力を有効な現在選択で準備、(2) 選択を失効、(3) 次の選択読取が拒否されることを確認、(4) **失効前に準備済みの同じ入力を旧v1物理Engineに渡すと実行される**ことを確認する診断を追加した。この試験が成功することは、「入力アダプターだけではStart前失効を防げない」という**既知の不足の実証**であり、安全な失効機能の合格ではない。`ObservePackageOutput`が古い入力から生成された正規バイトを確認できても、最新Host認可と実行開始の許可を証明しない。

Production Hostへの最小要求を次のように限定する。

- Application/Environmentの既存Authorityが発行した選択世代を、**実行開始を認める線形化点**まで検証／保護する。読取りとStartの間に失効またはScope変更が完了した場合、古い世代による新規Startを拒否する。
- 一度開始した処理の継続可否、実行中の取消し／Drain、既に生成された出力の受入は別途明示する。単なる再読込みだけで実行中取消しまで合格としない。
- 既存のHost Generation/Start Gate/Authorityの所有者と統合する。SB-1専用の第二のIssuer、Grant Store、署名方式、Resolverを作らない。案件プラグイン一般に強制する新Profileも設けない。

| ID | 検証対象 | 期待する製品受入 | Referenceの状態 |
| --- | --- | --- | --- |
| SB1-H09 | 有効なHost読取り直後、実Start直前に失効 | 失効が先に成立したなら旧世代の新規実行は拒否 | 実Processで**未閉塞の競合を再現**。Production Gate未達 |
| SB1-H10 | 過去の適正出力は一致、実行時点の権限が無効 | バイト一致とHost受入を混同しない | 出力観測APIは認可を付与しない。受入は未実証 |

本Referenceは診断試験であり、本番で競合を防止する実装は未着手。SB1-H04/H05/H06/H08/H09/H10の受理を宣言しない。

## 7. 実Start Gateとの対照実証（Reference。Production未接続）

`tools/reference-plugin-self-build/exact_transform_engine.go`の**参照用**実行関数に、物理プロセスの`cmd.Start`だけを包むHost所有の任意Callbackを受ける検証入口を追加。通常の実Process Engine実行は従来のままにする。

`inputprofile_v2_process_e2e_test.go`では、同じSB-1入力を使い、次の結果を区別する。

1. 有効時点で作成された`PreparedPackageInputs`は、失効後に旧来のStart入口へ渡すと起動可能（前章の未閉塞診断）。
2. 同じ入力に対して、実Start時点にHostが失効を拒否するCallbackを適用すれば、`cmd.Start`は呼ばれない。
3. Hostが明示的に許可を回復した後でのみ、同じ実v1 Process Engineが起動し、正規SB-1出力とGo比較対象の一致が得られる。

このCallbackは**試験用Hostの疑似判断**であり、真正なProduction HostのGrant元・失効の直列化・署名・Leaseを実装した証拠ではない。単一スレッドの順序試験では、読み取りとStartの並行失効に対する原子性も実証できない。

Linuxの製品Runtimeに存在する`internal/plugin/runtime.HostAdmittedPinnedLinuxProcess`と`LinuxHostStartAdmission`が、Host側の現在Authorityと`cmd.Start`を直列化する具体例である。しかし当該APIはPlugin Runtime向けのTarget固有実装であり、SB-1の変換Engineには接続されていない。一般化する場合は、Host側の既存Authority所有者が同じ世代／Plan／Scopeを束縛し、Build HostのStartへ適用する。**SB-1用の第二のAuthority・新Grant Storeは設けない。**

この段階のReference成功を、SB1-H09のProduction受入済みへ変更しない。実行開始後の失効・遠隔Host・Attestation・Artifact Acceptanceも引き続き独立した未完了条件とする。

## 8. Host世代とOS Startの並行競合（テスト限定の直列化Reference）

`tools/reference-plugin-self-build/inputprofile_host_start_generation_test.go`は、`sync.RWMutex`で単一の試験Hostの**現在選択、許可、世代、物理Start**を保護し、個別にビルドした旧v1 Process Engineで次を検証する。

- 有効な選択で入力を準備しても、`Start`前に失効した場合は旧世代を拒否する。
- **同じProfile・Source・Planに復旧しても新しい世代が必要**であり、古い入力束縛／世代をそのまま許可しない（ABA防止）。
- HostのStart側が読み取りロックを保持している間、並行する失効の書込みロックは完了できない。OS Start後はロックを解放し、子プロセスの終了までHostの失効を待たせない。
- 同じ旧世代による失効完了後の再Startは拒否する。開始済みプロセスの出力観測は、**失効後の新規Build受入**とは区別する。

これらは**テスト用Hostと局所実Process Engine**での順序実証である。真のBuild Hostが現在Source/PlanとProfileを解決している証明、分散Host・OS別Targetでの起動線形化、Host署名・Build Attempt・Attestation・Artifact Acceptanceにはならない。追加のIssuer／Grant Storeは導入していない。

製品の採用経路では、既存のHost所有世代／Authorityと物理開始機構を同じ線形化点に束縛し、対象Source・BuildPlan・Engine・実Artifactを固定する必要がある。なお、同時失効に対して「Startが先に成立した要求を後から強制停止する」意味は採用しておらず、実行中の取消しは別設計とする。**SB1-H09はProduction未達のまま。**

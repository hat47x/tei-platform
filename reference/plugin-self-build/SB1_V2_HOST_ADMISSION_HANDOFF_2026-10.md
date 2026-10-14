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

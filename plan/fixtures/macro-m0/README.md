# Macro M0：実CLIの標準操作・成果物を照合する限定Reference

- **対象**：[#2502](https://github.com/hat47x/tei-platform/issues/2502)（利用者マクロ）、[#2605](https://github.com/hat47x/tei-platform/issues/2605)（read-only Product Skill）、[Draft #2602](https://github.com/hat47x/tei-platform/pull/2602) の実証計画 M0。
- **状態**：独立検証の候補。製品のマクロ実装、受理済み業務意味、現在の実行権限、生成物の納品を表すものではない。
- **入力**：既存の公開合成 `reference/standard-crud/workspace`、Application `ledger`。参照Workspaceや`.tei`ファイルを編集しない。
- **実行環境**：Python 3標準ライブラリと、同じcheckoutでビルドしたTEIのGo CLIのみ。別Host/共通Runner/新Coreを追加しない。

## なぜこの試験が必要か

2026年11月2日に `tei operations` と `tei artifacts` がmainへ統合された。しかし、別々のコマンドで同じ意味の版を読み、未解決の操作を隠さず、Manifest/Findingsの存在と意味を正確にConsumerへ渡せるかは独立の検証である。単にCLI終了コード0や`validate.ok=true`だけではマクロが実行可能とは言えない。

同じ日に追加された既存の[Construction Context CLI受渡し検証](../construction-context/README.md)は`reference/workspace`の`model catalog / inspect / trace / validate`とSource移動を扱う。こちらは**別の既存`standard-crud`負例**を使い、`operations / artifacts`との接合だけを調べる。既存Scriptのコードや期待値を複製しない。

## 実行（再現手順）

~~~bash
python3 -B plan/fixtures/macro-m0/test_verify_cli.py

# 実際のTEIを使う場合。checkoutと版を固定して実施する。
go build -o /tmp/tei-macro-m0 ./cmd/tei
python3 -B plan/fixtures/macro-m0/verify_cli.py \
  --binary /tmp/tei-macro-m0 \
  --workspace reference/standard-crud/workspace \
  --expected-head "$(git rev-parse HEAD)"
~~~

実CLIの検証は `version -json`、`commands`、`validate -json`、`operations -application ledger`、`artifacts -application ledger` を実際に別プロセスで呼び出す。`artifacts -out`はローカルへファイルを書き込むため**M0では呼び出さない**。

## 何を確認するか

| 段階 | 確認する根拠 | その成功だけでは証明されないこと |
| --- | --- | --- |
| 実行物 | 実binaryの`vcsRevision`とexact checkout、変更状態 | 意味の受理・署名 |
| 発見 | `commands`は`selected-read-only-commands`。非掲載の`operations/artifacts`でも実dispatchを呼ぶ | 全Capabilityの網羅・権限付与 |
| 検証 | `validate.ok`、Warning/diagnostics、World/SourceIndexの照合値 | Operationの導出、現在の実行資格 |
| 操作面 | `operations.worldFingerprint == validate.worldFingerprint`、導出Rule/ID/Binding、負例の`Gaps` | 実Provider呼出しとCommit |
| 成果物 | `artifacts.inputRevision == operations.modelRevision`、Manifestの同版、Entries、Complete、Findings | Consumer必要Memberの網羅、物理配布、実Reader |
| 出力 | 結果件数・比較可否・検査済み範囲だけをJSONで報告する | Raw Sourceや診断本文・OS egressの安全性 |

`standard-crud`の期待する操作は5件、欠落や衝突は4件。Go側の`TestSurfaceWorkspaceStandardCRDFixture`と同じ公開合成Fixtureに由来する**制約された期待**であり、独立した業務判定基準の主張ではない。`create`と`update`が既存能力名に対応しても、現在のHost Grantがあるとは限らない。

別々のCLI呼出しのWorld/Revision照合値が一致しても、ファイル群を原子的に一回のSnapshotから読んだ保証はない。`sourceIndexFingerprint`と`modelRevision`を異なる定義の値であることを無視して直接比較しない。Source/業務対象/経路の鮮度が変わった時点では再資格化する。

## 証拠と残余

- `test_verify_cli.py`は**自作JSONの正負11件**で検査器の判別を試す。実Go/SQLite/PostgreSQL/OS環境の保証ではない。
- 専用GitHub Actions `.github/workflows/macro-m0-reference.yml` は実PR HEADのGo CLIをビルドし、上の5回のread-only呼出しを実行する。`tests-only`と`observed-macro-m0-cli`を別jobにして結果を混同しない。
- 観測結果は件数・真偽・状態だけをログに出す。公開FixtureのRaw Source、内部パス、診断message、実際の権限情報を表示しない。
- 次のM1は実ファイル・Reader byteの照合、M2はSkill/TAI、M3はOwner Action/実Grant/Commit、M4はSHO/EKIの保存・Body受領。**M0の合格をM1〜M4やP0の成功へ昇格させない**。
- Source移動や`NOT EXISTS`の完全性、Provider/Queryの実読取り範囲、現在のAuthorization/Disclosureは各Ownerの既存試験を利用し、必要な実証が残る。

**本書自身は実行結果を記録するものではない。** 実行済み証拠は対象HEADに対応するGitHub Actionsの各jobとログで示し、未実行は未実行と記載する。

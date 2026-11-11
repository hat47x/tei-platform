# REA外部Evidence受領の実CLI検証（real-CLI verifier）

対象Issue: [#3394](https://github.com/hat47x/tei-platform/issues/3394)
設計: [REA_EXTERNAL_EVIDENCE_INTAKE_2026-11.md](../../design/REA_EXTERNAL_EVIDENCE_INTAKE_2026-11.md)・[T1-A作業票](../../design/REA_T1A_EXTERNAL_OBSERVATION_ADMISSION_2026-11.md)

## 何をするか

`verify_real_rea_intake.py` は、外部逆解析ツール（REA）の**実CLI**を使って受領契約を検証する。合成のinert fixture（`index.js`は実行されると即座にthrowする）を解析し、bounded viewを投影し、Issueに記録された負例（受理しない`source.kind`・応答包装をEvidence本体として渡す）を実走し、その結果を`tei`バイナリで受領・検査・投影して突き合わせる。

**この検証は解析結果の正しさ・TEI意味の受理・SHO・Target・実DB観測の資格ではない。**

## 2つのモード（環境ができることだけを実行する）

| モード | 条件 | すること | 記録 |
|---|---|---|---|
| `real` | REAをinstallできる（npm到達） | 実解析・実view・負例を実走し、fresh bytesで受領・検査・投影を検証 | 捕捉bytesのdigest・各commandのexit |
| `fixture` | REAがない | `internal/externalanalysis/testdata/` の捕捉bytesで同じ契約を検証 | live partsは`notRun`として記録（fresh runとは表示しない） |

どちらのモードでも、工具が環境に無いことによる`tool-absent`は**whole receiptの注記**として扱う（bytesは完全、新規取得はできない）。

## 実行

```bash
go build -o /tmp/tei ./cmd/tei

# 実REAを呼べる環境（pin版をisolated prefixへinstallする）
python3 -B plan/fixtures/rea-external-evidence/verify_real_rea_intake.py \
    --binary /tmp/tei --version 6.8.0 --report /tmp/intake-real.json

# 既存のinstallを再利用する場合
python3 -B plan/fixtures/rea-external-evidence/verify_real_rea_intake.py \
    --binary /tmp/tei --version 6.8.0 --tool-prefix /path/to/rea-agents-prefix

# 実REAを呼べない環境（保存fixtureのみ）
python3 -B plan/fixtures/rea-external-evidence/verify_real_rea_intake.py \
    --binary /tmp/tei --fixtures-only --report /tmp/intake-fixture.json
```

`--report` のJSONが実行記録である。`mode`・`checks`・`captured`（digest）・`notRun`を読む。

## 検証していること

1. 受領: `state` がwhole（`complete`/`tool-absent`）で、Evidence ID・**受領bytesのdigest**・`promotion`が全false。
2. 派生入力: `source.kind=inline`にEvidence recordをembedし、**応答包装をembedしない**。派生请求は受領Evidenceに対してvalidateできる。
3. 投影: 1 Observation＋1 directed Relationを、Producer自身のnode/relation identityで受領し、`authorityLimits`が非昇格を表明する。
4. 配送検査: 捕捉/新鮮なview配送が各版のframingでwhole。
5. 負例: 工具が受理しない`source.kind`と、応答包装をEvidence本体として渡す請求を、**工具が**拒否する（intakeが代わりに拒否するのではない）。

## 版の再資格化（T0-06相当）

新しいREA版が現れたとき、`internal/externalanalysis` の`observedSourceKinds`/`observedFraming`は**実走で**更新する。手順:

1. `npm install --prefix <new-prefix> --ignore-scripts --save-exact rea-agents@<version>`。
2. 上記の`real`モードを`--version <version>`で走らせる。**未知の版はINVALID_ARGUMENTで拒否される**（受諾値・framingを推測しないため）。
3. 工具の拒否`details.issues`から`source.kind`受諾値を読み、`inspect-analysis-view … --format jsonl`の出力形状（終端metadataの有無・records数）を確認する。
4. `internal/externalanalysis/request.go` の`observedSourceKinds`と`internal/externalanalysis/view.go` の`observedFraming`に版を追加し、**実捕捉bytesを`testdata/rea-<version>/`へ保存**して`manifest.json`に行を追記する（digest・exit・Node/OS版）。
5. `go test ./internal/externalanalysis/ ./internal/operations/` を実行し、`cmd/tei/cli_contract.go` の`provider-version`受諾値と`cli_discovery_modes_test.go` のclosed setを更新する。
6. 既存版の捕捉bytesは**書き換えない**（歴史Evidenceは証拠として保持する）。

未知の版に対して「たぶん動く」はできない。表に載るまで、その版のbytesは`mismatch`/`INVALID_ARGUMENT`として扱われる。

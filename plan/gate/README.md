# 段階判定

段階判定に必要な条件、検証根拠の取得手順、判定記録をまとめます。各文書は対象と版を固定して扱います。現在の製品状況は[実装状況](../../docs/status.md)、全体の着手順は[開発計画](../README.md)を参照してください。

## 文書の役割

| 文書 | 役割 |
| --- | --- |
| `*_EVIDENCE_ACQUISITION.md` | 判定に必要な検証根拠と、それが証明する範囲 |
| `*_EVIDENCE_CONTRACT.md` | 検証根拠の区分と受入条件 |
| `*_EVIDENCE_RUNBOOK.md` | 検証根拠を取得する具体的な手順 |
| `*_READINESS_AUDIT.md` | 特定の対象・版に対する判定記録 |
| `*_HANDOVER_*.md` | 段階間で引き継ぐ基線と条件 |
| `*_PREFLIGHT_*.md` / `*_START_PACKET_*.md` | 次の実証へ進むための事前確認・開始資料 |

文書名だけで状態を判断せず、各本文に記した対象、版、根拠を確認してください。判定記録は固定した対象の結果を保管するもので、現在の`main`の状態を示すものではありません。

## 段階A

- [段階Aの判定記録](GATE_A_READINESS_AUDIT.md)

## 段階B

- [検証根拠の取得条件](GATE_B_EVIDENCE_ACQUISITION.md)
- [検証根拠の取得手順](GATE_B_EVIDENCE_RUNBOOK.md)
- [判定記録](GATE_B_READINESS_AUDIT.md)
- [段階間の引継ぎ契約](GATE_B_TO_C_HANDOVER_CONTRACT.md)
- [固定した引継ぎ基線](GATE_B_TO_C_HANDOVER_BASELINE_B.md)

## 段階C

- [検証根拠の契約](GATE_C_EVIDENCE_CONTRACT.md)
- [最初のウェブ実行先の検証根拠を取得する手順](GATE_C_FIRST_WEB_TARGET_EVIDENCE_RUNBOOK.md)
- [判定記録](GATE_C_READINESS_AUDIT.md)
- [第二のアプリケーションの事前確認](GATE_C_SECOND_APPLICATION_ENTRY_PREFLIGHT_2026-09.md)
- [第二のアプリケーションの開始資料](GATE_C_SECOND_APPLICATION_START_PACKET_2026-09.md)

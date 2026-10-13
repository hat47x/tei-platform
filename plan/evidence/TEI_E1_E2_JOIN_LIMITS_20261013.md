# #1119：保全済みE1・E2a-1の横断証拠照合（2026-10-13）

## 目的

合成Web E1の成功と、実PostgreSQL E2a-1の成功を単純に足して、**同一Accepted Source revisionから製品E2が成立した**と誤報することを防ぐ。両者は異なるGitHub仮統合SHA、Application、実行経路、Authority、Observationに属する。

| 証拠 | 原本保全先 | 対象と限界 |
| --- | --- | --- |
| [E1 Actions #38267656696](https://github.com/hat47x/tei-platform/actions/runs/38267656696) | [合成8件](TEI_E1_SYNTHETIC_ARCHIVE_README_20261013.md) | order_entry、scratch R0→R1、合成ProviderのWeb HTTP。R0/R2不一致は外部ハーネスで検知。仮統合SHA 466b1d10080460390d46b3b342d8bc9c19e09208 |
| [E2a-1 Actions #38311557882](https://github.com/hat47x/tei-platform/actions/runs/38311557882) | [実PostgreSQL原本](TEI_E2A1_REAL_POSTGRES_ARCHIVE_20261013.md) | request_log / intake.request、手書きSQLと実DBのwrite、同一プロセスの別poolから業務行read。仮統合SHA a2679f0d8d52b0562543464cbd32f12c2ebd72c3 |

検査器 tools/verify-critical-path-e1-e2-join.py は既存の2原本SHA検査器を再使用し、次の8条件がこの歴史的な二資料では満たされないことを機械的に報告する。

1. 同じAccepted Source revisionとRuntime自身による検証
2. ApplicationとBusiness Data subjectの同一性
3. Web CommandからPostgreSQL writeへの実行ID付き接続
4. R0および意味同値R2のWeb dispatch前拒否
5. Current Host write Authorityのcommit時照合
6. 別invocationによる独立read
7. 正当な人間によるSource Acceptance
8. Data-firstから派生した生成CRUD（手書きSQL・overlayとは異なる）

## 実行

~~~sh
# Archiveとして有効ならexit 0。ただし Product Gate はUNQUALIFIED。
python3 -B tools/verify-critical-path-e1-e2-join.py

# 製品資格を要求すると必ずexit 3でfail-closed。
python3 -B tools/verify-critical-path-e1-e2-join.py --require-g2

# 改ざん、根拠のないSource版、架空の権限などの回帰試験。
python3 -B tools/test-critical-path-e1-e2-join.py
~~~

historicalEvidenceOnly=true、productGate=UNQUALIFIED、canSatisfyG2=false を固定する。全8条件を偽装したメモリ上のJSONでもこの保存済み証拠専用器からPASSを出さない。対象は**この2件の保存史料に限り**、新しい正規E2証拠が出た場合は独立したTrusted Producer/Consumer判定で資格を決める。

## Ownerへの受渡し

- #1124→#1121：Studio Draft #1273の正式main統合、続いて実Web自身の受理済みR1/R0/R2拘束。#1119はSource ReaderとWeb runtimeを変更しない。
- #1127：既存Draft #1399の実PostgreSQL、#1406のPersistence宣言を、同じAccepted Source R1のorder_entry・通常Web Command・Host Authority・別invocation readへ接続。S02/S05/N11の負例は独立に保存。
- #1163：一回の正式な実行に対し、受理版、Source bytes、Host主体、Command、DB実行、別invocation Observationを相関する。2件の後日ハッシュ照合だけで製品E2とは称しない。

この検査は証拠の誤結合防止であり、G1〜G4の合格証明、新たな実行Authority、第二のメタモデルではない。

# Reference Model Transfer — source-derived Evidence

このdirectoryは[ISSUE-000078](../../plan/issue/000078-reference-model-transfer-gate.md)のRMT-0 / RMT-1 Evidenceを保持します。

原Reference Modelの内容を公開Repositoryへ複製する場所ではありません。Repository外のsource snapshotについて、exact fingerprint、集計、転写判断、loss / unresolvedを再現できる最小情報だけを保持します。

初回benchmarkはX-TEA / XEAD系「CONCEPTWARE／受注生産」です。

## 現在のartifact

- `source-baseline.json` — source code revision、model / manual artifact digest、公開境界
- `source-inventory.json` — model構造のaggregate inventory。source constructの母集団
- `initial-pressure-register.json` — 旧Runtimeでは暗黙でも成立した点をTEIへ移す際の初期pressure
- `transcription-contract-v0.json` — RMT-2のconstruct分類、Plugin / Extension責務、source locator停止線を定めるcandidate contract
- `loss-ledger.json` — 転写時に推測で埋めてはいけないloss / unresolved / unsupportedの初期台帳

これらはRMT-0 / RMT-1の開始Evidenceです。TEIへのtranscription完了、original Runtimeとのbehavior equivalence、Reference Modelの再配布許諾を意味しません。

## 更新原則

- source snapshotが変わったらdigestを変え、既存Evidenceへ上書きして同一source扱いしない
- aggregate countだけで意味の転写完了を判定しない
- source nodeを落とす場合は`loss-ledger`へ明示する
- model proseから抽出した意味をAccepted Canonicalへ自動昇格しない
- original contentの転載が必要な場合は利用条件を先に確認する

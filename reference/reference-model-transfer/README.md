# Reference Model Transfer — source-derived Evidence

このdirectoryは[ISSUE-000078](../../plan/issue/000078-reference-model-transfer-gate.md)のRMT-0 / RMT-1 Evidenceを保持します。

原Reference Modelの内容を公開Repositoryへ複製する場所ではありません。Repository外のsource snapshotについて、exact fingerprint、集計、転写判断、loss / unresolvedを再現できる最小情報だけを保持します。

初回benchmarkはX-TEA / XEAD系「CONCEPTWARE／受注生産」です。

## 現在のartifact

- `source-baseline.json` — source code revision、model / manual artifact digest、公開境界
- `source-inventory.json` — model構造のaggregate inventory。source constructの母集団
- `initial-pressure-register.json` — 旧Runtimeでは暗黙でも成立した点をTEIへ移す際の初期pressure
- `transcription-contract-v0.json` — RMT-2のconstruct分類、Plugin / Extension責務、source locator停止線を定めるcandidate contract
- `transcription-map-v0.json` — full-source inventoryをconstruct kind単位でclassification / Extensionへ割り当てたaggregate candidate map
- `cross-view-accounting-v0.json` — Data / Function-Interaction / Business / Xeaf correspondenceを原モデル本文なしで集約したRMT-2 cross-view Evidence
- `loss-ledger.json` — 転写時に推測で埋めてはいけないloss / unresolved / unsupportedの初期台帳

RMT-0 / RMT-1の開始Evidenceに加え、`transcription-map-v0.json`はRMT-2のaggregate candidate Evidenceです。TEIへのtranscription完了、original Runtimeとのbehavior equivalence、Reference Modelの再配布許諾を意味しません。

RMT-2のsource-faithful adapter候補は `internal/xeadsource` に置きます。そこではraw source treeに加え、legacy key / Relationship key reference / declared `KeyFields`をsource factとして機械可読化できます。ただし、それらはbusiness identifier、candidate key、functional dependency、Canonical identity、physical key authorityではありません。AI / Studio / CLIが利用する場合も、candidate classificationとauthority limitを同時に扱います。

relational factの存在はlowering可能性の証明にも使いません。内部Inspectionではfull-setの`relationalResolutionStatus`と、fact単位の`resolved / unresolved / unvalidated`を分離します。局所`resolved`は構造readinessに限り、Canonical meaningやbusiness identityのAcceptanceではありません。公開inspection report v2ではraw locatorやlegacy IDを複製せず、fact種別ごとのreadiness件数だけを保持します。

## 更新原則

- source snapshotが変わったらdigestを変え、既存Evidenceへ上書きして同一source扱いしない
- aggregate countだけで意味の転写完了を判定しない
- source nodeを落とす場合は`loss-ledger`へ明示する
- model proseから抽出した意味をAccepted Canonicalへ自動昇格しない
- original contentの転載が必要な場合は利用条件を先に確認する

## Source Adapter Reference

- `../../plan/design/REFERENCE_MODEL_SOURCE_ADAPTER_V0.md` — Repository外のObserved Documentとpublic Publication Receiptを分ける#950の設計境界
- `../../tools/reference-xead-source-adapter/` — Xead / Xeaf full-source accountingとsource fact projectionのReference実装

## Source profile boundary

`transcription-contract-v0.json` schema v2は、Xead / Xeafの対応source profileを明示する。現在のReferenceは両方ともroot `System`、`FormatVersion 1.2`を宣言する。

raw `Observe`は未対応versionでもローカルEvidenceとして読み取りを試みるが、classification / coverage / local candidate mappingは、contract-declared profileへ一致しなければfail closedとする。

`transcription-map-v0.json`とR0 coverageは旧contract blobに結び付いた履歴Evidenceである。current contractへ自動的に再帰属させず、source profile Gateを含む再実行前にはcurrent coverageを主張しない。

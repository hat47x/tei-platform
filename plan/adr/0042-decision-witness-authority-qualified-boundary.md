# ADR-0042: Decision witnessのactor・authority状態は、policy検証configurationによるexact bindingでのみ解決し、受理とは混同しない

- 状態: 提案中
- 日付: 2026-10-28
- 関連ADR: ADR-0022 / ADR-0035 / ADR-0041
- 関連Issue: #974 / #267 / #2017
- 関連文書: [`plan/research/AUTHORING_DECISION_AUTHORITY_EVIDENCE_GATE_2026-10.md`](../research/AUTHORING_DECISION_AUTHORITY_EVIDENCE_GATE_2026-10.md)（Evidence Gate E1〜E6の設計正本）
- 関連実装: [`tools/reference-shared-acceptance/README.md`](../../tools/reference-shared-acceptance/README.md)・[`tools/reference-shared-acceptance/witness.go`](../../tools/reference-shared-acceptance/witness.go)・[`tools/internal/opensshwitness/README.md`](../../tools/internal/opensshwitness/README.md)

## 背景

ADR-0022は、AIによる変更提案と意味の正本の受理境界を分けた。その結果として記録されたDecisionの`decisionActorStatus` / `decisionAuthorityStatus`は、誰が決定し、どの権限で決定したかのEvidenceが無い間は`unresolved`のままとされてきた（Evidence Gate 2026-10 §7）。

#974でOpenSSH（`ssh-keygen -Y sign / verify`）のproviderを、signed authoring witnessとして型付きlibraryに抽出し（#2310）、#2311でShared Acceptance側のDecision経路へexact witness bindingを実装した。これにより、operator管理のpolicy・pinned verify-time・任意のrevocation listというconfigurationが揃った場合に限り、provider検証を通じてDecisionのactor・authority状態を解決できる経路がReference単位で存在するようになった。

この状態遷移の規則を、Decision recordのschemaやprovider実装の詳細から独立した設計判断として記録しておく必要がある。将来identity providerを交換する（Evidence Gate §9の既定）ときにも、この規則は変わらないはずである。

## 判断

### 1. binding対象は「proposal全体」とし、Decision record本文から決定論的に再導出する

witness messageは、Decision record本文（candidate fingerprint、review binding fingerprint、基準model revision、決定semantics、決定理由のdigest、source scopeのdigest）だけから再導出する。namespaceも同じくrecord本文から決定論的に導出する。witness記録がnamespaceやmessage本文を記名して持つことを避けることで、recordの改変・飲み替えがdigest照合とprovider検証の両方で検知される。binding先はrecord本文の全体digestとし、一部のfieldだけに結び付けない。

### 2. policy・verify-time・revocationは、candidateやworkspaceの内容から補完しない

allowed-signers policy、検証時刻、失効listは、CLI flagsというoperator管理のboundaryだけから取得する。candidate、workspace、Decision record、Review Projectionからpolicyを推論してはならない。検証configurationが揃わなければ、provider検証を実行しない。これにより、witness検証がpolicy sourceの差し替えをworkspace側から起こせない。

### 3. 状態の解決は、providerの`authority-qualified`だけが行う

構造的なbinding（本文digest・message digest・signature digestの一致）が成立しても、状態は`recorded`に留まる。actor・authorityは解決しない。providerがpolicy・namespace・pinned時刻・revocationを含めて`authority-qualified`と出した場合に限り、`decisionActorStatus=identity-resolved`、`decisionAuthorityStatus=authority-qualified`に上がる。provider不在・verify-time非対応・policy不一致・満了・改変は、それぞれ`WITNESS_*`diagnosticに理由を残し、状態を未解決のまま保留する。自己申告actorへのfallbackが存在しないことが、この規則の中核である。

### 4. `authority-qualified`は受理・書込み・承認を意味しない

providerのqualified判定は、`Decision applied`でも`Review matched`でも`source committed`でもない（Evidence Gate §6）。witness ledgerはCanonical sourceの外に置かれ、削除すればclaimが消え、model revisionもaccepted meaningも変わらない。Runtime assertionのR3AN / R3AOのprincipal・scopeをauthoring Decision側へ流用しない。

### 5. 競合はdowngradeし、winnerを決めない

同一candidateに対しaccepted / rejected双方のwitnessがqualifiedになった場合、winnerを決められないので、両方を`authority-conflicted`へ落とす。複数principalが同一namespaceでqualifiedになること自体はpolicyの範囲であり、conflictにしない。異なるauthority Evidenceの競合を人手で解除する手続きは、このADRの範囲外とする（未決）。

### 6. review bindingを持たないDecisionはwitness対象外とする

E6のexact bindingにはreview Projection identityが必須であるため、review bindingを持たないDecision（current Referenceのreject）はwitness対象にしない。rejectをwitness可能にする拡張は、review bindingを持つrejectの経路（coverage付き）として実装する。

## 代替案

- **`origin.actor`やCLI実行者をDecision actorへ採用する**: 自己申告であり、Evidence Gate §3のdecision locusとactorの分離に反するため採用しない。
- **review fingerprint一致をhuman witnessへ昇格する**: 「読んだこと」ではなく「保存内容の一致」しか示さないため採用しない。
- **workspace・candidateからpolicyを推論する**: policy source差替えをworkspace側から可能にするため採用しない。
- **pinned verify-time無しで検証する**: 新鮮性が固定できず、満了・失効の判定が決定論的にならないため採用しない。
- **provider不在時に自己申告へfallbackする**: 静かにEvidence高さを偽るため採用しない。

## 影響

- Decision actorの人間による判定は、production identity providerが選ばれるまで解決しない。このReferenceのpolicy・鍵はtest供給のharnessであり、production identity providerではない。
- #974のprovider SPI採否判定は、この実装とは別に、E2〜E6相当の全条件が成立するかを確認して判断する。#267の達成判定も同様に未実施である。
- 将来のidentity provider交換では、provider側の検証出力を`identity-resolved` / `authority-qualified`の分類へ写像するadapterだけを差し替える。message schema・namespace規則・ledger配置をprovider固有にしない。

## 未決

- production identity providerの選択（Studio authenticated user context、SCM authority等。Evidence Gate §9）。
- 鍵・policyの配布・失効運用を行う主体と手順。
- `authority-conflicted`を人手で解除する手続きと、その操作自体のEvidence化。
- witness ledgerの保持期間・監査方針。

## 検証

- `tools/reference-shared-acceptance`のwitness試験（`witness_test.go`）: qualified exact binding、principal不一致、改変messageのstructural拒絶、accept→reject流用の拒絶、accept / reject競合のdowngrade、witness削除によるclaim消失（meaning不変）、verify configuration不備のusage errorを、実providerでPASS。
- `tools/internal/opensshwitness`のprovider試験: principal不一致、accept→reject流用、改変、満了・未開始・KRL失効、provider不在、verify-time非対応を実providerで確認（OpenSSH 9.5p2 / 9.6p1）。
- verify-time未対応OpenSSH（8.9p1）では、provider試験はskip（passではなく）であり、capability欠如を署名前に検知して停止する（`reference-authoring-witness-openssh`のpressure-test）。

このADRは提案中であり、採用判断は人間の受理境界に残す。効果が確認できない場合は、witness分類を参照実装の範囲へ戻し、`decisionActorStatus` / `decisionAuthorityStatus`を常時`unresolved`とする挙動へ戻すことで撤回できる（Decision record自体は`unresolved`実装の戻し方に対して前方・後方互換である）。

# E4f: 外部Semantic選択を意味変更として扱う参照契約

- 対象: [Shared Acceptance #267](https://github.com/hat47x/tei-platform/issues/267)、[外部業務知識 #1570](https://github.com/hat47x/tei-platform/issues/1570)
- 先行: [現在の案件受理の照合 E4e](CURRENT_PROJECT_MEANING_REFERENCE.md)
- 実装: `external_selection_change.go`、`external_selection_change_test.go`
- 状態: **読取り専用の候補／判断評価Reference。案件の採用を保存する正式なHost機能ではない。**

## なぜEvidenceではなく選択変更か

既存 `tools/reference-shared-acceptance/candidate.go` の意味fingerprintは
`baseModelRevision` と `delta` に由来する。
`Evidence`、`Origin`、`Intent` はそのfingerprintを変えない。
外部業務知識を案件で採用する対象（論理Scope、型付きNamespace ID、Alias、
`semanticDigest`）をEvidenceへ添えるだけでは、
**採用対象の意味変更を追跡できない**。

E4fは、Shared Acceptanceに後で組み込むための限定的な試作として、
`ExternalSelectionChange` の `before` と `after`、
`applicationId`、`baseModelRevision`、`baseAuthorityRevision` を
一つの指紋へ含める。順序を正規化し、同じ選択は同じ指紋となる。
型・Scope・Alias・Namespace ID・Digestの変更は指紋を変える。
採用解除（`after: []`）も意味を持つ変更候補として扱う。

現段階では**Path Workspace Import限定**。
`modelref.Ref.Version` を空とし、`Subject.Kind=semantic-namespace`、
`semanticDigest=sha256:<64桁小文字hex>` を要する。
Package/CAS版、Provider署名、Artifact Digestをこの指紋に流用しない。

## 検査と判断の境界

`PreviewExternalSelectionChange` は呼出側が渡す現在値について、
Source版・Hostの判断改訂・`before`と現在の外部選択の一致を検査する。
入力の判断記録やSource版を**真正と証明するものではない**。
実Hostでは必ず信頼済み現在Reader、Source Index、
Providerの現在の意味内容・取得権限を別途検証する。

`EvaluateExternalSelectionDecision` は判断の直前に同じ検査を行い、
表示済み指紋と現在計算した指紋が一致しなければ拒否する。
`reject` は新しい選択を一切提案しない。
`accept` の出力は、Hostへ渡すべき**選択変更案**に過ぎない。
この関数が返した`accept`という文字列は人的受理・権限・Commitではない。

これらの関数はファイル、Source、Hostレコードを書き換えない。
本番では既存Shared AcceptanceのWriterとTransactionへ接続し、
**一つのHost-owned判断**としてSourceと採用選択を記録し、
改訂や取消しもHost側で管理する。新しいRegistry、第二のLock、
Accepted World、外部業務知識受理用DBは導入しない。

## 負例で確認する内容

- 選択Digest、論理ID、別名、採用の追加・削除が意味差として扱われる
- 同じ選択集合の列挙順を変えても指紋が変わらない
- Managed Source改訂、Host Authority改訂、選択前提のずれを拒否する
- 不正Digest、重複Alias・Namespace、型・Scope・Version不一致を拒否する
- 未確認・不一致のレビュー指紋、未知の判断を拒否する
- Preview、Reject、合成Accept評価はHostの現在値を書き換えない
- 採用解除の候補を作れても、Hostの実際の取消しや失効処理は生じない

対象試験: `go test -count=1 ./reference/semantic-composition-preflight`

## 次の段階

1. Shared Acceptanceの変更候補へ外部選択Deltaを統合し、Source変更と選択変更の
   同時判断、差分レビュー、同一RevisionでのCAS/Commitを実証する。
2. Hostの改訂・取消し付きCurrent ReaderをE4e
   `CurrentProjectMeaningReader`へ接続する。
3. Providerの署名／配布権限、Path以外の取得元、Runtime Authorityは別境界で検証する。

合成試験は正式な受理や本番Hostの実証ではない。

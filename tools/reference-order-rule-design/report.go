package orderruledesign

import "encoding/json"

func jsonUnmarshal(raw []byte, v any) error { return json.Unmarshal(raw, v) }

// Gap is one platform gap the run observed.
type Gap struct {
	Requirement string `json:"requirement"`
	Area        string `json:"area"`
	Gap         string `json:"gap"`
	SampleSide  string `json:"sampleSide"`
}

// Gaps are the platform gaps the run observed.
func Gaps() []Gap {
	return []Gap{
		{"固有", "検証根拠", "出所と出所状態が、審査の入力(spec-reviewの入力は知識文脈、手順照合、共通受理の事前評価)と受理境界まで一本につながらない。共通受理の候補はtranscription evidenceを持てるが、出所状態(結び付いた、表だけ、対象なし)と未解決の件数を運ぶ欄がない",
			"出所状態と未解決の件数を添付(Attachment)として候補の隣に置き、審査パケットと署名の理由の両方に同じ値で写す。値の一致は通しの検証器(VerifyThrough)が各工程の記録で照合する。共通受理の意味の照合値には入らない"},
		{"固有", "プラグイン", "旧資料由来の未解決(契約が扱えない記述)の件数が、候補と審査へ同じ形で残る契約がない。source adapterは未解決を件数で出すが、候補の入力にはならない",
			"未解決を種別ごとの件数(Unresolved)にして各工程の記録へ読み戻して載せる。対象の節点がなければ検証根拠を作らず、対象なしとして運ぶ"},
		{"固有", "ツール", "確認器が自分の対象外を出力に示さない。business-dfdは受注業務の手順を描き、purpose-preflightの固定入力は承認規則を扱うが、どちらも単価規則を見ていないことを出力しない。subsystem-cudは空の事実で未解決を返すだけである",
			"サンプルが出力の内容(対象の識別子への言及、診断の符号)から、対象にできた/対象外/入力不足の三区分を決めて一覧にする。区分は確認器の宣言ではなくサンプルの判定である。「指摘なし」だけの一覧と、対象外を欠く一覧は失敗とする"},
		{"固有", "検証根拠", "認識状態(確認の鮮度)が審査の入力に接続されていない。reference-spec-reviewは基準版の不一致だけを古い前提として見る",
			"署名の前に、基準版、main、資料の照合値を再計算して工程4・5の記録と比べるサンプル側の前検査(Preflight)を置く。認識状態の参照実装は呼ばない(範囲外)"},
		{"P-WF-01", "コア", "署名による決定者の確認が共通受理の決定に接続されていない。witness-decisionは受理した決定記録(reviewBindingあり)だけを署名の対象にし、受理前の候補への署名は呼べない",
			"tools/internal/opensshwitnessのPayloadを直接使い、候補の意味の照合値、審査の照合値、基準版、理由の照合値、範囲の照合値を署名文面に入れる。決定の値accept(受理前の提出への署名)は受理の記録ではない"},
		{"固有", "ツール", "共通受理の経路と直接編集の経路が同じ意味に至ることの比較記録がない。資格確認の受領記録(qualify.sh)は全体の健全性検査を含み、変更範囲に応じた検証を選ぶ統合条件と整理が要る",
			"工程4で、共通受理のpreviewが示す版と、同じ変更を原文へ直接書いた複製の版を比べて記録する(観測であり必須条件にしない)。資格確認の受領記録は呼ばず、サンプルの受領記録(Receipt)を別に置く"},
		{"P-MIX-01", "ツール", "reference-change-assuranceの影響範囲は実行先ごとの派生を出力しない。reference-procedure-expansionは実現先の束縛を並べるが、どの規則の版で派生したかを持たない",
			"画面・サーバ・一括処理の三つの実行先と派生した版をサンプルの宣言(Derivation)として与え、派生していない実行先を再確認の範囲に残す。派生物の生成と判定の突合は範囲外"},
		{"P-WF-02", "コア", "実行中の案件の版を、改訂の影響として設計側へ返す経路がない",
			"旧版R0で開始した案件の一覧をサンプルの入力として影響範囲へ並べ、扱い(旧版のまま完了するか、移行するか)が決まるまで署名しない。移行そのものは行わない"},
	}
}

// Limits states what the run is not evidence for.
func Limits() []string {
	return []string{
		"受理: 共通受理へは提出して止め、accept・rejectは呼ばない。試験受理は使い捨ての複製だけで、状態はtrial-acceptance-on-disposable-copyである",
		"実際の旧資料: 資料はサンプルが書いた合成の最小XMLで、契約は合成の不完全な例である。実際のXeadの資料への適合を示さない",
		"署名の権限: 鍵と許可する署名者は試験用で、決定者の判断権限や組織の承認を示さない。時刻は固定の論理時刻で、壁時計を読まない",
		"派生物と実行中の案件: 三つの実行先の派生と実行中の案件は、サンプルの宣言と入力である。派生物の生成、判定の突合、案件の移行は行わない",
		"資格確認: 受領記録はサンプル独自で、qualify.shの全体の健全性検査を含む受領記録ではない",
		"実環境: 実際の提供元、データベース、本番環境は使わない。Dockerも使わない",
	}
}

// NotVerified lists what the run could not verify.
func NotVerified() []string {
	return []string{
		"署名文面の値がorder-workflowが共通受理へ渡す値と一致すること: order-workflowはmainに未統合で、受理は未接続である。共通受理(tools/reference-shared-acceptance)の再計算との一致だけを確かめた",
		"出所状態の三値のうち、実際の旧設計資料で得られる値: 合成資料では三値すべてを作れるが、実資料の値は未確認である",
	}
}

// AssetCalls records, per stage, whether the existing asset was called
// unchanged and how.
func AssetCalls() []AssetCall {
	return []AssetCall{
		{"1", "tools/reference-xead-source-adapter", "CLIで呼べる", "go buildしたCLIを合成資料と共有の合成契約の複製に実行し、受領記録、未解決、節点ごとのtranscription evidence(-candidate-evidence-node)を読む"},
		{"1", "tools/reference-xead-cross-view", "CLIで呼べる", "go buildしたCLIの集計(公開用)を読み、業務作業から機能と表への経路があるかを出所状態の判定に使う"},
		{"1", "reference/xead-source-transcription-contract.synthetic.json", "固定入力として複製して使う", "書き換えずにバイト列を複製し照合値を取る"},
		{"2", "tools/reference-business-dfd", "CLIで呼べる", "R0とR1の複製を-workspaceに渡す。受注業務の手順を描き、単価規則は対象外として扱われる"},
		{"2", "tools/reference-subsystem-cud", "CLIで呼べる(入力不足)", "R0のWorldと空の候補の事実を渡す。受理済みの所有・機能の入出力の事実がなく、EMPTY_CANDIDATE_SCOPEの未解決になる"},
		{"2", "reference/workspace、cmd/tei", "複製して呼べる", "作業領域を複製し、tei world、tei knowledge-context、共通受理のrevisionを読む"},
		{"3", "reference/business-spec-templates", "Node.jsで呼べる", "node --testの8件と、proposeAdoptionの提案(受理なし、未解決の質問が残る)を呼ぶ。入口は変更しない"},
		{"3", "reference/studio-authoring", "Node.jsで呼べる", "cli.mjs proposeにtable・add-validationの編集とapplicationUseを渡し、共通受理の候補を得る。内部でgo runを使うため、PATHにGoの実行体を置く"},
		{"3", "tools/reference-shared-acceptance", "CLIで呼べる", "attach-rmt-transcriptionで出所のevidenceを付け、previewで意味の照合値と受理可否を読む"},
		{"4", "tools/reference-tool-neutral-development/run.sh", "呼べる", "清潔な作業ツリーで変更せずに実行し、直接編集の経路の結果を読む。約1.5分かかる"},
		{"4", "tools/reference-tool-neutral-development/qualify.sh、integration-preflight.sh", "呼ばない", "qualify.shはrepository-health-checkの全項目を要し、AGENTS.mdは一律には実行しないとする。preflightはその受領記録を入力にする。サンプル独自の受領記録を使い、資格確認は未発行と記録する"},
		{"5", "tools/reference-change-assurance", "CLIで呼べる", "R0とR1の複製を-before-workspace、-after-workspaceに渡し、固定入力-inputの再利用候補も呼ぶ。実行先ごとの派生は出力に無い"},
		{"5", "tools/reference-procedure-expansion", "ライブラリとして呼べる", "Compareで条件が決まった前後の実現先の準備状態を比べる。画面は常に利用可能で、関数の束縛だけが変わる"},
		{"6", "tools/internal/opensshwitness", "ライブラリとして呼べる", "Payload、Sign、Verifyを呼ぶ。OpenSSHはverify-time対応の版が必要である"},
		{"6", "tools/reference-authoring-witness-openssh/pressure-test.sh", "呼べる", "7項目の探針を変更せずに実行する(固定のダミー文面)。サンプルの署名は別にopensshwitnessで行う"},
		{"6", "tools/reference-authoring-witness-openssh witness-decision(共通受理)", "呼べない", "受理した決定記録だけを署名の対象にする。このサンプルは受理しないため呼ばない"},
		{"6", "tools/reference-spec-review", "CLIで呼べる", "R1の知識文脈とpreviewを渡す。cross-viewが欠けるため入力不足が残る"},
		{"6", "tools/reference-purpose-preflight", "CLIで呼べる", "R1・R0の作業領域の入口と、承認規則の固定入力を呼ぶ"},
		{"6", "order-workflowの共通受理", "呼べない", "mainに未統合。SubmissionBundleを受ける提出先(Submitter)を差し替え点とし、受理は未接続と記録する"},
		{"6", "tools/reference-shared-acceptance", "preview・reviewだけ呼ぶ", "使い捨ての基準版の複製へ提出し、再計算した照合値と署名の値を比べる。accept、rejectは呼ばない"},
	}
}

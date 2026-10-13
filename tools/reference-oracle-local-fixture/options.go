package main

import (
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"
)

// confirmFlagは、管理者接続で専用ユーザーと検証用の表を作成・削除することへの明示的な確認である。
// 接続先は別用途の既存Oracleである可能性があるため、このflagがなければ接続しない。
const confirmFlag = "confirm-shared-database-fixture"

// evidenceOutNameMarkは、検証用データ由来の検証根拠であることがファイル名で分かるように、出力名へ必須とする文字列である。
const evidenceOutNameMark = "oracle-local-fixture"

type options struct {
	Workspace        string
	EvidenceOut      string
	CleanupRun       string
	Timeout          time.Duration
	Confirmed        bool
	SkipTimeoutCheck bool
}

// parseOptionsは、コマンド行を解析して確認flagなどの必須条件を検査する。DB接続より前に呼ぶ。
func parseOptions(args []string, output io.Writer) (options, error) {
	set := flag.NewFlagSet("reference-oracle-local-fixture", flag.ContinueOnError)
	set.SetOutput(output)
	var opts options
	set.StringVar(&opts.Workspace, "workspace", "reference/workspace", "契約問合せを解決する作業領域のディレクトリ")
	set.StringVar(&opts.EvidenceOut, "evidence-out", "", "受理した検証根拠JSONの出力先（Repository外を指定し、ファイル名に"+evidenceOutNameMark+"を含めること）")
	set.StringVar(&opts.CleanupRun, "cleanup", "", "実行IDを指定すると、その実行が作成した専用ユーザーだけを削除する（中断後の後始末用）")
	set.DurationVar(&opts.Timeout, "timeout", 10*time.Minute, "全体のタイムアウト")
	set.BoolVar(&opts.Confirmed, confirmFlag, false, "接続先が別用途の既存Oracleでも、このtoolが専用ユーザーと検証用の表を一時的に作成・削除してよいことを確認する（必須）")
	set.BoolVar(&opts.SkipTimeoutCheck, "skip-timeout-check", false, "待機中の問合せを時間切れと取消しで中断する確認を実施しない（この確認は共有データベースへ1〜2秒のCPU負荷をかける）")
	if err := set.Parse(args); err != nil {
		return options{}, err
	}
	if set.NArg() != 0 {
		return options{}, fmt.Errorf("不明な引数があります")
	}
	if !opts.Confirmed {
		return options{}, fmt.Errorf("-%sの指定が必要です。接続先は別用途の既存Oracleである可能性があり、このtoolは管理者接続で専用ユーザーと検証用の表を一時的に作成・削除します", confirmFlag)
	}
	if opts.EvidenceOut != "" && !strings.Contains(filepath.Base(opts.EvidenceOut), evidenceOutNameMark) {
		return options{}, fmt.Errorf("-evidence-outのファイル名には%sを含めてください。実資産の検証根拠と取り違えないためです", evidenceOutNameMark)
	}
	if opts.CleanupRun != "" && !runIDPattern.MatchString(opts.CleanupRun) {
		return options{}, fmt.Errorf("-cleanupの実行IDは大文字16進8桁でなければなりません")
	}
	return opts, nil
}

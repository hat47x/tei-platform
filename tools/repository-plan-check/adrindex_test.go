package plancheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeADRFile(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRepositoryIndexMatchesADRFiles(t *testing.T) {
	problems, err := CheckADRIndex(filepath.Join("..", "..", "plan", "adr"))
	if err != nil {
		t.Fatal(err)
	}
	if len(problems) > 0 {
		t.Fatalf("plan/adr/README.md とADRファイルが一致しない:\n%s", strings.Join(problems, "\n"))
	}
}

func TestCheckReportsEachKindOfDisagreement(t *testing.T) {
	dir := t.TempDir()
	writeADRFile(t, dir, "0001-a.md", "# ADR-0001: a\n\n- 状態: 採用\n")
	writeADRFile(t, dir, "0003-c.md", "# ADR-0003: c\n\n- **状態**：採用（目標構成）\n")
	writeADRFile(t, dir, "0004-d.md", "# ADR-0004: d\n\n- 状態: 提案中\n")
	writeADRFile(t, dir, "0005-e.md", "# ADR-0005: e\n\n- 状態: 採用\n")
	writeADRFile(t, dir, "0005-e2.md", "# ADR-0005: e2\n\n- 状態: 採用\n")
	writeADRFile(t, dir, "0006-f.md", "# ADR-0006: f\n\n本文だけ\n")
	writeADRFile(t, dir, "0007-g.md", "# ADR-0007: g\n\n- 状態: 採用\n")
	writeADRFile(t, dir, "README.md", strings.Join([]string{
		"# 設計判断記録",
		"- [ADR-0001: a](0001-a.md)",
		"- ADR-0002: 公開前に削除",
		"- [ADR-0003: c](0003-c.md)（採用。目標構成）",
		"- [ADR-0004: d](0004-d.md)",
		"- [ADR-0005: e](0005-e.md)",
		"- [ADR-0006: f](0006-f.md)",
		"- [ADR-0008: h](0008-h.md)",
		"- [ADR-0001: a](0001-a.md)",
		"",
	}, "\n"))

	problems, err := CheckADRIndex(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join(problems, "\n")
	for _, want := range []string{
		"ADR-0004: 状態が一致しない（索引: 採用、ファイル: 提案中）",
		"ADR-0005: 同じ番号のファイルが2件ある",
		"ADR-0006: 0006-f.md の冒頭に状態の行を読み取れない",
		"ADR-0007: ファイル 0007-g.md が索引に載っていない",
		"ADR-0008: 索引のリンク先 0008-h.md がない",
		"ADR-0001: 索引に複数回載っている",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("次の指摘が出ない: %s\n実際:\n%s", want, got)
		}
	}
	for _, unwanted := range []string{"ADR-0002", "ADR-0003"} {
		if strings.Contains(got, unwanted) {
			t.Errorf("%s は指摘されないはず:\n%s", unwanted, got)
		}
	}
}

func TestUnlinkedIndexLineForAnExistingFileIsReported(t *testing.T) {
	dir := t.TempDir()
	writeADRFile(t, dir, "0002-b.md", "# ADR-0002\n\n- 状態: 採用\n")
	writeADRFile(t, dir, "README.md", "- ADR-0002: リンクなし\n")
	problems, err := CheckADRIndex(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(problems) != 1 || !strings.Contains(problems[0], "索引にリンクがない") {
		t.Fatalf("problems = %v", problems)
	}
}

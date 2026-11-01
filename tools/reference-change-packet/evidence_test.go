package changepacket

import (
	"context"
	"os/exec"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func completePacket() Packet {
	return Packet{
		Schema: Schema, ID: "packet-a", Issue: 2017, Title: "evidence test", Level: LevelA2,
		Base:         "HEAD",
		AllowedPaths: []string{"tools/reference-change-packet/**"},
		Verification: []Command{{Name: "check", Run: "true"}},
		StopConditions: []string{
			"unexplained diff",
		},
	}
}

func TestReceiptRecordsEnvironmentAndUnverified(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash is unavailable")
	}
	repo := initRepo(t)
	modifyReadme(t, repo)
	packet := completePacket()
	packet.AllowedPaths = []string{"README.md"}
	packet.Unverified = []string{"real Windows", "real DB"}
	receipt, err := Run(context.Background(), packet, repo)
	if err != nil {
		t.Fatal(err)
	}
	if !receipt.Passed {
		t.Fatalf("receipt=%+v", receipt)
	}
	if receipt.Environment.GOOS == "" || receipt.Environment.GOARCH == "" || receipt.Environment.GoVersion == "" {
		t.Fatalf("environment is incomplete: %+v", receipt.Environment)
	}
	if receipt.Environment.GOOS != runtime.GOOS || receipt.Environment.GoVersion != runtime.Version() {
		t.Fatalf("environment=%+v want the process environment", receipt.Environment)
	}
	if !reflect.DeepEqual(receipt.Unverified, packet.Unverified) {
		t.Fatalf("unverified=%v want %v", receipt.Unverified, packet.Unverified)
	}
}

func TestHumanJudgmentRequiresA4(t *testing.T) {
	for _, level := range []string{LevelA0, LevelA1, LevelA2, LevelA3} {
		packet := completePacket()
		packet.Level = level
		if level == LevelA0 || level == LevelA1 {
			packet.Base = ""
		}
		packet.HumanJudgment = []string{"meaning"}
		err := Validate(packet)
		if err == nil || !strings.Contains(err.Error(), "use A4") {
			t.Errorf("level %s accepted a meaning change: %v", level, err)
		}
	}
}

func TestA4AcceptsHumanJudgment(t *testing.T) {
	packet := completePacket()
	packet.Level = LevelA4
	packet.HumanJudgment = []string{"meaning", "authority", "production", "secrets", "license", "destructive-migration"}
	if err := Validate(packet); err != nil {
		t.Fatalf("A4 rejected the declared human-judgment reasons: %v", err)
	}
}

func TestUnknownHumanJudgmentReasonIsRejected(t *testing.T) {
	packet := completePacket()
	packet.Level = LevelA4
	packet.HumanJudgment = []string{"vibes"}
	if err := Validate(packet); err == nil {
		t.Fatal("an unknown human-judgment reason was accepted")
	}
}

func TestOrdinaryA2PacketStillPasses(t *testing.T) {
	if err := Validate(completePacket()); err != nil {
		t.Fatalf("ordinary A2 packet was rejected: %v", err)
	}
}

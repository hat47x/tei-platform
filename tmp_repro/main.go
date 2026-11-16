package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"tei/internal/externalanalysis"
	"tei/internal/operations"
	"tei/internal/workspace"
)

func main() {
	raw, err := os.ReadFile(filepath.Join("internal", "externalanalysis", "testdata", "rea-6.8.0", "analysis.json"))
	if err != nil {
		panic(err)
	}
	root, err := os.MkdirTemp("", "rea-repro")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(root)
	if err := os.WriteFile(filepath.Join(root, "analysis.json"), raw, 0o600); err != nil {
		panic(err)
	}
	got, err := workspace.ReadSource(root, "analysis.json", 4<<20)
	fmt.Printf("ReadSource err=%v bytes=%d\n", err, len(got))
	if err != nil {
		return
	}
	h := sha256.New()
	h.Write(got)
	fmt.Println("digest", "sha256:"+hex.EncodeToString(h.Sum(nil)))
	_, err = operations.ExternalCandidatesFromBytes(operations.ExternalCandidateRequest{
		ToolVersion:            "6.8.0",
		ObservationID:          "jsrg_node_237478f2a27c9614a522cbfc7ab60fa96943a458d164945d25af634f1240b652",
		RelationID:             "jsrg_relation_0f6757a241c56fd84ecaec91f538d8297d222eb5625e4cdbcb2de7603f330718",
		ExpectedEvidenceID:     "ev_a88c70bcd066b2f04fdf97cb349369f0779b9c68cc4ac6c0e5f38f3539477a93",
		ExpectedReceivedDigest: "sha256:7ea3780d8dc23d00c380145af62d63bf194e39decbaefab3d82c5a3b4134bba9",
		Tool:                   externalanalysis.Tool{Name: "rea", Present: true},
	}, got, time.Now())
	fmt.Printf("projection err=%v\n", err)
}

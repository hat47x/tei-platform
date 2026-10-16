//go:build linux

package runtime

import (
 "context"
 "crypto/sha256"
 "encoding/hex"
 "encoding/json"
 "os"
 "path/filepath"
 "testing"
)

func TestSignerLedgerRejectsOldReservationFilename(t *testing.T) {
 dir := t.TempDir()
 if err := os.Chmod(dir, 0700); err != nil { t.Fatal(err) }
 op := LinuxSignerOperationReservation{
  HostID:"host/legacy",OperationID:"preupgrade",Principal:"admin",
  SignerRef:"signed/tool",Action:"revoke",
 }
 raw, err := json.Marshal(op)
 if err != nil { t.Fatal(err) }
 identity := append([]byte(op.HostID), byte(0))
 identity = append(identity, op.OperationID...)
 hash := sha256.Sum256(identity)
 name := filepath.Join(dir,hex.EncodeToString(hash[:])+".json")
 if err := os.WriteFile(name,raw,0600); err != nil { t.Fatal(err) }
 ledger,err := NewLinuxFileSignerOperationLedger(dir)
 if err != nil { t.Fatal(err) }
 defer ledger.Close()
 if ok,err := ledger.ReserveSignerOperation(context.Background(),op); ok||err!=nil {
  t.Fatalf("legacy reservation was replayed: ok=%v err=%v",ok,err)
 }
 fresh:=op
 fresh.OperationID="new-operation"
 if ok,err:=ledger.ReserveSignerOperation(context.Background(),fresh); !ok||err!=nil {
  t.Fatalf("new operation denied: ok=%v err=%v",ok,err)
 }
}

package actionboundary

import (
 "context"
 "encoding/json"
 "fmt"
 "math"
 "sync"
)

// JSONDocumentStore is a reference-only, in-memory document adapter.
// It is not SUI DocumentV1 persistence. It deliberately preserves unknown
// top-level and card fields while testing the action commit boundary.
type JSONDocumentStore struct {
 mu sync.Mutex
 revision string
 documentID string
 document []byte
 writes int
}

// MovePayload is an opaque host payload understood only by this sample store.
type MovePayload struct {
 CardID string `json:"cardId"`
 X float64 `json:"x"`
 Y float64 `json:"y"`
}

func NewJSONDocumentStore(revision string, document []byte) *JSONDocumentStore {
 return &JSONDocumentStore{revision:revision,document:append([]byte(nil),document...)}
}

// BindDocument scopes this reference store to one document. It must be set
// before the store is exposed to concurrent callers.
func (s *JSONDocumentStore) BindDocument(id string) { s.mu.Lock(); defer s.mu.Unlock(); s.documentID=id }

// Snapshot returns a copy so a caller cannot mutate the stored document.
func (s *JSONDocumentStore) Snapshot() (string,[]byte,int) {
 s.mu.Lock()
 defer s.mu.Unlock()
 return s.revision,append([]byte(nil),s.document...),s.writes
}

func (s *JSONDocumentStore) Commit(_ context.Context,in Intent)(string,error) {
 s.mu.Lock()
 defer s.mu.Unlock()
 if s.documentID!="" && in.DocumentID!=s.documentID {return "",ErrDenied}
 if in.ExpectedRevision!=s.revision {return "",ErrConflict}
 payload,ok:=in.Payload.(MovePayload)
 if !ok || payload.CardID=="" || math.IsNaN(payload.X) || math.IsNaN(payload.Y) || math.IsInf(payload.X,0) || math.IsInf(payload.Y,0) {return "",ErrInvalid}
 if in.ActionID!="sui.move" {return "",ErrInvalid}
 var doc map[string]json.RawMessage
 if err:=json.Unmarshal(s.document,&doc);err!=nil || doc==nil {return "",fmt.Errorf("%w: invalid document JSON: %v",ErrInvalid,err)}
 var cards []map[string]json.RawMessage
 if err:=json.Unmarshal(doc["cards"],&cards);err!=nil || cards==nil {return "",fmt.Errorf("%w: invalid cards: %v",ErrInvalid,err)}
 found:=false
 seen:=map[string]bool{}
 for i:=range cards {
  if cards[i]==nil {return "",ErrInvalid}
  var id string
  if err:=json.Unmarshal(cards[i]["id"],&id);err!=nil || id=="" || seen[id] {return "",ErrInvalid}
  seen[id]=true
  if id!=payload.CardID {continue}
  if found {return "",fmt.Errorf("%w: duplicate card identity",ErrInvalid)}
  found=true
  position,err:=json.Marshal(struct{X float64 `json:"x"`;Y float64 `json:"y"`}{payload.X,payload.Y})
  if err!=nil {return "",ErrInvalid}
  cards[i]["position"]=position
 }
 if !found {return "",fmt.Errorf("%w: unknown card",ErrInvalid)}
 rawCards,err:=json.Marshal(cards);if err!=nil{return "",ErrInvalid}
 doc["cards"]=rawCards
 next,err:=json.Marshal(doc);if err!=nil{return "",ErrInvalid}
 s.document=next
 s.writes++
 s.revision=fmt.Sprintf("r%d",s.writes+1)
 return s.revision,nil
}

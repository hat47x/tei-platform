package main
import("fmt";"go/format";"os";"strings";"testing")
func TestTemporaryQuoteChainFormatDiff(t *testing.T) {
 b,e:=os.ReadFile("order_quote_runtime_chain_test.go");if e!=nil{t.Fatal(e)}
 a,e:=format.Source(b);if e!=nil{t.Fatal(e)}
 old,new:=strings.Split(string(b),"\n"),strings.Split(string(a),"\n")
 var diffs []string
 for i:=range old {if i<len(new)&&old[i]!=new[i]{diffs=append(diffs,fmt.Sprintf("L%d BEFORE=%q AFTER=%q",i+1,old[i],new[i]))}}
 if len(diffs)>0{t.Fatal(strings.Join(diffs,"\n"))}
}
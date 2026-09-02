from pathlib import Path


def replace_once(path: Path, old: str, new: str) -> None:
    text = path.read_text()
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"{path}: expected exactly one match, got {count}")
    path.write_text(text.replace(old, new, 1))


anchor = Path("internal/syntax/semantic_anchor.go")
if 'Kind: "semantic-validation"' not in anchor.read_text():
    replace_once(
        anchor,
        '''\t\tcase "capability":
\t\t\tsym, err := expectLexerIdent(lx, "semantic capability symbol")
\t\t\tif err != nil {
\t\t\t\treturn nil, err
\t\t\t}
\t\t\tid, err := semanticCapabilityID(namespace, sym.Text)
\t\t\tif err != nil {
\t\t\t\treturn nil, err
\t\t\t}
\t\t\tanchors = append(anchors, SubjectAnchor{Kind: "semantic-capability", ID: string(id), Anchor: tokenAnchor(sym)})
\t\t\tif err := expectLexerKind(lx, TokenLBrace, "semantic capability body"); err != nil {
\t\t\t\treturn nil, err
\t\t\t}
\t\t\tif err := skipSemanticCapabilityBody(lx); err != nil {
\t\t\t\treturn nil, err
\t\t\t}
\t\tdefault:''',
        '''\t\tcase "capability":
\t\t\tsym, err := expectLexerIdent(lx, "semantic capability symbol")
\t\t\tif err != nil {
\t\t\t\treturn nil, err
\t\t\t}
\t\t\tid, err := semanticCapabilityID(namespace, sym.Text)
\t\t\tif err != nil {
\t\t\t\treturn nil, err
\t\t\t}
\t\t\tanchors = append(anchors, SubjectAnchor{Kind: "semantic-capability", ID: string(id), Anchor: tokenAnchor(sym)})
\t\t\tif err := expectLexerKind(lx, TokenLBrace, "semantic capability body"); err != nil {
\t\t\t\treturn nil, err
\t\t\t}
\t\t\tif err := skipSemanticCapabilityBody(lx); err != nil {
\t\t\t\treturn nil, err
\t\t\t}
\t\tcase "validation":
\t\t\tsym, err := expectLexerIdent(lx, "semantic validation identity")
\t\t\tif err != nil {
\t\t\t\treturn nil, err
\t\t\t}
\t\t\tid, err := semanticValidationID(namespace, sym.Text)
\t\t\tif err != nil {
\t\t\t\treturn nil, err
\t\t\t}
\t\t\tanchors = append(anchors, SubjectAnchor{Kind: "semantic-validation", ID: string(id), Anchor: tokenAnchor(sym)})
\t\t\tif _, err := expectLexerIdent(lx, "semantic validation kind"); err != nil {
\t\t\t\treturn nil, err
\t\t\t}
\t\t\tif err := expectLexerKind(lx, TokenLBrace, "semantic validation body"); err != nil {
\t\t\t\treturn nil, err
\t\t\t}
\t\t\tif err := skipSemanticValidationBody(lx); err != nil {
\t\t\t\treturn nil, err
\t\t\t}
\t\tdefault:''',
    )

    marker = '''func semanticChildID(parent core.CanonicalID, symbol string) (core.CanonicalID, error) {'''
    text = anchor.read_text()
    helper = '''func skipSemanticValidationBody(lx *Lexer) error {
\tfor {
\t\tt, err := lx.Next()
\t\tif err != nil {
\t\t\treturn err
\t\t}
\t\tif t.Kind == TokenRBrace {
\t\t\treturn nil
\t\t}
\t\tif t.Kind == TokenEOF {
\t\t\treturn fmt.Errorf("unterminated semantic validation body")
\t\t}
\t\tif t.Kind != TokenIdent || (t.Text != "attribute" && t.Text != "relation") {
\t\t\treturn fmt.Errorf("unexpected semantic validation member %q at %d:%d", t.Text, t.Start.Line, t.Start.Column)
\t\t}
\t\tif _, err := expectLexerIdent(lx, "semantic validation reference"); err != nil {
\t\t\treturn err
\t\t}
\t}
}

func semanticValidationID(namespace core.CanonicalID, symbol string) (core.CanonicalID, error) {
\tif strings.Contains(symbol, ".") {
\t\tid, err := core.ParseCanonicalID(symbol)
\t\tif err != nil {
\t\t\treturn "", err
\t\t}
\t\tif !strings.HasPrefix(string(id), string(namespace)+".") {
\t\t\treturn "", fmt.Errorf("semantic validation %s does not belong to namespace %s", id, namespace)
\t\t}
\t\treturn id, nil
\t}
\treturn semanticChildID(namespace, symbol)
}

'''
    if text.count(marker) != 1:
        raise SystemExit("semantic_anchor.go: semanticChildID marker mismatch")
    anchor.write_text(text.replace(marker, helper + marker, 1))

source_index = Path("internal/workspace/source_index.go")
replace_once(
    source_index,
    '''\tcase "semantic", "semantic-type", "semantic-data", "semantic-attribute", "semantic-relation", "semantic-capability":''',
    '''\tcase "semantic", "semantic-type", "semantic-data", "semantic-attribute", "semantic-relation", "semantic-capability", "semantic-validation":''',
)
replace_once(
    source_index,
    '''\tcase "query":
\t\tkeyword = "query"
\tcase "semantic-validation":
\t\tkeyword = "validation"
\tdefault:''',
    '''\tcase "query":
\t\tkeyword = "query"
\tdefault:''',
)

print("semantic validation anchors patched")

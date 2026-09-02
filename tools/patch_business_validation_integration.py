from pathlib import Path


def replace_once(path: Path, old: str, new: str) -> None:
    text = path.read_text()
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"{path}: expected exactly one match, got {count}")
    path.write_text(text.replace(old, new, 1))


validation = Path("internal/validation/validation.go")
if "validateSemanticBusinessValidations(w)" not in validation.read_text():
    replace_once(
        validation,
        '''func Validate(w ir.World) []Diagnostic {
\tout := validateApplicationPatterns(w)
''',
        '''func Validate(w ir.World) []Diagnostic {
\tout := validateApplicationPatterns(w)
\tout = append(out, validateSemanticBusinessValidations(w)...)
''',
    )

source_index = Path("internal/workspace/source_index.go")
if 'case "semantic-validation":' not in source_index.read_text():
    replace_once(
        source_index,
        '''\tcase "query":
\t\tkeyword = "query"
\tdefault:''',
        '''\tcase "query":
\t\tkeyword = "query"
\tcase "semantic-validation":
\t\tkeyword = "validation"
\tdefault:''',
    )

print("business validation integration patched")

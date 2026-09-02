package postgres

import (
	"fmt"
	"sort"
	"strings"
)

// BoundQueryは、PostgreSQLで実行できるplaceholderへ変換したSQLと、
// そのplaceholderに対応する実行時引数を保持する。
//
// ParameterNamesは値を含まず、どのnamed parameterがどの順序で束縛されたかだけを示す。
// 実行時の値を通常Evidenceへ残す用途には使わない。
type BoundQuery struct {
	SQL            string
	Arguments      []any
	ParameterNames []string
}

// BindNamedParametersは、Query authoring上の`:name`形式をPostgreSQLの`$n`形式へ変換する。
//
// 単純な文字列置換は行わず、SQL文字列、quoted identifier、comment、dollar-quoted文字列、
// PostgreSQLの`::` castを区別する。同じparameterが複数回現れた場合は、最初に割り当てた
// placeholderを再利用する。
//
// Queryで参照しているparameterに値がなければerrorにする。また、呼び出し側が渡した値を
// 黙って無視しないため、Queryで使われていないparameterもerrorにする。
func BindNamedParameters(source string, parameters map[string]any) (BoundQuery, error) {
	var out strings.Builder
	out.Grow(len(source))

	positions := map[string]int{}
	bound := BoundQuery{}

	for i := 0; i < len(source); {
		switch {
		case source[i] == '\'':
			next, err := copySingleQuoted(&out, source, i)
			if err != nil {
				return BoundQuery{}, err
			}
			i = next
		case source[i] == '"':
			next, err := copyDoubleQuoted(&out, source, i)
			if err != nil {
				return BoundQuery{}, err
			}
			i = next
		case hasPrefixAt(source, i, "--"):
			i = copyLineComment(&out, source, i)
		case hasPrefixAt(source, i, "/*"):
			next, err := copyBlockComment(&out, source, i)
			if err != nil {
				return BoundQuery{}, err
			}
			i = next
		case source[i] == '$':
			delimiter, ok := dollarQuoteDelimiter(source, i)
			if !ok {
				out.WriteByte(source[i])
				i++
				continue
			}
			next, err := copyDollarQuoted(&out, source, i, delimiter)
			if err != nil {
				return BoundQuery{}, err
			}
			i = next
		case source[i] == ':' && hasPrefixAt(source, i, "::"):
			out.WriteString("::")
			i += 2
		case source[i] == ':' && i+1 < len(source) && isParameterNameStart(source[i+1]):
			end := i + 2
			for end < len(source) && isParameterNamePart(source[end]) {
				end++
			}
			name := source[i+1 : end]
			value, exists := parameters[name]
			if !exists {
				return BoundQuery{}, fmt.Errorf("Query parameter %qの値が指定されていません", name)
			}
			position, seen := positions[name]
			if !seen {
				bound.Arguments = append(bound.Arguments, value)
				bound.ParameterNames = append(bound.ParameterNames, name)
				position = len(bound.Arguments)
				positions[name] = position
			}
			fmt.Fprintf(&out, "$%d", position)
			i = end
		default:
			out.WriteByte(source[i])
			i++
		}
	}

	unused := make([]string, 0)
	for name := range parameters {
		if _, used := positions[name]; !used {
			unused = append(unused, name)
		}
	}
	if len(unused) > 0 {
		sort.Strings(unused)
		return BoundQuery{}, fmt.Errorf("Queryで使われていないparameterが指定されています: %s", strings.Join(unused, ", "))
	}

	bound.SQL = out.String()
	return bound, nil
}

func copySingleQuoted(out *strings.Builder, source string, start int) (int, error) {
	out.WriteByte(source[start])
	for i := start + 1; i < len(source); i++ {
		out.WriteByte(source[i])
		if source[i] != '\'' {
			continue
		}
		if i+1 < len(source) && source[i+1] == '\'' {
			out.WriteByte(source[i+1])
			i++
			continue
		}
		return i + 1, nil
	}
	return 0, fmt.Errorf("SQL文字列リテラルが閉じられていません")
}

func copyDoubleQuoted(out *strings.Builder, source string, start int) (int, error) {
	out.WriteByte(source[start])
	for i := start + 1; i < len(source); i++ {
		out.WriteByte(source[i])
		if source[i] != '"' {
			continue
		}
		if i+1 < len(source) && source[i+1] == '"' {
			out.WriteByte(source[i+1])
			i++
			continue
		}
		return i + 1, nil
	}
	return 0, fmt.Errorf("quoted identifierが閉じられていません")
}

func copyLineComment(out *strings.Builder, source string, start int) int {
	for i := start; i < len(source); i++ {
		out.WriteByte(source[i])
		if source[i] == '\n' {
			return i + 1
		}
	}
	return len(source)
}

func copyBlockComment(out *strings.Builder, source string, start int) (int, error) {
	depth := 0
	for i := start; i < len(source); {
		switch {
		case hasPrefixAt(source, i, "/*"):
			out.WriteString("/*")
			depth++
			i += 2
		case hasPrefixAt(source, i, "*/"):
			out.WriteString("*/")
			depth--
			i += 2
			if depth == 0 {
				return i, nil
			}
		default:
			out.WriteByte(source[i])
			i++
		}
	}
	return 0, fmt.Errorf("SQL block commentが閉じられていません")
}

func dollarQuoteDelimiter(source string, start int) (string, bool) {
	if source[start] != '$' {
		return "", false
	}
	if start+1 < len(source) && source[start+1] == '$' {
		return "$$", true
	}
	if start+1 >= len(source) || !isDollarTagStart(source[start+1]) {
		return "", false
	}
	for i := start + 2; i < len(source); i++ {
		if source[i] == '$' {
			return source[start : i+1], true
		}
		if !isDollarTagPart(source[i]) {
			return "", false
		}
	}
	return "", false
}

func copyDollarQuoted(out *strings.Builder, source string, start int, delimiter string) (int, error) {
	endStart := strings.Index(source[start+len(delimiter):], delimiter)
	if endStart < 0 {
		return 0, fmt.Errorf("dollar-quoted文字列 %s が閉じられていません", delimiter)
	}
	end := start + len(delimiter) + endStart + len(delimiter)
	out.WriteString(source[start:end])
	return end, nil
}

func hasPrefixAt(source string, start int, prefix string) bool {
	return start+len(prefix) <= len(source) && source[start:start+len(prefix)] == prefix
}

func isParameterNameStart(b byte) bool {
	return b == '_' || b >= 'A' && b <= 'Z' || b >= 'a' && b <= 'z'
}

func isParameterNamePart(b byte) bool {
	return isParameterNameStart(b) || b >= '0' && b <= '9'
}

func isDollarTagStart(b byte) bool {
	return isParameterNameStart(b)
}

func isDollarTagPart(b byte) bool {
	return isParameterNamePart(b)
}

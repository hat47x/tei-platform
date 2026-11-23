package stdpart

import (
	"fmt"
	"math/big"
)

// ParseDecimal は、十進数の文字列を数に読み替えます。
// 書けるのは、省略できる負号、数字、省略できる小数部だけです。
// 指数、桁区切り、正号、前後の空白、分数、"007" のような先頭の 0、
// "5." のような小数部のない小数点は受け付けません。
// 別の数とも読める書き方は、近い数に丸めずに誤りとして返します。
func ParseDecimal(text string) (*big.Rat, error) {
	digits := text
	if len(digits) > 0 && digits[0] == '-' {
		digits = digits[1:]
	}
	seenDigit, seenPoint, fractionDigits := false, false, 0
	for i := 0; i < len(digits); i++ {
		c := digits[i]
		switch {
		case c >= '0' && c <= '9':
			seenDigit = true
			if seenPoint {
				fractionDigits++
			}
		case c == '.' && !seenPoint && seenDigit:
			seenPoint = true
		default:
			return nil, notDecimal(text)
		}
	}
	leadingZero := len(digits) > 1 && digits[0] == '0' && digits[1] != '.'
	if !seenDigit || seenPoint && fractionDigits == 0 || leadingZero {
		return nil, notDecimal(text)
	}
	// text は上の検査で数字・小数点・負号だけと確かめてあります。
	value, ok := new(big.Rat).SetString(text)
	if !ok {
		return nil, notDecimal(text)
	}
	return value, nil
}

func notDecimal(text string) error {
	return fmt.Errorf("%q is not a decimal number; write digits with an optional minus sign and fraction, for example -0.01", text)
}

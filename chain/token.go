package chain

import (
	"fmt"
	"strings"
	"regexp"

	"github.com/ericlagergren/decimal"
)

const tokenDenomRegexStr = `^([[:digit:]]+(?:\.[[:digit:]]+)?|\.[[:digit:]]+)[[:space:]]*([a-zA-Z][a-zA-Z0-9/:._-]{2,127})$`

var TokenDenomRegex = regexp.MustCompile(tokenDenomRegexStr)

type Token struct {
	Amount decimal.Big
	Symbol string
}

func (t *Token) String() string {
	return fmt.Sprintf("%s%s", t.Amount.String(), t.Symbol)
}

func (t *Token) Rebase(exponent int, symbol string) Token {
	scale := t.Amount.Scale()
	token := Token{
		Symbol: symbol,
		Amount: t.Amount,
	}
	token.Amount.SetScale(scale + exponent)
	return token
}

func ParseToken(s string) (Token, error) {
	matches := TokenDenomRegex.FindStringSubmatch(strings.TrimSpace(s))
	if len(matches) != 3 {
		return Token{}, fmt.Errorf("failed to parse token")
	}
	token := Token{
		Symbol: matches[2],
	}
	_, ok := token.Amount.SetString(matches[1])
	if !ok {
		return token, fmt.Errorf("failed to parse token amount")
	}
	return token, nil
}

func ParseTokens(s string) ([]Token, error) {
	tokens := []Token{}
	for _, tokenString := range strings.Split(s, ",") {
		token, err := ParseToken(tokenString)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
	}
	return tokens, nil
}

package token

import (
	"fmt"
	"strings"
)

const DefaultPairSeparator = "/"

type Pair struct {
	Base string `json:"base"`
	Quote string `json:"quote"`
}

func PairFromString(s string) (*Pair, error) {
	return PairFromStringWithSeparator(s, DefaultPairSeparator)
}

func PairFromStringWithSeparator(s string, separator string) (*Pair, error) {
	separatorIndex := strings.Index(s, separator)
	if separatorIndex == -1 {
		return nil, fmt.Errorf("separator not found in pair string")
	}
	pair := &Pair{
		Base: s[:separatorIndex],
		Quote: s[separatorIndex + 1:],
	}
	return pair, nil
}

func (p *Pair) String() string {
	return p.Base + DefaultPairSeparator + p.Quote
}

func (p *Pair) StringWithSeparator(separator string) string {
	return p.Base + separator + p.Quote
}

func (p *Pair) Reversed() *Pair {
	return &Pair{
		Base: p.Quote,
		Quote: p.Base,
	}
}

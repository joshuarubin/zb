package buildflags

import (
	"fmt"
	"strings"
)

// taken from go source src/cmd/go/build.go

func isSpaceByte(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

func splitQuotedFields(s string) ([]string, error) {
	// Split fields allowing '' or "" around elements.
	// Quotes further inside the string do not count.
	var f []string
	for len(s) > 0 {
		for len(s) > 0 && isSpaceByte(s[0]) {
			s = s[1:]
		}
		if len(s) == 0 {
			break
		}
		// Accepted quoted string. No unescaping inside.
		if s[0] == '"' || s[0] == '\'' {
			quote := s[0]
			s = s[1:]
			i := 0
			for i < len(s) && s[i] != quote {
				i++
			}
			if i >= len(s) {
				return nil, fmt.Errorf("unterminated %c string", quote)
			}
			f = append(f, s[:i])
			s = s[i+1:]
			continue
		}
		i := 0
		for i < len(s) && !isSpaceByte(s[i]) {
			i++
		}
		f = append(f, s[:i])
		s = s[i:]
	}
	return f, nil
}

type stringsFlag []string

func (v *stringsFlag) Set(s string) error {
	var err error
	*v, err = splitQuotedFields(s)
	if *v == nil {
		*v = []string{}
	}
	return err
}

func (v *stringsFlag) String() string {
	return strings.Join(*v, " ")
}

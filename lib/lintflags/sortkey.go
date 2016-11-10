package lintflags

import "github.com/pkg/errors"

type SortKey string

const (
	SortNone     SortKey = "none"
	SortPath             = "path"
	SortLine             = "line"
	SortColumn           = "column"
	SortSeverity         = "severity"
	SortMessage          = "message"
	SortLinter           = "linter"
)

var sortKeys = []SortKey{
	SortNone,
	SortPath,
	SortLine,
	SortColumn,
	SortSeverity,
	SortMessage,
	SortLinter,
}

var sortKeysStr []string

func init() {
	for _, key := range sortKeys {
		sortKeysStr = append(sortKeysStr, string(key))
	}
}

func (k SortKey) String() string {
	return string(k)
}

func (k *SortKey) Set(value string) error {
	for _, key := range sortKeys {
		if string(key) == value {
			*k = key
			return nil
		}
	}

	return errors.Errorf("unknown")
}

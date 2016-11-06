package project

import (
	"os"
	"time"

	"jrubin.io/slog"
)

var _ Dependency = (*GoFile)(nil)

type GoFile struct {
	Path   string
	Logger slog.Interface
}

func (e GoFile) Name() string {
	return e.Path
}

func (e GoFile) ModTime() time.Time {
	i, err := os.Stat(e.Path)
	if err != nil {
		return time.Time{}
	}
	return i.ModTime()
}

func (e GoFile) Dependencies() ([]Dependency, error) {
	// TODO(jrubin) go:generate
	// TODO(jrubin) zb:generate (to list deps of go:generate)
	// TODO(jrubin) return ModTime() time.Now() if go:generate and not
	// zb:generate
	// TODO(jrubin) cache results
	return nil, nil
}

func (e GoFile) Build() error {
	// TODO(jrubin)
	return nil
}

package dependency

import (
	"os"
	"time"

	"jrubin.io/zb/lib/zbcontext"
)

var _ Dependency = (*File)(nil)

type File string

func (f File) Name() string {
	return string(f)
}

func (f File) Build(zbcontext.Context) error {
	// noop
	return nil
}

func (f File) Install(zbcontext.Context) error {
	// noop
	return nil
}

func (f File) ModTime() time.Time {
	i, err := os.Stat(string(f))
	if err != nil {
		return time.Time{}
	}
	return i.ModTime()
}

func (f File) Dependencies(zbcontext.Context) ([]Dependency, error) {
	// noop
	return nil, nil
}

func (f File) Buildable() bool {
	return false
}

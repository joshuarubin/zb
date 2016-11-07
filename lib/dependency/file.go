package dependency

import (
	"os"
	"time"
)

var _ Dependency = (*File)(nil)

type File string

func (f File) Name() string {
	return string(f)
}

func (f File) Build() error {
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

func (f File) Dependencies() ([]Dependency, error) {
	// noop
	return nil, nil
}

func (f File) Buildable() bool {
	return false
}

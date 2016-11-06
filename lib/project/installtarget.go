package project

import (
	"os"
	"time"
)

type InstallTarget struct {
	*Package
	Path string
}

var _ Dependency = (*InstallTarget)(nil)

func (t *InstallTarget) Name() string {
	return t.Path
}

func (t *InstallTarget) ModTime() time.Time {
	i, err := os.Stat(t.Path)
	if err != nil {
		return time.Time{}
	}

	return i.ModTime()
}

package dependency

import (
	"os"
	"time"
)

var _ Dependency = (*GoGenerateFile)(nil)

type GoGenerateFile struct {
	GoFile  *GoFile
	Depends File
	Path    string
}

func (f GoGenerateFile) Name() string {
	return f.Path
}

func (f GoGenerateFile) Build() error {
	return f.GoFile.Generate()
}

func (f GoGenerateFile) ModTime() time.Time {
	i, err := os.Stat(f.Path)
	if err != nil {
		return time.Time{}
	}
	return i.ModTime()
}

func (f GoGenerateFile) Dependencies() ([]Dependency, error) {
	return []Dependency{f.Depends}, nil
}

func (f GoGenerateFile) Buildable() bool {
	return true
}

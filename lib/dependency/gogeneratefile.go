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
	// with patsubst, multiple GoGenerateFiles may exist pointing to the same
	// GoFile. we want to ensure that go generate only runs once in these cases.
	// we lock the GoFile to ensure no concurrent go generates and once we have
	// a lock, we recheck that it even needs to be run at all.

	f.GoFile.mu.Lock()
	defer f.GoFile.mu.Unlock()

	if !f.Depends.ModTime().After(f.ModTime()) {
		return nil
	}

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

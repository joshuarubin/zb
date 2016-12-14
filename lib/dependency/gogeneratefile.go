package dependency

import (
	"os"
	"time"

	"jrubin.io/zb/lib/zbcontext"
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

func (f GoGenerateFile) Build(ctx zbcontext.Context) error {
	// with patsubst, multiple GoGenerateFiles may exist pointing to the same
	// GoFile. we want to ensure that go generate only runs once in these cases.
	// we lock the GoFile to ensure no concurrent go generates and once we have
	// a lock, we recheck that it even needs to be run at all.

	f.GoFile.mu.Lock()
	defer f.GoFile.mu.Unlock()

	if !f.Depends.ModTime().After(f.ModTime()) {
		return nil
	}

	return f.GoFile.Generate(ctx)
}

func (f GoGenerateFile) Install(ctx zbcontext.Context) error {
	return f.Build(ctx)
}

func (f GoGenerateFile) ModTime() time.Time {
	i, err := os.Stat(f.Path)
	if err != nil {
		return time.Time{}
	}
	return i.ModTime()
}

func (f GoGenerateFile) Dependencies(zbcontext.Context) ([]Dependency, error) {
	return []Dependency{f.Depends}, nil
}

func (f GoGenerateFile) Buildable() bool {
	return true
}

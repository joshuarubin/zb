package project

import (
	"go/build"
	"path/filepath"
	"time"

	"jrubin.io/zb/lib/dependency"
)

// A Package is a single go Package
type Package struct {
	*build.Package
	IsVendored bool
	Project    *Project
}

// Executable returns the absolute path of the executable that this package generates
// when it is built
func (pkg *Package) Executable() *Executable {
	if pkg.Name != "main" {
		return nil
	}

	// TODO(jrubin) should executables be put in the project root or in the
	// directory of the "main" package?

	// return filepath.Join(pkg.Dir, filepath.Base(pkg.Dir))
	return &Executable{
		Path:    filepath.Join(pkg.Project.Dir, filepath.Base(pkg.Dir)),
		Package: pkg,
	}
}

var _ dependency.Dependency = (*Executable)(nil)

type Executable struct {
	Path    string
	Package *Package
}

func (e Executable) String() string {
	return e.Path
}

func (e Executable) Build() error {
	// TODO(jrubin)
	return nil
}

func (e Executable) ModTime() time.Time {
	// TODO(jrubin)
	return time.Time{}
}

func (e Executable) Dependencies() []dependency.Dependency {
	// TODO(jrubin)
	return nil
}

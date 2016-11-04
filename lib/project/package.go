package project

import (
	"go/build"
	"path/filepath"
)

type Package struct {
	*build.Package
	IsVendored bool
	Project    *Project
}

func (pkg *Package) Cmd() string {
	if pkg.Name != "main" {
		return ""
	}

	// TODO(jrubin) should executables be put in the project root or in the
	// directory of the "main" package?

	// return filepath.Join(pkg.Dir, filepath.Base(pkg.Dir))
	return filepath.Join(pkg.Project.Dir, filepath.Base(pkg.Dir))
}

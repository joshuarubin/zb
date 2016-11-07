package project

import (
	"go/build"

	"jrubin.io/zb/lib/dependency"
	"jrubin.io/zb/lib/zbcontext"
)

// A Package is a single go Package
type Package struct {
	*build.Package
	*Project
	IsVendored bool
}

func (pkg *Package) BuildPath() string {
	return zbcontext.BuildPath(pkg.Project.Dir, pkg.Package)
}

func (pkg *Package) InstallPath() string {
	return zbcontext.InstallPath(pkg.Package)
}

// Command returns the absolute path of the executable that this package generates
// when it is built
func (pkg *Package) BuildTarget() (*dependency.GoPackage, error) {
	if !pkg.IsCommand() {
		return pkg.InstallTarget()
	}

	hash, err := pkg.GitCommit()
	if err != nil {
		return nil, err
	}

	return &dependency.GoPackage{
		ProjectImportPath: pkg.DirToImportPath(pkg.Project.Dir),
		Path:              pkg.BuildPath(),
		Package:           pkg.Package,
		Context:           pkg.Context,
		GitCommit:         hash,
	}, nil
}

func (pkg *Package) InstallTarget() (*dependency.GoPackage, error) {
	hash, err := pkg.GitCommit()
	if err != nil {
		return nil, err
	}

	return &dependency.GoPackage{
		ProjectImportPath: pkg.DirToImportPath(pkg.Project.Dir),
		Path:              pkg.InstallPath(),
		Package:           pkg.Package,
		Context:           pkg.Context,
		GitCommit:         hash,
	}, nil
}

type TargetType int

const (
	TargetBuild TargetType = iota
	TargetInstall
)

func (pkg *Package) Targets(tt TargetType) (*dependency.Targets, error) {
	var fn func() (*dependency.GoPackage, error)

	switch tt {
	case TargetBuild:
		fn = pkg.BuildTarget
	case TargetInstall:
		fn = pkg.InstallTarget
	}

	gopkg, err := fn()
	if err != nil {
		return nil, err
	}

	queue := []*dependency.Target{dependency.NewTarget(gopkg, nil)}
	unique := dependency.Targets{}

	// recursively add all dependencies
	for len(queue) > 0 {
		// pop the queue
		target := queue[0]
		queue = queue[1:]

		if !unique.Insert(target) {
			continue
		}

		deps, err := target.Dependencies()
		if err != nil {
			return nil, err
		}

		// append these dependencies to the queue
		for _, dep := range deps {
			queue = append(queue, dependency.NewTarget(dep, target))
		}
	}

	return &unique, nil
}

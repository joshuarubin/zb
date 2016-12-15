package dependency

import (
	"go/build"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/src-d/go-git.v4/core"

	"jrubin.io/zb/lib/zbcontext"
)

type GoPackage struct {
	*build.Package
	Hash              *core.Hash
	Path              string
	ProjectImportPath string

	dependencies []Dependency
}

var _ Dependency = (*GoPackage)(nil)

func (pkg *GoPackage) Name() string {
	return pkg.Path
}

func (pkg *GoPackage) Build(ctx zbcontext.Context) error {
	if !pkg.IsCommand() {
		return pkg.Install(ctx)
	}

	path := pkg.Name()
	if rel, err := filepath.Rel(zbcontext.CWD, path); err == nil {
		path = rel
	}

	args := []string{"build"}
	args = append(args, pkg.BuildArgs(ctx)...)
	args = append(args, "-o", path)
	args = append(args, pkg.ImportPath)

	if err := ctx.GoExec(args...); err != nil {
		return err
	}

	return ctx.Touch(pkg.Name())
}

func (pkg *GoPackage) BuildArgs(ctx zbcontext.Context) []string {
	return ctx.BuildArgs(pkg.Package, pkg.Hash)
}

func (pkg *GoPackage) Install(ctx zbcontext.Context) error {
	args := []string{"install"}
	args = append(args, pkg.BuildArgs(ctx)...)
	args = append(args, pkg.ImportPath)

	if err := ctx.GoExec(args...); err != nil {
		return err
	}

	return ctx.Touch(pkg.Name())
}

func (pkg *GoPackage) ModTime() time.Time {
	i, err := os.Stat(pkg.Path)
	if err != nil {
		return time.Time{}
	}

	return i.ModTime()
}

func (pkg *GoPackage) files(ctx zbcontext.Context) []Dependency {
	var files []string

	files = append(files, pkg.GoFiles...)
	files = append(files, pkg.CgoFiles...)
	files = append(files, pkg.CFiles...)
	files = append(files, pkg.CXXFiles...)
	files = append(files, pkg.MFiles...)
	files = append(files, pkg.HFiles...)
	files = append(files, pkg.FFiles...)
	files = append(files, pkg.SFiles...)
	files = append(files, pkg.SwigFiles...)
	files = append(files, pkg.SwigCXXFiles...)
	files = append(files, pkg.SysoFiles...)
	// files = append(files, pkg.TestGoFiles...)
	// files = append(files, pkg.XTestGoFiles...)

	gofiles := make([]Dependency, len(files))
	for i, f := range files {
		gofiles[i] = NewGoFile(ctx, pkg, filepath.Join(pkg.Dir, f))
	}

	return gofiles
}

func (pkg *GoPackage) packages(ctx zbcontext.Context) ([]Dependency, error) {
	var pkgs []Dependency

	imports := pkg.Imports

	for _, i := range imports {
		if !strings.Contains(i, ".") {
			// skip standard library packages and "C"
			continue
		}

		p, err := ctx.Import(i, pkg.Dir)
		if err != nil {
			return nil, err
		}

		pkgs = append(pkgs, &GoPackage{
			ProjectImportPath: pkg.ProjectImportPath,
			Path:              p.PkgObj,
			Package:           p,
			Hash:              pkg.Hash,
		})
	}

	return pkgs, nil
}

func (pkg *GoPackage) Buildable() bool {
	return true
}

func (pkg *GoPackage) Dependencies(ctx zbcontext.Context) ([]Dependency, error) {
	if pkg.dependencies != nil {
		return pkg.dependencies, nil
	}

	pkgs, err := pkg.packages(ctx)
	if err != nil {
		return nil, err
	}

	pkg.dependencies = pkgs
	pkg.dependencies = append(pkg.dependencies, pkg.files(ctx)...)

	return pkg.dependencies, nil
}

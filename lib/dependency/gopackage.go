package dependency

import (
	"fmt"
	"go/build"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/src-d/go-git.v4/core"
	"jrubin.io/zb/lib/zbcontext"
)

type GoPackage struct {
	*build.Package
	zbcontext.Context
	Path              string
	GitCommit         core.Hash
	ProjectImportPath string

	dependencies []Dependency
}

var _ Dependency = (*GoPackage)(nil)

func (pkg GoPackage) Name() string {
	return pkg.Path
}

const dateFormat = "2006-01-02T15:04:05+00:00"

func (pkg GoPackage) buildFlags() ([]string, error) {
	args := pkg.BuildArgs()

	if !pkg.IsCommand() {
		return args, nil
	}

	if len(pkg.BuildFlags.LDFlags) > 0 {
		// don't override explicitly proviede ldflags
		return args, nil
	}

	ldflags := fmt.Sprintf("-X main.gitCommit=%s -X main.buildDate=%s",
		pkg.GitCommit,
		time.Now().UTC().Format(dateFormat),
	)

	return append(args, "-ldflags", ldflags), nil
}

func quoteCommand(command string, args []string) string {
	for _, a := range args {
		if strings.Contains(a, " ") {
			a = strconv.Quote(a)
		}
		command += " " + a
	}
	return command
}

func (pkg GoPackage) Build() error {
	if !pkg.IsCommand() {
		return pkg.Install()
	}

	buildFlags, err := pkg.buildFlags()
	if err != nil {
		return err
	}

	args := []string{"build"}
	args = append(args, buildFlags...)
	args = append(args, "-o", pkg.Name())
	args = append(args, pkg.ImportPath)

	if err := pkg.GoExec(args...); err != nil {
		return err
	}

	dir := pkg.ImportPathToDir(pkg.ProjectImportPath)
	path := zbcontext.BuildPath(dir, pkg.Package)

	return pkg.Touch(path)
}

func (pkg GoPackage) Install() error {
	buildFlags, err := pkg.buildFlags()
	if err != nil {
		return err
	}

	args := []string{"install"}
	args = append(args, buildFlags...)
	args = append(args, pkg.ImportPath)

	if err := pkg.GoExec(args...); err != nil {
		return err
	}

	return pkg.Touch(zbcontext.InstallPath(pkg.Package))
}

func (pkg GoPackage) ModTime() time.Time {
	i, err := os.Stat(pkg.Path)
	if err != nil {
		return time.Time{}
	}

	return i.ModTime()
}

func (pkg GoPackage) files() []Dependency {
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
		gofiles[i] = &GoFile{
			Context: pkg.Context,
			Path:    filepath.Join(pkg.Dir, f),
		}
	}

	return gofiles
}

func (pkg GoPackage) packages() ([]Dependency, error) {
	var pkgs []Dependency

	imports := pkg.Imports

	for _, i := range imports {
		if !strings.HasPrefix(i, pkg.ProjectImportPath) {
			continue
		}

		p, err := pkg.Import(i, "")
		if err != nil {
			return nil, err
		}

		pkgs = append(pkgs, &GoPackage{
			ProjectImportPath: pkg.ProjectImportPath,
			Path:              p.PkgObj,
			Package:           p,
			Context:           pkg.Context,
		})
	}

	return pkgs, nil
}

func (pkg *GoPackage) Buildable() bool {
	return true
}

func (pkg *GoPackage) Dependencies() ([]Dependency, error) {
	if pkg.dependencies != nil {
		return pkg.dependencies, nil
	}

	pkgs, err := pkg.packages()
	if err != nil {
		return nil, err
	}

	pkg.dependencies = pkgs
	pkg.dependencies = append(pkg.dependencies, pkg.files()...)

	return pkg.dependencies, nil
}

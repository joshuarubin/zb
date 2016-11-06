package project

import (
	"fmt"
	"go/build"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"jrubin.io/slog"
)

// A Package is a single go Package
type Package struct {
	*build.Package
	*Project
	IsVendored bool

	dependencies []Dependency
}

// Command returns the absolute path of the executable that this package generates
// when it is built
func (pkg *Package) Command() *InstallTarget {
	if !pkg.IsCommand() {
		return nil
	}

	file := filepath.Join(pkg.Project.Dir, filepath.Base(pkg.Package.Dir))
	return &InstallTarget{
		Path:    file,
		Package: pkg,
	}
}

func (pkg *Package) InstallTarget() *InstallTarget {
	path := pkg.PkgObj

	if pkg.IsCommand() {
		path = filepath.Join(pkg.BinDir, filepath.Base(pkg.Package.Dir))
	}

	return &InstallTarget{
		Path:    path,
		Package: pkg,
	}
}

func (pkg *Package) files(p *build.Package) []Dependency {
	var files []string

	files = append(files, p.GoFiles...)
	files = append(files, p.CgoFiles...)
	files = append(files, p.CFiles...)
	files = append(files, p.CXXFiles...)
	files = append(files, p.MFiles...)
	files = append(files, p.HFiles...)
	files = append(files, p.FFiles...)
	files = append(files, p.SFiles...)
	files = append(files, p.SwigFiles...)
	files = append(files, p.SwigCXXFiles...)
	files = append(files, p.SysoFiles...)
	files = append(files, p.TestGoFiles...)
	files = append(files, p.XTestGoFiles...)

	gofiles := make([]Dependency, len(files))
	for i, f := range files {
		gofiles[i] = &GoFile{
			Path:   filepath.Join(p.Dir, f),
			Logger: pkg.Logger,
		}
	}

	return gofiles
}

func (pkg *Package) testPackages() ([]*build.Package, error) {
	var pkgs []*build.Package

	for _, i := range append(pkg.TestImports, pkg.XTestImports...) {
		p, err := pkg.BuildContext.Import(i, "", build.ImportComment)
		if err != nil {
			return nil, err
		}
		pkgs = append(pkgs, p)
	}

	return pkgs, nil
}

// list of files this package depends on to be built and tested
func (pkg *Package) Dependencies() ([]Dependency, error) {
	if pkg.dependencies != nil {
		return pkg.dependencies, nil
	}

	// only include test imports for tests in this package
	pkgs, err := pkg.testPackages()
	if err != nil {
		return nil, err
	}

	pkgs = append(pkgs, pkg.Package)

	// unique list of imports
	imports := map[string]struct{}{}
	for _, p := range pkgs {
		imports[p.ImportPath] = struct{}{}
	}

	for len(pkgs) > 0 {
		// pop the front
		p := pkgs[0]
		pkgs = pkgs[1:]

		// exclude standard packages
		if !strings.Contains(p.ImportPath, ".") {
			continue
		}

		pkg.dependencies = append(pkg.dependencies, pkg.files(p)...)

		for _, i := range p.Imports {
			if _, ok := imports[i]; ok {
				// already loaded this import
				continue
			}

			ip, err := pkg.BuildContext.Import(i, "", build.ImportComment)
			if err != nil {
				return nil, err
			}

			imports[i] = struct{}{}
			pkgs = append(pkgs, ip)
		}
	}

	return pkg.dependencies, nil
}

func (pkg *Package) Targets() ([]*Target, error) {
	exe := pkg.Command()
	if exe == nil {
		return nil, nil
	}

	queue := []*Target{{Dependency: exe}}
	var targets []*Target

	// recursively add all dependencies
	for len(queue) > 0 {
		// pop the queue
		target := queue[0]
		queue = queue[1:]

		targets = append(targets, target)

		deps, err := target.Dependencies()
		if err != nil {
			return nil, err
		}

		// append these dependencies to the queue
		for _, dep := range deps {
			targets = append(targets, &Target{
				Dependency: dep,
				Parent:     target,
			})
		}
	}

	return targets, nil
}

const dateFormat = "2006-01-02T15:04:05+00:00"

func (pkg *Package) buildFlags() ([]string, error) {
	if pkg.Name != "main" {
		return pkg.BuildFlags, nil
	}

	for _, f := range pkg.BuildFlags {
		if f == "-ldflags" {
			// don't override explicitly proviede ldflags
			return pkg.BuildFlags, nil
		}
	}

	commit, err := pkg.GitCommit()
	if err != nil {
		return nil, err
	}

	ldflags := fmt.Sprintf("-X main.gitCommit=%s -X main.buildDate=%s",
		commit,
		time.Now().UTC().Format(dateFormat),
	)

	return append(pkg.BuildFlags, "-ldflags", ldflags), nil
}

func (pkg *Package) Build() error {
	buildFlags, err := pkg.buildFlags()
	if err != nil {
		return err
	}

	args := []string{"build"}
	args = append(args, buildFlags...)
	args = append(args, "-o", pkg.Command().Name())
	args = append(args, pkg.ImportPath)

	line := "> go"
	for _, a := range args {
		if strings.Contains(a, " ") {
			line += " '" + a + "'"
		} else {
			line += " " + a
		}
	}
	pkg.Logger.Info(line)

	writer := pkg.Logger.Writer(slog.InfoLevel).Prefix("< ")
	defer writer.Close()

	cmd := exec.Command("go", args...) // nosec
	cmd.Stdout = writer
	cmd.Stderr = writer
	return cmd.Run()
}

package project

import (
	"go/build"
	"os"
	"path/filepath"
	"strings"

	"jrubin.io/slog"

	"jrubin.io/zb/pkgs"

	"github.com/pkg/errors"
)

type Project struct {
	Dir          string
	Packages     map[string]*build.Package
	BuildContext build.Context
	Logger       slog.Interface
}

func Dir(value string) (string, error) {
	dir := value
	for {
		test := filepath.Join(dir, ".git")
		_, err := os.Stat(test)
		if err != nil {
			ndir := filepath.Dir(dir)
			if ndir == dir {
				return "", errors.Errorf("could not find project dir for: %s", value)
			}

			dir = ndir
			continue
		}

		return dir, nil
	}
}

func New(logger slog.Interface, bc build.Context, path, srcDir string) (*Project, error) {
	pkg, err := build.Import(path, srcDir, build.ImportComment)
	if err != nil {
		return nil, err
	}

	pd, err := Dir(pkg.Dir)
	if err != nil {
		return nil, err
	}

	return &Project{
		Dir: pd,
		Packages: map[string]*build.Package{
			pkg.Dir: pkg,
		},
		BuildContext: bc,
		Logger:       logger,
	}, nil
}

func (p *Project) baseImportPath() (string, error) {
	// path may be a/b/c/d
	// p.Dir may be /home/user/go/src/a/b
	// this will return a/b even if there are no .go files in it
	// e.g. it may not be a valid import path

	for _, src := range p.BuildContext.SrcDirs() {
		src += "/"
		if strings.Index(p.Dir, src) != 0 {
			continue
		}

		return p.Dir[len(src):], nil
	}

	return "", errors.Errorf("could not find base import path for: %s", p.Dir)
}

func (p *Project) FillPackages() error {
	base, err := p.baseImportPath()
	if err != nil {
		return err
	}

	// base should always be a fully qualified package import, never an absolute
	// or relative path

	paths := pkgs.ImportPaths(p.BuildContext, filepath.Join(base, "..."))
	for _, path := range paths {
		pkg, err := build.Import(path, "", build.ImportComment)
		if err != nil {
			return err
		}
		p.Packages[pkg.Dir] = pkg
	}

	return nil
}

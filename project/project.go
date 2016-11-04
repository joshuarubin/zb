package project

import (
	"go/build"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"jrubin.io/zb/pkgs"

	"github.com/pkg/errors"
)

type Package struct {
	*build.Package
	Vendor  bool
	Project *Project
}

type Project struct {
	Dir          string
	Packages     PackageList
	BuildContext build.Context

	filled bool
}

func projectDir(value string) (string, error) {
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

type ProjectList []*Project

func (l *ProjectList) Len() int {
	return len(*l)
}

func (l *ProjectList) Less(i, j int) bool {
	return (*l)[i].Dir < (*l)[j].Dir
}

func (l *ProjectList) Swap(i, j int) {
	(*l)[i], (*l)[j] = (*l)[j], (*l)[i]
}

func (l *ProjectList) Search(p *Project) int {
	return sort.Search(l.Len(), func(i int) bool {
		return (*l)[i].Dir >= p.Dir
	})
}

func (l *ProjectList) Insert(p *Project) bool {
	exists, i := l.Exists(p)
	if exists {
		return false
	}

	*l = append(*l, nil)
	copy((*l)[i+1:], (*l)[i:])
	(*l)[i] = p

	return true
}

func (l ProjectList) Exists(p *Project) (bool, int) {
	i := l.Search(p)
	return (i < l.Len() && l[i].Dir == p.Dir), i
}

// can't handle ellipsis (...), but does not require .go files to exist either
func importPathToDir(bc build.Context, importPath string) string {
	for _, srcDir := range bc.SrcDirs() {
		dir := filepath.Join(srcDir, importPath)
		info, err := os.Stat(dir)
		if err != nil || !info.IsDir() {
			continue
		}
		return dir
	}
	return ""
}

func dirToImportPath(bc build.Context, dir string) string {
	for _, srcDir := range bc.SrcDirs() {
		srcDir += "/"
		if strings.Index(dir, srcDir) == 0 {
			return dir[len(srcDir):]
		}
	}
	return ""
}

func importPathToProjectDir(bc build.Context, importPath string) string {
	dir := importPathToDir(bc, importPath)
	if dir == "" {
		return ""
	}
	dir, err := projectDir(dir)
	if err != nil || dir == "" {
		return ""
	}
	return dir
}

func noGoImportPathToProjectImportPaths(bc build.Context, warner pkgs.Warner, srcDir string, importPath string) []string {
	dir := importPathToProjectDir(bc, importPath)
	if dir == "" {
		return nil
	}

	// found project dir, now convert it back to an import path so
	// we can use ellipsis
	importPath = dirToImportPath(bc, dir)

	// lets see if we can find any packages under it
	return pkgs.ImportPaths(bc, warner, filepath.Join(importPath, "..."))
}

// does not run Projects.FillPackages as it can be expensive
func Projects(bc build.Context, srcDir string, warner pkgs.Warner, args ...string) (ProjectList, error) {
	if len(args) == 0 {
		args = append(args, ".")
	}

	importPaths := pkgs.ImportPaths(bc, warner, args...)

	var projects ProjectList

	// don't use range, using importPaths as a queue
	for len(importPaths) > 0 {
		// pop the queue
		importPath := importPaths[0]
		importPaths = importPaths[1:]

		// convert local imports to import paths
		if build.IsLocalImport(importPath) {
			// convert relative path to absolute
			if !filepath.IsAbs(importPath) {
				importPath = filepath.Join(srcDir, importPath)
			}

			if found := dirToImportPath(bc, importPath); found != "" {
				importPath = found
			}
		}

		p, err := New(bc, importPath, srcDir)

		if _, ok := err.(*build.NoGoError); ok && err != nil {
			// no buildable source files in the given dir
			// ok, as long as the project dir can still be found and at least
			// one subdir of the project dir has go files
			//
			// importPath may still be relative too, but it is guaranteed not to
			// have ellipsis

			newImportPaths := noGoImportPathToProjectImportPaths(bc, warner, srcDir, importPath)
			if len(newImportPaths) > 0 {
				// add the new paths to the queue and ignore the error
				importPaths = append(importPaths, newImportPaths...)
				continue
			}
		}

		if err != nil {
			return nil, err
		}

		projects.Insert(p)
	}

	return projects, nil
}

func New(bc build.Context, importPath, srcDir string) (*Project, error) {
	pkg, err := build.Import(importPath, srcDir, build.ImportComment)
	if err != nil {
		return nil, err
	}

	pd, err := projectDir(pkg.Dir)
	if err != nil {
		return nil, err
	}

	p := &Project{
		Dir:          pd,
		BuildContext: bc,
		Packages:     PackageList{},
	}

	p.Packages.Insert(&Package{
		Package: pkg,
		Project: p,
		Vendor:  false,
	})

	return p, nil
}

func (p *Project) baseImportPath() (string, error) {
	// path may be a/b/c/d
	// p.Dir may be /home/user/go/src/a/b
	// this will return a/b even if there are no .go files in it
	// e.g. it may not be a valid import path

	if dir := dirToImportPath(p.BuildContext, p.Dir); dir != "" {
		return dir, nil
	}

	return "", errors.Errorf("could not find base import path for: %s", p.Dir)
}

func (l ProjectList) FillPackages(warner pkgs.Warner) error {
	for _, p := range l {
		if err := p.FillPackages(warner); err != nil {
			return err
		}
	}
	return nil
}

type PackageList []*Package

func (l *PackageList) Len() int {
	return len(*l)
}

func (l *PackageList) Less(i, j int) bool {
	return (*l)[i].Dir < (*l)[j].Dir
}

func (l *PackageList) Swap(i, j int) {
	(*l)[i], (*l)[j] = (*l)[j], (*l)[i]
}

func (l *PackageList) Search(p *Package) int {
	return sort.Search(l.Len(), func(i int) bool {
		return (*l)[i].Dir >= p.Dir
	})
}

func (l *PackageList) Insert(p *Package) bool {
	exists, i := l.Exists(p)
	if exists {
		return false
	}

	*l = append(*l, nil)
	copy((*l)[i+1:], (*l)[i:])
	(*l)[i] = p

	return true
}

func (l PackageList) Exists(p *Package) (bool, int) {
	i := l.Search(p)
	return (i < l.Len() && l[i].Dir == p.Dir), i
}

// sorted by ProjectDir,PackageDir
func (l ProjectList) Packages(warner pkgs.Warner) (PackageList, error) {
	var ps PackageList

	for _, project := range l {
		if err := project.FillPackages(warner); err != nil {
			return nil, err
		}

		for _, pkg := range project.Packages {
			ps.Insert(pkg)
		}
	}

	return ps, nil
}

func (p *Project) FillPackages(warner pkgs.Warner) error {
	if p.filled {
		return nil
	}

	p.filled = true

	base, err := p.baseImportPath()
	if err != nil {
		return err
	}

	// base should always be a fully qualified package import, never an absolute
	// or relative path

	importPaths := pkgs.ImportPaths(p.BuildContext, warner, filepath.Join(base, "..."))
	for _, importPath := range importPaths {
		pkg, err := build.Import(importPath, "", build.ImportComment)
		if err != nil {
			return err
		}

		p.Packages.Insert(&Package{
			Package: pkg,
			Project: p,
			Vendor:  strings.Contains(pkg.ImportPath, "/vendor/"),
		})
	}

	return nil
}

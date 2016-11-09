package project

import (
	"path/filepath"
	"strings"

	"golang.org/x/sync/errgroup"

	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/core"

	"jrubin.io/zb/lib/dependency"
	"jrubin.io/zb/lib/zbcontext"

	"github.com/pkg/errors"
)

// A Project is a collection of Packages contained within a single repository
type Project struct {
	zbcontext.Context
	Dir      string
	Packages []*Package

	filled bool
}

func (p *Project) fillPackages() error {
	if p.filled {
		return nil
	}

	p.filled = true

	base := p.DirToImportPath(p.Dir)
	if base == "" {
		return errors.Errorf("could not find base import path for: %s", p.Dir)
	}

	// base should always be a fully qualified package import, never an absolute
	// or relative path

	list := (*packageList)(&p.Packages)

	importPaths := p.ExpandEllipsis(filepath.Join(base, "..."))
	for _, importPath := range importPaths {
		if dir := p.ImportPathToDir(importPath); dir != "" {
			if ok, _ := list.Exists(dir); ok {
				continue
			}
		}

		pkg, err := p.newPackage(importPath, p.Dir, true)
		if err != nil {
			return err
		}

		if p.ExcludeVendor && pkg.IsVendored {
			continue
		}

		list.Insert(pkg)
	}

	return nil
}

func (p *Project) Targets(tt TargetType) (*dependency.Targets, error) {
	unique := dependency.Targets{}
	var group errgroup.Group
	for _, pkg := range p.Packages {
		pp := pkg
		group.Go(func() error {
			ts, err := pp.Targets(tt)
			if err != nil {
				return err
			}

			unique.Append(ts)
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		return nil, err
	}
	return &unique, nil
}

func (p *Project) GitCommit() core.Hash {
	dir := filepath.Join(p.Dir, ".git")

	repo, err := git.NewFilesystemRepository(dir)
	if err != nil {
		p.Logger.WithError(err).Warn("could not determine git commit")
		return core.Hash{}
	}

	head, err := repo.Head()
	if err != nil {
		p.Logger.WithError(err).Warn("could not determine git commit")
		return core.Hash{}
	}

	return head.Hash()
}

var cache = map[string]*Package{}

func (p *Project) newPackage(importPath, srcDir string, includeTestImports bool) (*Package, error) {
	if pkg, ok := cache[importPath]; ok {
		return pkg, nil
	}

	pkg, err := p.Import(importPath, srcDir)
	if err != nil {
		return nil, err
	}

	isVendored := strings.Contains(pkg.ImportPath, "vendor/")

	ret := &Package{
		Package:            pkg,
		Project:            p,
		IsVendored:         isVendored,
		includeTestImports: !isVendored && includeTestImports,
	}

	cache[importPath] = ret

	return ret, nil
}

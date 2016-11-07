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

		// TODO(jrubin) this isn't a very accurate check
		isVendored := strings.Contains(importPath, "/vendor/")

		if p.ExcludeVendor && isVendored {
			continue
		}

		pkg, err := p.Import(importPath, p.Dir)
		if err != nil {
			return err
		}

		list.Insert(&Package{
			Package:    pkg,
			Project:    p,
			IsVendored: isVendored,
		})
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

func (p *Project) GitCommit() (core.Hash, error) {
	dir := filepath.Join(p.Dir, ".git")

	repo, err := git.NewFilesystemRepository(dir)
	if err != nil {
		return core.Hash{}, err
	}

	head, err := repo.Head()
	if err != nil {
		return core.Hash{}, err
	}

	return head.Hash(), nil
}

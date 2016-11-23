package project

import (
	"path/filepath"

	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/core"

	"jrubin.io/zb/lib/dependency"
	"jrubin.io/zb/lib/zbcontext"

	"github.com/pkg/errors"
)

// A Project is a collection of Packages contained within a single repository
type Project struct {
	*zbcontext.Context
	Dir      string
	Packages Packages

	gitCommit *core.Hash
	filled    bool
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

	importPaths := p.ExpandEllipsis(filepath.Join(base, "..."))
	for _, importPath := range importPaths {
		if dir := p.ImportPathToDir(importPath); dir != "" {
			if ok, _ := p.Packages.Exists(dir); ok {
				continue
			}
		}

		pkg, err := NewPackage(p.Context, importPath, p.Dir, true)
		if err != nil {
			return err
		}

		if p.ExcludeVendor && pkg.IsVendored {
			continue
		}

		p.Packages.Insert(pkg)
	}

	return nil
}

func (p *Project) Targets(tt dependency.TargetType) (*dependency.Targets, error) {
	return p.Packages.targets(tt, p.Dir, p.GitCommit())
}

func (p *Project) GitCommit() *core.Hash {
	if p.gitCommit != nil {
		return p.gitCommit
	}

	dir := filepath.Join(p.Dir, ".git")

	repo, err := git.NewFilesystemRepository(dir)
	if err != nil {
		p.Logger.WithError(err).Warn("could not determine git commit")
		return nil
	}

	head, err := repo.Head()
	if err != nil {
		p.Logger.WithError(err).Warn("could not determine git commit")
		return nil
	}

	h := head.Hash()
	p.gitCommit = &h
	return p.gitCommit
}

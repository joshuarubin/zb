package project

import (
	"path/filepath"

	git "srcd.works/go-git.v4"
	"srcd.works/go-git.v4/plumbing"

	"jrubin.io/slog"
	"jrubin.io/zb/lib/dependency"
	"jrubin.io/zb/lib/zbcontext"

	"github.com/pkg/errors"
)

// A Project is a collection of Packages contained within a single repository
type Project struct {
	Dir      string
	Packages Packages

	gitCommit *plumbing.Hash
	filled    bool
}

func (p *Project) fillPackages(ctx zbcontext.Context) error {
	if p.filled {
		return nil
	}

	p.filled = true

	base := ctx.DirToImportPath(p.Dir)
	if base == "" {
		return errors.Errorf("could not find base import path for: %s", p.Dir)
	}

	// base should always be a fully qualified package import, never an absolute
	// or relative path

	importPaths := ctx.ExpandEllipsis(filepath.Join(base, "..."))
	for _, importPath := range importPaths {
		if dir := ctx.ImportPathToDir(importPath); dir != "" {
			if ok, _ := p.Packages.Exists(dir); ok {
				continue
			}
		}

		pkg, err := NewPackage(ctx, importPath, p.Dir, true)
		if err != nil {
			return err
		}

		// if the -a build flag was specified, excluded vendored
		// packages as those will be built by the go tool through it's
		// dependency calculation
		if ctx.BuildArger != nil && ctx.RebuildAll() && pkg.IsVendored {
			continue
		}

		if ctx.ExcludeVendor && pkg.IsVendored {
			continue
		}

		p.Packages.Insert(pkg)
	}

	return nil
}

func (p *Project) Targets(ctx zbcontext.Context, tt dependency.TargetType) (*dependency.Targets, error) {
	return p.Packages.targets(ctx, tt, p.Dir, p.GitCommit(ctx.Logger))
}

func (p *Project) GitCommit(logger slog.Interface) *plumbing.Hash {
	if p.gitCommit != nil {
		return p.gitCommit
	}

	repo, err := git.PlainOpen(p.Dir)
	if err != nil {
		logger.WithField("dir", p.Dir).WithError(err).Warn("could not determine git commit")
		return nil
	}

	head, err := repo.Head()
	if err != nil {
		logger.WithField("dir", p.Dir).WithError(err).Warn("could not determine git commit")
		return nil
	}

	h := head.Hash()
	p.gitCommit = &h
	return p.gitCommit
}

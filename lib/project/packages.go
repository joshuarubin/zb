package project

import (
	"sort"

	"gopkg.in/src-d/go-git.v4/core"

	"golang.org/x/sync/errgroup"
	"jrubin.io/zb/lib/dependency"
	"jrubin.io/zb/lib/zbcontext"
)

type Packages []*Package

var _ sort.Interface = (*Packages)(nil)

func (p *Packages) Len() int {
	return len(*p)
}

func (p *Packages) Less(i, j int) bool {
	return (*p)[i].Dir < (*p)[j].Dir
}

func (p *Packages) Swap(i, j int) {
	(*p)[i], (*p)[j] = (*p)[j], (*p)[i]
}

func (p *Packages) Search(dir string) int {
	return sort.Search(p.Len(), func(i int) bool {
		return (*p)[i].Package.Dir >= dir
	})
}

func (p *Packages) Insert(n *Package) bool {
	exists, i := p.Exists(n.Package.Dir)
	if exists {
		return false
	}

	*p = append(*p, nil)
	copy((*p)[i+1:], (*p)[i:])
	(*p)[i] = n

	return true
}

func (p Packages) Exists(dir string) (bool, int) {
	i := p.Search(dir)
	return (i < p.Len() && p[i].Package.Dir == dir), i
}

func (p Packages) Append(r Packages) Packages {
	for _, pkg := range r {
		p.Insert(pkg)
	}

	return p
}

func (p Packages) targets(ctx zbcontext.Context, tt dependency.TargetType, projectDir string, gitCommit *core.Hash) (*dependency.Targets, error) {
	unique := dependency.Targets{}
	var group errgroup.Group

	for _, pkg := range p {
		pp := pkg
		group.Go(func() error {
			ts, err := pp.Targets(ctx, tt, projectDir, gitCommit)
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

func (p Packages) Targets(ctx zbcontext.Context, tt dependency.TargetType) ([]*dependency.Target, error) {
	unique, err := p.targets(ctx, tt, "", nil)
	if err != nil {
		return nil, err
	}

	targets := unique.TopologicalSort()

	// set up the waitgroup dependencies
	for _, t := range targets {
		target := t

		target.RequiredBy.Range(func(r *dependency.Target) {
			r.Add(1)
			target.OnDone(r.WaitGroup.Done)
		})
	}

	return targets, nil
}

func (p Packages) Build(ctx zbcontext.Context, tt dependency.TargetType) (int, error) {
	targets, err := p.Targets(ctx, tt)
	if err != nil {
		return 0, err
	}
	return dependency.Build(ctx, tt, targets)
}

func ListPackages(ctx zbcontext.Context, paths ...string) (Packages, error) {
	var pkgs Packages
	importPaths := ctx.ExpandEllipsis(paths...)

	for _, path := range importPaths {
		pkg, err := NewPackage(ctx, path, zbcontext.CWD, true)
		if err != nil {
			return nil, err
		}
		pkgs.Insert(pkg)
	}

	return pkgs, nil
}

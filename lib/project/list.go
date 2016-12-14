package project

import (
	"sort"

	"golang.org/x/sync/errgroup"
	"jrubin.io/zb/lib/dependency"
	"jrubin.io/zb/lib/zbcontext"
)

type List []*Project

func (l *List) Len() int {
	return len(*l)
}

func (l *List) Search(dir string) int {
	return sort.Search(l.Len(), func(i int) bool {
		return (*l)[i].Dir >= dir
	})
}

func (l *List) Insert(p *Project) bool {
	exists, i := l.Exists(p.Dir)
	if exists {
		return false
	}

	*l = append(*l, nil)
	copy((*l)[i+1:], (*l)[i:])
	(*l)[i] = p

	return true
}

func (l List) Exists(dir string) (bool, int) {
	i := l.Search(dir)
	return (i < l.Len() && l[i].Dir == dir), i
}

func (l List) Targets(ctx zbcontext.Context, tt dependency.TargetType) ([]*dependency.Target, error) {
	unique := dependency.Targets{}

	var group errgroup.Group

	for _, p := range l {
		pp := p
		group.Go(func() error {
			targets, err := pp.Targets(ctx, tt)
			if err != nil {
				return err
			}

			unique.Append(targets)
			return nil
		})
	}

	if err := group.Wait(); err != nil {
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

func (l List) Build(ctx zbcontext.Context, tt dependency.TargetType) (int, error) {
	targets, err := l.Targets(ctx, tt)
	if err != nil {
		return 0, err
	}
	return dependency.Build(ctx, tt, targets)
}

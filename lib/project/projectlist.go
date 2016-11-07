package project

import (
	"sort"

	"golang.org/x/sync/errgroup"

	"jrubin.io/zb/lib/dependency"
)

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

func (l *ProjectList) Search(dir string) int {
	return sort.Search(l.Len(), func(i int) bool {
		return (*l)[i].Dir >= dir
	})
}

func (l *ProjectList) Insert(p *Project) bool {
	exists, i := l.Exists(p.Dir)
	if exists {
		return false
	}

	*l = append(*l, nil)
	copy((*l)[i+1:], (*l)[i:])
	(*l)[i] = p

	return true
}

func (l ProjectList) Exists(dir string) (bool, int) {
	i := l.Search(dir)
	return (i < l.Len() && l[i].Dir == dir), i
}

func (l ProjectList) Targets(tt TargetType) ([]*dependency.Target, error) {
	unique := dependency.Targets{}

	var group errgroup.Group

	for _, p := range l {
		pp := p
		group.Go(func() error {
			targets, err := pp.Targets(tt)
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

func (l ProjectList) TargetsEach(tt TargetType, fn func(*dependency.Target) error) error {
	targets, err := l.Targets(tt)
	if err != nil {
		return err
	}

	var group errgroup.Group

	for _, t := range targets {
		target := t

		group.Go(func() error {
			defer target.Done()

			if !target.Buildable() {
				return nil
			}

			target.Wait()

			return fn(target)
		})
	}

	return group.Wait()
}

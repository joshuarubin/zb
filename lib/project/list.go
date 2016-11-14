package project

import (
	"sort"
	"sync/atomic"

	"golang.org/x/sync/errgroup"
	"jrubin.io/zb/lib/dependency"
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

func (l List) Targets(tt TargetType) ([]*dependency.Target, error) {
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

func (l List) TargetsEach(tt TargetType, fn func(*dependency.Target) error) error {
	targets, err := l.Targets(tt)
	if err != nil {
		return err
	}

	var group errgroup.Group

	for _, t := range targets {
		target := t

		deps, err := target.Dependencies()
		if err != nil {
			return err
		}

		group.Go(func() error {
			defer target.Done()

			if !target.Buildable() && len(deps) == 0 {
				return nil
			}

			target.Wait()

			if !target.Buildable() {
				return nil
			}

			return fn(target)
		})
	}

	return group.Wait()
}

func (l List) Build(tt TargetType) (int, error) {
	var built uint32
	err := l.TargetsEach(tt, func(target *dependency.Target) error {
		if tt == TargetGenerate {
			if _, ok := target.Dependency.(*dependency.GoGenerateFile); !ok {
				// exclude all dependencies that aren't go generate files
				return nil
			}
		}

		deps, err := target.Dependencies()
		if err != nil {
			return err
		}

		// build target if any of its dependencies are newer than itself
		for _, dep := range deps {
			// don't use .Before since filesystem time resolution might
			// cause files times to be within the same second
			if !dep.ModTime().After(target.ModTime()) {
				continue
			}

			if tt == TargetInstall {
				err = target.Install()
			} else {
				err = target.Build()
			}

			if err != nil {
				return err
			}

			atomic.AddUint32(&built, 1)
			return nil
		}

		return nil
	})
	return int(built), err
}

type packageList []*Package

func (l *packageList) Len() int {
	return len(*l)
}

func (l *packageList) Less(i, j int) bool {
	return (*l)[i].Package.Dir < (*l)[j].Package.Dir
}

func (l *packageList) Swap(i, j int) {
	(*l)[i], (*l)[j] = (*l)[j], (*l)[i]
}

func (l *packageList) Search(dir string) int {
	return sort.Search(l.Len(), func(i int) bool {
		return (*l)[i].Package.Dir >= dir
	})
}

func (l *packageList) Insert(p *Package) bool {
	exists, i := l.Exists(p.Package.Dir)
	if exists {
		return false
	}

	*l = append(*l, nil)
	copy((*l)[i+1:], (*l)[i:])
	(*l)[i] = p

	return true
}

func (l packageList) Exists(dir string) (bool, int) {
	i := l.Search(dir)
	return (i < l.Len() && l[i].Package.Dir == dir), i
}

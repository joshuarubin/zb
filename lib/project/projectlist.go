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

	return unique.TopologicalSort(), nil
}

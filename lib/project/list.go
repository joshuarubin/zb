package project

import "sort"

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

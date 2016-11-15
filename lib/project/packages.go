package project

import "sort"

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

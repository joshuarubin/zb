package project

import (
	"sort"

	"github.com/pkg/errors"

	"jrubin.io/zb/lib/dag"
)

type Projects []*Project

func (l *Projects) Len() int {
	return len(*l)
}

func (l *Projects) Less(i, j int) bool {
	return (*l)[i].Dir < (*l)[j].Dir
}

func (l *Projects) Swap(i, j int) {
	(*l)[i], (*l)[j] = (*l)[j], (*l)[i]
}

func (l *Projects) Search(dir string) int {
	return sort.Search(l.Len(), func(i int) bool {
		return (*l)[i].Dir >= dir
	})
}

func (l *Projects) Insert(p *Project) bool {
	exists, i := l.Exists(p.Dir)
	if exists {
		return false
	}

	*l = append(*l, nil)
	copy((*l)[i+1:], (*l)[i:])
	(*l)[i] = p

	return true
}

func (l Projects) Exists(dir string) (bool, int) {
	i := l.Search(dir)
	return (i < l.Len() && l[i].Dir == dir), i
}

type Target struct {
	Dependency
	Parent *Target

	node dag.Node
}

func (l Projects) Targets() ([]*Target, error) {
	// build a list of dependencies
	graph := dag.Graph{}

	// start with the final targets, the executables
	for _, p := range l {
		targets, err := p.Targets()
		if err != nil {
			return nil, err
		}

		for _, target := range targets {
			target.node = graph.MakeNode(target)

			if target.Parent != nil {
				graph.MakeEdge(target.node, target.Parent.node)
			}
		}
	}

	var targets []*Target

	// the graph now contains all possible dependencies
	// sort it by dependency order
	for _, node := range graph.TopologicalSort() {
		target, ok := (*node.Value).(*Target)
		if !ok {
			return nil, errors.New("node was not a Target")
		}

		targets = append(targets, target)
	}

	return targets, nil
}

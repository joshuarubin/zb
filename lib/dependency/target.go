package dependency

import (
	"reflect"
	"sort"
	"sync"

	"github.com/pkg/errors"

	"jrubin.io/zb/lib/dag"
)

type Target struct {
	Dependency
	RequiredBy *Targets
	Data       interface{}
}

var targetCache = Targets{}

func NewTarget(dep Dependency, req *Target) *Target {
	var t *Target

	if ok, i := targetCache.Exists(dep); ok {
		t = targetCache.Get(i)
	} else {
		t = &Target{
			Dependency: dep,
			RequiredBy: &Targets{},
		}
		targetCache.Insert(t)
	}

	if req != nil {
		t.RequiredBy.Insert(req)
	}

	return t
}

func (t Target) typeName() string {
	return reflect.Indirect(reflect.ValueOf(t.Dependency)).Type().Name()
}

func typeName(d Dependency) string {
	return reflect.Indirect(reflect.ValueOf(d)).Type().Name()
}

type Targets struct {
	list []*Target
	mu   sync.RWMutex
}

func (ts *Targets) Len() int {
	ts.mu.RLock()
	l := len(ts.list)
	ts.mu.RUnlock()
	return l
}

func (ts *Targets) Less(i, j int) bool {
	if ts.Get(i).Name() == ts.Get(j).Name() {
		return ts.Get(i).typeName() < ts.Get(j).typeName()
	}

	return ts.Get(i).Name() < ts.Get(j).Name()
}

func (ts *Targets) Swap(i, j int) {
	ts.mu.Lock()
	ts.list[i], ts.list[j] = ts.list[j], ts.list[i]
	ts.mu.Unlock()
}

func (ts *Targets) Search(d Dependency) int {
	return sort.Search(ts.Len(), func(i int) bool {
		if ts.Get(i).Name() == d.Name() {
			return ts.Get(i).typeName() >= typeName(d)
		}
		return ts.Get(i).Name() > d.Name()
	})
}

func (ts *Targets) Get(i int) *Target {
	ts.mu.RLock()
	t := ts.list[i]
	ts.mu.RUnlock()
	return t
}

func (ts *Targets) Set(i int, t *Target) {
	ts.mu.Lock()
	ts.list[i] = t
	ts.mu.Unlock()
}

type TargetsRangeFunc func(index int, target *Target)

func (ts *Targets) Range(fn TargetsRangeFunc) {
	ts.mu.RLock()
	for i, t := range ts.list {
		fn(i, t)
	}
	ts.mu.RUnlock()
}

func (ts *Targets) Insert(t *Target) bool {
	exists, i := ts.Exists(t)

	if exists {
		e := ts.Get(i)
		t.RequiredBy.Range(func(_ int, t *Target) {
			e.RequiredBy.Insert(t)
		})

		return false
	}

	ts.mu.Lock()
	ts.list = append(ts.list, nil)
	copy(ts.list[i+1:], ts.list[i:])
	ts.list[i] = t
	ts.mu.Unlock()

	return true
}

func (ts *Targets) Exists(d Dependency) (bool, int) {
	i := ts.Search(d)
	if i >= ts.Len() {
		return false, i
	}

	if ts.Get(i).Name() != d.Name() {
		return false, i
	}

	if ts.Get(i).typeName() != typeName(d) {
		return false, i
	}

	return true, i
}

func (ts *Targets) TopologicalSort() []*Target {
	// build a list of dependencies
	graph := dag.Graph{}

	ts.mu.RLock()

	for _, target := range ts.list {
		target.Data = graph.MakeNode(target)
	}

	for _, target := range ts.list {
		target.RequiredBy.Range(func(_ int, t *Target) {
			graph.MakeEdge(target.Data.(dag.Node), t.Data.(dag.Node))
		})
	}

	ts.mu.RUnlock()

	ret := make([]*Target, ts.Len())

	// the graph now contains all possible dependencies
	// sort it by dependency order
	for i, node := range graph.TopologicalSort() {
		target, ok := (*node.Value).(*Target)
		if !ok {
			panic(errors.New("node was not a Target"))
		}

		ret[i] = target
	}

	return ret
}

func (ts *Targets) Append(r *Targets) {
	r.Range(func(_ int, t *Target) {
		ts.Insert(t)
	})
}

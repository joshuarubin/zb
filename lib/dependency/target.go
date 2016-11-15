package dependency

import (
	"reflect"
	"sync"
	"sync/atomic"

	"golang.org/x/sync/errgroup"

	"github.com/pkg/errors"

	"jrubin.io/zb/lib/dag"
)

type Target struct {
	Dependency

	RequiredBy Targets
	Data       interface{}

	sync.WaitGroup

	mu        sync.Mutex
	doneFuncs []func()
}

func (t *Target) OnDone(fn func()) {
	t.mu.Lock()
	t.doneFuncs = append(t.doneFuncs, fn)
	t.mu.Unlock()
}

func (t *Target) Done() {
	t.mu.Lock()

	for len(t.doneFuncs) > 0 {
		fn := t.doneFuncs[0]
		t.doneFuncs = t.doneFuncs[1:]

		fn()
	}

	t.mu.Unlock()
}

var targetCache = Targets{}

func NewTarget(dep Dependency, req *Target) *Target {
	t := &Target{Dependency: dep}
	targetCache.Insert(t)

	t, _ = targetCache.exists(t)

	if req != nil {
		t.RequiredBy.Insert(req)
	}

	return t
}

func (t *Target) key() string {
	return t.Name() + t.typeName()
}

func (t *Target) typeName() string {
	return reflect.Indirect(reflect.ValueOf(t.Dependency)).Type().Name()
}

type Targets struct {
	list map[string]*Target
	mu   sync.RWMutex
}

func (ts *Targets) lenNoLock() int {
	return len(ts.list)
}

type TargetsRangeFunc func(target *Target)

func (ts *Targets) Range(fn TargetsRangeFunc) {
	ts.mu.Lock()
	for _, t := range ts.list {
		fn(t)
	}
	ts.mu.Unlock()
}

func (ts *Targets) Insert(t *Target) bool {
	ts.mu.Lock()
	ret := ts.insertNoLock(t)
	ts.mu.Unlock()
	return ret
}

func (ts *Targets) insertNoLock(t *Target) bool {
	if exists, ok := ts.existsNoLock(t); ok {
		if exists != t {
			exists.RequiredBy.Append(&t.RequiredBy)
		}
		return false
	}

	if ts.list == nil {
		ts.list = map[string]*Target{}
	}

	ts.list[t.key()] = t

	return true
}

func (ts *Targets) exists(t *Target) (*Target, bool) {
	ts.mu.Lock()
	t, i := ts.existsNoLock(t)
	ts.mu.Unlock()
	return t, i
}

func (ts *Targets) existsNoLock(t *Target) (*Target, bool) {
	if ts.list == nil {
		return nil, false
	}
	exists, ok := ts.list[t.key()]
	return exists, ok
}

func (ts *Targets) TopologicalSort() []*Target {
	// build a list of dependencies
	graph := dag.Graph{}

	ts.mu.RLock()

	for _, target := range ts.list {
		target.Data = graph.MakeNode(target)
	}

	for _, target := range ts.list {
		target.RequiredBy.Range(func(t *Target) {
			if err := graph.MakeEdge(target.Data.(dag.Node), t.Data.(dag.Node)); err != nil {
				panic(err)
			}
		})
	}

	ret := make([]*Target, ts.lenNoLock())

	ts.mu.RUnlock()

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
	ts.mu.Lock()
	r.Range(func(t *Target) {
		ts.insertNoLock(t)
	})
	ts.mu.Unlock()
}

type TargetFunc func(*Target) error

func Each(targets []*Target, fn TargetFunc) error {
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

type TargetType int

const (
	TargetBuild TargetType = iota
	TargetInstall
	TargetGenerate
)

func Build(tt TargetType, targets []*Target) (int, error) {
	var built uint32
	err := Each(targets, func(target *Target) error {
		if tt == TargetGenerate {
			if _, ok := target.Dependency.(*GoGenerateFile); !ok {
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

package dependency

import (
	"time"

	"jrubin.io/zb/lib/zbcontext"
)

type Dependency interface {
	Name() string
	Build(ctx zbcontext.Context) error
	Install(ctx zbcontext.Context) error
	ModTime() time.Time
	Dependencies(ctx zbcontext.Context) ([]Dependency, error)
	Buildable() bool
}

package dependency

import "time"

type Dependency interface {
	Name() string
	Build() error
	Install() error
	ModTime() time.Time
	Dependencies() ([]Dependency, error)
	Buildable() bool
}

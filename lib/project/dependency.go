package project

import "time"

type Dependency interface {
	Name() string
	Build() error
	ModTime() time.Time
	Dependencies() ([]Dependency, error)
}

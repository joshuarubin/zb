package dependency

import "time"

type Dependency interface {
	String() string
	Build() error
	ModTime() time.Time
	Dependencies() []Dependency
}

var _ Dependency = (*GoFile)(nil)

type GoFile string

func (e GoFile) String() string {
	return string(e)
}

func (e GoFile) Build() error {
	// TODO(jrubin)
	return nil
}

func (e GoFile) ModTime() time.Time {
	// TODO(jrubin)
	return time.Time{}
}

func (e GoFile) Dependencies() []Dependency {
	// TODO(jrubin)
	return nil
}

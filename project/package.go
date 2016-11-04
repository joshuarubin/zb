package project

import "go/build"

type Package struct {
	*build.Package
	IsVendored bool
	Project    *Project
}

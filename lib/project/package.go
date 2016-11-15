package project

import (
	"crypto/sha1"
	"fmt"
	"go/build"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/src-d/go-git.v4/core"

	"github.com/pkg/errors"

	"jrubin.io/zb/lib/buildflags"
	"jrubin.io/zb/lib/dependency"
	"jrubin.io/zb/lib/lintflags"
	"jrubin.io/zb/lib/zbcontext"
)

// A Package is a single go Package
type Package struct {
	*build.Package
	*zbcontext.Context

	IsVendored bool

	deps                        Packages
	depsBuilt                   bool
	includeTestImports          bool
	pkgHash, testHash, lintHash string
	depMap                      map[string]*Package
}

func (pkg *Package) BuildPath(projectDir string) string {
	return zbcontext.BuildPath(projectDir, pkg.Package)
}

func (pkg *Package) InstallPath() string {
	return zbcontext.InstallPath(pkg.Package)
}

// BuildTarget returns the absolute path of the binary that this package
// generates when it is built
func (pkg *Package) BuildTarget(projectDir string, gitCommit *core.Hash) *dependency.GoPackage {
	if !pkg.IsCommand() {
		return pkg.InstallTarget(projectDir, gitCommit)
	}

	if projectDir == "" {
		projectDir = pkg.Dir
	}

	return &dependency.GoPackage{
		ProjectImportPath: pkg.DirToImportPath(projectDir),
		Path:              pkg.BuildPath(projectDir),
		Package:           pkg.Package,
		Context:           pkg.Context,
		GitCommit:         gitCommit,
	}
}

func (pkg *Package) InstallTarget(projectDir string, gitCommit *core.Hash) *dependency.GoPackage {
	if projectDir == "" {
		projectDir = pkg.Dir
	}

	return &dependency.GoPackage{
		ProjectImportPath: pkg.DirToImportPath(projectDir),
		Path:              pkg.InstallPath(),
		Package:           pkg.Package,
		Context:           pkg.Context,
		GitCommit:         gitCommit,
	}
}

func (pkg *Package) Targets(tt dependency.TargetType, projectDir string, gitCommit *core.Hash) (*dependency.Targets, error) {
	var fn func(string, *core.Hash) *dependency.GoPackage

	switch tt {
	case dependency.TargetBuild, dependency.TargetGenerate:
		fn = pkg.BuildTarget
	case dependency.TargetInstall:
		fn = pkg.InstallTarget
	default:
		panic(errors.New("unknown TargetType"))
	}

	if projectDir == "" {
		projectDir = pkg.Dir
	}

	gopkg := fn(projectDir, gitCommit)

	queue := []*dependency.Target{dependency.NewTarget(gopkg, nil)}
	unique := dependency.Targets{}

	// recursively add all dependencies
	for len(queue) > 0 {
		// pop the queue
		target := queue[0]
		queue = queue[1:]

		if !unique.Insert(target) {
			continue
		}

		deps, err := target.Dependencies()
		if err != nil {
			return nil, err
		}

		// append these dependencies to the queue
		for _, dep := range deps {
			queue = append(queue, dependency.NewTarget(dep, target))
		}
	}

	return &unique, nil
}

func (pkg *Package) Deps() ([]*Package, error) {
	// sorted, recursive

	if pkg.depsBuilt {
		return pkg.deps, nil
	}

	pkg.depsBuilt = true

	pkg.depMap = map[string]*Package{}
	pkg.depMap[pkg.ImportPath] = pkg

	queue := []string{pkg.ImportPath}

	for len(queue) > 0 {
		path := queue[0]
		queue = queue[1:]

		p, ok := pkg.depMap[path]
		if !ok {
			return nil, errors.Errorf("error loading package: %s", path)
		}

		var toImport []string
		toImport = append(toImport, p.Imports...)
		if p.includeTestImports {
			toImport = append(toImport, p.TestImports...)
			toImport = append(toImport, p.XTestImports...)
		}

		for _, path := range toImport {
			if path == "C" {
				continue
			}

			dep, err := NewPackage(p.Context, path, p.Package.Dir, false)
			if err != nil {
				return nil, errors.Wrapf(err, "error importing package: %s", path)
			}

			if _, ok := pkg.depMap[dep.ImportPath]; ok {
				continue
			}

			pkg.depMap[dep.ImportPath] = dep
			queue = append(queue, dep.ImportPath)
			pkg.deps = append(pkg.deps, dep)
		}
	}

	sort.Sort(&pkg.deps)
	return pkg.deps, nil
}

const cycle = "cycle"

func (pkg *Package) LintHash(flag *lintflags.Data) (string, error) {
	if pkg.lintHash != "" {
		return pkg.lintHash, nil
	}

	pkg.lintHash = cycle

	h := sha1.New()
	fmt.Fprintf(h, "lint\n")

	for _, arg := range flag.LintArgs() {
		fmt.Fprintf(h, "%s\n", arg)
	}

	// don't check dependencies when hashing for lint as lint checks the source
	// of the package, not if any of its dependencies have changed

	var files []string

	files = append(files, pkg.GoFiles...)
	files = append(files, pkg.CgoFiles...)
	files = append(files, pkg.CFiles...)
	files = append(files, pkg.CXXFiles...)
	files = append(files, pkg.MFiles...)
	files = append(files, pkg.HFiles...)
	files = append(files, pkg.SFiles...)
	files = append(files, pkg.SwigFiles...)
	files = append(files, pkg.SwigCXXFiles...)
	files = append(files, pkg.SysoFiles...)

	if !flag.NoTests {
		files = append(files, pkg.TestGoFiles...)
		files = append(files, pkg.XTestGoFiles...)
	}

	if err := hashFiles(h, pkg.Package.Dir, files); err != nil {
		return "", err
	}

	pkg.lintHash = fmt.Sprintf("%x", h.Sum(nil))
	return pkg.lintHash, nil
}

func (pkg *Package) TestHash(flag *buildflags.TestFlagsData) (string, error) {
	if pkg.testHash != "" {
		return pkg.testHash, nil
	}

	pkg.testHash = cycle

	h := sha1.New()
	fmt.Fprintf(h, "test\n")

	if flag.Race {
		fmt.Fprintf(h, "-race\n")
	}

	if flag.Short {
		fmt.Fprintf(h, "-short\n")
	}

	if flag.V || flag.Data.V {
		fmt.Fprintf(h, "-v\n")
	}

	pkgHash, err := pkg.Hash()
	if err != nil {
		return "", err
	}
	fmt.Fprintf(h, "pkg %s\n", pkgHash)

	imports := map[string][]string{
		"testimport":  pkg.TestImports,
		"xtestimport": pkg.XTestImports,
	}

	for name, imps := range imports {
		for _, imp := range imps {
			p1 := pkg.depMap[imp]
			hash, err := p1.Hash()
			if err != nil {
				return "", err
			}
			fmt.Fprintf(h, "%s %s\n", name, hash)
		}
	}

	var files []string
	files = append(files, pkg.TestGoFiles...)
	files = append(files, pkg.XTestGoFiles...)

	if err := hashFiles(h, pkg.Package.Dir, files); err != nil {
		return "", err
	}

	pkg.testHash = fmt.Sprintf("%x", h.Sum(nil))
	return pkg.testHash, nil
}

func (pkg *Package) Hash() (string, error) {
	if pkg.pkgHash != "" {
		return pkg.pkgHash, nil
	}

	pkg.pkgHash = cycle

	deps, err := pkg.Deps()
	if err != nil {
		return "", err
	}

	h := sha1.New()

	fmt.Fprintf(h, "pkg\n")

	for _, p1 := range deps {
		hash, err := p1.Hash()
		if err != nil {
			return "", err
		}
		fmt.Fprintf(h, "import %s\n", hash)
	}

	var files []string
	files = append(files, pkg.GoFiles...)
	files = append(files, pkg.CgoFiles...)
	files = append(files, pkg.CFiles...)
	files = append(files, pkg.CXXFiles...)
	files = append(files, pkg.MFiles...)
	files = append(files, pkg.HFiles...)
	files = append(files, pkg.SFiles...)
	files = append(files, pkg.SwigFiles...)
	files = append(files, pkg.SwigCXXFiles...)
	files = append(files, pkg.SysoFiles...)

	if err := hashFiles(h, pkg.Package.Dir, files); err != nil {
		return "", err
	}

	pkg.pkgHash = fmt.Sprintf("%x", h.Sum(nil))
	return pkg.pkgHash, nil
}

func hashFiles(h io.Writer, dir string, files []string) error {
	for _, file := range files {
		f, err := os.Open(filepath.Join(dir, file))
		if err != nil {
			return err
		}

		fmt.Fprintf(h, "file %s\n", file)
		n, err := io.Copy(h, f)
		if err != nil {
			return err
		}

		fmt.Fprintf(h, "%d bytes\n", n)
		if err := f.Close(); err != nil {
			return err
		}
	}
	return nil
}

var cache = map[string]*Package{}

func NewPackage(ctx *zbcontext.Context, importPath, srcDir string, includeTestImports bool) (*Package, error) {
	importPath = ctx.NormalizeImportPath(importPath)

	if pkg, ok := cache[importPath]; ok {
		return pkg, nil
	}

	pkg, err := ctx.Import(importPath, srcDir)
	if err != nil {
		return nil, err
	}

	isVendored := strings.Contains(pkg.ImportPath, "vendor/")

	ret := &Package{
		Context:            ctx,
		Package:            pkg,
		IsVendored:         isVendored,
		includeTestImports: !isVendored && includeTestImports,
	}

	cache[importPath] = ret

	return ret, nil
}

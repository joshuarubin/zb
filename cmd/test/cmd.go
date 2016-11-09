package test

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"fmt"
	"go/build"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/urfave/cli"
	"jrubin.io/zb/cmd"
	"jrubin.io/zb/lib/project"
	"jrubin.io/zb/lib/zbcontext"
)

var (
	// Cmd is the test command
	Cmd cmd.Constructor = &cc{}

	fail  = []byte("FAIL")
	endRE = regexp.MustCompile(`\A(\?|ok|FAIL) ? ? ?\t([^ \t]+)\t([0-9.]+s|\[.*\])\n\z`)
)

const cycle = "cycle"

// NOTE: inspired by, and much of the code from https://github.com/rsc/gt

type Package struct {
	*build.Package

	deps               Packages
	depsBuilt          bool
	includeTestImports bool

	testHash string
	pkgHash  string
}

type Packages []*Package

func (p *Packages) Len() int {
	return len(*p)
}

func (p *Packages) Less(i, j int) bool {
	return (*p)[i].ImportPath < (*p)[j].ImportPath
}

func (p *Packages) Swap(i, j int) {
	(*p)[i], (*p)[j] = (*p)[j], (*p)[i]
}

type cc struct {
	zbcontext.Context
	cacheDir          string
	pkgInfo, pkgCache map[string]*Package
	failed            bool
}

func (cmd *cc) New(_ *cli.App, config *cmd.Config) cli.Command {
	cmd.Logger = config.Logger
	cmd.SrcDir = config.Cwd

	return cli.Command{
		Name:      "test",
		Usage:     "test all of the packages in each of the projects and cache the results",
		ArgsUsage: "[build/test flags] [packages] [test binary flags]",
		Before: func(c *cli.Context) error {
			return cmd.setup()
		},
		Action: func(c *cli.Context) error {
			return cmd.run(c.App.Writer, c.Args()...)
		},
		SkipArgReorder: true,
		Flags: append(cmd.TestFlags(), []cli.Flag{
			cli.BoolFlag{
				Name:        "f",
				Destination: &cmd.Force,
				Usage: `

			treat all test results as uncached, as does the use of any 'go test'
			flag other than -short and -v`,
			},
			cli.BoolFlag{
				Name:        "l",
				Destination: &cmd.List,
				Usage:       "list the uncached tests it would run",
			},
			cli.StringFlag{
				Name:        "cache",
				Destination: &cmd.cacheDir,
				EnvVar:      "CACHE",
				Value:       defaultCacheDir(),
				Usage:       "test results are saved in this directory",
			},
		}...),
	}
}

func defaultCacheDir() string {
	if runtime.GOOS == "darwin" {
		return filepath.Join(os.Getenv("HOME"), "Library", "Caches")
	}

	return filepath.Join(os.Getenv("HOME"), ".cache", "go-test-cache")
}

func (cmd *cc) setup() error {
	if filepath.Base(cmd.cacheDir) != "go-test-cache" {
		cmd.cacheDir = filepath.Join(cmd.cacheDir, "go-test-cache")
	}
	return nil
}

func (cmd *cc) run(w io.Writer, args ...string) error {
	var packageNames, passToTest []string

	for i := 0; i < len(args); i++ {
		if !strings.HasPrefix(args[i], "-") {
			packageNames = append(packageNames, args[i])
			continue
		}

		if packageNames == nil {
			// make non-nil: we have seen the empty package list
			packageNames = []string{}
		}

		passToTest = append(passToTest, args[i])
	}

	projects, err := project.Projects(cmd.Context, packageNames...)
	if err != nil {
		return err
	}

	cmd.pkgInfo = map[string]*Package{}
	cmd.pkgCache = map[string]*Package{}
	var queue []string

	for _, proj := range projects {
		for _, pkg := range proj.Packages {
			if _, ok := cmd.pkgInfo[pkg.Package.ImportPath]; ok {
				continue
			}

			t := &Package{
				Package:            pkg.Package,
				includeTestImports: !pkg.IsVendored,
			}
			if !pkg.IsVendored {
				cmd.pkgInfo[pkg.Package.ImportPath] = t
			}
			cmd.pkgCache[pkg.Package.ImportPath] = t

			queue = append(queue, pkg.Package.ImportPath)
		}
	}

	for len(queue) > 0 {
		path := queue[0]
		queue = queue[1:]

		pkg, ok := cmd.pkgCache[path]
		if !ok {
			return errors.Errorf("error loading package: %s", path)
		}

		var toImport []string
		toImport = append(toImport, pkg.Imports...)

		if pkg.includeTestImports {
			toImport = append(toImport, pkg.TestImports...)
			toImport = append(toImport, pkg.XTestImports...)
		}

		for _, imp := range toImport {
			if imp == "C" {
				continue
			}

			if dep, ok := cmd.pkgCache[imp]; ok {
				pkg.deps = append(pkg.deps, dep)
				continue
			}

			dep, err := cmd.Import(imp, pkg.Dir)
			if err != nil {
				return errors.Wrapf(err, "error importing package: %s", imp)
			}

			dp := &Package{Package: dep}

			cmd.pkgCache[dep.ImportPath] = dp
			queue = append(queue, dep.ImportPath)
			pkg.deps = append(pkg.deps, dp)
		}

		sort.Sort(&pkg.deps)
	}

	pkgs := make([]string, len(cmd.pkgInfo))
	i := 0
	for pkg := range cmd.pkgInfo {
		pkgs[i] = pkg
		i++
	}
	sort.Strings(pkgs)

	if err := cmd.computeStale(pkgs); err != nil {
		return err
	}

	var toRun []string
	for _, pkg := range pkgs {
		if !cmd.haveTestResult(pkg) {
			toRun = append(toRun, pkg)
		}
	}

	if cmd.List {
		for _, pkg := range toRun {
			// TODO(jrubin) logger?
			p := cmd.pkgInfo[pkg]
			fmt.Fprintf(w, "%s (%s)\n", pkg, cmd.cacheFile(p))
		}
		return nil
	}

	var ecmd *exec.Cmd
	pr, pw := io.Pipe()
	r := bufio.NewReader(pr)
	if len(toRun) > 0 {
		if err := os.MkdirAll(cmd.cacheDir, 0700); err != nil {
			log.Fatal(err) // TODO(jrubin)
		}

		args := []string{"test"}
		args = append(args, cmd.TestArgs(nil, nil)...)
		args = append(args, toRun...)

		// TODO(jrubin) logger the command
		ecmd = exec.Command("go", args...)
		ecmd.Stdout = pw
		ecmd.Stderr = pw
		if err := ecmd.Start(); err != nil {
			log.Fatalf("go test: %v", err) // TODO(jrubin)
		}
	}

	// TODO(jrubin)
	var cmdErr error
	done := make(chan bool)
	go func() {
		if ecmd != nil {
			cmdErr = ecmd.Wait()
		}
		pw.Close()
		done <- true
	}()

	for _, pkg := range pkgs {
		if len(toRun) > 0 && toRun[0] == pkg {
			cmd.readTestResult(r, pkg)
			toRun = toRun[1:]
		} else {
			cmd.showTestResult(w, pkg)
		}
	}

	io.Copy(os.Stdout, r)

	<-done
	if cmdErr != nil && !cmd.failed {
		log.Fatalf("go test: %v", cmdErr) // TODO(jrubin)
	}

	if cmd.failed {
		os.Exit(1)
	}

	return nil
}

func (cmd *cc) computeStale(pkgs []string) error {
	for _, p := range pkgs {
		pkg := cmd.pkgInfo[p]
		if err := cmd.computeTestHash(pkg); err != nil {
			return err
		}
	}
	return nil
}

func (cmd *cc) computeTestHash(p *Package) error {
	if p.testHash != "" {
		return nil
	}

	// TODO(jrubin) clean this up

	p.testHash = cycle
	cmd.computePkgHash(p)
	h := sha1.New()
	fmt.Fprintf(h, "test\n")
	if cmd.Race {
		fmt.Fprintf(h, "-race\n")
	}
	if cmd.Short {
		fmt.Fprintf(h, "-short\n")
	}
	if cmd.V || cmd.BuildFlagsData.V {
		fmt.Fprintf(h, "-v\n")
	}
	fmt.Fprintf(h, "pkg %s\n", p.pkgHash)
	for _, imp := range p.TestImports {
		p1 := cmd.pkgCache[imp]
		cmd.computePkgHash(p1)
		fmt.Fprintf(h, "testimport %s\n", p1.pkgHash)
	}
	for _, imp := range p.XTestImports {
		p1 := cmd.pkgCache[imp]
		cmd.computePkgHash(p1)
		fmt.Fprintf(h, "xtestimport %s\n", p1.pkgHash)
	}
	if err := hashFiles(h, p.Dir, p.TestGoFiles); err != nil {
		return err
	}
	if err := hashFiles(h, p.Dir, p.XTestGoFiles); err != nil {
		return err
	}
	p.testHash = fmt.Sprintf("%x", h.Sum(nil))
	return nil
}

func (p *Package) Deps() []*Package {
	if p.depsBuilt {
		return p.deps
	}

	p.depsBuilt = true

	deps := map[string]*Package{}
	queue := []*Package{p}

	for len(queue) > 0 {
		pkg := queue[0]
		queue = queue[1:]

		for _, dep := range pkg.deps {
			if _, ok := deps[dep.ImportPath]; ok {
				continue
			}

			deps[dep.ImportPath] = dep
			queue = append(queue, dep)
		}
	}

	ret := make(Packages, len(deps))
	i := 0
	for _, dep := range deps {
		ret[i] = dep
		i++
	}
	sort.Sort(&ret)

	p.deps = ret
	return ret
}

func (cmd *cc) computePkgHash(p *Package) error {
	// TODO(jrubin) clean this up

	if p.pkgHash != "" {
		return nil
	}

	p.pkgHash = cycle
	h := sha1.New()
	fmt.Fprintf(h, "pkg\n")
	for _, p1 := range p.Deps() {
		cmd.computePkgHash(p1)
		fmt.Fprintf(h, "import %s\n", p1.pkgHash)
	}

	var files []string
	files = append(files, p.GoFiles...)
	files = append(files, p.CgoFiles...)
	files = append(files, p.CFiles...)
	files = append(files, p.CXXFiles...)
	files = append(files, p.MFiles...)
	files = append(files, p.HFiles...)
	files = append(files, p.SFiles...)
	files = append(files, p.SwigFiles...)
	files = append(files, p.SwigCXXFiles...)
	files = append(files, p.SysoFiles...)

	if err := hashFiles(h, p.Dir, files); err != nil {
		return err
	}

	p.pkgHash = fmt.Sprintf("%x", h.Sum(nil))

	return nil
}

func hashFiles(h io.Writer, dir string, files []string) error {
	// TODO(jrubin) clean this up
	for _, file := range files {
		f, err := os.Open(filepath.Join(dir, file))
		if err != nil {
			fmt.Fprintf(h, "%s error\n", file)
			continue
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

func (cmd *cc) haveTestResult(path string) bool {
	// TODO(jrubin) clean this up

	if cmd.Force {
		return false
	}

	p := cmd.pkgInfo[path]
	if p.testHash == cycle {
		return false
	}

	fi, err := os.Stat(cmd.cacheFile(p))
	return err == nil && fi.Mode().IsRegular()
}

func (cmd *cc) readTestResult(r *bufio.Reader, path string) {
	// TODO(jrubin) clean this up
	// scanner?

	var buf bytes.Buffer
	for {
		line, err := r.ReadString('\n')
		os.Stdout.WriteString(line)
		if err != nil {
			log.Fatalf("reading test output for %s: %v", path, err) // TODO(jrubin)
		}
		m := endRE.FindStringSubmatch(line)
		if m == nil {
			buf.WriteString(line)
			continue
		}

		if m[1] == "FAIL" {
			cmd.failed = true
		}

		fmt.Fprintf(&buf, "%s (cached)\n", strings.TrimSuffix(line, "\n"))
		file := cmd.cacheFile(cmd.pkgInfo[path])
		if err := os.MkdirAll(filepath.Dir(file), 0700); err != nil {
			log.Print(err) // TODO(jrubin)
		} else if err := ioutil.WriteFile(file, buf.Bytes(), 0600); err != nil {
			log.Print(err) // TODO(jrubin)
		}

		break
	}
}

func (cmd *cc) cacheFile(p *Package) string {
	return filepath.Join(cmd.cacheDir, p.testHash[:3], fmt.Sprintf("%s.test", p.testHash[3:]))
}

func (cmd *cc) showTestResult(w io.Writer, path string) {
	// TODO(jrubin) clean this up

	p := cmd.pkgInfo[path]
	if p.testHash == "cycle" {
		return
	}
	data, err := ioutil.ReadFile(cmd.cacheFile(p))
	if err != nil {
		fmt.Fprintf(w, "%v\n", err)
		fmt.Fprintf(w, "FAIL\t%s\t(cached)\n", path)
		return
	}
	os.Stdout.Write(data)
	data = bytes.TrimSpace(data)
	i := bytes.LastIndex(data, []byte{'\n'})
	line := data[i+1:]
	if bytes.HasPrefix(line, fail) {
		cmd.failed = true
	}
}

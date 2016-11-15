package buildflags

import (
	"fmt"
	"go/build"
	"runtime"
	"strings"
	"time"

	"gopkg.in/src-d/go-git.v4/core"

	"github.com/urfave/cli"
)

var (
	defaultP         = runtime.NumCPU()
	defaultBuildmode = "default"
)

// Data are all the flags that are shared by the go build, clean, get, install,
// list, run and test commands
type Data struct {
	A             bool
	N             bool
	P             int
	Race          bool
	Msan          bool
	V             bool
	Work          bool
	X             bool
	AsmFlags      stringsFlag
	BuildMode     string
	Compiler      string
	GCCGoFlags    stringsFlag
	GCFlags       stringsFlag
	InstallSuffix string
	LDFlags       stringsFlag
	LinkShared    bool
	PkgDir        string
	Tags          stringsFlag
	ToolExec      stringsFlag

	context *build.Context
}

const dateFormat = "2006-01-02T15:04:05+00:00"

// BuildArgs returns strings suitable for passing to the go command line
func (f *Data) BuildArgs(pkg *build.Package, gitCommit *core.Hash) []string {
	var args []string

	if f.A {
		args = append(args, "-a")
	}

	if f.N {
		args = append(args, "-n")
	}

	if f.P != 0 && f.P != defaultP {
		args = append(args, "-p", fmt.Sprintf("%d", f.P))
	}

	if f.Race {
		args = append(args, "-race")
	}

	if f.Msan {
		args = append(args, "-msan")
	}

	if f.V {
		args = append(args, "-v")
	}

	if f.Work {
		args = append(args, "-work")
	}

	if f.X {
		args = append(args, "-x")
	}

	if len(f.AsmFlags) > 0 {
		args = append(args, "-asmflags", strings.Join(f.AsmFlags, " "))
	}

	if f.BuildMode != "" && f.BuildMode != defaultBuildmode {
		args = append(args, "-buildmode", f.BuildMode)
	}

	if f.Compiler != "" {
		args = append(args, "-compiler", f.Compiler)
	}

	if len(f.GCCGoFlags) > 0 {
		args = append(args, "-gccgoflags", strings.Join(f.GCCGoFlags, " "))
	}

	if len(f.GCFlags) > 0 {
		args = append(args, "-gcflags", strings.Join(f.GCFlags, " "))
	}

	if f.InstallSuffix != "" {
		args = append(args, "-installsuffix", f.InstallSuffix)
	}

	var ldflags []string

	if pkg != nil && pkg.IsCommand() && gitCommit != nil {
		ldflags = []string{
			fmt.Sprintf("-X main.gitCommit=%s -X main.buildDate=%s",
				*gitCommit,
				time.Now().UTC().Format(dateFormat),
			),
		}
	}

	if len(f.LDFlags) > 0 {
		ldflags = append(ldflags, f.LDFlags...)
	}

	if len(ldflags) > 0 {
		args = append(args, "-ldflags", strings.Join(ldflags, " "))
	}

	if f.LinkShared {
		args = append(args, "-linkshared")
	}

	if f.PkgDir != "" {
		args = append(args, "-pkgdir", f.PkgDir)
	}

	if len(f.Tags) > 0 {
		args = append(args, "-tags", strings.Join(f.Tags, " "))
	}

	if len(f.ToolExec) > 0 {
		args = append(args, "-toolexec", strings.Join(f.ToolExec, " "))
	}

	return args
}

// BuildContext returns a build context based on environment variables GOARCH,
// GOOS, GOROOT, GOPATH, CGO_ENABLED and command line flags
func (f *Data) BuildContext() *build.Context {
	if f.context != nil {
		return f.context
	}

	c := build.Default

	if f.Compiler != "" {
		c.Compiler = f.Compiler
	}

	c.BuildTags = f.Tags
	c.InstallSuffix = f.InstallSuffix

	f.context = &c

	return f.context
}

// BuildFlags returns cli.Flags to use with cli.Command
func (f *Data) BuildFlags() []cli.Flag {
	return []cli.Flag{
		cli.BoolFlag{
			Name:        "a",
			Destination: &f.A,
			Usage: `

			force rebuilding of packages that are already up-to-date.`,
		},
		cli.BoolFlag{
			Name:        "n",
			Destination: &f.N,
			Usage: `
			
			print the commands but do not run them.`,
		},
		cli.IntFlag{
			Name:        "p",
			Value:       defaultP,
			Destination: &f.P,
			Usage: `
			
			the number of programs, such as build commands or test binaries,
			that can be run in parallel. The default is the number of CPUs
			available.`,
		},
		cli.BoolFlag{
			Name:        "race",
			Destination: &f.Race,
			Usage: `

			enable data race detection. Supported only on linux/amd64,
			freebsd/amd64, darwin/amd64 and windows/amd64.`,
		},
		cli.BoolFlag{
			Name:        "msan",
			Destination: &f.Msan,
			Usage: `
			
			enable interoperation with memory sanitizer. Supported only on
			linux/amd64, and only with Clang/LLVM as the host C compiler.`,
		},
		cli.BoolFlag{
			Name:        "v",
			Destination: &f.V,
			Usage: `
			
			print the names of packages as they are compiled.`,
		},
		cli.BoolFlag{
			Name:        "work",
			Destination: &f.Work,
			Usage: `
			
			print the name of the temporary work directory and do not delete it
			when exiting.`,
		},
		cli.BoolFlag{
			Name:        "x",
			Destination: &f.X,
			Usage: `
			
			print the commands.`,
		},
		cli.GenericFlag{
			Name:  "asmflags",
			Value: &f.AsmFlags,
			Usage: `
			
			arguments to pass on each go tool asm invocation.`,
		},
		cli.StringFlag{
			Name:        "buildmode",
			Value:       defaultBuildmode,
			Destination: &f.BuildMode,
			Usage: `
			
			build mode to use. See 'go help buildmode' for more.`,
		},
		cli.StringFlag{
			Name:        "compiler",
			Destination: &f.Compiler,
			Usage: `
			
			name of compiler to use, as in runtime.Compiler (gccgo or gc).`,
		},
		cli.GenericFlag{
			Name:  "gccgoflags",
			Value: &f.GCCGoFlags,
			Usage: `
			
			arguments to pass on each gccgo compiler/linker invocation.`,
		},
		cli.GenericFlag{
			Name:  "gcflags",
			Value: &f.GCFlags,
			Usage: `
			
			arguments to pass on each go tool compile invocation.`,
		},
		cli.StringFlag{
			Name:        "installsuffix",
			Destination: &f.InstallSuffix,
			Usage: `

			a suffix to use in the name of the package installation directory,
			in order to keep output separate from default builds. If using the
			-race flag, the install suffix is automatically set to race or, if
			set explicitly, has _race appended to it.  Likewise for the -msan
			flag.  Using a -buildmode option that requires non-default compile
			flags has a similar effect.`,
		},
		cli.GenericFlag{
			Name:  "ldflags",
			Value: &f.LDFlags,
			Usage: `
			
			arguments to pass on each go tool link invocation.`,
		},
		cli.BoolFlag{
			Name:        "linkshared",
			Destination: &f.LinkShared,
			Usage: `

			link against shared libraries previously created with
			-buildmode=shared.`,
		},
		cli.StringFlag{
			Name:        "pkgdir",
			Destination: &f.PkgDir,
			Usage: `

			install and load all packages from dir instead of the usual
			locations. For example, when building with a non-standard
			configuration, use -pkgdir to keep generated packages in a separate
			location.`,
		},
		cli.GenericFlag{
			Name:  "tags",
			Value: &f.Tags,
			Usage: `

			a list of build tags to consider satisfied during the build. For
			more information about build tags, see the description of build
			constraints in the documentation for the go/build package.`,
		},
		cli.GenericFlag{
			Name:  "toolexec",
			Value: &f.ToolExec,
			Usage: `

			a program to use to invoke toolchain programs like vet and asm. For
			example, instead of running asm, the go command will run 'cmd args
			/path/to/asm <arguments for asm>'.`,
		},
	}
}

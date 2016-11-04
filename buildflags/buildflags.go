package buildflags

import (
	"fmt"
	"go/build"
	"runtime"
	"strings"

	"github.com/urfave/cli"
)

var (
	defaultP         = runtime.NumCPU()
	defaultBuildmode = "default"
)

type BuildFlags struct {
	a             bool
	n             bool
	p             int
	race          bool
	msan          bool
	v             bool
	work          bool
	x             bool
	asmflags      stringsFlag
	buildmode     string
	compiler      string
	gccgoflags    stringsFlag
	gcflags       stringsFlag
	installsuffix string
	ldflags       stringsFlag
	linkshared    bool
	pkgdir        string
	tags          stringsFlag
	toolexec      stringsFlag
}

func (f *BuildFlags) Args() []string {
	var args []string

	if f.a {
		args = append(args, "-a")
	}

	if f.n {
		args = append(args, "-n")
	}

	if f.p != defaultP {
		args = append(args, "-p", fmt.Sprintf("%d", f.p))
	}

	if f.race {
		args = append(args, "-race")
	}

	if f.msan {
		args = append(args, "-msan")
	}

	if f.v {
		args = append(args, "-v")
	}

	if f.work {
		args = append(args, "-work")
	}

	if f.x {
		args = append(args, "-x")
	}

	if len(f.asmflags) > 0 {
		args = append(args, "-asmflags", strings.Join(f.asmflags, " "))
	}

	if f.buildmode != defaultBuildmode {
		args = append(args, "-buildmode", f.buildmode)
	}

	if f.compiler != "" {
		args = append(args, "-compiler", f.compiler)
	}

	if len(f.gccgoflags) > 0 {
		args = append(args, "-gccgoflags", strings.Join(f.gccgoflags, " "))
	}

	if len(f.gcflags) > 0 {
		args = append(args, "-gcflags", strings.Join(f.gcflags, " "))
	}

	if f.installsuffix != "" {
		args = append(args, "-installsuffix", f.installsuffix)
	}

	if len(f.ldflags) > 0 {
		args = append(args, "-ldflags", strings.Join(f.ldflags, " "))
	}

	if f.linkshared {
		args = append(args, "-linkshared")
	}

	if f.pkgdir != "" {
		args = append(args, "-pkgdir", f.pkgdir)
	}

	if len(f.tags) > 0 {
		args = append(args, "-tags", strings.Join(f.tags, " "))
	}

	if len(f.toolexec) > 0 {
		args = append(args, "-toolexec", strings.Join(f.toolexec, " "))
	}

	return args
}

func (f *BuildFlags) BuildContext() build.Context {
	c := build.Default

	if f.compiler != "" {
		c.Compiler = f.compiler
	}

	c.BuildTags = f.tags
	c.InstallSuffix = f.installsuffix

	return c
}

func (f *BuildFlags) Flags() []cli.Flag {
	return []cli.Flag{
		cli.BoolFlag{
			Name:        "a",
			Destination: &f.a,
			Usage: `

			force rebuilding of packages that are already up-to-date.`,
		},
		cli.BoolFlag{
			Name:        "n",
			Destination: &f.n,
			Usage: `
			
			print the commands but do not run them.`,
		},
		cli.IntFlag{
			Name:        "p",
			Value:       defaultP,
			Destination: &f.p,
			Usage: `
			
			the number of programs, such as build commands or test binaries,
			that can be run in parallel. The default is the number of CPUs
			available.`,
		},
		cli.BoolFlag{
			Name:        "race",
			Destination: &f.race,
			Usage: `

			enable data race detection. Supported only on linux/amd64,
			freebsd/amd64, darwin/amd64 and windows/amd64.`,
		},
		cli.BoolFlag{
			Name:        "msan",
			Destination: &f.msan,
			Usage: `
			
			enable interoperation with memory sanitizer. Supported only on
			linux/amd64, and only with Clang/LLVM as the host C compiler.`,
		},
		cli.BoolFlag{
			Name:        "v",
			Destination: &f.v,
			Usage: `
			
			print the names of packages as they are compiled.`,
		},
		cli.BoolFlag{
			Name:        "work",
			Destination: &f.work,
			Usage: `
			
			print the name of the temporary work directory and do not delete it
			when exiting.`,
		},
		cli.BoolFlag{
			Name:        "x",
			Destination: &f.x,
			Usage: `
			
			print the commands.`,
		},
		cli.GenericFlag{
			Name:  "asmflags",
			Value: &f.asmflags,
			Usage: `
			
			arguments to pass on each go tool asm invocation.`,
		},
		cli.StringFlag{
			Name:        "buildmode",
			Value:       defaultBuildmode,
			Destination: &f.buildmode,
			Usage: `
			
			build mode to use. See 'go help buildmode' for more.`,
		},
		cli.StringFlag{
			Name:        "compiler",
			Destination: &f.compiler,
			Usage: `
			
			name of compiler to use, as in runtime.Compiler (gccgo or gc).`,
		},
		cli.GenericFlag{
			Name:  "gccgoflags",
			Value: &f.gccgoflags,
			Usage: `
			
			arguments to pass on each gccgo compiler/linker invocation.`,
		},
		cli.GenericFlag{
			Name:  "gcflags",
			Value: &f.gcflags,
			Usage: `
			
			arguments to pass on each go tool compile invocation.`,
		},
		cli.StringFlag{
			Name:        "installsuffix",
			Destination: &f.installsuffix,
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
			Value: &f.ldflags,
			Usage: `
			
			arguments to pass on each go tool link invocation.`,
		},
		cli.BoolFlag{
			Name:        "linkshared",
			Destination: &f.linkshared,
			Usage: `

			link against shared libraries previously created with
			-buildmode=shared.`,
		},
		cli.StringFlag{
			Name:        "pkgdir",
			Destination: &f.pkgdir,
			Usage: `

			install and load all packages from dir instead of the usual
			locations. For example, when building with a non-standard
			configuration, use -pkgdir to keep generated packages in a separate
			location.`,
		},
		cli.GenericFlag{
			Name:  "tags",
			Value: &f.tags,
			Usage: `

			a list of build tags to consider satisfied during the build. For
			more information about build tags, see the description of build
			constraints in the documentation for the go/build package.`,
		},
		cli.GenericFlag{
			Name:  "toolexec",
			Value: &f.toolexec,
			Usage: `

			a program to use to invoke toolchain programs like vet and asm. For
			example, instead of running asm, the go command will run 'cmd args
			/path/to/asm <arguments for asm>'.`,
		},
	}
}

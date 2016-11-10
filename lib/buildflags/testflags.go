package buildflags

import (
	"fmt"
	"go/build"
	"os"
	"runtime"
	"strings"
	"time"

	"gopkg.in/src-d/go-git.v4/core"

	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

type TestFlagsData struct {
	Data

	C                bool
	Exec             stringsFlag
	I                bool
	O                string
	Bench            string
	BenchMem         bool
	BenchTime        time.Duration
	BlockProfile     string
	BlockProfileRate int
	Count            int
	Cover            bool
	CoverMode        string
	CoverPkg         string
	CoverProfile     string
	CPU              string
	CPUProfile       string
	MemProfile       string
	MemProfileRate   int
	OutputDir        string
	Parallel         int
	Run              string
	Short            bool
	Timeout          time.Duration
	Trace            string
	V                bool
}

var (
	defaultBenchTime = 1 * time.Second
	defaultCount     = 1
	defaultCPU       = fmt.Sprintf("%d", runtime.GOMAXPROCS(-1))
	defaultOutputDir string
	defaultParallel  = runtime.GOMAXPROCS(-1)
	defaultTimeout   = 10 * time.Minute
)

func init() {
	var err error
	defaultOutputDir, err = os.Getwd()
	if err != nil {
		panic(err)
	}
}

func (f *TestFlagsData) TestFlags() []cli.Flag {
	flags := []cli.Flag{
		cli.BoolFlag{
			Name:        "c",
			Destination: &f.C,
			Usage: `

			Compile the test binary to pkg.test but do not run it (where pkg is
			the last element of the package's import path). The file name can be
			changed with the -o flag.`,
		},
		cli.GenericFlag{
			Name:  "exec",
			Value: &f.Exec,
			Usage: `

			Run the test binary using xprog. The behavior is the same as in 'go
			run'. See 'go help run' for details.`,
		},
		cli.BoolFlag{
			Name:        "i",
			Destination: &f.I,
			Usage: `

			Install packages that are dependencies of the test. Do not run the
			test.`,
		},
		cli.StringFlag{
			Name:        "o",
			Destination: &f.O,
			Usage: `

			Compile the test binary to the named file. The test still runs
			(unless -c or -i is specified).`,
		},
		cli.StringFlag{
			Name:        "bench",
			Destination: &f.Bench,
			Usage: `

			Run (sub)benchmarks matching a regular expression. The given regular
			expression is split into smaller ones by top-level '/', where each
			must match the corresponding part of a benchmark's identifier. By
			default, no benchmarks run. To run all benchmarks, use '-bench .' or
			'-bench=.'.`,
		},
		cli.BoolFlag{
			Name:        "benchmem",
			Destination: &f.BenchMem,
			Usage: `

			Print memory allocation statistics for benchmarks.`,
		},
		cli.DurationFlag{
			Name:        "benchtime",
			Destination: &f.BenchTime,
			Value:       defaultBenchTime,
			Usage: `

			Run enough iterations of each benchmark to take t, specified as a
			time.Duration (for example, -benchtime 1h30s).`,
		},
		cli.StringFlag{
			Name:        "blockprofile",
			Destination: &f.BlockProfile,
			Usage: `

			Write a goroutine blocking profile to the specified file when all
			tests are complete. Writes test binary as -c would.`,
		},
		cli.IntFlag{
			Name:        "blockprofilerate",
			Destination: &f.BlockProfileRate,
			Usage: `

			Control the detail provided in goroutine blocking profiles by
			calling runtime.SetBlockProfileRate with n. See 'go doc
			runtime.SetBlockProfileRate'. The profiler aims to sample, on
			average, one blocking event every n nanoseconds the program spends
			blocked.  By default, if -test.blockprofile is set without this
			flag, all blocking events are recorded, equivalent to
			-test.blockprofilerate=1.`,
		},
		cli.IntFlag{
			Name:        "count",
			Destination: &f.Count,
			Value:       defaultCount,
			Usage: `

			Run each test and benchmark n times. If -cpu is set, run n times for
			each GOMAXPROCS value. Examples are always run once.`,
		},
		cli.BoolFlag{
			Name:        "cover",
			Destination: &f.Cover,
			Usage: `

			Enable coverage analysis.`,
		},
		cli.StringFlag{
			Name:        "covermode",
			Destination: &f.CoverMode,
			Usage: `

			Set the mode for coverage analysis for the package[s] being tested.
			The default is "set" unless -race is enabled, in which case it is
			"atomic".
			The values:
			set:    bool: does this statement run?
			count:  int: how many times does this statement run?
			atomic: int: count, but correct in multithreaded tests;
			        significantly more expensive.
			Sets -cover.`,
		},
		cli.StringFlag{
			Name:        "coverpkg",
			Destination: &f.CoverPkg,
			Usage: `

			Apply coverage analysis in each test to the given list of packages.
			The default is for each test to analyze only the package being
			tested. Packages are specified as import paths. Sets -cover.`,
		},
		cli.StringFlag{
			Name:        "coverprofile",
			Destination: &f.CoverProfile,
			Usage: `

			Write a coverage profile to the file after all tests have passed.
			Sets -cover.`,
		},
		cli.StringFlag{
			Name:        "cpu",
			Destination: &f.CPU,
			Value:       defaultCPU,
			Usage: `

			Specify a list of GOMAXPROCS values for which the tests or
			benchmarks should be executed.  The default is the current value of
			GOMAXPROCS.`,
		},
		cli.StringFlag{
			Name:        "cpuprofile",
			Destination: &f.CPUProfile,
			Usage: `

			Write a CPU profile to the specified file before exiting. Writes
			test binary as -c would.`,
		},
		cli.StringFlag{
			Name:        "memprofile",
			Destination: &f.MemProfile,
			Usage: `

			Write a memory profile to the file after all tests have passed.
			Writes test binary as -c would.`,
		},
		cli.IntFlag{
			Name:        "memprofilerate",
			Destination: &f.MemProfileRate,
			Usage: `

			Enable more precise (and expensive) memory profiles by setting
			runtime.MemProfileRate.  See 'go doc runtime.MemProfileRate'. To
			profile all memory allocations, use -test.memprofilerate=1 and pass
			--alloc_space flag to the pprof tool.`,
		},
		cli.StringFlag{
			Name:        "outputdir",
			Destination: &f.OutputDir,
			Value:       defaultOutputDir,
			Usage: `

			Place output files from profiling in the specified directory, by
			default the directory in which "go test" is running.`,
		},
		cli.IntFlag{
			Name:        "parallel",
			Destination: &f.Parallel,
			Value:       defaultParallel,
			Usage: `

			Allow parallel execution of test functions that call t.Parallel. The
			value of this flag is the maximum number of tests to run
			simultaneously; by default, it is set to the value of GOMAXPROCS.
			Note that -parallel only applies within a single test binary. The
			'go test' command may run tests for different packages in parallel
			as well, according to the setting of the -p flag (see 'go help
			build').`,
		},
		cli.StringFlag{
			Name:        "run",
			Destination: &f.Run,
			Usage: `

			Run only those tests and examples matching the regular expression.
			For tests the regular expression is split into smaller ones by
			top-level '/', where each must match the corresponding part of a
			test's identifier.`,
		},
		cli.BoolFlag{
			Name:        "short",
			Destination: &f.Short,
			Usage: `

			Tell long-running tests to shorten their run time.
			It is off by default but set during all.bash so that installing
			the Go tree can run a sanity check but not spend time running
			exhaustive tests.`,
		},
		cli.DurationFlag{
			Name:        "timeout",
			Destination: &f.Timeout,
			Value:       defaultTimeout,
			Usage: `

			If a test runs longer than t, panic.`,
		},
		cli.StringFlag{
			Name:        "trace",
			Destination: &f.Trace,
			Usage: `

			Write an execution trace to the specified file before exiting.`,
		},
	}

	l := len(flags)
	for i := 0; i < l; i++ {
		switch f := flags[i].(type) {
		case cli.BoolFlag:
			flags = append(flags, cli.BoolFlag{
				Name:        "test." + f.Name,
				Destination: f.Destination,
				Hidden:      true,
			})
		case cli.StringFlag:
			flags = append(flags, cli.StringFlag{
				Name:        "test." + f.Name,
				Destination: f.Destination,
				Hidden:      true,
			})
		case cli.DurationFlag:
			flags = append(flags, cli.DurationFlag{
				Name:        "test." + f.Name,
				Destination: f.Destination,
				Hidden:      true,
			})
		case cli.IntFlag:
			flags = append(flags, cli.IntFlag{
				Name:        "test." + f.Name,
				Destination: f.Destination,
				Hidden:      true,
			})
		case cli.GenericFlag:
			flags = append(flags, cli.GenericFlag{
				Name:   "test." + f.Name,
				Value:  f.Value,
				Usage:  f.Usage,
				Hidden: true,
			})
		default:
			panic(errors.Errorf("invalid flag type: %T", f))
		}
	}

	flags = append(flags, cli.BoolFlag{
		Name:        "test.v",
		Destination: &f.V,
		Hidden:      true,
	})

	return append(f.BuildFlags(), flags...)
}

func (f *TestFlagsData) TestArgs(pkg *build.Package, gitCommit *core.Hash) []string {
	args := f.BuildArgs(pkg, gitCommit)

	if f.C {
		args = append(args, "-c")
	}

	if len(f.Exec) > 0 {
		args = append(args, "-exec", strings.Join(f.Exec, " "))
	}

	if f.I {
		args = append(args, "-i")
	}

	if f.O != "" {
		args = append(args, "-o", f.O)
	}

	if f.Bench != "" {
		args = append(args, "-bench", f.Bench)
	}

	if f.BenchMem {
		args = append(args, "-benchmem")
	}

	if f.BenchTime != 0 && f.BenchTime != defaultBenchTime {
		args = append(args, "-benchtime", f.BenchTime.String())
	}

	if f.BlockProfile != "" {
		args = append(args, "-blockprofile", f.BlockProfile)
	}

	if f.BlockProfileRate > 0 {
		args = append(args, "-blockprofilerate", fmt.Sprintf("%d", f.BlockProfileRate))
	}

	if f.Count != 0 && f.Count != defaultCount {
		args = append(args, "-count", fmt.Sprintf("%d", f.Count))
	}

	if f.Cover {
		args = append(args, "-cover")
	}

	if f.CoverMode != "" {
		args = append(args, "-covermode", f.CoverMode)
	}

	if f.CoverPkg != "" {
		args = append(args, "-coverpkg", f.CoverPkg)
	}

	if f.CoverProfile != "" {
		args = append(args, "-coverprofile", f.CoverProfile)
	}

	if f.CPU != "" && f.CPU != defaultCPU {
		args = append(args, "-cpu", f.CPU)
	}

	if f.CPUProfile != "" {
		args = append(args, "-cpuprofile", f.CPUProfile)
	}

	if f.MemProfile != "" {
		args = append(args, "-memprofile", f.MemProfile)
	}

	if f.MemProfileRate > 0 {
		args = append(args, "-memprofilerate", fmt.Sprintf("%d", f.MemProfileRate))
	}

	if f.OutputDir != "" && f.OutputDir != defaultOutputDir {
		args = append(args, "-outputdir", f.OutputDir)
	}

	if f.Parallel != 0 && f.Parallel != defaultParallel {
		args = append(args, "-parallel", fmt.Sprintf("%d", f.Parallel))
	}

	if f.Run != "" {
		args = append(args, "-run", f.Run)
	}

	if f.Short {
		args = append(args, "-short")
	}

	if f.Timeout != 0 && f.Timeout != defaultTimeout {
		args = append(args, "-timeout", f.Timeout.String())
	}

	if f.Trace != "" {
		args = append(args, "-trace", f.Trace)
	}

	if !f.Data.V && f.V {
		args = append(args, "-test.v")
	}

	return args
}

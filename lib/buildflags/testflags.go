package buildflags

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

type TestFlags struct {
	C                bool
	Exec             string
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

// TODO(jrubin) default values

func (f *TestFlags) Flags() []cli.Flag {
	flags := []cli.Flag{
		cli.BoolFlag{
			Name:        "c",
			Destination: &f.C,
			Usage: `

			Compile the test binary to pkg.test but do not run it (where pkg is
			the last element of the package's import path). The file name can be
			changed with the -o flag.`,
		},
		cli.StringFlag{
			Name:        "exec",
			Destination: &f.Exec,
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
			// TODO(jrubin)
		},
		cli.BoolFlag{
			Name:        "benchmem",
			Destination: &f.BenchMem,
			// TODO(jrubin)
		},
		cli.DurationFlag{
			Name:        "benchtime",
			Destination: &f.BenchTime,
			// TODO(jrubin)
		},
		cli.StringFlag{
			Name:        "blockprofile",
			Destination: &f.BlockProfile,
			// TODO(jrubin)
		},
		cli.IntFlag{
			Name:        "blockprofilerate",
			Destination: &f.BlockProfileRate,
			// TODO(jrubin)
		},
		cli.IntFlag{
			Name:        "count",
			Destination: &f.Count,
			// TODO(jrubin)
		},
		cli.BoolFlag{
			Name:        "cover",
			Destination: &f.Cover,
			// TODO(jrubin)
		},
		cli.StringFlag{
			Name:        "covermode",
			Destination: &f.CoverMode,
			// TODO(jrubin)
		},
		cli.StringFlag{
			Name:        "coverpkg",
			Destination: &f.CoverPkg,
			// TODO(jrubin)
		},
		cli.StringFlag{
			Name:        "coverprofile",
			Destination: &f.CoverProfile,
			// TODO(jrubin)
		},
		cli.StringFlag{
			Name:        "cpu",
			Destination: &f.CPU,
			// TODO(jrubin)
		},
		cli.StringFlag{
			Name:        "cpuprofile",
			Destination: &f.CPUProfile,
			// TODO(jrubin)
		},
		cli.StringFlag{
			Name:        "memprofile",
			Destination: &f.MemProfile,
			// TODO(jrubin)
		},
		cli.IntFlag{
			Name:        "memprofilerate",
			Destination: &f.MemProfileRate,
			// TODO(jrubin)
		},
		cli.StringFlag{
			Name:        "outputdir",
			Destination: &f.OutputDir,
			// TODO(jrubin)
		},
		cli.IntFlag{
			Name:        "parallel",
			Destination: &f.Parallel,
			// TODO(jrubin)
		},
		cli.StringFlag{
			Name:        "run",
			Destination: &f.Run,
			// TODO(jrubin)
		},
		cli.BoolFlag{
			Name:        "short",
			Destination: &f.Short,
			// TODO(jrubin)
		},
		cli.DurationFlag{
			Name:        "timeout",
			Destination: &f.Timeout,
			// TODO(jrubin)
		},
		cli.StringFlag{
			Name:        "trace",
			Destination: &f.Trace,
			// TODO(jrubin)
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
		default:
			panic(errors.Errorf("invalid flag type: %T", f))
		}
	}

	flags = append(flags, cli.BoolFlag{
		Name:        "test.v",
		Destination: &f.V,
		Hidden:      true,
	})

	return flags
}

/*
	-bench regexp
	    Run (sub)benchmarks matching a regular expression.
	    The given regular expression is split into smaller ones by
	    top-level '/', where each must match the corresponding part of a
	    benchmark's identifier.
	    By default, no benchmarks run. To run all benchmarks,
	    use '-bench .' or '-bench=.'.

	-benchmem
	    Print memory allocation statistics for benchmarks.

	-benchtime t
	    Run enough iterations of each benchmark to take t, specified
	    as a time.Duration (for example, -benchtime 1h30s).
	    The default is 1 second (1s).

	-blockprofile block.out
	    Write a goroutine blocking profile to the specified file
	    when all tests are complete.
	    Writes test binary as -c would.

	-blockprofilerate n
	    Control the detail provided in goroutine blocking profiles by
	    calling runtime.SetBlockProfileRate with n.
	    See 'go doc runtime.SetBlockProfileRate'.
	    The profiler aims to sample, on average, one blocking event every
	    n nanoseconds the program spends blocked.  By default,
	    if -test.blockprofile is set without this flag, all blocking events
	    are recorded, equivalent to -test.blockprofilerate=1.

	-count n
	    Run each test and benchmark n times (default 1).
	    If -cpu is set, run n times for each GOMAXPROCS value.
	    Examples are always run once.

	-cover
	    Enable coverage analysis.

	-covermode set,count,atomic
	    Set the mode for coverage analysis for the package[s]
	    being tested. The default is "set" unless -race is enabled,
	    in which case it is "atomic".
	    The values:
		set: bool: does this statement run?
		count: int: how many times does this statement run?
		atomic: int: count, but correct in multithreaded tests;
			significantly more expensive.
	    Sets -cover.

	-coverpkg pkg1,pkg2,pkg3
	    Apply coverage analysis in each test to the given list of packages.
	    The default is for each test to analyze only the package being tested.
	    Packages are specified as import paths.
	    Sets -cover.

	-coverprofile cover.out
	    Write a coverage profile to the file after all tests have passed.
	    Sets -cover.

	-cpu 1,2,4
	    Specify a list of GOMAXPROCS values for which the tests or
	    benchmarks should be executed.  The default is the current value
	    of GOMAXPROCS.

	-cpuprofile cpu.out
	    Write a CPU profile to the specified file before exiting.
	    Writes test binary as -c would.

	-memprofile mem.out
	    Write a memory profile to the file after all tests have passed.
	    Writes test binary as -c would.

	-memprofilerate n
	    Enable more precise (and expensive) memory profiles by setting
	    runtime.MemProfileRate.  See 'go doc runtime.MemProfileRate'.
	    To profile all memory allocations, use -test.memprofilerate=1
	    and pass --alloc_space flag to the pprof tool.

	-outputdir directory
	    Place output files from profiling in the specified directory,
	    by default the directory in which "go test" is running.

	-parallel n
	    Allow parallel execution of test functions that call t.Parallel.
	    The value of this flag is the maximum number of tests to run
	    simultaneously; by default, it is set to the value of GOMAXPROCS.
	    Note that -parallel only applies within a single test binary.
	    The 'go test' command may run tests for different packages
	    in parallel as well, according to the setting of the -p flag
	    (see 'go help build').

	-run regexp
	    Run only those tests and examples matching the regular expression.
	    For tests the regular expression is split into smaller ones by
	    top-level '/', where each must match the corresponding part of a
	    test's identifier.

	-short
	    Tell long-running tests to shorten their run time.
	    It is off by default but set during all.bash so that installing
	    the Go tree can run a sanity check but not spend time running
	    exhaustive tests.

	-timeout t
	    If a test runs longer than t, panic.
	    The default is 10 minutes (10m).

	-trace trace.out
	    Write an execution trace to the specified file before exiting.

	-v
	    Verbose output: log all tests as they are run. Also print all
	    text from Log and Logf calls even if the test succeeds.
*/

func (f *TestFlags) TestArgs() []string {
	var args []string

	if f.C {
		args = append(args, "-c")
	}

	if f.Exec != "" {
		args = append(args, "-exec", f.Exec)
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

	if f.BenchTime > 0 {
		args = append(args, "-benchtime", f.BenchTime.String())
	}

	if f.BlockProfile != "" {
		args = append(args, "-blockprofile", f.BlockProfile)
	}

	if f.BlockProfileRate > 0 {
		args = append(args, "-blockprofilerate", fmt.Sprintf("%d", f.BlockProfileRate))
	}

	if f.Count != 0 {
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

	if f.CPU != "" {
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

	if f.OutputDir != "" {
		args = append(args, "-outputdir", f.OutputDir)
	}

	if f.Parallel > 0 {
		args = append(args, "-parallel", fmt.Sprintf("%d", f.Parallel))
	}

	if f.Run != "" {
		args = append(args, "-run", f.Run)
	}

	if f.Short {
		args = append(args, "-short")
	}

	if f.Timeout > 0 {
		args = append(args, "-timeout", f.Timeout.String())
	}

	if f.Trace != "" {
		args = append(args, "-trace", f.Trace)
	}

	if f.V {
		args = append(args, "-test.v")
	}

	return args
}

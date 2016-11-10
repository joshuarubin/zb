package lintflags

import (
	"fmt"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/urfave/cli"
)

var (
	// defaults as gometalinter expects them
	defaultConcurrency    = 16
	defaultCycloOver      = 10
	defaultLineLength     = 80
	defaultMinConfidence  = 0.80
	defaultMinOccurrences = 3
	defaultMinConstLength = 3
	defaultDuplThreshold  = 50
	defaultDeadline       = 5 * time.Second
	defaultSort           = SortNone
	// defaultFormat         = "{{.Path}}:{{.Line}}:{{if .Col}}{{.Col}}{{end}}:{{.Severity}}: {{.Message}} ({{.Linter}})"
)

type Data struct {
	NoVendoredLinters bool
	Fast              bool
	Install           bool
	Update            bool
	Force             bool
	Debug             bool
	Concurrency       int
	Exclude           cli.StringSlice
	Include           cli.StringSlice
	Skip              cli.StringSlice
	// Vendor            bool
	CycloOver        int
	LineLength       int
	MinConfidence    float64
	MinOccurrences   int
	MinConstLength   int
	DuplThreshold    int
	Sort             SortKey
	NoTests          bool
	Deadline         time.Duration
	Errors           bool
	JSON             bool
	Checkstyle       bool
	NoEnableGC       bool
	Aggregate        bool
	Disable          cli.StringSlice
	Enable           cli.StringSlice
	Linter           cli.StringSlice
	MessageOverrides cli.StringSlice
	Severity         cli.StringSlice
	DisableAll       bool
	EnableAll        bool
	// Format           string
}

func (f *Data) LintFlags() []cli.Flag {
	return []cli.Flag{
		cli.BoolFlag{
			Name:        "no-vendored-linters",
			Usage:       "Do not use vendored linters (not recommended).",
			Destination: &f.NoVendoredLinters,
		},
		cli.BoolFlag{
			Name:        "fast",
			Usage:       "Only run fast linters.",
			Destination: &f.Fast,
		},
		cli.BoolFlag{
			Name:        "install, i",
			Usage:       "Attempt to install all known linters.",
			Destination: &f.Install,
		},
		cli.BoolFlag{
			Name:        "update, u",
			Usage:       "Pass -u to go tool when installing.",
			Destination: &f.Update,
		},
		cli.BoolFlag{
			Name:        "force, f",
			Usage:       "Pass -f to go tool when installing. When linting treats all lint results as uncached.",
			Destination: &f.Force,
		},
		cli.BoolFlag{
			Name:        "debug, d",
			Usage:       "Display messages for failed linters, etc.",
			Destination: &f.Debug,
		},
		cli.IntFlag{
			Name:        "concurrency, j",
			Usage:       "Number of concurrent linters to run.",
			Value:       getConcurrency(),
			Destination: &f.Concurrency,
		},
		cli.StringSliceFlag{
			Name:  "exclude, e",
			Usage: "Exclude messages matching these regular expressions.",
			Value: &f.Exclude,
		},
		cli.StringSliceFlag{
			Name:  "include, I",
			Usage: "Include messages matching these regular expressions.",
			Value: &f.Include,
		},
		cli.StringSliceFlag{
			Name:  "skip, s",
			Usage: "Skip directories with this name when expanding '...'.",
			Value: &f.Skip,
		},
		// cli.BoolFlag{
		// 	Name:        "vendor",
		// 	Usage:       "Enable vendoring support (skips 'vendor' directories and sets GO15VENDOREXPERIMENT=1).",
		// 	Destination: &f.Vendor,
		// },
		cli.IntFlag{
			Name:        "cyclo-over",
			Usage:       "Report functions with cyclomatic complexity over N (using gocyclo).",
			Value:       defaultCycloOver,
			Destination: &f.CycloOver,
		},
		cli.IntFlag{
			Name:        "line-length",
			Usage:       "Report lines longer than N (using lll).",
			Value:       defaultLineLength,
			Destination: &f.LineLength,
		},
		cli.Float64Flag{
			Name:        "min-confidence",
			Usage:       "Minimum confidence interval to pass to golint.",
			Value:       defaultMinConfidence,
			Destination: &f.MinConfidence,
		},
		cli.IntFlag{
			Name:        "min-occurrences",
			Usage:       "Minimum occurrences to pass to goconst.",
			Value:       defaultMinOccurrences,
			Destination: &f.MinOccurrences,
		},
		cli.IntFlag{
			Name:        "min-const-length",
			Usage:       "Minimumum constant length.",
			Value:       defaultMinConstLength,
			Destination: &f.MinConstLength,
		},
		cli.IntFlag{
			Name:        "dupl-threshold",
			Usage:       "Minimum token sequence as a clone for dupl.",
			Value:       defaultDuplThreshold,
			Destination: &f.DuplThreshold,
		},
		cli.GenericFlag{
			Name:  "sort",
			Usage: fmt.Sprintf("Sort output by any of %s.", strings.Join(sortKeysStr, ", ")),
			Value: &f.Sort,
		},
		cli.BoolFlag{
			Name:        "no-tests",
			Usage:       "Do not include test files for linters that support this option",
			Destination: &f.NoTests,
		},
		cli.DurationFlag{
			Name:        "deadline",
			Usage:       "Cancel linters if they have not completed within this duration.",
			Value:       30 * time.Second,
			Destination: &f.Deadline,
		},
		cli.BoolFlag{
			Name:        "errors",
			Usage:       "Only show errors.",
			Destination: &f.Errors,
		},
		cli.BoolFlag{
			Name:        "json",
			Usage:       "Generate structured JSON rather than standard line-based output.",
			Destination: &f.JSON,
		},
		cli.BoolFlag{
			Name:        "checkstyle",
			Usage:       "Generate checkstyle XML rather than standard line-based output.",
			Destination: &f.Checkstyle,
		},
		cli.BoolFlag{
			Name:        "no-enable-gc",
			Usage:       "Do not enable GC for linters (useful on large repositories).",
			Destination: &f.NoEnableGC,
		},
		cli.BoolFlag{
			Name:        "aggregate",
			Usage:       "Aggregate issues reported by several linters.",
			Destination: &f.Aggregate,
		},
		cli.StringSliceFlag{
			Name:  "disable, D",
			Usage: fmt.Sprintf("List of linters to disable (%s).", strings.Join(disabledLinters, ",")),
			Value: &f.Disable,
		},
		cli.StringSliceFlag{
			Name:  "enable, E",
			Usage: fmt.Sprintf("Enable previously disabled linters (%s).", strings.Join(enabledLinters, ",")),
			Value: &f.Enable,
		},
		cli.StringSliceFlag{
			Name:  "linter",
			Usage: "Specify a linter.",
			Value: &f.Linter,
		},
		cli.StringSliceFlag{
			Name:  "message-overrides",
			Usage: "Override message from linter. {message} will be expanded to the original message.",
			Value: &f.MessageOverrides,
		},
		cli.StringSliceFlag{
			Name:  "severity",
			Usage: "Map of linter severities.",
			Value: &f.Severity,
		},
		cli.BoolFlag{
			Name:        "disable-all",
			Usage:       "Disable all linters.",
			Destination: &f.DisableAll,
		},
		cli.BoolFlag{
			Name:        "enable-all",
			Usage:       "Enable all linters.",
			Destination: &f.EnableAll,
		},
		// cli.StringFlag{
		// 	Name:        "format",
		// 	Usage:       "Output format.",
		// 	Value:       defaultFormat,
		// 	Destination: &f.Format,
		// },
	}
}

func getConcurrency() int {
	// return at least 1
	// the lesser of:
	// * if even number of cpus, 1 less than half that number
	// * if odd number of cpus, half that number rounded down (truncated)
	// * GOMAXPROCS
	n := runtime.NumCPU()
	var cpu int
	if n%2 == 0 { // even
		cpu = n/2 - 1
	} else { // odd
		cpu = n / 2
	}

	m := runtime.GOMAXPROCS(-1)

	r := m
	if cpu < m {
		r = cpu
	}

	if r <= 0 {
		r = 1
	}

	return r
}

// these are disabled in zb by default
var disabledLinters = []string{"aligncheck", "dupl", "gocyclo", "lll", "structcheck", "test", "testify"}

// these are enabled in zb by default
var enabledLinters = []string{"gofmt", "goimports", "unused"}

// the following are disabled in gometalinter by default:
// testify
// test
// gofmt
// goimports
// lll
// misspell
// unused

// the following are considered slow:
// structcheck
// varcheck
// errcheck
// aligncheck
// testify
// test
// interfacer
// unconvert
// deadcode

// by default enable all linters except:
// gocyclo
// dupl
// aligncheck (slow)
// test (already disabled)
// testify (already disabled)
// lll (already disabled)
// structcheck (slow)
//
// if --fast is given, use default and additionally disable:
// varcheck (slow)
// errcheck (slow)
// interfacer (slow)
// unconvert (slow)
// deadcode (slow)
//
// enable the following explicitly by default:
// gofmt
// goimports
// misspell
// unused

const (
	disable = "-D"
	enable  = "-E"
)

func (f *Data) linters() []string {
	// relevant flags:
	// --enable
	// --disable
	// --enable-all
	// --disable-all
	// --fast

	lm := map[string]string{
		// DISABLED
		"aligncheck":  disable,
		"dupl":        disable,
		"gocyclo":     disable,
		"structcheck": disable,
		// lll     disabled by default in gometalinter
		// test    disabled by default in gometalinter
		// testify disabled by default in gometalinter

		// ENABLED
		"gofmt":     enable,
		"goimports": enable,
		"unused":    enable,
	}

	for _, v := range f.Disable {
		lm[v] = disable
	}

	for _, v := range f.Enable {
		lm[v] = enable
	}

	var disabled, enabled []string

	for l, f := range lm {
		if f == disable {
			disabled = append(disabled, l)
		} else {
			enabled = append(enabled, l)
		}
	}

	sort.Strings(disabled)
	sort.Strings(enabled)

	var args []string
	for _, v := range disabled {
		args = append(args, disable, v)
	}
	for _, v := range enabled {
		args = append(args, enable, v)
	}

	if f.Fast {
		args = append(args, "--fast")
	}

	if f.DisableAll {
		args = append(args, "--disable-all")
	}

	if f.EnableAll {
		args = append(args, "--enable-all")
	}

	return args
}

func (f *Data) LintArgs() []string {
	var args []string

	if f.NoVendoredLinters {
		args = append(args, "--no-vendored-linters")
	}

	if f.Install {
		args = append(args, "--install")

		if f.Update {
			args = append(args, "--update")
		}

		if f.Force {
			args = append(args, "--force")
		}
	}

	if f.Debug {
		args = append(args, "--debug")
	}

	if f.Concurrency != 0 && f.Concurrency != defaultConcurrency {
		args = append(args, "-j", fmt.Sprintf("%d", f.Concurrency))
	}

	for _, v := range f.Exclude {
		args = append(args, "--exclude", v)
	}

	for _, v := range f.Include {
		args = append(args, "--include", v)
	}

	for _, v := range f.Skip {
		args = append(args, "--skip", v)
	}

	// if f.Vendor {
	// 	args = append(args, "--vendor")
	// }

	if f.CycloOver != 0 && f.CycloOver != defaultCycloOver {
		args = append(args, "--cyclo-over", fmt.Sprintf("%d", f.CycloOver))
	}

	if f.LineLength != 0 && f.LineLength != defaultLineLength {
		args = append(args, "--line-length", fmt.Sprintf("%d", f.LineLength))
	}

	if f.MinConfidence != 0 && f.MinConfidence != defaultMinConfidence {
		args = append(args, "--min-confidence", fmt.Sprintf("%f", f.MinConfidence))
	}

	if f.MinOccurrences != 0 && f.MinOccurrences != defaultMinOccurrences {
		args = append(args, "--min-occurrences", fmt.Sprintf("%d", f.MinOccurrences))
	}

	if f.MinConstLength != 0 && f.MinConstLength != defaultMinConstLength {
		args = append(args, "--min-const-length", fmt.Sprintf("%d", f.MinConstLength))
	}

	if f.DuplThreshold != 0 && f.DuplThreshold != defaultDuplThreshold {
		args = append(args, "--dupl-threshold", fmt.Sprintf("%d", f.DuplThreshold))
	}

	if f.Sort != "" && f.Sort != defaultSort {
		args = append(args, "--sort", string(f.Sort))
	}

	if !f.NoTests {
		args = append(args, "--tests")
	}

	if f.Deadline != 0 && f.Deadline != defaultDeadline {
		args = append(args, "--deadline", f.Deadline.String())
	}

	if f.Errors {
		args = append(args, "--errors")
	}

	if f.JSON {
		args = append(args, "--json")
	}

	if f.Checkstyle {
		args = append(args, "--checkstyle")
	}

	if !f.NoEnableGC {
		args = append(args, "--enable-gc")
	}

	if f.Aggregate {
		args = append(args, "--aggregate")
	}

	for _, v := range f.Linter {
		args = append(args, "--linter", v)
	}

	for _, v := range f.MessageOverrides {
		args = append(args, "--message-overrides", v)
	}

	for _, v := range f.Severity {
		args = append(args, "--severity", v)
	}

	// if f.Format != "" && f.Format != defaultFormat {
	// 	args = append(args, "--format", f.Format)
	// }

	return append(args, f.linters()...)
}

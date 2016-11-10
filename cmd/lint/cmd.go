package lint

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strings"

	"golang.org/x/sync/errgroup"

	"github.com/urfave/cli"
	"jrubin.io/slog"
	"jrubin.io/zb/cmd"
	"jrubin.io/zb/lib/project"
	"jrubin.io/zb/lib/zbcontext"
)

// Cmd is the lint command
var Cmd cmd.Constructor = &cc{}

type cc struct {
	zbcontext.Context
	NoMissingComment bool
	IgnoreSuffixes   cli.StringSlice

	ignoreSuffixMap map[string]struct{}
}

var defaultIgnoreSuffixes = []string{".pb.go", "_string.go"}

func (cmd *cc) New(_ *cli.App, config *cmd.Config) cli.Command {
	cmd.Logger = config.Logger
	cmd.SrcDir = config.Cwd
	cmd.ExcludeVendor = true

	return cli.Command{
		Name:      "lint",
		Usage:     "gometalinter with cache and better defaults",
		ArgsUsage: "[packages]",
		Before: func(c *cli.Context) error {
			cmd.setup()
			return nil
		},
		Action: func(c *cli.Context) error {
			return cmd.run(c.App.Writer, c.Args()...)
		},
		Flags: append(cmd.LintFlags(),
			cli.BoolFlag{
				Name:        "n",
				Usage:       "Hide golint missing comment warnings",
				Destination: &cmd.NoMissingComment,
			},
			cli.StringSliceFlag{
				Name:  "ignore-suffix",
				Usage: fmt.Sprintf("Filter out lint lines from files that have these suffixes (default: %s)", strings.Join(defaultIgnoreSuffixes, ",")),
				Value: &cmd.IgnoreSuffixes,
			},
		),
	}
}

func (cmd *cc) setup() {
	if len(cmd.IgnoreSuffixes) == 0 {
		cmd.IgnoreSuffixes = defaultIgnoreSuffixes
	}

	cmd.ignoreSuffixMap = map[string]struct{}{}

	for _, is := range cmd.IgnoreSuffixes {
		if is == "" {
			continue
		}
		cmd.ignoreSuffixMap[is] = struct{}{}
	}
}

func (cmd *cc) run(w io.Writer, args ...string) error {
	projects, err := project.Projects(&cmd.Context, args...)
	if err != nil {
		return err
	}

	// run go generate as necessary
	if _, err = projects.Build(project.TargetGenerate); err != nil {
		return err
	}

	// TODO(jrubin) make sure gometalinter is installed (and up to date?)
	// TODO(jrubin) cache results like test

	code := zbcontext.ExitOK

	for _, p := range projects {
		for _, pkg := range p.Packages {
			ecode, err := cmd.exec(w, pkg.Package.Dir)
			if err != nil {
				return err
			}

			if code == zbcontext.ExitOK {
				code = ecode
			}
		}
	}

	if code != zbcontext.ExitOK {
		return cli.NewExitError("", int(code))
	}

	return nil
}

func (cmd *cc) exec(w io.Writer, path string) (int, error) {
	args := cmd.LintArgs()
	args = append(args, path)

	cmd.Logger.Debug(zbcontext.QuoteCommand("→ gometalinter", args))

	pr, pw := io.Pipe()

	ecmd := exec.Command("gometalinter", args...) // nosec
	ecmd.Stdout = pw
	ecmd.Stderr = pw

	code := zbcontext.ExitOK

	if err := ecmd.Start(); err != nil {
		return code, err
	}

	var group errgroup.Group
	group.Go(func() error {
		defer pw.Close()

		if ecmd == nil {
			return nil
		}

		ecode, err := zbcontext.ExitCode(ecmd.Wait())
		if err != nil {
			return err
		}

		if code == zbcontext.ExitOK {
			code = ecode
		}

		return nil
	})

	if err := cmd.readLintOutput(pr); err != nil {
		return code, err
	}

	if err := group.Wait(); err != nil {
		return code, err
	}

	return code, nil
}

var (
	levelRE   = regexp.MustCompile(`\A([^:]*):(\d*):(\d*):(\w+): (.*) \((\w+)\)\n\z`)
	commentRE = regexp.MustCompile(` should have comment.* or be unexported`)
)

// Part enum representing each field in a gometalinter line
type Part int

// The different fields of the gometalinter line
const (
	LintFile Part = 1 + iota
	LintLine
	LintColumn
	LintLevel
	LintMessage
	LintLinter
)

func (cmd *cc) readLintOutput(pr io.Reader) error {
	ew := cmd.Logger.Writer(slog.ErrorLevel).Prefix("← ")
	defer ew.Close()

	ww := cmd.Logger.Writer(slog.WarnLevel).Prefix("← ")
	defer ww.Close()

	iw := cmd.Logger.Writer(slog.InfoLevel).Prefix("← ")
	defer iw.Close()

	r := bufio.NewReader(pr)

LOOP:
	for eof := false; !eof; {
		line, err := r.ReadString('\n')
		if err == io.EOF {
			eof = true
		} else if err != nil {
			return err
		}

		m := levelRE.FindStringSubmatch(line)
		if m == nil {
			iw.Write([]byte(line))
			continue
		}

		if cmd.NoMissingComment &&
			m[LintLinter] == "golint" &&
			commentRE.MatchString(m[LintMessage]) {
			continue
		}

		for is := range cmd.ignoreSuffixMap {
			if strings.HasSuffix(m[LintFile], is) {
				continue LOOP
			}
		}

		switch m[LintLevel] {
		case "warning":
			ww.Write([]byte(line))
		case "error":
			ew.Write([]byte(line))
		default:
			iw.Write([]byte(line))
		}
	}

	return nil
}

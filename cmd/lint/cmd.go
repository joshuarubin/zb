package lint

import (
	"bufio"
	"io"
	"os/exec"
	"regexp"

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
	NC bool
}

func (cmd *cc) New(_ *cli.App, config *cmd.Config) cli.Command {
	cmd.Logger = config.Logger
	cmd.SrcDir = config.Cwd
	cmd.ExcludeVendor = true

	return cli.Command{
		Name:      "lint",
		Usage:     "TODO(jrubin)",
		ArgsUsage: "[packages]",
		Action: func(c *cli.Context) error {
			return cmd.run(c.App.Writer, c.Args()...)
		},
		Flags: append(cmd.LintFlags(),
			cli.BoolFlag{
				Name:        "nc",
				Usage:       "Hide golint missing comment warnings",
				Destination: &cmd.NC,
			},
		),
	}
}

func (cmd *cc) run(w io.Writer, args ...string) error {
	projects, err := project.Projects(&cmd.Context, args...)
	if err != nil {
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
	levelRE   = regexp.MustCompile(`\A([^:]*):(\d*):(\d*):(\w+): (.*)\n\z`)
	commentRE = regexp.MustCompile(` should have comment.* or be unexported.*\(golint\)`)
)

func (cmd *cc) readLintOutput(pr io.Reader) error {
	ew := cmd.Logger.Writer(slog.ErrorLevel).Prefix("← ")
	defer ew.Close()

	ww := cmd.Logger.Writer(slog.WarnLevel).Prefix("← ")
	defer ww.Close()

	iw := cmd.Logger.Writer(slog.InfoLevel).Prefix("← ")
	defer iw.Close()

	r := bufio.NewReader(pr)

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

		if cmd.NC && commentRE.MatchString(line) {
			continue
		}

		switch m[4] {
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

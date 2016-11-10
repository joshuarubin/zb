package test

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"

	"golang.org/x/sync/errgroup"

	"github.com/urfave/cli"
	"jrubin.io/slog"
	"jrubin.io/zb/cmd"
	"jrubin.io/zb/lib/project"
	"jrubin.io/zb/lib/zbcontext"
	"jrubin.io/zb/lib/zbtest"
)

// Cmd is the test command
var Cmd cmd.Constructor = &cc{}

// NOTE: inspired by, and much of the code from https://github.com/rsc/gt

type cc struct {
	zbtest.ZBTest
}

func (cmd *cc) New(_ *cli.App, config *cmd.Config) cli.Command {
	cmd.Logger = config.Logger
	cmd.SrcDir = config.Cwd

	return cli.Command{
		Name:      "test",
		Usage:     "test all of the packages in each of the projects and cache the results",
		ArgsUsage: "[build/test flags] [packages]",
		Before: func(c *cli.Context) error {
			return cmd.setup()
		},
		Action: func(c *cli.Context) error {
			return cmd.run(c.App.Writer, c.Args()...)
		},
		Flags: append(cmd.TestFlags(), []cli.Flag{
			cli.BoolFlag{
				Name:        "f",
				Destination: &cmd.Force,
				Usage: `

				treat all test results as uncached, as does the use of any 'go
				test' flag other than -short and -v`,
			},
			cli.BoolFlag{
				Name:        "l",
				Destination: &cmd.List,
				Usage:       "list the uncached tests it would run",
			},
			cli.StringFlag{
				Name:        "cache",
				Destination: &cmd.CacheDir,
				EnvVar:      "CACHE",
				Value:       zbtest.DefaultCacheDir(),
				Usage: `

				test results are saved in the 'go-test-cache' directory under
				this base directory`,
			},
		}...),
	}
}

func (cmd *cc) setup() error {
	if filepath.Base(cmd.CacheDir) != "go-test-cache" {
		cmd.CacheDir = filepath.Join(cmd.CacheDir, "go-test-cache")
	}
	return nil
}

func (cmd *cc) run(w io.Writer, args ...string) error {
	projects, err := project.Projects(cmd.Context, args...)
	if err != nil {
		return err
	}

	// run go generate as necessary
	if _, err = projects.Build(project.TargetGenerate); err != nil {
		return err
	}

	pkgs, toRun, err := cmd.buildLists(projects)
	if err != nil {
		return err
	}

	if cmd.List {
		for _, pkg := range toRun {
			fmt.Fprintf(w, "%s\n", pkg.ImportPath)
		}
		return nil
	}

	return cmd.runOneTest(w, pkgs, toRun)
}

func (cmd *cc) buildLists(projects project.ProjectList) (pkgs, toRun project.Packages, err error) {
	for _, proj := range projects {
		for _, pkg := range proj.Packages {
			if pkg.IsVendored {
				continue
			}

			pkgs = append(pkgs, pkg)

			var foundResult bool
			if foundResult, err = cmd.HaveResult(pkg); err != nil {
				return
			}

			if !foundResult {
				toRun = append(toRun, pkg)
			}
		}
	}

	sort.Sort(&pkgs)
	sort.Sort(&toRun)

	return
}

func (cmd *cc) runOneTest(w io.Writer, pkgs, toRun project.Packages) error {
	var ecmd *exec.Cmd
	pr, pw := io.Pipe()
	r := bufio.NewReader(pr)
	if len(toRun) > 0 {
		if err := os.MkdirAll(cmd.CacheDir, 0700); err != nil {
			return err
		}

		args := []string{"test"}
		args = append(args, cmd.TestArgs(nil, nil)...)
		for _, pkg := range toRun {
			args = append(args, pkg.ImportPath)
		}

		cmd.Logger.Debug(zbcontext.QuoteCommand("→ go", args))

		ecmd = exec.Command("go", args...) // nosec
		ecmd.Stdout = pw
		ecmd.Stderr = pw
		if err := ecmd.Start(); err != nil {
			return err
		}
	}

	code := zbcontext.ExitOK
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

	ew := cmd.Logger.Writer(slog.ErrorLevel)
	defer ew.Close()

	ow := cmd.Logger.Writer(slog.InfoLevel)
	defer ow.Close()

	for _, pkg := range pkgs {
		if len(toRun) > 0 && toRun[0] == pkg {
			if err := cmd.ReadResult(ow, ew, r, pkg); err != nil {
				return err
			}
			toRun = toRun[1:]
		} else {
			passed, err := cmd.ShowResult(ow, ew, pkg)
			if err != nil {
				return err
			}
			if code == zbcontext.ExitOK && !passed {
				code = zbcontext.ExitFailed
			}
		}
	}

	io.Copy(w, r)

	if err := group.Wait(); err != nil {
		return err
	}

	if code != zbcontext.ExitOK {
		return cli.NewExitError("", int(code))
	}

	return nil
}

package test

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"golang.org/x/sync/errgroup"

	"github.com/urfave/cli"
	"jrubin.io/slog"
	"jrubin.io/zb/cmd"
	"jrubin.io/zb/lib/dependency"
	"jrubin.io/zb/lib/project"
	"jrubin.io/zb/lib/zbcontext"
	"jrubin.io/zb/lib/zbtest"
)

// Cmd is the test command
var Cmd cmd.Constructor = &cc{}

// NOTE: inspired by, and much of the code from https://github.com/rsc/gt

type cc struct {
	zbtest.ZBTest
	Generate bool
}

func (cmd *cc) New(_ *cli.App, config *cmd.Config) cli.Command {
	cmd.Config = config

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
			cli.BoolFlag{
				Name:        "generate, g",
				Usage:       "run go generate as necessary before execution",
				Destination: &cmd.Generate,
			},
		}...),
	}
}

func (cmd *cc) setup() error {
	if filepath.Base(cmd.CacheDir) != "test" {
		cmd.CacheDir = filepath.Join(cmd.CacheDir, "test")
	}
	return nil
}

func (cmd *cc) run(w io.Writer, args ...string) error {
	var pkgs, toRun project.Packages
	var err error

	if cmd.Package {
		pkgs, toRun, err = cmd.runPackages(w, args...)
	} else {
		pkgs, toRun, err = cmd.runProjects(w, args...)
	}

	if err != nil {
		return err
	}

	if cmd.List {
		for _, pkg := range toRun {
			fmt.Fprintf(w, "%s\n", pkg.ImportPath)
		}
		return nil
	}

	return cmd.runTest(w, pkgs, toRun)
}

func (cmd *cc) runPackages(w io.Writer, args ...string) (pkgs, toRun project.Packages, err error) {
	pkgs, err = project.ListPackages(&cmd.Context, args...)
	if err != nil {
		return
	}

	return cmd.buildPackagesLists(pkgs)
}

func (cmd *cc) runProjects(w io.Writer, args ...string) (pkgs, toRun project.Packages, err error) {
	var projects project.List
	projects, err = project.Projects(&cmd.Context, args...)
	if err != nil {
		return
	}

	// run go generate as necessary
	if cmd.Generate {
		if _, err = projects.Build(dependency.TargetGenerate); err != nil {
			return
		}
	}

	return cmd.buildProjectsLists(projects)
}

func (cmd *cc) buildPackagesLists(in project.Packages) (pkgs, toRun project.Packages, err error) {
	for _, pkg := range in {
		if pkg.IsVendored {
			continue
		}

		pkgs.Insert(pkg)

		var foundResult bool
		if foundResult, err = cmd.HaveResult(pkg); err != nil {
			return
		}

		if !foundResult {
			toRun.Insert(pkg)
		}
	}

	return
}

func (cmd *cc) buildProjectsLists(projects project.List) (pkgs, toRun project.Packages, err error) {
	for _, proj := range projects {
		var p, r project.Packages
		p, r, err = cmd.buildPackagesLists(proj.Packages)
		if err != nil {
			return
		}

		pkgs = pkgs.Append(p)
		toRun = toRun.Append(r)
	}

	return
}

func (cmd *cc) runTest(w io.Writer, pkgs, toRun project.Packages) error {
	var ecmd *exec.Cmd
	pr, pw := io.Pipe()
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
		defer func() { _ = pw.Close() }() // nosec

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
	defer func() { _ = ew.Close() }() // nosec

	ow := cmd.Logger.Writer(slog.InfoLevel)
	defer func() { _ = ow.Close() }() // nosec

	r := bufio.NewReader(pr)

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

	if _, err := io.Copy(w, r); err != nil {
		return err
	}

	if err := group.Wait(); err != nil {
		return err
	}

	if code != zbcontext.ExitOK {
		return cli.NewExitError("", code)
	}

	return nil
}

package test

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"

	"golang.org/x/sync/errgroup"

	"github.com/urfave/cli"
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
	List bool
}

func (co *cc) New(*cli.App) cli.Command {
	return cli.Command{
		Name:      "test",
		Usage:     "test all of the packages in each of the projects and cache the results",
		ArgsUsage: "[build/test flags] [packages]",
		Action: func(c *cli.Context) error {
			ctx := cmd.Context(c)
			return co.run(ctx, c.App.Writer, c.Args()...)
		},
		Flags: append(co.TestFlags(), []cli.Flag{
			cli.BoolFlag{
				Name:        "f",
				Destination: &co.Force,
				Usage: `

				treat all test results as uncached, as does the use of any 'go
				test' flag other than -short and -v`,
			},
			cli.BoolFlag{
				Name:        "l",
				Destination: &co.List,
				Usage:       "list the uncached tests it would run",
			},
		}...),
	}
}

func (co *cc) run(ctx zbcontext.Context, w io.Writer, args ...string) error {
	ctx = co.TestSetup(ctx)

	var pkgs, toRun project.Packages
	var err error

	if ctx.Package {
		pkgs, toRun, err = co.runPackages(ctx, w, args...)
	} else {
		pkgs, toRun, err = co.runProjects(ctx, w, args...)
	}

	if err != nil {
		return err
	}

	if co.List {
		for _, pkg := range toRun {
			fmt.Fprintf(w, "%s\n", pkg.ImportPath)
		}
		return nil
	}

	return co.runTest(ctx, w, pkgs, toRun)
}

func (co *cc) runPackages(ctx zbcontext.Context, w io.Writer, args ...string) (pkgs, toRun project.Packages, err error) {
	pkgs, err = project.ListPackages(ctx, args...)
	if err != nil {
		return
	}

	return co.buildPackagesLists(ctx, pkgs)
}

func (co *cc) runProjects(ctx zbcontext.Context, w io.Writer, args ...string) (pkgs, toRun project.Packages, err error) {
	var projects project.List
	projects, err = project.Projects(ctx, args...)
	if err != nil {
		return
	}

	return co.buildProjectsLists(ctx, projects)
}

func (co *cc) buildPackagesLists(ctx zbcontext.Context, in project.Packages) (pkgs, toRun project.Packages, err error) {
	for _, pkg := range in {
		if pkg.IsVendored {
			continue
		}

		pkgs.Insert(pkg)

		var foundResult bool
		if foundResult, err = co.HaveResult(ctx, pkg); err != nil {
			return
		}

		if !foundResult {
			toRun.Insert(pkg)
		}
	}

	return
}

func (co *cc) buildProjectsLists(ctx zbcontext.Context, projects project.List) (pkgs, toRun project.Packages, err error) {
	for _, proj := range projects {
		var p, r project.Packages
		p, r, err = co.buildPackagesLists(ctx, proj.Packages)
		if err != nil {
			return
		}

		pkgs = pkgs.Append(p)
		toRun = toRun.Append(r)
	}

	return
}

func (co *cc) runTest(ctx zbcontext.Context, w io.Writer, pkgs, toRun project.Packages) error {
	var ecmd *exec.Cmd
	pr, pw := io.Pipe()
	if len(toRun) > 0 {
		if err := os.MkdirAll(ctx.CacheDir, 0700); err != nil {
			return err
		}

		args := []string{"test"}
		args = append(args, co.TestArgs(nil, nil)...)
		for _, pkg := range toRun {
			args = append(args, pkg.ImportPath)
		}

		ctx.Logger.Debug(zbcontext.QuoteCommand("→ go", args))

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

	r := bufio.NewReader(io.TeeReader(pr, w))

	for _, pkg := range pkgs {
		if len(toRun) > 0 && toRun[0] == pkg {
			if err := co.ReadResult(ctx, r, pkg); err != nil {
				return err
			}
			toRun = toRun[1:]
		} else {
			passed, err := co.ShowResult(ctx, w, pkg)
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

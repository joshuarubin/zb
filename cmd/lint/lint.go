package lint

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/sync/errgroup"

	"github.com/urfave/cli"
	"jrubin.io/zb/cmd"
	"jrubin.io/zb/lib/project"
	"jrubin.io/zb/lib/zbcontext"
	"jrubin.io/zb/lib/zblint"
)

// Cmd is the lint command
var Cmd cmd.Constructor = &cc{}

type cc struct {
	zblint.ZBLint
}

func (co *cc) New(*cli.App) cli.Command {
	return cli.Command{
		Name:      "lint",
		Usage:     "gometalinter with cache and better defaults",
		ArgsUsage: "[arguments] [packages]",
		Action: func(c *cli.Context) error {
			ctx := cmd.Context(c)
			ctx = co.LintSetup(ctx)
			return co.run(ctx, c.App.Writer, c.Args()...)
		},
		Flags: append(co.LintFlags(),
			cli.BoolFlag{
				Name:        "n",
				Usage:       "Hide golint missing comment warnings",
				Destination: &co.NoMissingComment,
			},
			cli.StringSliceFlag{
				Name:  "ignore-suffix",
				Usage: fmt.Sprintf("Filter out lint lines from files that have these suffixes (default: %s)", strings.Join(zblint.DefaultIgnoreSuffixes, ",")),
				Value: &co.IgnoreSuffixes,
			},
		),
	}
}

func (co *cc) run(ctx zbcontext.Context, w io.Writer, args ...string) error {
	if _, err := exec.LookPath("gometalinter"); err != nil {
		return err
	}

	if ctx.Package {
		return co.runPackage(ctx, w, args...)
	}

	return co.runProject(ctx, w, args...)
}

func (co *cc) runPackage(ctx zbcontext.Context, w io.Writer, args ...string) error {
	pkgs, err := project.ListPackages(ctx, args...)
	if err != nil {
		return err
	}

	pkgs, toRun, err := co.buildListsPackages(ctx, pkgs)
	if err != nil {
		return err
	}

	return co.exec(ctx, w, pkgs, toRun)
}

func (co *cc) runProject(ctx zbcontext.Context, w io.Writer, args ...string) error {
	projects, err := project.Projects(ctx, args...)
	if err != nil {
		return err
	}

	pkgs, toRun, err := co.buildListsProjects(ctx, projects)
	if err != nil {
		return err
	}

	return co.exec(ctx, w, pkgs, toRun)
}

func (co *cc) exec(ctx zbcontext.Context, w io.Writer, pkgs, toRun project.Packages) error {
	code := zbcontext.ExitOK

	for _, pkg := range pkgs {
		file, err := co.CacheFile(ctx, pkg)
		if err != nil {
			return err
		}

		if len(toRun) > 0 && toRun[0] == pkg {
			path := pkg.Package.Dir
			if rel, err := filepath.Rel(zbcontext.CWD, path); err == nil {
				path = rel
			}

			ecode, err := co.runLinter(ctx, w, path, file)
			if err != nil {
				return err
			}
			if code == zbcontext.ExitOK {
				code = ecode
			}

			toRun = toRun[1:]
		} else {
			failed, err := co.ShowResult(w, file)
			if err != nil {
				return err
			}
			if code == zbcontext.ExitOK && failed {
				code = zbcontext.ExitFailed
			}
		}
	}

	if code != zbcontext.ExitOK {
		return cli.NewExitError("", code)
	}

	return nil
}

func (co *cc) buildListsPackages(ctx zbcontext.Context, in project.Packages) (pkgs, toRun project.Packages, err error) {
	for _, pkg := range in {
		if pkg.IsVendored {
			continue
		}

		pkgs = append(pkgs, pkg)

		var foundResult bool
		if foundResult, err = co.HaveResult(ctx, pkg); err != nil {
			return
		}

		if !foundResult {
			toRun = append(toRun, pkg)
		}
	}

	sort.Sort(&pkgs)
	sort.Sort(&toRun)

	return
}

func (co *cc) buildListsProjects(ctx zbcontext.Context, projects project.List) (pkgs, toRun project.Packages, err error) {
	for _, proj := range projects {
		var p, r project.Packages
		p, r, err = co.buildListsPackages(ctx, proj.Packages)
		if err != nil {
			return
		}
		pkgs = pkgs.Append(p)
		toRun = toRun.Append(r)
	}

	sort.Sort(&pkgs)
	sort.Sort(&toRun)

	return
}

func (co *cc) runLinter(ctx zbcontext.Context, w io.Writer, path, cacheFile string) (int, error) {
	code := zbcontext.ExitOK

	if err := os.MkdirAll(ctx.CacheDir, 0700); err != nil {
		return code, err
	}

	args := co.LintArgs()
	args = append(args, path)

	ctx.Logger.Debug(zbcontext.QuoteCommand("→ gometalinter", args))

	pr, pw := io.Pipe()

	ecmd := exec.Command("gometalinter", args...) // nosec
	ecmd.Stdout = pw
	ecmd.Stderr = pw

	if err := ecmd.Start(); err != nil {
		return code, err
	}

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

	if err := co.ReadResult(w, pr, cacheFile); err != nil {
		return code, err
	}

	if err := group.Wait(); err != nil {
		return code, err
	}

	return code, nil
}

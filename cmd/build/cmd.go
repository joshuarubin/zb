package build

import (
	"sync/atomic"

	"golang.org/x/sync/errgroup"

	"github.com/urfave/cli"
	"jrubin.io/zb/cmd"
	"jrubin.io/zb/lib/dependency"
	"jrubin.io/zb/lib/project"
	"jrubin.io/zb/lib/zbcontext"
)

// TODO(jrubin) automatically add missing imports to vendor/

// Cmd is the build command
var Cmd cmd.Constructor = &cc{}

type cc struct {
	zbcontext.Context
}

func (cmd *cc) New(_ *cli.App, config *cmd.Config) cli.Command {
	cmd.Logger = config.Logger
	cmd.SrcDir = config.Cwd

	return cli.Command{
		Name:      "build",
		Usage:     "build all of the executables in each of the projects",
		ArgsUsage: "[build flags] [packages]",
		Action: func(c *cli.Context) error {
			return cmd.run(c.Args()...)
		},
		Flags: append(cmd.Flags(),
			cli.StringFlag{
				Name: "run",
				Usage: `

				passed to "go generate" if non-empty, specifies a regular
				expression to select directives whose full original source text
				(excluding any trailing spaces and final newline) matches the
				expression.`,
				Destination: &cmd.GenerateRun,
			},
		),
	}
}

func (cmd *cc) run(args ...string) error {
	projects, err := project.Projects(cmd.Context, args...)
	if err != nil {
		return err
	}

	targets, err := projects.Targets(project.TargetBuild)
	if err != nil {
		return err
	}

	// set up the waitgroup dependencies
	for _, t := range targets {
		target := t

		target.RequiredBy.Range(func(r *dependency.Target) {
			r.Add(1)
			target.OnDone(r.WaitGroup.Done)
		})
	}

	var built uint32
	var group errgroup.Group

	for _, t := range targets {
		target := t

		deps, err := target.Dependencies()
		if err != nil {
			return err
		}

		group.Go(func() error {
			defer target.Done()
			target.Wait()

			if !target.Buildable() {
				return nil
			}

			// build target if any of its dependencies are newer than itself
			for _, dep := range deps {
				// don't use .Before since filesystem time resolution might
				// cause files times to be within the same second
				if !dep.ModTime().After(target.ModTime()) {
					continue
				}

				if err := target.Build(); err != nil {
					return err
				}

				atomic.AddUint32(&built, 1)
				return nil
			}

			return nil
		})
	}

	if err := group.Wait(); err != nil {
		return err
	}

	if built == 0 {
		cmd.Logger.Info("nothing to build")
	}

	return nil
}

package test

import (
	"fmt"

	"github.com/urfave/cli"
	"jrubin.io/zb/cmd"
	"jrubin.io/zb/lib/project"
	"jrubin.io/zb/lib/zbcontext"
)

// Cmd is the test command
var Cmd cmd.Constructor = &cc{}

type cc struct {
	zbcontext.Context
}

func (cmd *cc) New(_ *cli.App, config *cmd.Config) cli.Command {
	cmd.Logger = config.Logger
	cmd.SrcDir = config.Cwd
	cmd.ExcludeVendor = true

	ret := cli.Command{
		Name:      "test",
		Usage:     "test all of the packages in each of the projects and cache the results",
		ArgsUsage: "[build/test flags] [packages] [build/test flags & test binary flags]",
		Action: func(c *cli.Context) error {
			return cmd.run(c.Args()...)
		},
	}

	ret.Flags = append(ret.Flags, cmd.BuildFlags.Flags()...)
	ret.Flags = append(ret.Flags, cmd.TestFlags.Flags()...)
	ret.Flags = append(ret.Flags, []cli.Flag{
		cli.BoolFlag{
			Name:        "gt.timing",
			Destination: &cmd.Timing,
			// TODO(jrubin)
		},
		cli.BoolFlag{
			Name:        "f",
			Destination: &cmd.Force,
			Usage: `

			treat all test results as uncached, as does the use of any 'go test'
			flag other than -short and -v`,
		},
		cli.BoolFlag{
			Name:        "l",
			Destination: &cmd.List,
			Usage:       "list the uncached tests it would run",
		},
	}...)

	return ret
}

// TODO(jrubin) test (with cache like gt)
func (cmd *cc) run(args ...string) error {
	projects, err := project.Projects(cmd.Context, args...)
	if err != nil {
		return err
	}

	for _, p := range projects {
		for _, pkg := range p.Packages {
			fmt.Println(pkg.Package.Dir, pkg.ImportPath)
		}
	}

	return nil
}

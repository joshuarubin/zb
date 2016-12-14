package version

import (
	"fmt"
	"runtime"

	"github.com/urfave/cli"
	"jrubin.io/zb/cmd"
)

// Cmd is the version command
var Cmd cmd.Constructor = &cc{}

type cc struct {
	Short bool
}

func (co *cc) New(app *cli.App) cli.Command {
	return cli.Command{
		Name:   "version",
		Usage:  fmt.Sprintf("prints the version of %s", app.Name),
		Action: co.run,
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:        "short, s",
				Destination: &co.Short,
			},
		},
	}
}

func (co *cc) run(c *cli.Context) error {
	if co.Short {
		fmt.Fprintf(c.App.Writer, "%s\n", c.App.Version)
		return nil
	}

	ctx := cmd.Context(c)

	var commit string
	if ctx.GitCommit != nil {
		commit = *ctx.GitCommit
		if len(commit) >= 7 {
			commit = commit[:7]
		}
	}

	var buildDate string
	if ctx.BuildDate != nil {
		buildDate = *ctx.BuildDate
	}

	fmt.Fprintf(c.App.Writer, "%s version %s (git: %s, date: %s, %s)\n",
		c.App.Name,
		c.App.Version,
		commit,
		buildDate,
		runtime.Version(),
	)
	return nil
}

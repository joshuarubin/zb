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
	*cmd.Config
	Short bool
}

func (cmd *cc) New(app *cli.App, config *cmd.Config) cli.Command {
	cmd.Config = config

	return cli.Command{
		Name:   "version",
		Usage:  fmt.Sprintf("prints the version of %s", app.Name),
		Action: cmd.run,
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:        "short, s",
				Destination: &cmd.Short,
			},
		},
	}
}

func (cmd *cc) run(c *cli.Context) error {
	if cmd.Short {
		fmt.Fprintf(c.App.Writer, "%s\n", c.App.Version)
		return nil
	}

	var commit string
	if cmd.GitCommit != nil {
		commit = *cmd.GitCommit
		if len(commit) >= 7 {
			commit = commit[:7]
		}
	}

	var buildDate string
	if cmd.BuildDate != nil {
		buildDate = *cmd.BuildDate
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

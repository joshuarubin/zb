package version

import (
	"fmt"
	"runtime"

	"github.com/urfave/cli"
	"jrubin.io/zb/cmd"
)

var _ cmd.Constructor = (*Cmd)(nil)

type Cmd struct {
	*cmd.Config
	Short bool
}

func (cmd *Cmd) New(app *cli.App, config *cmd.Config) cli.Command {
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

func (cmd *Cmd) run(c *cli.Context) error {
	if cmd.Short {
		fmt.Fprintf(c.App.Writer, "%s\n", c.App.Version)
		return nil
	}

	fmt.Fprintf(c.App.Writer, "%s version %s (git: %s, date: %s, %s)\n",
		c.App.Name,
		c.App.Version,
		(*cmd.GitCommit)[:7],
		*cmd.BuildDate,
		runtime.Version(),
	)
	return nil
}

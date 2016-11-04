package version

import (
	"fmt"

	"github.com/urfave/cli"
	"jrubin.io/zb/cmd"
)

var _ cmd.Constructor = (*Cmd)(nil)

type Cmd struct {
	Short bool
}

func (v *Cmd) New(app *cli.App, _ *cmd.Config) cli.Command {
	return cli.Command{
		Name:   "version",
		Usage:  fmt.Sprintf("prints the version of %s", app.Name),
		Action: v.run,
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:        "short, s",
				Destination: &v.Short,
			},
		},
	}
}

func (v *Cmd) run(c *cli.Context) error {
	if v.Short {
		fmt.Fprintf(c.App.Writer, "%s\n", c.App.Version)
		return nil
	}

	// TODO(jrubin) add go version, git commit (-dirty), build datestamp
	fmt.Fprintf(c.App.Writer, "%s version %s\n", c.App.Name, c.App.Version)
	return nil
}

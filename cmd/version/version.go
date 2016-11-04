package version

import (
	"fmt"

	"github.com/urfave/cli"
	"jrubin.io/zb/cmd"
)

var _ cmd.Constructor = (*Version)(nil)

type Version struct {
	Short bool
}

func (v *Version) New(app *cli.App, _ *cmd.Config) cli.Command {
	return cli.Command{
		Name:   "version",
		Usage:  fmt.Sprintf("prints the version of %s", app.Name),
		Action: v.version,
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:        "short, s",
				Destination: &v.Short,
			},
		},
	}
}

func (v *Version) version(c *cli.Context) error {
	if v.Short {
		fmt.Fprintf(c.App.Writer, "%s\n", c.App.Version)
		return nil
	}

	fmt.Fprintf(c.App.Writer, "%s version %s\n", c.App.Name, c.App.Version)
	return nil
}

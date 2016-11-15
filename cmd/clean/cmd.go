package clean

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli"
	"jrubin.io/zb/cmd"
	"jrubin.io/zb/lib/project"
	"jrubin.io/zb/lib/zbcontext"
)

// Cmd is the clean command
var Cmd cmd.Constructor = &cc{}

type cc struct {
	zbcontext.Context
}

func (cmd *cc) New(_ *cli.App, config *cmd.Config) cli.Command {
	cmd.Config = config
	cmd.ExcludeVendor = true

	return cli.Command{
		Name:      "clean",
		Usage:     "remove executables in repo produced by build",
		ArgsUsage: "[packages]",
		Action: func(c *cli.Context) error {
			return cmd.run(c.App.Writer, c.Args()...)
		},
	}
}

func (cmd *cc) run(w io.Writer, args ...string) error {
	projects, err := project.Projects(&cmd.Context, args...)
	if err != nil {
		return err
	}

	prefix := cmd.SrcDir + string(filepath.Separator)
	for _, p := range projects {
		for _, pkg := range p.Packages {
			if pkg.IsCommand() {
				path := strings.TrimPrefix(pkg.BuildPath(p), prefix)
				logger := cmd.Logger.WithField("path", path)

				err := os.Remove(path)

				if err == nil {
					logger.Info("removed")
					continue
				}

				if os.IsNotExist(err) {
					logger.Info(err.(*os.PathError).Err.Error())
					continue
				}

				logger.WithError(err).Error("error removing executable")
			}
		}
	}

	return nil
}

package lint

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"golang.org/x/sync/errgroup"

	"github.com/urfave/cli"
	"jrubin.io/slog"
	"jrubin.io/zb/cmd"
	"jrubin.io/zb/lib/project"
	"jrubin.io/zb/lib/zbcontext"
)

// Cmd is the lint command
var Cmd cmd.Constructor = &cc{}

type cc struct {
	zbcontext.Context
	NoMissingComment bool
	IgnoreSuffixes   cli.StringSlice
	Raw              bool

	ignoreSuffixMap map[string]struct{}
}

var defaultIgnoreSuffixes = []string{".pb.go", "_string.go"}

func (cmd *cc) New(_ *cli.App, config *cmd.Config) cli.Command {
	cmd.Config = config

	return cli.Command{
		Name:      "lint",
		Usage:     "gometalinter with cache and better defaults",
		ArgsUsage: "[arguments] [packages]",
		Before: func(c *cli.Context) error {
			cmd.setup()
			return nil
		},
		Action: func(c *cli.Context) error {
			return cmd.run(c.App.Writer, c.Args()...)
		},
		Flags: append(cmd.LintFlags(),
			cli.BoolFlag{
				Name:        "n",
				Usage:       "Hide golint missing comment warnings",
				Destination: &cmd.NoMissingComment,
			},
			cli.StringSliceFlag{
				Name:  "ignore-suffix",
				Usage: fmt.Sprintf("Filter out lint lines from files that have these suffixes (default: %s)", strings.Join(defaultIgnoreSuffixes, ",")),
				Value: &cmd.IgnoreSuffixes,
			},
			cli.BoolFlag{
				Name:        "raw",
				Usage:       "match gometalinter output exactly, don't use logger",
				Destination: &cmd.Raw,
			},
		),
	}
}

func (cmd *cc) setup() {
	if len(cmd.IgnoreSuffixes) == 0 {
		cmd.IgnoreSuffixes = defaultIgnoreSuffixes
	}

	cmd.ignoreSuffixMap = map[string]struct{}{}

	for _, is := range cmd.IgnoreSuffixes {
		if is == "" {
			continue
		}
		cmd.ignoreSuffixMap[is] = struct{}{}
	}

	if filepath.Base(cmd.CacheDir) != "lint" {
		cmd.CacheDir = filepath.Join(cmd.CacheDir, "lint")
	}
}

func (cmd *cc) run(w io.Writer, args ...string) error {
	projects, err := project.Projects(&cmd.Context, args...)
	if err != nil {
		return err
	}

	// run go generate as necessary
	if _, err = projects.Build(project.TargetGenerate); err != nil {
		return err
	}

	// TODO(jrubin) make sure gometalinter is installed (and up to date?)

	pkgs, toRun, err := cmd.buildLists(projects)
	if err != nil {
		return err
	}

	code := zbcontext.ExitOK

	for _, pkg := range pkgs {
		file, err := cmd.CacheFile(pkg)
		if err != nil {
			return err
		}

		if len(toRun) > 0 && toRun[0] == pkg {
			ecode, err := cmd.RunLinter(w, pkg.Package.Dir, file)
			if err != nil {
				return err
			}
			if code == zbcontext.ExitOK {
				code = ecode
			}

			toRun = toRun[1:]
		} else {
			failed, err := cmd.ShowResult(w, file)
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

func (cmd *cc) CacheFile(p *project.Package) (string, error) {
	lintHash, err := p.LintHash(&cmd.Data)
	if err != nil {
		return "", err
	}

	return filepath.Join(
		cmd.CacheDir,
		lintHash[:3],
		fmt.Sprintf("%s.lint", lintHash[3:]),
	), nil
}

const cycle = "cycle"

func (cmd *cc) HaveResult(p *project.Package) (bool, error) {
	if cmd.Force {
		return false, nil
	}

	hash, err := p.Hash()
	if err != nil {
		return false, err
	}

	if hash == cycle {
		return false, nil
	}

	file, err := cmd.CacheFile(p)
	if err != nil {
		return false, err
	}

	fi, err := os.Stat(file)
	return err == nil && fi.Mode().IsRegular(), nil
}

func (cmd *cc) buildLists(projects project.List) (pkgs, toRun project.Packages, err error) {
	for _, proj := range projects {
		for _, pkg := range proj.Packages {
			if pkg.IsVendored {
				continue
			}

			pkgs = append(pkgs, pkg)

			var foundResult bool
			if foundResult, err = cmd.HaveResult(pkg); err != nil {
				return
			}

			if !foundResult {
				toRun = append(toRun, pkg)
			}
		}
	}

	sort.Sort(&pkgs)
	sort.Sort(&toRun)

	return
}

func (cmd *cc) RunLinter(w io.Writer, path, cacheFile string) (int, error) {
	code := zbcontext.ExitOK

	if err := os.MkdirAll(cmd.CacheDir, 0700); err != nil {
		return code, err
	}

	args := cmd.LintArgs()
	args = append(args, path)

	cmd.Logger.Debug(zbcontext.QuoteCommand("→ gometalinter", args))

	pr, pw := io.Pipe()

	ecmd := exec.Command("gometalinter", args...) // nosec
	ecmd.Stdout = pw
	ecmd.Stderr = pw

	if err := ecmd.Start(); err != nil {
		return code, err
	}

	var group errgroup.Group
	group.Go(func() error {
		defer pw.Close()

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

	if err := cmd.readLintOutput(w, pr, cacheFile); err != nil {
		return code, err
	}

	if err := group.Wait(); err != nil {
		return code, err
	}

	return code, nil
}

func (cmd *cc) readLintOutput(w io.Writer, pr io.Reader, file string) error {
	if err := os.MkdirAll(filepath.Dir(file), 0700); err != nil {
		return err
	}

	fd, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}

	defer func() { _ = fd.Close() }() // nosec

	_, err = cmd.readCommon(w, pr, fd)
	return err
}

var (
	levelRE   = regexp.MustCompile(`\A([^:]*):(\d*):(\d*):(\w+): (.*?) \((\w+)\)( \(cached\))?\n\z`)
	commentRE = regexp.MustCompile(` should have comment.* or be unexported`)
)

// Part enum representing each field in a gometalinter line
type Part int

// The different fields of the gometalinter line
const (
	LintFile Part = 1 + iota
	LintLine
	LintColumn
	LintLevel
	LintMessage
	LintLinter
)

func (cmd *cc) readCommon(c io.Writer, pr io.Reader, w io.Writer) (bool, error) {
	r := bufio.NewReader(pr)

	var ew, ww, iw io.WriteCloser
	var def io.Writer

	if cmd.Raw {
		defer func() {
			_, _ = r.WriteTo(c)
		}()
		def = c
	} else {
		ew = cmd.Logger.Writer(slog.ErrorLevel).Prefix("← ")
		ww = cmd.Logger.Writer(slog.WarnLevel).Prefix("← ")
		iw = cmd.Logger.Writer(slog.InfoLevel).Prefix("← ")

		defer func() {
			_ = ew.Close()       // nosec
			_ = ww.Close()       // nosec
			_, _ = r.WriteTo(iw) // nosec
			_ = iw.Close()       // nosec
		}()

		def = iw
	}

	var foundLines bool

LOOP:
	for eof := false; !eof; {
		line, err := r.ReadString('\n')
		if err == io.EOF {
			eof = true
		} else if err != nil {
			return foundLines, err
		}

		m := levelRE.FindStringSubmatch(line)
		if m == nil {
			if w != nil {
				fmt.Fprintf(w, "%s", line)
			}
			if _, err := def.Write([]byte(line)); err != nil {
				return foundLines, err
			}
			continue
		}

		foundLines = true

		if w != nil {
			fmt.Fprintf(w, "%s (cached)\n", strings.TrimSuffix(line, "\n"))
		}

		if cmd.NoMissingComment &&
			m[LintLinter] == "golint" &&
			commentRE.MatchString(m[LintMessage]) {
			continue
		}

		for is := range cmd.ignoreSuffixMap {
			if strings.HasSuffix(m[LintFile], is) {
				continue LOOP
			}
		}

		w := def
		if !cmd.Raw {
			switch m[LintLevel] {
			case "warning":
				w = ww
			case "error":
				w = ew
			}
		}

		if _, err := w.Write([]byte(line)); err != nil {
			return foundLines, err
		}
	}

	return foundLines, nil
}

func (cmd *cc) ShowResult(w io.Writer, cacheFile string) (bool, error) {
	fd, err := os.Open(cacheFile)
	if err != nil {
		return false, err
	}
	defer func() { _ = fd.Close() }() // nosec

	return cmd.readCommon(w, fd, nil)
}

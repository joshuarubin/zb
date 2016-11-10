package zblint

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/urfave/cli"

	"jrubin.io/slog"
	"jrubin.io/zb/lib/project"
	"jrubin.io/zb/lib/zbcontext"
)

type ZBLint struct {
	zbcontext.Context
	NoMissingComment bool
	IgnoreSuffixes   cli.StringSlice
	Raw              bool

	ignoreSuffixMap map[string]struct{}
}

var DefaultIgnoreSuffixes = []string{".pb.go", "_string.go"}

func (l *ZBLint) LintSetup() {
	if l.Raw {
		l.NoWarnTodoFixme = true
	}

	if len(l.IgnoreSuffixes) == 0 {
		l.IgnoreSuffixes = DefaultIgnoreSuffixes
	}

	l.ignoreSuffixMap = map[string]struct{}{}

	for _, is := range l.IgnoreSuffixes {
		if is == "" {
			continue
		}
		l.ignoreSuffixMap[is] = struct{}{}
	}

	if filepath.Base(l.CacheDir) != "lint" {
		l.CacheDir = filepath.Join(l.CacheDir, "lint")
	}
}

func (l *ZBLint) CacheFile(p *project.Package) (string, error) {
	lintHash, err := p.LintHash(&l.Data)
	if err != nil {
		return "", err
	}

	return filepath.Join(
		l.CacheDir,
		lintHash[:3],
		fmt.Sprintf("%s.lint", lintHash[3:]),
	), nil
}

const cycle = "cycle"

func (l *ZBLint) HaveResult(p *project.Package) (bool, error) {
	if l.Data.Force {
		return false, nil
	}

	hash, err := p.Hash()
	if err != nil {
		return false, err
	}

	if hash == cycle {
		return false, nil
	}

	file, err := l.CacheFile(p)
	if err != nil {
		return false, err
	}

	fi, err := os.Stat(file)
	return err == nil && fi.Mode().IsRegular(), nil
}

func (l *ZBLint) ReadResult(w io.Writer, pr io.Reader, file string) error {
	if err := os.MkdirAll(filepath.Dir(file), 0700); err != nil {
		return err
	}

	fd, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}

	defer func() { _ = fd.Close() }() // nosec

	_, err = l.readCommon(w, pr, fd)
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

func (l *ZBLint) readCommon(c io.Writer, pr io.Reader, w io.Writer) (bool, error) {
	r := bufio.NewReader(pr)

	var ew, ww, iw io.WriteCloser
	var def io.Writer

	if l.Raw {
		defer func() {
			_, _ = r.WriteTo(c) // nosec
		}()
		def = c
	} else {
		ew = l.Logger.Writer(slog.ErrorLevel).Prefix("← ")
		ww = l.Logger.Writer(slog.WarnLevel).Prefix("← ")
		iw = l.Logger.Writer(slog.InfoLevel).Prefix("← ")

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

		if l.NoMissingComment &&
			m[LintLinter] == "golint" &&
			commentRE.MatchString(m[LintMessage]) {
			continue
		}

		for is := range l.ignoreSuffixMap {
			if strings.HasSuffix(m[LintFile], is) {
				continue LOOP
			}
		}

		w := def
		if !l.Raw {
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

func (l *ZBLint) ShowResult(w io.Writer, cacheFile string) (bool, error) {
	fd, err := os.Open(cacheFile)
	if err != nil {
		return false, err
	}
	defer func() { _ = fd.Close() }() // nosec

	return l.readCommon(w, fd, nil)
}

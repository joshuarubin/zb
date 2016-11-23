package zblint

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/urfave/cli"

	"jrubin.io/zb/lib/lintflags"
	"jrubin.io/zb/lib/project"
	"jrubin.io/zb/lib/zbcontext"
)

// ZBLint provides methods for working with cached lint result files
type ZBLint struct {
	*zbcontext.Context
	lintflags.Data
	NoMissingComment bool
	IgnoreSuffixes   cli.StringSlice

	ignoreSuffixMap map[string]struct{}
}

// DefaultIgnoreSuffixes lists the file suffixes for which lint results will be
// filtered out
var DefaultIgnoreSuffixes = []string{
	".pb.go",
	".pb.gw.go",
	"_string.go",
	"bindata.go",
	"bindata_assetfs.go",
	"static.go",
}

// LintSetup must be called before other methods to complete the configuration
// from the context
func (l *ZBLint) LintSetup() {
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

// CacheFile returns the location of the lint cache file for a given package
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

// HaveResult checks to see if a lint result is available for a given package
func (l *ZBLint) HaveResult(p *project.Package) (bool, error) {
	if l.Data.Force {
		return false, nil
	}

	file, err := l.CacheFile(p)
	if err != nil {
		return false, err
	}

	fi, err := os.Stat(file)
	return err == nil && fi.Mode().IsRegular(), nil
}

// ReadResult reads lint results from the Reader and writes the unfiltered data
// to the file and the filtered data to the Writer
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

func (l *ZBLint) readCommon(w io.Writer, pr io.Reader, fd io.Writer) (bool, error) {
	r := bufio.NewReader(pr)
	defer func() { _, _ = io.Copy(w, r) }() // nosec

	var buf bytes.Buffer
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
			if fd != nil {
				fmt.Fprintf(&buf, "%s", line)
			}
			if _, err := w.Write([]byte(line)); err != nil {
				return foundLines, err
			}
			continue
		}

		foundLines = true

		if fd != nil {
			fmt.Fprintf(&buf, "%s (cached)\n", strings.TrimSuffix(line, "\n"))
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

		if _, err := w.Write([]byte(line)); err != nil {
			return foundLines, err
		}
	}

	if fd != nil {
		if _, err := buf.WriteTo(fd); err != nil {
			return foundLines, err
		}
	}

	return foundLines, nil
}

// ShowResult reads data from cacheFile and writes the filtered data to the
// Writer
func (l *ZBLint) ShowResult(w io.Writer, cacheFile string) (bool, error) {
	fd, err := os.Open(cacheFile)
	if err != nil {
		return false, err
	}
	defer func() { _ = fd.Close() }() // nosec

	return l.readCommon(w, fd, nil)
}

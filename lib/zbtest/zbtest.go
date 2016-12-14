package zbtest

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"jrubin.io/zb/lib/buildflags"
	"jrubin.io/zb/lib/project"
	"jrubin.io/zb/lib/zbcontext"
)

// ZBTest provides methods for working with cached test result files
type ZBTest struct {
	buildflags.TestFlagsData
	Force bool
}

func (t *ZBTest) TestSetup(ctx zbcontext.Context) zbcontext.Context {
	if filepath.Base(ctx.CacheDir) != "test" {
		ctx.CacheDir = filepath.Join(ctx.CacheDir, "test")
	}
	ctx.BuildArger = &t.TestFlagsData
	ctx.BuildContext = t.TestFlagsData.BuildContext()
	return ctx
}

var endRE = regexp.MustCompile(`\A(\?|ok|FAIL) {0,3}\t([^ \t]+)[ \t]([0-9.]+s|\[.*\])\n\z`)

// CacheFile returns the location of the test cache file for a given package
func (t *ZBTest) CacheFile(ctx zbcontext.Context, p *project.Package) (string, error) {
	testHash, err := p.TestHash(ctx, &t.TestFlagsData)
	if err != nil {
		return "", err
	}

	return filepath.Join(
		ctx.CacheDir,
		testHash[:3],
		fmt.Sprintf("%s.test", testHash[3:]),
	), nil
}

// HaveResult checks to see if a test result is available for a given package
func (t *ZBTest) HaveResult(ctx zbcontext.Context, p *project.Package) (bool, error) {
	if t.Force {
		return false, nil
	}

	file, err := t.CacheFile(ctx, p)
	if err != nil {
		return false, err
	}

	fi, err := os.Stat(file)
	return err == nil && fi.Mode().IsRegular(), nil
}

// StringReader is satisfied by bufio.Reader
type StringReader interface {
	io.Reader
	ReadString(byte) (string, error)
}

// ReadResult from the StringReader and write it to the CacheFile for the
// given package
func (t *ZBTest) ReadResult(ctx zbcontext.Context, r StringReader, p *project.Package) error {
	file, err := t.CacheFile(ctx, p)
	if err != nil {
		return err
	}

	var buf bytes.Buffer

	for eof := false; !eof; {
		line, err := r.ReadString('\n')
		if err == io.EOF {
			eof = true
		} else if err != nil {
			return err
		}

		m := endRE.FindStringSubmatch(line)
		if m == nil {
			if _, err := buf.WriteString(line); err != nil {
				return err
			}
			continue
		}

		fmt.Fprintf(&buf, "%s (cached)\n", strings.TrimSuffix(line, "\n"))

		if err := os.MkdirAll(filepath.Dir(file), 0700); err != nil {
			return err
		}

		if err := ioutil.WriteFile(file, buf.Bytes(), 0600); err != nil {
			return err
		}

		break
	}

	return nil
}

// ShowResult reads the CacheFile for the given package and writes it to the
// Writer
func (t *ZBTest) ShowResult(ctx zbcontext.Context, w io.Writer, p *project.Package) (bool, error) {
	file, err := t.CacheFile(ctx, p)
	if err != nil {
		return false, err
	}

	data, err := ioutil.ReadFile(file)
	if err != nil {
		return false, err
	}

	check := bytes.TrimSpace(data)
	i := bytes.LastIndex(check, []byte{'\n'})
	line := check[i+1:]

	if bytes.HasPrefix(line, []byte("FAIL")) {
		_, err = w.Write(data)
		return false, err
	}

	_, err = w.Write(data)
	return true, err
}

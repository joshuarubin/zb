package zbtest

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"jrubin.io/zb/lib/project"
	"jrubin.io/zb/lib/zbcontext"
)

type ZBTest struct {
	zbcontext.Context
	CacheDir string
}

const (
	cycle = "cycle"
	fail  = "FAIL"
)

var endRE = regexp.MustCompile(`\A(\?|ok|FAIL) {0,3}\t([^ \t]+)[ \t]([0-9.]+s|\[.*\])\n\z`)

func DefaultCacheDir() string {
	if runtime.GOOS == "darwin" {
		return filepath.Join(os.Getenv("HOME"), "Library", "Caches")
	}

	return filepath.Join(os.Getenv("HOME"), ".cache", "go-test-cache")
}

func (t *ZBTest) CacheFile(p *project.Package) (string, error) {
	testHash, err := p.TestHash(&t.TestFlagsData)
	if err != nil {
		return "", err
	}

	return filepath.Join(
		t.CacheDir,
		testHash[:3],
		fmt.Sprintf("%s.test", testHash[3:]),
	), nil
}

func (t *ZBTest) HaveResult(p *project.Package) (bool, error) {
	if t.Force {
		return false, nil
	}

	hash, err := p.Hash()
	if err != nil {
		return false, err
	}

	if hash == cycle {
		return false, nil
	}

	file, err := t.CacheFile(p)
	if err != nil {
		return false, err
	}

	fi, err := os.Stat(file)
	return err == nil && fi.Mode().IsRegular(), nil
}

type StringReader interface {
	io.Reader
	ReadString(byte) (string, error)
}

func (t *ZBTest) ReadResult(ow, ew io.Writer, r StringReader, p *project.Package) error {
	file, err := t.CacheFile(p)
	if err != nil {
		return err
	}

	w := ow
	var buf, obuf bytes.Buffer

	defer func(w *io.Writer) { obuf.WriteTo(*w) }(&w)

	var eof bool
	for !eof {
		line, err := r.ReadString('\n')
		obuf.WriteString(line)
		if err == io.EOF {
			eof = true
		} else if err != nil {
			return err
		}

		m := endRE.FindStringSubmatch(line)
		if m == nil {
			buf.WriteString(line)
			continue
		}

		if m[1] == fail {
			w = ew
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

func (t *ZBTest) ShowResult(ow, ew io.Writer, p *project.Package) (bool, error) {
	testHash, err := p.TestHash(&t.TestFlagsData)
	if err != nil {
		return false, err
	}

	if testHash == "cycle" {
		return false, nil
	}

	file, err := t.CacheFile(p)
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
	if bytes.HasPrefix(line, []byte(fail)) {
		ew.Write(data)
		return false, nil
	}

	ow.Write(data)
	return true, nil
}

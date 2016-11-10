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

	"jrubin.io/zb/lib/project"
	"jrubin.io/zb/lib/zbcontext"
)

type ZBTest struct {
	zbcontext.Context
}

const (
	cycle = "cycle"
	fail  = "FAIL"
)

var endRE = regexp.MustCompile(`\A(\?|ok|FAIL) {0,3}\t([^ \t]+)[ \t]([0-9.]+s|\[.*\])\n\z`)

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

	defer func(w *io.Writer) { _, _ = obuf.WriteTo(*w) }(&w) // nosec

	for eof := false; !eof; {
		line, err := r.ReadString('\n')
		_, oerr := obuf.WriteString(line)
		if err == io.EOF {
			eof = true
		} else if err != nil {
			return err
		}
		if oerr != nil {
			return oerr
		}

		m := endRE.FindStringSubmatch(line)
		if m == nil {
			if _, err := buf.WriteString(line); err != nil {
				return err
			}
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
		_, err = ew.Write(data)
		return false, err
	}

	_, err = ow.Write(data)
	return true, err
}

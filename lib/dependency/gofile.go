package dependency

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"jrubin.io/slog"
	"jrubin.io/zb/lib/zbcontext"
)

var _ Dependency = (*GoFile)(nil)

type GoFile struct {
	*zbcontext.Context
	Path              string
	ProjectImportPath string

	mu           sync.RWMutex
	dependencies []Dependency
}

var goFileCache = map[string]*GoFile{}
var goFileCacheMu sync.RWMutex

func NewGoFile(ctx *zbcontext.Context, projectimportPath, path string) *GoFile {
	goFileCacheMu.RLock()

	if f, ok := goFileCache[path]; ok {
		goFileCacheMu.RUnlock()
		return f
	}

	goFileCacheMu.RUnlock()

	goFileCacheMu.Lock()
	defer goFileCacheMu.Unlock()

	if f, ok := goFileCache[path]; ok {
		return f
	}

	f := &GoFile{
		Context:           ctx,
		Path:              path,
		ProjectImportPath: projectimportPath,
	}

	goFileCache[path] = f

	return f
}

func (e *GoFile) Name() string {
	return e.Path
}

func (e *GoFile) ModTime() time.Time {
	i, err := os.Stat(e.Path)
	if err != nil {
		return time.Time{}
	}
	return i.ModTime()
}

func isZBGenerate(buf []byte) bool {
	return bytes.HasPrefix(buf, []byte("//zb:generate ")) ||
		bytes.HasPrefix(buf, []byte("//zb:generate\t"))
}

func isTodoOrFixme(buf []byte) bool {
	return bytes.Contains(buf, []byte(strings.ToUpper("todo"))) ||
		bytes.Contains(buf, []byte(strings.ToUpper("fixme")))
}

func (e *GoFile) Dependencies() ([]Dependency, error) {
	e.mu.RLock()

	if e.dependencies != nil {
		defer e.mu.RUnlock()
		return e.dependencies, nil
	}

	e.mu.RUnlock()

	e.mu.Lock()
	defer e.mu.Unlock()

	if e.dependencies != nil {
		return e.dependencies, nil
	}

	e.dependencies = []Dependency{}

	file, err := os.Open(e.Path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }() // nosec

	// the following loop taken largely from go source src/cmd/go/generate.go

	// Scan for lines that start "//zb:generate".
	// Can't use bufio.Scanner because it can't handle long lines,
	// which are likely to appear when using generate.
	input := bufio.NewReader(file)
	// One line per loop.
	for i := 1; ; i++ {
		var buf []byte
		buf, err = input.ReadSlice('\n')
		if err == bufio.ErrBufferFull {
			// Line too long - consume and ignore.
			if isZBGenerate(buf) {
				return nil, errors.Errorf("zb:generate directive too long")
			}
			for err == bufio.ErrBufferFull {
				_, err = input.ReadSlice('\n')
			}
			if err != nil {
				return nil, err
			}
			continue
		}

		if err != nil {
			// Check for marker at EOF without final \n.
			if err == io.EOF && isZBGenerate(buf) {
				err = io.ErrUnexpectedEOF
			}
			break
		}

		if !e.NoWarnTodoFixme {
			base := e.Context.ImportPathToDir(e.ProjectImportPath) + string(filepath.Separator)
			if strings.HasPrefix(e.Path, base) &&
				!strings.Contains(e.Path, "vendor/") &&
				isTodoOrFixme(buf) {
				e.Logger.Warn(fmt.Sprintf("%s:%d:%s",
					strings.TrimPrefix(e.Path, base),
					i,
					strings.TrimSpace(string(buf)),
				))
			}
		}

		if !isZBGenerate(buf) {
			continue
		}

		var words []string
		words, err = split(string(buf))
		if err != nil {
			return nil, err
		}

		if len(words) == 0 {
			return nil, errors.New("no arguments to directive")
		}

		var deps []*GoGenerateFile
		deps, err = e.parseZBGenerate(words)
		if err != nil {
			return nil, err
		}

		for _, dep := range deps {
			var source string
			source, err = filepath.Rel(e.SrcDir, dep.Path)
			if err != nil {
				source = dep.Path
			}

			var dependsOn string
			dependsOn, err = filepath.Rel(e.SrcDir, dep.Depends.Name())
			if err != nil {
				dependsOn = dep.Depends.Name()
			}

			var fromGo string
			fromGo, err = filepath.Rel(e.SrcDir, e.Path)
			if err != nil {
				fromGo = e.Path
			}

			e.Logger.WithFields(slog.Fields{
				"source":       source,
				"depends_on":   dependsOn,
				"from_go_file": fromGo,
			}).Debug("found go:generate dependency")

			e.dependencies = append(e.dependencies, dep)
		}
	}
	if err != nil && err != io.EOF {
		return nil, errors.Wrapf(err, "error reading %s", e.Path)
	}

	return e.dependencies, nil
}

func split(line string) ([]string, error) {
	// Parse line, obeying quoted strings.
	var words []string
	line = line[len("//zb:generate ") : len(line)-1] // Drop preamble and final newline.
	// There may still be a carriage return.
	if len(line) > 0 && line[len(line)-1] == '\r' {
		line = line[:len(line)-1]
	}
	// One (possibly quoted) word per iteration.
Words:
	for {
		line = strings.TrimLeft(line, " \t")
		if len(line) == 0 {
			break
		}
		if line[0] == '"' {
			for i := 1; i < len(line); i++ {
				c := line[i] // Only looking for ASCII so this is OK.
				switch c {
				case '\\':
					if i+1 == len(line) {
						return nil, errors.New("bad backslash")
					}
					i++ // Absorb next byte (If it's a multibyte we'll get an error in Unquote).
				case '"':
					word, err := strconv.Unquote(line[0 : i+1])
					if err != nil {
						return nil, errors.New("bad quoted string")
					}
					words = append(words, word)
					line = line[i+1:]
					// Check the next character is space or end of line.
					if len(line) > 0 && line[0] != ' ' && line[0] != '\t' {
						return nil, errors.New("expect space after quoted argument")
					}
					continue Words
				}
			}
			return nil, errors.New("mismatched quoted string")
		}
		i := strings.IndexAny(line, " \t")
		if i < 0 {
			i = len(line)
		}
		words = append(words, line[0:i])
		line = line[i:]
	}

	return words, nil
}

func (e *GoFile) parseGlobs(words []string) ([]string, error) {
	var files []string

	for _, word := range words {
		word = filepath.Join(filepath.Dir(e.Path), word)

		if !strings.Contains(word, "*") {
			files = append(files, word)
			continue
		}

		matches, err := filepath.Glob(word)
		if err != nil {
			return nil, err
		}

		files = append(files, matches...)
	}

	return files, nil
}

func (e *GoFile) parseZBGenerate(words []string) ([]*GoGenerateFile, error) {
	// formats to parse:
	// 1. -patsubst %pattern %replacement glob glob... (like Make)
	// 2. -target (basically -patsubst % word glob...)
	// 3. glob glob...

	if words[0] == "-patsubst" {
		return e.parsePatSubst(words[1:])
	}

	if words[0] == "-target" {
		return e.parseTarget(words[1:])
	}

	files, err := e.parseGlobs(words)
	if err != nil {
		return nil, err
	}

	var deps []*GoGenerateFile
	for _, file := range files {
		// run go generate on `e.Path` if `e.Path` is newer than `file`
		deps = append(deps, &GoGenerateFile{
			GoFile:  e,
			Depends: File(e.Path),
			Path:    file,
		})
	}
	return deps, nil
}

func (e *GoFile) parseTarget(words []string) ([]*GoGenerateFile, error) {
	if len(words) < 2 {
		return nil, errors.New("invalid target")
	}

	target := words[0]
	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(e.Path), target)
	}

	files, err := e.parseGlobs(words[1:])
	if err != nil {
		return nil, err
	}

	var deps []*GoGenerateFile
	for _, file := range files {
		deps = append(deps, &GoGenerateFile{
			GoFile:  e,
			Depends: File(file),
			Path:    target,
		})
	}

	return deps, nil
}

func (e *GoFile) parsePatSubst(words []string) ([]*GoGenerateFile, error) {
	if len(words) < 3 {
		return nil, errors.New("invalid patsubst")
	}

	pattern := words[0]
	replacement := words[1]

	files, err := e.parseGlobs(words[2:])
	if err != nil {
		return nil, err
	}

	var deps []*GoGenerateFile
	for _, file := range files {
		sfile := substitute(pattern, replacement, file)
		if !filepath.IsAbs(sfile) {
			sfile = filepath.Join(filepath.Dir(e.Path), sfile)
		}
		if sfile != "" {
			// run go generate on `e.Path` if `file` is newer than `sfile`
			deps = append(deps, &GoGenerateFile{
				GoFile:  e,
				Depends: File(file),
				Path:    sfile,
			})
		}
	}

	return deps, nil
}

func findPercent(value string) int {
	// find first % not preceded by \ and return its position
	// returns -1 if not found
	var prev rune
	for i, c := range value {
		if c == '%' && prev != '\\' {
			return i
		}

		prev = c
	}

	return -1
}

func substitute(pattern, replacement, file string) string {
	pp := findPercent(pattern)

	if pp == -1 {
		// there was no % in the pattern

		if pattern == file {
			return replacement
		}

		return ""
	}

	// pattern had a %

	prefix := pattern[:pp]
	if !strings.HasPrefix(file, prefix) {
		return ""
	}

	match := file
	if len(file) >= len(prefix) {
		match = file[len(prefix):]
	}

	suffix := ""
	if len(pattern) > pp {
		suffix = pattern[pp+1:]
	}

	if !strings.HasSuffix(match, suffix) {
		return ""
	}

	match = match[:len(match)-len(suffix)]

	// the pattern matched

	pr := findPercent(replacement)
	if pr == -1 {
		// there was no % in the replacement
		return replacement
	}

	ret := replacement[:pr] + match
	if len(replacement) > pr {
		ret += replacement[pr+1:]
	}

	return ret
}

func (e *GoFile) Buildable() bool {
	return false
}

func (e *GoFile) Build() error {
	// noop
	return nil
}

func (e *GoFile) Install() error {
	// noop
	return nil
}

func (e *GoFile) Generate() error {
	args := []string{"generate"}
	if e.GenerateRun != "" {
		args = append(args, "-run", e.GenerateRun)
	}
	args = append(args, e.BuildArgs(nil, nil)...)
	args = append(args, e.Path)

	err := e.GoExec(args...)
	return err
}

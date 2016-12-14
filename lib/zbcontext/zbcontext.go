package zbcontext

import (
	"bytes"
	"go/build"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"gopkg.in/src-d/go-git.v4/core"

	"github.com/urfave/cli"

	"jrubin.io/slog"
	"jrubin.io/zb/lib/ellipsis"
)

type BuildArger interface {
	BuildArgs(*build.Package, *core.Hash) []string
}

// Context for package related commands
type Context struct {
	Logger               slog.Interface
	GitCommit, BuildDate *string
	NoWarnTodoFixme      bool
	CacheDir             string
	Package              bool
	BuildContext         *build.Context
	BuildArger

	ExcludeVendor bool
}

// Import returns details about the Go package named by the import path,
// interpreting local import paths relative to the srcDir directory.
// If the path is a local import path naming a package that can be imported
// using a standard import path, the returned package will set p.ImportPath
// to that path.
//
// In the directory containing the package, .go, .c, .h, and .s files are
// considered part of the package except for:
//
//	- .go files in package documentation
//	- files starting with _ or . (likely editor temporary files)
//	- files with build constraints not satisfied by the context
//
// If an error occurs, Import returns a non-nil error and a non-nil
// *Package containing partial information.
func (ctx Context) Import(path, srcDir string) (*build.Package, error) {
	return ctx.buildContext().Import(path, srcDir, build.ImportComment)
}

func (ctx Context) buildContext() *build.Context {
	if ctx.BuildContext != nil {
		return ctx.BuildContext
	}
	return &build.Default
}

var CWD string

func init() {
	var err error
	if CWD, err = os.Getwd(); err != nil {
		panic(err)
	}
}

func (ctx *Context) NormalizeImportPath(importPath string) string {
	if !build.IsLocalImport(importPath) {
		return importPath
	}

	// convert local imports to import paths

	if !filepath.IsAbs(importPath) {
		// convert relative path to absolute
		importPath = filepath.Join(CWD, importPath)
	}

	if found := ctx.DirToImportPath(importPath); found != "" {
		return found
	}

	return importPath
}

func (ctx *Context) NoGoImportPathToProjectImportPaths(importPath string) []string {
	dir := ctx.ImportPathToProjectDir(importPath)
	if dir == "" {
		return nil
	}

	// found project dir, now convert it back to an import path so
	// we can use ellipsis
	importPath = ctx.DirToImportPath(dir)
	if importPath == "" {
		return nil
	}

	// add the ellipsis
	importPath = filepath.Join(importPath, "...")

	// lets see if we can find any packages under it
	return ctx.ExpandEllipsis(importPath)
}

func (ctx *Context) DirToImportPath(dir string) string {
	// path may be a/b/c/d
	// p.Dir may be /home/user/go/src/a/b
	// this will return a/b even if there are no .go files in it
	// e.g. it may not be a valid import path

	for _, srcDir := range ctx.buildContext().SrcDirs() {
		srcDir += string(filepath.Separator)
		if strings.HasPrefix(dir, srcDir) {
			return strings.TrimPrefix(dir, srcDir)
		}

		// this can happen if the project dir is outside the $GOPATH but
		// includes its own `src` dir that is in the $GOPATH
		if strings.HasPrefix(srcDir, dir+string(filepath.Separator)) {
			// return the relative path to it from CWD
			// this is necessary since the ellipsis can't expand absolute paths
			rel, err := filepath.Rel(CWD, srcDir)
			if err == nil {
				return rel
			}
		}
	}
	return ""
}

func QuoteCommand(command string, args []string) string {
	for _, a := range args {
		if strings.Contains(a, " ") {
			a = strconv.Quote(a)
		}
		command += " " + a
	}
	return command
}

func (ctx *Context) GoExec(args ...string) error {
	ctx.Logger.Info(QuoteCommand("→ go", args))

	var buf bytes.Buffer
	cmd := exec.Command("go", args...) // nosec
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	code, err := ExitCode(cmd.Run())
	if err != nil {
		return err
	}

	level := slog.InfoLevel
	if code != ExitOK {
		level = slog.ErrorLevel
	}
	w := ctx.Logger.Writer(level).Prefix("← ")
	defer func() { _ = w.Close() }() // nosec

	if _, err = io.Copy(w, &buf); err != nil {
		return err
	}

	if code != ExitOK {
		return cli.NewExitError("", code)
	}

	return nil
}

const (
	ExitOK = iota
	ExitFailed
	ExitSignaled = 98 + iota
	ExitStopped
	ExitContinued
	ExitCoreDump
)

func ExitCode(err error) (int, error) {
	if err == nil {
		return ExitOK, nil
	}

	eerr, ok := err.(*exec.ExitError)
	if !ok {
		return 0, err
	}

	status, ok := eerr.Sys().(syscall.WaitStatus)
	if !ok {
		return 0, err
	}

	switch {
	case status.Exited():
		return status.ExitStatus(), nil
	case status.Signaled():
		return ExitSignaled, nil
	case status.Stopped():
		return ExitStopped, nil
	case status.Continued():
		return ExitContinued, nil
	case status.CoreDump():
		return ExitCoreDump, nil
	}

	return 0, err
}

func (ctx *Context) Touch(path string) error {
	now := time.Now()
	// ctx.Logger.WithField("path", path).Debug("touch")
	return os.Chtimes(path, now, now)
}

func (ctx *Context) ImportPathToProjectDir(importPath string) string {
	dir := ctx.ImportPathToDir(importPath)
	if dir == "" {
		return ""
	}
	return GitDir(dir)
}

func (ctx *Context) ImportPathToDir(importPath string) string {
	// can't handle ellipsis (...), but does not require .go files to exist either

	for _, srcDir := range ctx.buildContext().SrcDirs() {
		dir := filepath.Join(srcDir, importPath)
		info, err := os.Stat(dir)
		if err != nil || !info.IsDir() {
			continue
		}
		return dir
	}
	return ""
}

func (ctx *Context) ExpandEllipsis(args ...string) []string {
	return ellipsis.Expand(ctx.buildContext(), ctx.Logger, args...)
}

// GitDir checks the directory value for the presence of .git and will walk up
// the filesystem hierarchy to find one. Returns an empty string if no directory
// containing .git was found.
func GitDir(value string) string {
	dir := value
	for {
		test := filepath.Join(dir, ".git")
		_, err := os.Stat(test)
		if err != nil {
			ndir := filepath.Dir(dir)
			if ndir == dir {
				return ""
			}

			dir = ndir
			continue
		}

		return dir
	}
}

func BuildPath(baseDir string, pkg *build.Package) string {
	if !pkg.IsCommand() {
		return InstallPath(pkg)
	}

	if baseDir == "" {
		baseDir = pkg.Dir
	}

	name := filepath.Base(pkg.Dir)

	path := filepath.Join(baseDir, name)

	if pkg.Dir == path {
		path = filepath.Join(pkg.Dir, name)
	}

	return path
}

func InstallPath(pkg *build.Package) string {
	path := pkg.PkgObj

	if pkg.IsCommand() {
		path = filepath.Join(pkg.BinDir, filepath.Base(pkg.Dir))
	}

	return path
}

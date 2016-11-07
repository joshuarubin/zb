package zbcontext

import (
	"go/build"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"jrubin.io/slog"
	"jrubin.io/zb/lib/buildflags"
	"jrubin.io/zb/lib/ellipsis"
)

// Context for package related commands
type Context struct {
	buildflags.BuildFlags
	SrcDir        string
	ExcludeVendor bool
	Logger        *slog.Logger
	GenerateRun   string
}

func (ctx *Context) NoGoImportPathToProjectImportPaths(importPath string) []string {
	dir := ctx.ImportPathToProjectDir(importPath)
	if dir == "" {
		return nil
	}

	// found project dir, now convert it back to an import path so
	// we can use ellipsis
	importPath = ctx.DirToImportPath(dir)

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

	for _, srcDir := range ctx.BuildContext().SrcDirs() {
		srcDir += "/"
		if strings.HasPrefix(dir, srcDir) {
			return dir[len(srcDir):]
		}
	}
	return ""
}

func quoteCommand(command string, args []string) string {
	for _, a := range args {
		if strings.Contains(a, " ") {
			a = strconv.Quote(a)
		}
		command += " " + a
	}
	return command
}

func (ctx Context) GoExec(args ...string) error {
	ctx.Logger.Info(quoteCommand("> go", args))

	writer := ctx.Logger.Writer(slog.InfoLevel).Prefix("< ")
	defer writer.Close()

	cmd := exec.Command("go", args...) // nosec
	cmd.Stdout = writer
	cmd.Stderr = writer
	return cmd.Run()
}

func (ctx Context) Touch(path string) error {
	ctx.Logger.Debug("> touch " + path)
	now := time.Now()
	return os.Chtimes(path, now, now)
}

func (ctx Context) ImportPathToProjectDir(importPath string) string {
	dir := ctx.ImportPathToDir(importPath)
	if dir == "" {
		return ""
	}
	return GitDir(dir)
}

// can't handle ellipsis (...), but does not require .go files to exist either
func (ctx *Context) ImportPathToDir(importPath string) string {
	for _, srcDir := range ctx.BuildContext().SrcDirs() {
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
	return ellipsis.Expand(ctx.BuildContext(), ctx.Logger, args...)
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

func BuildPath(projectDir string, pkg *build.Package) string {
	if !pkg.IsCommand() {
		return InstallPath(pkg)
	}

	name := filepath.Base(pkg.Dir)

	path := filepath.Join(projectDir, name)
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

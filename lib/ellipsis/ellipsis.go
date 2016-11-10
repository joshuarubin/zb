package ellipsis

import (
	"go/build"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"jrubin.io/slog"
)

// Taken nearly verbatim from go source src/cmd/go/main.go
// Only change is to export importPaths()

const (
	std = "std"
	cmd = "cmd"
)

var (
	goroot    = filepath.Clean(runtime.GOROOT())
	gorootSrc = filepath.Join(goroot, "src")
)

// Expand ellipsis in any string in args
func Expand(buildContext *build.Context, logger slog.Interface, args ...string) []string {
	args = importPathsNoDotExpansion(buildContext, args, logger)
	var out []string
	for _, a := range args {
		if strings.Contains(a, "...") {
			if build.IsLocalImport(a) {
				out = append(out, allPackagesInFS(buildContext, a, logger)...)
			} else {
				out = append(out, allPackages(buildContext, a, logger)...)
			}
			continue
		}
		out = append(out, a)
	}
	return out
}

// importPathsNoDotExpansion returns the import paths to use for the given
// command line, but it does no ... expansion.
func importPathsNoDotExpansion(buildContext *build.Context, args []string, logger slog.Interface) []string {
	if len(args) == 0 {
		return []string{"."}
	}
	var out []string
	for _, a := range args {
		// Arguments are supposed to be import paths, but
		// as a courtesy to Windows developers, rewrite \ to /
		// in command-line arguments. Handles .\... and so on.
		if filepath.Separator == '\\' {
			a = strings.Replace(a, `\`, `/`, -1)
		}

		// Put argument in canonical form, but preserve leading ./.
		if strings.HasPrefix(a, "./") {
			a = "./" + path.Clean(a)
			if a == "./." {
				a = "."
			}
		} else {
			a = path.Clean(a)
		}
		if isMetaPackage(a) {
			out = append(out, allPackages(buildContext, a, logger)...)
			continue
		}
		out = append(out, a)
	}
	return out
}

// allPackages returns all the packages that can be found
// under the $GOPATH directories and $GOROOT matching pattern.
// The pattern is either "all" (all packages), "std" (standard packages),
// "cmd" (standard commands), or a path including "...".
func allPackages(buildContext *build.Context, pattern string, logger slog.Interface) []string {
	pkgs := matchPackages(buildContext, pattern)
	if len(pkgs) == 0 {
		logger.WithField("pattern", pattern).Warn("matched no packages")
	}
	return pkgs
}

// treeCanMatchPattern(pattern)(name) reports whether
// name or children of name can possibly match pattern.
// Pattern is the same limited glob accepted by matchPattern.
func treeCanMatchPattern(pattern string) func(name string) bool {
	wildCard := false
	if i := strings.Index(pattern, "..."); i >= 0 {
		wildCard = true
		pattern = pattern[:i]
	}
	return func(name string) bool {
		return len(name) <= len(pattern) && hasPathPrefix(pattern, name) ||
			wildCard && strings.HasPrefix(name, pattern)
	}
}

// hasPathPrefix reports whether the path s begins with the
// elements in prefix.
func hasPathPrefix(s, prefix string) bool {
	switch {
	default:
		return false
	case len(s) == len(prefix):
		return s == prefix
	case len(s) > len(prefix):
		if prefix != "" && prefix[len(prefix)-1] == '/' {
			return strings.HasPrefix(s, prefix)
		}
		return s[len(prefix)] == '/' && s[:len(prefix)] == prefix
	}
}

func matchPackages(buildContext *build.Context, pattern string) []string {
	match := func(string) bool { return true }
	treeCanMatch := func(string) bool { return true }
	if !isMetaPackage(pattern) {
		match = matchPattern(pattern)
		treeCanMatch = treeCanMatchPattern(pattern)
	}

	have := map[string]bool{
		"builtin": true, // ignore pseudo-package that exists only for documentation
	}
	if !buildContext.CgoEnabled {
		have["runtime/cgo"] = true // ignore during walk
	}
	var pkgs []string

	for _, src := range buildContext.SrcDirs() {
		if (pattern == std || pattern == cmd) && src != gorootSrc {
			continue
		}
		src = filepath.Clean(src) + string(filepath.Separator)
		root := src
		if pattern == cmd {
			root += cmd + string(filepath.Separator)
		}
		err := filepath.Walk(root, func(path string, fi os.FileInfo, err error) error {
			if err != nil || !fi.IsDir() || path == src {
				return nil
			}

			// Avoid .foo, _foo, and testdata directory trees.
			_, elem := filepath.Split(path)
			if strings.HasPrefix(elem, ".") || strings.HasPrefix(elem, "_") || elem == "testdata" {
				return filepath.SkipDir
			}

			name := filepath.ToSlash(path[len(src):])
			if pattern == std && (!isStandardImportPath(name) || name == cmd) {
				// The name "std" is only the standard library.
				// If the name is cmd, it's the root of the command tree.
				return filepath.SkipDir
			}
			if !treeCanMatch(name) {
				return filepath.SkipDir
			}
			if have[name] {
				return nil
			}
			have[name] = true
			if !match(name) {
				return nil
			}
			_, err = buildContext.ImportDir(path, 0)
			if err != nil {
				if _, noGo := err.(*build.NoGoError); noGo {
					return nil
				}
			}
			pkgs = append(pkgs, name)
			return nil
		})
		if err != nil {
			panic(err)
		}
	}
	return pkgs
}

// isStandardImportPath reports whether $GOROOT/src/path should be considered
// part of the standard distribution. For historical reasons we allow people to add
// their own code to $GOROOT instead of using $GOPATH, but we assume that
// code will start with a domain name (dot in the first element).
func isStandardImportPath(path string) bool {
	i := strings.Index(path, "/")
	if i < 0 {
		i = len(path)
	}
	elem := path[:i]
	return !strings.Contains(elem, ".")
}

// matchPattern(pattern)(name) reports whether
// name matches pattern. Pattern is a limited glob
// pattern in which '...' means 'any string' and there
// is no other special syntax.
func matchPattern(pattern string) func(name string) bool {
	re := regexp.QuoteMeta(pattern)
	re = strings.Replace(re, `\.\.\.`, `.*`, -1)
	// Special case: foo/... matches foo too.
	if strings.HasSuffix(re, `/.*`) {
		re = re[:len(re)-len(`/.*`)] + `(/.*)?`
	}
	reg := regexp.MustCompile(`^` + re + `$`)
	return func(name string) bool {
		return reg.MatchString(name)
	}
}

// isMetaPackage checks if name is a reserved package name that expands to multiple packages
func isMetaPackage(name string) bool {
	return name == std || name == cmd || name == "all"
}

// allPackagesInFS is like allPackages but is passed a pattern
// beginning ./ or ../, meaning it should scan the tree rooted
// at the given directory. There are ... in the pattern too.
func allPackagesInFS(buildContext *build.Context, pattern string, logger slog.Interface) []string {
	pkgs := matchPackagesInFS(buildContext, pattern, logger)
	if len(pkgs) == 0 {
		logger.WithField("pattern", pattern).Warn("matched no packages")
	}
	return pkgs
}

func matchPackagesInFS(buildContext *build.Context, pattern string, logger slog.Interface) []string {
	// Find directory to begin the scan.
	// Could be smarter but this one optimization
	// is enough for now, since ... is usually at the
	// end of a path.
	i := strings.Index(pattern, "...")
	dir, _ := path.Split(pattern[:i])

	// pattern begins with ./ or ../.
	// path.Clean will discard the ./ but not the ../.
	// We need to preserve the ./ for pattern matching
	// and in the returned import paths.
	prefix := ""
	if strings.HasPrefix(pattern, "./") {
		prefix = "./"
	}
	match := matchPattern(pattern)

	var pkgs []string
	err := filepath.Walk(dir, func(path string, fi os.FileInfo, err error) error {
		if err != nil || !fi.IsDir() {
			return nil
		}
		if path == dir {
			// filepath.Walk starts at dir and recurses. For the recursive case,
			// the path is the result of filepath.Join, which calls filepath.Clean.
			// The initial case is not Cleaned, though, so we do this explicitly.
			//
			// This converts a path like "./io/" to "io". Without this step, running
			// "cd $GOROOT/src; go list ./io/..." would incorrectly skip the io
			// package, because prepending the prefix "./" to the unclean path would
			// result in "././io", and match("././io") returns false.
			path = filepath.Clean(path)
		}

		// Avoid .foo, _foo, and testdata directory trees, but do not avoid "." or "..".
		_, elem := filepath.Split(path)
		dot := strings.HasPrefix(elem, ".") && elem != "." && elem != ".."
		if dot || strings.HasPrefix(elem, "_") || elem == "testdata" {
			return filepath.SkipDir
		}

		name := prefix + filepath.ToSlash(path)
		if !match(name) {
			return nil
		}

		// We keep the directory if we can import it, or if we can't import it
		// due to invalid Go source files. This means that directories containing
		// parse errors will be built (and fail) instead of being silently skipped
		// as not matching the pattern. Go 1.5 and earlier skipped, but that
		// behavior means people miss serious mistakes.
		// See golang.org/issue/11407.
		if p, err := buildContext.ImportDir(path, 0); err != nil && (p == nil || len(p.InvalidGoFiles) == 0) {
			if _, noGo := err.(*build.NoGoError); !noGo {
				logger.WithError(err).Warn("")
			}
			return nil
		}
		pkgs = append(pkgs, name)
		return nil
	})
	if err != nil {
		panic(err)
	}
	return pkgs
}

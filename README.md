# zb — an opinionated repo based tool for linting, testing and building go source

[![GoDoc](https://godoc.org/jrubin.io/zb?status.svg)](https://godoc.org/jrubin.io/zb) [![Go Report Card](https://goreportcard.com/badge/jrubin.io/zb)](https://goreportcard.com/report/jrubin.io/zb)

<pre>
███████╗██████╗     ██████╗  ██████╗ ███████╗███████╗    ██╗████████╗
╚══███╔╝██╔══██╗    ██╔══██╗██╔═══██╗██╔════╝██╔════╝    ██║╚══██╔══╝
  ███╔╝ ██████╔╝    ██║  ██║██║   ██║█████╗  ███████╗    ██║   ██║
 ███╔╝  ██╔══██╗    ██║  ██║██║   ██║██╔══╝  ╚════██║    ██║   ██║
███████╗██████╔╝    ██████╔╝╚██████╔╝███████╗███████║    ██║   ██║
╚══════╝╚═════╝     ╚═════╝  ╚═════╝ ╚══════╝╚══════╝    ╚═╝   ╚═╝
</pre>

[[Help! your logo here](https://github.com/joshuarubin/zb/issues/1)]

## Benefits

* Faster builds (by defaulting to `go install` except for `main` packages, and by running concurrent `go install` commands when the dependency tree allows)
* Faster testing (by caching test results and not retesting except when necessary)
* Faster linting (by caching lint results from [`gometalinter`](https://github.com/alecthomas/gometalinter))
* Did I mention _fast_!
* Automatically runs `go generate` if its dependency calculation determines it's required
* Operates on all packages in a repository (by default) with intelligent support for vendored packages
* Can complement other build tools like `make`
* Does not interfere with tools like [`govendor`](https://github.com/kardianos/govendor) or [`gb`](https://getgb.io/)

## Installation

Simply run `go get jrubin.io/zb`

## Rationale

Many go repositories have multiple packages. It is often necessary to build/install, test, lint, etc. across all packages within the repository. Go "ellipsis" wildcards (`...`) can be used to select all subdirectories of a given repository, but don't exclude vendored packages which makes running tests and linting complicated. Some operations should be aware of vendored packages (e.g. build/install), while others should ignore them (e.g. lint). Yet others need to be aware of changes in vendored packages, but should not operate directly on them (e.g. test). Traditional build tools, like `make`, can be used to supplement the `go` command to work only on the intended packages. Building dependency lists for make targets, which are required, for example, to dynamically identify modified `.go` files and what would need to be rebuilt as a result, is at the very least complicated (consider `.go` files modified outside the repo) and at best slow.

### `zb` fixes all of this

## Packages → Repositories

`zb` is aware of the directory it is called from. If any of its commands is called without a package argument, it will use the current working directory.

`zb` can also be passed packages just like the `go` command. Both package names (e.g. `fmt`, `jrubin.io/zb`) and relative package names (e.g. `./jrubin.io/zb`) are supported, as are ellipsis (`...`).

`zb` will identify the repository associated with each package by locating the directory and walking up the directory tree to find the repository directory (containing `.git` [only `git` is supported at present]). It will then execute the command for all packages in each repository it identified.

## go generate

`go generate` is great, but sometimes it needs to be executed before a build. Forgetting to execute `go generate` can be a major problem if, for example, new values were added to a `stringer`.

`zb` does its own dependency calculation and can identify `go generate` dependencies provided an additional annotation is also present.

The following formats are available to define dependencies of `go generate`

### `zb:generate` formats

* `//zb:generate glob glob...`
  Causes `go generate` to be executed on the go file with the annotation if the go file itself is newer than the files expanded from the globs. This is useful with commands like stringer:

    ```go
    //go:generate stringer -type=YourType
    //zb:generate yourtype_string.go
    ```

* `//zb:generate -patsubst %pattern %replacement glob glob...`
  Works like `make`'s [patsubst](https://www.gnu.org/software/make/manual/html_node/Text-Functions.html).
  Causes `go generate` to be executed on the go file with the annotation if any of the files expanded from the globs is newer than any of the filenames generated through the pattern substitution.

    ```go
    //go:generate make proto
    //zb:generate -patsubst %.proto %.pb.go *.proto
    ```

    This command first matches all files matching `*.proto` and then performs the substitution by extracting the part of each of those file names before `.proto` (identified with the `%` in `%.proto`) and taking that extracted pattern and inserting it into the `%` part of `%.pb.go`.

    So if there were files `message.proto` and `types.proto`, `go generate` would be executed if either of those files were newer than `message.pb.go` or `types.pb.go` (including if the `.pb.go` files did not yet exist).

* `//zb:generate -target file glob glob...`
  Basically a simplified `-patsubst`. Causes `go generate` to be executed if any of the files expanded from the globs is newer than `file`
  Can also be written as `//zb:generate -patsubst % file glob glob...`

## Commands

### install

Initially, `zb install` appears to do the same things as `go install` (just for all packages in the repositories). In fact, `zb install` just calls `go install` under the hood and supports all of its flags. There are a few differences though.

* `go generate` may be called before building the package according to the `//go:generate` and `//zb:generate` annotations
* `main` packages (commands) are built with extra linker flags that cause `main.gitCommit` and `main.buildDate` variables to be set if they exist. See [`zb/main.go`](https://github.com/joshuarubin/zb/blob/master/main.go) as an example of how to utilize this.
* Executes `go install` for each stale package it finds and will execute concurrent `go install` processes when the dependency tree allows. Concurrency can be limited with `$GOMAXPROCS`.
* If any of the non-vendored `.go` files in the repository contain `TODO` or `FIXME` these lines will be emitted to the console as warnings (unless the global `-n` flag is enabled).

### build

`zb build` differs from `go build` in that non `main` packages will be installed with `go install`. Commands, however, will not be installed (to `$GOPATH/bin`), they are built with the binary being placed in the root of the repository tree. If there is already a directory in the root of the repository with the same name as the command, the command will be placed in that directory instead.

Otherwise, `zb build` is identical to `zb install`.

### lint

Delegates functionality to [`gometalinter`](https://github.com/alecthomas/gometalinter) but with more useful defaults and caching of results.

* The `--concurrency, -j` flag is dynamically calculated to be `1` less than half the number of CPU cores (but at least `1`) [`gometalinter` default is `16`]
* `--tests` is enabled by default
* `--deadline` is set to `30s` [`gometalinter` default is `5s`]
* `--enable-gc` is enabled by default
* `alighcheck`, `dupl`, `gocyclo` and `structcheck` are disabled by default
* `errcheck`, `gofmt`, `goimports` and `unused` are enabled (in addition to all other default enabled checkers) by default

The `-n` flag can be used to hide `golint` warnings about missing comments.

Since dependency calculation can sometimes add a non-trivial amount of time to the `zb lint` command, `go generate` will not be executed.

Files matching certain suffixes will be excluded from the results. This list can be modified with the `--ignore-suffix` flag. By default files with the following suffixes will be excluded:

* `.pb.go`
* `.pb.gw.go`
* `_string.go`
* `bindata.go`
* `bindata_assetfs.go`
* `static.go`

All other [`gometalinter`](https://github.com/alecthomas/gometalinter) flags will be honored as defined.

### test

Delegates functionality to `go test` but caches the results (like [`gt`](https://godoc.org/rsc.io/gt)).
Honors all other flags just like `go test` except those intended to be passed directly to the test binary.

Use the `-f` flag to treat the test results as uncached, forcing the tests to be executed (and cached) again.

To see which tests would be executed (because their results are not-cached or the `-f` flag was provided), use the `-l` flag.

Since dependency calculation can sometimes add a non-trivial amount of time to the `zb test` command, `go generate` will not be executed.

### complete

`zb` has full support for shell autocompletion in both `bash` and `zsh`.
Simply execute `eval "$(zb complete)"` (or put in your init files) to enable.

### clean

Removes the executables produced by `zb build`

### commands

Lists the absolute paths where each of the commands (from `main` packages) will be placed with `zb build`

### list

Similar to `go list` (and takes the same flags) but will list all of the packages in each of the repositories. Use the `--vendor` flag to exclude vendored packages.

### help

`zb` contains a built-in, comprehensive help system. Running `zb` by itself (or with the `-h` or `--help` flags) will list the commands and global flags. `zb help <command>`, `zb <command> -h` and `zb <command> --help` will show contextual help for the given command.

## Global Flags

### `--log-level, -l, $LOG_LEVEL`

Defaults to `info`. Available levels are:

* `error`
* `warn`
* `info`
* `debug`

### `--no-warn-todo-fixme, -n, $NO_WARN_TODO_FIXME`

Do not warn when finding WARN or FIXME in `.go` files

### `--cache, $CACHE`

Modify the base directory used for storing results of commands that cache their results (`test` and `lint`).
Defaults to `$HOME/Library/Caches/zb` on mac and `$HOME/.cache/zb` elsewhere.

### `--package, -p`

Causes `zb` to execute only on the explicitly listed packages and not on all packages in their repositories.

## Still Planned

* Support for other version control systems [[#2](https://github.com/joshuarubin/zb/issues/2)]
* Complete all `godoc` documentation [[#3](https://github.com/joshuarubin/zb/issues/3)]
* Add comprehensive testing [[#4](https://github.com/joshuarubin/zb/issues/4)]
* Detect import cycles in dependency calculation [[#5](https://github.com/joshuarubin/zb/issues/5)]
* Wrap [`govendor`](https://github.com/kardianos/govendor) in an opinionated way [[#6](https://github.com/joshuarubin/zb/issues/5)]
* Setup continuous integration [[#7](https://github.com/joshuarubin/zb/issues/7)]

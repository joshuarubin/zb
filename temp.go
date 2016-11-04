package main

import (
	"os"
	"path/filepath"
	"time"

	git "gopkg.in/src-d/go-git.v4"
	"jrubin.io/zb/lib/project"
)

func init() {
	var err error
	gitCommit, err = getGitCommit()
	if err != nil {
		panic(err)
	}

	buildDate = getBuildDate()
}

func getGitCommit() (string, error) {
	// TODO(jrubin) delete this when set by zb build

	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	dir := project.Dir(cwd)
	if dir == "" {
		return "", err
	}

	dir = filepath.Join(dir, ".git")

	repo, err := git.NewFilesystemRepository(dir)
	if err != nil {
		return "", err
	}

	head, err := repo.Head()
	if err != nil {
		return "", err
	}

	return head.Hash().String(), nil
}

const dateFormat = "2006-01-02T15:04:05+00:00"

func getBuildDate() string {
	// TODO(jrubin) delete this when set by zb build
	return time.Now().UTC().Format(dateFormat)
}

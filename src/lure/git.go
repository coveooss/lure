package main

import (
	"regexp"
	"strings"
	"fmt"
	"log"
)

type GitRepo struct {
	localPath  string
	remotePath string

	userPass   UserPassAuth
}

func GitSanitizeBranchName(name string) string {
	reg, _ := regexp.Compile("[^a-zA-Z0-9_-]+")
	safe := reg.ReplaceAllString(name, "_")
	return safe
}

func GitClone(auth Authentication, source string, to string) (GitRepo, error) {
	var repo GitRepo

	switch auth := auth.(type) {
	case TokenAuth:
		source = strings.Replace(source, "://", fmt.Sprintf("://x-token-auth:%s@", auth.token) , 1)
	case UserPassAuth:
		source = strings.Replace(source, "://", fmt.Sprintf("://%s:%s@", auth.username, auth.password) , 1)
	}

	args := []string{ "clone", source, to }

	if _, err := execute("", "git", args...); err != nil {
		return repo, err
	}

	repo = GitRepo{
		localPath: to,
		remotePath: source,
	}

	return repo, nil
}

func (gitRepo GitRepo) LocalPath() string {
	return gitRepo.localPath
}

func (gitRepo GitRepo) RemotePath() string {
	return gitRepo.remotePath
}

func (gitRepo GitRepo) Cmd(args ...string) (string, error) {
	return execute(gitRepo.localPath, "git", args...)
}

func (gitRepo GitRepo) Update(rev string) (string, error) {
	return gitRepo.Cmd("checkout", rev)
}

func (gitRepo GitRepo) Branch(branchname string) (string, error) {
	return gitRepo.Cmd("checkout", "-b", GitSanitizeBranchName(branchname))
}

func (gitRepo GitRepo) Commit(message string) (string, error) {
	return gitRepo.Cmd("commit", "-m", message)
}

func (gitRepo GitRepo) Push() (string, error) {
	return gitRepo.Cmd("push", gitRepo.remotePath)
}

func (gitRepo GitRepo) LogCommitsBetween(baseRev string, secondRev string) ([]string, error) {
	out, err := gitRepo.Cmd("log", "--pretty=%h", fmt.Sprintf("%s...%s", baseRev, secondRev))
	if err != nil {
		return []string{}, err
	}

	lines := strings.Split(out, "\n")
	return append(lines[:0], lines[:len(lines)-1]...), nil
}
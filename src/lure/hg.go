package main

import (
	"regexp"
	"strings"
	"errors"
	"fmt"
	"os"
)

type Authentication interface {}

type TokenAuth struct {
	token string
}

type UserPassAuth struct {
	username string
	password string
}

type HgRepo struct {
	localPath  string
	remotePath string

	userPass   UserPassAuth
}

func HgSanitizeBranchName(name string) string {
	reg, _ := regexp.Compile("[^a-zA-Z0-9_-]+")
	safe := reg.ReplaceAllString(name, "_")
	return safe
}

func HgClone(auth Authentication, source string, to string) (HgRepo, error) {
	var repo HgRepo

	args := []string{ "clone", source, to }

	switch auth := auth.(type) {
	case TokenAuth:
		source = strings.Replace(source, "://", fmt.Sprintf("://x-token-auth:%s@", auth.token) , 1)
	case UserPassAuth:
		args = append([]string{
			"--config", "auth.repo.prefix=*",
			"--config", "auth.repo.username=" + auth.username,
			"--config", "auth.repo.password=" + auth.password,
		}, args...)
	}

	if _, err := execute("", "hg", args...); err != nil {
		return repo, err
	}

	repo = HgRepo{
		localPath: to,
		remotePath: source,
	}

	switch auth := auth.(type) {
	case UserPassAuth:
		repo.SetUserPas(auth.username, auth.password)
	}

	return repo, nil
}

func (hgRepo *HgRepo) Hg(args ...string) (string, error) {
	return execute(hgRepo.localPath, "hg", args...)
}

func (hgRepo *HgRepo) SetUserPas(user string, pass string) (error) {
	f, err := os.OpenFile(fmt.Sprintf("%s/.hg/hgrc", hgRepo.localPath), os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		return err
	}

	f.WriteString("[auth]\n")
	f.WriteString("repo.prefix=*\n")
	f.WriteString(fmt.Sprintf("repo.username=%s\n", user))
	f.WriteString(fmt.Sprintf("repo.password=%s\n", pass))
	// keep credentials private
	return f.Close()
}

func (hgRepo *HgRepo) Update(rev string) (string, error) {
	return hgRepo.Hg("update", rev)
}

func (hgRepo *HgRepo) Branch(branchname string) (string, error) {
	return hgRepo.Hg("branch", HgSanitizeBranchName(branchname))
}

func (hgRepo *HgRepo) Commit(message string) (string, error) {
	return hgRepo.Hg("commit", "-m", message)
}

func (hgRepo *HgRepo) Merge(branch string) (string, error) {
	_, err := hgRepo.Hg("merge", branch)
	if err != nil {
		return "", errors.New(fmt.Sprintf("Error: \"Could not merge %s into current branch\" %s", branch, err.Error()))
	}
	return "", nil
}

func (hgRepo *HgRepo) Push() (string, error) {
	return hgRepo.Hg("push", "--new-branch", hgRepo.remotePath)
}

func (hgRepo *HgRepo) PushDefault() (string, error) {
	return hgRepo.Hg("push", "--new-branch")
}

func (hgRepo *HgRepo) LogCommitsBetween(baseRev string, secondRev string) ([]string, error) {
	out, err := hgRepo.Hg("log", "-r", "ancestors(" + secondRev + ") and not ancestors(" + baseRev + ")", "--template", "{node}\n")
	if err != nil {
		return []string{}, err
	}

	lines := strings.Split(out, "\n")
	return append(lines[:0], lines[:len(lines)-1]...), nil
}
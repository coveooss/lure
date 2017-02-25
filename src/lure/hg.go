package main

import (
	"regexp"
	"strings"
	"errors"
	"fmt"
)

type HgRepo struct {
	localPath string
	remotePath string

}

func HgSanitizeBranchName(name string) string {
	reg, _ := regexp.Compile("[^a-zA-Z0-9_-]+")
	safe := reg.ReplaceAllString(name, "_")
	return safe
}

func HgClone(source string, to string) (HgRepo, error) {
	var repo HgRepo

	if _, err := execute("", "hg", "clone", source, to); err != nil {
		return repo, err
	}

	repo = HgRepo{
		localPath: to,
		remotePath: source,
	}
	return repo, nil
}

func (hgRepo *HgRepo) Hg(args ...string) (string, error) {
	return execute(hgRepo.localPath, "hg", args...)
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
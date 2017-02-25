package main

import (
	"regexp"
	"strings"
	"errors"
	"fmt"
)

func hgSanitizeBranchName(name string) string {
	reg, _ := regexp.Compile("[^a-zA-Z0-9_-]+")
	safe := reg.ReplaceAllString(name, "_")
	return safe
}

func hgClone(source, to string) (string, error) {
	return execute("", "hg", "clone", source, to)
}

func hgUpdate(repository, rev string) (string, error) {
	return execute(repository, "hg", "update", rev)
}

func hgBranch(repository, branchname string) (string, error) {
	return execute(repository, "hg", "branch", hgSanitizeBranchName(branchname))
}

func hgCommit(repository, message string) (string, error) {
	return execute(repository, "hg", "commit", "-m", message)
}

func hgMerge(repository, branch string) (string, error) {
	_, err := execute(repository, "hg", "merge", branch)
	if err != nil {
		return "", errors.New(fmt.Sprintf("Error: \"Could not merge %s into current branch\" %s", branch, err.Error()))
	}
	return "", nil
}

func hgPush(repository, remote string) (string, error) {
	return execute(repository, "hg", "push", "--new-branch", remote)
}

func hgPushDefault(repository string) (string, error) {
	return execute(repository, "hg", "push", "--new-branch")
}

func hgLogCommitsBetween(repository, baseRev string, secondRev string) ([]string, error) {
	out, err := execute(repository, "hg", "log", "-r", "ancestors(" + secondRev + ") and not ancestors(" + baseRev + ")", "--template", "{node}\n")
	if err != nil {
		return []string{}, err
	}

	lines := strings.Split(out, "\n")
	return append(lines[:0], lines[:len(lines)-1]...), nil
}
package main

import (
	"regexp"
)

func hgSanitizeBranchName(name string) string {
	reg, _ := regexp.Compile("[^a-zA-Z0-9_-]*")
	safe := reg.ReplaceAllString(name, "_")
	return safe
}

func hgClone(source, to string) error {
	return execute("", "hg", "clone", source, to)
}

func hgUpdate(repository, rev string) error {
	return execute(repository, "hg", "update", rev)
}

func hgBranch(repository, branchname string) error {
	return execute(repository, "hg", "branch", hgSanitizeBranchName(branchname))
}

func hgCommit(repository, message string) error {
	return execute(repository, "hg", "commit", "-m", message)
}

func hgPush(repository, remote string) error {
	return execute(repository, "hg", "push", "--new-branch", remote)
}

func hgPushDefault(repository string) error {
	return execute(repository, "hg", "push", "--new-branch")
}

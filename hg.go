package main

func hgClone(source, to string) error {
	return execute("hg", "clone", source, to)
}

func hgUpdate(repository, rev string) error {
	return execute(repository, "hg", "update", rev)
}

func hgBranch(repository, branchname string) error {
	return execute(repository, "hg", "branch", branchname)
}

func hgCommit(repository, message string) error {
	return execute(repository, "hg", "commit", message)
}

func hgPush(repository, remote string) error {
	return execute(repository, "hg", "push", remote)
}

package lure

import (
	"fmt"
	"regexp"
	"strings"
)

type GitRepo struct {
	workingPath string
	localPath   string
	remotePath  string

	userPass UserPassAuth
}

func GitSanitizeBranchName(name string) string {
	reg, _ := regexp.Compile("[^a-zA-Z0-9_-]+")
	safe := reg.ReplaceAllString(name, "_")
	return safe
}

func GitClone(auth Authentication, source string, to string, basePath string) (GitRepo, error) {
	var repo GitRepo

	switch auth := auth.(type) {
	case TokenAuth:
		source = strings.Replace(source, "://", fmt.Sprintf("://x-token-auth:%s@", auth.Token), 1)
	case UserPassAuth:
		source = strings.Replace(source, "://", fmt.Sprintf("://%s:%s@", auth.Username, auth.Password), 1)
	}

	args := []string{"clone", source, to}

	if _, err := Execute("", "git", args...); err != nil {
		return repo, err
	}

	var workingPath strings.Builder
	workingPath.WriteString(to)
	workingPath.WriteString("/")
	workingPath.WriteString(basePath)

	repo = GitRepo{
		workingPath: workingPath.String(),
		localPath:   to,
		remotePath:  source,
	}

	return repo, nil
}

func (gitRepo GitRepo) WorkingPath() string {
	return gitRepo.workingPath
}

func (gitRepo GitRepo) LocalPath() string {
	return gitRepo.localPath
}

func (gitRepo GitRepo) RemotePath() string {
	return gitRepo.remotePath
}

func (gitRepo GitRepo) Cmd(args ...string) (string, error) {
	return Execute(gitRepo.localPath, "git", args...)
}

func (gitRepo GitRepo) Update(rev string) (string, error) {
	return gitRepo.Cmd("checkout", rev)
}

func (gitRepo GitRepo) Branch(branchname string) (string, error) {
	return gitRepo.Cmd("checkout", "-b", GitSanitizeBranchName(branchname))
}

func (gitRepo GitRepo) Commit(message string) (string, error) {
	add, err := gitRepo.Cmd("add", "--all")
	if err != nil {
		return add, err
	}
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

// GetActiveBranches returns all currently active branches without origin/ prefix
func (gitRepo GitRepo) GetActiveBranches() ([]string, error) {
	out, err := gitRepo.Cmd("branch", "-r")
	if err != nil {
		return nil, err
	}
	branches := strings.Split(strings.TrimSpace(out), "\n")

	// removing the remote prefix (origin/ most of the time)
	for i := range branches {
		if strings.Contains(branches[i], "origin/HEAD ->") {
			branches[i] = strings.SplitN(branches[i], "/", 3)[2]
		} else {
			branches[i] = strings.SplitN(branches[i], "/", 2)[1]
		}
	}
	return branches, nil
}

// CloseBranch deletes the branch for the remote repository
func (gitRepo GitRepo) CloseBranch(branch string) error {
	Logger.Infof("Closing branch %s.", branch)
	_, err := gitRepo.Cmd("push", gitRepo.remotePath, "--delete", branch)
	return err
}

package lure

import (
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
)

type HgRepo struct {
	localPath   string
	remotePath  string
	trashBranch string
}

func HgSanitizeBranchName(name string) string {
	reg, _ := regexp.Compile("[^a-zA-Z0-9/_-]+")
	safe := reg.ReplaceAllString(name, "_")
	return safe
}

func HgClone(auth Authentication, source string, to string, trashBranch string) (HgRepo, error) {
	var repo HgRepo

	args := []string{"clone", source, to}

	if os.Getenv("LURE_LOCAL_PROJECT_PATH") == "" {
		switch auth := auth.(type) {
		case TokenAuth:
			source = strings.Replace(source, "://", fmt.Sprintf("://x-token-auth:%s@", auth.Token), 1)
			args = []string{"clone", source, to}
		case UserPassAuth:
			args = append([]string{
				"--config", "auth.repo.prefix=*",
				"--config", "auth.repo.username=" + auth.Username,
				"--config", "auth.repo.password=" + auth.Password,
			}, args...)
		}
	}

	if _, err := Execute("", "hg", args...); err != nil {
		return repo, err
	}

	repo = HgRepo{
		localPath:   to,
		remotePath:  source,
		trashBranch: trashBranch,
	}

	switch auth := auth.(type) {
	case UserPassAuth:
		repo.SetUserPas(auth.Username, auth.Password)
	}

	return repo, nil
}

func (hgRepo HgRepo) LocalPath() string {
	return hgRepo.localPath
}

func (hgRepo HgRepo) RemotePath() string {
	return hgRepo.remotePath
}

func (hgRepo HgRepo) Cmd(args ...string) (string, error) {
	return Execute(hgRepo.localPath, "hg", args...)
}

func (hgRepo HgRepo) SetUserPas(user string, pass string) error {
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

func (hgRepo HgRepo) Update(rev string) (string, error) {
	return hgRepo.Cmd("update", rev)
}

func (hgRepo HgRepo) Branch(branchname string) (string, error) {
	return hgRepo.Cmd("branch", HgSanitizeBranchName(branchname))
}

func (hgRepo HgRepo) Commit(message string) (string, error) {
	return hgRepo.Cmd("commit", "-m", message)
}

func (hgRepo HgRepo) Merge(branch string) (string, error) {
	_, err := hgRepo.Cmd("merge", branch)
	if err != nil {
		return "", errors.New(fmt.Sprintf("Error: \"Could not merge %s into current branch\" %s", branch, err.Error()))
	}
	return "", nil
}

func (hgRepo HgRepo) Push() (string, error) {
	return hgRepo.Cmd("push", "--new-branch", hgRepo.remotePath)
}

func (hgRepo HgRepo) PushDefault() (string, error) {
	return hgRepo.Cmd("push", "--new-branch")
}

func (hgRepo HgRepo) LogCommitsBetween(baseRev string, secondRev string) ([]string, error) {
	out, err := hgRepo.Cmd("log", "-r", "ancestors("+secondRev+") and not ancestors("+baseRev+")", "--template", "{node}\n")
	if err != nil {
		return []string{}, err
	}

	lines := strings.Split(out, "\n")
	return append(lines[:0], lines[:len(lines)-1]...), nil
}

func (hgRepo HgRepo) CloseBranch(branch string) error {
	if hgRepo.trashBranch == "" {
		log.Printf("Info: Repo has no trash branch defined. Cannot close branch %s.", branch)
		return nil
	}

	log.Printf("Closing branch %s.", branch)

	if _, err := hgRepo.Cmd("update", "-C", branch); err != nil {
		log.Printf("Error: \"Could not switch to branch %s\" %s", branch, err)
		return err
	}

	if _, err := hgRepo.Cmd("commit", "-m", "Close branch "+branch, "--close-branch"); err != nil {
		log.Printf("Error: \"Could not commit\" %s", err)
		return err
	}

	if _, err := hgRepo.Update(hgRepo.trashBranch); err != nil {
		log.Printf("Error: \"Could not switch to branch %s\" %s", hgRepo.trashBranch, err)
		return err
	}

	if err := hgRepo.fakeMerge(branch, hgRepo.trashBranch); err != nil {
		log.Printf("Error: \"Could not fake merge branch %s to branch %s\" %s", branch, hgRepo.trashBranch, err)
		return err
	}

	return nil
}

func (hgRepo HgRepo) fakeMerge(branch string, toBranch string) error {

	hgRepo.Cmd("-y", "merge", "--tool=internal:fail", branch) // Always produces an err
	if _, err := hgRepo.Cmd("revert", "--all", "--rev", "."); err != nil {
		return err
	}
	if _, err := hgRepo.Cmd("resolve", "-a", "-m"); err != nil {
		return err
	}
	if _, err := hgRepo.Commit(fmt.Sprintf("Fake merge to close %s into %s", branch, toBranch)); err != nil {
		return err
	}

	return nil
}

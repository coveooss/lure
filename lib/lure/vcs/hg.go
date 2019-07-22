package vcs

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/coveooss/lure/lib/lure/log"
	osUtils "github.com/coveooss/lure/lib/lure/os"
)

type HgRepo struct {
	workingPath   string
	localPath     string
	remotePath    string
	trashBranch   string
	defaultBranch string
	authArgs      []string
}

func NewHg(auth Authentication, source string, to string, defaultBranch string, trashBranch string, basePath string) (HgRepo, error) {
	var workingPath strings.Builder
	workingPath.WriteString(to)
	workingPath.WriteString("/")
	workingPath.WriteString(basePath)

	repo := HgRepo{
		workingPath:   workingPath.String(),
		localPath:     to,
		remotePath:    auth.AuthenticateURL(source),
		defaultBranch: defaultBranch,
		trashBranch:   trashBranch,
	}

	//TODO : validate that it works
	// switch auth := auth.(type) {
	// case TokenAuth:
	// 	source = strings.Replace(source, "://", fmt.Sprintf("://x-token-auth:%s@", auth.Token), 1)
	// case UserPassAuth:
	// 	repo.SetUserPas(auth.Username, auth.Password)
	// 	repo.authArgs = append([]string{
	// 		"--config", "auth.repo.prefix=*",
	// 		"--config", "auth.repo.username=" + auth.Username,
	// 		"--config", "auth.repo.password=" + auth.Password,
	// 	}, repo.authArgs...)
	// }

	return repo, nil
}

func (hgRepo HgRepo) SanitizeBranchName(branchName string) string {
	reg, _ := regexp.Compile("[^a-zA-Z0-9/_-]+")
	safe := reg.ReplaceAllString(branchName, "_")
	return safe
}

func (hgRepo HgRepo) Clone() error {
	log.Logger.Infof("cloning to %s", hgRepo.localPath)
	args := []string{"clone", hgRepo.remotePath, hgRepo.localPath}

	if _, err := osUtils.Execute("", "hg", args...); err != nil {
		return err
	}
	return nil
}

func (hgRepo HgRepo) WorkingPath() string {
	return hgRepo.workingPath
}

func (hgRepo HgRepo) GetName() string {
	return Hg
}

func (hgRepo HgRepo) LocalPath() string {
	return hgRepo.localPath
}

func (hgRepo HgRepo) RemotePath() string {
	return hgRepo.remotePath
}

func (hgRepo HgRepo) Cmd(args ...string) (string, error) {
	return osUtils.Execute(hgRepo.localPath, "hg", args...)
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
	//return hgRepo.Cmd("branch", hgRepo.SanitizeBranchName(branchname))

	branch, err := hgRepo.Cmd("branch", hgRepo.SanitizeBranchName(branchname))
	if err != nil {
		return "", err
	}

	_, err = hgRepo.Commit(fmt.Sprintf("creating branch %s", branch))
	if err != nil {
		return "", err
	}

	return branch, nil
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

// GetActiveBranches returns all currently active branches
func (hgRepo HgRepo) GetActiveBranches() ([]string, error) {
	out, err := hgRepo.Cmd("branches", "--active", "--template", "{branch}\n")
	if err != nil {
		return nil, err
	}

	return strings.Split(out, "\n"), nil
}

// CloseBranch closes the branch then it merges it to a trash branch so no heads are left
func (hgRepo HgRepo) CloseBranch(branch string) error {
	log.Logger.Infof("Closing branch %s.", branch)

	if _, err := hgRepo.Cmd("update", "-C", branch); err != nil {
		log.Logger.Errorf("Error: \"Could not switch to branch %s\" %s", branch, err)
		return err
	}

	if _, err := hgRepo.Cmd("commit", "-m", "Close branch "+branch, "--close-branch"); err != nil {
		log.Logger.Errorf("Error: \"Could not commit\" %s", err)
		return err
	}

	if _, err := hgRepo.Update(hgRepo.trashBranch); err != nil {
		log.Logger.Errorf("Error: \"Could not switch to branch %s, trying to create it.\" %s", hgRepo.trashBranch, err)
		if _, err := hgRepo.Update(hgRepo.defaultBranch); err != nil {
			log.Logger.Errorf("Error: \"Could not switch to branch %s\" %s", hgRepo.defaultBranch, err)
			return err
		}
		if _, err := hgRepo.Branch(hgRepo.trashBranch); err != nil {
			log.Logger.Errorf("Error: \"Could not create branch %s\" %s", hgRepo.trashBranch, err)
			return err
		}
	}

	if err := hgRepo.fakeMerge(branch, hgRepo.trashBranch); err != nil {
		log.Logger.Errorf("Error: \"Could not fake merge branch %s to branch %s\" %s", branch, hgRepo.trashBranch, err)
		return err
	}

	if _, err := hgRepo.Push(); err != nil {
		log.Logger.Errorf("Error: \"Could not push closed branch %s\" %s", branch, err)
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

package lure

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"strings"

	"github.com/vsekhar/govtil/guid"
)

// This part interesting
// https://github.com/golang/go/blob/1441f76938bf61a2c8c2ed1a65082ddde0319633/src/cmd/go/vcs.go

func appendIfMissing(modules []moduleVersion, modulesToAdd []moduleVersion) []moduleVersion {
	for _, moduleToAdd := range modulesToAdd {
		exist := false
		for _, module := range modules {
			if (moduleToAdd.Name != "" && module.Name == moduleToAdd.Name) || reflect.DeepEqual(module, moduleToAdd) {
				exist = true
				break
			}
		}
		if !exist {
			modules = append(modules, moduleToAdd)
		}
	}

	return modules
}

func cloneRepo(hgAuth Authentication, project Project) (Repo, error) {
	repoGUID, err := guid.V4()

	var repo Repo
	if err != nil {
		Logger.Errorf("\"Could not generate guid\" %s", err)
		return repo, err
	}
	repoPath := "/tmp/" + repoGUID.String()

	projectRemote := "https://bitbucket.org/" + project.Owner + "/" + project.Name

	Logger.Infof("cloning: %s to %s", projectRemote, repoPath)

	switch project.Vcs {
	case Hg:
		repo, err = HgClone(hgAuth, projectRemote, repoPath, project.DefaultBranch, project.TrashBranch, project.BasePath)
	case Git:
		repo, err = GitClone(hgAuth, projectRemote, repoPath, project.BasePath)
	default:
		repo = nil
		err = fmt.Errorf("Unknown VCS '%s' - must be one of %s, %s", project.Vcs, Git, Hg)
	}
	if err != nil {
		Logger.Errorf("\"Could not clone\" %s", err)
		return repo, err
	}

	return repo, nil
}

func CheckForUpdatesJobCommand(auth Authentication, project Project, args map[string]string) error {
	return checkForUpdatesJob(auth, project, args["commitMessage"])
}

func checkForUpdatesJob(auth Authentication, project Project, commitMessage string) error {
	repo, err := cloneRepo(auth, project)
	if err != nil {
		return err
	}

	Logger.Infof("switching %s to default branch: %s", repo.RemotePath(), project.DefaultBranch)
	if _, err := repo.Update(project.DefaultBranch); err != nil {
		return fmt.Errorf("Error: \"Could not switch to branch %s\" %s", project.DefaultBranch, err)
	}

	modulesToUpdate := make([]moduleVersion, 0, 0)

	if project.SkipPackageManager == nil || project.SkipPackageManager["npm"] != true {
		modulesToUpdate = appendIfMissing(modulesToUpdate, npmOutdated(repo.WorkingPath()))
	}

	if project.SkipPackageManager == nil || project.SkipPackageManager["mvn"] != true {
		err, modulesToAdd := mvnOutdated(repo.WorkingPath())
		if err != nil {
			return err
		}
		modulesToUpdate = appendIfMissing(modulesToUpdate, modulesToAdd)
	}

	Logger.Infof("Modules to update : %q", modulesToUpdate)

	ignoreDeclinedPRs := os.Getenv("IGNORE_DECLINED_PR") == "1"
	pullRequests, err := getPullRequests(auth, project.Owner, project.Name, ignoreDeclinedPRs)
	if err != nil {
		return err
	}

	for _, moduleToUpdate := range modulesToUpdate {
		updateModule(auth, moduleToUpdate, project, repo, pullRequests, commitMessage)
	}

	err = closeOldBranchesWithoutOpenPR(auth, project, repo)
	if err != nil {
		return err
	}

	Logger.Infof("`[Check for updates done.")

	return nil
}

func updateModule(auth Authentication, moduleToUpdate moduleVersion, project Project, repo Repo, existingPRs []PullRequest, commitMessage string) {
	var dependencyName string
	if moduleToUpdate.Name != "" {
		dependencyName = moduleToUpdate.Name
	} else {
		dependencyName = moduleToUpdate.Module
	}

	title := fmt.Sprintf("Update %s dependency %s to version %s", moduleToUpdate.Type, dependencyName, moduleToUpdate.Latest)

	branchPrefix := project.BranchPrefix
	if branchPrefix == "" {
		branchPrefix = "lure-"
	}
	dependencyBranchPrefix := HgSanitizeBranchName(branchPrefix + dependencyName)
	dependencyBranchVersionPrefix := dependencyBranchPrefix + "-" + HgSanitizeBranchName(moduleToUpdate.Latest)
	branchGUID, _ := guid.V4()
	var branch = dependencyBranchVersionPrefix + "-" + branchGUID.String()

	var openPRAlreadyExists = false
	var declinedPRAlreadyExists = false
	for _, pr := range existingPRs {
		if !openPRAlreadyExists && strings.HasPrefix(pr.Source.Branch.Name, dependencyBranchVersionPrefix) {
			if pr.State == "OPEN" {
				Logger.Infof("There already is an open PR for: '%s'. The branch name is: %s.", title, pr.Source.Branch.Name)
				openPRAlreadyExists = true
			} else {
				Logger.Infof("There was a declined PR for: '%s'. The branch name is: %s.", title, pr.Source.Branch.Name)
				declinedPRAlreadyExists = true
			}
			continue
		}

		if pr.State == "OPEN" && strings.HasPrefix(pr.Source.Branch.Name, dependencyBranchPrefix) {
			if os.Getenv("DRY_RUN") == "1" {
				Logger.Infof("Running in DryRun mode. PR '%s' made for older version would be declined.", pr.Title)
			} else {
				Logger.Infof("Declining PR '%s' made for older version.", pr.Title)
				declinePullRequest(auth, project.Owner, project.Name, pr.ID)
			}
		}
	}
	if openPRAlreadyExists || declinedPRAlreadyExists {
		return
	}

	Logger.Infof("switching %s to default branch: %s", repo.LocalPath(), project.DefaultBranch)
	if _, err := repo.Update(project.DefaultBranch); err != nil {
		Logger.Fatalf("\"Could not switch to branch %s\" %s", project.DefaultBranch, err)
	}

	Logger.Infof("Creating branch %s\n", branch)
	if _, err := repo.Branch(branch); err != nil {
		Logger.Errorf("\"Could not create branch\" %s", err)
		return
	}

	hasChanges := false

	switch moduleToUpdate.Type {
	case "maven":
		hasChanges, _ = mvnUpdateDep(repo.WorkingPath(), moduleToUpdate)
	case "npm":
		hasChanges, _ = readPackageJSON(repo.WorkingPath(), moduleToUpdate.Module, moduleToUpdate.Latest)
	}

	if hasChanges == false {
		return
	}

	if _, err := repo.Commit(Tprintf(commitMessage, map[string]interface{}{"module": moduleToUpdate.Module, "version": moduleToUpdate.Latest})); err != nil {
		Logger.Errorf("\"Could not commit\" %s", err)
		return
	}

	if os.Getenv("DRY_RUN") == "1" {
		Logger.Info("Running in DryRun mode, not doing the pull request nor pushing the changes")
	} else {
		Logger.Info("Pushing changes")
		if _, err := repo.Push(); err != nil {
			Logger.Fatalf("\"Could not push\" %s", err)
			return
		}

		Logger.Infof("Creating PR")

		description := Tprintf(commitMessage, map[string]interface{}{"module": moduleToUpdate.Module, "version": moduleToUpdate.Latest})
		createPullRequest(auth, branch, project.DefaultBranch, project.Owner, project.Name, title, description, *project.UseDefaultReviewers)
	}
}

func Execute(pwd string, command string, params ...string) (string, error) {
	Logger.Tracef("%s %q\n", command, params)

	cmd := exec.Command(command, params...)
	cmd.Dir = pwd

	var buff bytes.Buffer
	var stderr bytes.Buffer

	cmd.Stdout = &buff
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		Logger.Error(stderr.String())
		return "", err
	}

	out := buff.String()

	Logger.Tracef("\t%s\n", out)

	return out, nil
}

func closeOldBranchesWithoutOpenPR(auth Authentication, project Project, repo Repo) error {
	Logger.Info("Cleaning up lure branches with no associated PRs.")

	branchPrefix := project.BranchPrefix
	branches, err := repo.GetActiveBranches()
	if err != nil {
		return err
	}
	existingPRs, err := getPullRequests(auth, project.Owner, project.Name, false)
	if err != nil {
		return err
	}

	for _, branch := range branches {
		if strings.HasPrefix(branch, branchPrefix) {
			if isBranchDead(repo, branch, existingPRs) {
				if os.Getenv("DRY_RUN") == "1" {
					Logger.Infof("Running in DryRun mode. Branch '%s' would of been closed.", branch)
				} else {
					if err := repo.CloseBranch(branch); err != nil {
						println(err)
						return err
					}
				}
			}
		}
	}
	Logger.Info("Lure branches clean up done.")

	return nil
}

func isBranchDead(repo Repo, branch string, existingPRs []PullRequest) bool {
	for _, pr := range existingPRs {
		if pr.State == "OPEN" && branch == pr.Source.Branch.Name {
			return false
		}
	}
	return true
}

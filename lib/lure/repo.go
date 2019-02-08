package lure

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"reflect"
	"regexp"
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
		log.Printf("Error: \"Could not generate guid\" %s", err)
		return repo, err
	}
	repoPath := "/tmp/" + repoGUID.String()

	projectRemote := "https://bitbucket.org/" + project.Owner + "/" + project.Name

	log.Printf("Info: cloning: %s to %s", projectRemote, repoPath)

	switch project.Vcs {
	case Hg:
		repo, err = HgClone(hgAuth, projectRemote, repoPath)
	case Git:
		repo, err = GitClone(hgAuth, projectRemote, repoPath)
	default:
		repo = nil
		err = fmt.Errorf("Unknown VCS '%s' - must be one of %s, %s", project.Vcs, Git, Hg)
	}
	if err != nil {
		log.Printf("Error: \"Could not clone\" %s", err)
		return repo, err
	}

	return repo, nil
}

func CheckForUpdatesJobCommand(auth Authentication, project Project, args map[string]string) error {
	return checkForUpdatesJob(auth, project)
}

func checkForUpdatesJob(auth Authentication, project Project) error {

	repo, err := cloneRepo(auth, project)
	if err != nil {
		return err
	}

	log.Printf("Info: switching %s to default branch: %s", repo.RemotePath(), project.DefaultBranch)
	if _, err := repo.Update(project.DefaultBranch); err != nil {
		return fmt.Errorf("Error: \"Could not switch to branch %s\" %s", project.DefaultBranch, err)
	}

	modulesToUpdate := make([]moduleVersion, 0, 0)
	modulesToUpdate = appendIfMissing(modulesToUpdate, npmOutdated(repo.LocalPath()+"/"+project.BasePath))

	err, modulesToAdd := mvnOutdated(repo.LocalPath() + "/" + project.BasePath)
	modulesToUpdate = appendIfMissing(modulesToUpdate, modulesToAdd)
	log.Printf("Modules to update : %q", modulesToUpdate)

	ignoreDeclinedPRs := os.Getenv("IGNORE_DECLINED_PR") == "1"
	pullRequests := getPullRequests(auth, project.Owner, project.Name, ignoreDeclinedPRs)

	for _, moduleToUpdate := range modulesToUpdate {
		updateModule(auth, moduleToUpdate, project, repo, pullRequests)
	}

	log.Printf("Info: Check for updates done.")

	cleanupUpdateBranches(auth, project, repo)

	return nil
}

func updateModule(auth Authentication, moduleToUpdate moduleVersion, project Project, repo Repo, existingPRs []PullRequest) {
	var title string
	var dependencyName string
	if moduleToUpdate.Name != "" {
		dependencyName = moduleToUpdate.Name
	} else {
		dependencyName = moduleToUpdate.Module
	}
	title = fmt.Sprintf("Update %s dependency %s to version %s", moduleToUpdate.Type, dependencyName, moduleToUpdate.Latest)
	for _, pr := range existingPRs {
		if pr.Title == title {
			log.Printf("There already is a PR for: %s", title)
			return
		}
	}

	log.Printf("Info: switching %s to default branch: %s", repo.LocalPath(), project.DefaultBranch)
	if _, err := repo.Update(project.DefaultBranch); err != nil {
		log.Fatalf("Error: \"Could not switch to branch %s\" %s", project.DefaultBranch, err)
	}

	branchGUID, _ := guid.V4()
	branchPrefix := project.BranchPrefix
	if branchPrefix == "" {
		branchPrefix = "lure-"
	}
	var branch = HgSanitizeBranchName(branchPrefix + dependencyName + "-" + moduleToUpdate.Latest + "-" + branchGUID.String())
	log.Printf("Creating branch %s\n", branch)
	if _, err := repo.Branch(branch); err != nil {
		log.Printf("Error: \"Could not create branch\" %s", err)
		return
	}

	hasChanges := false

	switch moduleToUpdate.Type {
	case "maven":
		hasChanges, _ = mvnUpdateDep(repo.LocalPath(), moduleToUpdate)
	case "npm":
		hasChanges, _ = readPackageJSON(repo.LocalPath(), moduleToUpdate.Module, moduleToUpdate.Latest)
	}

	if hasChanges == false {
		return
	}

	if _, err := repo.Commit("Update " + dependencyName + " to " + moduleToUpdate.Latest); err != nil {
		log.Printf("Error: \"Could not commit\" %s", err)
		return
	}

	if os.Getenv("DRY_RUN") == "1" {
		log.Println("Running in DryRun mode, not doing the pull request nor pushing the changes")
	} else {
		log.Printf("Pushing changes\n")
		if _, err := repo.Push(); err != nil {
			log.Fatalf("Error: \"Could not push\" %s", err)
			return
		}

		log.Printf("Creating PR\n")
		description := fmt.Sprintf("%s version %s is now available! Please update.", moduleToUpdate.Module, moduleToUpdate.Latest)
		createPullRequest(auth, branch, project.DefaultBranch, project.Owner, project.Name, title, description)
	}
}

func Execute(pwd string, command string, params ...string) (string, error) {
	log.Printf("%s %q\n", command, params)

	cmd := exec.Command(command, params...)
	cmd.Dir = pwd

	var buff bytes.Buffer
	var stderr bytes.Buffer

	cmd.Stdout = &buff
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		log.Println(stderr.String())
		return "", err
	}

	out := buff.String()

	log.Printf("\t%s\n", out)

	return out, nil
}

func cleanupUpdateBranches(auth Authentication, project Project, repo Repo) error {

	if project.Vcs != Hg {
		return nil
	}

	trashBranch := project.TrashBranch
	if trashBranch == "" {
		log.Printf("Info: Project has no trash branch defined. Skipping cleanup.")
		return nil
	}

	log.Printf("Info: Cleaning up lure branches with no associated PRs.")

	out, err := repo.Cmd("branches", "--active", "--template", "{branches}\n")
	if err != nil {
		return err
	}

	branches := strings.Split(out, "\n")
	branchPrefix := project.BranchPrefix
	if branchPrefix == "" {
		branchPrefix = "lure-"
	}
	msgRegex, err := regexp.Compile(`Update (\S*) to (\S*)`)
	if err != nil {
		return err
	}

	existingPRs := getPullRequests(auth, project.Owner, project.Name, true)

	for _, branch := range branches {
		if strings.HasPrefix(branch, branchPrefix) {

			dead, err := isBranchDead(repo, branch, existingPRs, msgRegex)
			if err != nil {
				continue
			}

			if dead {
				closeDeadBranch(repo, branch, trashBranch)
			}
		}
	}

	if os.Getenv("DRY_RUN") == "1" {
		log.Println("Running in DryRun mode, not doing the pull request nor pushing the changes")
	} else {
		if _, err := repo.Push(); err != nil {
			return err
		}
	}

	log.Printf("Info: Lure branches clean up done.")

	return nil
}

func isBranchDead(repo Repo, branch string, existingPRs []PullRequest, msgRegex *regexp.Regexp) (bool, error) {

	// Get dependency name and version from latest commit message
	message, err := repo.Cmd("log", "--limit", "1", "--branch", branch, "--template", "{desc}")
	if err != nil {
		return false, err
	}

	matches := msgRegex.FindStringSubmatch(message)
	if len(matches) != 3 {
		return false, err
	}
	dependencyName := matches[1]
	dependencyVersion := matches[2]

	// Look for a matching opened PR
	titleSuffix := fmt.Sprintf("dependency %s to version %s", dependencyName, dependencyVersion)
	foundPR := false
	for _, pr := range existingPRs {
		if pr.State == "OPEN" && strings.Contains(pr.Title, titleSuffix) {
			foundPR = true
			break
		}
	}

	return !foundPR, nil
}

func closeDeadBranch(repo Repo, branch string, trashBranch string) error {

	log.Printf("Closing branch %s.", branch)

	if _, err := repo.Cmd("update", "-C", branch); err != nil {
		log.Printf("Error: \"Could not switch to branch %s\" %s", branch, err)
		return err
	}

	if _, err := repo.Cmd("commit", "-m", "Close branch "+branch, "--close-branch"); err != nil {
		log.Printf("Error: \"Could not commit\" %s", err)
		return err
	}

	if _, err := repo.Update(trashBranch); err != nil {
		log.Printf("Error: \"Could not switch to branch %s\" %s", trashBranch, err)
		return err
	}

	if err := fakeMerge(repo, branch, trashBranch); err != nil {
		log.Printf("Error: \"Could not fake merge branch %s to branch %s\" %s", branch, trashBranch, err)
		return err
	}

	return nil
}

func fakeMerge(repo Repo, branch string, trashBranch string) error {

	repo.Cmd("-y", "merge", "--tool=internal:fail", branch) // Always produces an err
	if _, err := repo.Cmd("revert", "--all", "--rev", "."); err != nil {
		return err
	}
	if _, err := repo.Cmd("resolve", "-a", "-m"); err != nil {
		return err
	}
	if _, err := repo.Commit(fmt.Sprintf("Fake merge to close %s into %s", branch, trashBranch)); err != nil {
		return err
	}

	return nil
}

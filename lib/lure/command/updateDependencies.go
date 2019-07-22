package command

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/coveooss/lure/lib/lure"
	"github.com/coveooss/lure/lib/lure/log"
	"github.com/coveooss/lure/lib/lure/project"
	p "github.com/coveooss/lure/lib/lure/provider"
	"github.com/coveooss/lure/lib/lure/versionManager"
	"github.com/coveooss/lure/lib/lure/versionManager/mvn"
	"github.com/coveooss/lure/lib/lure/versionManager/npm"

	"github.com/vsekhar/govtil/guid"
)

// This part interesting
// https://github.com/golang/go/blob/1441f76938bf61a2c8c2ed1a65082ddde0319633/src/cmd/go/vcs.go

func appendIfMissing(modules []versionManager.ModuleVersion, modulesToAdd []versionManager.ModuleVersion) []versionManager.ModuleVersion {
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

func CheckForUpdatesJobCommand(project project.Project, sourceControl sourceControl, provider provider, args map[string]string) error {
	return checkForUpdatesJob(project, sourceControl, provider, args["commitMessage"])
}

func checkForUpdatesJob(project project.Project, sourceControl sourceControl, provider provider, commitMessage string) error {
	log.Logger.Infof("switching to default branch: %s", project.DefaultBranch)
	if _, err := sourceControl.Update(project.DefaultBranch); err != nil {
		return fmt.Errorf("Error: \"Could not switch to branch %s\" %s", project.DefaultBranch, err)
	}

	modulesToUpdate := make([]versionManager.ModuleVersion, 0, 0)

	if project.SkipPackageManager == nil || project.SkipPackageManager["npm"] != true {
		modulesToUpdate = appendIfMissing(modulesToUpdate, npm.NpmOutdated(sourceControl.WorkingPath()))
	}

	if project.SkipPackageManager == nil || project.SkipPackageManager["mvn"] != true {
		err, modulesToAdd := mvn.MvnOutdated(sourceControl.WorkingPath())
		if err != nil {
			return err
		}
		modulesToUpdate = appendIfMissing(modulesToUpdate, modulesToAdd)
	}

	log.Logger.Infof("Modules to update : %q", modulesToUpdate)

	ignoreDeclinedPRs := os.Getenv("IGNORE_DECLINED_PR") == "1"
	pullRequests, err := provider.GetPullRequests(project.Owner, project.Name, ignoreDeclinedPRs)
	if err != nil {
		return err
	}

	for _, moduleToUpdate := range modulesToUpdate {
		updateModule(moduleToUpdate, project, sourceControl, provider, pullRequests, commitMessage)
	}

	err = closeOldBranchesWithoutOpenPR(project, sourceControl, provider)
	if err != nil {
		return err
	}

	log.Logger.Infof("`[Check for updates done.")

	return nil
}

func updateModule(moduleToUpdate versionManager.ModuleVersion, project project.Project, sourceControl sourceControl, provider provider, existingPRs []p.PullRequest, commitMessage string) {
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
	dependencyBranchPrefix := branchPrefix + dependencyName
	dependencyBranchVersionPrefix := dependencyBranchPrefix + "-" + moduleToUpdate.Latest
	branchGUID, _ := guid.V4()
	var branch = sourceControl.SanitizeBranchName(dependencyBranchVersionPrefix + "-" + branchGUID.String())

	var openPRAlreadyExists = false
	var declinedPRAlreadyExists = false
	for _, pr := range existingPRs {
		if !openPRAlreadyExists && strings.HasPrefix(pr.Source.Branch.Name, dependencyBranchVersionPrefix) {
			if pr.State == "OPEN" {
				log.Logger.Infof("There already is an open PR for: '%s'. The branch name is: %s.", title, pr.Source.Branch.Name)
				openPRAlreadyExists = true
			} else {
				log.Logger.Infof("There was a declined PR for: '%s'. The branch name is: %s.", title, pr.Source.Branch.Name)
				declinedPRAlreadyExists = true
			}
			continue
		}

		if pr.State == "OPEN" && strings.HasPrefix(pr.Source.Branch.Name, dependencyBranchPrefix) {
			if os.Getenv("DRY_RUN") == "1" {
				log.Logger.Infof("Running in DryRun mode. PR '%s' made for older version would be declined.", pr.Title)
			} else {
				log.Logger.Infof("Declining PR '%s' made for older version.", pr.Title)
				provider.DeclinePullRequest(project.Owner, project.Name, pr.ID)
			}
		}
	}
	if openPRAlreadyExists || declinedPRAlreadyExists {
		return
	}

	log.Logger.Infof("switching %s to default branch: %s", sourceControl.LocalPath(), project.DefaultBranch)
	if _, err := sourceControl.Update(project.DefaultBranch); err != nil {
		log.Logger.Fatalf("\"Could not switch to branch %s\" %s", project.DefaultBranch, err)
	}

	log.Logger.Infof("Creating branch %s", branch)
	if _, err := sourceControl.Branch(branch); err != nil {
		log.Logger.Errorf("\"Could not create branch\" %s", err)
		return
	}

	hasChanges := false

	switch moduleToUpdate.Type {
	case "maven":
		hasChanges, _ = mvn.UpdateDependency(sourceControl.WorkingPath(), moduleToUpdate)
	case "npm":
		hasChanges, _ = npm.UpdateDependency(sourceControl.WorkingPath(), moduleToUpdate)
	}

	if hasChanges == false {
		log.Logger.Warnf("An update was available for %s but Lure could not update it", moduleToUpdate.Name)
		return
	}

	if _, err := sourceControl.Commit(lure.Tprintf(commitMessage, map[string]interface{}{"module": moduleToUpdate.Module, "version": moduleToUpdate.Latest})); err != nil {
		log.Logger.Errorf("\"Could not commit\" %s", err)
		return
	}

	if os.Getenv("DRY_RUN") == "1" {
		log.Logger.Info("Running in DryRun mode, not doing the pull request nor pushing the changes for ", branch)
	} else {
		log.Logger.Info("Pushing changes")
		if _, err := sourceControl.Push(); err != nil {
			log.Logger.Fatalf("\"Could not push\" %s", err)
			return
		}

		log.Logger.Infof("Creating PR")

		description := lure.Tprintf(commitMessage, map[string]interface{}{"module": moduleToUpdate.Module, "version": moduleToUpdate.Latest})
		provider.CreatePullRequest(branch, project.DefaultBranch, project.Owner, project.Name, title, description, *project.UseDefaultReviewers)
	}
}

func closeOldBranchesWithoutOpenPR(project project.Project, sourceControl sourceControl, provider provider) error {
	log.Logger.Info("Cleaning up lure branches with no associated PRs.")

	branchPrefix := project.BranchPrefix
	branches, err := sourceControl.GetActiveBranches()
	if err != nil {
		return err
	}
	existingPRs, err := provider.GetPullRequests(project.Owner, project.Name, false)
	if err != nil {
		return err
	}

	for _, branch := range branches {
		if strings.HasPrefix(branch, branchPrefix) {
			if isBranchDead(branch, existingPRs) {
				if os.Getenv("DRY_RUN") == "1" {
					log.Logger.Infof("Running in DryRun mode. Branch '%s' would of been closed.", branch)
				} else {
					if err := sourceControl.CloseBranch(branch); err != nil {
						println(err)
						return err
					}
				}
			}
		}
	}
	log.Logger.Info("Lure branches clean up done.")

	return nil
}

func isBranchDead(branch string, existingPRs []p.PullRequest) bool {
	for _, pr := range existingPRs {
		if pr.State == "OPEN" && branch == pr.Source.Branch.Name {
			return false
		}
	}
	return true
}

package main

import (
	"log"
	"os/exec"
	"fmt"
	"reflect"

	"github.com/k0kubun/pp"
	"github.com/vsekhar/govtil/guid"
	"golang.org/x/oauth2"
)

// This part interesting
// https://github.com/golang/go/blob/1441f76938bf61a2c8c2ed1a65082ddde0319633/src/cmd/go/vcs.go

func checkForUpdatesJob(token *oauth2.Token, projects []Project) {
	for _, project := range projects {
		pp.Println("Updating Project: ", project.Owner + "/" + project.Name)
		updateProject(token, project)
	}
}

func appendIfMissing(modules []moduleVersion, modulesToAdd []moduleVersion) []moduleVersion {
	for _, moduleToAdd := range modulesToAdd {
		exist := false
		for _, module := range modules {
			if (reflect.DeepEqual(module, moduleToAdd)) {
				exist = true
				break;
			}
		}
		if (!exist) {
			modules = append(modules, moduleToAdd)
		}
	}

	return modules
}

func updateProject(token *oauth2.Token, project Project) {

	if token == nil {
		pp.Printf("Error: \"Cant update no token\" %s", project)
		return
	}

	repoGUID, err := guid.V4()
	if err != nil {
		log.Printf("Error: \"Could not generate guid\" %s", err)
		return
	}
	repoPath := "/tmp/" + repoGUID.String()

	projectRemote := "bitbucket.org/" + project.Owner + "/" + project.Name

	log.Printf("Info: cloning: %s to %s", projectRemote, repoPath)

	repoRemote := "https://x-token-auth:" + token.AccessToken + "@" + projectRemote
	if err := hgClone(repoRemote, repoPath); err != nil {
		log.Printf("Error: \"Could not clone\" %s", err)
		return
	}

	log.Printf("Info: switching %s to default branch: %s", projectRemote, project.DefaultBranch)
	if err := hgUpdate(repoPath, project.DefaultBranch); err != nil {
		log.Fatalf("Error: \"Could not switch to branch %s\" %s", project.DefaultBranch, err)
	}

	modulesToUpdate := make([]moduleVersion, 0, 0)
	modulesToUpdate = appendIfMissing(modulesToUpdate, npmOutdated(repoPath))
	modulesToUpdate = appendIfMissing(modulesToUpdate, mvnOutdated(repoPath))
	pullRequests := getPullRequests(token.AccessToken, project.Owner, project.Name)

	for _, moduleToUpdate := range modulesToUpdate {
		updateModule(token, moduleToUpdate, project, repoPath, repoRemote, pullRequests)
	}
}

func updateModule(token *oauth2.Token, moduleToUpdate moduleVersion, project Project, repoPath string, repoRemote string, existingPRs []PullRequest) {

	title := fmt.Sprintf("Update %s dependency %s to version %s", moduleToUpdate.Type, moduleToUpdate.Module, moduleToUpdate.Latest)
	for _, pr := range existingPRs {
		if (pr.Title == title) {
			log.Printf("There already is a PR for: %s", title)
			return
		}
	}

	log.Printf("Info: switching %s to default branch: %s", repoPath, project.DefaultBranch)
	if err := hgUpdate(repoPath, project.DefaultBranch); err != nil {
		log.Fatalf("Error: \"Could not switch to branch %s\" %s", project.DefaultBranch, err)
	}

	branchGUID, _ := guid.V4()
	branch := hgSanitizeBranchName("lure-" + moduleToUpdate.Module + "-" + moduleToUpdate.Latest + "-" + branchGUID.String())
	log.Printf("Creating branch %s\n", branch)
	if err := hgBranch(repoPath, branch); err != nil {
		log.Printf("Error: \"Could not create branch\" %s", err)
		return
	}

	switch moduleToUpdate.Type {
	case "maven": mvnUpdateDep(repoPath, moduleToUpdate.Module, moduleToUpdate.Latest)
	case "npm": readPackageJSON(repoPath, moduleToUpdate.Module, moduleToUpdate.Latest)
	}

	if err := hgCommit(repoPath, "Update "+moduleToUpdate.Module+" to "+moduleToUpdate.Latest); err != nil {
		log.Printf("Error: \"Could not commit\" %s", err)
		return
	}

	log.Printf("Pushing changes\n")
	if err := hgPush(repoPath, repoRemote); err != nil {
		log.Fatalf("Error: \"Could not push\" %s", err)
		return
	}

	log.Printf("Creating PR\n")
	description := fmt.Sprintf("%s version %s is now available! Please update.", moduleToUpdate.Module, moduleToUpdate.Latest)
	createPullRequest(branch, token.AccessToken, project.Owner, project.Name, title, description)
}

func execute(pwd string, command string, params ...string) error {
	log.Printf("%s %q\n", command, params)
	cmd := exec.Command(command, params...)
	cmd.Dir = pwd

	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

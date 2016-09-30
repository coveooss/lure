package main

import (
	"log"
	"os/exec"
	"time"
	"fmt"

	"github.com/k0kubun/pp"
	"github.com/vsekhar/govtil/guid"
	"golang.org/x/oauth2"
)

// This part interesting
// https://github.com/golang/go/blob/1441f76938bf61a2c8c2ed1a65082ddde0319633/src/cmd/go/vcs.go

func checkForUpdatesJob(token *oauth2.Token, projects []Project) {
	for {
		for _, project := range projects {
			pp.Println("updating: ", project.Owner + "/" + project.Name)
			updateProject(token, project)
		}
		time.Sleep(10 * time.Second)
	}
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

	log.Printf("Info: updating %s to: %s", projectRemote, project.DefaultBranch)
	if err := hgUpdate(repoPath, project.DefaultBranch); err != nil {
		log.Fatalf("Error: \"Could not update\" %s", err)
	}

	modulesToUpdate := mvnOutdated(repoPath)
	pullRequests := getPullRequests(token.AccessToken, project.Owner, project.Name)

	for _, moduleToUpdate := range modulesToUpdate {
		updateModule(token, moduleToUpdate, project, repoPath, repoRemote, pullRequests)
	}
}

func updateModule(token *oauth2.Token, moduleToUpdate moduleVersion, project Project, repoPath string, repoRemote string, existingPRs []PullRequest) {

	title := fmt.Sprintf("Update %s to version %s", moduleToUpdate.Module, moduleToUpdate.Latest)
	for _, pr := range existingPRs {
		if (pr.Title == title) {
			log.Printf("There already is a PR for: %s", title)
			return
		}
	}

	pp.Println("project needs update of ", moduleToUpdate)

	pp.Println("updating to default branch:", moduleToUpdate)
	if err := hgUpdate(repoPath, project.DefaultBranch); err != nil {
		log.Printf("Error: \"Could not switch to default branch\" %s", err)
		return
	}

	branchGUID, _ := guid.V4()
	branch := hgSanitizeBranchName("lure-" + moduleToUpdate.Module + "-" + moduleToUpdate.Latest + "-" + branchGUID.String())
	pp.Println("creating branch", branch)
	if err := hgBranch(repoPath, branch); err != nil {
		log.Printf("Error: \"Could not create branch\" %s", err)
		return
	}

	//readPackageJSON(repoPath, moduleToUpdate.Module, moduleToUpdate.Latest)
	mvnUpdateDep(repoPath, moduleToUpdate.Module, moduleToUpdate.Latest)

	if err := hgCommit(repoPath, "Update "+moduleToUpdate.Module+" to "+moduleToUpdate.Latest); err != nil {
		log.Printf("Error: \"Could not commit\" %s", err)
		return
	}

	pp.Println("Pushing changes")
	if err := hgPush(repoPath, repoRemote); err != nil {
		log.Fatalf("Error: \"Could not push\" %s", err)
		return
	}

	pp.Println("creating PR")
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

package main

import (
	"log"
	"os/exec"
	"time"

	"github.com/k0kubun/pp"
	"github.com/vsekhar/govtil/guid"
)

// This part interesting
// https://github.com/golang/go/blob/1441f76938bf61a2c8c2ed1a65082ddde0319633/src/cmd/go/vcs.go

func checkForUpdatesJob(projects []*Project) {
	for {
		for _, project := range projects {
			pp.Println("updating: ", project.Remote)
			updateProject(*project)
		}
		time.Sleep(10 * time.Second)
	}
}

func updateProject(project Project) {

	if project.Token == nil {
		pp.Printf("Error: \"Cant update no token\" %s", project)
		return
	}

	repoGUID, err := guid.V4()
	if err != nil {
		log.Printf("Error: \"Could not generate guid\" %s", err)
		return
	}
	repoPath := "/tmp/" + repoGUID.String()

	log.Printf("Info: cloning: %s to %s", project.Remote, repoPath)

	repoRemote := "https://x-token-auth:" + project.Token.AccessToken + "@" + project.Remote
	if err := hgClone(repoRemote, repoPath); err != nil {
		log.Printf("Error: \"Could not clone\" %s", err)
		return
	}

	log.Printf("Info: updating %s to: %s", project.Remote, project.DefaultBranch)
	if err := hgUpdate(repoPath, project.DefaultBranch); err != nil {
		log.Fatalf("Error: \"Could not update\" %s", err)
	}

	modulesToUpdate := mvnOutdated(repoPath)

	for _, moduleToUpdate := range modulesToUpdate {
		updateModule(moduleToUpdate, project, repoPath, repoRemote)
	}
}

func updateModule(moduleToUpdate moduleVersion, project Project, repoPath string, repoRemote string) {
	pp.Println("project needs update of ", moduleToUpdate)

	pp.Println("updating to default branch:", moduleToUpdate)
	if err := hgUpdate(repoPath, project.DefaultBranch); err != nil {
		log.Printf("Error: \"Could not switch to default branch\" %s", err)
		return
	}

	branch := "lure-" + moduleToUpdate.Module + "-" + moduleToUpdate.Latest
	pp.Println("creating branch", branch)
	if err := hgBranch(repoPath, branch); err != nil {
		log.Printf("Error: \"Could not create branch\" %s", err)
		return
	}

	readPackageJSON(repoPath, moduleToUpdate.Module, moduleToUpdate.Latest)

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
	createPullRequest(branch, project.Token.AccessToken, "pastjean", "dummy", moduleToUpdate.Module, moduleToUpdate.Latest)
}

func execute(pwd string, command string, params ...string) error {
	cmd := exec.Command(command, params...)
	cmd.Dir = pwd

	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

package main

import (
	"log"
	"os/exec"

	"github.com/vsekhar/govtil/guid"
)

// This part interesting
// https://github.com/golang/go/blob/1441f76938bf61a2c8c2ed1a65082ddde0319633/src/cmd/go/vcs.go

func main() {
	repo := "ssh://hg@bitbucket.org/pastjean/dummy"
	repoGUID, err := guid.V4()
	if err != nil {
		log.Fatalf("Error: \"Could not generate guid\" %s", err)
	}

	repoPath := "/tmp/" + repoGUID.String()

	if err := hgClone(repo, repoPath); err != nil {
		log.Fatalf("Error: \"Could not clone\" %s", err)
	}

	if err := hgUpdate(repoPath, "default"); err != nil {
		log.Fatalf("Error: \"Could not update\" %s", err)
	}

	// TODO: verifier les dépendances

	// TODO: for each dependency to update
	// for () {
	if err := hgUpdate(repoPath, "default"); err != nil {
		log.Fatalf("Error: \"Could not update\" %s", err)
	}

	if err := hgBranch(repoPath, "lure-yournewbranchname"); err != nil {
		log.Fatalf("Error: \"Could not update\" %s", err)
	}
	// TODO: update dependency

	if err := hgCommit(repoPath, "MOTHERFUKING NEW DEPENDENCY"); err != nil {
		log.Fatalf("Error: \"Could not update\" %s", err)
	}

	// }
}

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

func execute(pwd string, command string, params ...string) error {
	cmd := exec.Command(command, params...)
	cmd.Dir = pwd

	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

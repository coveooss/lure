package main

import (
	"fmt"
	"log"

	"github.com/k0kubun/pp"
	"golang.org/x/oauth2"
)

func checkForBranchDifferencesJob(token *oauth2.Token, projects []Project, fromBranch string, toBranch string) {
	for _, project := range projects {
		pp.Println(fmt.Sprintf("Updating Project: %s/%s", project.Owner , project.Name))
		if err := checkForBranchDifferences(token, project, fromBranch, toBranch); err != nil {
			log.Fatal(err)
		}
	}
}

func checkForBranchDifferences(token *oauth2.Token, project Project, fromBranch string, toBranch string) (error) {

	repo, err := cloneRepo(token, project)
	if err != nil {
		return err
	}

	if _, err := repo.Update(toBranch); err != nil {
		return err
	}

	commits, err := repo.LogCommitsBetween(toBranch, fromBranch)
	if err != nil {
		return err
	}

	if len(commits) == 0 {
		log.Printf("Branches %s and %s are identical\n", fromBranch, toBranch)
		return nil
	}
	log.Printf("Found %d commits in %s missing from %s: %s\n", len(commits), fromBranch, toBranch, commits)

	mergeBranch := "lure_merge_" + fromBranch + "_into_" + toBranch

	if _, err := repo.Branch(mergeBranch); err != nil {
		return err
	}

	if _, err := repo.Merge(fromBranch); err != nil {
		return err
	}

	if _, err := repo.Commit(fmt.Sprintf("merge %s into %s", fromBranch, toBranch)); err != nil {
		return err
	}

	if _, err := repo.Push(); err != nil {
		return err
	}

	if err := createPullRequest(token.AccessToken, mergeBranch, toBranch, project.Owner, project.Name, fmt.Sprintf("Merge %s into %s", fromBranch, toBranch), ""); err != nil {
		return err
	}

	return nil
}

package main

import (
	"fmt"
	"log"

	"github.com/k0kubun/pp"
)

func checkForBranchDifferencesJob(auth Authentication, projects []Project, fromBranch string, toBranch string) {
	for _, project := range projects {
		pp.Println(fmt.Sprintf("Updating Project: %s/%s", project.Owner , project.Name))
		if err := checkForBranchDifferences(auth, project, fromBranch, toBranch); err != nil {
			log.Fatal(err)
		}
	}
}

func checkForBranchDifferences(auth Authentication, project Project, fromBranch string, toBranch string) (error) {

	repo, err := cloneRepo(auth, project)
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

	if err := createPullRequest(auth, mergeBranch, toBranch, project.Owner, project.Name, fmt.Sprintf("Merge %s into %s", fromBranch, toBranch), ""); err != nil {
		return err
	}

	return nil
}

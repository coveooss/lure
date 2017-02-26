package main

import (
	"fmt"
	"log"
	"errors"
)

func synchronizedBranchesCommand(auth Authentication, project Project, args map[string]string) (error) {
	fromBranch, ok := args["from"]
	if !ok {
		return errors.New("Missing argument 'from'")
	}
	toBranch, ok := args["to"]
	if !ok {
		return errors.New("Missing argument 'to'")
	}

	return synchronizedBranches(auth, project, fromBranch, toBranch)
}

func synchronizedBranches(auth Authentication, project Project, fromBranch string, toBranch string) (error) {

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

	mergeBranch := "lure_merge_" + fromBranch + "_into_" + toBranch + "_" + commits[len(commits)-1]

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
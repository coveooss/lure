package lure

import (
	"errors"
	"fmt"
	"log"
	"os"
)

func SynchronizedBranchesCommand(auth Authentication, project Project, args map[string]string) error {
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

func synchronizedBranches(auth Authentication, project Project, fromBranch string, toBranch string) error {

	repo, err := cloneRepo(auth, project)
	if err != nil {
		return err
	}

	// if git, setup a tracking branch to avoid having to prefix origin/ later
	if _, err := repo.Update(toBranch); err != nil {
		return err
	}

	if _, err := repo.Update(fromBranch); err != nil {
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

	// hg requires a commit in order to create a branch
	switch repo := repo.(type) {
	case HgRepo:
		if _, err := repo.Commit(fmt.Sprintf("merge %s into %s", fromBranch, toBranch)); err != nil {
			return err
		}
	}

	if os.Getenv("DRY_RUN") == "1" {
		log.Println("Running in DryRun mode, not doing the pull request nor pushing the changes")
	} else {
		if _, err := repo.Push(); err != nil {
			return err
		}

		if err := createPullRequest(auth, mergeBranch, toBranch, project.Owner, project.Name, fmt.Sprintf("Merge %s into %s", fromBranch, toBranch), "", *project.UseDefaultReviewers); err != nil {
			return err
		}
	}

	return nil
}

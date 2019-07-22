package command

import (
	"errors"
	"fmt"
	"os"

	"github.com/coveooss/lure/lib/lure/project"

	"github.com/coveooss/lure/lib/lure/log"
)

func SynchronizedBranchesCommand(project project.Project, sourceControl sourceControl, provider provider, args map[string]string) error {
	fromBranch, ok := args["from"]
	if !ok {
		return errors.New("Missing argument 'from'")
	}
	toBranch, ok := args["to"]
	if !ok {
		return errors.New("Missing argument 'to'")
	}

	return synchronizedBranches(project, sourceControl, provider, fromBranch, toBranch)
}

func synchronizedBranches(project project.Project, sourceControl sourceControl, provider provider, fromBranch string, toBranch string) error {

	// if git, setup a tracking branch to avoid having to prefix origin/ later
	if _, err := sourceControl.Update(toBranch); err != nil {
		return err
	}

	if _, err := sourceControl.Update(fromBranch); err != nil {
		return err
	}

	commits, err := sourceControl.LogCommitsBetween(toBranch, fromBranch)
	if err != nil {
		return err
	}

	if len(commits) == 0 {
		log.Logger.Infof("Branches %s and %s are identical\n", fromBranch, toBranch)
		return nil
	}
	log.Logger.Infof("Found %d commits in %s missing from %s: %s\n", len(commits), fromBranch, toBranch, commits)

	mergeBranch := "lure_merge_" + fromBranch + "_into_" + toBranch + "_" + commits[len(commits)-1]

	if _, err := sourceControl.Branch(mergeBranch); err != nil {
		return err
	}

	if os.Getenv("DRY_RUN") == "1" {
		log.Logger.Info("Running in DryRun mode, not doing the pull request nor pushing the changes")
	} else {
		if _, err := sourceControl.Push(); err != nil {
			return err
		}

		if err := provider.CreatePullRequest(mergeBranch, toBranch, project.Owner, project.Name, fmt.Sprintf("Merge %s into %s", fromBranch, toBranch), "", *project.UseDefaultReviewers); err != nil {
			return err
		}
	}

	return nil
}

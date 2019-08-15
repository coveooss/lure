package command

import (
	"github.com/coveooss/lure/lib/lure/project"
	managementsystem "github.com/coveooss/lure/lib/lure/repositorymanagementsystem"
	"github.com/coveooss/lure/lib/lure/vcs"
)

type sourceControl interface {
	Update(string) (string, error)
	CommitsBetween(string, string) ([]string, error)
	Branch(string) (string, error)
	SoftBranch(string) (string, error)
	Push() (string, error)
	WorkingPath() string
	ActiveBranches() ([]string, error)
	CloseBranch(string) error
	LocalPath() string
	SanitizeBranchName(string) string
	Commit(string) (string, error)
}

type repository interface {
	CreatePullRequest(sourceBranch string, destBranch string, owner string, repo string, title string, description string, useDefaultReviewers bool) error
	GetPullRequests(string, string, bool) ([]managementsystem.PullRequest, error)
	DeclinePullRequest(string, string, int) error
}

type Func func(project project.Project, sourceControl vcs.SourceControl, repository repository, args map[string]string) error

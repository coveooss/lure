package command

import (
	"github.com/coveooss/lure/lib/lure/project"
	p "github.com/coveooss/lure/lib/lure/provider"
	"github.com/coveooss/lure/lib/lure/vcs"
)

type sourceControl interface {
	Update(string) (string, error)
	LogCommitsBetween(string, string) ([]string, error)
	Branch(string) (string, error)
	Push() (string, error)
	WorkingPath() string
	GetActiveBranches() ([]string, error)
	CloseBranch(string) error
	LocalPath() string
	SanitizeBranchName(string) string
	Commit(string) (string, error)
}

type provider interface {
	CreatePullRequest(string, string, string, string, string, string, bool) error
	GetPullRequests(string, string, bool) ([]p.PullRequest, error)
	DeclinePullRequest(string, string, int) error
}

type Func func(project project.Project, sourceControl vcs.SourceControl, provider provider, args map[string]string) error

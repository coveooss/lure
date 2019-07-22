package vcs

import (
	"fmt"
	"strings"
)

type header interface {
	Add(key, value string)
}

type Authentication interface {
	AuthenticateURL(url string) string
	AuthenticateHTTPRequest(header header)
}

type TokenAuth struct {
	Token string
}

func (auth TokenAuth) AuthenticateURL(url string) string {
	return strings.Replace(url, "://", fmt.Sprintf("://x-token-auth:%s@", auth.Token), 1)
}

func (auth TokenAuth) AuthenticateHTTPRequest(header header) {
	header.Add("Authorization", "Bearer "+auth.Token)
}

type UserPassAuth struct {
	Username string
	Password string
}

func (auth UserPassAuth) AuthenticateURL(url string) string {
	return strings.Replace(url, "://", fmt.Sprintf("://%s:%s@", auth.Username, auth.Password), 1)
}

func (auth UserPassAuth) AuthenticateHTTPRequest(header header) {
	//NOOP
}

type SourceControl interface {
	WorkingPath() string
	LocalPath() string
	RemotePath() string

	Cmd(args ...string) (string, error)
	Update(rev string) (string, error)
	Branch(branchname string) (string, error)
	Commit(message string) (string, error)
	Push() (string, error)
	GetActiveBranches() ([]string, error)
	CloseBranch(branch string) error
	Clone() error
	SanitizeBranchName(branchName string) string

	LogCommitsBetween(baseRev string, secondRev string) ([]string, error)

	GetName() string
}

const (
	Git = "git"
	Hg  = "hg"
)

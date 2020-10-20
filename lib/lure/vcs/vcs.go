package vcs

import (
	"fmt"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"net/http"
	"strings"
)

type header interface {
	Add(key, value string)
}

type Authentication interface {
	AuthenticateURL(url string) string
	AuthenticateHTTPRequest(header header)
	AuthenticateWithToken() *http.Client
}

type TokenAuth struct {
	User string  // The user to use when cloning (x-token-auth for Bitbucket, x-access-token for GitHub
	Token string
}

func (auth TokenAuth) AuthenticateURL(url string) string {
	return strings.Replace(url, "://", fmt.Sprintf("://%s:%s@", auth.User, auth.Token), 1)
}

func (auth TokenAuth) AuthenticateHTTPRequest(header header) {
	header.Add("Authorization", "Bearer " + auth.Token)
}

func (auth TokenAuth) AuthenticateWithToken() *http.Client {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: auth.Token},
	)
	return oauth2.NewClient(ctx, ts)
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

func (auth UserPassAuth) AuthenticateWithToken() *http.Client {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: auth.Password},
	)
	return oauth2.NewClient(ctx, ts)
}

type SourceControl interface {
	WorkingPath() string
	LocalPath() string
	RemotePath() string

	Cmd(args ...string) (string, error)
	Update(rev string) (string, error)
	Branch(branchname string) (string, error)
	SoftBranch(branchname string) (string, error)
	Commit(message string) (string, error)
	Push() (string, error)
	ActiveBranches() ([]string, error)
	CloseBranch(branch string) error
	Clone() error
	SanitizeBranchName(branchName string) string

	CommitsBetween(baseRev string, secondRev string) ([]string, error)

	GetName() string
}

const (
	Git = "git"
	Hg  = "hg"

	Bitbucket = "bitbucket"
	GitHub = "github"
)

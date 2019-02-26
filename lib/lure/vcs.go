package lure

type Authentication interface{}

type TokenAuth struct {
	Token string
}

type UserPassAuth struct {
	Username string
	Password string
}

type Repo interface {
	LocalPath() string
	RemotePath() string

	Cmd(args ...string) (string, error)
	Update(rev string) (string, error)
	Branch(branchname string) (string, error)
	Commit(message string) (string, error)
	Push() (string, error)
	CloseBranch(branch string) error

	LogCommitsBetween(baseRev string, secondRev string) ([]string, error)
}

const (
	Git = "git"
	Hg  = "hg"
)

package repositorymanagementsystem

type Branch interface {
	GetName() string
}

type PullRequest struct {
	ID                int
	Title             string
	Description       string
	Source            Branch
	Dest              Branch
	CloseSourceBranch bool
	State             string
	Reviewers         []user
}

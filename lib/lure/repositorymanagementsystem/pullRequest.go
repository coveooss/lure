package repositorymanagementsystem

type Branch interface {
	GetName() string
}

type PullRequest struct {
	ID                int    `json:"id"`
	Title             string `json:"title"`
	Description       string `json:"description"`
	Source            Branch `json:"source"`
	Dest              Branch `json:"dest"`
	CloseSourceBranch bool   `json:"close_source_branch"`
	State             string `json:"state"`
	Reviewers         []user `json:"reviewers"`
}

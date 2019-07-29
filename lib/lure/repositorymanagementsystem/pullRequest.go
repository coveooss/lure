package repositorymanagementsystem

type PullRequest struct {
	ID                int    `json:"id"`
	Title             string `json:"title"`
	Description       string `json:"description"`
	Source            source `json:"source"`
	Dest              dest   `json:"destination"`
	CloseSourceBranch bool   `json:"close_source_branch"`
	State             string `json:"state"`
	Reviewers         []user `json:"reviewers"`
}

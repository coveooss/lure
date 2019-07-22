package project

type Command struct {
	Name string            `json:"name"`
	Args map[string]string `json:"args"`
}

type Project struct {
	Vcs                 string          `json:"vcs"`
	Owner               string          `json:"owner"`
	Name                string          `json:"name"`
	DefaultBranch       string          `json:"defaultBranch"`
	BranchPrefix        string          `json:"branchPrefix"`
	TrashBranch         string          `json:"trashBranch"`
	BasePath            string          `json:"basePath"`
	SkipPackageManager  map[string]bool `json:"skipPackageManager"`
	UseDefaultReviewers *bool           `json:"useDefaultReviewers"`
	Commands            []Command       `json:"commands"`
}

func (project Project) GetDefaultBranch() string {
	return project.DefaultBranch
}

func (project Project) GetTrashBranch() string {
	return project.TrashBranch
}
func (project Project) GetBasePath() string {
	return project.BasePath
}
func (project Project) GetOwner() string {
	return project.Owner
}

func (project Project) GetName() string {
	return project.Name
}

type LureConfig struct {
	Projects []Project `json:"projects"`
}

const (
	defaultBranchPrefix           string = "lure-"
	defaultTrashBranch            string = "closed-branch-trash"
	defaultCommitMessage          string = "Update {{.module}} to {{.version}}"
	defaultPullRequestDescription string = "{{.module}} version {{.version}} is now available! Please update."
)

func newTrue() *bool {
	b := true
	return &b
}

// InitProjectDefaultValues initializes project with default values as necessary
func InitProjectDefaultValues(project *Project) {
	if project.BranchPrefix == "" {
		project.BranchPrefix = defaultBranchPrefix
	}
	if project.TrashBranch == "" {
		project.TrashBranch = defaultTrashBranch
	}
	if project.UseDefaultReviewers == nil {
		project.UseDefaultReviewers = newTrue()
	}
	for i := range project.Commands {
		cmd := &project.Commands[i]

		if cmd.Args == nil {
			cmd.Args = map[string]string{}
		}

		_, haveCommitMessage := cmd.Args["commitMessage"]
		if !haveCommitMessage {
			cmd.Args["commitMessage"] = defaultCommitMessage
		}

		_, havePullRequestDescription := cmd.Args["pullRequestDescription"]
		if !havePullRequestDescription {
			cmd.Args["pullRequestDescription"] = defaultPullRequestDescription
		}
	}
}

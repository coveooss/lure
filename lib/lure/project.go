package lure

import "strings"

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

type LureConfig struct {
	Projects []Project `json:"projects"`
}

const (
	defaultBranchPrefix  string = "lure-"
	defaultTrashBranch   string = "closed-branch-trash"
	defaultCommitMessage string = "Update {{.module}} to {{.version}}"
)

func newTrue() *bool {
	b := true
	return &b
}

// InitProjectDefaultValues initializes project with default values as necessary
func InitProjectDefaultValues(project *Project, repo Repo) {
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
		_, ok := cmd.Args["commitMessage"]
		if !ok {
			cmd.Args["commitMessage"] = defaultCommitMessage
		}
	}
	var completeBasePath strings.Builder
	completeBasePath.WriteString(repo.LocalPath())
	completeBasePath.WriteString("/")
	completeBasePath.WriteString(project.BasePath)
	project.BasePath = completeBasePath.String()
}

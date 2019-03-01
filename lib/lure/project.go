package lure

type Command struct {
	Name string            `json:"name"`
	Args map[string]string `json:"args"`
}

type Project struct {
	Vcs           string    `json:"vcs"`
	Owner         string    `json:"owner"`
	Name          string    `json:"name"`
	DefaultBranch string    `json:"defaultBranch"`
	BranchPrefix  string    `json:"branchPrefix"`
	TrashBranch   string    `json:"trashBranch"`
	BasePath      string    `json:"basePath"`
	Commands      []Command `json:"commands"`
}

type LureConfig struct {
	Projects []Project `json:"projects"`
}

const (
	defaultBranchPrefix string = "lure-"
	defaultTrashBranch  string = "closed-branch-trash"
)

// InitProjectDefaultValues initializes project with default values as necessary
func InitProjectDefaultValues(project *Project) {
	if project.BranchPrefix == "" {
		project.BranchPrefix = defaultBranchPrefix
	}
	if project.TrashBranch == "" {
		project.TrashBranch = defaultTrashBranch
	}
}

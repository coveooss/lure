package lure

type Command struct {
	Name string            `json:"name"`
	Args map[string]string `json:"args"`
}

type Project struct {
	Vcs           string          `json:"vcs"`
	Owner         string          `json:"owner"`
	Name          string          `json:"name"`
	DefaultBranch string          `json:"defaultBranch"`
	BranchPrefix  string          `json:"branchPrefix"`
	BasePath      string          `json:"basePath"`
	PackagesTypes map[string]bool `json:"packageTypes"`
	Commands      []Command       `json:"commands"`
}

type LureConfig struct {
	Projects []Project `json:"projects"`
}

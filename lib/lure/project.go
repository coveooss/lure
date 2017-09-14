package lure

type Command struct {
	Name string            `json:"name"`
	Args map[string]string `json:"args"`
}

type Project struct {
	Vcs		  string    `json:"vcs"`
	Owner		  string `json:"owner"`
	Name		  string `json:"name"`
	DefaultBranch 	  string `json:"defaultBranch"`
	BasePath	  string `json:"basePath"`
	Commands 	  []Command `json:"commands"`
}

type LureConfig struct {
	Projects []Project `json:"projects"`
}

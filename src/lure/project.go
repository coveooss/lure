package main

type Command struct {
	Name string            `json:"name"`
	Args map[string]string `json:"args"`
}

type Project struct {
	Owner		  string    `json:"owner"`
	Name		  string    `json:"name"`
	DefaultBranch 	  string    `json:"defaultBranch"`
	Commands 	  []Command `json:"commands"`
}

type LureConfig struct {
	Projects []Project `json:"projects"`
}

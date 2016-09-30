package main

// Project is a ...
type Project struct {
	Owner		  string `json:"owner"`
	Name		  string `json:"name"`
	DefaultBranch string `json:"defaultBranch"`
}

type LureConfig struct {
	Projects []Project `json:"projects"`
}

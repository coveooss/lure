package main

import "golang.org/x/oauth2"

// Project is a ...
type Project struct {
	Owner		  string
	Name		  string
	Remote        string `json:"remote"`
	DefaultBranch string `json:"default_branch"`
	Token         *oauth2.Token
}

package main

import "golang.org/x/oauth2"

// Project is a ...
type Project struct {
	Remote        string `json:"remote"`
	DefaultBranch string `json:"default_branch"`
	Token         *oauth2.Token
}
